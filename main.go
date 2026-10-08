package main

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	"github.com/longhorn/go-spdk-helper/app/cmd/advanced"
	"github.com/longhorn/go-spdk-helper/app/cmd/basic"
	"github.com/longhorn/go-spdk-helper/app/cmd/dmsetup"
	"github.com/longhorn/go-spdk-helper/app/cmd/nvmecli"
	"github.com/longhorn/go-spdk-helper/app/cmd/spdksetup"
	"github.com/longhorn/go-spdk-helper/app/cmd/spdktgt"
)

func main() {
	a := &cli.Command{
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			if c.Bool("debug") {
				logrus.SetLevel(logrus.DebugLevel)
			}
			return nil, nil
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: "debug",
			},
		},
		Commands: []*cli.Command{
			basic.BdevCmd(),
			basic.BdevAioCmd(),
			basic.BdevVirtioCmd(),
			basic.BdevLvstoreCmd(),
			basic.BdevLvolCmd(),
			basic.BdevNvmeCmd(),
			basic.BdevRaidCmd(),
			basic.BdevEcCmd(),
			basic.NvmfCmd(),
			basic.LogCmd(),
			basic.UblkCmd(),
			basic.SpdkKillInstanceCmd(),

			advanced.DeviceCmd(),
			advanced.ExposeCmd(),

			nvmecli.Cmd(),

			dmsetup.Cmd(),

			spdktgt.Cmd(),
			spdksetup.Cmd(),
		},
	}
	if err := a.Run(context.Background(), os.Args); err != nil {
		logrus.WithError(err).Fatal("Failed to execute command")
	}
}
