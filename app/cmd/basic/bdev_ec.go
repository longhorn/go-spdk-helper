package basic

import (
	"context"
	"fmt"

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
			BdevEcReplaceCmd(),
			BdevEcRebuildStartCmd(),
			BdevEcRebuildStopCmd(),
			BdevEcRebuildProgressCmd(),
			BdevEcRebuildQosSetCmd(),
			BdevEcResizeCmd(),
			BdevEcWibStatusCmd(),
			BdevEcUnmapStatusCmd(),
			BdevEcScrubProgressCmd(),
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
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("EC bdev name is required")
	}

	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	deleted, err := spdkCli.BdevEcDelete(name)
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

func BdevEcReplaceCmd() cli.Command {
	return cli.Command{
		Name:  "replace",
		Usage: "hot-swap a failed base bdev slot with a new bdev: replace --name <NAME> --slot <SLOT> --new-bdev <NEW>",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "name,n",
				Usage:    "Name of the EC bdev",
				Required: true,
			},
			cli.UintFlag{
				Name:     "slot",
				Usage:    "Slot index of the failed base bdev to replace",
				Required: true,
			},
			cli.StringFlag{
				Name:     "new-bdev",
				Usage:    "Name of the replacement base bdev",
				Required: true,
			},
		},
		Action: func(c *cli.Context) {
			if err := bdevEcReplace(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run replace bdev ec command")
			}
		},
	}
}

func bdevEcReplace(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	resp, err := spdkCli.BdevEcReplaceBaseBdev(c.String("name"), uint32(c.Uint("slot")), c.String("new-bdev"))
	if err != nil {
		return err
	}

	return util.PrintObject(resp)
}

func BdevEcRebuildStartCmd() cli.Command {
	return cli.Command{
		Name:  "rebuild-start",
		Usage: "start background rebuild of all REPLACING slots: rebuild-start <NAME>",
		Action: func(c *cli.Context) {
			if err := bdevEcRebuildStart(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run rebuild-start bdev ec command")
			}
		},
	}
}

func bdevEcRebuildStart(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("EC bdev name is required")
	}

	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	resp, err := spdkCli.BdevEcStartRebuild(name)
	if err != nil {
		return err
	}

	return util.PrintObject(resp)
}

func BdevEcRebuildStopCmd() cli.Command {
	return cli.Command{
		Name:  "rebuild-stop",
		Usage: "stop a running rebuild: rebuild-stop <NAME>",
		Action: func(c *cli.Context) {
			if err := bdevEcRebuildStop(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run rebuild-stop bdev ec command")
			}
		},
	}
}

func bdevEcRebuildStop(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("EC bdev name is required")
	}

	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	stopped, err := spdkCli.BdevEcStopRebuild(name)
	if err != nil {
		return err
	}

	return util.PrintObject(stopped)
}

func BdevEcRebuildQosSetCmd() cli.Command {
	return cli.Command{
		Name:  "rebuild-qos-set",
		Usage: "set rebuild rate limit: rebuild-qos-set --name <NAME> --max-stripes-per-sec <N> [--paused]",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "name,n",
				Usage:    "Name of the EC bdev",
				Required: true,
			},
			cli.UintFlag{
				Name:  "max-stripes-per-sec",
				Usage: "Rebuild rate limit in stripes/sec; 0 means unlimited",
				Value: 0,
			},
			cli.BoolFlag{
				Name:  "paused",
				Usage: "Suspend the rebuild poller without cancelling it",
			},
		},
		Action: func(c *cli.Context) {
			if err := bdevEcRebuildQosSet(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run rebuild-qos-set bdev ec command")
			}
		},
	}
}

func bdevEcRebuildQosSet(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	set, err := spdkCli.BdevEcSetRebuildQos(c.String("name"), uint32(c.Uint("max-stripes-per-sec")), c.Bool("paused"))
	if err != nil {
		return err
	}

	return util.PrintObject(set)
}

func BdevEcRebuildProgressCmd() cli.Command {
	return cli.Command{
		Name:  "rebuild-progress",
		Usage: "query rebuild progress: rebuild-progress <NAME>",
		Action: func(c *cli.Context) {
			if err := bdevEcRebuildProgress(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run rebuild-progress bdev ec command")
			}
		},
	}
}

func bdevEcRebuildProgress(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("EC bdev name is required")
	}

	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	progress, err := spdkCli.BdevEcGetRebuildProgress(name)
	if err != nil {
		return err
	}

	return util.PrintObject(progress)
}

func BdevEcResizeCmd() cli.Command {
	return cli.Command{
		Name:  "resize",
		Usage: "expand EC bdev capacity in-place (base bdevs must be resized first): resize <NAME>",
		Action: func(c *cli.Context) {
			if err := bdevEcResize(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run resize bdev ec command")
			}
		},
	}
}

func bdevEcResize(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("EC bdev name is required")
	}

	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	resp, err := spdkCli.BdevEcResize(name)
	if err != nil {
		return err
	}

	return util.PrintObject(resp)
}

func BdevEcWibStatusCmd() cli.Command {
	return cli.Command{
		Name:  "wib-status",
		Usage: "query Write-Intent Bitmap state: wib-status <NAME>",
		Action: func(c *cli.Context) {
			if err := bdevEcWibStatus(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run wib-status bdev ec command")
			}
		},
	}
}

func bdevEcWibStatus(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("EC bdev name is required")
	}

	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	status, err := spdkCli.BdevEcGetWibStatus(name)
	if err != nil {
		return err
	}

	return util.PrintObject(status)
}

func BdevEcUnmapStatusCmd() cli.Command {
	return cli.Command{
		Name:  "unmap-status",
		Usage: "query in-band unmapped-bitmap state: unmap-status <NAME>",
		Action: func(c *cli.Context) {
			if err := bdevEcUnmapStatus(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run unmap-status bdev ec command")
			}
		},
	}
}

func bdevEcUnmapStatus(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("EC bdev name is required")
	}

	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	status, err := spdkCli.BdevEcGetUnmapStatus(name)
	if err != nil {
		return err
	}

	return util.PrintObject(status)
}

func BdevEcScrubProgressCmd() cli.Command {
	return cli.Command{
		Name:  "scrub-progress",
		Usage: "query startup scrub progress: scrub-progress <NAME>",
		Action: func(c *cli.Context) {
			if err := bdevEcScrubProgress(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run scrub-progress bdev ec command")
			}
		},
	}
}

func bdevEcScrubProgress(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("EC bdev name is required")
	}

	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	progress, err := spdkCli.BdevEcGetScrubProgress(name)
	if err != nil {
		return err
	}

	return util.PrintObject(progress)
}
