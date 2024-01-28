package nvmecli

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	commonTypes "github.com/longhorn/go-common-libs/types"

	"github.com/longhorn/go-spdk-helper/pkg/nvme"
	"github.com/longhorn/go-spdk-helper/pkg/types"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func Cmd() cli.Command {
	return cli.Command{
		Name: "nvmecli",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "host-proc",
				Usage: fmt.Sprintf("The host proc path of namespace executor. By default %v", commonTypes.ProcDirectory),
				Value: commonTypes.ProcDirectory,
			},
		},
		Subcommands: []cli.Command{
			DiscoverCmd(),
			ConnectCmd(),
			DisconnectCmd(),
			GetCmd(),
			StartCmd(),
			StopCmd(),
		},
	}
}

func DiscoverCmd() cli.Command {
	return cli.Command{
		Name: "discover",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "traddr",
				Usage:    "NVMe-oF target address: a ip or BDF",
				Required: true,
				Value:    types.LocalIP,
			},
			cli.StringFlag{
				Name:     "trsvcid",
				Usage:    "NVMe-oF target trsvcid: a port number",
				Required: true,
			},
		},
		Usage: "Discover a NVMe-oF target: discover --traddr <IP> --trsvcid <PORT NUMBER>",
		Action: func(c *cli.Context) {
			if err := discover(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run nvme-cli discover command")
			}
		},
	}
}

func discover(c *cli.Context) error {
	executor, err := util.NewExecutor(c.GlobalString("host-proc"))
	if err != nil {
		return err
	}

	subnqn, err := nvme.DiscoverTarget(c.String("traddr"), c.String("trsvcid"), executor)
	if err != nil {
		return err
	}

	return util.PrintObject(map[string]string{"subnqn": subnqn})
}

func ConnectCmd() cli.Command {
	return cli.Command{
		Name: "connect",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "traddr",
				Usage:    "NVMe-oF target address: a ip or BDF",
				Required: true,
				Value:    types.LocalIP,
			},
			cli.StringFlag{
				Name:     "trsvcid",
				Usage:    "NVMe-oF target trsvcid: a port number",
				Required: true,
			},
			cli.StringFlag{
				Name:     "nqn",
				Usage:    "NVMe-oF target subsystem nqn",
				Required: true,
			},
		},
		Usage: "Connect a NVMe-oF target subsystem as a NVMe device/initiator: connect --traddr <IP> --trsvcid <PORT NUMBER> --nqn <SUBSYSTEM NQN> ",
		Action: func(c *cli.Context) {
			if err := connect(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run nvme-cli connect command")
			}
		},
	}
}

func connect(c *cli.Context) error {
	executor, err := util.NewExecutor(c.String("host-proc"))
	if err != nil {
		return err
	}

	controllerName, err := nvme.ConnectTarget(c.String("traddr"), c.String("trsvcid"), c.String("nqn"), executor)
	if err != nil {
		return err
	}

	return util.PrintObject(map[string]string{"controllerName": controllerName})
}

func DisconnectCmd() cli.Command {
	return cli.Command{
		Name:  "disconnect",
		Usage: "Disconnect a NVMe-oF target subsystem to stop a NVMe device/initiator: disconnect <SUBSYSTEM NQN>",
		Action: func(c *cli.Context) {
			if err := disconnect(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run nvme-cli disconnect command")
			}
		},
	}
}

func disconnect(c *cli.Context) error {
	executor, err := util.NewExecutor(c.String("host-proc"))
	if err != nil {
		return err
	}

	return nvme.DisconnectTarget(c.Args().First(), executor)
}

func GetCmd() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "Get all connected NVMe-oF devices/initiators if a subsystem nqn or address is not specified: \"get\" or \"get --traddr <IP> --trsvcid <PORT NUMBER> --nqn <SUBSYSTEM NQN>\"",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "traddr",
				Usage: "NVMe-oF target address: a ip or BDF",
				Value: types.LocalIP,
			},
			cli.StringFlag{
				Name:  "trsvcid",
				Usage: "NVMe-oF target trsvcid: a port number",
			},
			cli.StringFlag{
				Name:  "nqn",
				Usage: "NVMe-oF target subsystem nqn",
			},
		},
		Action: func(c *cli.Context) {
			if err := get(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run nvme-cli get command")
			}
		},
	}
}

func get(c *cli.Context) error {
	executor, err := util.NewExecutor(c.String("host-proc"))
	if err != nil {
		return err
	}

	getResp, err := nvme.GetDevices(c.String("traddr"), c.String("trsvcid"), c.String("nqn"), executor)
	if err != nil {
		return err
	}

	return util.PrintObject(getResp)
}

func StartCmd() cli.Command {
	return cli.Command{
		Name: "start",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "name",
				Usage:    "The name of initiator. The initiator will make the device to `/dev/longhorn/<name>`",
				Required: true,
			},
			cli.StringFlag{
				Name:     "traddr",
				Usage:    "NVMe-oF target address: a ip or BDF",
				Required: true,
				Value:    types.LocalIP,
			},
			cli.StringFlag{
				Name:     "trsvcid",
				Usage:    "NVMe-oF target trsvcid: a port number",
				Required: true,
			},
			cli.StringFlag{
				Name:     "nqn",
				Usage:    "NVMe-oF target subsystem nqn",
				Required: true,
			},
		},
		Usage: "Start a NVMe-oF initiator and make a device based on the name: start --name <NAME> --traddr <IP> --trsvcid <PORT NUMBER> --nqn <SUBSYSTEM NQN>",
		Action: func(c *cli.Context) {
			if err := start(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run initiator start command")
			}
		},
	}
}

func start(c *cli.Context) error {
	initiator, err := nvme.NewInitiator(c.String("name"), c.String("nqn"), c.String("host-proc"))
	if err != nil {
		return err
	}

	if _, err := initiator.Start(c.String("traddr"), c.String("trsvcid"), true); err != nil {
		return err
	}

	return util.PrintObject(map[string]string{
		"controller_name": initiator.GetControllerName(),
		"namespace_name":  initiator.GetNamespaceName(),
		"endpoint":        initiator.GetEndpoint(),
	})
}

func StopCmd() cli.Command {
	return cli.Command{
		Name: "stop",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "name",
				Usage:    "The name of initiator. The initiator will make the device to `/dev/longhorn/<name>`",
				Required: true,
			},
			cli.StringFlag{
				Name:     "nqn",
				Usage:    "NVMe-oF target subsystem nqn",
				Required: true,
			},
		},
		Usage: "Stop a NVMe-oF initiator and remove the corresponding device: stop --name <NAME> --nqn <SUBSYSTEM NQN>",
		Action: func(c *cli.Context) {
			if err := stop(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run initiator stop command")
			}
		},
	}
}

func stop(c *cli.Context) error {
	initiator, err := nvme.NewInitiator(c.String("name"), c.String("nqn"), c.String("host-proc"))
	if err != nil {
		return err
	}

	if _, err := initiator.Stop(true, false, true); err != nil {
		return err
	}

	return nil
}
