package basic

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func BdevRaidCmd() cli.Command {
	return cli.Command{
		Name:      "bdev-raid",
		ShortName: "raid",
		Subcommands: []cli.Command{
			BdevRaidCreateCmd(),
			BdevRaidDeleteCmd(),
			BdevRaidGetCmd(),
			BdevRaidRemoveBaseBdevCmd(),
			BdevRaidGrowBaseBdevCmd(),
		},
	}
}

func BdevRaidCreateCmd() cli.Command {
	return cli.Command{
		Name:  "create",
		Usage: "create a bdev raid based on a bunch of existing bdevs: create --name <RAID NAME> --level <RAID LEVEL> --strip-size-kb <STRIP SIZE KB> --base-bdevs <BASE BDEV1> --base-bdevs <BASE BDEV2> ...",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "name,n",
				Usage:    "User defined raid bdev name",
				Required: true,
			},
			cli.StringFlag{
				Name:  "level,l",
				Usage: "Raid level of raid bdev, it can be \"0\"/\"raid0\", \"1\"/\"raid1\", \"5f\"/\"raid5f\", or \"concat\"",
				Value: string(spdktypes.BdevRaidLevelRaid1),
			},
			cli.Uint64Flag{
				Name:  "strip-size-kb,s",
				Usage: "The strip size of raid bdev in KB, supported values like 0, 4, 8, 16, 32, 64, 128, 256, etc. This works when the raid level is \"0\" or \"5f\"",
				Value: 0,
			},
			cli.StringSliceFlag{
				Name:     "base-bdevs,b",
				Usage:    "Names of Nvme bdevs, the input is like \"--base-devs Nvme0n1 --base-devs Nvme1n1\"",
				Required: true,
			},
			cli.StringFlag{
				Name:     "uuid,u",
				Usage:    "User defined raid bdev uuid, optional",
				Required: false,
			},
		},
		Action: func(c *cli.Context) {
			if err := bdevRaidCreate(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run create bdev raid command")
			}
		},
	}
}

func bdevRaidCreate(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	created, err := spdkCli.BdevRaidCreate(c.String("name"), spdktypes.BdevRaidLevel(c.String("level")), uint32(c.Uint64("strip-size-kb")), c.StringSlice("base-bdevs"), c.String("uuid"))
	if err != nil {
		return err
	}

	return util.PrintObject(created)
}

func BdevRaidDeleteCmd() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "delete a bdev raid using a block device: delete <RAID NAME>",
		Action: func(c *cli.Context) {
			if err := bdevRaidDelete(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run delete bdev raid command")
			}
		},
	}
}

func bdevRaidDelete(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	deleted, err := spdkCli.BdevRaidDelete(c.Args().First())
	if err != nil {
		return err
	}

	return util.PrintObject(deleted)
}

func BdevRaidGetCmd() cli.Command {
	return cli.Command{
		Name: "get",
		Flags: []cli.Flag{
			cli.Uint64Flag{
				Name:  "timeout, t",
				Usage: "Determine the timeout of the execution",
				Value: 0,
			},
		},
		Usage: "get all RAID bdevs if a RAID bdev name is not specified: \"get\", or \"get <RAID BDEV NAME>\"",
		Action: func(c *cli.Context) {
			if err := bdevRaidGet(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get bdev raid command")
			}
		},
	}
}

func bdevRaidGet(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	bdevRaidGetResp, err := spdkCli.BdevRaidGet(c.Args().First(), c.Uint64("timeout"))
	if err != nil {
		return err
	}

	return util.PrintObject(bdevRaidGetResp)
}

func BdevRaidRemoveBaseBdevCmd() cli.Command {
	return cli.Command{
		Name:  "remove-base-bdev",
		Usage: "remove base bdev from a raid bdev: remove-base-bdev <BASE BDEV NAME>",
		Action: func(c *cli.Context) {
			if err := bdevRaidRemoveBaseBdev(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run remove base bdev from raid command")
			}
		},
	}
}

func bdevRaidRemoveBaseBdev(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	deleted, err := spdkCli.BdevRaidRemoveBaseBdev(c.Args().First())
	if err != nil {
		return err
	}

	return util.PrintObject(deleted)
}

func BdevRaidGrowBaseBdevCmd() cli.Command {
	return cli.Command{
		Name:  "grow-base-bdev",
		Usage: "add a bdev to the base bdev list of an existing raid bdev, grow the raid's size if there isn't an empty base bdev slot: grow-base-bdev --raid-name <RAID BDEV NAME> --base-name <BASE BDEV NAME>",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "raid-name",
				Required: true,
			},
			cli.StringFlag{
				Name:     "base-name",
				Required: true,
			},
		},
		Action: func(c *cli.Context) {
			if err := bdevRaidGrowBaseBdev(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run grow base bdev to raid command")
			}
		},
	}
}

func bdevRaidGrowBaseBdev(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	growed, err := spdkCli.BdevRaidGrowBaseBdev(c.String("raid-name"), c.String("base-name"))
	if err != nil {
		return err
	}

	return util.PrintObject(growed)
}
