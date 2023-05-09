package nvmecli

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/nvme"
	"github.com/longhorn/go-spdk-helper/pkg/types"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func Cmd() cli.Command {
	return cli.Command{
		Name: "nvmecli",
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
				Name:  "traddr",
				Usage: "NVMe-oF target address: a ip or BDF",
				Value: types.LocalIP,
			},
			cli.StringFlag{
				Name:  "trsvcid",
				Usage: "NVMe-oF target trsvcid: a port number",
			},
		},
		Usage: "Discover a NVMe-oF target: discover --traddr <IP> --trsvcid <PORT NUMBER>",
		Action: func(c *cli.Context) {
			if err := discover(c); err != nil {
				logrus.WithError(err).Fatalf("Error running nvme-cli discover command")
			}
		},
	}
}

func discover(c *cli.Context) error {
	subnqn, err := nvme.DiscoverTarget(c.String("traddr"), c.String("trsvcid"), util.NewTimeoutExecutor(util.CmdTimeout))
	if err != nil {
		return err
	}

	discoverRespJSON, err := json.MarshalIndent(map[string]string{"subnqn": subnqn}, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(discoverRespJSON))

	return nil
}

func ConnectCmd() cli.Command {
	return cli.Command{
		Name: "connect",
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
				Usage: "NVMe-oF target subsystem nqn.",
			},
		},
		Usage: "Connect a NVMe-oF target subsystem as a NVMe device/initiator: connect --traddr <IP> --trsvcid <PORT NUMBER> --nqn <SUBSYSTEM NQN> ",
		Action: func(c *cli.Context) {
			if err := connect(c); err != nil {
				logrus.WithError(err).Fatalf("Error running nvme-cli connect command")
			}
		},
	}
}

func connect(c *cli.Context) error {
	controllerName, err := nvme.ConnectTarget(c.String("traddr"), c.String("trsvcid"), c.String("nqn"), util.NewTimeoutExecutor(util.CmdTimeout))
	if err != nil {
		return err
	}

	connectRespJSON, err := json.MarshalIndent(map[string]string{"controllerName": controllerName}, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(connectRespJSON))

	return nil
}

func DisconnectCmd() cli.Command {
	return cli.Command{
		Name:  "disconnect",
		Usage: "Disconnect a NVMe-oF target subsystem to stop a NVMe device/initiator: disconnect <SUBSYSTEM NQN>",
		Action: func(c *cli.Context) {
			if err := disconnect(c); err != nil {
				logrus.WithError(err).Fatalf("Error running nvme-cli disconnect command")
			}
		},
	}
}

func disconnect(c *cli.Context) error {
	return nvme.DisconnectTarget(c.Args().First(), util.NewTimeoutExecutor(util.CmdTimeout))
}

func GetCmd() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "Get all connected NVMe-oF devices/initiators if a subsystem nqn or address is not specified: \"get\" or \"get --traddr <IP> --trsvcid <PORT NUMBER> --nqn <SUBSYSTEM NQN>\"",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "traddr",
				Usage: "Optional. NVMe-oF target address: a ip or BDF",
				Value: types.LocalIP,
			},
			cli.StringFlag{
				Name:  "trsvcid",
				Usage: "Optional. NVMe-oF target trsvcid: a port number",
			},
			cli.StringFlag{
				Name:  "nqn",
				Usage: "Optional. NVMe-oF target subsystem nqn.",
			},
		},
		Action: func(c *cli.Context) {
			if err := get(c); err != nil {
				logrus.WithError(err).Fatalf("Error running nvme-cli get command")
			}
		},
	}
}

func get(c *cli.Context) error {
	getResp, err := nvme.GetDevices(c.String("traddr"), c.String("trsvcid"), c.String("nqn"), util.NewTimeoutExecutor(util.CmdTimeout))
	if err != nil {
		return err
	}

	getRespJSON, err := json.MarshalIndent(getResp, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(getRespJSON))

	return nil
}

func StartCmd() cli.Command {
	return cli.Command{
		Name: "start",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "name",
				Usage: "The name of initiator. The initiator will make the device to `/dev/longhorn/<name>`.",
			},
			cli.StringFlag{
				Name:  "traddr",
				Usage: "NVMe-oF target address: a ip or BDF.",
				Value: types.LocalIP,
			},
			cli.StringFlag{
				Name:  "trsvcid",
				Usage: "NVMe-oF target trsvcid: a port number.",
			},
			cli.StringFlag{
				Name:  "nqn",
				Usage: "NVMe-oF target subsystem nqn.",
			},
			cli.StringFlag{
				Name:  "host-proc",
				Usage: "The host proc path of namespace executor. Empty means not using a namespace executor. By default empty.",
				Value: "",
			},
		},
		Usage: "Start a NVMe-oF initiator and make a device based on the name: start --name <NAME> --traddr <IP> --trsvcid <PORT NUMBER> --nqn <SUBSYSTEM NQN>",
		Action: func(c *cli.Context) {
			if err := start(c); err != nil {
				logrus.WithError(err).Fatalf("Error running initiator start command")
			}
		},
	}
}

func start(c *cli.Context) error {
	initiator, err := nvme.NewInitiator(c.String("name"), c.String("nqn"), c.String("host-proc"))
	if err != nil {
		return err
	}

	if err := initiator.Start(c.String("traddr"), c.String("trsvcid")); err != nil {
		return err
	}

	startRespJSON, err := json.MarshalIndent(
		map[string]string{
			"controller_name": initiator.GetControllerName(),
			"namespace_name":  initiator.GetNamespaceName(),
			"endpoint":        initiator.GetEndpoint(),
		}, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(startRespJSON))

	return nil
}

func StopCmd() cli.Command {
	return cli.Command{
		Name: "stop",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "name",
				Usage: "The name of initiator. The initiator will make the device to `/dev/longhorn/<name>`.",
			},
			cli.StringFlag{
				Name:  "nqn",
				Usage: "NVMe-oF target subsystem nqn.",
			},
			cli.StringFlag{
				Name:  "host-proc",
				Usage: "The host proc path of namespace executor. Empty means not using a namespace executor. By default empty.",
				Value: "",
			},
		},
		Usage: "Stop a NVMe-oF initiator and remove the corresponding device: stop --name <NAME> --nqn <SUBSYSTEM NQN>",
		Action: func(c *cli.Context) {
			if err := stop(c); err != nil {
				logrus.WithError(err).Fatalf("Error running initiator stop command")
			}
		},
	}
}

func stop(c *cli.Context) error {
	initiator, err := nvme.NewInitiator(c.String("name"), c.String("nqn"), c.String("host-proc"))
	if err != nil {
		return err
	}

	if err := initiator.Stop(); err != nil {
		return err
	}

	return nil
}
