package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
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

		var params map[string]interface{}
		if msg.Params != nil {
			var ok bool
			params, ok = msg.Params.(map[string]interface{})
			if !ok {
				serverErrCh <- nil
				t.Errorf("unexpected params type %T", msg.Params)
				return
			}
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

type jsonRPCScriptStep struct {
	method        string
	params        map[string]interface{}
	result        interface{}
	responseError *jsonrpc.ResponseError
}

func runJSONRPCScriptTest(t *testing.T, steps []jsonRPCScriptStep, fn func(*Client)) {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	cleanup := func() {
		cancel()
		_ = serverConn.Close()
		_ = clientConn.Close()
	}
	defer cleanup()

	serverErrCh := make(chan error, 1)
	go func() {
		defer close(serverErrCh)
		scriptFailed := true
		defer func() {
			if scriptFailed {
				cancel()
			}
		}()

		decoder := json.NewDecoder(serverConn)
		encoder := json.NewEncoder(serverConn)
		for i, step := range steps {
			var msg jsonrpc.Message
			if err := decoder.Decode(&msg); err != nil {
				serverErrCh <- fmt.Errorf("step %d: failed to decode request: %w", i, err)
				return
			}
			if msg.Method != step.method {
				serverErrCh <- fmt.Errorf("step %d: got method %q, want %q", i, msg.Method, step.method)
				return
			}

			var params map[string]interface{}
			if msg.Params != nil {
				var ok bool
				params, ok = msg.Params.(map[string]interface{})
				if !ok {
					serverErrCh <- fmt.Errorf("step %d: unexpected params type %T", i, msg.Params)
					return
				}
			}
			if !reflect.DeepEqual(params, step.params) {
				serverErrCh <- fmt.Errorf("step %d: got params %#v, want %#v", i, params, step.params)
				return
			}

			response := &jsonrpc.Response{
				ID:        msg.ID,
				Version:   "2.0",
				Result:    step.result,
				ErrorInfo: step.responseError,
			}
			if err := encoder.Encode(response); err != nil {
				serverErrCh <- fmt.Errorf("step %d: failed to encode response: %w", i, err)
				return
			}
		}

		scriptFailed = false
		serverErrCh <- nil
	}()

	cli := &Client{
		conn:    clientConn,
		jsonCli: jsonrpc.NewClient(ctx, clientConn),
	}
	fn(cli)

	cleanup()
	if err := <-serverErrCh; err != nil {
		t.Fatalf("scripted server failed: %v", err)
	}
}

func testLvol(name string, snapshot bool) spdktypes.BdevInfo {
	return spdktypes.BdevInfo{
		BdevInfoBasic: spdktypes.BdevInfoBasic{
			Name:        name,
			ProductName: spdktypes.BdevProductNameLvol,
		},
		DriverSpecific: &spdktypes.BdevDriverSpecific{
			Lvol: &spdktypes.BdevDriverSpecificLvol{Snapshot: snapshot},
		},
	}
}

func listLvolsStep(lvols ...spdktypes.BdevInfo) jsonRPCScriptStep {
	return jsonRPCScriptStep{
		method: "bdev_get_bdevs",
		result: lvols,
	}
}

func getXattrStep(lvolName, xattrName, result string, responseError *jsonrpc.ResponseError) jsonRPCScriptStep {
	return jsonRPCScriptStep{
		method: "bdev_lvol_get_xattr",
		params: map[string]interface{}{
			"name":       lvolName,
			"xattr_name": xattrName,
		},
		result:        result,
		responseError: responseError,
	}
}

func getSnapshotChecksumStep(lvolName string, responseError *jsonrpc.ResponseError) jsonRPCScriptStep {
	return jsonRPCScriptStep{
		method:        "bdev_lvol_get_snapshot_checksum",
		params:        map[string]interface{}{"name": lvolName},
		responseError: responseError,
	}
}

func TestBdevLvolGetWithFilterHandlesExpectedMetadata(t *testing.T) {
	missingXattr := &jsonrpc.ResponseError{
		Code:    jsonrpc.RespErrorCodeInternalError,
		Message: jsonrpc.RespErrorMsgNoSuchFileOrDirectory,
	}
	checksumNotRegistered := &jsonrpc.ResponseError{
		Code:    jsonrpc.RespErrorCodeNoSuchDevice,
		Message: "No such device",
	}

	tests := []struct {
		name       string
		lvol       spdktypes.BdevInfo
		steps      []jsonRPCScriptStep
		wantXattrs map[string]string
	}{
		{
			name: "missing user-created defaults to true",
			lvol: testLvol("lvol-missing-user-created", false),
			steps: []jsonRPCScriptStep{
				getXattrStep("lvol-missing-user-created", UserCreated, "", missingXattr),
				getXattrStep("lvol-missing-user-created", SnapshotTimestamp, "2026-07-16T04:13:30Z", nil),
			},
			wantXattrs: map[string]string{
				UserCreated:       "true",
				SnapshotTimestamp: "2026-07-16T04:13:30Z",
			},
		},
		{
			name: "missing timestamp is explicitly empty",
			lvol: testLvol("lvol-missing-timestamp", false),
			steps: []jsonRPCScriptStep{
				getXattrStep("lvol-missing-timestamp", UserCreated, "false", nil),
				getXattrStep("lvol-missing-timestamp", SnapshotTimestamp, "", missingXattr),
			},
			wantXattrs: map[string]string{
				UserCreated:       "false",
				SnapshotTimestamp: "",
			},
		},
		{
			name: "unregistered snapshot checksum is omitted",
			lvol: testLvol("snapshot-without-checksum", true),
			steps: []jsonRPCScriptStep{
				getXattrStep("snapshot-without-checksum", UserCreated, "true", nil),
				getXattrStep("snapshot-without-checksum", SnapshotTimestamp, "2026-07-16T04:13:30Z", nil),
				getSnapshotChecksumStep("snapshot-without-checksum", checksumNotRegistered),
			},
			wantXattrs: map[string]string{
				UserCreated:       "true",
				SnapshotTimestamp: "2026-07-16T04:13:30Z",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			steps := append([]jsonRPCScriptStep{listLvolsStep(test.lvol)}, test.steps...)
			runJSONRPCScriptTest(t, steps, func(cli *Client) {
				got, err := cli.BdevLvolGetWithFilter("", 0, func(*spdktypes.BdevInfo) bool { return true })
				if err != nil {
					t.Fatalf("BdevLvolGetWithFilter failed: %v", err)
				}
				if len(got) != 1 {
					t.Fatalf("got %d lvols, want 1", len(got))
				}
				if !reflect.DeepEqual(got[0].DriverSpecific.Lvol.Xattrs, test.wantXattrs) {
					t.Fatalf("got xattrs %#v, want %#v", got[0].DriverSpecific.Lvol.Xattrs, test.wantXattrs)
				}
			})
		})
	}
}

func TestBdevLvolGetWithFilterRejectsUnexpectedMetadataErrors(t *testing.T) {
	unexpectedError := &jsonrpc.ResponseError{
		Code:    jsonrpc.RespErrorCodeNoEntry,
		Message: "metadata unavailable",
	}

	tests := []struct {
		name  string
		steps []jsonRPCScriptStep
	}{
		{
			name: "user-created error",
			steps: []jsonRPCScriptStep{
				listLvolsStep(testLvol("lvol-user-created-error", false)),
				getXattrStep("lvol-user-created-error", UserCreated, "", unexpectedError),
			},
		},
		{
			name: "timestamp error returns no partially enriched lvols",
			steps: []jsonRPCScriptStep{
				listLvolsStep(testLvol("lvol-complete", false), testLvol("lvol-timestamp-error", false)),
				getXattrStep("lvol-complete", UserCreated, "true", nil),
				getXattrStep("lvol-complete", SnapshotTimestamp, "2026-07-16T04:13:29Z", nil),
				getXattrStep("lvol-timestamp-error", UserCreated, "true", nil),
				getXattrStep("lvol-timestamp-error", SnapshotTimestamp, "", unexpectedError),
			},
		},
		{
			name: "snapshot checksum error",
			steps: []jsonRPCScriptStep{
				listLvolsStep(testLvol("snapshot-checksum-error", true)),
				getXattrStep("snapshot-checksum-error", UserCreated, "true", nil),
				getXattrStep("snapshot-checksum-error", SnapshotTimestamp, "2026-07-16T04:13:30Z", nil),
				getSnapshotChecksumStep("snapshot-checksum-error", unexpectedError),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runJSONRPCScriptTest(t, test.steps, func(cli *Client) {
				got, err := cli.BdevLvolGetWithFilter("", 0, func(*spdktypes.BdevInfo) bool { return true })
				if err == nil {
					t.Fatal("BdevLvolGetWithFilter unexpectedly succeeded")
				}
				if got != nil {
					t.Fatalf("got partial lvol list %#v, want nil", got)
				}
				if !jsonrpc.IsJSONRPCRespErrorNoEntry(err) {
					t.Fatalf("got unclassifiable error %v", err)
				}
			})
		})
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

func TestBdevNvmeResetControllerSendsCorrectMethod(t *testing.T) {
	runJSONRPCRequestTest(t,
		func(cli *Client) error {
			_, err := cli.BdevNvmeResetController("Nvme0")
			return err
		},
		func(t *testing.T, method string, params map[string]interface{}) {
			t.Helper()
			if method != "bdev_nvme_reset_controller" {
				t.Fatalf("unexpected method %s", method)
			}
			if params["name"] != "Nvme0" {
				t.Fatalf("expected name Nvme0, got %#v", params["name"])
			}
		},
		true,
	)
}

func TestBdevLvolGrowLvstoreRPCRequests(t *testing.T) {
	cases := []struct {
		name       string
		call       func(*Client) error
		expectKeys map[string]any
		absentKeys []string
	}{
		{
			name: "BdevLvolGrowLvstore with lvs-name",
			call: func(cli *Client) error {
				_, err := cli.BdevLvolGrowLvstore("lvs0", "")
				return err
			},
			expectKeys: map[string]any{"lvs_name": "lvs0"},
			absentKeys: []string{"uuid"},
		},
		{
			name: "BdevLvolGrowLvstore with uuid",
			call: func(cli *Client) error {
				_, err := cli.BdevLvolGrowLvstore("", "abc-uuid")
				return err
			},
			expectKeys: map[string]any{"uuid": "abc-uuid"},
			absentKeys: []string{"lvs_name"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runJSONRPCRequestTest(t,
				tc.call,
				func(t *testing.T, method string, params map[string]interface{}) {
					t.Helper()
					if method != "bdev_lvol_grow_lvstore" {
						t.Fatalf("unexpected method %s", method)
					}
					for k, want := range tc.expectKeys {
						got, exists := params[k]
						if !exists {
							t.Fatalf("expected key %q present, missing", k)
						}
						if !reflect.DeepEqual(got, want) {
							t.Fatalf("key %q: got %#v, want %#v", k, got, want)
						}
					}
					for _, k := range tc.absentKeys {
						if v, exists := params[k]; exists {
							t.Fatalf("expected key %q absent, got %#v", k, v)
						}
					}
				},
				true,
			)
		})
	}
}

func TestBdevLvolCreateLvstoreRPCRequests(t *testing.T) {
	cases := []struct {
		name       string
		call       func(*Client) error
		expectKeys map[string]any
		absentKeys []string
	}{
		{
			name: "BdevLvolCreateLvstoreWithMdRatio",
			call: func(cli *Client) error {
				_, err := cli.BdevLvolCreateLvstoreWithMdRatio("bdev0", "lvs0", 4194304, 100)
				return err
			},
			expectKeys: map[string]any{
				"bdev_name":                      "bdev0",
				"lvs_name":                       "lvs0",
				"cluster_sz":                     float64(4194304),
				"num_md_pages_per_cluster_ratio": float64(100),
			},
		},
		{
			name: "BdevLvolCreateLvstore omits zero-value options",
			call: func(cli *Client) error {
				_, err := cli.BdevLvolCreateLvstore("bdev0", "lvs0", 0)
				return err
			},
			expectKeys: map[string]any{
				"bdev_name": "bdev0",
				"lvs_name":  "lvs0",
			},
			absentKeys: []string{"cluster_sz", "num_md_pages_per_cluster_ratio"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runJSONRPCRequestTest(t,
				tc.call,
				func(t *testing.T, method string, params map[string]interface{}) {
					t.Helper()
					if method != "bdev_lvol_create_lvstore" {
						t.Fatalf("unexpected method %s", method)
					}
					for k, want := range tc.expectKeys {
						got, exists := params[k]
						if !exists {
							t.Fatalf("expected key %q present, missing", k)
						}
						if !reflect.DeepEqual(got, want) {
							t.Fatalf("key %q: got %#v, want %#v", k, got, want)
						}
					}
					for _, k := range tc.absentKeys {
						if v, exists := params[k]; exists {
							t.Fatalf("expected key %q absent, got %#v", k, v)
						}
					}
				},
				"lvs-uuid-0",
			)
		})
	}
}
