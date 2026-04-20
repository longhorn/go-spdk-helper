package util

import (
	"testing"
)

func TestNormalizeNvmeAddr(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"bare IPv6", "::1", "::1"},
		{"bracketed IPv6", "[::1]", "::1"},
		{"IPv4", "10.0.0.1", "10.0.0.1"},
		{"bracketed fd00", "[fd00::1]", "fd00::1"},
		{"empty", "", ""},
		{"single bracket", "[", "["},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeNvmeAddr(tc.input)
			if got != tc.expected {
				t.Errorf("NormalizeNvmeAddr(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}
