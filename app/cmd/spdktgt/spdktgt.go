package spdktgt

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	commontypes "github.com/longhorn/go-common-libs/types"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/target"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func Cmd() *cli.Command {
	return &cli.Command{
		Name:    "spdk-tgt",
		Aliases: []string{"tgt"},
		Usage:   "Start SPDK target: tgt --spdk-dir <SPDK DIRECTORY>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "spdk-dir",
				Usage:    "The SPDK directory that contains the setup scripts and binary \"spdk_tgt\"",
				Required: true,
				Value:    os.Getenv("SPDK_DIR"),
			},
			&cli.StringSliceFlag{
				Name:  "opts",
				Usage: "The spdk_tgt command line flags",
			},
			&cli.Int64Flag{
				Name:  "timeout",
				Usage: "The timeout in second for the command",
				Value: 3600,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := spdkTGT(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run spdk-tgt start command")
				return err
			}
			return nil
		},
		DisableSliceFlagSeparator: true,
	}
}

func spdkTGT(c *cli.Command) error {
	ne, err := util.NewExecutor(commontypes.ProcDirectory)
	if err != nil {
		return err
	}
	return target.StartTarget(c.String("spdk-dir"), c.StringSlice("opts"), time.Duration(c.Int64("timeout"))*time.Second, ne.Execute)
}
