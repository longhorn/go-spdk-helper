package basic

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	"github.com/longhorn/go-spdk-helper/pkg/types"
)

func BdevLvstoreCmd() cli.Command {
	return cli.Command{
		Name:      "bdev-lvstore",
		ShortName: "lvs",
		Subcommands: []cli.Command{
			BdevLvstoreCreateCmd(),
			BdevLvstoreDeleteCmd(),
			BdevLvstoreGetCmd(),
			BdevLvstoreRenameCmd(),
		},
	}
}

func BdevLvstoreCreateCmd() cli.Command {
	return cli.Command{
		Name: "create",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "bdev-name",
				Usage: "Required. The bdev on which to construct logical volume store",
			},
			cli.StringFlag{
				Name:  "lvs-name",
				Usage: "Required. Name of the logical volume store to create",
			},
			cli.UintFlag{
				Name:  "cluster-size",
				Usage: "Optional. Logical volume store cluster size, by default 1MiB.",
				Value: types.MiB,
			},
		},
		Usage: "create a bdev lvstore based on a block device: \"create --bdev-name <BDEV NAME> --lvs-name <LVSTORE NAME>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvstoreCreate(c); err != nil {
				logrus.WithError(err).Fatalf("Error running create bdev lvstore command")
			}
		},
	}
}

func bdevLvstoreCreate(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	uuid, err := spdkCli.BdevLvolCreateLvstore(c.String("bdev-name"), c.String("lvs-name"), uint32(c.Uint("cluster-size")))
	if err != nil {
		return err
	}

	bdevLvstoreCreateRespJSON, err := json.Marshal(map[string]string{"uuid": uuid})
	if err != nil {
		return err
	}
	fmt.Println(string(bdevLvstoreCreateRespJSON))

	return nil
}

func BdevLvstoreRenameCmd() cli.Command {
	return cli.Command{
		Name: "rename",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "old-name",
				Usage: "Required. Old name of the logical volume store",
			},
			cli.StringFlag{
				Name:  "new-name",
				Usage: "Required. New name of the logical volume store",
			},
		},
		Usage: "rename a bdev lvstore: \"rename --old-name <OLD NAME> --new-name <NEW NAME>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvstoreRename(c); err != nil {
				logrus.WithError(err).Fatalf("Error running rename bdev lvstore command")
			}
		},
	}
}

func bdevLvstoreRename(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	renamed, err := spdkCli.BdevLvolRenameLvstore(c.String("old-name"), c.String("new-name"))
	if err != nil {
		return err
	}

	bdevLvstoreRenameRespJSON, err := json.Marshal(renamed)
	if err != nil {
		return err
	}
	fmt.Println(string(bdevLvstoreRenameRespJSON))

	return nil
}

func BdevLvstoreDeleteCmd() cli.Command {
	return cli.Command{
		Name: "delete",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "lvs-name",
				Usage: "Optional. Specify this or uuid",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "Optional. Specify this or lvs-name",
			},
		},
		Usage: "delete a bdev lvstore using a block device: \"delete --lvs-name <LVSTORE NAME>\" or \"delete --uuid <UUID>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvstoreDelete(c); err != nil {
				logrus.WithError(err).Fatalf("Error running delete bdev lvstore command")
			}
		},
	}
}

func bdevLvstoreDelete(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	deleted, err := spdkCli.BdevLvolDeleteLvstore(c.String("lvs-name"), c.String("uuid"))
	if err != nil {
		return err
	}

	bdevLvstoreDeleteRespJSON, err := json.Marshal(deleted)
	if err != nil {
		return err
	}
	fmt.Println(string(bdevLvstoreDeleteRespJSON))

	return nil
}

func BdevLvstoreGetCmd() cli.Command {
	return cli.Command{
		Name: "get",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "lvs-name",
				Usage: "Optional. If you want to get one specific Lvstore info, please input this or uuid",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "Optional. If you want to get one specific Lvstore info, please input this or lvs-name",
			},
		},
		Usage: "get all bdev lvstore if the info is not specified: \"get\", or \"get --lvs-name <LVSTORE NAME>\", or \"get --uuid <UUID>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvstoreGet(c); err != nil {
				logrus.WithError(err).Fatalf("Error running get bdev lvstore command")
			}
		},
	}
}

func bdevLvstoreGet(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	bdevLvstoreGetResp, err := spdkCli.BdevLvolGetLvstore(c.String("lvs-name"), c.String("uuid"))
	if err != nil {
		return err
	}

	bdevLvstoreGetRespJSON, err := json.MarshalIndent(bdevLvstoreGetResp, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(bdevLvstoreGetRespJSON))

	return nil
}
