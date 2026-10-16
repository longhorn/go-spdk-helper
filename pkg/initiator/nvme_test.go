package initiator

import (
	"os"
	"path/filepath"
	"strings"

	. "gopkg.in/check.v1"
)

const (
	testSubsystemNQN      = "nqn.2023-01.io.longhorn.spdk:volume-test"
	testOtherSubsystemNQN = "nqn.2023-01.io.longhorn.spdk:volume-other"
	// testOtherDevicePath is the device the stub reports under testOtherSubsystemNQN.
	testOtherDevicePath = "/dev/nvme1n1"
)

// fakeNvmeScript builds an `nvme` stub. The per-device `list-subsys` is what the
// address matching runs on, and it reports no path while one is being torn down,
// so it is set separately from the global listing.
func fakeNvmeScript(listOutput, perDevicePaths, globalSubsysPaths string) string {
	return `#!/bin/sh
case "$1" in
	--version) echo "nvme version 1.16" ;;
	list) echo '` + listOutput + `' ;;
	list-subsys)
		case "$4" in
		"") echo '{"Subsystems":[{"NQN":"` + testSubsystemNQN + `","Paths":[` + globalSubsysPaths + `]}]}' ;;
		` + testOtherDevicePath + `) echo '{"Subsystems":[{"NQN":"` + testOtherSubsystemNQN + `","Paths":[` + testLivePath + `]}]}' ;;
		*) echo '{"Subsystems":[{"NQN":"` + testSubsystemNQN + `","Paths":[` + perDevicePaths + `]}]}' ;;
		esac
		;;
esac
exit 0
`
}

const (
	testDeletingPath = `{"Name":"nvme0","Transport":"tcp","Address":"traddr=10.0.0.1,trsvcid=20006","State":"deleting"}`
	testLivePath     = `{"Name":"nvme1","Transport":"tcp","Address":"traddr=10.0.0.2,trsvcid=20343","State":"live"}`
	testUnknownPath  = `{"Name":"nvme2","Transport":"tcp","Address":"traddr=10.0.0.3,trsvcid=20344"}`
	// testSecondLivePath is a second usable path of the same subsystem, which is what
	// native multipath leaves behind after a switchover.
	testSecondLivePath = `{"Name":"nvme3","Transport":"tcp","Address":"traddr=10.0.0.4,trsvcid=20345","State":"live"}`
	testDeviceList     = `{"Devices":[{"DevicePath":"/dev/nvme0n1","Namespace":1,"SectorSize":512}]}`
	testEmptyList      = `{"Devices":[]}`
	// testMixedDeviceList also lists a device of another subsystem, which is what a
	// host scan actually returns.
	testMixedDeviceList = `{"Devices":[{"DevicePath":"/dev/nvme0n1","Namespace":1,"SectorSize":512},{"DevicePath":"` + testOtherDevicePath + `","Namespace":1,"SectorSize":512}]}`
)

// disableSysfsForTest temporarily points sysfs discovery globals to non-existent
// paths so that CLI-fallback tests can execute deterministically without being
// shadowed by the host's sysfs filesystem.
func disableSysfsForTest() func() {
	origBlock := sysfsBlockPath
	origSubsys := sysfsNvmeSubsystemPath
	sysfsBlockPath = "/nonexistent/sys/block"
	sysfsNvmeSubsystemPath = "/nonexistent/sys/devices/virtual/nvme-subsystem"
	return func() {
		sysfsBlockPath = origBlock
		sysfsNvmeSubsystemPath = origSubsys
	}
}

// The scan sees every NVMe device on the host, so one that answers with another
// subsystem NQN must not be taken for the requested one.
func (s *InitiatorTestSuite) TestGetDevicesSkipsDeviceOfAnotherSubsystem(c *C) {
	defer disableSysfsForTest()()
	restorePath := setupFakeCommandPath(c, map[string]string{
		"nvme": fakeNvmeScript(testMixedDeviceList, testLivePath, testLivePath),
	})
	defer restorePath()

	executor, err := newExecutorWithoutNamespace()
	c.Assert(err, IsNil)

	devices, err := GetDevices("", "", testSubsystemNQN, executor)
	c.Assert(err, IsNil)
	c.Assert(devices, HasLen, 1)
	c.Assert(devices[0].SubsystemNQN, Equals, testSubsystemNQN)
	c.Assert(devices[0].Namespaces[0].NameSpace, Equals, "nvme0n1")
}

func (s *InitiatorTestSuite) TestGetDevicesMatchesRequestedAddress(c *C) {
	defer disableSysfsForTest()()
	restorePath := setupFakeCommandPath(c, map[string]string{
		"nvme": fakeNvmeScript(testDeviceList, testDeletingPath+","+testLivePath, testDeletingPath+","+testLivePath),
	})
	defer restorePath()

	executor, err := newExecutorWithoutNamespace()
	c.Assert(err, IsNil)

	devices, err := GetDevices("10.0.0.2", "20343", testSubsystemNQN, executor)
	c.Assert(err, IsNil)
	c.Assert(devices, HasLen, 1)
	c.Assert(devices[0].Namespaces[0].NameSpace, Equals, "nvme0n1")
}

// An address no controller answers to leaves the device unmatched, and the state of
// the paths that are there is what explains why.
func (s *InitiatorTestSuite) TestGetDevicesSkipsMismatchedAddress(c *C) {
	defer disableSysfsForTest()()
	restorePath := setupFakeCommandPath(c, map[string]string{
		"nvme": fakeNvmeScript(testDeviceList, testLivePath, testLivePath),
	})
	defer restorePath()

	executor, err := newExecutorWithoutNamespace()
	c.Assert(err, IsNil)

	_, err = GetDevices("10.0.0.9", "20343", testSubsystemNQN, executor)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "live state"), Equals, true)
}

func (s *InitiatorTestSuite) TestGetDevicesIgnoresPathBeingTornDown(c *C) {
	defer disableSysfsForTest()()
	restorePath := setupFakeCommandPath(c, map[string]string{
		"nvme": fakeNvmeScript(testDeviceList, "", testDeletingPath+","+testLivePath),
	})
	defer restorePath()

	executor, err := newExecutorWithoutNamespace()
	c.Assert(err, IsNil)

	devices, err := GetDevices("", "", testSubsystemNQN, executor)
	c.Assert(err, IsNil)
	c.Assert(devices, HasLen, 1)
	c.Assert(devices[0].Namespaces[0].NameSpace, Equals, "nvme0n1")
}

// The missing device is the real problem; a path on its way out must not be reported
// in its place.
func (s *InitiatorTestSuite) TestGetDevicesReportsMissingDeviceNotDyingPath(c *C) {
	defer disableSysfsForTest()()
	restorePath := setupFakeCommandPath(c, map[string]string{
		"nvme": fakeNvmeScript(testEmptyList, "", testDeletingPath),
	})
	defer restorePath()

	executor, err := newExecutorWithoutNamespace()
	c.Assert(err, IsNil)

	_, err = GetDevices("", "", testSubsystemNQN, executor)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "deleting state"), Equals, false)
	c.Assert(strings.Contains(err.Error(), "cannot find a valid NVMe device"), Equals, true)
}

// A path the kernel is still failing must be left to ctrl_loss_tmo: disconnecting it
// re-arms the I/O requeueing that failfast had just stopped.
func (s *InitiatorTestSuite) TestDisconnectUsableTargetPathsSkipsUnusablePaths(c *C) {
	disconnected := filepath.Join(c.MkDir(), "disconnected")
	script := `#!/bin/sh
case "$1" in
	--version) echo "nvme version 1.16" ;;
	list-subsys) echo '{"Subsystems":[{"NQN":"` + testSubsystemNQN + `","Paths":[` + testDeletingPath + `,` + testUnknownPath + `,` + testLivePath + `]}]}' ;;
	disconnect) echo "$3" >> ` + disconnected + ` ;;
esac
exit 0
`
	restorePath := setupFakeCommandPath(c, map[string]string{"nvme": script})
	defer restorePath()

	executor, err := newExecutorWithoutNamespace()
	c.Assert(err, IsNil)

	c.Assert(DisconnectUsableTargetPaths(testSubsystemNQN, executor), IsNil)

	recorded, err := os.ReadFile(disconnected)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(string(recorded), "/dev/nvme1"), Equals, true)
	c.Assert(strings.Contains(string(recorded), "/dev/nvme0"), Equals, false)
	// A state we cannot read is not evidence that the path is safe to delete.
	c.Assert(strings.Contains(string(recorded), "/dev/nvme2"), Equals, false)
}

// Every path that refused to disconnect has to reach the caller. Keeping only the
// last one hides how much of the subsystem is still connected.
func (s *InitiatorTestSuite) TestDisconnectUsableTargetPathsReportsEveryFailure(c *C) {
	script := `#!/bin/sh
case "$1" in
	--version) echo "nvme version 1.16" ;;
	list-subsys) echo '{"Subsystems":[{"NQN":"` + testSubsystemNQN + `","Paths":[` + testLivePath + `,` + testSecondLivePath + `]}]}' ;;
	disconnect) echo "disconnect refused" >&2; exit 1 ;;
esac
exit 0
`
	restorePath := setupFakeCommandPath(c, map[string]string{"nvme": script})
	defer restorePath()

	executor, err := newExecutorWithoutNamespace()
	c.Assert(err, IsNil)

	err = DisconnectUsableTargetPaths(testSubsystemNQN, executor)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "nvme1"), Equals, true)
	c.Assert(strings.Contains(err.Error(), "nvme3"), Equals, true)
}
