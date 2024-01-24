package basic

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func BdevVirtioCmd() cli.Command {
	return cli.Command{
		Name:      "bdev-virtio",
		ShortName: "virtio",
		Subcommands: []cli.Command{
			BdevVirtioAttachControllerCmd(),
			BdevVirtioDetachControllerCmd(),
			// BdevVirtioGetCmd(),
		},
	}
}

func BdevVirtioAttachControllerCmd() cli.Command {
	return cli.Command{
		Name:  "attach",
		Usage: "attach a bdev virtio based on a block device: attach --name <BDEV NAME> --trtype <TRTYPE> --traddr <TRADDR> --dev-type <DEV TYPE>",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "trtype",
				Usage:    "Virtio target trtype: pci or user",
				Required: true,
			},
			cli.StringFlag{
				Name:     "traddr",
				Usage:    "Target address: BDF or UNIX socket file path",
				Required: true,
			},
			cli.StringFlag{
				Name:     "dev-type",
				Usage:    "Virtio device type: blk or scsi",
				Required: true,
			},
		},
		Action: func(c *cli.Context) {
			if err := bdevVirtioAttachController(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run attach bdev virtio command")
			}
		},
	}
}

func bdevVirtioAttachController(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	name := c.Args().First()

	bdevNameList, err := spdkCli.BdevVirtioAttachController(name, c.String("trtype"), c.String("traddr"), c.String("dev-type"))
	if err != nil {
		return err
	}

	return util.PrintObject(bdevNameList)
}

func BdevVirtioDetachControllerCmd() cli.Command {
	return cli.Command{
		Name:  "detach",
		Usage: "detach a bdev virtio using a block device: detach <BDEV NAME>",
		Action: func(c *cli.Context) {
			if err := bdevVirtioDetachControllerCmd(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run detach bdev virtio command")
			}
		},
	}
}

func bdevVirtioDetachControllerCmd(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	detached, err := spdkCli.BdevVirtioDetachController(c.Args().First())
	if err != nil {
		return err
	}

	return util.PrintObject(detached)
}
