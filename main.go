package main

import (
	"os"

	"github.com/longhorn/go-spdk-helper/app/cmd/spdktgt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/app/cmd/advanced"
	"github.com/longhorn/go-spdk-helper/app/cmd/basic"
	"github.com/longhorn/go-spdk-helper/app/cmd/nvmecli"
)

func main() {
	a := cli.NewApp()

	a.Before = func(c *cli.Context) error {
		if c.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	a.Flags = []cli.Flag{
		cli.BoolFlag{
			Name: "debug",
		},
	}
	a.Commands = []cli.Command{
		basic.BdevCmd(),
		basic.BdevAioCmd(),
		basic.BdevLvstoreCmd(),
		basic.BdevLvolCmd(),
		basic.BdevNvmeCmd(),
		basic.BdevRaidCmd(),
		basic.NvmfCmd(),

		advanced.DeviceCmd(),
		advanced.ExposeCmd(),

		nvmecli.Cmd(),

		spdktgt.Cmd(),
	}
	if err := a.Run(os.Args); err != nil {
		logrus.WithError(err).Fatal("Failed to execute command")
	}
}
