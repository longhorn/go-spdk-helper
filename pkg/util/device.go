package util

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	lsblkBinary    = "lsblk"
	blockdevBinary = "blockdev"
)

type KernelDevice struct {
	Name  string
	Major int
	Minor int

	dmDeviceName  string
	dmDeviceMajor int
	dmDeviceMinor int
}

func RemoveDevice(dev string) error {
	if _, err := os.Stat(dev); err == nil {
		if err := remove(dev); err != nil {
			return errors.Wrapf(err, "failed to removing device %s", dev)
		}
	}
	return nil
}

func GetKnownDevices(executor Executor) (map[string]*KernelDevice, error) {
	knownDevices := make(map[string]*KernelDevice)

	/* Example command output
	   $ lsblk -l -n -o NAME,MAJ:MIN
	   sda           8:0
	   sdb           8:16
	   sdc           8:32
	   nvme0n1     259:0
	   nvme0n1p1   259:1
	   nvme0n1p128 259:2
	   nvme1n1     259:3
	*/

	opts := []string{
		"-l", "-n", "-o", "NAME,MAJ:MIN",
	}

	output, err := executor.Execute(lsblkBinary, opts)
	if err != nil {
		return knownDevices, err
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		f := strings.Fields(line)
		if len(f) == 2 {
			dev := &KernelDevice{
				Name: f[0],
			}
			if _, err := fmt.Sscanf(f[1], "%d:%d", &dev.Major, &dev.Minor); err != nil {
				return nil, fmt.Errorf("invalid major:minor %s for device %s", dev.Name, f[1])
			}
			knownDevices[dev.Name] = dev
		}
	}

	return knownDevices, nil
}

func DetectDevice(path string, executor Executor) (*KernelDevice, error) {
	/* Example command output
	   $ lsblk -l -n <Device Path> -o NAME,MAJ:MIN
	   nvme1n1     259:3
	*/

	opts := []string{
		"-l", "-n", path, "-o", "NAME,MAJ:MIN",
	}

	output, err := executor.Execute(lsblkBinary, opts)
	if err != nil {
		return nil, err
	}

	var dev *KernelDevice
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		f := strings.Fields(line)
		if len(f) == 2 {
			dev = &KernelDevice{
				Name: f[0],
			}
			if _, err := fmt.Sscanf(f[1], "%d:%d", &dev.Major, &dev.Minor); err != nil {
				return nil, fmt.Errorf("invalid major:minor %s for device %s with path %s", dev.Name, f[1], path)
			}
		}
		break
	}
	if dev == nil {
		return nil, fmt.Errorf("failed to get device with path %s", path)
	}

	return dev, nil
}

type BlockDevice struct {
	MajMin string `json:"maj:min"`
}

type BlockDevices struct {
	Devices []BlockDevice `json:"blockdevices"`
}

func parseMajorMinorFromJSON(jsonStr string) (int, int, error) {
	var data BlockDevices
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to parse JSON")
	}

	if len(data.Devices) != 1 {
		return 0, 0, fmt.Errorf("number of devices is not 1")
	}

	majMinParts := splitMajMin(data.Devices[0].MajMin)

	if len(majMinParts) != 2 {
		return 0, 0, fmt.Errorf("invalid maj:min format: %s", data.Devices[0].MajMin)
	}

	major, err := parseNumber(majMinParts[0])
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to parse major number")
	}

	minor, err := parseNumber(majMinParts[1])
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to parse minor number")
	}

	return major, minor, nil
}

func splitMajMin(majMin string) []string {
	return splitIgnoreEmpty(majMin, ":")
}

func splitIgnoreEmpty(str string, sep string) []string {
	parts := []string{}
	for _, part := range strings.Split(str, sep) {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func parseNumber(str string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(str))
}

func SuspendDeviceMapperDevice(dmDeviceName string, executor Executor) error {
	logrus.Infof("Suspending device mapper device %s", dmDeviceName)

	return DmsetupSuspend(dmDeviceName, executor)
}

func ResumeDeviceMapperDevice(dmDeviceName string, executor Executor) error {
	logrus.Infof("Resuming device mapper device %s", dmDeviceName)

	return DmsetupResume(dmDeviceName, executor)
}

func ReloadDeviceMapperDevice(dmDeviceName string, dev *KernelDevice, executor Executor) error {
	devPath := fmt.Sprintf("/dev/%s", dev.Name)

	// Get the size of the device
	opts := []string{
		"--getsize", devPath,
	}
	output, err := executor.Execute(blockdevBinary, opts)
	if err != nil {
		return err
	}
	sectors, err := strconv.ParseInt(strings.TrimSpace(output), 10, 64)
	if err != nil {
		return err
	}

	table := fmt.Sprintf("0 %v linear %v 0", sectors, devPath)

	logrus.Infof("Reloading device mapper device %s with table %s", dmDeviceName, table)

	return DmsetupReload(dmDeviceName, table, executor)
}

func CreateDeviceMapperDevice(dmDeviceName string, dev *KernelDevice, executor Executor) error {
	if dev == nil {
		return fmt.Errorf("found nil device for device mapper device creation")
	}

	devPath := fmt.Sprintf("/dev/%s", dev.Name)

	// Get the size of the device
	opts := []string{
		"--getsize", devPath,
	}
	output, err := executor.Execute(blockdevBinary, opts)
	if err != nil {
		return err
	}
	sectors, err := strconv.ParseInt(strings.TrimSpace(output), 10, 64)
	if err != nil {
		return err
	}

	// Create a device mapper device with the same size as the original device
	table := fmt.Sprintf("0 %v linear %v 0", sectors, devPath)

	logrus.Infof("Creating device mapper device %s with table %s", dmDeviceName, table)
	err = DmsetupCreate(dmDeviceName, table, executor)
	if err != nil {
		return err
	}

	// Get the major:minor of the device mapper device
	opts = []string{
		"-l", "-J", "-n", "-o", "MAJ:MIN", fmt.Sprintf("/dev/mapper/%s", dmDeviceName),
	}
	output, err = executor.Execute(lsblkBinary, opts)
	if err != nil {
		return err
	}

	major, minor, err := parseMajorMinorFromJSON(output)
	if err != nil {
		return err
	}

	dev.dmDeviceName = dmDeviceName
	dev.dmDeviceMajor = major
	dev.dmDeviceMinor = minor

	return nil
}

func RemoveDeviceMapperDevice(dmDeviceName string, executor Executor) error {
	devicePath := fmt.Sprintf("/dev/mapper/%s", dmDeviceName)
	if _, err := os.Stat(devicePath); err != nil {
		logrus.WithError(err).Warnf("Failed to stat device %s", devicePath)
		return nil
	}

	logrus.Infof("Removing device mapper device %s", dmDeviceName)
	return DmsetupRemove(dmDeviceName, executor)
}

func DuplicateDevice(dev *KernelDevice, dest string) error {
	if dev == nil {
		return fmt.Errorf("found nil device for device duplication")
	}
	if dest == "" {
		return fmt.Errorf("found empty destination for device duplication")
	}
	dir := filepath.Dir(dest)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			logrus.WithError(err).Fatalf("device %v: Failed to create directory for %v", dev.Name, dest)
		}
	}
	if err := mknod(dest, dev.dmDeviceMajor, dev.dmDeviceMinor); err != nil {
		return errors.Wrapf(err, "cannot create device node %s for device %s", dest, dev.Name)
	}
	if err := os.Chmod(dest, 0660); err != nil {
		return errors.Wrapf(err, "cannot change permission of the device %s", dest)
	}
	return nil
}

func mknod(device string, major, minor int) error {
	var fileMode os.FileMode = 0660
	fileMode |= unix.S_IFBLK
	dev := int(unix.Mkdev(uint32(major), uint32(minor)))

	logrus.Infof("Creating device %s %d:%d", device, major, minor)
	return unix.Mknod(device, uint32(fileMode), dev)
}

func removeAsync(path string, done chan<- error) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		logrus.WithError(err).Errorf("Failed to remove %v", path)
		done <- err
	}
	done <- nil
}

func remove(path string) error {
	done := make(chan error)
	go removeAsync(path, done)
	select {
	case err := <-done:
		return err
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout trying to delete %s", path)
	}
}
