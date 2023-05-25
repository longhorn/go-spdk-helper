package basic

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func BdevLvolCmd() cli.Command {
	return cli.Command{
		Name:      "bdev-lvol",
		ShortName: "lvol",
		Subcommands: []cli.Command{
			BdevLvolCreateCmd(),
			BdevLvolDeleteCmd(),
			BdevLvolGetCmd(),
			BdevLvolSnapshotCmd(),
			BdevLvolCloneCmd(),
			BdevLvolDecoupleParentCmd(),
			BdevLvolResizeCmd(),
		},
	}
}

func BdevLvolCreateCmd() cli.Command {
	return cli.Command{
		Name: "create",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "lvs-name",
				Usage: "Specify this or lvs-uuid",
			},
			cli.StringFlag{
				Name:  "lvs-uuid",
				Usage: "Specify this or lvs-name",
			},
			cli.StringFlag{
				Name:     "lvol-name",
				Required: true,
			},
			cli.Uint64Flag{
				Name:     "size",
				Usage:    "Specify bdev lvol size in MiB",
				Required: true,
			},
		},
		Usage: "create a bdev lvol on a lvstore: \"create --lvs-name <LVSTORE NAME> --lvol-name <LVOL NAME> --size <LVOL SIZE in MIB>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolCreate(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run create bdev lvol command")
			}
		},
	}
}

func bdevLvolCreate(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	lvsName, lvsUUID := c.String("lvs-name"), c.String("lvs-uuid")
	lvolName := c.String("lvol-name")
	size := c.Uint64("size")

	uuid, err := spdkCli.BdevLvolCreate(lvsName, lvsUUID, lvolName, size,
		spdktypes.BdevLvolClearMethodUnmap, true)
	if err != nil {
		return err
	}

	return util.PrintObject(map[string]string{"uuid": uuid, "alias": fmt.Sprintf("%s/%s", lvsName, lvolName)})
}

func BdevLvolDeleteCmd() cli.Command {
	return cli.Command{
		Name: "delete",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "alias",
				Usage: "The alias of a lvol is <LVSTORE NAME>/<LVOL NAME>. Specify this or uuid",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "Specify this or alias",
			},
		},
		Usage: "delete a bdev lvol using a block device: \"delete --alias <LVSTORE NAME>/<LVOL NAME>\" or \"delete --uuid <UUID>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolDelete(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run delete bdev lvol command")
			}
		},
	}
}

func bdevLvolDelete(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	name := c.String("alias")
	if name == "" {
		name = c.String("uuid")
	}

	deleted, err := spdkCli.BdevLvolDelete(name)
	if err != nil {
		return err
	}

	return util.PrintObject(deleted)
}

func BdevLvolGetCmd() cli.Command {
	return cli.Command{
		Name: "get",
		Flags: []cli.Flag{
			cli.Uint64Flag{
				Name:  "timeout, t",
				Usage: "Determine the timeout of the execution",
				Value: 0,
			},
		},
		Usage: "get all bdev lvol if the info is not specified: \"get\", or \"get <LVSTORE NAME>/<LVOL NAME>\", or \"get <UUID>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolGet(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get bdev lvol command")
			}
		},
	}
}

func bdevLvolGet(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	bdevLvolGetResp, err := spdkCli.BdevLvolGet(c.Args().First(), c.Uint64("timeout"))
	if err != nil {
		return err
	}

	return util.PrintObject(bdevLvolGetResp)
}

func BdevLvolSnapshotCmd() cli.Command {
	return cli.Command{
		Name: "snapshot",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "alias",
				Usage: "The alias of a lvol is <LVSTORE NAME>/<LVOL NAME>. Specify this or uuid",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "Specify this or alias",
			},
			cli.StringFlag{
				Name:     "snapshot-name",
				Usage:    "The snapshot lvol name",
				Required: true,
			},
		},
		Usage: "create a snapshot as a new bdev lvol based on an existing one: \"snapshot --alias <LVSTORE NAME>/<LVOL NAME> --snapshot-name <SNAPSHOT NAME>\", or \"snapshot --uuid <UUID> --snapshot-name <SNAPSHOT NAME>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolSnapshot(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run snapshot bdev lvol command")
			}
		},
	}
}

func bdevLvolSnapshot(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	name := c.String("alias")
	if name == "" {
		name = c.String("uuid")
	}

	uuid, err := spdkCli.BdevLvolSnapshot(name, c.String("snapshot-name"))
	if err != nil {
		return err
	}

	return util.PrintObject(uuid)
}

func BdevLvolCloneCmd() cli.Command {
	return cli.Command{
		Name: "clone",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "alias",
				Usage: "The alias of a snapshot lvol is <LVSTORE NAME>/<LVOL NAME>. Specify this or uuid",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "Specify this or alias",
			},
			cli.StringFlag{
				Name:     "clone-name",
				Usage:    "The cloned lvol name",
				Required: true,
			},
		},
		Usage: "create a clone lvol based on an existing snapshot lvol: \"clone --alias <LVSTORE NAME>/<SNAPSHOT LVOL NAME> --clone-name <CLONE NAME>\", or \"clone --uuid <SNAPSHOT LVOL UUID> --clone-name <CLONE NAME>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolClone(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run clone bdev lvol command")
			}
		},
	}
}

func bdevLvolClone(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	name := c.String("alias")
	if name == "" {
		name = c.String("uuid")
	}

	uuid, err := spdkCli.BdevLvolClone(name, c.String("clone-name"))
	if err != nil {
		return err
	}

	return util.PrintObject(uuid)
}

func BdevLvolDecoupleParentCmd() cli.Command {
	return cli.Command{
		Name: "decouple",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "alias",
				Usage: "The alias of a lvol is <LVSTORE NAME>/<LVOL NAME>. Specify this or uuid",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "Specify this or alias",
			},
		},
		Usage: "decouple a lvol from its parent lvol: \"decouple --alias <LVSTORE NAME>/<LVOL NAME>\", or \"decouple --uuid <LVOL UUID>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolDecoupleParent(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run decouple parent bdev lvol command")
			}
		},
	}
}

func bdevLvolDecoupleParent(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	name := c.String("alias")
	if name == "" {
		name = c.String("uuid")
	}

	decoupled, err := spdkCli.BdevLvolDecoupleParent(name)
	if err != nil {
		return err
	}

	return util.PrintObject(decoupled)
}

func BdevLvolResizeCmd() cli.Command {
	return cli.Command{
		Name: "resize",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "alias",
				Usage: "The alias of a snapshot lvol is <LVSTORE NAME>/<LVOL NAME>. Specify this or uuid",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "Specify this or alias",
			},
			cli.Uint64Flag{
				Name:     "size",
				Required: true,
			},
		},
		Usage: "resize a lvol to a new size: \"resize --alias <LVSTORE NAME>/<LVOL NAME> --size <SIZE>\", or \"resize --uuid <LVOL UUID> --size <SIZE>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolResize(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run resize bdev lvol command")
			}
		},
	}
}

func bdevLvolResize(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	name := c.String("alias")
	if name == "" {
		name = c.String("uuid")
	}

	resized, err := spdkCli.BdevLvolResize(name, c.Uint64("size"))
	if err != nil {
		return err
	}

	return util.PrintObject(resized)
}
