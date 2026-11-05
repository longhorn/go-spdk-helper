package basic

import (
	"context"
	"io"
	"testing"

	"github.com/urfave/cli/v3"

	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
)

// runBdevRaidSetOptions parses args with the flags of the set-options command and returns
// the request it would send, without connecting to SPDK.
func runBdevRaidSetOptions(t *testing.T, args ...string) (spdktypes.BdevRaidSetOptionsRequest, error) {
	t.Helper()

	var req spdktypes.BdevRaidSetOptionsRequest
	cmd := BdevRaidSetOptionsCmd()
	cmd.Writer = io.Discard
	cmd.ErrWriter = io.Discard
	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		req = bdevRaidSetOptionsRequest(c)
		return nil
	}
	err := cmd.Run(context.Background(), append([]string{"set-options"}, args...))
	return req, err
}

func TestBdevRaidSetOptionsRequest(t *testing.T) {
	req, err := runBdevRaidSetOptions(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.ProcessWindowSizeKb != nil || req.ProcessMaxBandwidthMbSec != nil || req.ResizeOnBaseBdevResize != nil {
		t.Fatalf("expected no option without flags, got %+v", req)
	}

	req, err = runBdevRaidSetOptions(t, "--process-max-bandwidth-mb-sec=0", "--resize-on-base-bdev-resize=false")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.ProcessMaxBandwidthMbSec == nil || *req.ProcessMaxBandwidthMbSec != 0 {
		t.Fatalf("expected an explicit zero bandwidth, got %+v", req.ProcessMaxBandwidthMbSec)
	}
	if req.ResizeOnBaseBdevResize == nil || *req.ResizeOnBaseBdevResize {
		t.Fatalf("expected an explicit false, got %+v", req.ResizeOnBaseBdevResize)
	}
	if req.ProcessWindowSizeKb != nil {
		t.Fatalf("expected no window size, got %v", *req.ProcessWindowSizeKb)
	}

	req, err = runBdevRaidSetOptions(t, "--process-window-size-kb=4294967295", "--process-max-bandwidth-mb-sec=100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.ProcessWindowSizeKb == nil || *req.ProcessWindowSizeKb != 4294967295 {
		t.Fatalf("expected window size 4294967295, got %+v", req.ProcessWindowSizeKb)
	}
	if req.ProcessMaxBandwidthMbSec == nil || *req.ProcessMaxBandwidthMbSec != 100 {
		t.Fatalf("expected bandwidth 100, got %+v", req.ProcessMaxBandwidthMbSec)
	}
}

func TestBdevRaidSetOptionsRejectsOutOfRangeValues(t *testing.T) {
	for _, arg := range []string{
		"--process-window-size-kb=4294967296",
		"--process-max-bandwidth-mb-sec=4294967297",
		"--process-max-bandwidth-mb-sec=-1",
	} {
		if _, err := runBdevRaidSetOptions(t, arg); err == nil {
			t.Fatalf("expected an error for %s", arg)
		}
	}
}
