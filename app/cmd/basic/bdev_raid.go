package basic

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
)

func BdevRaidCmd() cli.Command {
	return cli.Command{
		Name:      "bdev-raid",
		ShortName: "raid",
		Subcommands: []cli.Command{
			BdevRaidCreateCmd(),
			BdevRaidDeleteCmd(),
			BdevRaidGetCmd(),
		},
	}
}

func BdevRaidCreateCmd() cli.Command {
	return cli.Command{
		Name:  "create",
		Usage: "create a bdev raid based on a bunch of existing bdevs: create --name <RAID NAME> --level <RAID LEVEL> --strip-size-kb <STRIP SIZE KB> --base-bdevs <BASE BDEV1> --base-bdevs <BASE BDEV2> ......",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "name,n",
				Usage: "User defined raid bdev name",
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
				Name:  "base-bdevs,b",
				Usage: "Names of Nvme bdevs, the input is like \"--base-devs Nvme0n1 --base-devs Nvme1n1\"",
			},
		},
		Action: func(c *cli.Context) {
			if err := bdevRaidCreate(c); err != nil {
				logrus.WithError(err).Fatalf("Error running create bdev raid command")
			}
		},
	}
}

func bdevRaidCreate(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	created, err := spdkCli.BdevRaidCreate(c.String("name"), spdktypes.BdevRaidLevel(c.String("level")), uint32(c.Uint64("strip-size-kb")), c.StringSlice("base-bdevs"))
	if err != nil {
		return err
	}

	bdevRaidCreateRespJSON, err := json.Marshal(created)
	if err != nil {
		return err
	}
	fmt.Println(string(bdevRaidCreateRespJSON))

	return nil
}

func BdevRaidDeleteCmd() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "delete a bdev raid using a block device: delete <RAID NAME>",
		Action: func(c *cli.Context) {
			if err := bdevRaidDelete(c); err != nil {
				logrus.WithError(err).Fatalf("Error running delete bdev raid command")
			}
		},
	}
}

func bdevRaidDelete(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	deleted, err := spdkCli.BdevRaidDelete(c.Args().First())
	if err != nil {
		return err
	}

	bdevRaidDeleteRespJSON, err := json.Marshal(deleted)
	if err != nil {
		return err
	}
	fmt.Println(string(bdevRaidDeleteRespJSON))

	return nil
}

func BdevRaidGetCmd() cli.Command {
	return cli.Command{
		Name: "get",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "name",
				Usage: "Optional. If you want to get one specific raid bdev info, please input this or uuid.",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "Optional. If you want to get one specific raid bdev info, please input this or name",
			},
			cli.Uint64Flag{
				Name:  "timeout, t",
				Usage: "Optional. Determine the timeout of the execution",
				Value: 0,
			},
		},
		Usage: "get all raid bdevs if the info is not specified: \"get\", or \"get --name <RAID NAME>\", or \"get --uuid <UUID>\"",
		Action: func(c *cli.Context) {
			if err := bdevRaidGet(c); err != nil {
				logrus.WithError(err).Fatalf("Error running get bdev raid command")
			}
		},
	}
}

func bdevRaidGet(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	name := c.String("name")
	if name == "" {
		name = c.String("uuid")
	}

	bdevRaidGetResp, err := spdkCli.BdevRaidGet(name, c.Uint64("timeout"))
	if err != nil {
		return err
	}

	bdevRaidGetRespJSON, err := json.MarshalIndent(bdevRaidGetResp, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(bdevRaidGetRespJSON))

	return nil
}
