package initiator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	. "gopkg.in/check.v1"

	commonns "github.com/longhorn/go-common-libs/ns"
	commontypes "github.com/longhorn/go-common-libs/types"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	"github.com/longhorn/go-spdk-helper/pkg/types"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func Test(t *testing.T) { TestingT(t) }

type InitiatorTestSuite struct{}

var _ = Suite(&InitiatorTestSuite{})

func (s *InitiatorTestSuite) TestNewInitiatorValidation(c *C) {
	testCases := []struct {
		name        string
		initiator   string
		nvmeInfo    *NVMeTCPInfo
		ublkInfo    *UblkInfo
		expectError string
	}{
		{
			name:        "empty initiator name",
			initiator:   "",
			nvmeInfo:    &NVMeTCPInfo{SubsystemNQN: "nqn.test"},
			expectError: "empty name for initiator creation",
		},
		{
			name:        "both infos nil",
			initiator:   "vol-1",
			expectError: "both nvmeTCPInfo and ublkInfo are nil or non-nil",
		},
		{
			name:        "both infos are non nil",
			initiator:   "vol-1",
			nvmeInfo:    &NVMeTCPInfo{SubsystemNQN: "nqn.test"},
			ublkInfo:    &UblkInfo{BdevName: "bdev-1"},
			expectError: "both nvmeTCPInfo and ublkInfo are nil or non-nil",
		},
		{
			name:        "nvme info has empty subsystem nqn",
			initiator:   "vol-1",
			nvmeInfo:    &NVMeTCPInfo{},
			expectError: "empty subsystem for NVMe/TCP initiator creation",
		},
		{
			name:        "ublk info has empty bdev",
			initiator:   "vol-1",
			ublkInfo:    &UblkInfo{},
			expectError: "empty BdevName for ublk initiator creation",
		},
	}

	for _, tc := range testCases {
		i, err := NewInitiator(tc.initiator, "", tc.nvmeInfo, tc.ublkInfo)
		c.Assert(i, IsNil, Commentf("case=%s", tc.name))
		c.Assert(err, NotNil, Commentf("case=%s", tc.name))
		c.Assert(err.Error(), Matches, ".*"+tc.expectError+".*", Commentf("case=%s", tc.name))
	}
}

func (s *InitiatorTestSuite) TestNewInitiatorSuccessWithNVMeInfo(c *C) {
	nvmeInfo := &NVMeTCPInfo{
		SubsystemNQN: "nqn.2014-08.org.nvmexpress.discovery",
	}

	i, err := NewInitiator("vol-1", "", nvmeInfo, nil)
	c.Assert(err, IsNil)
	c.Assert(i, NotNil)
	c.Assert(i.Name, Equals, "vol-1")
	c.Assert(i.Endpoint, Equals, util.GetLonghornDevicePath("vol-1"))
	c.Assert(i.NVMeTCPInfo, Equals, nvmeInfo)
	c.Assert(i.UblkInfo, IsNil)
	c.Assert(i.executor, NotNil)
	c.Assert(i.logger, NotNil)
}

func (s *InitiatorTestSuite) TestNewInitiatorSuccessWithUblkInfo(c *C) {
	ublkInfo := &UblkInfo{
		BdevName: "bdev-1",
	}

	i, err := NewInitiator("vol-2", "", nil, ublkInfo)
	c.Assert(err, IsNil)
	c.Assert(i, NotNil)
	c.Assert(i.Name, Equals, "vol-2")
	c.Assert(i.Endpoint, Equals, util.GetLonghornDevicePath("vol-2"))
	c.Assert(i.NVMeTCPInfo, IsNil)
	c.Assert(i.UblkInfo, Equals, ublkInfo)
	c.Assert(i.executor, NotNil)
	c.Assert(i.logger, NotNil)
}

func (s *InitiatorTestSuite) TestNewLockInvalidHostProc(c *C) {
	i := &Initiator{
		Name:     "vol-1",
		hostProc: "/invalid/host/proc",
	}

	lock, err := i.newLock("TestNewLockInvalidHostProc")
	c.Assert(lock, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, ".*invalid host proc path.*")
}

func (s *InitiatorTestSuite) TestWaitForNVMeTCPConnectNilInfo(c *C) {
	i := &Initiator{}
	err := i.WaitForNVMeTCPConnect(1, time.Millisecond)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, ".*nvmeTCPInfo is nil.*")
}

func (s *InitiatorTestSuite) TestWaitForNVMeTCPTargetDisconnectNilInfo(c *C) {
	i := &Initiator{}
	err := i.WaitForNVMeTCPTargetDisconnect(1, time.Millisecond)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, ".*nvmeTCPInfo is nil.*")
}

func (s *InitiatorTestSuite) TestWaitForNVMeTCPTargetDisconnectZeroRetries(c *C) {
	i := &Initiator{
		NVMeTCPInfo: &NVMeTCPInfo{},
	}

	err := i.WaitForNVMeTCPTargetDisconnect(0, time.Millisecond)
	c.Assert(err, IsNil)
}

func (s *InitiatorTestSuite) TestDiscoverAndConnectNVMeTCPTargetFailsFastOnDuplicateCntlid(c *C) {
	attemptsFile := filepath.Join(c.MkDir(), "connect-attempts")
	restorePath := setupFakeCommandPath(c, map[string]string{
		"cat": `#!/bin/sh
case "$1" in
	/etc/nvme/hostid) echo "test-host-id"; exit 0 ;;
	/sys/class/dmi/id/product_uuid) echo "test-product-uuid"; exit 0 ;;
esac
exit 1
`,
		"nvme": `#!/bin/sh
case "$1" in
	--version) echo "nvme version 1.16"; exit 0 ;;
	--show-hostnqn) echo "nqn.2014-08.org.nvmexpress:uuid:test-host"; exit 0 ;;
	list) echo '{"Devices":[]}'; exit 0 ;;
	list-subsys) echo '{"Subsystems":[]}'; exit 0 ;;
	connect) echo connect >> ` + attemptsFile + `; echo "Failed to write to /dev/nvme-fabrics: Duplicate cntlid" >&2; exit 1 ;;
esac
exit 0
`,
	})
	defer restorePath()

	executor, err := newExecutorWithoutNamespace()
	c.Assert(err, IsNil)

	i := &Initiator{
		Name:        "vol-a",
		NVMeTCPInfo: &NVMeTCPInfo{SubsystemNQN: testSubsystemNQN},
		executor:    executor,
		logger:      logrus.New(),
	}

	_, _, err = i.discoverAndConnectNVMeTCPTarget("10.0.0.1", "20006", 3, time.Millisecond)
	c.Assert(err, NotNil)
	c.Assert(types.ErrorIsDuplicateCntlid(err), Equals, true)

	attempts, err := os.ReadFile(attemptsFile)
	c.Assert(err, IsNil)
	c.Assert(strings.Count(string(attempts), "connect\n"), Equals, 1)
}

func (s *InitiatorTestSuite) TestNVMeInfoGetters(c *C) {
	i := &Initiator{
		Endpoint: "/tmp/endpoint",
	}

	c.Assert(i.GetControllerName(), Equals, "")
	c.Assert(i.GetNamespaceName(), Equals, "")
	c.Assert(i.GetTransportAddress(), Equals, "")
	c.Assert(i.GetTransportServiceID(), Equals, "")
	c.Assert(i.GetEndpoint(), Equals, "")

	i.NVMeTCPInfo = &NVMeTCPInfo{
		ControllerName:     "nvme1",
		NamespaceName:      "nvme1n1",
		TransportAddress:   "10.0.0.1",
		TransportServiceID: "4420",
	}
	i.isUp = true

	c.Assert(i.GetControllerName(), Equals, "nvme1")
	c.Assert(i.GetNamespaceName(), Equals, "nvme1n1")
	c.Assert(i.GetTransportAddress(), Equals, "10.0.0.1")
	c.Assert(i.GetTransportServiceID(), Equals, "4420")
	c.Assert(i.GetEndpoint(), Equals, "/tmp/endpoint")
}

func (s *InitiatorTestSuite) TestLoadNVMeDeviceInfoWithoutLockNilInfo(c *C) {
	i := &Initiator{}
	err := i.loadNVMeDeviceInfoWithoutLock("10.0.0.1", "4420", "nqn.test")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, ".*nvmeTCPInfo is nil.*")
}

func (s *InitiatorTestSuite) TestLoadEndpointForNvmeTcpFrontendNilInfo(c *C) {
	i := &Initiator{}
	err := i.LoadEndpointForNvmeTcpFrontend(false)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, ".*nvmeTCPInfo is nil.*")
}

func (s *InitiatorTestSuite) TestRecordConnectedNVMeTCPInfoUpdatesSubsystemBeforeControllerValidation(c *C) {
	i := &Initiator{
		NVMeTCPInfo: &NVMeTCPInfo{
			SubsystemNQN:   "nqn.old",
			ControllerName: "nvme-old",
		},
	}

	err := i.recordConnectedNVMeTCPInfo("nqn.new", "")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "controller name is empty")
	c.Assert(i.NVMeTCPInfo.SubsystemNQN, Equals, "nqn.new")
	c.Assert(i.NVMeTCPInfo.ControllerName, Equals, "nvme-old")
}

func (s *InitiatorTestSuite) TestRecordConnectedNVMeTCPInfoNilInfo(c *C) {
	i := &Initiator{}

	err := i.recordConnectedNVMeTCPInfo("nqn.new", "nvme1")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "nvmeTCPInfo is nil")
}

func (s *InitiatorTestSuite) TestIsNamespaceExist(c *C) {
	i := &Initiator{}
	c.Assert(i.isNamespaceExist([]string{"nvme0n1"}), Equals, false)

	i.NVMeTCPInfo = &NVMeTCPInfo{NamespaceName: "nvme1n1"}
	c.Assert(i.isNamespaceExist([]string{"nvme0n1", "nvme1n1"}), Equals, true)
	c.Assert(i.isNamespaceExist([]string{"nvme0n1"}), Equals, false)
}

func (s *InitiatorTestSuite) TestIsEndpointExist(c *C) {
	dir := c.MkDir()
	endpoint := filepath.Join(dir, "endpoint")

	i := &Initiator{
		Endpoint: endpoint,
	}

	exists, err := i.isEndpointExist()
	c.Assert(err, IsNil)
	c.Assert(exists, Equals, false)

	f, err := os.Create(endpoint)
	c.Assert(err, IsNil)
	c.Assert(f.Close(), IsNil)

	exists, err = i.isEndpointExist()
	c.Assert(err, IsNil)
	c.Assert(exists, Equals, true)
}

func (s *InitiatorTestSuite) TestCreateEndpointSkipWhenExists(c *C) {
	dir := c.MkDir()
	endpoint := filepath.Join(dir, "endpoint")

	f, err := os.Create(endpoint)
	c.Assert(err, IsNil)
	c.Assert(f.Close(), IsNil)

	i := &Initiator{
		Name:     "vol-1",
		Endpoint: endpoint,
		logger:   logrus.New(),
	}

	err = i.createEndpoint()
	c.Assert(err, IsNil)
}

func (s *InitiatorTestSuite) TestRemoveEndpointEmptyPath(c *C) {
	i := &Initiator{
		Endpoint: "",
	}

	err := i.removeEndpoint()
	c.Assert(err, IsNil)
}

func (s *InitiatorTestSuite) TestStopWithoutLockUblkNilClientOrInfo(c *C) {
	i := &Initiator{
		Name: "vol-1",
	}

	busy, err := i.stopWithoutLock(nil, false, false, false)
	c.Assert(busy, Equals, false)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, ".*spdkClient or UblkInfo is nil.*")
}

func (s *InitiatorTestSuite) TestStopWithoutLockUblkUninitializedID(c *C) {
	i := &Initiator{
		Name:     "vol-1",
		UblkInfo: &UblkInfo{UblkID: UnInitializedUblkId},
	}

	busy, err := i.stopWithoutLock(&client.Client{}, false, false, false)
	c.Assert(err, IsNil)
	c.Assert(busy, Equals, false)
}

func (s *InitiatorTestSuite) TestValidateDiskCreationInvalidRetries(c *C) {
	i := &Initiator{}

	err := i.validateDiskCreation("/dev/not-used", 0, time.Millisecond)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, ".*maxRetries must be > 0.*")
}

func (s *InitiatorTestSuite) TestSyncDmDeviceSizeValidation(c *C) {
	i := &Initiator{}

	err := i.SyncDmDeviceSize(1024)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "initiator device source is not initialized")
}

func (s *InitiatorTestSuite) TestSyncDmDeviceSizeValidationEmptySourceName(c *C) {
	i := &Initiator{
		dev: &util.LonghornBlockDevice{},
	}

	err := i.SyncDmDeviceSize(1024)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "initiator device source is not initialized")
}

func (s *InitiatorTestSuite) TestSyncDmDeviceSizeReloadFailure(c *C) {
	restorePath := setupFakeCommandPath(c, map[string]string{
		util.BlockdevBinary: "#!/bin/sh\necho 8\n",
		"dmsetup":           "#!/bin/sh\nexit 1\n",
	})
	defer restorePath()

	executor, err := newExecutorWithoutNamespace()
	c.Assert(err, IsNil)

	i := &Initiator{
		Name: "vol-sync-reload-fail",
		dev: &util.LonghornBlockDevice{
			Source: util.BlockDevice{Name: "fakeblk"},
		},
		executor: executor,
		logger:   logrus.New(),
	}

	err = i.SyncDmDeviceSize(DmSectorSize)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, ".*dmsetup reload.*")
}

func (s *InitiatorTestSuite) TestWaitForDeviceSizeDoesNotReloadDmDevice(c *C) {
	restorePath := setupFakeCommandPath(c, map[string]string{
		util.BlockdevBinary: "#!/bin/sh\necho 8\n",
		"dmsetup":           "#!/bin/sh\nexit 1\n",
	})
	defer restorePath()

	executor, err := newExecutorWithoutNamespace()
	c.Assert(err, IsNil)

	i := &Initiator{
		Name: "vol-wait-size-success",
		dev: &util.LonghornBlockDevice{
			Source: util.BlockDevice{Name: "fakeblk"},
		},
		executor: executor,
		logger:   logrus.New(),
	}

	err = i.WaitForDeviceSize(DmSectorSize)
	c.Assert(err, IsNil)
}

func (s *InitiatorTestSuite) TestSyncDmDeviceSizeSuccess(c *C) {
	restorePath := setupFakeCommandPath(c, map[string]string{
		util.BlockdevBinary: "#!/bin/sh\necho 8\n",
		"dmsetup":           "#!/bin/sh\nexit 0\n",
		"lsblk":             "#!/bin/sh\necho '{\"blockdevices\":[{\"maj:min\":\"253:7\"}]}'\n",
	})
	defer restorePath()

	executor, err := newExecutorWithoutNamespace()
	c.Assert(err, IsNil)

	i := &Initiator{
		Name: "vol-sync-success",
		dev: &util.LonghornBlockDevice{
			Source: util.BlockDevice{Name: "fakeblk"},
		},
		executor: executor,
		logger:   logrus.New(),
	}

	err = i.SyncDmDeviceSize(1024)
	c.Assert(err, IsNil)
	c.Assert(i.dev.Export.Name, Equals, "vol-sync-success")
	c.Assert(i.dev.Export.Major, Equals, 253)
	c.Assert(i.dev.Export.Minor, Equals, 7)
}

func (s *InitiatorTestSuite) TestGetDmDevicePath(c *C) {
	c.Assert(getDmDevicePath("vol-1"), Equals, "/dev/mapper/vol-1")
}

// New multipath tests (plain testing.T)

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

func newExecutorWithoutNamespace() (*commonns.Executor, error) {
	return commonns.NewNamespaceExecutor(commontypes.ProcessNone, commontypes.ProcDirectory, nil)
}

func setupFakeCommandPath(c *C, scripts map[string]string) func() {
	dir, err := findExecutableTempDir()
	if err != nil {
		c.Skip(fmt.Sprintf("skip fake command tests because no executable temp directory is available: %v", err))
	}
	oldPath := os.Getenv("PATH")

	fakeScripts := map[string]string{}
	for name, content := range scripts {
		fakeScripts[name] = content
	}
	// Ensure nsenter behavior is deterministic in tests across distros/versions.
	// The namespace executor only uses "-V" and program execution in these tests.
	if _, ok := fakeScripts[commontypes.NsBinary]; !ok {
		fakeScripts[commontypes.NsBinary] = "#!/bin/sh\nif [ \"$1\" = \"-V\" ]; then echo \"nsenter from util-linux\"; exit 0; fi\nexec \"$@\"\n"
	}

	for name, content := range fakeScripts {
		path := filepath.Join(dir, name)
		err = os.WriteFile(path, []byte(content), 0755)
		c.Assert(err, IsNil)
	}

	err = os.Setenv("PATH", dir+":"+oldPath)
	c.Assert(err, IsNil)

	return func() {
		_ = os.Setenv("PATH", oldPath)
		_ = os.RemoveAll(dir)
	}
}

func findExecutableTempDir() (string, error) {
	candidates := []string{}
	if cacheDir, err := os.UserCacheDir(); err == nil && cacheDir != "" {
		candidates = append(candidates, cacheDir)
	}
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		candidates = append(candidates, runtimeDir)
	}
	if tmpDir := os.TempDir(); tmpDir != "" {
		candidates = append(candidates, tmpDir)
	}
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		candidates = append(candidates, cwd)
	}
	candidates = append(candidates, "/var/tmp")

	seen := map[string]struct{}{}
	for _, base := range candidates {
		if base == "" {
			continue
		}
		if _, ok := seen[base]; ok {
			continue
		}
		seen[base] = struct{}{}

		dir, err := os.MkdirTemp(base, "fake-cmd-")
		if err != nil {
			continue
		}
		if canExecuteInDir(dir) {
			return dir, nil
		}
		_ = os.RemoveAll(dir)
	}

	return "", fmt.Errorf("failed to find an executable writable directory")
}

func canExecuteInDir(dir string) bool {
	probePath := filepath.Join(dir, "probe")
	if err := os.WriteFile(probePath, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		return false
	}

	cmd := exec.Command(probePath)
	return cmd.Run() == nil
}

// fakeDmsetupScript reports an existing, active dm device whose table maps depDevice.
// When the table is not expected to change, suspend/reload/resume fail the test.
func fakeDmsetupScript(depDevice string, allowTableChange bool) string {
	reject := "echo \"unexpected $1\" >&2; exit 1"
	if allowTableChange {
		reject = "exit 0"
	}
	return `#!/bin/sh
case "$1" in
	deps) echo "1 dependencies  : (` + depDevice + `)" ;;
	info)
		if [ "$2" = "--columns" ]; then
			echo "vol-replace 253:7 L--w 253 7 1 1 0"
		else
			echo "State:             ACTIVE"
		fi
		;;
	suspend|reload|resume) ` + reject + ` ;;
esac
exit 0
`
}

// The plain invocation and the -J one both contain "MAJ:MIN", so dispatch on -J.
// sourceMajMin is what the source device reports, which is how a table naming a
// reused device is told apart from one that still maps it.
func fakeLsblkScript(sourceMajMin string) string {
	return `#!/bin/sh
case "$*" in
	*-J*/dev/nvme0n1) echo '{"blockdevices":[{"maj:min":"` + sourceMajMin + `"}]}' ;;
	*-J*) echo '{"blockdevices":[{"maj:min":"253:7"}]}' ;;
	*) echo "nvme0n1 259:0" ;;
esac
exit 0
`
}

const testSourceMajMin = "259:0"

func newReplaceTargetTestInitiator(c *C) *Initiator {
	executor, err := newExecutorWithoutNamespace()
	c.Assert(err, IsNil)

	return &Initiator{
		Name: "vol-replace",
		dev: &util.LonghornBlockDevice{
			Source: util.BlockDevice{Name: "nvme0n1", Major: 259, Minor: 0},
		},
		executor: executor,
		logger:   logrus.New(),
	}
}

// Suspending to reload a table that needs no change is what deadlocks when the old
// path of the namespace head is still draining.
func (s *InitiatorTestSuite) TestReplaceDmDeviceTargetSkipsUnneededTableChange(c *C) {
	restorePath := setupFakeCommandPath(c, map[string]string{
		"dmsetup": fakeDmsetupScript("nvme0n1", false),
		"lsblk":   fakeLsblkScript(testSourceMajMin),
	})
	defer restorePath()

	i := newReplaceTargetTestInitiator(c)

	c.Assert(i.replaceDmDeviceTarget(), IsNil)
	c.Assert(i.dev.Export.Name, Equals, "vol-replace")
	c.Assert(i.dev.Export.Major, Equals, 253)
	c.Assert(i.dev.Export.Minor, Equals, 7)
}

func (s *InitiatorTestSuite) TestReplaceDmDeviceTargetReloadsOnDeviceChange(c *C) {
	restorePath := setupFakeCommandPath(c, map[string]string{
		"dmsetup":           fakeDmsetupScript("nvme1n1", true),
		"lsblk":             fakeLsblkScript(testSourceMajMin),
		util.BlockdevBinary: "#!/bin/sh\necho 8\n",
	})
	defer restorePath()

	i := newReplaceTargetTestInitiator(c)

	c.Assert(i.replaceDmDeviceTarget(), IsNil)
	c.Assert(i.dev.Export.Major, Equals, 253)
	c.Assert(i.dev.Export.Minor, Equals, 7)
}

// The table names the device the initiator resolved, but the name has since been
// handed to a different one, so the target still has to be replaced. The stub rejects
// suspend, which is how the attempt is made visible.
func (s *InitiatorTestSuite) TestReplaceDmDeviceTargetReloadsOnReusedDeviceName(c *C) {
	restorePath := setupFakeCommandPath(c, map[string]string{
		"dmsetup": fakeDmsetupScript("nvme0n1", false),
		"lsblk":   fakeLsblkScript("259:5"),
	})
	defer restorePath()

	i := newReplaceTargetTestInitiator(c)

	c.Assert(i.replaceDmDeviceTarget(), ErrorMatches, ".*failed to suspend.*")
}

// The dm device is removed based on what device-mapper reports, so a node udev never
// published does not make the removal look unnecessary.
func (s *InitiatorTestSuite) TestRemoveLinearDmDeviceIgnoresMissingUdevNode(c *C) {
	restorePath := setupFakeCommandPath(c, map[string]string{
		"dmsetup": fakeDmsetupScript("nvme0n1", true),
	})
	defer restorePath()

	i := newReplaceTargetTestInitiator(c)

	c.Assert(getDmDevicePath(i.Name), Not(Equals), "")
	if _, err := os.Stat(getDmDevicePath(i.Name)); !os.IsNotExist(err) {
		c.Skip("the test requires the dm device node to be absent")
	}

	c.Assert(i.removeLinearDmDevice(false, false), IsNil)
}

func (s *InitiatorTestSuite) TestRemoveLinearDmDeviceReportsMissingDevice(c *C) {
	restorePath := setupFakeCommandPath(c, map[string]string{
		"dmsetup": "#!/bin/sh\necho 'Device does not exist.' >&2\nexit 1\n",
	})
	defer restorePath()

	i := newReplaceTargetTestInitiator(c)

	c.Assert(os.IsNotExist(i.removeLinearDmDevice(false, false)), Equals, true)
}

// recordingDmsetupScript reports an existing device and appends every invocation to
// argsFile.
func recordingDmsetupScript(argsFile string) string {
	return `#!/bin/sh
echo "$@" >> ` + argsFile + `
case "$1" in
	info)
		if [ "$2" = "--columns" ]; then
			echo "vol-replace 253:7 L--w 253 7 1 1 0"
		else
			echo "State:             ACTIVE"
		fi
		;;
esac
exit 0
`
}

// udev is one of the things that stops responding, so the removal must not wait on it
// and the node it would have dropped is reconciled here instead.
func (s *InitiatorTestSuite) TestRemoveLinearDmDeviceDoesNotWaitForUdev(c *C) {
	argsFile := filepath.Join(c.MkDir(), "args")
	restorePath := setupFakeCommandPath(c, map[string]string{
		"dmsetup": recordingDmsetupScript(argsFile),
	})
	defer restorePath()

	i := newReplaceTargetTestInitiator(c)

	c.Assert(i.removeLinearDmDevice(false, false), IsNil)

	recorded, err := os.ReadFile(argsFile)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(string(recorded), "--noudevsync remove vol-replace"), Equals, true)
	c.Assert(strings.Contains(string(recorded), "mknodes"), Equals, true)
}

func TestSelectControllerForNVMeDeviceSkipsStaleController(t *testing.T) {
	device := Device{
		SubsystemNQN: "nqn.test",
		Controllers: []Controller{
			{Controller: "nvme0", Address: "traddr=10.0.0.1 trsvcid=4420", State: "connecting"},
			{Controller: "nvme1", Address: "traddr=10.0.0.2 trsvcid=4420", State: NvmeControllerStateLive},
		},
	}

	controller, err := selectControllerForNVMeDevice(device, "", "", "")
	if err != nil {
		t.Fatalf("unexpected error selecting controller: %v", err)
	}
	if controller.Controller != "nvme1" {
		t.Fatalf("expected live controller nvme1, got %s", controller.Controller)
	}
}

func TestSelectControllerForNVMeDevicePrefersLiveControllerAtSameAddress(t *testing.T) {
	device := Device{
		SubsystemNQN: "nqn.test",
		Controllers: []Controller{
			{Controller: "nvme0", Address: "traddr=10.0.0.1 trsvcid=4420", State: "connecting"},
			{Controller: "nvme3", Address: "traddr=10.0.0.1 trsvcid=4420", State: NvmeControllerStateLive},
		},
	}

	controller, err := selectControllerForNVMeDevice(device, "10.0.0.1", "4420", "")
	if err != nil {
		t.Fatalf("unexpected error selecting controller: %v", err)
	}
	if controller.Controller != "nvme3" {
		t.Fatalf("expected live controller nvme3, got %s", controller.Controller)
	}
}
