package basic

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
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
				Name:     "subnqn",
				Usage:    "NVMe-oF target subnqn. It can be the nvmf subsystem nqn",
				Required: true,
			},
			cli.StringFlag{
				Name:     "traddr",
				Usage:    "NVMe-oF target address: a ip or BDF",
				Required: true,
			},
			cli.StringFlag{
				Name:     "trsvcid",
				Usage:    "NVMe-oF target trsvcid: a port number",
				Required: true,
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
		Action: func(c *cli.Context) {
			if err := bdevNvmeAttachController(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run attach nvme controller command")
			}
		},
	}
}

func bdevNvmeAttachController(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	bdevNameList, err := spdkCli.BdevNvmeAttachController(c.String("name"), c.String("subnqn"),
		c.String("traddr"), c.String("trsvcid"),
		spdktypes.NvmeTransportType(c.String("trtype")), spdktypes.NvmeAddressFamily(c.String("adrfam")))
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
	spdkCli, err := client.NewClient()
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
	spdkCli, err := client.NewClient()
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
			cli.StringFlag{
				Name:  "name",
				Usage: "The name of a nvme bdev is typically \"<Nvme Controller NAME>n1\". If you want to get one specific Nvme info, please input this or uuid",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "If you want to get one specific Nvme info, please input this or name",
			},
			cli.Uint64Flag{
				Name:  "timeout, t",
				Usage: "Determine the timeout of the execution",
				Value: 0,
			},
		},
		Usage: "get all Nvme bdevs if the name is not specified: \"get\", or \"get --name <NVME CONTROLLER NAME>n1\", or \"get --uuid <UUID>\"",
		Action: func(c *cli.Context) {
			if err := bdevNvmeGet(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get nvme controller command")
			}
		},
	}
}

func bdevNvmeGet(c *cli.Context) error {
	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	name := c.String("name")
	if name == "" {
		name = c.String("uuid")
	}

	bdevNvmeGetResp, err := spdkCli.BdevNvmeGet(name, c.Uint64("timeout"))
	if err != nil {
		return err
	}

	return util.PrintObject(bdevNvmeGetResp)
}
