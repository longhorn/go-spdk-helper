package basic

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
)

func BdevNvmeCmd() cli.Command {
	return cli.Command{
		Name:      "bdev-nvme",
		ShortName: "nvme",
		Subcommands: []cli.Command{
			BdevNvmeAttachControllerCmd(),
			BdevNvmeDetachControllerCmd(),
			BdevNvmeGetControllersCmd(),
		},
	}
}

func BdevNvmeAttachControllerCmd() cli.Command {
	return cli.Command{
		Name:  "controller-attach",
		Usage: "attach a nvme controller to the current host: attach-controller --name <CONTROLLER NAME> --subnqn <SUBSYSTEM NQN> --traddr <IP ADDRESS> --trsvcid <PORT NUMBER>",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "name",
				Usage: "Name of the NVMe controller, prefix for each bdev name.",
			},
			cli.StringFlag{
				Name:  "subnqn",
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
		Action: func(c *cli.Context) {
			if err := bdevNvmeAttachController(c); err != nil {
				logrus.WithError(err).Fatalf("Error running attach nvme controller command")
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

	bdevNvmeAttachControllerRespJSON, err := json.Marshal(bdevNameList)
	if err != nil {
		return err
	}
	fmt.Println(string(bdevNvmeAttachControllerRespJSON))

	return nil
}

func BdevNvmeDetachControllerCmd() cli.Command {
	return cli.Command{
		Name:  "controller-detach",
		Usage: "detach a nvme controller from the current host: detach-controller <CONTROLLER NAME>",
		Action: func(c *cli.Context) {
			if err := bdevNvmeDetachController(c); err != nil {
				logrus.WithError(err).Fatalf("Error running detach nvme controller command")
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

	bdevNvmeDetachControllerRespJSON, err := json.Marshal(detached)
	if err != nil {
		return err
	}
	fmt.Println(string(bdevNvmeDetachControllerRespJSON))

	return nil
}

func BdevNvmeGetControllersCmd() cli.Command {
	return cli.Command{
		Name:  "controller-get",
		Usage: "get all nvme controllers if the name is not specified: get <CONTROLLER NAME>",
		Action: func(c *cli.Context) {
			if err := bdevNvmeGetControllers(c); err != nil {
				logrus.WithError(err).Fatalf("Error running get nvme controller command")
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

	bdevNvmeGetControllersRespJSON, err := json.MarshalIndent(bdevNvmeGetControllersResp, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(bdevNvmeGetControllersRespJSON))

	return nil
}
