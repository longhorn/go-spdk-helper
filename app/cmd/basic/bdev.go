package basic

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func BdevCmd() *cli.Command {
	return &cli.Command{
		Name: "bdev",
		Commands: []*cli.Command{
			BdevGetCmd(),
		},
	}
}

func BdevGetCmd() *cli.Command {
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
		Usage: "get all bdevs if a bdev name is not specified: get <BDEV NAME>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := bdevGet(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get bdev command")
				return err
			}
			return nil
		},
	}
}

func bdevGet(c *cli.Command) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	bdevGetResp, err := spdkCli.BdevGetBdevs(c.Args().First(), 0)
	if err != nil {
		return err
	}

	return util.PrintObject(bdevGetResp)
}
