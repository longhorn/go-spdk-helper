package jsonrpc

import (
	"errors"
	"testing"
)

func TestIsJSONRPCRespErrorNoSuchFileOrDirectory(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "exact missing file response",
			err: JSONClientError{ErrorDetail: &ResponseError{
				Code:    RespErrorCodeInternalError,
				Message: RespErrorMsgNoSuchFileOrDirectory,
			}},
			want: true,
		},
		{
			name: "different internal error",
			err: JSONClientError{ErrorDetail: &ResponseError{
				Code:    RespErrorCodeInternalError,
				Message: "metadata unavailable",
			}},
		},
		{
			name: "same message with different code",
			err: JSONClientError{ErrorDetail: &ResponseError{
				Code:    RespErrorCodeNoEntry,
				Message: RespErrorMsgNoSuchFileOrDirectory,
			}},
		},
		{
			name: "client timeout",
			err:  JSONClientError{ErrorDetail: errors.New("timeout waiting for response")},
		},
		{
			name: "unwrapped response error",
			err: &ResponseError{
				Code:    RespErrorCodeInternalError,
				Message: RespErrorMsgNoSuchFileOrDirectory,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsJSONRPCRespErrorNoSuchFileOrDirectory(test.err); got != test.want {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}
