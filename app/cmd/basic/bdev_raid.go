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
			BdevRaidGetBaseBdevDeltaMapCmd(),
			BdevRaidStopBaseBdevDeltaMapCmd(),
			BdevRaidClearBaseBdevFaultyStateCmd(),
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
				Name:     "UUID",
				Usage:    "UUID for this raid bdev",
				Required: false,
			},
			cli.BoolFlag{
				Name:     "superblock",
				Usage:    "Raid bdev info will be stored in superblock on each base bdev",
				Required: false,
			},
			cli.BoolFlag{
				Name:     "delta-bitmap",
				Usage:    "A delta bitmap for faulty base bdevs will be recorded",
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

	created, err := spdkCli.BdevRaidCreate(c.String("name"), spdktypes.BdevRaidLevel(c.String("level")), uint32(c.Uint64("strip-size-kb")), c.StringSlice("base-bdevs"),
		c.String("UUID"), c.Bool("superblock"), c.Bool("delta-bitmap"))
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

func BdevRaidGetBaseBdevDeltaMapCmd() cli.Command {
	return cli.Command{
		Name:      "get-base-bdev-delta-map",
		Usage:     "get the delta bitmap of a faulty base bdev",
		ArgsUsage: "<BASE BDEV NAME>",
		Action: func(c *cli.Context) {
			if c.NArg() != 1 {
				logrus.Fatal("BASE BDEV NAME argument required")
			}
			if err := bdevRaidGetBaseBdevDeltaMap(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get base bdev delta map to raid command")
			}
		},
	}
}

func bdevRaidGetBaseBdevDeltaMap(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	deltaMap, err := spdkCli.BdevRaidGetBaseBdevDeltaMap(c.Args().First())
	if err != nil {
		return err
	}

	return util.PrintObject(deltaMap)
}

func BdevRaidStopBaseBdevDeltaMapCmd() cli.Command {
	return cli.Command{
		Name:      "stop-base-bdev-delta-map",
		Usage:     "stop the updating of the delta bitmap of a faulty base bdev",
		ArgsUsage: "<BASE BDEV NAME>",
		Action: func(c *cli.Context) {
			if c.NArg() != 1 {
				logrus.Fatal("BASE BDEV NAME argument required")
			}
			if err := bdevRaidStopBaseBdevDeltaMap(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run stop base bdev delta map to raid command")
			}
		},
	}
}

func bdevRaidStopBaseBdevDeltaMap(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	stopped, err := spdkCli.BdevRaidStopBaseBdevDeltaMap(c.Args().First())
	if err != nil {
		return err
	}

	return util.PrintObject(stopped)
}

func BdevRaidClearBaseBdevFaultyStateCmd() cli.Command {
	return cli.Command{
		Name:      "clear-base-bdev-faulty-state",
		Usage:     "clear the faulty state of a base bdev",
		ArgsUsage: "<BASE BDEV NAME>",
		Action: func(c *cli.Context) {
			if c.NArg() != 1 {
				logrus.Fatal("BASE BDEV NAME argument required")
			}
			if err := bdevRaidClearBaseBdevFaultyState(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run clear base bdev faulty state to raid command")
			}
		},
	}
}

func bdevRaidClearBaseBdevFaultyState(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	cleared, err := spdkCli.BdevRaidClearBaseBdevFaultyState(c.Args().First())
	if err != nil {
		return err
	}

	return util.PrintObject(cleared)
}
