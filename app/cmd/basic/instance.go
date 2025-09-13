package basic

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func SpdkKillInstanceCmd() cli.Command {
	return cli.Command{
		Name: "kill-instance",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "sig-name",
				Usage:    "The signal to send to the SPDK instance (e.g., SIGINT, SIGTERM, SIGKILL)",
				Required: true,
			},
		},
		Usage: "Send a signal to the SPDK instance: kill-instance --sig-name <SIGNAL>",
		Action: func(c *cli.Context) {
			if err := spdkKillInstanceCmd(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run spdk kill instance command")
			}
		},
	}
}

func spdkKillInstanceCmd(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	result, err := spdkCli.SpdkKillInstance(c.String("sig-name"))
	if err != nil {
		return err
	}

	return util.PrintObject(result)
}
