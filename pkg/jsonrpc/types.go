package jsonrpc

import (
	"fmt"
)

type Message struct {
	ID      uint32      `json:"id"`
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

func NewMessage(id uint32, method string, params interface{}) *Message {
	return &Message{
		ID:      id,
		Version: "2.0",
		Method:  method,
		Params:  params,
	}
}

type RespErrorMsg string

type RespErrorCode int32

const (
	RespErrorCodeNoSuchDevice = -19
)

type Response struct {
	ID        uint32         `json:"id"`
	Version   string         `json:"jsonrpc"`
	Result    interface{}    `json:"result,omitempty"`
	ErrorInfo *ResponseError `json:"error,omitempty"`
}

func (re ResponseError) Error() string {
	return fmt.Sprintf("{\"code\": %d,\"message\": \"%s\"}", re.Code, re.Message)
}

type ResponseError struct {
	Code    RespErrorCode `json:"code"`
	Message RespErrorMsg  `json:"message"`
}

type JSONClientError struct {
	ID          uint32
	Method      string
	Params      interface{}
	ErrorDetail error
}

func (re JSONClientError) Error() string {
	return fmt.Sprintf("error sending message, id %d, method %s, params %v: %v",
		re.ID, re.Method, re.Params, re.ErrorDetail)
}

func IsJSONRPCRespErrorNoSuchDevice(err error) bool {
	jsonRPCError, ok := err.(JSONClientError)
	if !ok {
		return false
	}
	responseError, ok := jsonRPCError.ErrorDetail.(*ResponseError)
	if !ok {
		return false
	}

	return responseError.Code == RespErrorCodeNoSuchDevice
}
