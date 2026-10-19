package initiator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"

	. "gopkg.in/check.v1"

	"github.com/longhorn/go-spdk-helper/pkg/types"
)

const (
	testSysfsAddress      = "traddr=10.0.0.1,trsvcid=20006"
	testSysfsOtherAddress = "traddr=10.0.0.2,trsvcid=20007"
)

// isNvmeDeviceNotFound matches the message the way callers of the initiator do.
func isNvmeDeviceNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), types.ErrorMessageCannotFindValidNvmeDevice)
}

func getDevicesFromSysfs(root, ip, port, nqn string) ([]Device, error) {
	defaultSysfsRoot := sysfsRoot
	defer func() { sysfsRoot = defaultSysfsRoot }()
	sysfsRoot = root
	return GetDevices(ip, port, nqn, nil)
}

type fakeSysfsController struct {
	name       string
	state      string
	address    string
	namespaces []string
	// ana is the ANA state of its namespace devices and paths, none reported when empty.
	ana string
	// transport is tcp when empty.
	transport string
}

// fakeSysfsSubsystem puts namespace devices under the subsystem with native multipath and under the controller without.
func fakeSysfsSubsystem(c *C, root, name, nqn string, controllers []fakeSysfsController, namespaces ...string) {
	dir := filepath.Join(root, "class", "nvme-subsystem", name)
	c.Assert(os.MkdirAll(dir, 0755), IsNil)
	c.Assert(os.WriteFile(filepath.Join(dir, "subsysnqn"), []byte(nqn+"\n"), 0644), IsNil)
	for _, ns := range namespaces {
		fakeSysfsNamespace(c, filepath.Join(dir, ns), "")
	}
	for _, ctrl := range controllers {
		ctrlDir := filepath.Join(dir, ctrl.name)
		c.Assert(os.MkdirAll(ctrlDir, 0755), IsNil)
		transport := ctrl.transport
		if transport == "" {
			transport = "tcp"
		}
		for attribute, value := range map[string]string{"state": ctrl.state, "transport": transport, "address": ctrl.address} {
			if value == "" {
				continue
			}
			c.Assert(os.WriteFile(filepath.Join(ctrlDir, attribute), []byte(value+"\n"), 0644), IsNil)
		}
		for _, ns := range ctrl.namespaces {
			fakeSysfsNamespace(c, filepath.Join(ctrlDir, ns), ctrl.ana)
		}
	}
}

func fakeSysfsNamespace(c *C, dir, ana string) {
	c.Assert(os.MkdirAll(dir, 0755), IsNil)
	c.Assert(os.MkdirAll(filepath.Join(dir, "queue"), 0755), IsNil)
	for attribute, value := range map[string]string{"nsid": "1", "size": "4194304", "queue/logical_block_size": "512"} {
		c.Assert(os.WriteFile(filepath.Join(dir, attribute), []byte(value+"\n"), 0644), IsNil)
	}
	if ana != "" {
		c.Assert(os.WriteFile(filepath.Join(dir, "ana_state"), []byte(ana+"\n"), 0644), IsNil)
	}
}

// testNamespace is a namespace device as fakeSysfsNamespace lays it out: 2 GiB of 512-byte sectors.
func testNamespace(name string) Namespace {
	return Namespace{NameSpace: name, NSID: 1, UsedBytes: 2147483648, MaximumLBA: 4194304, PhysicalSize: 2147483648, SectorSize: 512}
}

func (ctrl fakeSysfsController) usable() bool {
	return (ctrl.state == NvmeControllerStateLive || ctrl.state == NvmeControllerStateDeleting) &&
		(ctrl.ana == "" || ctrl.ana == "optimized" || ctrl.ana == "non-optimized")
}

func (s *InitiatorTestSuite) TestGetDevicesFromSysfsMultipath(c *C) {
	root := c.MkDir()
	fakeSysfsSubsystem(c, root, "nvme-subsys0", testOtherSubsystemNQN,
		[]fakeSysfsController{{name: "nvme0", state: "live", address: testSysfsOtherAddress}}, "nvme0n1")
	// The head number need not match the controller number.
	fakeSysfsSubsystem(c, root, "nvme-subsys1", testSubsystemNQN,
		[]fakeSysfsController{{name: "nvme3", state: "live", address: testSysfsAddress + ",src_addr=10.0.0.9", namespaces: []string{"nvme1c3n2"}}}, "nvme1n2", "ng1n2")

	devices, err := getDevicesFromSysfs(root, "10.0.0.1", "20006", testSubsystemNQN)
	c.Assert(err, IsNil)
	c.Assert(devices, HasLen, 1)
	c.Assert(devices[0].Subsystem, Equals, "nvme-subsys1")
	c.Assert(devices[0].Namespaces, DeepEquals, []Namespace{testNamespace("nvme1n2")})
	c.Assert(devices[0].Controllers, DeepEquals, []Controller{
		{Controller: "nvme3", Transport: "tcp", Address: testSysfsAddress + ",src_addr=10.0.0.9", State: "live"},
	})
}

func (s *InitiatorTestSuite) TestGetDevicesFromSysfsWithoutMultipath(c *C) {
	root := c.MkDir()
	fakeSysfsSubsystem(c, root, "nvme-subsys2", testSubsystemNQN,
		[]fakeSysfsController{{name: "nvme2", state: "live", address: testSysfsAddress, namespaces: []string{"nvme2n1"}}})

	devices, err := getDevicesFromSysfs(root, "10.0.0.1", "20006", testSubsystemNQN)
	c.Assert(err, IsNil)
	c.Assert(devices, HasLen, 1)
	c.Assert(devices[0].Namespaces, DeepEquals, []Namespace{testNamespace("nvme2n1")})
}

// A reconnect to a new address adds a controller while the old one retries.
func (s *InitiatorTestSuite) TestGetDevicesFromSysfsReturnsEveryController(c *C) {
	root := c.MkDir()
	fakeSysfsSubsystem(c, root, "nvme-subsys0", testSubsystemNQN, []fakeSysfsController{
		{name: "nvme0", state: "connecting", address: testSysfsAddress, namespaces: []string{"nvme0c0n1"}},
		{name: "nvme4", state: "live", address: testSysfsOtherAddress, namespaces: []string{"nvme0c4n1"}},
	}, "nvme0n1")

	devices, err := getDevicesFromSysfs(root, "10.0.0.2", "20007", testSubsystemNQN)
	c.Assert(err, IsNil)
	c.Assert(devices, HasLen, 1)
	c.Assert(devices[0].Controllers, HasLen, 2)
	controller, err := selectControllerForNVMeDevice(devices[0], "10.0.0.2", "20007", "")
	c.Assert(err, IsNil)
	c.Assert(controller.Controller, Equals, "nvme4")
}

func (s *InitiatorTestSuite) TestGetDevicesFromSysfsNotFound(c *C) {
	root := c.MkDir()
	_, err := getDevicesFromSysfs(root, "10.0.0.1", "20006", testSubsystemNQN)
	c.Assert(isNvmeDeviceNotFound(err), Equals, true, Commentf("no subsystem class: %v", err))

	fakeSysfsSubsystem(c, root, "nvme-subsys0", testOtherSubsystemNQN,
		[]fakeSysfsController{{name: "nvme0", state: "live", address: testSysfsAddress}}, "nvme0n1")
	_, err = getDevicesFromSysfs(root, "10.0.0.1", "20006", testSubsystemNQN)
	c.Assert(isNvmeDeviceNotFound(err), Equals, true, Commentf("other subsystem only: %v", err))

	// A subsystem can outlive its last controller.
	fakeSysfsSubsystem(c, root, "nvme-subsys1", testSubsystemNQN, nil)
	_, err = getDevicesFromSysfs(root, "10.0.0.1", "20006", testSubsystemNQN)
	c.Assert(isNvmeDeviceNotFound(err), Equals, true, Commentf("no controller: %v", err))
}

// A controller still connecting has no namespace device yet, and its state tells the caller to wait.
func (s *InitiatorTestSuite) TestGetDevicesFromSysfsReportsControllerState(c *C) {
	root := c.MkDir()
	fakeSysfsSubsystem(c, root, "nvme-subsys0", testSubsystemNQN,
		[]fakeSysfsController{{name: "nvme1", state: "connecting", address: testSysfsAddress}})

	_, err := getDevicesFromSysfs(root, "10.0.0.1", "20006", testSubsystemNQN)
	c.Assert(err, ErrorMatches, ".*subsystem NQN .* path nvme1 address .* is in connecting state.*")
	c.Assert(isNvmeDeviceNotFound(err), Equals, false)
}

func (s *InitiatorTestSuite) TestGetDevicesFromSysfsReportsMismatchedAddress(c *C) {
	root := c.MkDir()
	fakeSysfsSubsystem(c, root, "nvme-subsys0", testSubsystemNQN,
		[]fakeSysfsController{{name: "nvme1", state: "live", address: testSysfsOtherAddress, namespaces: []string{"nvme0c1n1"}}}, "nvme0n1")

	_, err := getDevicesFromSysfs(root, "10.0.0.1", "20006", testSubsystemNQN)
	c.Assert(err, ErrorMatches, ".*path nvme1 .* is in live state.*")

	devices, err := getDevicesFromSysfs(root, "", "", testSubsystemNQN)
	c.Assert(err, IsNil)
	c.Assert(devices, HasLen, 1)
}

func (s *InitiatorTestSuite) TestGetDevicesFromSysfsIgnoresPathBeingTornDown(c *C) {
	root := c.MkDir()
	fakeSysfsSubsystem(c, root, "nvme-subsys0", testSubsystemNQN,
		[]fakeSysfsController{{name: "nvme1", state: "deleting (no IO)", address: testSysfsAddress}})

	_, err := getDevicesFromSysfs(root, "10.0.0.1", "20006", testSubsystemNQN)
	c.Assert(isNvmeDeviceNotFound(err), Equals, true, Commentf("%v", err))
}

// An entry removed while being read is left out, as nvme-cli does.
func (s *InitiatorTestSuite) TestGetDevicesFromSysfsEntryDisappearing(c *C) {
	root := c.MkDir()
	// nvme1 lost its attributes while being removed.
	fakeSysfsSubsystem(c, root, "nvme-subsys0", testSubsystemNQN, []fakeSysfsController{
		{name: "nvme1", address: testSysfsAddress, namespaces: []string{"nvme0c1n1"}},
		{name: "nvme4", state: "live", address: testSysfsOtherAddress, namespaces: []string{"nvme0c4n1"}},
	}, "nvme0n1")

	devices, err := getDevicesFromSysfs(root, "10.0.0.2", "20007", testSubsystemNQN)
	c.Assert(err, IsNil)
	c.Assert(devices, HasLen, 1)
	c.Assert(devices[0].Controllers, DeepEquals, []Controller{
		{Controller: "nvme4", Transport: "tcp", Address: testSysfsOtherAddress, State: "live"},
	})

	_, err = getDevicesFromSysfs(root, "10.0.0.1", "20006", testSubsystemNQN)
	c.Assert(err, ErrorMatches, ".*path nvme4 .* is in live state.*")
}

// A sysfs file opened before the kernel removed its entry fails with ENODEV instead.
func (s *InitiatorTestSuite) TestIsSysfsEntryGone(c *C) {
	c.Assert(isSysfsEntryGone(&fs.PathError{Op: "read", Path: "state", Err: syscall.ENOENT}), Equals, true)
	c.Assert(isSysfsEntryGone(&fs.PathError{Op: "read", Path: "state", Err: syscall.ENODEV}), Equals, true)
	c.Assert(isSysfsEntryGone(&fs.PathError{Op: "read", Path: "state", Err: syscall.EIO}), Equals, false)
}

func (s *InitiatorTestSuite) TestGetDevicesFromSysfsEverythingDisappearing(c *C) {
	root := c.MkDir()
	fakeSysfsSubsystem(c, root, "nvme-subsys0", testSubsystemNQN,
		[]fakeSysfsController{{name: "nvme1", address: testSysfsAddress}}, "nvme0n1")
	_, err := getDevicesFromSysfs(root, "10.0.0.1", "20006", testSubsystemNQN)
	c.Assert(isNvmeDeviceNotFound(err), Equals, true, Commentf("controller gone: %v", err))

	root = c.MkDir()
	fakeSysfsSubsystem(c, root, "nvme-subsys0", testSubsystemNQN,
		[]fakeSysfsController{{name: "nvme1", state: "live", address: testSysfsAddress}}, "nvme0n1")
	c.Assert(os.Remove(filepath.Join(root, "class", "nvme-subsystem", "nvme-subsys0", "subsysnqn")), IsNil)
	_, err = getDevicesFromSysfs(root, "10.0.0.1", "20006", testSubsystemNQN)
	c.Assert(isNvmeDeviceNotFound(err), Equals, true, Commentf("subsystem gone: %v", err))
}

// Without native multipath the requested address decides which controller's device is the device.
func (s *InitiatorTestSuite) TestGetDevicesFromSysfsWithoutMultipathPicksDeviceAtAddress(c *C) {
	root := c.MkDir()
	fakeSysfsSubsystem(c, root, "nvme-subsys1", testSubsystemNQN, []fakeSysfsController{
		{name: "nvme1", state: "live", address: testSysfsAddress, namespaces: []string{"nvme1n1"}},
		{name: "nvme4", state: "live", address: testSysfsOtherAddress, namespaces: []string{"nvme4n1"}},
	})

	devices, err := getDevicesFromSysfs(root, "10.0.0.2", "20007", testSubsystemNQN)
	c.Assert(err, IsNil)
	c.Assert(devices, HasLen, 1)
	c.Assert(devices[0].Namespaces, DeepEquals, []Namespace{testNamespace("nvme4n1")})
	c.Assert(devices[0].Controllers, DeepEquals, []Controller{
		{Controller: "nvme4", Transport: "tcp", Address: testSysfsOtherAddress, State: "live"},
	})

	devices, err = getDevicesFromSysfs(root, "10.0.0.1", "20006", testSubsystemNQN)
	c.Assert(err, IsNil)
	c.Assert(devices, HasLen, 1)
	c.Assert(devices[0].Namespaces, DeepEquals, []Namespace{testNamespace("nvme1n1")})
}

// compatScenario is one host state, laid out as sysfs and as nvme-cli output to compare both lookups.
type compatScenario struct {
	name string
	// controllers are in name order, as nvme-cli and sysfs list them.
	controllers []fakeSysfsController
	// heads are the namespace devices with native multipath.
	heads []string
	// headPaths are the controllers with a path to the head, as nvme-cli 2.x reports them; all when nil.
	headPaths []string
	ip, port  string
}

var compatScenarios = []compatScenario{
	{name: "multipath, live", ip: "10.0.0.1", port: "20006", heads: []string{"nvme1n1"},
		controllers: []fakeSysfsController{{name: "nvme1", state: "live", address: testSysfsAddress}}},
	{name: "multipath, no address", heads: []string{"nvme1n1"},
		controllers: []fakeSysfsController{{name: "nvme1", state: "live", address: testSysfsAddress}}},
	{name: "multipath, other address", ip: "10.0.0.3", port: "20008", heads: []string{"nvme1n1"},
		controllers: []fakeSysfsController{{name: "nvme1", state: "live", address: testSysfsAddress}}},
	{name: "multipath, new address, both paths reach the head", ip: "10.0.0.2", port: "20007", heads: []string{"nvme1n1"},
		controllers: []fakeSysfsController{
			{name: "nvme1", state: "connecting", address: testSysfsAddress},
			{name: "nvme4", state: "live", address: testSysfsOtherAddress}}},
	{name: "multipath, new address without a path yet", ip: "10.0.0.2", port: "20007", heads: []string{"nvme1n1"}, headPaths: []string{"nvme1"},
		controllers: []fakeSysfsController{
			{name: "nvme1", state: "connecting", address: testSysfsAddress},
			{name: "nvme4", state: "live", address: testSysfsOtherAddress}}},
	{name: "multipath, old address, torn down path still there", ip: "10.0.0.1", port: "20006", heads: []string{"nvme1n1"},
		controllers: []fakeSysfsController{
			{name: "nvme1", state: "deleting (no IO)", address: testSysfsAddress},
			{name: "nvme4", state: "live", address: testSysfsOtherAddress}}},
	{name: "multipath, old address, torn down path already gone", ip: "10.0.0.1", port: "20006", heads: []string{"nvme1n1"}, headPaths: []string{"nvme4"},
		controllers: []fakeSysfsController{
			{name: "nvme1", state: "deleting (no IO)", address: testSysfsAddress},
			{name: "nvme4", state: "live", address: testSysfsOtherAddress}}},
	{name: "multipath, no address, new path without namespace yet", heads: []string{"nvme1n1"}, headPaths: []string{"nvme1"},
		controllers: []fakeSysfsController{
			{name: "nvme1", state: "connecting", address: testSysfsAddress},
			{name: "nvme4", state: "live", address: testSysfsOtherAddress}}},
	{name: "connecting, no namespace yet", ip: "10.0.0.1", port: "20006",
		controllers: []fakeSysfsController{{name: "nvme1", state: "connecting", address: testSysfsAddress}}},
	{name: "torn down, no namespace", ip: "10.0.0.1", port: "20006",
		controllers: []fakeSysfsController{{name: "nvme1", state: "deleting (no IO)", address: testSysfsAddress}}},
	{name: "subsystem without controller", ip: "10.0.0.1", port: "20006"},
	{name: "without multipath, live", ip: "10.0.0.1", port: "20006",
		controllers: []fakeSysfsController{{name: "nvme1", state: "live", address: testSysfsAddress, namespaces: []string{"nvme1n1"}}}},
	{name: "without multipath, new address", ip: "10.0.0.2", port: "20007",
		controllers: []fakeSysfsController{
			{name: "nvme1", state: "connecting", address: testSysfsAddress, namespaces: []string{"nvme1n1"}},
			{name: "nvme4", state: "live", address: testSysfsOtherAddress, namespaces: []string{"nvme4n1"}}}},
	{name: "without multipath, old address", ip: "10.0.0.1", port: "20006",
		controllers: []fakeSysfsController{
			{name: "nvme1", state: "connecting", address: testSysfsAddress, namespaces: []string{"nvme1n1"}},
			{name: "nvme4", state: "live", address: testSysfsOtherAddress, namespaces: []string{"nvme4n1"}}}},
	{name: "without multipath, two devices, no address",
		controllers: []fakeSysfsController{
			{name: "nvme1", state: "connecting", address: testSysfsAddress, namespaces: []string{"nvme1n1"}},
			{name: "nvme4", state: "live", address: testSysfsOtherAddress, namespaces: []string{"nvme4n1"}}}},
	{name: "without multipath, new address without namespace yet", ip: "10.0.0.2", port: "20007",
		controllers: []fakeSysfsController{
			{name: "nvme1", state: "live", address: testSysfsAddress, namespaces: []string{"nvme1n1"}},
			{name: "nvme4", state: "connecting", address: testSysfsOtherAddress}}},
	{name: "multipath, every path reconnecting", ip: "10.0.0.1", port: "20006", heads: []string{"nvme1n1"},
		controllers: []fakeSysfsController{
			{name: "nvme1", state: "connecting", address: testSysfsAddress},
			{name: "nvme4", state: "resetting", address: testSysfsOtherAddress}}},
	{name: "multipath, every path reconnecting, no address", heads: []string{"nvme1n1"},
		controllers: []fakeSysfsController{{name: "nvme1", state: "connecting", address: testSysfsAddress}}},
	{name: "multipath, only path ANA inaccessible", ip: "10.0.0.1", port: "20006", heads: []string{"nvme1n1"},
		controllers: []fakeSysfsController{{name: "nvme1", state: "live", address: testSysfsAddress, ana: "inaccessible"}}},
	{name: "multipath, new path optimized, old inaccessible", ip: "10.0.0.2", port: "20007", heads: []string{"nvme1n1"},
		controllers: []fakeSysfsController{
			{name: "nvme1", state: "live", address: testSysfsAddress, ana: "inaccessible"},
			{name: "nvme4", state: "live", address: testSysfsOtherAddress, ana: "optimized"}}},
	{name: "multipath, path deleting with I/O allowed", ip: "10.0.0.1", port: "20006", heads: []string{"nvme1n1"},
		controllers: []fakeSysfsController{{name: "nvme1", state: "deleting", address: testSysfsAddress}}},
	{name: "without multipath, own controller reconnecting", ip: "10.0.0.1", port: "20006",
		controllers: []fakeSysfsController{{name: "nvme1", state: "connecting", address: testSysfsAddress, namespaces: []string{"nvme1n1"}}}},
	{name: "without multipath, other address", ip: "10.0.0.3", port: "20008",
		controllers: []fakeSysfsController{
			{name: "nvme1", state: "deleting", address: testSysfsAddress, namespaces: []string{"nvme1n1"}},
			{name: "nvme4", state: "live", address: testSysfsOtherAddress, namespaces: []string{"nvme4n1"}}}},
}

// The device of another volume is on every host the scan runs on.
const (
	compatOtherController = "nvme8"
	compatOtherDevice     = "nvme8n1"
	compatOtherAddress    = "traddr=10.0.0.8,trsvcid=20010"
)

func (sc compatScenario) sysfs(c *C) string {
	root := c.MkDir()
	fakeSysfsSubsystem(c, root, "nvme-subsys0", testOtherSubsystemNQN,
		[]fakeSysfsController{{name: compatOtherController, state: "live", address: compatOtherAddress, namespaces: []string{"nvme8c8n1"}}}, compatOtherDevice)
	// A head reaches a controller through a hidden path device under it.
	controllers := []fakeSysfsController{}
	for _, ctrl := range sc.controllers {
		for _, head := range sc.heads {
			if sc.headPaths != nil && !slices.Contains(sc.headPaths, ctrl.name) {
				continue
			}
			var subsys, nsid int
			_, err := fmt.Sscanf(head, "nvme%dn%d", &subsys, &nsid)
			c.Assert(err, IsNil)
			ctrl.namespaces = append(slices.Clone(ctrl.namespaces), fmt.Sprintf("nvme%dc%sn%d", subsys, strings.TrimPrefix(ctrl.name, "nvme"), nsid))
		}
		controllers = append(controllers, ctrl)
	}
	fakeSysfsSubsystem(c, root, "nvme-subsys1", testSubsystemNQN, controllers, sc.heads...)
	return root
}

func compatPathsJSON(controllers []fakeSysfsController) string {
	paths := []string{}
	for _, ctrl := range controllers {
		path := `{"Name":"` + ctrl.name + `","Transport":"tcp","Address":"` + ctrl.address + `"`
		if ctrl.state != "" {
			path += `,"State":"` + ctrl.state + `"`
		}
		paths = append(paths, path+"}")
	}
	return "[" + strings.Join(paths, ",") + "]"
}

func compatSubsystemJSON(name, nqn string, controllers []fakeSysfsController) string {
	return `{"Name":"` + name + `","NQN":"` + nqn + `","Paths":` + compatPathsJSON(controllers) + `}`
}

// nvmeScript answers like nvme-cli 2.16 on a kernel without the namespace csi attribute.
func (sc compatScenario) nvmeScript() string {
	other := []fakeSysfsController{{name: compatOtherController, state: "live", address: compatOtherAddress}}
	devices := []string{compatOtherDevice}
	perDevice := map[string][]fakeSysfsController{compatOtherDevice: nil}
	// libnvme identifies every namespace there and leaves out those without a usable path.
	for _, head := range sc.heads {
		paths := sc.controllers
		if sc.headPaths != nil {
			paths = nil
			for _, ctrl := range sc.controllers {
				for _, name := range sc.headPaths {
					if ctrl.name == name {
						paths = append(paths, ctrl)
					}
				}
			}
		}
		usable := false
		for _, ctrl := range paths {
			usable = usable || ctrl.usable()
		}
		if !usable {
			continue
		}
		devices = append(devices, head)
		perDevice[head] = paths
	}
	for _, ctrl := range sc.controllers {
		if !ctrl.usable() {
			continue
		}
		for _, ns := range ctrl.namespaces {
			devices = append(devices, ns)
			perDevice[ns] = []fakeSysfsController{ctrl}
		}
	}

	list := []string{}
	cases := ""
	for _, dev := range devices {
		list = append(list, `{"NameSpace":1,"DevicePath":"/dev/`+dev+`","UsedBytes":2147483648,"MaximumLBA":4194304,"PhysicalSize":2147483648,"SectorSize":512}`)
		subsystem := compatSubsystemJSON("nvme-subsys1", testSubsystemNQN, perDevice[dev])
		if dev == compatOtherDevice {
			subsystem = compatSubsystemJSON("nvme-subsys0", testOtherSubsystemNQN, other)
		}
		cases += "\t\t/dev/" + dev + `) echo '[{"HostNQN":"nqn.test:host","Subsystems":[` + subsystem + `]}]' ;;` + "\n"
	}
	global := compatSubsystemJSON("nvme-subsys0", testOtherSubsystemNQN, other) + "," +
		compatSubsystemJSON("nvme-subsys1", testSubsystemNQN, sc.controllers)

	return `#!/bin/sh
case "$1" in
	--version) printf 'nvme version 2.16 (git 2.16)\nlibnvme version 1.16 (git 1.16)\n' ;;
	list) echo '{"Devices":[` + strings.Join(list, ",") + `]}' ;;
	list-subsys)
		case "$4" in
		"") echo '[{"HostNQN":"nqn.test:host","Subsystems":[` + global + `]}]' ;;
` + cases + `		esac
		;;
esac
exit 0
`
}

var compatStatePattern = regexp.MustCompile(`path (\S+) address (\S+) is in (.+?) state`)

// compatOutcome is what the initiator acts on: the error kind, or each device with the controller it selects.
func compatOutcome(devices []Device, err error, ip, port string) string {
	switch {
	case err == nil:
	case isNvmeDeviceNotFound(err):
		return "not found"
	case compatStatePattern.MatchString(err.Error()):
		return "state: " + compatStatePattern.FindString(err.Error())
	default:
		return "error: " + err.Error()
	}
	outcome := []string{}
	for _, d := range devices {
		namespaces := []string{}
		for _, ns := range d.Namespaces {
			namespaces = append(namespaces, fmt.Sprintf("%s(nsid=%d,lba=%d,size=%d,used=%d,sector=%d)",
				ns.NameSpace, ns.NSID, ns.MaximumLBA, ns.PhysicalSize, ns.UsedBytes, ns.SectorSize))
		}
		controllers := []string{}
		for _, c := range d.Controllers {
			controllers = append(controllers, c.Controller)
		}
		controller, selectErr := selectControllerForNVMeDevice(d, ip, port, "")
		selected := controller.Controller
		if selectErr != nil {
			selected = "error"
		}
		outcome = append(outcome, d.SubsystemNQN+" "+strings.Join(namespaces, ",")+" ["+strings.Join(controllers, ",")+"] via "+selected)
	}
	return strings.Join(outcome, "; ")
}

// The initiator must end up with the same device and controller, or the same error kind.
func (s *InitiatorTestSuite) TestGetDevicesMatchesNvmeCli(c *C) {
	defaultSysfsRoot := sysfsRoot
	defer func() { sysfsRoot = defaultSysfsRoot }()

	for _, sc := range compatScenarios {
		sysfsRoot = sc.sysfs(c)
		// Without an NQN both list every device of the host.
		for _, nqn := range []string{testSubsystemNQN, ""} {
			restorePath := setupFakeCommandPath(c, map[string]string{"nvme": sc.nvmeScript()})
			executor, err := newExecutorWithoutNamespace()
			c.Assert(err, IsNil)
			cliDevices, cliErr := getDevicesWithNvmeCli(sc.ip, sc.port, nqn, executor)
			devices, err := GetDevices(sc.ip, sc.port, nqn, executor)
			restorePath()

			want := compatOutcome(cliDevices, cliErr, sc.ip, sc.port)
			c.Assert(compatOutcome(devices, err, sc.ip, sc.port), Equals, want, Commentf("scenario %q nqn %q", sc.name, nqn))
			c.Logf("%s (nqn %q): %s", sc.name, nqn, want)
		}
	}
}

// size counts 512-byte sectors whatever the logical block size is.
func (s *InitiatorTestSuite) TestGetDevicesFromSysfsNamespaceMetadata(c *C) {
	root := c.MkDir()
	fakeSysfsSubsystem(c, root, "nvme-subsys0", testSubsystemNQN,
		[]fakeSysfsController{{name: "nvme0", state: "live", address: testSysfsAddress, namespaces: []string{"nvme0c0n1"}}}, "nvme0n1")
	head := filepath.Join(root, "class", "nvme-subsystem", "nvme-subsys0", "nvme0n1")
	c.Assert(os.WriteFile(filepath.Join(head, "nsid"), []byte("7\n"), 0644), IsNil)
	c.Assert(os.WriteFile(filepath.Join(head, "queue", "logical_block_size"), []byte("4096\n"), 0644), IsNil)

	devices, err := getDevicesFromSysfs(root, "", "", testSubsystemNQN)
	c.Assert(err, IsNil)
	c.Assert(devices, HasLen, 1)
	c.Assert(devices[0].Namespaces, DeepEquals, []Namespace{
		{NameSpace: "nvme0n1", NSID: 7, UsedBytes: 2147483648, MaximumLBA: 524288, PhysicalSize: 2147483648, SectorSize: 4096},
	})
}

func (s *InitiatorTestSuite) TestGetDevicesFromSysfsListsNothingWithoutNvme(c *C) {
	devices, err := getDevicesFromSysfs(c.MkDir(), "", "", "")
	c.Assert(err, IsNil)
	c.Assert(devices, HasLen, 0)
}

func (s *InitiatorTestSuite) TestGetDevicesFromSysfsLoopControllerWithoutAddress(c *C) {
	root := c.MkDir()
	fakeSysfsSubsystem(c, root, "nvme-subsys0", testSubsystemNQN,
		[]fakeSysfsController{{name: "nvme0", state: "live", transport: "loop", namespaces: []string{"nvme0c0n1"}}}, "nvme0n1")
	fakeSysfsSubsystem(c, root, "nvme-subsys1", testOtherSubsystemNQN,
		[]fakeSysfsController{{name: "nvme1", state: "live", namespaces: []string{"nvme1c1n1"}}}, "nvme1n1")

	devices, err := getDevicesFromSysfs(root, "", "", "")
	c.Assert(err, IsNil)
	c.Assert(devices, HasLen, 1, Commentf("a tcp controller without an address is skipped as nvme-cli does: %+v", devices))
	c.Assert(devices[0].Controllers, DeepEquals, []Controller{{Controller: "nvme0", Transport: "loop", State: "live"}})
}

// Linux 6.15+ lists the paths of a head in its multipath directory, which libnvme follows.
func (s *InitiatorTestSuite) TestGetDevicesFromSysfsFollowsMultipathDirectory(c *C) {
	root := c.MkDir()
	fakeSysfsSubsystem(c, root, "nvme-subsys0", testSubsystemNQN, []fakeSysfsController{
		{name: "nvme1", state: "live", address: testSysfsAddress, namespaces: []string{"nvme0c1n1"}},
		{name: "nvme10", state: "live", address: testSysfsOtherAddress, namespaces: []string{"nvme0c10n1"}},
	}, "nvme0n1")
	controllerNames := func() []string {
		devices, err := getDevicesFromSysfs(root, "", "", testSubsystemNQN)
		c.Assert(err, IsNil)
		c.Assert(devices, HasLen, 1)
		names := []string{}
		for _, controller := range devices[0].Controllers {
			names = append(names, controller.Controller)
		}
		return names
	}
	c.Assert(controllerNames(), DeepEquals, []string{"nvme1", "nvme10"})

	multipath := filepath.Join(root, "class", "nvme-subsystem", "nvme-subsys0", "nvme0n1", "multipath")
	for _, path := range []string{"nvme0c1n1", "nvme0c10n1"} {
		c.Assert(os.MkdirAll(filepath.Join(multipath, path), 0755), IsNil)
	}
	c.Assert(controllerNames(), DeepEquals, []string{"nvme10", "nvme1"})
}
