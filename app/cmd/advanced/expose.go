package advanced

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
)

func ExposeCmd() cli.Command {
	return cli.Command{
		Name: "expose",
		Subcommands: []cli.Command{
			StarExposeCmd(),
			StopExposeCmd(),
		},
	}
}

func StarExposeCmd() cli.Command {
	return cli.Command{
		Name:  "start",
		Usage: "Expose a bdev via nvmf: start --nqn <NVMF SUBSYSTEM NQN> --bdev-name <BDEV ALIAS or BDEV UUID> --ip <IP ADDRESS> --port <PORT NUMBER>",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "nqn",
				Usage:    "NVMe-oF target subsystem NQN",
				Required: true,
			},
			cli.StringFlag{
				Name:     "bdev-name",
				Usage:    "Name of the exported bdev lvol",
				Required: true,
			},
			cli.StringFlag{
				Name:     "ip",
				Usage:    "This can be host IP or localhost IP",
				Required: true,
			},
			cli.StringFlag{
				Name:     "port",
				Usage:    "Port number",
				Required: true,
			},
		},
		Action: func(c *cli.Context) {
			if err := startExpose(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run start expose command")
			}
		},
	}
}

func startExpose(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	if err := spdkCli.StartExposeBdev(c.String("nqn"), c.String("bdev-name"), c.String("ip"), c.String("port")); err != nil {
		return err
	}

	fmt.Println("true")

	return nil
}

func StopExposeCmd() cli.Command {
	return cli.Command{
		Name:  "stop",
		Usage: "Stop exposing a bdev via nvmf: stop --nqn <NVMF SUBSYSTEM NQN>",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "nqn",
				Usage:    "NVMe-oF target subsystem NQN",
				Required: true,
			},
		},
		Action: func(c *cli.Context) {
			if err := stopExpose(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run stop expose command")
			}
		},
	}
}

func stopExpose(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	if err := spdkCli.StopExposeBdev(c.String("nqn")); err != nil {
		return err
	}

	fmt.Println("true")

	return nil
}
