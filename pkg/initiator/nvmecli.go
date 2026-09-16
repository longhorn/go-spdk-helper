package initiator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"

	commonns "github.com/longhorn/go-common-libs/ns"

	"github.com/longhorn/go-spdk-helper/pkg/types"

	spdkutil "github.com/longhorn/go-spdk-helper/pkg/util"
)

const (
	nvmeBinary = "nvme"

	// nvmeListTimeoutSec is the timeout for the nvme list command, used as a
	// hard deadline via the timeout(1) command. It is set to 5 seconds to ensure
	// rapid failure and avoid cascading process accumulation during device hangs.
	nvmeListTimeoutSec = 5

	DefaultTransportType = "tcp"
)

var (
	sysfsBlockPath         = "/sys/block"
	sysfsNvmeSubsystemPath = "/sys/devices/virtual/nvme-subsystem"
)

const (

	// Set short ctrlLossTimeoutSec for quick response to the controller loss.
	defaultCtrlLossTmo    = 30
	defaultKeepAliveTmo   = 5
	defaultReconnectDelay = 2
	// defaultFastIOFailTmo makes the kernel fail the I/O queued on a namespace whose
	// paths are all down. Its default is off, meaning the I/O is requeued for as long
	// as any controller of the subsystem lingers, which wedges every holder of the
	// device in uninterruptible sleep and cannot be undone from user space. Must stay
	// below defaultCtrlLossTmo.
	defaultFastIOFailTmo = 15
	// NvmeControllerStateLive is the state nvme-cli reports for a usable path.
	NvmeControllerStateLive = "live"
	// NvmeControllerStateDeleting prefixes the states of a path the kernel is already
	// tearing down (e.g. "deleting", "deleting (no IO)").
	NvmeControllerStateDeleting = "deleting"
)

type Device struct {
	Subsystem    string
	SubsystemNQN string
	Controllers  []Controller
	Namespaces   []Namespace
}

type DiscoveryPageEntry struct {
	PortID  uint16 `json:"portid"`
	TrsvcID string `json:"trsvcid"`
	Subnqn  string `json:"subnqn"`
	Traddr  string `json:"traddr"`
	SubType string `json:"subtype"`
}

type Controller struct {
	Controller string
	Transport  string
	Address    string
	State      string
}

// Namespace fields use signed integers instead, because the output of buggy nvme-cli 2.x is possibly negative.
type Namespace struct {
	NameSpace    string
	NSID         int32
	UsedBytes    int64
	MaximumLBA   int64
	PhysicalSize int64
	SectorSize   int32
}

type Subsystem struct {
	Name  string `json:"Name,omitempty"`
	NQN   string `json:"NQN,omitempty"`
	Paths []Path `json:"Paths,omitempty"`
}

type Path struct {
	Name      string `json:"Name,omitempty"`
	Transport string `json:"Transport,omitempty"`
	Address   string `json:"Address,omitempty"`
	State     string `json:"State,omitempty"`
}

func cliVersion(executor *commonns.Executor) (major, minor int, err error) {
	opts := []string{
		"--version",
	}
	outputStr, err := executor.Execute(nil, nvmeBinary, opts, types.ExecuteTimeout)
	if err != nil {
		return 0, 0, err
	}

	versionStr := ""
	lines := strings.Split(string(outputStr), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "nvme") {
			versionStr = strings.TrimSpace(line)
			break
		}
	}

	var version string
	for _, s := range strings.Split(versionStr, " ") {
		if strings.Contains(s, ".") {
			version = s
			break
		}
	}
	if version == "" {
		return 0, 0, fmt.Errorf("failed to get version from %s", outputStr)
	}
	versionArr := strings.Split(version, ".")
	if len(versionArr) >= 1 {
		major, _ = strconv.Atoi(versionArr[0])
	}
	if len(versionArr) >= 2 {
		minor, _ = strconv.Atoi(versionArr[1])
	}
	return major, minor, nil
}

func showHostNQN(executor *commonns.Executor) (string, error) {
	opts := []string{
		"--show-hostnqn",
	}

	outputStr, err := executor.Execute(nil, nvmeBinary, opts, types.ExecuteTimeout)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(outputStr), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "nqn.") {
			return strings.TrimSpace(line), nil
		}
	}
	return "", fmt.Errorf("failed to get host NQN from %s", outputStr)
}

func listSubsystems(devicePath string, executor *commonns.Executor) ([]Subsystem, error) {
	major, _, err := cliVersion(executor)
	if err != nil {
		return nil, err
	}

	opts := []string{
		"-s", "SIGKILL",
		"-k", "1s",
		strconv.Itoa(nvmeListTimeoutSec),
		nvmeBinary,
		"list-subsys",
		"-o", "json",
	}

	if devicePath != "" {
		opts = append(opts, devicePath)
	}

	outputStr, err := executor.Execute(nil, "timeout", opts, (nvmeListTimeoutSec+2)*time.Second)
	if err != nil {
		return nil, err
	}
	jsonStr, err := extractJSONString(outputStr)
	if err != nil {
		return nil, err
	}

	if major == 1 {
		return listSubsystemsV1(jsonStr)
	}
	return listSubsystemsV2(jsonStr)
}

func listSubsystemsV1(jsonStr string) ([]Subsystem, error) {
	output := map[string][]Subsystem{}
	if err := json.Unmarshal([]byte(jsonStr), &output); err != nil {
		return nil, err
	}

	return output["Subsystems"], nil
}

type ListSubsystemsV2Output struct {
	HostNQN    string      `json:"HostNQN"`
	HostID     string      `json:"HostID"`
	Subsystems []Subsystem `json:"Subsystems"`
}

func listSubsystemsV2(jsonStr string) ([]Subsystem, error) {
	var output []ListSubsystemsV2Output
	if err := json.Unmarshal([]byte(jsonStr), &output); err != nil {
		return nil, err
	}

	subsystems := []Subsystem{}
	for _, o := range output {
		subsystems = append(subsystems, o.Subsystems...)
	}

	return subsystems, nil
}

// CliDevice fields use signed integers instead, because the output of buggy nvme-cli 2.x is possibly negative.
type CliDevice struct {
	NameSpace    int32  `json:"Namespace,omitempty"`
	DevicePath   string `json:"DevicePath,omitempty"`
	Firmware     string `json:"Firmware,omitempty"`
	Index        int32  `json:"Index,omitempty"`
	ModelNumber  string `json:"ModelNumber,omitempty"`
	SerialNumber string `json:"SerialNumber,omitempty"`
	UsedBytes    int64  `json:"UsedBytes,omitempty"`
	MaximumLBA   int64  `json:"MaximumLBA,omitempty"`
	PhysicalSize int64  `json:"PhysicalSize,omitempty"`
	SectorSize   int32  `json:"SectorSize,omitempty"`
}

var nvmeNamespaceRegex = regexp.MustCompile(`^nvme[0-9]+n([0-9]+)$`)

func listRecognizedNvmeDevices(executor *commonns.Executor) ([]CliDevice, error) {
	// 1. Fast Path: Directly query sysfs (/sys/block) to avoid spawning external processes and issuing blocking ioctls.
	// If /sys/block is successfully read, its result is authoritative (even if empty).
	devices, err := listRecognizedNvmeDevicesFromSysfs(sysfsBlockPath)
	if err == nil {
		return devices, nil
	}

	// If /sys/block is inaccessible (e.g. in restricted containers), try the subsystem path.
	devices, err = listRecognizedNvmeDevicesFromSysfs(sysfsNvmeSubsystemPath)
	if err == nil {
		return devices, nil
	}

	logrus.WithError(err).Debug("Failed to list NVMe devices from sysfs, falling back to nvme cli")

	// 2. Fallback Path: Wrap nvme list with the timeout(1) command using SIGKILL (-s SIGKILL -k 1s)
	opts := []string{
		"-s", "SIGKILL",
		"-k", "1s",
		strconv.Itoa(nvmeListTimeoutSec),
		nvmeBinary,
		"list",
		"-o", "json",
	}
	outputStr, err := executor.Execute(nil, "timeout", opts, (nvmeListTimeoutSec+2)*time.Second)
	if err != nil {
		return nil, err
	}
	jsonStr, err := extractJSONString(outputStr)
	if err != nil {
		return nil, err
	}
	output := map[string][]CliDevice{}
	if err := json.Unmarshal([]byte(jsonStr), &output); err != nil {
		return nil, err
	}

	return output["Devices"], nil
}

// parseSysfsNamespaceMetadata extracts NSID, SectorSize, PhysicalSize, MaximumLBA, and UsedBytes
// directly from sysfs without issuing blocking IOCTLs.
// blockDevDir is the canonical gendisk path (e.g. /sys/block/<nsName>).
func parseSysfsNamespaceMetadata(blockDevDir, nsName string) CliDevice {
	dev := CliDevice{
		DevicePath: filepath.Join("/dev", nsName),
		SectorSize: 512, // default fallback
	}

	// 1. NSID: read from /sys/block/<nsName>/nsid.
	// Note: The n<Y> suffix in the block device name (e.g. nvme0n1) is the subsystem's
	// instance counter (head->instance allocated from ns_ida), not the controller NSID
	// (head->ns_id). For example, a namespace with controller NSID 7 will be named nvme0n1
	// if it is the first namespace on the subsystem.
	if data, err := os.ReadFile(filepath.Join(blockDevDir, "nsid")); err == nil {
		if val, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 32); err == nil && val > 0 {
			dev.NameSpace = int32(val)
		}
	}
	// Fallback to regex submatch if nsid attribute file is unavailable (e.g. mock test environments)
	if dev.NameSpace == 0 {
		if matches := nvmeNamespaceRegex.FindStringSubmatch(nsName); len(matches) == 2 {
			if val, err := strconv.ParseInt(matches[1], 10, 32); err == nil {
				dev.NameSpace = int32(val)
			}
		}
	}

	// 2. SectorSize: try hw_sector_size first as it reflects the hardware LBA format (LBAF)
	// reported by Identify Namespace, then fall back to logical_block_size, taking the first valid positive value.
	for _, attr := range []string{"hw_sector_size", "logical_block_size"} {
		if data, err := os.ReadFile(filepath.Join(blockDevDir, "queue", attr)); err == nil {
			if val, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 32); err == nil && val > 0 {
				dev.SectorSize = int32(val)
				break
			}
		}
	}

	// 3. PhysicalSize: read from size (number of 512-byte sectors in the Linux block layer).
	if data, err := os.ReadFile(filepath.Join(blockDevDir, "size")); err == nil {
		if sectors, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil && sectors > 0 {
			dev.PhysicalSize = sectors * 512

			// 4. MaximumLBA: in nvme-cli / libnvme, MaximumLBA outputs the total LBA count
			// (lba_count = size >> (lba_shift - SECTOR_SHIFT) = PhysicalSize / SectorSize), not (lba_count - 1).
			if dev.SectorSize > 0 {
				dev.MaximumLBA = dev.PhysicalSize / int64(dev.SectorSize)
			}

			// 5. UsedBytes: in nvme-cli, UsedBytes is (nuse * lba_size), where nuse is Namespace Utilization.
			// Generic sysfs block layer attributes do not expose nuse without blocking NVMe ioctls.
			// Setting UsedBytes equal to PhysicalSize is valid for this target because SPDK NVMf reports
			// nuse == nsze for an exported bdev namespace.
			dev.UsedBytes = dev.PhysicalSize
		}
	}

	return dev
}

// listRecognizedNvmeDevicesFromSysfs scans sysfs (/sys/block or /sys/devices/virtual/nvme-subsystem)
// for active NVMe namespace block devices and populates metadata from the corresponding sysfs paths.
func listRecognizedNvmeDevicesFromSysfs(sysfsPath string) ([]CliDevice, error) {
	entries, err := os.ReadDir(sysfsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read sysfs path %s", sysfsPath)
	}

	var devices []CliDevice
	seen := make(map[string]bool)

	// Check direct entries (e.g. /sys/block/nvme*n*)
	for _, entry := range entries {
		if nvmeNamespaceRegex.MatchString(entry.Name()) {
			devNode := filepath.Join("/dev", entry.Name())
			if !seen[devNode] {
				seen[devNode] = true
				blockDevDir := filepath.Join(sysfsPath, entry.Name())
				devices = append(devices, parseSysfsNamespaceMetadata(blockDevDir, entry.Name()))
			}
		}
	}

	// If no direct namespace entries found, check subsystem subdirectories (e.g. /sys/devices/virtual/nvme-subsystem/nvme-subsys*/nvme*n*)
	if len(devices) == 0 {
		var readErr error
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), "nvme-subsys") {
				subsysDir := filepath.Join(sysfsPath, entry.Name())
				nsEntries, err := os.ReadDir(subsysDir)
				if err != nil {
					readErr = err
					continue
				}

				for _, nsEntry := range nsEntries {
					if nvmeNamespaceRegex.MatchString(nsEntry.Name()) {
						devNode := filepath.Join("/dev", nsEntry.Name())
						if !seen[devNode] {
							seen[devNode] = true
							blockDevDir := filepath.Join(subsysDir, nsEntry.Name())
							devices = append(devices, parseSysfsNamespaceMetadata(blockDevDir, nsEntry.Name()))
						}
					}
				}
			}
		}
		if readErr != nil {
			return nil, errors.Wrapf(readErr, "failed to read subsystem entries under %s", sysfsPath)
		}
	}

	return devices, nil
}

func getHostID(executor *commonns.Executor) (string, error) {
	outputStr, err := executor.Execute(nil, "cat", []string{"/etc/nvme/hostid"}, types.ExecuteTimeout)
	if err == nil {
		return strings.TrimSpace(string(outputStr)), nil
	}

	outputStr, err = executor.Execute(nil, "cat", []string{"/sys/class/dmi/id/product_uuid"}, types.ExecuteTimeout)
	if err == nil {
		return strings.TrimSpace(string(outputStr)), nil
	}

	return "", err
}

func discovery(hostID, hostNQN, ip, port string, executor *commonns.Executor) ([]DiscoveryPageEntry, error) {
	ip = spdkutil.NormalizeNvmeAddr(ip)

	opts := []string{
		"discover",
		"-t", DefaultTransportType,
		// nvme-cli -a accepts bare IPv6 (no brackets). net.SplitHostPort callers
		// upstream strip brackets; util.NormalizeNvmeAddr is a safety net.
		"-a", ip,
		"-s", port,
		"-o", "json",
	}
	if hostID != "" {
		opts = append(opts, "-I", hostID)
	}
	if hostNQN != "" {
		opts = append(opts, "-q", hostNQN)
	}

	// A valid output is like below:
	// # nvme discover -t tcp -a 10.42.2.20 -s 20011 -o json
	//	{
	//		"device" : "nvme0",
	//		"genctr" : 2,
	//		"records" : [
	//		  {
	//			"trtype" : "tcp",
	//			"adrfam" : "ipv4",
	//			"subtype" : "nvme subsystem",
	//			"treq" : "not required",
	//			"portid" : 0,
	//			"trsvcid" : "20001",
	//			"subnqn" : "nqn.2023-01.io.longhorn.spdk:pvc-81bab972-8e6b-48be-b691-18eaa430a897-r-0881c7b4",
	//			"traddr" : "10.42.2.20",
	//			"sectype" : "none"
	//		  },
	//		  {
	//			"trtype" : "tcp",
	//			"adrfam" : "ipv4",
	//			"subtype" : "nvme subsystem",
	//			"treq" : "not required",
	//			"portid" : 0,
	//			"trsvcid" : "20011",
	//			"subnqn" : "nqn.2023-01.io.longhorn.spdk:pvc-5f94d59d-baec-40e5-9e8b-25b79909d14e-e-49c947f5",
	//			"traddr" : "10.42.2.20",
	//			"sectype" : "none"
	//		  }
	//		]
	//	  }

	// nvme discover does not respect the -s option, so we need to filter the output
	outputStr, err := executor.Execute(nil, nvmeBinary, opts, types.ExecuteTimeout)
	if err != nil {
		return nil, err
	}

	jsonStr, err := extractJSONString(outputStr)
	if err != nil {
		return nil, err
	}

	var output struct {
		Entries []DiscoveryPageEntry `json:"records"`
	}

	err = json.Unmarshal([]byte(jsonStr), &output)
	if err != nil {
		return nil, err
	}

	return output.Entries, nil
}

func connect(hostID, hostNQN, nqn, transpotType, ip, port string, nrIoQueues int32, executor *commonns.Executor) (string, error) {
	ip = spdkutil.NormalizeNvmeAddr(ip)

	var err error

	opts := []string{
		"connect",
		"-t", transpotType,
		"--nqn", nqn,
		"--ctrl-loss-tmo", strconv.Itoa(defaultCtrlLossTmo),
		"--keep-alive-tmo", strconv.Itoa(defaultKeepAliveTmo),
		"--reconnect-delay", strconv.Itoa(defaultReconnectDelay),
		// Spelled with underscores, unlike the options above.
		"--fast_io_fail_tmo", strconv.Itoa(defaultFastIOFailTmo),
		"-o", "json",
	}

	if nrIoQueues > 0 {
		opts = append(opts, "--nr-io-queues", strconv.Itoa(int(nrIoQueues)))
	}

	if hostID != "" {
		opts = append(opts, "-I", hostID)
	}
	if hostNQN != "" {
		opts = append(opts, "-q", hostNQN)
	}
	if ip != "" {
		// nvme-cli -a accepts bare IPv6 (no brackets). net.SplitHostPort callers
		// upstream strip brackets; util.NormalizeNvmeAddr is a safety net.
		opts = append(opts, "-a", ip)
	}
	if port != "" {
		opts = append(opts, "-s", port)
	}

	// The output example:
	// {
	//  "device" : "nvme0"
	// }
	outputStr, err := executor.Execute(nil, nvmeBinary, opts, types.ExecuteTimeout)
	if err != nil {
		return "", err
	}

	jsonStr, err := extractJSONString(outputStr)
	if err != nil {
		return "", err
	}

	output := map[string]string{}
	if err := json.Unmarshal([]byte(jsonStr), &output); err != nil {
		return "", err
	}

	return output["device"], nil
}

func disconnect(nqn string, executor *commonns.Executor) error {
	opts := []string{
		"disconnect",
		"--nqn", nqn,
	}

	// The output example:
	// NQN:nqn.2023-01.io.spdk:raid01 disconnected 1 controller(s)
	//
	// And trying to disconnect a non-existing target would return exit code 0
	//
	// This tears down every controller of the subsystem, including any the kernel
	// can block on, so keep the wait bounded like the per-controller disconnect.
	_, err := executor.Execute(nil, nvmeBinary, opts, types.NvmeDisconnectTimeout)
	if types.ErrorIsTimeoutExecuting(err) {
		logrus.WithError(err).Warnf("Timed out waiting for NVMe subsystem %s to disconnect, leaving it to the kernel", nqn)
		return nil
	}
	return err
}

// disconnectController disconnects a single NVMe controller by device name
// (e.g. "nvme4"). This removes one multipath path without affecting other
// controllers for the same subsystem NQN.
func disconnectController(controllerName string, executor *commonns.Executor) error {
	devPath := filepath.Join("/dev", controllerName)
	opts := []string{
		"disconnect",
		"--device", devPath,
	}
	// Deleting a controller that is stuck reconnecting can block in the kernel, so
	// keep the wait bounded.
	_, err := executor.Execute(nil, nvmeBinary, opts, types.NvmeDisconnectTimeout)
	if types.ErrorIsTimeoutExecuting(err) {
		// Giving up on the wait is the point of the bound. The kernel carries the
		// deletion on without us, and retrying only adds another blocked process.
		logrus.WithError(err).Warnf("Timed out waiting for NVMe controller %s to disconnect, leaving it to the kernel", devPath)
		return nil
	}
	return err
}

func extractJSONString(str string) (string, error) {
	startIndex := strings.Index(str, "{")
	startIndexBracket := strings.Index(str, "[")

	if startIndex == -1 && startIndexBracket == -1 {
		return "", fmt.Errorf("invalid JSON string")
	}

	if startIndex != -1 && (startIndexBracket == -1 || startIndex < startIndexBracket) {
		endIndex := strings.LastIndex(str, "}")
		if endIndex == -1 {
			return "", fmt.Errorf("invalid JSON string")
		}
		return str[startIndex : endIndex+1], nil
	} else if startIndexBracket != -1 {
		endIndex := strings.LastIndex(str, "]")
		if endIndex == -1 {
			return "", fmt.Errorf("invalid JSON string")
		}
		return str[startIndexBracket : endIndex+1], nil
	}

	return "", fmt.Errorf("invalid JSON string")
}

// GetIPAndPortFromControllerAddress returns the IP and port from the controller address.
// Input can be either "traddr=10.42.2.18 trsvcid=20006" or "traddr=10.42.2.18,trsvcid=20006"
// for IPv4, or "traddr=fd00::1 trsvcid=20006" for IPv6 (traddr may contain colons).
func GetIPAndPortFromControllerAddress(address string) (string, string) {
	var traddr, trsvcid string

	parts := strings.FieldsFunc(address, func(r rune) bool {
		return r == ',' || r == ' '
	})

	for _, part := range parts {
		keyVal := strings.SplitN(part, "=", 2)
		if len(keyVal) == 2 {
			key := strings.TrimSpace(keyVal[0])
			value := strings.TrimSpace(keyVal[1])
			switch key {
			case "traddr":
				traddr = value
			case "trsvcid":
				trsvcid = value
			}
		}
	}

	return traddr, trsvcid
}

func flush(devicePath, namespaceID string, executor *commonns.Executor) (string, error) {

	opts := []string{
		"flush",
		devicePath,
		"-o", "json",
	}

	if namespaceID != "" {
		opts = append(opts, "-n", namespaceID)
	}

	return executor.Execute(nil, nvmeBinary, opts, types.ExecuteTimeout)
}
