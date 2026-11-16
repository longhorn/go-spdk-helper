package basic

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	"github.com/longhorn/go-spdk-helper/pkg/util"

	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
)

func BdevRaidCmd() *cli.Command {
	return &cli.Command{
		Name:    "bdev-raid",
		Aliases: []string{"raid"},
		Commands: []*cli.Command{
			BdevRaidCreateCmd(),
			BdevRaidDeleteCmd(),
			BdevRaidGetCmd(),
			BdevRaidRemoveBaseBdevCmd(),
			BdevRaidGrowBaseBdevCmd(),
			BdevRaidResizeCmd(),
		},
	}
}

func BdevRaidCreateCmd() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "create a bdev raid based on a bunch of existing bdevs: create --name <RAID NAME> --level <RAID LEVEL> --strip-size-kb <STRIP SIZE KB> --base-bdevs <BASE BDEV1> --base-bdevs <BASE BDEV2> ...",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Usage:    "User defined raid bdev name",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "level",
				Aliases: []string{"l"},
				Usage:   "Raid level of raid bdev, it can be \"0\"/\"raid0\", \"1\"/\"raid1\", \"5f\"/\"raid5f\", or \"concat\"",
				Value:   string(spdktypes.BdevRaidLevelRaid1),
			},
			&cli.Uint64Flag{
				Name:    "strip-size-kb",
				Aliases: []string{"s"},
				Usage:   "The strip size of raid bdev in KB, supported values like 0, 4, 8, 16, 32, 64, 128, 256, etc. This works when the raid level is \"0\" or \"5f\"",
				Value:   0,
			},
			&cli.StringSliceFlag{
				Name:     "base-bdevs",
				Aliases:  []string{"b"},
				Usage:    "Names of Nvme bdevs, the input is like \"--base-devs Nvme0n1 --base-devs Nvme1n1\"",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "uuid",
				Aliases:  []string{"u"},
				Usage:    "User defined raid bdev uuid, optional",
				Required: false,
			},
			&cli.BoolFlag{
				Name:  "disable-auto-resize",
				Usage: "Optional. If set, the raid bdev does not resize when a base bdev is resized, and grows only through bdev-raid resize",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := bdevRaidCreate(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run create bdev raid command")
				return err
			}
			return nil
		},
		DisableSliceFlagSeparator: true,
	}
}

func bdevRaidCreate(c *cli.Command) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	opts := spdktypes.BdevRaidCreateOptions{DisableAutoResize: c.Bool("disable-auto-resize")}
	created, err := spdkCli.BdevRaidCreateWithOptions(c.String("name"), spdktypes.BdevRaidLevel(c.String("level")), uint32(c.Uint64("strip-size-kb")), c.StringSlice("base-bdevs"), c.String("uuid"), opts)
	if err != nil {
		return err
	}

	return util.PrintObject(created)
}

func BdevRaidDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "delete a bdev raid using a block device: delete <RAID NAME>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := bdevRaidDelete(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run delete bdev raid command")
				return err
			}
			return nil
		},
	}
}

func bdevRaidDelete(c *cli.Command) error {
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

func BdevRaidGetCmd() *cli.Command {
	return &cli.Command{
		Name: "get",
		Flags: []cli.Flag{
			&cli.Uint64Flag{
				Name:    "timeout",
				Aliases: []string{"t"},
				Usage:   "Determine the timeout of the execution",
				Value:   0,
			},
		},
		Usage: "get all RAID bdevs if a RAID bdev name is not specified: \"get\", or \"get <RAID BDEV NAME>\"",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := bdevRaidGet(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get bdev raid command")
				return err
			}
			return nil
		},
	}
}

func bdevRaidGet(c *cli.Command) error {
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

func BdevRaidRemoveBaseBdevCmd() *cli.Command {
	return &cli.Command{
		Name:  "remove-base-bdev",
		Usage: "remove base bdev from a raid bdev: remove-base-bdev <BASE BDEV NAME>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := bdevRaidRemoveBaseBdev(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run remove base bdev from raid command")
				return err
			}
			return nil
		},
	}
}

func bdevRaidRemoveBaseBdev(c *cli.Command) error {
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

func BdevRaidGrowBaseBdevCmd() *cli.Command {
	return &cli.Command{
		Name:  "grow-base-bdev",
		Usage: "add a bdev to the base bdev list of an existing raid bdev, grow the raid's size if there isn't an empty base bdev slot: grow-base-bdev --raid-name <RAID BDEV NAME> --base-name <BASE BDEV NAME>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "raid-name",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "base-name",
				Required: true,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := bdevRaidGrowBaseBdev(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run grow base bdev to raid command")
				return err
			}
			return nil
		},
	}
}

func bdevRaidGrowBaseBdev(c *cli.Command) error {
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

func BdevRaidResizeCmd() *cli.Command {
	return &cli.Command{
		Name:  "resize",
		Usage: "recalculate the size of a raid bdev from its current base bdevs: resize [--expected-size-in-mib <MIB>] <RAID BDEV NAME>",
		Flags: []cli.Flag{
			&cli.Uint64Flag{
				Name:  "expected-size-in-mib",
				Usage: "Optional. The minimum size in MiB that the raid bdev must reach (raid1 only); otherwise the resize is refused and the size is unchanged",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := bdevRaidResize(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run resize bdev raid command")
				return err
			}
			return nil
		},
	}
}

func bdevRaidResize(c *cli.Command) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	resp, err := spdkCli.BdevRaidResize(c.Args().First(), c.Uint64("expected-size-in-mib"))
	if err != nil {
		return err
	}

	return util.PrintObject(resp)
}
