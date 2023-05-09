package basic

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
)

func NvmfCmd() cli.Command {
	return cli.Command{
		Name:      "nvme-of",
		ShortName: "nvmf",
		Subcommands: []cli.Command{
			NvmfCreateTransportCmd(),
			NvmfGetTransportsCmd(),
			NvmfCreateSubsystemCmd(),
			NvmfDeleteSubsystemCmd(),
			NvmfGetSubsystemsCmd(),
			NvmfSubsystemAddNsCmd(),
			NvmfSubsystemRemoveNsCmd(),
			NvmfSubsystemGetNssCmd(),
			NvmfSubsystemAddListenerCmd(),
			NvmfSubsystemRemoveListenerCmd(),
			NvmfSubsystemRemoveListenerCmd(),
			NvmfSubsystemGetListenersCmd(),
		},
	}
}

func NvmfCreateTransportCmd() cli.Command {
	return cli.Command{
		Name:  "transport-create",
		Usage: "create a transport for nvmf: transport-create",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "trtype",
				Usage: "Optional. NVMe-oF target trtype: \"tcp\", \"rdma\" or \"pcie\"",
				Value: string(spdktypes.NvmeTransportTypeTCP),
			},
		},
		Action: func(c *cli.Context) {
			if err := nvmfCreateTransport(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run create nvmf transport command")
			}
		},
	}
}

func nvmfCreateTransport(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	created, err := spdkCli.NvmfCreateTransport(spdktypes.NvmeTransportType(c.String("trtype")))
	if err != nil {
		return err
	}

	bdevNvmfCreateTransportRespJSON, err := json.MarshalIndent(created, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(bdevNvmfCreateTransportRespJSON))

	return nil
}

func NvmfGetTransportsCmd() cli.Command {
	return cli.Command{
		Name:  "transport-get",
		Usage: "get all transports if trtype or tgt-name is not specified: transport-get",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "trtype",
				Usage: "Optional. NVMe-oF target trtype: \"tcp\", \"rdma\" or \"pcie\"",
			},
			cli.StringFlag{
				Name:  "tgt-name",
				Usage: "Optional. Parent NVMe-oF target name.",
			},
		},
		Action: func(c *cli.Context) {
			if err := nvmfGetTransports(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get nvmf transports command")
			}
		},
	}
}

func nvmfGetTransports(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	transportList, err := spdkCli.NvmfGetTransports(spdktypes.NvmeTransportType(c.String("trtype")), c.String("tgt-name"))
	if err != nil {
		return err
	}

	bdevNvmfGetTransportRespJSON, err := json.MarshalIndent(transportList, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(bdevNvmfGetTransportRespJSON))

	return nil
}

func NvmfCreateSubsystemCmd() cli.Command {
	return cli.Command{
		Name:  "subsystem-create",
		Usage: "create a subsystem for nvmf: subsystem-create <SUBSYSTEM NQN>",
		Action: func(c *cli.Context) {
			if err := nvmfCreateSubsystem(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run create nvmf subsystem command")
			}
		},
	}
}

func nvmfCreateSubsystem(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	created, err := spdkCli.NvmfCreateSubsystem(c.Args().First())
	if err != nil {
		return err
	}

	bdevNvmfCreateSubsystemRespJSON, err := json.MarshalIndent(created, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(bdevNvmfCreateSubsystemRespJSON))

	return nil
}

func NvmfDeleteSubsystemCmd() cli.Command {
	return cli.Command{
		Name:  "subsystem-delete",
		Usage: "delete a subsystem for nvmf: subsystem-delete <SUBSYSTEM NQN>",
		Action: func(c *cli.Context) {
			if err := nvmfDeleteSubsystem(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run delete nvmf subsystem command")
			}
		},
	}
}

func nvmfDeleteSubsystem(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	deleted, err := spdkCli.NvmfDeleteSubsystem(c.Args().First(), "")
	if err != nil {
		return err
	}

	bdevNvmfDeleteSubsystemRespJSON, err := json.MarshalIndent(deleted, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(bdevNvmfDeleteSubsystemRespJSON))

	return nil
}

func NvmfGetSubsystemsCmd() cli.Command {
	return cli.Command{
		Name:  "subsystem-get",
		Usage: "list all subsystem for the specified NVMe-oF target: subsystem-get <TGT NAME>",
		Action: func(c *cli.Context) {
			if err := nvmfGetSubsystems(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get nvmf subsystems command")
			}
		},
	}
}

func nvmfGetSubsystems(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	subsystemList, err := spdkCli.NvmfGetSubsystems(c.String("tgt-name"))
	if err != nil {
		return err
	}

	bdevNvmfGetSubsystemRespJSON, err := json.MarshalIndent(subsystemList, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(bdevNvmfGetSubsystemRespJSON))

	return nil
}

func NvmfSubsystemAddNsCmd() cli.Command {
	return cli.Command{
		Name: "ns-add",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "nqn",
				Usage: "Required. Subsystem NQN.",
			},
			cli.StringFlag{
				Name:  "bdev-name",
				Usage: "Required. Name of bdev to expose as a namespace.",
			},
		},
		Usage: "add a bdev as a namespace for subsystem of nvmf: ns-add <SUBSYSTEM NQN>",
		Action: func(c *cli.Context) {
			if err := nvmfSubsystemAddNs(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run add nvmf subsystem namespace command")
			}
		},
	}
}

func nvmfSubsystemAddNs(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	added, err := spdkCli.NvmfSubsystemAddNs(c.String("nqn"), c.String("bdev-name"))
	if err != nil {
		return err
	}

	bdevNvmfSubsystemAddNsRespJSON, err := json.MarshalIndent(added, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(bdevNvmfSubsystemAddNsRespJSON))

	return nil
}

func NvmfSubsystemRemoveNsCmd() cli.Command {
	return cli.Command{
		Name: "ns-remove",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "nqn",
				Usage: "Required. Subsystem NQN.",
			},
			cli.UintFlag{
				Name:  "nsid",
				Usage: "Required. Removing namespace ID.",
			},
		},
		Usage: "remove a namespace from a subsystem of nvmf: ns-remove --nqn <SUBSYSTEM NQN> --nsid <NAMESPACE ID>",
		Action: func(c *cli.Context) {
			if err := nvmfSubsystemRemoveNs(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run remove nvmf subsystem namespace command")
			}
		},
	}
}

func nvmfSubsystemRemoveNs(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	deleted, err := spdkCli.NvmfSubsystemRemoveNs(c.String("nqn"), uint32(c.Uint("nsid")))
	if err != nil {
		return err
	}

	bdevNvmfSubsystemRemoveNsRespJSON, err := json.MarshalIndent(deleted, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(bdevNvmfSubsystemRemoveNsRespJSON))

	return nil
}

func NvmfSubsystemGetNssCmd() cli.Command {
	return cli.Command{
		Name: "ns-get",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "nqn",
				Usage: "Required. Subsystem NQN.",
			},
			cli.StringFlag{
				Name:  "bdev-name",
				Usage: "Optional. Name of bdev to expose as a namespace. It's better not to specify this and \"nsid\" simultaneously.",
			},
			cli.UintFlag{
				Name:  "nsid",
				Usage: "Optional. The specified namespace ID. It's better not to specify this and \"bdev-name\" simultaneously.",
			},
		},
		Usage: "list all namespaces for a subsystem of nvmf: ns-get <SUBSYSTEM NQN>",
		Action: func(c *cli.Context) {
			if err := nvmfSubsystemGetNss(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get nvmf subsystem namespaces command")
			}
		},
	}
}

func nvmfSubsystemGetNss(c *cli.Context) error {
	nqn := c.String("nqn")
	if nqn == "" {
		return fmt.Errorf("subsystem NQN is required")
	}

	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	nsList, err := spdkCli.NvmfSubsystemsGetNss(nqn, c.String("bdev-name"), uint32(c.Uint("nsid")))
	if err != nil {
		return err
	}

	bdevNvmfSubsystemGetNssRespJSON, err := json.MarshalIndent(nsList, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(bdevNvmfSubsystemGetNssRespJSON))

	return nil
}

func NvmfSubsystemAddListenerCmd() cli.Command {
	return cli.Command{
		Name: "listener-add",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "nqn",
				Usage: "NVMe-oF target subnqn. It can be the nvmf subsystem nqn.",
			},
			cli.StringFlag{
				Name:  "traddr",
				Usage: "NVMe-oF target address: a ip or BDF",
			},
			cli.StringFlag{
				Name:  "trsvcid",
				Usage: "NVMe-oF target trsvcid: a port number",
			},
			cli.StringFlag{
				Name:  "trtype",
				Usage: "NVMe-oF target trtype: \"tcp\", \"rdma\" or \"pcie\"",
				Value: string(spdktypes.NvmeTransportTypeTCP),
			},
			cli.StringFlag{
				Name:  "adrfam",
				Usage: "NVMe-oF target adrfam: \"ipv4\", \"ipv6\", \"ib\", \"fc\", \"intra_host\"",
				Value: string(spdktypes.NvmeAddressFamilyIPv4),
			},
		},
		Usage: "add a listener for subsystem of nvmf: listener-add --nqn <SUBSYSTEM NQN> --traddr <IP> --trsvcid <PORT NUMBER>",
		Action: func(c *cli.Context) {
			if err := nvmfSubsystemAddListener(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run add nvmf subsystem listener command")
			}
		},
	}
}

func nvmfSubsystemAddListener(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	added, err := spdkCli.NvmfSubsystemAddListener(c.String("nqn"), c.String("traddr"), c.String("trsvcid"),
		spdktypes.NvmeTransportType(c.String("trtype")), spdktypes.NvmeAddressFamily(c.String("adrfam")))
	if err != nil {
		return err
	}

	bdevNvmfSubsystemAddListenerRespJSON, err := json.MarshalIndent(added, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(bdevNvmfSubsystemAddListenerRespJSON))

	return nil
}

func NvmfSubsystemRemoveListenerCmd() cli.Command {
	return cli.Command{
		Name: "listener-remove",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "nqn",
				Usage: "NVMe-oF target subnqn. It can be the nvmf subsystem nqn.",
			},
			cli.StringFlag{
				Name:  "traddr",
				Usage: "NVMe-oF target address: a ip or BDF",
			},
			cli.StringFlag{
				Name:  "trsvcid",
				Usage: "NVMe-oF target trsvcid: a port number",
			},
			cli.StringFlag{
				Name:  "trtype",
				Usage: "NVMe-oF target trtype: \"tcp\", \"rdma\" or \"pcie\"",
				Value: string(spdktypes.NvmeTransportTypeTCP),
			},
			cli.StringFlag{
				Name:  "adrfam",
				Usage: "NVMe-oF target adrfam: \"ipv4\", \"ipv6\", \"ib\", \"fc\", \"intra_host\"",
				Value: string(spdktypes.NvmeAddressFamilyIPv4),
			},
		},
		Usage: "remove a listener from a subsystem of nvmf: listener-remove --nqn <SUBSYSTEM NQN> --traddr <IP> --trsvcid <PORT NUMBER>",
		Action: func(c *cli.Context) {
			if err := nvmfSubsystemRemoveListener(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run remove nvmf subsystem listener command")
			}
		},
	}
}

func nvmfSubsystemRemoveListener(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	deleted, err := spdkCli.NvmfSubsystemRemoveListener(c.String("nqn"), c.String("traddr"), c.String("trsvcid"),
		spdktypes.NvmeTransportType(c.String("trtype")), spdktypes.NvmeAddressFamily(c.String("adrfam")))
	if err != nil {
		return err
	}

	bdevNvmfSubsystemRemoveListenerRespJSON, err := json.MarshalIndent(deleted, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(bdevNvmfSubsystemRemoveListenerRespJSON))

	return nil
}

func NvmfSubsystemGetListenersCmd() cli.Command {
	return cli.Command{
		Name:  "listener-get",
		Usage: "list all listeners for a subsystem of nvmf: listener-get <SUBSYSTEM NQN>",
		Action: func(c *cli.Context) {
			if err := nvmfSubsystemGetListeners(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get nvmf subsystem listeners command")
			}
		},
	}
}

func nvmfSubsystemGetListeners(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	listenerList, err := spdkCli.NvmfSubsystemGetListeners(c.Args().First(), "")
	if err != nil {
		return err
	}

	bdevNvmfSubsystemGetListenersRespJSON, err := json.MarshalIndent(listenerList, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(bdevNvmfSubsystemGetListenersRespJSON))

	return nil
}
