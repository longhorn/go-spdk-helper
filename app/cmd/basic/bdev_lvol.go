package basic

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
	"github.com/longhorn/go-spdk-helper/pkg/util"
)

func BdevLvolCmd() cli.Command {
	return cli.Command{
		Name:      "bdev-lvol",
		ShortName: "lvol",
		Subcommands: []cli.Command{
			BdevLvolCreateCmd(),
			BdevLvolDeleteCmd(),
			BdevLvolGetCmd(),
			BdevLvolSnapshotCmd(),
			BdevLvolCloneCmd(),
			BdevLvolCloneBdevCmd(),
			BdevLvolSetParentCmd(),
			BdevLvolDecoupleParentCmd(),
			BdevLvolResizeCmd(),
			BdevLvolStartShallowCopyCmd(),
			BdevLvolCheckShallowCopyCmd(),
			BdevLvolGetXattrCmd(),
			BdevLvolGetFragmapCmd(),
			BdevLvolRenameCmd(),
			BdevLvolRegisterSnapshotChecksumCmd(),
			BdevLvolGetSnapshotChecksumCmd(),
		},
	}
}

func BdevLvolCreateCmd() cli.Command {
	return cli.Command{
		Name: "create",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "lvs-name",
				Usage: "Specify this or lvs-uuid",
			},
			cli.StringFlag{
				Name:  "lvs-uuid",
				Usage: "Specify this or lvs-name",
			},
			cli.StringFlag{
				Name:     "lvol-name",
				Required: true,
			},
			cli.Uint64Flag{
				Name:     "size",
				Usage:    "Specify bdev lvol size in MiB",
				Required: true,
			},
		},
		Usage: "create a bdev lvol on a lvstore: \"create --lvs-name <LVSTORE NAME> --lvol-name <LVOL NAME> --size <LVOL SIZE in MIB>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolCreate(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run create bdev lvol command")
			}
		},
	}
}

func bdevLvolCreate(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	lvsName, lvsUUID := c.String("lvs-name"), c.String("lvs-uuid")
	lvolName := c.String("lvol-name")
	size := c.Uint64("size")

	uuid, err := spdkCli.BdevLvolCreate(lvsName, lvsUUID, lvolName, size,
		spdktypes.BdevLvolClearMethodUnmap, true)
	if err != nil {
		return err
	}

	return util.PrintObject(map[string]string{"uuid": uuid, "alias": fmt.Sprintf("%s/%s", lvsName, lvolName)})
}

func BdevLvolDeleteCmd() cli.Command {
	return cli.Command{
		Name: "delete",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "alias",
				Usage: "The alias of a lvol is <LVSTORE NAME>/<LVOL NAME>. Specify this or uuid",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "Specify this or alias",
			},
		},
		Usage: "delete a bdev lvol using a block device: \"delete --alias <LVSTORE NAME>/<LVOL NAME>\" or \"delete --uuid <UUID>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolDelete(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run delete bdev lvol command")
			}
		},
	}
}

func bdevLvolDelete(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	name := c.String("alias")
	if name == "" {
		name = c.String("uuid")
	}

	deleted, err := spdkCli.BdevLvolDelete(name)
	if err != nil {
		return err
	}

	return util.PrintObject(deleted)
}

func BdevLvolGetCmd() cli.Command {
	return cli.Command{
		Name: "get",
		Flags: []cli.Flag{
			cli.Uint64Flag{
				Name:  "timeout, t",
				Usage: "Determine the timeout of the execution",
				Value: 0,
			},
		},
		Usage: "get all bdev lvol if the info is not specified: \"get\", or \"get <LVSTORE NAME>/<LVOL NAME>\", or \"get <UUID>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolGet(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get bdev lvol command")
			}
		},
	}
}

func bdevLvolGet(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	bdevLvolGetResp, err := spdkCli.BdevLvolGet(c.Args().First(), c.Uint64("timeout"))
	if err != nil {
		return err
	}

	return util.PrintObject(bdevLvolGetResp)
}

func BdevLvolSnapshotCmd() cli.Command {
	return cli.Command{
		Name: "snapshot",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "alias",
				Usage: "The alias of a lvol is <LVSTORE NAME>/<LVOL NAME>. Specify this or uuid",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "Specify this or alias",
			},
			cli.StringSliceFlag{
				Name:  "xattr",
				Usage: "Xattr for the snapshot in the format name=value. Optional",
			},
			cli.StringFlag{
				Name:     "snapshot-name",
				Usage:    "The snapshot lvol name",
				Required: true,
			},
		},
		Usage: "create a snapshot as a new bdev lvol based on an existing one: \"snapshot --alias <LVSTORE NAME>/<LVOL NAME> --snapshot-name <SNAPSHOT NAME>\", or \"snapshot --uuid <UUID> --snapshot-name <SNAPSHOT NAME>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolSnapshot(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run snapshot bdev lvol command")
			}
		},
	}
}

func bdevLvolSnapshot(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	name := c.String("alias")
	if name == "" {
		name = c.String("uuid")
	}

	var xattrs []client.Xattr
	xattrs_args := c.StringSlice("xattr")
	for _, s := range xattrs_args {
		parts := strings.Split(s, "=")
		if len(parts) != 2 {
			return errors.Errorf("xattr %q not in name=value format", s)
		}

		xattr := client.Xattr{
			Name:  parts[0],
			Value: parts[1],
		}
		xattrs = append(xattrs, xattr)
	}

	uuid, err := spdkCli.BdevLvolSnapshot(name, c.String("snapshot-name"), xattrs)
	if err != nil {
		return err
	}

	return util.PrintObject(uuid)
}

func BdevLvolCloneCmd() cli.Command {
	return cli.Command{
		Name: "clone",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "snapshot",
				Usage:    "UUID or alias of the snapshot lvol to clone. Alias is <LVSTORE NAME>/<LVOL NAME>",
				Required: true,
			},
			cli.StringFlag{
				Name:     "clone-name",
				Usage:    "Name for the logical volume to create",
				Required: true,
			},
		},
		Usage: "create a lvol based on an existing snapshot lvol: \"clone --snapshot <LVSTORE NAME>/<SNAPSHOT LVOL NAME> --clone-name <CLONE NAME>\", or \"clone --snapshot <SNAPSHOT LVOL UUID> --clone-name <CLONE NAME>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolClone(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run clone bdev lvol command")
			}
		},
	}
}

func bdevLvolClone(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	uuid, err := spdkCli.BdevLvolClone(c.String("snapshot"), c.String("clone-name"))
	if err != nil {
		return err
	}

	return util.PrintObject(uuid)
}

func BdevLvolCloneBdevCmd() cli.Command {
	return cli.Command{
		Name: "clone-bdev",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "bdev",
				Usage: "Name or UUID for bdev that acts as the external snapshot",
			},
			cli.StringFlag{
				Name: "lvs-name",
			},
			cli.StringFlag{
				Name:     "clone-name",
				Usage:    "Name for the logical volume to create",
				Required: true,
			},
		},
		Usage: "create a lvol based on an external snapshot bdev: \"clone-bdev --bdev <BDEV NAME or UUID> --lvs-name <LVSTORE NAME> --clone-name <CLONE NAME>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolCloneBdev(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run clone bdev command")
			}
		},
	}
}

func bdevLvolCloneBdev(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	uuid, err := spdkCli.BdevLvolCloneBdev(c.String("bdev"), c.String("lvs-name"), c.String("clone-name"))
	if err != nil {
		return err
	}

	return util.PrintObject(uuid)
}

func BdevLvolDecoupleParentCmd() cli.Command {
	return cli.Command{
		Name: "decouple",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "alias",
				Usage: "The alias of a lvol is <LVSTORE NAME>/<LVOL NAME>. Specify this or uuid",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "Specify this or alias",
			},
		},
		Usage: "decouple a lvol from its parent lvol: \"decouple --alias <LVSTORE NAME>/<LVOL NAME>\", or \"decouple --uuid <LVOL UUID>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolDecoupleParent(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run decouple parent bdev lvol command")
			}
		},
	}
}

func bdevLvolDecoupleParent(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	name := c.String("alias")
	if name == "" {
		name = c.String("uuid")
	}

	decoupled, err := spdkCli.BdevLvolDecoupleParent(name)
	if err != nil {
		return err
	}

	return util.PrintObject(decoupled)
}

func BdevLvolSetParentCmd() cli.Command {
	return cli.Command{
		Name: "set-parent",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "lvol",
				Usage:    "Alias or UUID for the lvol to set parent of. The alias of a lvol is <LVSTORE NAME>/<LVOL NAME>.",
				Required: true,
			},
			cli.StringFlag{
				Name:     "parent",
				Usage:    "Alias or UUID for the snapshot lvol to become the parent",
				Required: true,
			},
		},
		Usage: "set a snapshot as the parent of a lvol: \"set-parent --lvol <LVSTORE NAME>/<CLONE LVOL NAME>\" --parent <LVSTORE NAME>/<PARENT SNAPSHOT LVOL NAME>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolSetParent(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run set parent bdev lvol command")
			}
		},
	}
}

func bdevLvolSetParent(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	set, err := spdkCli.BdevLvolSetParent(c.String("lvol"), c.String("parent"))
	if err != nil {
		return err
	}

	return util.PrintObject(set)
}

func BdevLvolResizeCmd() cli.Command {
	return cli.Command{
		Name: "resize",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "alias",
				Usage: "The alias of a snapshot lvol is <LVSTORE NAME>/<LVOL NAME>. Specify this or uuid",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "Specify this or alias",
			},
			cli.Uint64Flag{
				Name:     "size-in-mib",
				Required: true,
			},
		},
		Usage: "resize a lvol to a new size: \"resize --alias <LVSTORE NAME>/<LVOL NAME> --size-in-mib <SIZE>\", or \"resize --uuid <LVOL UUID> --size-in-mib <SIZE>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolResize(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run resize bdev lvol command")
			}
		},
	}
}

func bdevLvolResize(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	name := c.String("alias")
	if name == "" {
		name = c.String("uuid")
	}

	resized, err := spdkCli.BdevLvolResize(name, c.Uint64("size-in-mib"))
	if err != nil {
		return err
	}

	return util.PrintObject(resized)
}

func BdevLvolStartShallowCopyCmd() cli.Command {
	return cli.Command{
		Name: "shallow-copy-start",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "src-lvol-alias",
				Usage: "The alias of a snapshot lvol to create a copy from, which is <LVSTORE NAME>/<LVOL NAME>. Specify this or uuid",
			},
			cli.StringFlag{
				Name:  "src-lvol-uuid",
				Usage: "Specify this or alias",
			},
			cli.StringFlag{
				Name:     "dst-bdev-name",
				Usage:    "Name of the bdev that acts as destination for the copy",
				Required: true,
			},
		},
		Usage: "start a copy of active clusters/data from a read-only logical volume to a bdev: \"shallow-copy-start --src-lvol-alias <LVSTORE NAME>/<LVOL NAME> --dst-bdev-name <BDEV NAME>\", or \"shallow-copy --uuid <LVOL UUID> --dst-bdev-name <BDEV NAME>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolStartShallowCopy(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run start shallow copy bdev lvol command")
			}
		},
	}
}

func bdevLvolStartShallowCopy(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	srcLvolName := c.String("src-lvol-alias")
	if srcLvolName == "" {
		srcLvolName = c.String("src-lvol-uuid")
	}

	operationId, err := spdkCli.BdevLvolStartShallowCopy(srcLvolName, c.String("dst-bdev-name"))
	if err != nil {
		return err
	}

	return util.PrintObject(operationId)
}

func BdevLvolCheckShallowCopyCmd() cli.Command {
	return cli.Command{
		Name: "shallow-copy-check",
		Flags: []cli.Flag{
			cli.UintFlag{
				Name:     "operation-id",
				Usage:    "The operation ID returned by the command shallow-copy-start",
				Required: true,
			},
		},
		Usage: "check the status of a previously started shallow copy: \"shallow-copy-check --operation-id <OPERATION ID>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolCheckShallowCopy(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run check shallow copy bdev lvol command")
			}
		},
	}
}

func bdevLvolCheckShallowCopy(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	operationId := c.Uint("operation-id")

	copied, err := spdkCli.BdevLvolCheckShallowCopy(uint32(operationId))
	if err != nil {
		return err
	}

	return util.PrintObject(copied)
}

func BdevLvolGetXattrCmd() cli.Command {
	return cli.Command{
		Name: "get-xattr",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "alias",
				Usage: "The alias of a lvol is <LVSTORE NAME>/<LVOL NAME>. Specify this or uuid",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "Specify this or alias",
			},
			cli.StringFlag{
				Name:  "xattr-name",
				Usage: "Specify the xattr name",
			},
		},
		Usage: "get xattr value of a lvol: \"get-xattr --name <LVSTORE NAME>/<LVOL NAME> --xattr-name <XATTR NAME>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolGetXattr(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get bdev lvol xattr command")
			}
		},
	}
}

func bdevLvolGetXattr(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	name := c.String("alias")
	if name == "" {
		name = c.String("uuid")
	}

	bdevLvolGetResp, err := spdkCli.BdevLvolGetXattr(name, c.String("xattr-name"))
	if err != nil {
		return err
	}

	return util.PrintObject(bdevLvolGetResp)
}

func BdevLvolGetFragmapCmd() cli.Command {
	return cli.Command{
		Name: "get-fragmap",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "alias",
				Usage: "The alias of a lvol is <LVSTORE NAME>/<LVOL NAME>. Specify this or uuid",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "Specify this or alias",
			},
			cli.Uint64Flag{
				Name:     "offset",
				Usage:    "Offset in bytes of the specific segment of the logical volume (Default: 0)",
				Required: false,
			},
			cli.Uint64Flag{
				Name:     "size",
				Usage:    "Size in bytes of the specific segment of the logical volume (Default: 0 for representing the entire file)",
				Required: false,
			},
		},
		Usage: "Get fragmap of the specific segment of the logical volume: \"get-fragmap --uuid <LVOL UUID> --offset <OFFSET> --size <SIZE>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolGetFragmap(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get lvol get fragmap command")
			}
		},
	}
}

func bdevLvolGetFragmap(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	name := c.String("alias")
	if name == "" {
		name = c.String("uuid")
	}
	offset := c.Uint64("offset")
	size := c.Uint64("size")

	output, err := spdkCli.BdevLvolGetFragmap(name, offset, size)
	if err != nil {
		return err
	}

	return util.PrintObject(output)
}

func BdevLvolRenameCmd() cli.Command {
	return cli.Command{
		Name: "rename",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "old-name",
				Usage:    "The UUID or alias (<LVSTORE NAME>/<LVOL NAME>) of the existing logical volume",
				Required: true,
			},
			cli.StringFlag{
				Name:     "new-name",
				Usage:    "New logical volume name.",
				Required: true,
			},
		},
		Usage: "Rename a logical volume. New name will rename only the alias of the logical volume: \"rename --old-name <LVSTORE NAME>/<LVOL NAME> --new-name <LVOL NAME>\" or \"rename --old-name <UUID> --new-name <LVOL NAME>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolRename(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run rename bdev lvol command")
			}
		},
	}
}

func bdevLvolRename(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	oldName := c.String("old-name")
	newName := c.String("new-name")

	if oldName == "" || newName == "" {
		return fmt.Errorf("both old-name and new-name must be provided")
	}
	if oldName == newName {
		return fmt.Errorf("old-name and new-name must be different")
	}

	renamed, err := spdkCli.BdevLvolRename(oldName, newName)
	if err != nil {
		return fmt.Errorf("failed to rename logical volume from %q to %q: %v", oldName, newName, err)
	}

	return util.PrintObject(renamed)
}

func BdevLvolRegisterSnapshotChecksumCmd() cli.Command {
	return cli.Command{
		Name: "register-snapshot-checksum",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "alias",
				Usage: "The alias of a snapshot is <LVSTORE NAME>/<SNAPSHOT NAME>. Specify this or uuid",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "Specify this or alias",
			},
		},
		Usage: "compute and store checksum of snapshot's data: \"register-snapshot-checksum --alias <LVSTORE NAME>/<LVOL NAME>\"," +
			" or \"register-snapshot-checksum --uuid <LVOL UUID>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolRegisterSnapshotChecksum(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run register snapshot checksum command")
			}
		},
	}
}

func bdevLvolRegisterSnapshotChecksum(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	name := c.String("alias")
	if name == "" {
		name = c.String("uuid")
	}
	if name == "" {
		return fmt.Errorf("either alias or uuid must be provided")
	}

	registered, err := spdkCli.BdevLvolRegisterSnapshotChecksum(name)
	if err != nil {
		return fmt.Errorf("failed to register checksum for snapshot %q: %v", name, err)
	}

	return util.PrintObject(registered)
}

func BdevLvolGetSnapshotChecksumCmd() cli.Command {
	return cli.Command{
		Name: "get-snapshot-checksum",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "alias",
				Usage: "The alias of a snapshot is <LVSTORE NAME>/<SNAPSHOT NAME>. Specify this or uuid",
			},
			cli.StringFlag{
				Name:  "uuid",
				Usage: "Specify this or alias",
			},
		},
		Usage: "get checksum of snapshot's data: \"get-snapshot-checksum --alias <LVSTORE NAME>/<LVOL NAME>\"," +
			" or \"get-snapshot-checksum --uuid <LVOL UUID>\"",
		Action: func(c *cli.Context) {
			if err := bdevLvolGetSnapshotChecksum(c); err != nil {
				logrus.WithError(err).Fatalf("Failed to run get snapshot checksum command")
			}
		},
	}
}

func bdevLvolGetSnapshotChecksum(c *cli.Context) error {
	spdkCli, err := client.NewClient(context.Background())
	if err != nil {
		return err
	}

	name := c.String("alias")
	if name == "" {
		name = c.String("uuid")
	}
	if name == "" {
		return fmt.Errorf("either alias or uuid must be provided")
	}

	checksum, err := spdkCli.BdevLvolGetSnapshotChecksum(name)
	if err != nil {
		return fmt.Errorf("failed to get checksum for snapshot %q: %v", name, err)
	}
	if checksum == "" {
		return fmt.Errorf("no checksum found for snapshot %q", name)
	}

	return util.PrintObject(checksum)
}
