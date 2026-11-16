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
			name:   "BdevRaidCreateKeepsAutoResize",
			method: "bdev_raid_create",
			call: func(cli *Client) error {
				_, err := cli.BdevRaidCreate("raid0", spdktypes.BdevRaidLevelRaid1, 0, []string{"a", "b"}, "")
				return err
			},
			expectKeys: map[string]any{"name": "raid0", "raid_level": "raid1"},
			absentKeys: []string{"disable_auto_resize"},
			result:     true,
		},
		{
			name:   "BdevRaidCreateWithOptionsDefault",
			method: "bdev_raid_create",
			call: func(cli *Client) error {
				_, err := cli.BdevRaidCreateWithOptions("raid0", spdktypes.BdevRaidLevelRaid1, 0, []string{"a", "b"}, "", spdktypes.BdevRaidCreateOptions{})
				return err
			},
			expectKeys: map[string]any{"name": "raid0"},
			absentKeys: []string{"disable_auto_resize"},
			result:     true,
		},
		{
			name:   "BdevRaidCreateWithOptionsDisableAutoResize",
			method: "bdev_raid_create",
			call: func(cli *Client) error {
				created, err := cli.BdevRaidCreateWithOptions("raid0", spdktypes.BdevRaidLevelRaid1, 0, []string{"a", "b"}, "uuid0",
					spdktypes.BdevRaidCreateOptions{DisableAutoResize: true})
				if err != nil {
					return err
				}
				if !created {
					return fmt.Errorf("unexpected result %v, want true", created)
				}
				return nil
			},
			expectKeys: map[string]any{
				"name":                "raid0",
				"raid_level":          "raid1",
				"strip_size_kb":       float64(0),
				"base_bdevs":          []any{"a", "b"},
				"uuid":                "uuid0",
				"disable_auto_resize": true,
			},
			result: true,
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
