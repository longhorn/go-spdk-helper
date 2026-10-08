package spdksetup

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	commontypes "github.com/longhorn/go-common-libs/types"
	spdksetup "github.com/longhorn/go-spdk-helper/pkg/spdk/setup"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func Cmd() *cli.Command {
	return &cli.Command{
		Name:    "spdk-setup",
		Aliases: []string{"setup"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "host-proc",
				Usage: fmt.Sprintf("The host proc path of namespace executor. By default %v", commontypes.ProcDirectory),
				Value: commontypes.ProcDirectory,
			},
		},
		Commands: []*cli.Command{
			BindCmd(),
			UnbindCmd(),
			DiskDriverCmd(),
			DiskStatusCmd(),
		},
	}
}

func BindCmd() *cli.Command {
	return &cli.Command{
		Name: "bind",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "device-driver",
				Usage:    "The userspace I/O driver to bind to",
				Required: true,
			},
		},
		Usage: "Bind the device to the specified userspace I/O driver: bind --device-driver <driver name> <device address>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := bind(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to bind device %v to driver %v", c.Args().First(), c.String("device-driver"))
				return err
			}
			return nil
		},
	}
}

func bind(c *cli.Command) error {
	executor, err := util.NewExecutor(c.String("host-proc"))
	if err != nil {
		return err
	}

	deviceAddr := c.Args().First()

	_, err = spdksetup.Bind(deviceAddr, c.String("device-driver"), executor)
	return err
}

func UnbindCmd() *cli.Command {
	return &cli.Command{
		Name:  "unbind",
		Flags: []cli.Flag{},
		Usage: "Unbind the device from the userspace I/O driver: unbind <device address>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := unbind(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to unbind device %v", c.Args().First())
				return err
			}
			return nil
		},
	}
}

func unbind(c *cli.Command) error {
	executor, err := util.NewExecutor(c.String("host-proc"))
	if err != nil {
		return err
	}

	deviceAddr := c.Args().First()

	_, err = spdksetup.Unbind(deviceAddr, executor)
	return err
}

func DiskDriverCmd() *cli.Command {
	return &cli.Command{
		Name:  "disk-driver",
		Flags: []cli.Flag{},
		Usage: "Get the driver name associated with a given PCI device's BDF address: disk-driver <device address>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := diskDriverCmd(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to get disk driver of device %v", c.Args().First())
				return err
			}
			return nil
		},
	}
}

func diskDriverCmd(c *cli.Command) error {
	executor, err := util.NewExecutor(c.String("host-proc"))
	if err != nil {
		return err
	}

	deviceAddr := c.Args().First()

	output, err := spdksetup.GetDiskDriver(deviceAddr, executor)
	if err != nil {
		return err
	}

	fmt.Println(output)

	return nil
}

func DiskStatusCmd() *cli.Command {
	return &cli.Command{
		Name:  "disk-status",
		Flags: []cli.Flag{},
		Usage: "Get the disk status of the device: disk-status <device address>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := diskStatusCmd(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to get disk status of device %v", c.Args().First())
				return err
			}
			return nil
		},
	}
}

func diskStatusCmd(c *cli.Command) error {
	executor, err := util.NewExecutor(c.String("host-proc"))
	if err != nil {
		return err
	}

	deviceAddr := c.Args().First()

	status, err := spdksetup.GetDiskStatus(deviceAddr, executor)
	if err != nil {
		return err
	}

	output, err := json.Marshal(status)
	if err != nil {
		return err
	}

	fmt.Println(string(output))

	return nil
}
