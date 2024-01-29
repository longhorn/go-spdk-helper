package spdksetup

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	commonTypes "github.com/longhorn/go-common-libs/types"
	spdksetup "github.com/longhorn/go-spdk-helper/pkg/spdk/setup"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func Cmd() cli.Command {
	return cli.Command{
		Name:      "spdk-setup",
		ShortName: "setup",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "host-proc",
				Usage: fmt.Sprintf("The host proc path of namespace executor. By default %v", commonTypes.ProcDirectory),
				Value: commonTypes.ProcDirectory,
			},
		},
		Subcommands: []cli.Command{
			BindCmd(),
			UnbindCmd(),
			DiskDriverCmd(),
			DiskStatusCmd(),
		},
	}
}

func BindCmd() cli.Command {
	return cli.Command{
		Name: "bind",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "device-driver",
				Usage:    "The userspace I/O driver to bind to",
				Required: true,
			},
		},
		Usage: "Bind the device to the specified userspace I/O driver: bind --device-driver <driver name> <device address>",
		Action: func(c *cli.Context) {
			if err := bind(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to bind device %v to driver %v", c.Args().First(), c.String("device-driver"))
			}
		},
	}
}

func bind(c *cli.Context) error {
	executor, err := util.NewExecutor(c.GlobalString("host-proc"))
	if err != nil {
		return err
	}

	deviceAddr := c.Args().First()

	_, err = spdksetup.Bind(deviceAddr, c.String("device-driver"), executor)
	return err
}

func UnbindCmd() cli.Command {
	return cli.Command{
		Name:  "unbind",
		Flags: []cli.Flag{},
		Usage: "Unbind the device from the userspace I/O driver: unbind <device address>",
		Action: func(c *cli.Context) {
			if err := unbind(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to unbind device %v", c.Args().First())
			}
		},
	}
}

func unbind(c *cli.Context) error {
	executor, err := util.NewExecutor(c.GlobalString("host-proc"))
	if err != nil {
		return err
	}

	deviceAddr := c.Args().First()

	_, err = spdksetup.Unbind(deviceAddr, executor)
	return err
}

func DiskDriverCmd() cli.Command {
	return cli.Command{
		Name:  "disk-driver",
		Flags: []cli.Flag{},
		Usage: "Get the driver name associated with a given PCI device's BDF address: disk-driver <device address>",
		Action: func(c *cli.Context) {
			if err := diskDriverCmd(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to get disk driver of device %v", c.Args().First())
			}
		},
	}
}

func diskDriverCmd(c *cli.Context) error {
	executor, err := util.NewExecutor(c.GlobalString("host-proc"))
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

func DiskStatusCmd() cli.Command {
	return cli.Command{
		Name:  "disk-status",
		Flags: []cli.Flag{},
		Usage: "Get the disk status of the device: disk-status <device address>",
		Action: func(c *cli.Context) {
			if err := diskStatusCmd(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to get disk status of device %v", c.Args().First())
			}
		},
	}
}

func diskStatusCmd(c *cli.Context) error {
	executor, err := util.NewExecutor(c.GlobalString("host-proc"))
	if err != nil {
		return err
	}

	deviceAddr := c.Args().First()

	output, err := spdksetup.GetDiskStatus(deviceAddr, executor)
	if err != nil {
		return err
	}

	fmt.Println(output)

	return nil
}
