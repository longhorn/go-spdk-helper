package client

import (
	"fmt"
	"reflect"
	"testing"

	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
)

func TestBdevRaidRPCRequests(t *testing.T) {
	resizeOff := false
	resizeOn := true
	windowSizeKb := uint32(1024)
	maxBandwidthMbSec := uint32(100)
	noBandwidthLimit := uint32(0)

	cases := []struct {
		name       string
		method     string
		call       func(*Client) error
		expectKeys map[string]any
		absentKeys []string
		result     any
	}{
		{
			name:   "BdevRaidResize",
			method: "bdev_raid_resize",
			call: func(cli *Client) error {
				resp, err := cli.BdevRaidResize("raid0")
				if err != nil {
					return err
				}
				want := spdktypes.BdevRaidResizeResponse{Name: "raid0", OldBlockcnt: 100, NewBlockcnt: 200, Resized: true}
				if resp != want {
					return fmt.Errorf("unexpected response %+v, want %+v", resp, want)
				}
				return nil
			},
			expectKeys: map[string]any{"name": "raid0"},
			absentKeys: []string{"raid_name"},
			result: map[string]any{
				"name":         "raid0",
				"old_blockcnt": 100,
				"new_blockcnt": 200,
				"resized":      true,
			},
		},
		{
			name:   "BdevRaidSetOptions disables the automatic resize only",
			method: "bdev_raid_set_options",
			call: func(cli *Client) error {
				_, err := cli.BdevRaidSetOptions(spdktypes.BdevRaidSetOptionsRequest{ResizeOnBaseBdevResize: &resizeOff})
				return err
			},
			expectKeys: map[string]any{"resize_on_base_bdev_resize": false},
			absentKeys: []string{"process_window_size_kb", "process_max_bandwidth_mb_sec"},
			result:     true,
		},
		{
			name:   "BdevRaidSetOptions with every option",
			method: "bdev_raid_set_options",
			call: func(cli *Client) error {
				_, err := cli.BdevRaidSetOptions(spdktypes.BdevRaidSetOptionsRequest{
					ProcessWindowSizeKb:      &windowSizeKb,
					ProcessMaxBandwidthMbSec: &maxBandwidthMbSec,
					ResizeOnBaseBdevResize:   &resizeOn,
				})
				return err
			},
			expectKeys: map[string]any{
				"process_window_size_kb":       float64(1024),
				"process_max_bandwidth_mb_sec": float64(100),
				"resize_on_base_bdev_resize":   true,
			},
			result: true,
		},
		{
			name:   "BdevRaidSetOptions sends an explicit zero bandwidth to remove the limit",
			method: "bdev_raid_set_options",
			call: func(cli *Client) error {
				_, err := cli.BdevRaidSetOptions(spdktypes.BdevRaidSetOptionsRequest{ProcessMaxBandwidthMbSec: &noBandwidthLimit})
				return err
			},
			expectKeys: map[string]any{"process_max_bandwidth_mb_sec": float64(0)},
			absentKeys: []string{"process_window_size_kb", "resize_on_base_bdev_resize"},
			result:     true,
		},
		{
			name:   "BdevRaidSetOptions without options",
			method: "bdev_raid_set_options",
			call: func(cli *Client) error {
				_, err := cli.BdevRaidSetOptions(spdktypes.BdevRaidSetOptionsRequest{})
				return err
			},
			absentKeys: []string{"process_window_size_kb", "process_max_bandwidth_mb_sec", "resize_on_base_bdev_resize"},
			result:     true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runJSONRPCRequestTest(t,
				tc.call,
				func(t *testing.T, method string, params map[string]any) {
					t.Helper()
					if method != tc.method {
						t.Fatalf("unexpected method %s, want %s", method, tc.method)
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
				tc.result,
			)
		})
	}
}
