package advanced

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	"github.com/longhorn/go-spdk-helper/pkg/types"
)

func DeviceCmd() cli.Command {
	return cli.Command{
		Name: "device",
		Subcommands: []cli.Command{
			DeviceAddCmd(),
			DeviceDeleteCmd(),
		},
	}
}

func DeviceAddCmd() cli.Command {
	return cli.Command{
		Name:  "add",
		Usage: "Add a device for SPDK. The file device file name would be the aio name as well as the lvs name: add <device path>",
		Flags: []cli.Flag{
			cli.UintFlag{
				Name:  "cluster-size",
				Usage: "Optional. Logical volume store cluster size, by default 1MiB.",
				Value: types.MiB,
			},
		},
		Action: func(c *cli.Context) {
			if err := deviceAdd(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run add device command")
			}
		},
	}
}

func deviceAdd(c *cli.Context) error {
	devicePath := c.Args().First()

	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	bdevAioName, lvsName, lvsUUID, err := spdkCli.AddDevice(devicePath, "", uint32(c.Uint("cluster-size")))
	if err != nil {
		return err
	}

	deviceAddRespJSON, err := json.Marshal(
		map[string]string{
			"bdev_aio_name": bdevAioName,
			"lvs_name":      lvsName,
			"lvs_uuid":      lvsUUID,
		})
	if err != nil {
		return err
	}
	fmt.Println(string(deviceAddRespJSON))

	return nil
}

func DeviceDeleteCmd() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "Delete a device for SPDK. The aio name and the lvs name should be the file device file name: delete <device path>",
		Action: func(c *cli.Context) {
			if err := deviceDelete(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run delete device command")
			}
		},
	}
}

func deviceDelete(c *cli.Context) error {
	devicePath := c.Args().First()
	fileName := filepath.Base(devicePath)

	spdkCli, err := client.NewClient()
	if err != nil {
		return err
	}

	if err := spdkCli.DeleteDevice(fileName, fileName); err != nil {
		return err
	}

	fmt.Println("true")

	return nil
}
