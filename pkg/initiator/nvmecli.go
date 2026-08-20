package initiator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	sysfsNvmeSubsystemPath = "/sys/devices/virtual/nvme-subsystem"

	DefaultTransportType = "tcp"

	// Set short ctrlLossTimeoutSec for quick response to the controller loss.
	defaultCtrlLossTmo    = 30
	defaultKeepAliveTmo   = 5
	defaultReconnectDelay = 2
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
		"list-subsys",
		"-o", "json",
	}

	if devicePath != "" {
		opts = append(opts, devicePath)
	}

	outputStr, err := executor.Execute(nil, nvmeBinary, opts, types.ExecuteTimeout)
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

func listRecognizedNvmeDevices(executor *commonns.Executor) ([]CliDevice, error) {
	// 1. Fast Path: Directly query sysfs to avoid spawning external processes and issuing blocking ioctls
	devices, err := listRecognizedNvmeDevicesFromSysfs(sysfsNvmeSubsystemPath)
	if err == nil && len(devices) > 0 {
		return devices, nil
	}
	if err != nil {
		logrus.WithError(err).Debug("Failed to list NVMe devices from sysfs, falling back to nvme cli")
	}

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

func listRecognizedNvmeDevicesFromSysfs(sysfsPath string) ([]CliDevice, error) {
	subsysDirs, err := filepath.Glob(filepath.Join(sysfsPath, "nvme-subsys*"))
	if err != nil || len(subsysDirs) == 0 {
		return nil, errors.Errorf("no nvme subsystems found in %s: %v", sysfsPath, err)
	}

	var devices []CliDevice
	for _, subsysDir := range subsysDirs {
		nsEntries, err := os.ReadDir(subsysDir)
		if err != nil {
			continue
		}

		for _, entry := range nsEntries {
			// Look for block namespace entries like nvme0n1, nvme1n1, etc.
			if strings.HasPrefix(entry.Name(), "nvme") && strings.Contains(entry.Name(), "n") {
				devNode := filepath.Join("/dev", entry.Name())
				devices = append(devices, CliDevice{
					DevicePath: devNode,
				})
			}
		}
	}

	if len(devices) == 0 {
		return nil, errors.New("no recognized nvme devices in sysfs")
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

	opts := []string{
		"connect",
		"-t", transpotType,
		"--nqn", nqn,
		"--ctrl-loss-tmo", strconv.Itoa(defaultCtrlLossTmo),
		"--keep-alive-tmo", strconv.Itoa(defaultKeepAliveTmo),
		"--reconnect-delay", strconv.Itoa(defaultReconnectDelay),
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

	_, err := executor.Execute(nil, nvmeBinary, opts, types.ExecuteTimeout)
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
	_, err := executor.Execute(nil, nvmeBinary, opts, types.ExecuteTimeout)
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
