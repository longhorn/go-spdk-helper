package jsonrpc

import (
	"encoding/json"
	"io"
	"testing"
	"time"
)

// TestClient_cleanupExpiredEntries is a regression test for
// https://github.com/longhorn/longhorn/issues/13553: responseChans and
// responseChanInfoMap were only ever cleaned up by handleRecv, when a
// matching response actually arrived. If a request's response never arrives
// (the SPDK target hangs, crashes, or the message is otherwise dropped),
// SendMsgAsyncWithTimeout returns a timeout error to the caller, but the
// dispatcher-side bookkeeping for that request id was left behind forever,
// growing unbounded on a long-lived client. cleanupExpiredEntries is the
// sweep (run periodically from dispatcher) that reclaims entries whose
// recorded expiry has already passed.
func TestClient_cleanupExpiredEntries(t *testing.T) {
	c := &Client{
		responseChans:       make(map[uint32]chan *Response),
		responseChanInfoMap: make(map[uint32]string),
		responseChanExpiry:  make(map[uint32]time.Time),
	}

	// id 1: its caller's timeout has already elapsed and no response ever
	// arrived - this is the entry that used to leak forever.
	c.responseChans[1] = make(chan *Response)
	c.responseChanInfoMap[1] = "method: expired_call"
	c.responseChanExpiry[1] = time.Now().Add(-time.Second)

	// id 2: still within its timeout window - must be left alone, since its
	// caller may still be legitimately waiting for a response.
	c.responseChans[2] = make(chan *Response)
	c.responseChanInfoMap[2] = "method: still_pending_call"
	c.responseChanExpiry[2] = time.Now().Add(time.Hour)

	c.cleanupExpiredEntries()

	if _, ok := c.responseChans[1]; ok {
		t.Error("expired entry 1 should have been removed from responseChans")
	}
	if _, ok := c.responseChanInfoMap[1]; ok {
		t.Error("expired entry 1 should have been removed from responseChanInfoMap")
	}
	if _, ok := c.responseChanExpiry[1]; ok {
		t.Error("expired entry 1 should have been removed from responseChanExpiry")
	}

	if _, ok := c.responseChans[2]; !ok {
		t.Error("still-pending entry 2 should not have been removed from responseChans")
	}
	if _, ok := c.responseChanInfoMap[2]; !ok {
		t.Error("still-pending entry 2 should not have been removed from responseChanInfoMap")
	}
	if _, ok := c.responseChanExpiry[2]; !ok {
		t.Error("still-pending entry 2 should not have been removed from responseChanExpiry")
	}
}

// TestClient_handleSend_recordsExpiry confirms handleSend wires each
// request's own timeout (carried on messageWrapper) into
// responseChanExpiry, so cleanupExpiredEntries can later tell an abandoned
// request apart from one that is merely slow but still within its caller's
// configured timeout (e.g. DefaultLongTimeout).
func TestClient_handleSend_recordsExpiry(t *testing.T) {
	c := &Client{
		encoder:             json.NewEncoder(io.Discard),
		responseChans:       make(map[uint32]chan *Response),
		responseChanInfoMap: make(map[uint32]string),
		responseChanExpiry:  make(map[uint32]time.Time),
	}

	const timeout = 5 * time.Minute
	before := time.Now()
	c.handleSend(&messageWrapper{
		method:       "test_method",
		responseChan: make(chan *Response, 1),
		timeout:      timeout,
	})
	after := time.Now()

	expiry, ok := c.responseChanExpiry[0]
	if !ok {
		t.Fatal("handleSend should have recorded an expiry for the new request")
	}
	if expiry.Before(before.Add(timeout)) || expiry.After(after.Add(timeout)) {
		t.Errorf("expiry %v not within expected window [%v, %v]", expiry, before.Add(timeout), after.Add(timeout))
	}
}
