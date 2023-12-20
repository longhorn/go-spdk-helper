package spdktgt

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	nslib "github.com/longhorn/go-common-libs/ns"
	typeslib "github.com/longhorn/go-common-libs/types"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/target"
)

func Cmd() cli.Command {
	return cli.Command{
		Name:      "spdk-tgt",
		ShortName: "tgt",
		Usage:     "Start SPDK target: tgt --spdk-dir <SPDK DIRECTORY>",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "spdk-dir",
				Usage:    "The SPDK directory that contains the setup scripts and binary \"spdk_tgt\"",
				Required: true,
				Value:    os.Getenv("SPDK_DIR"),
			},
			cli.StringSliceFlag{
				Name:  "opts",
				Usage: "The spdk_tgt command line flags",
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
	namespaces := []typeslib.Namespace{typeslib.NamespaceMnt, typeslib.NamespaceIpc, typeslib.NamespaceNet}
	ne, err := nslib.NewNamespaceExecutor(typeslib.ProcessNone, typeslib.ProcDirectory, namespaces)
	if err != nil {
		return err
	}
	return target.StartTarget(c.String("spdk-dir"), c.StringSlice("opts"), ne.Execute)
}
