package nvme

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func DiscoverTarget(ip, port string, executor util.Executor) (subnqn string, err error) {
	entries, err := discovery(ip, port, executor)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.TrsvcID == port {
			return entry.Subnqn, nil
		}
	}

	return "", fmt.Errorf("found empty subnqn after nvme discover for %s:%s", ip, port)
}

func ConnectTarget(ip, port, nqn string, executor util.Executor) (controllerName string, err error) {
	// Trying to connect an existing subsystem will error out with exit code 114.
	// Hence, it's better to check the existence first.
	if devices, err := GetDevices(ip, port, nqn, executor); err == nil && len(devices) > 0 {
		return devices[0].Controllers[0].Controller, nil
	}

	return connect(ip, port, nqn, executor)
}

func DisconnectTarget(nqn string, executor util.Executor) error {
	return disconnect(nqn, executor)
}

// GetDevices returns all devices
func GetDevices(ip, port, nqn string, executor util.Executor) (devices []Device, err error) {
	defer func() {
		err = errors.Wrapf(err, "failed to get devices for address %s:%s and nqn %s", ip, port, nqn)
	}()

	devices = []Device{}

	nvmeDevices, err := listControllers(executor)
	if err != nil {
		return nil, err
	}
	for _, d := range nvmeDevices {
		// Get subsystem
		subsystems, err := listSubsystems(d.DevicePath, executor)
		if err != nil {
			logrus.WithError(err).Warnf("failed to get subsystem for nvme device %s", d.DevicePath)
			continue
		}
		if len(subsystems) == 0 {
			return nil, fmt.Errorf("no subsystem found for nvme device %s", d.DevicePath)
		}
		if len(subsystems) > 1 {
			return nil, fmt.Errorf("multiple subsystems found for nvme device %s", d.DevicePath)
		}
		sys := subsystems[0]

		// Reconstruct controller list
		controllers := []Controller{}
		for _, path := range sys.Paths {
			controller := Controller{
				Controller: path.Name,
				Transport:  path.Transport,
				Address:    path.Address,
				State:      path.State,
			}
			controllers = append(controllers, controller)
		}

		namespace := Namespace{
			NameSpace:    filepath.Base(d.DevicePath),
			NSID:         d.NameSpace,
			UsedBytes:    d.UsedBytes,
			MaximumLBA:   d.MaximumLBA,
			PhysicalSize: d.PhysicalSize,
			SectorSize:   d.SectorSize,
		}

		device := Device{
			Subsystem:    sys.Name,
			SubsystemNQN: sys.NQN,
			Controllers:  controllers,
			Namespaces:   []Namespace{namespace},
		}

		devices = append(devices, device)
	}

	if nqn == "" {
		return devices, err
	}

	res := []Device{}
	for _, d := range devices {
		match := false
		if d.SubsystemNQN != nqn {
			continue
		}
		for _, c := range d.Controllers {
			controllerIP, controllerPort := GetIPAndPortFromControllerAddress(c.Address)
			if ip != "" && ip != controllerIP {
				continue
			}
			if port != "" && port != controllerPort {
				continue
			}
			match = true
			break
		}
		if len(d.Namespaces) == 0 {
			continue
		}
		if match {
			res = append(res, d)
		}
	}

	if len(res) == 0 {
		subsystems, err := listSubsystems("", executor)
		if err != nil {
			return nil, err
		}
		for _, sys := range subsystems {
			if sys.NQN != nqn {
				continue
			}
			for _, path := range sys.Paths {
				return nil, fmt.Errorf("subsystem NQN %s path %v address %v is in %s state",
					nqn, path.Name, path.Address, path.State)
			}
		}

		return nil, fmt.Errorf("cannot find a valid nvme device with subsystem NQN %s and address %s:%s", nqn, ip, port)
	}
	return res, nil
}
