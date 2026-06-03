package basic

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func BdevEcCmd() cli.Command {
	return cli.Command{
		Name:      "bdev-ec",
		ShortName: "ec",
		Subcommands: []cli.Command{
			BdevEcCreateCmd(),
			BdevEcDeleteCmd(),
			BdevEcGetCmd(),
		},
	}
}

func BdevEcCreateCmd() cli.Command {
	return cli.Command{
		Name:  "create",
		Usage: "create an EC bdev: create --name <NAME> --data-chunks <DATA CHUNKS> --parity-chunks <PARITY CHUNKS> --strip-size-kb <KB> --base-bdevs <BDEV1> --base-bdevs <BDEV2> ...",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "name,n",
				Usage:    "Name for the new EC bdev",
				Required: true,
			},
			cli.UintFlag{
				Name:     "data-chunks",
				Usage:    "Number of data chunks per stripe",
				Required: true,
			},
			cli.UintFlag{
				Name:     "parity-chunks",
				Usage:    "Number of parity chunks per stripe",
				Required: true,
			},
			cli.UintFlag{
				Name:     "strip-size-kb,s",
				Usage:    "Chunk size in KiB (e.g. 64)",
				Required: true,
			},
			cli.StringSliceFlag{
				Name:     "base-bdevs,b",
				Usage:    "Ordered list of (data + parity) base bdev names, e.g. --base-bdevs bdev0 --base-bdevs bdev1",
				Required: true,
			},
			cli.BoolFlag{
				Name:  "salvage",
				Usage: "Refuse to fresh-zero a torn on-disk unmapped bitmap; set on operator-driven recovery",
			},
		},
		Action: func(c *cli.Context) {
			if err := bdevEcCreate(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run create bdev ec command")
			}
		},
	}
}

func bdevEcCreate(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	bdevName, err := spdkCli.BdevEcCreate(
		c.String("name"),
		uint32(c.Uint("data-chunks")),
		uint32(c.Uint("parity-chunks")),
		uint32(c.Uint("strip-size-kb")),
		c.StringSlice("base-bdevs"),
		c.Bool("salvage"),
	)
	if err != nil {
		return err
	}

	return util.PrintObject(bdevName)
}

func BdevEcDeleteCmd() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "delete an EC bdev: delete <NAME>",
		Action: func(c *cli.Context) {
			if err := bdevEcDelete(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run delete bdev ec command")
			}
		},
	}
}

func bdevEcDelete(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	deleted, err := spdkCli.BdevEcDelete(c.Args().First())
	if err != nil {
		return err
	}

	return util.PrintObject(deleted)
}

func BdevEcGetCmd() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "list EC bdevs; optionally filter by name: get [<NAME>]",
		Action: func(c *cli.Context) {
			if err := bdevEcGet(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get bdev ec command")
			}
		},
	}
}

func bdevEcGet(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	bdevEcInfoList, err := spdkCli.BdevEcGetBdevs(c.Args().First())
	if err != nil {
		return err
	}

	return util.PrintObject(bdevEcInfoList)
}
