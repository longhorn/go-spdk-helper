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
	t.Run("sysfs block directory layout", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Mock /sys/block layout with valid namespaces, controllers, partitions, and non-nvme devices
		validEntries := []string{"nvme0n1", "nvme1n1", "nvme10n2"}
		ignoredEntries := []string{"nvme0", "nvme1", "nvme0n1p1", "sda", "vda", "loop0"}

		for _, name := range append(validEntries, ignoredEntries...) {
			if err := os.MkdirAll(filepath.Join(tmpDir, name), 0755); err != nil {
				t.Fatal(err)
			}
		}

		// Mock metadata for nvme0n1: 10GiB, hw_sector_size 512
		nvme0n1Dir := filepath.Join(tmpDir, "nvme0n1")
		if err := os.WriteFile(filepath.Join(nvme0n1Dir, "size"), []byte("20971520\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(nvme0n1Dir, "queue"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(nvme0n1Dir, "queue", "hw_sector_size"), []byte("512\n"), 0644); err != nil {
			t.Fatal(err)
		}

		// Mock metadata for nvme1n1: 2GiB, hw_sector_size invalid/0, fallback to logical_block_size 4096
		nvme1n1Dir := filepath.Join(tmpDir, "nvme1n1")
		if err := os.WriteFile(filepath.Join(nvme1n1Dir, "size"), []byte("4194304\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(nvme1n1Dir, "queue"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(nvme1n1Dir, "queue", "hw_sector_size"), []byte("0\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(nvme1n1Dir, "queue", "logical_block_size"), []byte("4096\n"), 0644); err != nil {
			t.Fatal(err)
		}

		// nvme10n2 has no size/queue files, tests default fallback values

		devices, err := listRecognizedNvmeDevicesFromSysfs(tmpDir)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(devices) != len(validEntries) {
			t.Fatalf("expected %d devices, got %d: %+v", len(validEntries), len(devices), devices)
		}

		devMap := make(map[string]CliDevice)
		for _, d := range devices {
			devMap[d.DevicePath] = d
		}

		for _, v := range validEntries {
			expected := "/dev/" + v
			if _, exists := devMap[expected]; !exists {
				t.Errorf("expected %s to be present in devices: %+v", expected, devices)
			}
		}

		for _, ign := range ignoredEntries {
			unexpected := "/dev/" + ign
			if _, exists := devMap[unexpected]; exists {
				t.Errorf("unexpected entry %s found in devices: %+v", unexpected, devices)
			}
		}

		// Verify metadata parsing for nvme0n1
		d0 := devMap["/dev/nvme0n1"]
		if d0.NameSpace != 1 {
			t.Errorf("expected NSID 1 for nvme0n1, got %d", d0.NameSpace)
		}
		if d0.SectorSize != 512 {
			t.Errorf("expected SectorSize 512, got %d", d0.SectorSize)
		}
		if d0.PhysicalSize != 20971520*512 {
			t.Errorf("expected PhysicalSize %d, got %d", int64(20971520*512), d0.PhysicalSize)
		}
		if d0.MaximumLBA != 20971520 {
			t.Errorf("expected MaximumLBA 20971520, got %d", d0.MaximumLBA)
		}
		if d0.UsedBytes != 20971520*512 {
			t.Errorf("expected UsedBytes %d, got %d", int64(20971520*512), d0.UsedBytes)
		}

		// Verify fallback for nvme1n1 (logical_block_size = 4096)
		d1 := devMap["/dev/nvme1n1"]
		if d1.NameSpace != 1 {
			t.Errorf("expected NSID 1 for nvme1n1, got %d", d1.NameSpace)
		}
		if d1.SectorSize != 4096 {
			t.Errorf("expected SectorSize 4096, got %d", d1.SectorSize)
		}
		if d1.PhysicalSize != 4194304*512 {
			t.Errorf("expected PhysicalSize %d, got %d", int64(4194304*512), d1.PhysicalSize)
		}
		if d1.MaximumLBA != (4194304*512)/4096 {
			t.Errorf("expected MaximumLBA %d, got %d", (4194304*512)/4096, d1.MaximumLBA)
		}
		if d1.UsedBytes != 4194304*512 {
			t.Errorf("expected UsedBytes %d, got %d", int64(4194304*512), d1.UsedBytes)
		}

		// Verify defaults for nvme10n2
		d10 := devMap["/dev/nvme10n2"]
		if d10.NameSpace != 2 {
			t.Errorf("expected NSID 2 for nvme10n2, got %d", d10.NameSpace)
		}
		if d10.SectorSize != 512 {
			t.Errorf("expected default SectorSize 512, got %d", d10.SectorSize)
		}
		if d10.PhysicalSize != 0 {
			t.Errorf("expected default PhysicalSize 0, got %d", d10.PhysicalSize)
		}
	})

	t.Run("sysfs subsystem directory layout", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Mock /sys/devices/virtual/nvme-subsystem layout
		subsys0 := filepath.Join(tmpDir, "nvme-subsys0")
		if err := os.MkdirAll(filepath.Join(subsys0, "nvme0"), 0755); err != nil { // controller (should be ignored)
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(subsys0, "nvme0n1"), 0755); err != nil { // namespace
			t.Fatal(err)
		}

		subsys1 := filepath.Join(tmpDir, "nvme-subsys1")
		if err := os.MkdirAll(filepath.Join(subsys1, "nvme1"), 0755); err != nil { // controller (should be ignored)
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(subsys1, "nvme1n1"), 0755); err != nil { // namespace
			t.Fatal(err)
		}

		devices, err := listRecognizedNvmeDevicesFromSysfs(tmpDir)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(devices) != 2 {
			t.Fatalf("expected 2 devices, got: %d", len(devices))
		}

		devMap := make(map[string]CliDevice)
		for _, d := range devices {
			devMap[d.DevicePath] = d
		}

		if d0, ok := devMap["/dev/nvme0n1"]; !ok || d0.NameSpace != 1 {
			t.Fatalf("expected /dev/nvme0n1 with NSID 1, got: %+v", devMap)
		}
		if d1, ok := devMap["/dev/nvme1n1"]; !ok || d1.NameSpace != 1 {
			t.Fatalf("expected /dev/nvme1n1 with NSID 1, got: %+v", devMap)
		}
		if _, ok := devMap["/dev/nvme0"]; ok {
			t.Fatalf("controller devices should not be included, got: %+v", devices)
		}
		if _, ok := devMap["/dev/nvme1"]; ok {
			t.Fatalf("controller devices should not be included, got: %+v", devices)
		}
	})

	t.Run("empty sysfs directory returns empty slice without error", func(t *testing.T) {
		tmpDir := t.TempDir()
		devices, err := listRecognizedNvmeDevicesFromSysfs(tmpDir)
		if err != nil {
			t.Fatalf("expected no error for empty directory, got: %v", err)
		}
		if len(devices) != 0 {
			t.Fatalf("expected 0 devices, got: %d", len(devices))
		}
	})
}
