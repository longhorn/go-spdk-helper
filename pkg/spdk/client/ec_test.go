package client

import (
	"reflect"
	"testing"
)

func TestBdevEcRPCRequests(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		call       func(*Client) error
		expectKeys map[string]any
		absentKeys []string
		result     any
	}{
		{
			name:   "BdevEcCreate without salvage",
			method: "bdev_ec_create",
			call: func(cli *Client) error {
				_, err := cli.BdevEcCreate("ec0", 4, 2, 64, []string{"b0", "b1", "b2", "b3", "b4", "b5"}, false)
				return err
			},
			expectKeys: map[string]any{
				"name":               "ec0",
				"data_chunk_count":   float64(4),
				"parity_chunk_count": float64(2),
				"strip_size_kb":      float64(64),
				"base_bdevs":         []any{"b0", "b1", "b2", "b3", "b4", "b5"},
			},
			absentKeys: []string{"salvage_requested"},
			result:     true,
		},
		{
			name:   "BdevEcCreate with salvage",
			method: "bdev_ec_create",
			call: func(cli *Client) error {
				_, err := cli.BdevEcCreate("ec0", 4, 2, 64, []string{"b0", "b1", "b2", "b3", "b4", "b5"}, true)
				return err
			},
			expectKeys: map[string]any{"salvage_requested": true},
			result:     true,
		},
		{
			name:   "BdevEcDelete",
			method: "bdev_ec_delete",
			call: func(cli *Client) error {
				_, err := cli.BdevEcDelete("ec0")
				return err
			},
			expectKeys: map[string]any{"name": "ec0"},
			absentKeys: []string{"ec_name"},
			result:     true,
		},
		{
			name:   "BdevEcGetBdevs without name",
			method: "bdev_ec_get_bdevs",
			call: func(cli *Client) error {
				_, err := cli.BdevEcGetBdevs("")
				return err
			},
			absentKeys: []string{"name"},
			result:     []map[string]any{},
		},
		{
			name:   "BdevEcGetBdevs with name",
			method: "bdev_ec_get_bdevs",
			call: func(cli *Client) error {
				_, err := cli.BdevEcGetBdevs("ec0")
				return err
			},
			expectKeys: map[string]any{"name": "ec0"},
			result:     []map[string]any{},
		},
		{
			name:   "BdevEcReplaceBaseBdev",
			method: "bdev_ec_replace_base_bdev",
			call: func(cli *Client) error {
				_, err := cli.BdevEcReplaceBaseBdev("ec0", 3, "n1")
				return err
			},
			expectKeys: map[string]any{
				"ec_name":       "ec0",
				"slot":          float64(3),
				"new_bdev_name": "n1",
			},
			absentKeys: []string{"name"},
			result:     map[string]any{},
		},
		{
			name:   "BdevEcStartRebuild",
			method: "bdev_ec_start_rebuild",
			call: func(cli *Client) error {
				_, err := cli.BdevEcStartRebuild("ec0")
				return err
			},
			expectKeys: map[string]any{"ec_name": "ec0"},
			absentKeys: []string{"name"},
			result:     map[string]any{},
		},
		{
			name:   "BdevEcGetRebuildProgress",
			method: "bdev_ec_get_rebuild_progress",
			call: func(cli *Client) error {
				_, err := cli.BdevEcGetRebuildProgress("ec0")
				return err
			},
			expectKeys: map[string]any{"ec_name": "ec0"},
			absentKeys: []string{"name"},
			result:     map[string]any{},
		},
		{
			name:   "BdevEcStopRebuild",
			method: "bdev_ec_stop_rebuild",
			call: func(cli *Client) error {
				_, err := cli.BdevEcStopRebuild("ec0")
				return err
			},
			expectKeys: map[string]any{"ec_name": "ec0"},
			absentKeys: []string{"name"},
			result:     true,
		},
		{
			name:   "BdevEcSetRebuildQos",
			method: "bdev_ec_set_rebuild_qos",
			call: func(cli *Client) error {
				_, err := cli.BdevEcSetRebuildQos("ec0", 5000, true)
				return err
			},
			expectKeys: map[string]any{
				"ec_name":             "ec0",
				"max_stripes_per_sec": float64(5000),
				"paused":              true,
			},
			result: true,
		},
		{
			name:   "BdevEcResize",
			method: "bdev_ec_resize",
			call: func(cli *Client) error {
				_, err := cli.BdevEcResize("ec0")
				return err
			},
			expectKeys: map[string]any{"ec_name": "ec0"},
			absentKeys: []string{"name"},
			result:     map[string]any{},
		},
		{
			name:   "BdevEcGetWibStatus",
			method: "bdev_ec_get_wib_status",
			call: func(cli *Client) error {
				_, err := cli.BdevEcGetWibStatus("ec0")
				return err
			},
			expectKeys: map[string]any{"ec_name": "ec0"},
			absentKeys: []string{"name"},
			result:     map[string]any{},
		},
		{
			name:   "BdevEcGetUnmapStatus",
			method: "bdev_ec_get_unmap_status",
			call: func(cli *Client) error {
				_, err := cli.BdevEcGetUnmapStatus("ec0")
				return err
			},
			expectKeys: map[string]any{"ec_name": "ec0"},
			absentKeys: []string{"name"},
			result:     map[string]any{},
		},
		{
			name:   "BdevEcGetScrubProgress",
			method: "bdev_ec_get_scrub_progress",
			call: func(cli *Client) error {
				_, err := cli.BdevEcGetScrubProgress("ec0")
				return err
			},
			expectKeys: map[string]any{"ec_name": "ec0"},
			absentKeys: []string{"name"},
			result:     map[string]any{},
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
