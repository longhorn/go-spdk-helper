package basic

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
	"github.com/longhorn/go-spdk-helper/pkg/types"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func BdevNvmeCmd() cli.Command {
	return cli.Command{
		Name:      "bdev-nvme",
		ShortName: "nvme",
		Subcommands: []cli.Command{
			BdevNvmeAttachControllerCmd(),
			BdevNvmeDetachControllerCmd(),
			BdevNvmeGetControllersCmd(),
			BdevNvmeGetCmd(),
			BdevNvmeSetOptionsCmd(),
		},
	}
}

func BdevNvmeAttachControllerCmd() cli.Command {
	return cli.Command{
		Name:  "controller-attach",
		Usage: "attach a nvme controller to the current host: attach-controller --name <CONTROLLER NAME> --subnqn <SUBSYSTEM NQN> --traddr <IP ADDRESS> --trsvcid <PORT NUMBER>",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "name",
				Usage:    "Name of the NVMe controller, prefix for each bdev name",
				Required: true,
			},
			cli.StringFlag{
				Name:  "subnqn",
				Usage: "NVMe-oF target subnqn. It can be the nvmf subsystem nqn",
			},
			cli.StringFlag{
				Name:     "traddr",
				Usage:    "NVMe-oF target address: a ip or BDF",
				Required: true,
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
			cli.IntFlag{
				Name:  "ctrlr-loss-timeout-sec",
				Usage: "NVMe-oF controller loss timeout in seconds for error cases",
				Value: types.DefaultCtrlrLossTimeoutSec,
			},
			cli.IntFlag{
				Name:  "reconnect-delay-sec",
				Usage: "NVMe-oF controller reconnect delay in seconds for error cases",
				Value: types.DefaultReconnectDelaySec,
			},
			cli.IntFlag{
				Name:  "fast-io-fail-timeout-sec",
				Usage: "NVMe-oF controller fast I/O fail timeout in seconds for error cases",
				Value: types.DefaultFastIOFailTimeoutSec,
			},
			cli.IntFlag{
				Name:  "keep_alive_timeout_ms",
				Usage: "NVMe-oF keep alive timeout in milliseconds",
				Value: types.DefaultKeepAliveTimeoutMs,
			},
			cli.StringFlag{
				Name:  "multipath",
				Usage: "Multipathing behavior: disable, failover, multipath. Default is failover",
				Value: string(spdktypes.NvmeMultipathBehaviorFailover),
			},
		},
		Action: func(c *cli.Context) {
			if err := bdevNvmeAttachController(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run attach nvme controller command")
			}
		},
	}
}

func bdevNvmeAttachController(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	bdevNameList, err := spdkCli.BdevNvmeAttachController(c.String("name"), c.String("subnqn"),
		c.String("traddr"), c.String("trsvcid"),
		spdktypes.NvmeTransportType(c.String("trtype")), spdktypes.NvmeAddressFamily(c.String("adrfam")),
		int32(c.Int("ctrlr-loss-timeout-sec")), int32(c.Int("reconnect-delay-sec")), int32(c.Int("fast-io-fail-timeout-sec")),
		c.String("multipath"))
	if err != nil {
		return err
	}

	return util.PrintObject(bdevNameList)
}

func BdevNvmeDetachControllerCmd() cli.Command {
	return cli.Command{
		Name:  "controller-detach",
		Usage: "detach a nvme controller from the current host: detach-controller <CONTROLLER NAME>",
		Action: func(c *cli.Context) {
			if err := bdevNvmeDetachController(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run detach nvme controller command")
			}
		},
	}
}

func bdevNvmeDetachController(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	detached, err := spdkCli.BdevNvmeDetachController(c.Args().First())
	if err != nil {
		return err
	}

	return util.PrintObject(detached)
}

func BdevNvmeGetControllersCmd() cli.Command {
	return cli.Command{
		Name:  "controller-get",
		Usage: "get all nvme controllers if the name is not specified: get <CONTROLLER NAME>",
		Action: func(c *cli.Context) {
			if err := bdevNvmeGetControllers(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get nvme controller command")
			}
		},
	}
}

func bdevNvmeGetControllers(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	bdevNvmeGetControllersResp, err := spdkCli.BdevNvmeGetControllers(c.Args().First())
	if err != nil {
		return err
	}

	return util.PrintObject(bdevNvmeGetControllersResp)
}

func BdevNvmeGetCmd() cli.Command {
	return cli.Command{
		Name: "get",
		Flags: []cli.Flag{
			cli.Uint64Flag{
				Name:  "timeout, t",
				Usage: "Determine the timeout of the execution",
				Value: 0,
			},
		},
		Usage: "get all Nvme bdevs if the name is not specified: \"get\", or \"get <NVME NAMESPACE NAME>\", or \"get <UUID>\"",
		Action: func(c *cli.Context) {
			if err := bdevNvmeGet(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get nvme controller command")
			}
		},
	}
}

func bdevNvmeGet(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	bdevNvmeGetResp, err := spdkCli.BdevNvmeGet(c.Args().First(), c.Uint64("timeout"))
	if err != nil {
		return err
	}

	return util.PrintObject(bdevNvmeGetResp)
}

func BdevNvmeSetOptionsCmd() cli.Command {
	return cli.Command{
		Name:  "option-set",
		Usage: "set options for NVMe-oF controllers: option-set [options]",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "ctrlr-loss-timeout-sec",
				Usage: "NVMe-oF controller loss timeout in seconds for error cases",
				Value: types.DefaultCtrlrLossTimeoutSec,
			},
			cli.IntFlag{
				Name:  "reconnect-delay-sec",
				Usage: "NVMe-oF controller reconnect delay in seconds for error cases",
				Value: types.DefaultReconnectDelaySec,
			},
			cli.IntFlag{
				Name:  "fast-io-fail-timeout-sec",
				Usage: "NVMe-oF controller fast I/O fail timeout in seconds for error cases",
				Value: types.DefaultFastIOFailTimeoutSec,
			},
			cli.IntFlag{
				Name:  "transport-ack-timeout",
				Usage: "Time to wait ack until retransmission for RDMA or connection close for TCP. Range 0-31 where 0 means use default.",
				Value: types.DefaultTransportAckTimeout,
			},
		},
		Action: func(c *cli.Context) {
			if err := bdevNvmeSetOptions(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run set nvme options command")
			}
		},
	}
}

func bdevNvmeSetOptions(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	result, err := spdkCli.BdevNvmeSetOptions(int32(c.Int("ctrlr-loss-timeout-sec")),
		int32(c.Int("reconnect-delay-sec")), int32(c.Int("fast-io-fail-timeout-sec")),
		int32(c.Int("transport-ack-timeout")), int32(c.Int("keep_alive_timeout_ms")))
	if err != nil {
		return err
	}

	return util.PrintObject(result)
}
