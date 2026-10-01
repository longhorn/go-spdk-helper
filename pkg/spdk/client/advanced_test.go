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

func TestSharedTransportIobufCacheSize(t *testing.T) {
	testCases := []struct {
		name           string
		small, large   uint64
		pollGroupCount int
		wantSmall      uint32
		wantLarge      uint32
	}{
		{"defaults with 8 poll groups", 8192, 1024, 8, 256, 32},
		{"defaults with 2 poll groups", 8192, 1024, 2, 1024, 128},
		{"larger pools", 32768, 4096, 8, 1024, 128},
		{"unknown poll group count", 8192, 1024, 0, 2048, 256},
		{"tiny pools never go to zero", 16, 2, 8, 1, 1},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			small, large := sharedTransportIobufCacheSize(&spdktypes.IobufOptions{SmallPoolCount: tc.small, LargePoolCount: tc.large}, tc.pollGroupCount)
			if small != tc.wantSmall || large != tc.wantLarge {
				t.Errorf("sharedTransportIobufCacheSize(%v, %v, %v) = %v/%v, want %v/%v", tc.small, tc.large, tc.pollGroupCount, small, large, tc.wantSmall, tc.wantLarge)
			}
		})
	}
}
