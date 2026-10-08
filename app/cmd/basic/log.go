package basic

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func LogCmd() *cli.Command {
	return &cli.Command{
		Name: "log",
		Commands: []*cli.Command{
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

func LogSetFlagCmd() *cli.Command {
	return &cli.Command{
		Name:  "set-flag",
		Usage: "set log flag: set-flag <FLAG>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := logSetFlag(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run set log flag command")
				return err
			}
			return nil
		},
	}
}

func logSetFlag(c *cli.Command) error {
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

func LogClearFlagCmd() *cli.Command {
	return &cli.Command{
		Name:  "clear-flag",
		Usage: "clear log flag: clear-flag <FLAG>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := logClearFlag(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run clear log flag command")
				return err
			}
			return nil
		},
	}
}

func logClearFlag(c *cli.Command) error {
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

func LogGetFlagsCmd() *cli.Command {
	return &cli.Command{
		Name:  "get-flags",
		Usage: "get log flags",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := logGetFlags(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get log flags command")
				return err
			}
			return nil
		},
	}
}

func logGetFlags(c *cli.Command) error {
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

func LogSetLevelCmd() *cli.Command {
	return &cli.Command{
		Name:  "set-level",
		Usage: "set log level: set-level <LEVEL>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := logSetLevel(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run set log level command")
				return err
			}
			return nil
		},
	}
}

func logSetLevel(c *cli.Command) error {
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

func LogGetLevelCmd() *cli.Command {
	return &cli.Command{
		Name:  "get-level",
		Usage: "get log level",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := logGetLevel(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get log level command")
				return err
			}
			return nil
		},
	}
}

func logGetLevel(c *cli.Command) error {
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

func LogSetPrintLevelCmd() *cli.Command {
	return &cli.Command{
		Name:  "set-print-level",
		Usage: "set log print level: set-print-level <LEVEL>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := logSetPrintLevel(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run set log print level command")
				return err
			}
			return nil
		},
	}
}

func logSetPrintLevel(c *cli.Command) error {
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

func LogGetPrintLevelCmd() *cli.Command {
	return &cli.Command{
		Name:  "get-print-level",
		Usage: "get log print level",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := logGetPrintLevel(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get log print level command")
				return err
			}
			return nil
		},
	}
}

func logGetPrintLevel(c *cli.Command) error {
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
