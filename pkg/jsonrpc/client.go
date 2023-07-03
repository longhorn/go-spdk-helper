package jsonrpc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	DefaultConcurrentLimit = 1024

	DefaultResponseReadWaitPeriod = 10 * time.Millisecond

	DefaultShortTimeout = 30 * time.Second
	DefaultLongTimeout  = 24 * time.Hour
)

type Client struct {
	ctx context.Context

	conn net.Conn

	idCounter uint32

	encoder *json.Encoder
	decoder *json.Decoder

	sem               chan interface{}
	msgWrapperQueue   chan *messageWrapper
	respReceiverQueue chan *Response
	responseChans     map[uint32]chan *Response
}

type messageWrapper struct {
	method       string
	params       interface{}
	responseChan chan *Response
}

func NewClient(ctx context.Context, conn net.Conn) *Client {
	c := &Client{
		ctx: ctx,

		conn: conn,

		// idCounter is required for each SPDK rpc request.
		// If it starts from 1, there may be a conflict when there are other clients try to talk with the spdk_tgt.
		// Hence, we choose a random small number as the first id counter.
		// Notice that there is still a chance to encounter the number conflict case.
		// Since it's not a main blocker or frequently happened case, we won't take too much time on a better solution now.
		idCounter: rand.Uint32() % 10000,

		encoder: json.NewEncoder(conn),
		decoder: json.NewDecoder(conn),

		sem:               make(chan interface{}, DefaultConcurrentLimit),
		msgWrapperQueue:   make(chan *messageWrapper, DefaultConcurrentLimit),
		respReceiverQueue: make(chan *Response, DefaultConcurrentLimit),
		responseChans:     make(map[uint32]chan *Response),
	}
	c.encoder.SetIndent("", "\t")

	go c.dispatcher()
	go c.read()

	return c
}

func (c *Client) SendMsgWithTimeout(method string, params interface{}, timeout time.Duration) (res []byte, err error) {
	id := atomic.AddUint32(&c.idCounter, 1)
	msg := NewMessage(id, method, params)
	var resp Response

	defer func() {
		if err != nil {
			err = JSONClientError{
				ID:          id,
				Method:      method,
				Params:      params,
				ErrorDetail: err,
			}
		}

		// For debug purpose
		stdenc := json.NewEncoder(os.Stdin)
		stdenc.SetIndent("", "\t")
		stdenc.Encode(msg)
		stdenc.Encode(&resp)
	}()

	marshaledParams, err := json.Marshal(msg.Params)
	if err != nil {
		return nil, err
	}
	if string(marshaledParams) == "{}" {
		msg.Params = nil
	}

	connEncoder := json.NewEncoder(c.conn)
	connEncoder.SetIndent("", "\t")
	if err = connEncoder.Encode(msg); err != nil {
		return nil, err
	}

	connDecoder := json.NewDecoder(bufio.NewReader(c.conn))
	for count := 0; count <= int(timeout/time.Second); count++ {
		if connDecoder.More() {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !connDecoder.More() {
		return nil, fmt.Errorf("timeout %v second waiting for the response from the SPDK JSON RPC server", timeout.Seconds())
	}

	if err = connDecoder.Decode(&resp); err != nil {
		return nil, err
	}
	if resp.ID != id {
		return nil, fmt.Errorf("received response but the id mismatching, message id %d, response id %d", id, resp.ID)
	}

	if resp.ErrorInfo != nil {
		return nil, resp.ErrorInfo
	}

	buf := bytes.Buffer{}
	jsonEncoder := json.NewEncoder(&buf)
	if err := jsonEncoder.Encode(resp.Result); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *Client) SendCommand(method string, params interface{}) ([]byte, error) {
	return c.SendMsgWithTimeout(method, params, DefaultShortTimeout)
}

func (c *Client) SendCommandWithLongTimeout(method string, params interface{}) ([]byte, error) {
	return c.SendMsgWithTimeout(method, params, DefaultLongTimeout)
}

func (c *Client) handleShutdown() {
	for _, ch := range c.responseChans {
		close(ch)
	}
}

func (c *Client) handleSend(msgWrapper *messageWrapper) {
	id := c.idCounter
	c.idCounter++
	c.responseChans[id] = msgWrapper.responseChan

	if err := c.encoder.Encode(NewMessage(id, msgWrapper.method, msgWrapper.params)); err != nil {
		logrus.WithError(err).Errorf("Failed to encode during handleSend")

		// In case of the cached error info of the old encoder fails the following response, it's better to recreate the encoder.
		c.encoder = json.NewEncoder(c.conn)
		c.encoder.SetIndent("", "\t")

	}
}

func (c *Client) handleRecv(resp *Response) {
	ch, exists := c.responseChans[resp.ID]
	if !exists {
		logrus.Warnf("Cannot find the response channel during handleRecv, will discard response: %+v", resp)
		return
	}
	delete(c.responseChans, resp.ID)

	ch <- resp
	close(ch)
}

func (c *Client) dispatcher() {
	for {
		select {
		case <-c.ctx.Done():
			c.handleShutdown()
			return
		case msg := <-c.msgWrapperQueue:
			c.handleSend(msg)
		case resp := <-c.respReceiverQueue:
			c.handleRecv(resp)
		}
	}
}

func (c *Client) read() {
	ticker := time.NewTicker(DefaultResponseReadWaitPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if !c.decoder.More() {
				continue
			}

			var resp Response
			if err := c.decoder.Decode(&resp); err != nil {
				logrus.WithError(err).Errorf("Failed to decoding response during read")

				// In case of the cached error info of the old decoder fails the following response, it's better to recreate the decoder.
				c.decoder = json.NewDecoder(c.conn)
				continue
			}
			c.respReceiverQueue <- &resp
		}
	}
}

func (c *Client) SendMsgAsyncWithTimeout(method string, params interface{}, timeout time.Duration) (res []byte, err error) {
	var resp *Response

	defer func() {
		if err != nil {
			id := uint32(0)
			if resp != nil {
				id = resp.ID
			}
			err = JSONClientError{
				ID:          id,
				Method:      method,
				Params:      params,
				ErrorDetail: err,
			}
		}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-c.ctx.Done():
		return nil, fmt.Errorf("context done during async message send")
	case c.sem <- nil:
		defer func() {
			<-c.sem
		}()
	case <-timer.C:
		return nil, fmt.Errorf("timeout %v getting semaphores during async message send", timeout)
	}

	marshaledParams, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	if string(marshaledParams) == "{}" {
		params = nil
	}

	responseChan := make(chan *Response)
	msgWrapper := &messageWrapper{
		method:       method,
		params:       params,
		responseChan: responseChan,
	}

	select {
	case <-c.ctx.Done():
		return nil, fmt.Errorf("context done during async message send")
	case c.msgWrapperQueue <- msgWrapper:
	case <-timer.C:
		return nil, fmt.Errorf("timeout %v queueing message during async message send", timeout)
	}

	select {
	case <-c.ctx.Done():
		return nil, fmt.Errorf("context done during async message send")
	case resp = <-responseChan:
		if resp == nil {
			return nil, fmt.Errorf("received nil response during async message send, maybe the response channel somehow is closed")
		}
	case <-timer.C:
		return nil, fmt.Errorf("timeout %v waiting for response during async message send", timeout)
	}

	if resp.ErrorInfo != nil {
		return nil, resp.ErrorInfo
	}

	buf := bytes.Buffer{}
	e := json.NewEncoder(&buf)
	if err := e.Encode(resp.Result); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
