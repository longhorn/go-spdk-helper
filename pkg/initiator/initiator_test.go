package initiator

import (
	"testing"
)

func TestConnectNVMeTCPPathRequiresNVMeInfo(t *testing.T) {
	i := &Initiator{Name: "vol-a"}
	if err := i.ConnectNVMeTCPPath("10.0.0.1", "4420"); err == nil {
		t.Fatal("expected error when NVMeTCPInfo is nil")
	}
}

func TestReconnectNVMeTCPPathRequiresNVMeInfo(t *testing.T) {
	i := &Initiator{Name: "vol-a"}
	if err := i.ReconnectNVMeTCPPath("10.0.0.1", "4420"); err == nil {
		t.Fatal("expected error when NVMeTCPInfo is nil")
	}
}

func TestReconnectNVMeTCPPathRequiresTargetAddress(t *testing.T) {
	i := &Initiator{
		Name:        "vol-a",
		NVMeTCPInfo: &NVMeTCPInfo{SubsystemNQN: "nqn.test"},
	}
	if err := i.ReconnectNVMeTCPPath("", "4420"); err == nil {
		t.Fatal("expected error when target address is empty")
	}
}

func TestSelectControllerForNVMeDevicePrefersRequestedPath(t *testing.T) {
	device := Device{
		SubsystemNQN: "nqn.test",
		Controllers: []Controller{
			{Controller: "nvme0", Address: "traddr=10.0.0.1 trsvcid=4420"},
			{Controller: "nvme1", Address: "traddr=10.0.0.2 trsvcid=4420"},
		},
	}

	controller, err := selectControllerForNVMeDevice(device, "10.0.0.2", "4420", "nvme0")
	if err != nil {
		t.Fatalf("unexpected error selecting controller: %v", err)
	}
	if controller.Controller != "nvme1" {
		t.Fatalf("expected requested path controller nvme1, got %s", controller.Controller)
	}
}

func TestSelectControllerForNVMeDeviceFallsBackToRecordedController(t *testing.T) {
	device := Device{
		SubsystemNQN: "nqn.test",
		Controllers: []Controller{
			{Controller: "nvme0", Address: "traddr=10.0.0.1 trsvcid=4420"},
			{Controller: "nvme1", Address: "traddr=10.0.0.2 trsvcid=4420"},
		},
	}

	controller, err := selectControllerForNVMeDevice(device, "10.0.0.3", "4420", "nvme1")
	if err != nil {
		t.Fatalf("unexpected error selecting controller: %v", err)
	}
	if controller.Controller != "nvme1" {
		t.Fatalf("expected recorded controller nvme1, got %s", controller.Controller)
	}
}

func TestSelectControllerForNVMeDeviceFallsBackToFirstController(t *testing.T) {
	device := Device{
		SubsystemNQN: "nqn.test",
		Controllers: []Controller{
			{Controller: "nvme0", Address: "traddr=10.0.0.1 trsvcid=4420"},
			{Controller: "nvme1", Address: "traddr=10.0.0.2 trsvcid=4420"},
		},
	}

	controller, err := selectControllerForNVMeDevice(device, "", "", "")
	if err != nil {
		t.Fatalf("unexpected error selecting controller: %v", err)
	}
	if controller.Controller != "nvme0" {
		t.Fatalf("expected first controller nvme0, got %s", controller.Controller)
	}
}

func TestSelectControllerForNVMeDeviceRequiresControllers(t *testing.T) {
	_, err := selectControllerForNVMeDevice(Device{SubsystemNQN: "nqn.test"}, "10.0.0.1", "4420", "")
	if err == nil {
		t.Fatal("expected error when device has no controllers")
	}
}
