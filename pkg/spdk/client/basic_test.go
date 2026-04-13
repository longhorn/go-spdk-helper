package client

import (
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/longhorn/go-spdk-helper/pkg/jsonrpc"

	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
)

func runJSONRPCRequestTest(t *testing.T, fn func(*Client) error, verify func(t *testing.T, method string, params map[string]interface{}), result interface{}) {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	defer func() {
		_ = serverConn.Close()
	}()
	defer func() {
		_ = clientConn.Close()
	}()

	serverErrCh := make(chan error, 1)
	go func() {
		defer close(serverErrCh)

		decoder := json.NewDecoder(serverConn)
		encoder := json.NewEncoder(serverConn)

		var msg jsonrpc.Message
		if err := decoder.Decode(&msg); err != nil {
			serverErrCh <- err
			return
		}

		params, ok := msg.Params.(map[string]interface{})
		if !ok {
			serverErrCh <- nil
			t.Errorf("unexpected params type %T", msg.Params)
			return
		}

		verify(t, msg.Method, params)

		if err := encoder.Encode(&jsonrpc.Response{ID: msg.ID, Version: "2.0", Result: result}); err != nil {
			serverErrCh <- err
			return
		}

		serverErrCh <- nil
	}()

	cli := &Client{
		conn:    clientConn,
		jsonCli: jsonrpc.NewClient(context.Background(), clientConn),
	}

	if err := fn(cli); err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if err := <-serverErrCh; err != nil {
		t.Fatalf("server verification failed: %v", err)
	}
}

func TestNvmfSubsystemAddNsUsesDefaultANAGroup(t *testing.T) {
	runJSONRPCRequestTest(t,
		func(cli *Client) error {
			_, err := cli.NvmfSubsystemAddNsWithUUID("nqn.test", "bdev0", "nguid0", "")
			return err
		},
		func(t *testing.T, method string, params map[string]interface{}) {
			t.Helper()
			if method != "nvmf_subsystem_add_ns" {
				t.Fatalf("unexpected method %s", method)
			}

			namespace, ok := params["namespace"].(map[string]interface{})
			if !ok {
				t.Fatalf("unexpected namespace payload %T", params["namespace"])
			}
			if _, exists := namespace["anagrpid"]; exists {
				t.Fatalf("expected add_ns to omit anagrpid, got %#v", namespace["anagrpid"])
			}
		},
		float64(1),
	)
}

func TestNvmfSubsystemListenerSetANAStateDefaultsANAGroup(t *testing.T) {
	runJSONRPCRequestTest(t,
		func(cli *Client) error {
			_, err := cli.NvmfSubsystemListenerSetANAState(
				"nqn.test",
				"10.0.0.1",
				"20006",
				spdktypes.NvmeTransportTypeTCP,
				spdktypes.NvmeAddressFamilyIPv4,
				spdktypes.NvmfSubsystemListenerAnaStateOptimized,
				0,
			)
			return err
		},
		func(t *testing.T, method string, params map[string]interface{}) {
			t.Helper()
			if method != "nvmf_subsystem_listener_set_ana_state" {
				t.Fatalf("unexpected method %s", method)
			}
			if params["anagrpid"] != float64(spdktypes.DefaultNvmfANAGroupID) {
				t.Fatalf("expected anagrpid %d, got %#v", spdktypes.DefaultNvmfANAGroupID, params["anagrpid"])
			}
		},
		true,
	)
}
