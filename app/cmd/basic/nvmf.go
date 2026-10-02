package basic

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func NvmfCmd() *cli.Command {
	return &cli.Command{
		Name:    "nvme-of",
		Aliases: []string{"nvmf"},
		Commands: []*cli.Command{
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
			NvmfSubsystemGetListenersCmd(),
		},
	}
}

func NvmfCreateTransportCmd() *cli.Command {
	return &cli.Command{
		Name:  "transport-create",
		Usage: "create a transport for nvmf: transport-create",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "trtype",
				Usage: "NVMe-oF target trtype: \"tcp\", \"rdma\" or \"pcie\"",
				Value: string(spdktypes.NvmeTransportTypeTCP),
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := nvmfCreateTransport(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run create nvmf transport command")
				return err
			}
			return nil
		},
	}
}

func nvmfCreateTransport(c *cli.Command) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	created, err := spdkCli.NvmfCreateTransport(spdktypes.NvmeTransportType(c.String("trtype")))
	if err != nil {
		return err
	}

	return util.PrintObject(created)
}

func NvmfGetTransportsCmd() *cli.Command {
	return &cli.Command{
		Name:  "transport-get",
		Usage: "get all transports if trtype or tgt-name is not specified: transport-get",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "trtype",
				Usage: "NVMe-oF target trtype: \"tcp\", \"rdma\" or \"pcie\"",
			},
			&cli.StringFlag{
				Name:  "tgt-name",
				Usage: "Parent NVMe-oF target name",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := nvmfGetTransports(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get nvmf transports command")
				return err
			}
			return nil
		},
	}
}

func nvmfGetTransports(c *cli.Command) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	transportList, err := spdkCli.NvmfGetTransports(spdktypes.NvmeTransportType(c.String("trtype")), c.String("tgt-name"))
	if err != nil {
		return err
	}

	return util.PrintObject(transportList)
}

func NvmfCreateSubsystemCmd() *cli.Command {
	return &cli.Command{
		Name:  "subsystem-create",
		Usage: "create a subsystem for nvmf: subsystem-create <SUBSYSTEM NQN>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := nvmfCreateSubsystem(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run create nvmf subsystem command")
				return err
			}
			return nil
		},
	}
}

func nvmfCreateSubsystem(c *cli.Command) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	created, err := spdkCli.NvmfCreateSubsystem(c.Args().First(), true)
	if err != nil {
		return err
	}

	return util.PrintObject(created)
}

func NvmfDeleteSubsystemCmd() *cli.Command {
	return &cli.Command{
		Name:  "subsystem-delete",
		Usage: "delete a subsystem for nvmf: subsystem-delete <SUBSYSTEM NQN>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := nvmfDeleteSubsystem(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run delete nvmf subsystem command")
				return err
			}
			return nil
		},
	}
}

func nvmfDeleteSubsystem(c *cli.Command) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	deleted, err := spdkCli.NvmfDeleteSubsystem(c.Args().First(), "")
	if err != nil {
		return err
	}

	return util.PrintObject(deleted)
}

func NvmfGetSubsystemsCmd() *cli.Command {
	return &cli.Command{
		Name:  "subsystem-get",
		Usage: "list all subsystem for the specified NVMe-oF target: subsystem-get <TGT NAME>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "nqn",
				Usage: "NVMe-oF target subsystem NQN",
			},
			&cli.StringFlag{
				Name:  "tgt-name",
				Usage: "Parent NVMe-oF target name",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := nvmfGetSubsystems(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get nvmf subsystems command")
				return err
			}
			return nil
		},
	}
}

func nvmfGetSubsystems(c *cli.Command) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	subsystemList, err := spdkCli.NvmfGetSubsystems(c.String("nqn"), c.String("tgt-name"))
	if err != nil {
		return err
	}

	return util.PrintObject(subsystemList)
}

func NvmfSubsystemAddNsCmd() *cli.Command {
	return &cli.Command{
		Name: "ns-add",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "nqn",
				Usage:    "Subsystem NQN",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "bdev-name",
				Usage:    "Name of bdev to expose as a namespace",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "nguid",
				Usage:    "Namespace globally unique identifier",
				Required: false,
			},
		},
		Usage: "add a bdev as a namespace for subsystem of nvmf: ns-add <SUBSYSTEM NQN>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := nvmfSubsystemAddNs(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run add nvmf subsystem namespace command")
				return err
			}
			return nil
		},
	}
}

func nvmfSubsystemAddNs(c *cli.Command) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	added, err := spdkCli.NvmfSubsystemAddNs(c.String("nqn"), c.String("bdev-name"), c.String("nguid"))
	if err != nil {
		return err
	}

	return util.PrintObject(added)
}

func NvmfSubsystemRemoveNsCmd() *cli.Command {
	return &cli.Command{
		Name: "ns-remove",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "nqn",
				Usage:    "Subsystem NQN",
				Required: true,
			},
			&cli.UintFlag{
				Name:     "nsid",
				Usage:    "Removing namespace ID",
				Required: true,
			},
		},
		Usage: "remove a namespace from a subsystem of nvmf: ns-remove --nqn <SUBSYSTEM NQN> --nsid <NAMESPACE ID>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := nvmfSubsystemRemoveNs(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run remove nvmf subsystem namespace command")
				return err
			}
			return nil
		},
	}
}

func nvmfSubsystemRemoveNs(c *cli.Command) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	deleted, err := spdkCli.NvmfSubsystemRemoveNs(c.String("nqn"), uint32(c.Uint("nsid")))
	if err != nil {
		return err
	}

	return util.PrintObject(deleted)
}

func NvmfSubsystemGetNssCmd() *cli.Command {
	return &cli.Command{
		Name: "ns-get",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "nqn",
				Usage:    "Subsystem NQN",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "bdev-name",
				Usage: "Name of bdev to expose as a namespace. It's better not to specify this and \"nsid\" simultaneously",
			},
			&cli.UintFlag{
				Name:  "nsid",
				Usage: "The specified namespace ID. It's better not to specify this and \"bdev-name\" simultaneously",
			},
		},
		Usage: "list all namespaces for a subsystem of nvmf: ns-get <SUBSYSTEM NQN>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := nvmfSubsystemGetNss(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get nvmf subsystem namespaces command")
				return err
			}
			return nil
		},
	}
}

func nvmfSubsystemGetNss(c *cli.Command) error {
	nqn := c.String("nqn")
	if nqn == "" {
		return fmt.Errorf("subsystem NQN is required")
	}

	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	nsList, err := spdkCli.NvmfSubsystemsGetNss(nqn, c.String("bdev-name"), uint32(c.Uint("nsid")))
	if err != nil {
		return err
	}

	return util.PrintObject(nsList)
}

func NvmfSubsystemAddListenerCmd() *cli.Command {
	return &cli.Command{
		Name: "listener-add",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "nqn",
				Usage:    "NVMe-oF target subnqn. It can be the nvmf subsystem nqn",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "traddr",
				Usage:    "NVMe-oF target address: a ip or BDF",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "trsvcid",
				Usage:    "NVMe-oF target trsvcid: a port number",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "trtype",
				Usage: "NVMe-oF target trtype: \"tcp\", \"rdma\" or \"pcie\"",
				Value: string(spdktypes.NvmeTransportTypeTCP),
			},
			&cli.StringFlag{
				Name:  "adrfam",
				Usage: "NVMe-oF target adrfam: \"ipv4\", \"ipv6\", \"ib\", \"fc\", \"intra_host\"",
				Value: string(spdktypes.NvmeAddressFamilyIPv4),
			},
		},
		Usage: "add a listener for subsystem of nvmf: listener-add --nqn <SUBSYSTEM NQN> --traddr <IP> --trsvcid <PORT NUMBER>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := nvmfSubsystemAddListener(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run add nvmf subsystem listener command")
				return err
			}
			return nil
		},
	}
}

func nvmfSubsystemAddListener(c *cli.Command) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	added, err := spdkCli.NvmfSubsystemAddListener(c.String("nqn"), c.String("traddr"), c.String("trsvcid"),
		spdktypes.NvmeTransportType(c.String("trtype")), spdktypes.NvmeAddressFamily(c.String("adrfam")))
	if err != nil {
		return err
	}

	return util.PrintObject(added)
}

func NvmfSubsystemRemoveListenerCmd() *cli.Command {
	return &cli.Command{
		Name: "listener-remove",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "nqn",
				Usage:    "NVMe-oF target subnqn. It can be the nvmf subsystem nqn",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "traddr",
				Usage:    "NVMe-oF target address: a ip or BDF",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "trsvcid",
				Usage:    "NVMe-oF target trsvcid: a port number",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "trtype",
				Usage: "NVMe-oF target trtype: \"tcp\", \"rdma\" or \"pcie\"",
				Value: string(spdktypes.NvmeTransportTypeTCP),
			},
			&cli.StringFlag{
				Name:  "adrfam",
				Usage: "NVMe-oF target adrfam: \"ipv4\", \"ipv6\", \"ib\", \"fc\", \"intra_host\"",
				Value: string(spdktypes.NvmeAddressFamilyIPv4),
			},
		},
		Usage: "remove a listener from a subsystem of nvmf: listener-remove --nqn <SUBSYSTEM NQN> --traddr <IP> --trsvcid <PORT NUMBER>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := nvmfSubsystemRemoveListener(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run remove nvmf subsystem listener command")
				return err
			}
			return nil
		},
	}
}

func nvmfSubsystemRemoveListener(c *cli.Command) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	deleted, err := spdkCli.NvmfSubsystemRemoveListener(c.String("nqn"), c.String("traddr"), c.String("trsvcid"),
		spdktypes.NvmeTransportType(c.String("trtype")), spdktypes.NvmeAddressFamily(c.String("adrfam")))
	if err != nil {
		return err
	}

	return util.PrintObject(deleted)
}

func NvmfSubsystemGetListenersCmd() *cli.Command {
	return &cli.Command{
		Name:  "listener-get",
		Usage: "list all listeners for a subsystem of nvmf: listener-get <SUBSYSTEM NQN>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := nvmfSubsystemGetListeners(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get nvmf subsystem listeners command")
				return err
			}
			return nil
		},
	}
}

func nvmfSubsystemGetListeners(c *cli.Command) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	listenerList, err := spdkCli.NvmfSubsystemGetListeners(c.Args().First(), "")
	if err != nil {
		return err
	}

	return util.PrintObject(listenerList)
}
