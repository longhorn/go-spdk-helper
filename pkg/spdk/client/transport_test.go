package client

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/longhorn/go-spdk-helper/pkg/jsonrpc"

	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
)

// fakeTransportTarget mimics how spdk_tgt handles nvmf_create_transport: the "already exists"
// check happens when the RPC is received, but the transport only becomes visible after a delay.
type fakeTransportTarget struct {
	mu         sync.Mutex
	transports []spdktypes.NvmfTransport
	creates    atomic.Int32
	// createErr makes nvmf_create_transport add the transport but fail with this error, like spdk_tgt
	// does when it finds a duplicate only after the transport setup.
	createErr *jsonrpc.ResponseError
	// createFailsWithoutTransport makes the failing create leave no transport behind.
	createFailsWithoutTransport bool
}

func (f *fakeTransportTarget) handle(method string) (interface{}, *jsonrpc.ResponseError) {
	switch method {
	case "nvmf_get_transports":
		f.mu.Lock()
		defer f.mu.Unlock()
		return append([]spdktypes.NvmfTransport{}, f.transports...), nil
	case "nvmf_create_transport":
		// Count every request, including the ones rejected because the transport exists.
		f.creates.Add(1)
		f.mu.Lock()
		exists := len(f.transports) > 0
		f.mu.Unlock()
		if exists {
			return nil, nil
		}
		time.Sleep(50 * time.Millisecond)
		if f.createErr != nil && f.createFailsWithoutTransport {
			return nil, f.createErr
		}
		f.mu.Lock()
		f.transports = append(f.transports, spdktypes.NvmfTransport{Trtype: spdktypes.NvmeTransportTypeTCP})
		f.mu.Unlock()
		return true, f.createErr
	case "nvmf_subsystem_add_ns":
		return uint32(1), nil
	default:
		return true, nil
	}
}

func newFakeTransportClient(t *testing.T, target *fakeTransportTarget) *Client {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		_ = serverConn.Close()
		_ = clientConn.Close()
	})

	go func() {
		decoder := json.NewDecoder(serverConn)
		encoder := json.NewEncoder(serverConn)
		var writeMu sync.Mutex
		for {
			var msg jsonrpc.Message
			if err := decoder.Decode(&msg); err != nil {
				return
			}
			go func() {
				result, respErr := target.handle(msg.Method)
				writeMu.Lock()
				defer writeMu.Unlock()
				_ = encoder.Encode(&jsonrpc.Response{ID: msg.ID, Version: "2.0", Result: result, ErrorInfo: respErr})
			}()
		}
	}()

	// The JSON-RPC client goroutines only exit when their context is canceled.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	return &Client{
		conn:    clientConn,
		jsonCli: jsonrpc.NewClient(ctx, clientConn),
	}
}

func TestStartExposeBdevCreatesOneTransportWhenCalledConcurrently(t *testing.T) {
	target := &fakeTransportTarget{}
	cli := newFakeTransportClient(t, target)

	const callers = 8
	errs := make(chan error, callers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			if i%2 == 0 {
				errs <- cli.StartExposeBdev("nqn.test:"+string(rune('a'+i)), "bdev", "nguid", "10.0.0.1", "20001")
				return
			}
			errs <- cli.StartExposeBdevWithANAState("nqn.test:"+string(rune('a'+i)), "bdev", "nguid", "", "10.0.0.1", "20001",
				spdktypes.NvmfSubsystemListenerAnaStateOptimized, 0, 0)
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("expose failed: %v", err)
		}
	}

	if got := target.creates.Load(); got != 1 {
		t.Fatalf("nvmf_create_transport was requested %d times, want 1", got)
	}
	if got := len(target.transports); got != 1 {
		t.Fatalf("target has %d transports, want 1", got)
	}
}

func TestEnsureNvmfTransport(t *testing.T) {
	tests := []struct {
		name        string
		newTarget   func() *fakeTransportTarget
		wantErr     bool
		wantCreates int32
	}{
		{
			name: "skips the create when a transport exists",
			newTarget: func() *fakeTransportTarget {
				return &fakeTransportTarget{transports: []spdktypes.NvmfTransport{{Trtype: spdktypes.NvmeTransportTypeTCP}}}
			},
			wantCreates: 0,
		},
		{
			name: "ignores a duplicate reported after the transport setup",
			newTarget: func() *fakeTransportTarget {
				return &fakeTransportTarget{createErr: &jsonrpc.ResponseError{
					Code:    -32603,
					Message: "Failed to add transport to tgt.(-17)",
				}}
			},
			wantCreates: 1,
		},
		{
			name: "returns the create error when no transport exists",
			newTarget: func() *fakeTransportTarget {
				return &fakeTransportTarget{
					createErr: &jsonrpc.ResponseError{
						Code:    -32603,
						Message: "Failed to create transport.",
					},
					createFailsWithoutTransport: true,
				}
			},
			wantErr:     true,
			wantCreates: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			target := tc.newTarget()
			cli := newFakeTransportClient(t, target)

			if err := cli.EnsureNvmfTransport(); (err != nil) != tc.wantErr {
				t.Fatalf("EnsureNvmfTransport error = %v, want error %v", err, tc.wantErr)
			}
			if got := target.creates.Load(); got != tc.wantCreates {
				t.Fatalf("nvmf_create_transport was requested %d times, want %d", got, tc.wantCreates)
			}
		})
	}
}
