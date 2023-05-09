package basic

import (
	"encoding/json"
	"fmt"
	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func BdevCmd() cli.Command {
	return cli.Command{
		Name: "bdev",
		Subcommands: []cli.Command{
			BdevGetCmd(),
		},
	}
}

func BdevGetCmd() cli.Command {
	return cli.Command{
		Name: "get",
		Flags: []cli.Flag{
			cli.Uint64Flag{
				Name:  "timeout, t",
				Usage: "Optional. Determine the timeout of the execution",
				Value: 0,
			},
		},
		Usage: "get all bdevs if a bdev name is not specified: get <BDEV NAME>",
		Action: func(c *cli.Context) {
			if err := bdevGet(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get bdev command")
			}
		},
	}
}

func bdevGet(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	bdevGetResp, err := spdkCli.BdevGetBdevs(c.Args().First(), 0)
	if err != nil {
		return err
	}

	bdevGetRespJSON, err := json.MarshalIndent(bdevGetResp, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(bdevGetRespJSON))

	return nil
}
