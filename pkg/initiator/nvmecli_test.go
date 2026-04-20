package initiator

import (
	"testing"
)

func TestGetIPAndPortFromControllerAddressIPv6(t *testing.T) {
	testCases := []struct {
		name         string
		address      string
		expectedIP   string
		expectedPort string
	}{
		{"IPv6 space-separated", "traddr=fd00::1 trsvcid=20001", "fd00::1", "20001"},
		{"IPv6 comma-separated", "traddr=fd00::1,trsvcid=20001", "fd00::1", "20001"},
		{"IPv6 loopback", "traddr=::1 trsvcid=4420", "::1", "4420"},
		{"IPv4 space-separated", "traddr=10.42.2.18 trsvcid=20006", "10.42.2.18", "20006"},
		{"IPv4 comma-separated", "traddr=10.42.2.18,trsvcid=20006", "10.42.2.18", "20006"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotIP, gotPort := GetIPAndPortFromControllerAddress(tc.address)
			if gotIP != tc.expectedIP {
				t.Errorf("GetIPAndPortFromControllerAddress(%q) IP = %q, want %q", tc.address, gotIP, tc.expectedIP)
			}
			if gotPort != tc.expectedPort {
				t.Errorf("GetIPAndPortFromControllerAddress(%q) Port = %q, want %q", tc.address, gotPort, tc.expectedPort)
			}
		})
	}
}
