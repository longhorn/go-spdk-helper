package initiator

import (
	"os"
	"path/filepath"
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

func TestListRecognizedNvmeDevicesFromSysfs(t *testing.T) {
	tmpDir := t.TempDir()
	// Create mock subsys directories and namespace block device entries
	subsys1 := filepath.Join(tmpDir, "nvme-subsys0")
	if err := os.MkdirAll(subsys1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(subsys1, "nvme0n1"), 0755); err != nil {
		t.Fatal(err)
	}

	subsys2 := filepath.Join(tmpDir, "nvme-subsys1")
	if err := os.MkdirAll(subsys2, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(subsys2, "nvme1n1"), 0755); err != nil {
		t.Fatal(err)
	}

	devices, err := listRecognizedNvmeDevicesFromSysfs(tmpDir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got: %d", len(devices))
	}
	if devices[0].DevicePath != "/dev/nvme0n1" || devices[1].DevicePath != "/dev/nvme1n1" {
		t.Fatalf("unexpected device paths: %+v", devices)
	}
}
