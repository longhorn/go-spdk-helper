package client

import (
	"fmt"
	"reflect"
	"testing"

	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
)

func TestBdevRaidRPCRequests(t *testing.T) {
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
				resp, err := cli.BdevRaidResize("raid0", 0)
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
			absentKeys: []string{"raid_name", "expected_size_in_mib"},
			result: map[string]any{
				"name":         "raid0",
				"old_blockcnt": 100,
				"new_blockcnt": 200,
				"resized":      true,
			},
		},
		{
			name:   "BdevRaidSetRaidOptionsExplicitResizeTrue",
			method: "bdev_raid_set_raid_options",
			call: func(cli *Client) error {
				explicitResize := true
				ok, err := cli.BdevRaidSetRaidOptions(spdktypes.BdevRaidSetRaidOptionsRequest{Name: "raid0", ExplicitResize: &explicitResize})
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("unexpected result %v, want true", ok)
				}
				return nil
			},
			expectKeys: map[string]any{"name": "raid0", "explicit_resize": true},
			result:     true,
		},
		{
			name:   "BdevRaidSetRaidOptionsExplicitResizeFalse",
			method: "bdev_raid_set_raid_options",
			call: func(cli *Client) error {
				explicitResize := false
				_, err := cli.BdevRaidSetRaidOptions(spdktypes.BdevRaidSetRaidOptionsRequest{Name: "raid0", ExplicitResize: &explicitResize})
				return err
			},
			expectKeys: map[string]any{"name": "raid0", "explicit_resize": false},
			result:     true,
		},
		{
			name:   "BdevRaidSetRaidOptionsUnset",
			method: "bdev_raid_set_raid_options",
			call: func(cli *Client) error {
				_, err := cli.BdevRaidSetRaidOptions(spdktypes.BdevRaidSetRaidOptionsRequest{Name: "raid0"})
				return err
			},
			expectKeys: map[string]any{"name": "raid0"},
			absentKeys: []string{"explicit_resize"},
			result:     true,
		},
		{
			name:   "BdevRaidResizeExpectedSize",
			method: "bdev_raid_resize",
			call: func(cli *Client) error {
				_, err := cli.BdevRaidResize("raid0", 1024)
				return err
			},
			expectKeys: map[string]any{"name": "raid0", "expected_size_in_mib": float64(1024)},
			result: map[string]any{
				"name":         "raid0",
				"old_blockcnt": 100,
				"new_blockcnt": 262144,
				"resized":      true,
			},
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
