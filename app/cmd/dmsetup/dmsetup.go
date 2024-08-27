package dmsetup

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	commontypes "github.com/longhorn/go-common-libs/types"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func Cmd() cli.Command {
	return cli.Command{
		Name: "dmsetup",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "host-proc",
				Usage: fmt.Sprintf("The host proc path of namespace executor. By default %v", commontypes.ProcDirectory),
				Value: commontypes.ProcDirectory,
			},
		},
		Subcommands: []cli.Command{
			CreateCmd(),
			RemoveCmd(),
			SuspendCmd(),
			ResumeCmd(),
			ReloadCmd(),
			DepsCmd(),
		},
	}
}

func CreateCmd() cli.Command {
	return cli.Command{
		Name: "create",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "table",
				Usage:    "One-line table directly on the command line.",
				Required: true,
			},
		},
		Usage: "Create a device mapper device with the given name and table: create --table <device name>",
		Action: func(c *cli.Context) {
			if err := create(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to create device %v with table %v", c.Args().First(), c.String("table"))
			}
		},
	}
}

func create(c *cli.Context) error {
	executor, err := util.NewExecutor(c.GlobalString("host-proc"))
	if err != nil {
		return err
	}

	deviceName := c.Args().First()
	if deviceName == "" {
		return fmt.Errorf("device name is required")
	}

	logrus.Infof("Creating device %v with table %v", deviceName, c.String("table"))

	return util.DmsetupCreate(deviceName, c.String("table"), executor)
}

func SuspendCmd() cli.Command {
	return cli.Command{
		Name: "suspend",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:     "noflush",
				Usage:    "Do not flush outstanding I/O when suspending a device. Default: false.",
				Required: false,
			},
			cli.BoolFlag{
				Name:     "nolockfs",
				Usage:    "Do not attempt to synchronize filesystem. Default: false.",
				Required: false,
			},
		},
		Usage: "Suspend the device mapper device with the given name: suspend --noflush --nolockfs <device name>",
		Action: func(c *cli.Context) {
			if err := suspend(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to suspend device %v", c.Args().First())
			}
		},
	}
}

func suspend(c *cli.Context) error {
	executor, err := util.NewExecutor(c.GlobalString("host-proc"))
	if err != nil {
		return err
	}

	deviceName := c.Args().First()
	if deviceName == "" {
		return fmt.Errorf("device name is required")
	}

	logrus.Infof("Suspending device %v with noflush %v and nolockfs %v", deviceName, c.Bool("noflush"), c.Bool("nolockfs"))

	return util.DmsetupSuspend(deviceName, c.Bool("noflush"), c.Bool("nolockfs"), executor)
}

func ResumeCmd() cli.Command {
	return cli.Command{
		Name:  "resume",
		Flags: []cli.Flag{},
		Usage: "Resume the device mapper device with the given name: resume <device name>",
		Action: func(c *cli.Context) {
			if err := resume(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to resume device %v", c.Args().First())
			}
		},
	}
}

func resume(c *cli.Context) error {
	executor, err := util.NewExecutor(c.GlobalString("host-proc"))
	if err != nil {
		return err
	}

	deviceName := c.Args().First()
	if deviceName == "" {
		return fmt.Errorf("device name is required")
	}

	logrus.Infof("Resuming device %v", deviceName)

	return util.DmsetupResume(deviceName, executor)
}

func ReloadCmd() cli.Command {
	return cli.Command{
		Name: "reload",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "table",
				Usage:    "One-line table directly on the command line.",
				Required: true,
			},
		},
		Usage: "Reload the table of the device mapper device with the given name and table: reload --table <device name>",
		Action: func(c *cli.Context) {
			if err := reload(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to reload device %v with table %v", c.Args().First(), c.String("table"))
			}
		},
	}
}

func reload(c *cli.Context) error {
	executor, err := util.NewExecutor(c.GlobalString("host-proc"))
	if err != nil {
		return err
	}

	deviceName := c.Args().First()
	if deviceName == "" {
		return fmt.Errorf("device name is required")
	}

	logrus.Infof("Reloading device %v with table %v", deviceName, c.String("table"))

	return util.DmsetupReload(deviceName, c.String("table"), executor)
}

func RemoveCmd() cli.Command {
	return cli.Command{
		Name: "remove",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:     "force",
				Usage:    "Try harder to complete operation. Default: false.",
				Required: false,
			},
			cli.BoolFlag{
				Name:     "deferred",
				Usage:    "Enable deferred removal of open devices. The device will be removed when the last user closes it. Default: false.",
				Required: false,
			},
		},
		Usage: "Remove the device mapper device with the given name: remove --force --deferred <device name>",
		Action: func(c *cli.Context) {
			if err := remove(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to create device %v with table %v", c.Args().First(), c.String("table"))
			}
		},
	}
}

func remove(c *cli.Context) error {
	executor, err := util.NewExecutor(c.GlobalString("host-proc"))
	if err != nil {
		return err
	}

	deviceName := c.Args().First()
	if deviceName == "" {
		return fmt.Errorf("device name is required")
	}

	logrus.Infof("Removing device %v with force %v and deferred %v", deviceName, c.Bool("force"), c.Bool("deferred"))

	return util.DmsetupRemove(deviceName, c.Bool("force"), c.Bool("deferred"), executor)
}

func DepsCmd() cli.Command {
	return cli.Command{
		Name:  "deps",
		Flags: []cli.Flag{},
		Usage: "Outputting a list of devices referenced by the live table for the specified device: deps <device name>",
		Action: func(c *cli.Context) {
			if err := deps(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to output a list of devices referenced by the live table for the specified device %v", c.Args().First())
			}
		},
	}
}

func deps(c *cli.Context) error {
	executor, err := util.NewExecutor(c.GlobalString("host-proc"))
	if err != nil {
		return err
	}

	deviceName := c.Args().First()
	if deviceName == "" {
		return fmt.Errorf("device name is required")
	}

	logrus.Infof("Outputting a list of devices referenced by the live table for the specified device %v", deviceName)

	output, err := util.DmsetupDeps(deviceName, executor)
	if err != nil {
		return err
	}
	fmt.Printf("Dependent devices: %v", output)
	return nil
}
