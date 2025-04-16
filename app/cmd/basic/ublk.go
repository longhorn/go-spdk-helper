package basic

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func UblkCmd() cli.Command {
	return cli.Command{
		Name: "ublk",
		Subcommands: []cli.Command{
			UblkCreateTargetCmd(),
			UblkDestroyTargetCmd(),
			UblkGetDisksCmd(),
			UblkStartDiskCmd(),
			UblkRecoverDiskCmd(),
			UblkStopDiskCmd(),
		},
	}
}

func UblkCreateTargetCmd() cli.Command {
	return cli.Command{
		Name:  "create-target",
		Usage: "Start to create ublk threads and initialize ublk target",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "cpumask",
				Usage: "Specify CPU mask for ublk target. It will use current cpumask in SPDK when user does not specify cpumask option.",
			},
			cli.BoolFlag{
				Name:  "disable-user-copy",
				Usage: "Disable user copy feature",
			},
		},
		Action: func(c *cli.Context) {
			if err := ublkCreateTarget(c); err != nil {
				logrus.WithError(err).Fatal("Failed to run UblkCreateTargetCmd")
			}
		},
	}
}

func ublkCreateTarget(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	return spdkCli.UblkCreateTarget(c.String("cpumask"), c.Bool("disable-user-copy"))
}

func UblkDestroyTargetCmd() cli.Command {
	return cli.Command{
		Name:  "destroy-target",
		Usage: "Release all UBLK devices and destroy ublk target",
		Action: func(c *cli.Context) {
			if err := ublkDestroyTarget(c); err != nil {
				logrus.WithError(err).Fatal("Failed to run UblkDestroyTargetCmd")
			}
		},
	}
}

func ublkDestroyTarget(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}
	return spdkCli.UblkDestroyTarget()
}

func UblkGetDisksCmd() cli.Command {
	return cli.Command{
		Name:  "get-disks",
		Usage: "Display full or specified ublk device list",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "ublk-id",
				Usage: "ID of ublk device. If not specified or specified as 0, return the full device list"},
		},
		Action: func(c *cli.Context) {
			if err := ublkGetDisks(c); err != nil {
				logrus.WithError(err).Fatal("Failed to run UblkGetDisksCmd")
			}
		},
	}
}

func ublkGetDisks(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}
	resp, err := spdkCli.UblkGetDisks(int32(c.Int("ublk-id")))
	if err != nil {
		return err
	}
	return util.PrintObject(resp)
}

func UblkStartDiskCmd() cli.Command {
	return cli.Command{
		Name:  "start-disk",
		Usage: "Start to export one SPDK bdev as a UBLK device",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "bdev-name",
				Usage:    "Name of the bdev which will be exported",
				Required: true,
			},
			cli.IntFlag{
				Name:     "ublk-id",
				Usage:    "ID of ublk device being created",
				Required: true,
			},
			cli.IntFlag{
				Name:  "queue-depth",
				Usage: "Physical queue depth of ublk device",
				Value: 128,
			},
			cli.IntFlag{
				Name:  "num-queues",
				Usage: "Number of queues of ublk device",
				Value: 1,
			},
		},
		Action: func(c *cli.Context) {
			if err := ublkStartDisk(c); err != nil {
				logrus.WithError(err).Fatal("Failed to run UblkStartDiskCmd")
			}
		},
	}
}

func ublkStartDisk(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}
	return spdkCli.UblkStartDisk(c.String("bdev-name"), int32(c.Int("ublk-id")), int32(c.Int("queue-depth")), int32(c.Int("num-queues")))
}

func UblkRecoverDiskCmd() cli.Command {
	return cli.Command{
		Name:  "recover-disk",
		Usage: "Recover original UBLK device with ID and block device",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "bdev-name",
				Usage:    "Bdev name to export",
				Required: true,
			},
			cli.IntFlag{
				Name:     "ublk-id",
				Usage:    "Device id",
				Required: true,
			},
		},
		Action: func(c *cli.Context) {
			if err := ublkRecoverDisk(c); err != nil {
				logrus.WithError(err).Fatal("Failed to run UblkRecoverDiskCmd")
			}
		},
	}
}

func ublkRecoverDisk(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}
	return spdkCli.UblkRecoverDisk(c.String("bdev-name"), int32(c.Int("ublk-id")))
}

func UblkStopDiskCmd() cli.Command {
	return cli.Command{
		Name:  "stop-disk",
		Usage: "Delete a UBLK device",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:     "ublk-id",
				Usage:    "ID of ublk device being stopped",
				Required: true,
			},
		},
		Action: func(c *cli.Context) {
			if err := ublkStopDisk(c); err != nil {
				logrus.WithError(err).Fatal("Failed to run UblkStopDiskCmd")
			}
		},
	}
}

func ublkStopDisk(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}
	return spdkCli.UblkStopDisk(int32(c.Int("ublk-id")))
}
