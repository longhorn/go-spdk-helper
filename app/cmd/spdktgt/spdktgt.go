package spdktgt

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/target"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func Cmd() cli.Command {
	return cli.Command{
		Name:      "spdk-tgt",
		ShortName: "tgt",
		Usage:     "Start SPDK target: tgt --spdk-dir <SPDK DIRECTORY>",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "spdk-dir",
				Usage: "The SPDK directory that contains the setup scripts and binary \"spdk_tgt\"",
				Value: os.Getenv("SPDK_DIR"),
			},
		},
		Action: func(c *cli.Context) {
			if err := spdkTGT(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run spdk-tgt start command")
			}
		},
	}
}

func spdkTGT(c *cli.Context) error {
	return target.StartTarget(c.String("spdk-dir"), util.Execute)
}
