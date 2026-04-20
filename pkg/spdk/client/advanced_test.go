package client

import (
	"testing"

	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
)

func TestDetectAddressFamily(t *testing.T) {
	testCases := []struct {
		name     string
		ip       string
		expected spdktypes.NvmeAddressFamily
	}{
		{"IPv4", "192.168.1.1", spdktypes.NvmeAddressFamilyIPv4},
		{"IPv6", "fd00::1", spdktypes.NvmeAddressFamilyIPv6},
		{"bracketed IPv6", "[fd00::1]", spdktypes.NvmeAddressFamilyIPv6},
		{"IPv6 loopback", "::1", spdktypes.NvmeAddressFamilyIPv6},
		{"empty", "", spdktypes.NvmeAddressFamilyIPv4},
		{"malformed", "not-an-ip", spdktypes.NvmeAddressFamilyIPv4},
		{"IPv4-mapped v6", "::ffff:10.0.0.1", spdktypes.NvmeAddressFamilyIPv4},
		{"bracketed IPv4-mapped v6", "[::ffff:10.0.0.1]", spdktypes.NvmeAddressFamilyIPv4},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := DetectAddressFamily(tc.ip)
			if got != tc.expected {
				t.Errorf("DetectAddressFamily(%q) = %q, want %q", tc.ip, got, tc.expected)
			}
		})
	}
}
