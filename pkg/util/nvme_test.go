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
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := NormalizeNvmeAddr(testCase.input)
			if got != testCase.expected {
				t.Errorf("NormalizeNvmeAddr(%q) = %q, want %q", testCase.input, got, testCase.expected)
			}
		})
	}
}

func TestIsSameNvmeAddr(t *testing.T) {
	testCases := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{"identical IPv4", "10.0.0.1", "10.0.0.1", true},
		{"different IPv4", "10.0.0.1", "10.0.0.2", false},
		{"identical IPv6", "fd00:168:3::3", "fd00:168:3::3", true},
		{"expanded IPv6", "fd00:168:3::3", "fd00:168:3:0:0:0:0:3", true},
		{"uppercase IPv6", "fd00:168:3::a", "FD00:168:3::A", true},
		{"bracketed IPv6", "[fd00:168:3::3]", "fd00:168:3::3", true},
		{"different IPv6", "fd00:168:3::3", "fd00:168:1::2", false},
		{"IPv4-mapped IPv6", "::ffff:10.0.0.1", "10.0.0.1", true},
		{"empty pair", "", "", true},
		{"empty and address", "", "fd00:168:3::3", false},
		{"non-IP equal", "host-a", "host-a", true},
		{"non-IP different", "host-a", "host-b", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := IsSameNvmeAddr(testCase.a, testCase.b); got != testCase.expected {
				t.Errorf("IsSameNvmeAddr(%q, %q) = %v, want %v", testCase.a, testCase.b, got, testCase.expected)
			}
		})
	}
}
