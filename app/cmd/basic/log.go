package basic

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func LogCmd() cli.Command {
	return cli.Command{
		Name: "log",
		Subcommands: []cli.Command{
			LogSetFlagCmd(),
			LogClearFlagCmd(),
			LogGetFlagsCmd(),
			LogSetLevelCmd(),
			LogGetLevelCmd(),
			LogSetPrintLevelCmd(),
			LogGetPrintLevelCmd(),
		},
	}
}

func LogSetFlagCmd() cli.Command {
	return cli.Command{
		Name:  "set-flag",
		Usage: "set log flag: set-flag <FLAG>",
		Action: func(c *cli.Context) {
			if err := logSetFlag(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run set log flag command")
			}
		},
	}
}

func logSetFlag(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	result, err := spdkCli.LogSetFlag(c.Args().First())
	if err != nil {
		return err
	}

	return util.PrintObject(result)
}

func LogClearFlagCmd() cli.Command {
	return cli.Command{
		Name:  "clear-flag",
		Usage: "clear log flag: clear-flag <FLAG>",
		Action: func(c *cli.Context) {
			if err := logClearFlag(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run clear log flag command")
			}
		},
	}
}

func logClearFlag(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	result, err := spdkCli.LogClearFlag(c.Args().First())
	if err != nil {
		return err
	}

	return util.PrintObject(result)
}

func LogGetFlagsCmd() cli.Command {
	return cli.Command{
		Name:  "get-flags",
		Usage: "get log flags",
		Action: func(c *cli.Context) {
			if err := logGetFlags(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get log flags command")
			}
		},
	}
}

func logGetFlags(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	logFlags, err := spdkCli.LogGetFlags()
	if err != nil {
		return err
	}

	return util.PrintObject(logFlags)
}

func LogSetLevelCmd() cli.Command {
	return cli.Command{
		Name:  "set-level",
		Usage: "set log level: set-level <LEVEL>",
		Action: func(c *cli.Context) {
			if err := logSetLevel(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run set log level command")
			}
		},
	}
}

func logSetLevel(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	result, err := spdkCli.LogSetLevel(c.Args().First())
	if err != nil {
		return err
	}

	return util.PrintObject(result)
}

func LogGetLevelCmd() cli.Command {
	return cli.Command{
		Name:  "get-level",
		Usage: "get log level",
		Action: func(c *cli.Context) {
			if err := logGetLevel(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get log level command")
			}
		},
	}
}

func logGetLevel(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	logLevel, err := spdkCli.LogGetLevel()
	if err != nil {
		return err
	}

	return util.PrintObject(logLevel)
}

func LogSetPrintLevelCmd() cli.Command {
	return cli.Command{
		Name:  "set-print-level",
		Usage: "set log print level: set-print-level <LEVEL>",
		Action: func(c *cli.Context) {
			if err := logSetPrintLevel(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run set log print level command")
			}
		},
	}
}

func logSetPrintLevel(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	result, err := spdkCli.LogSetPrintLevel(c.Args().First())
	if err != nil {
		return err
	}

	return util.PrintObject(result)
}

func LogGetPrintLevelCmd() cli.Command {
	return cli.Command{
		Name:  "get-print-level",
		Usage: "get log print level",
		Action: func(c *cli.Context) {
			if err := logGetPrintLevel(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get log print level command")
			}
		},
	}
}

func logGetPrintLevel(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	logPrintLevel, err := spdkCli.LogGetPrintLevel()
	if err != nil {
		return err
	}

	return util.PrintObject(logPrintLevel)
}
