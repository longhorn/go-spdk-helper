package spdk

import (
	"fmt"
	"github.com/longhorn/go-spdk-helper/pkg/types"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	"github.com/longhorn/go-spdk-helper/pkg/spdk/target"
	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
	"github.com/longhorn/go-spdk-helper/pkg/util"

	. "gopkg.in/check.v1"
)

var (
	defaultDeviceName = "test-device"
	defaultDevicePath = filepath.Join("/tmp", defaultDeviceName)

	defaultDeviceSize    = uint64(100 * types.MiB)
	defaultLvolSizeInMiB = uint64(10)

	defaultPort1 = "4421"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})

func GetSPDKDir() string {
	spdkDir := os.Getenv("SPDK_DIR")
	if spdkDir != "" {
		return spdkDir
	}
	return filepath.Join(os.Getenv("GOPATH"), "src/github.com/longhorn/spdk")
}

func LaunchTestSPDKTarget(c *C, execute func(name string, args []string) (string, error)) {
	targetReady := false
	if spdkCli, err := client.NewClient(); err == nil {
		if _, err := spdkCli.BdevGetBdevs("", 0); err == nil {
			targetReady = true
		}
	}

	if !targetReady {
		go func() {
			err := target.StartTarget(GetSPDKDir(), execute)
			c.Assert(err, IsNil)
		}()

		for cnt := 0; cnt < 30; cnt++ {
			if spdkCli, err := client.NewClient(); err == nil {
				if _, err := spdkCli.BdevGetBdevs("", 0); err == nil {
					targetReady = true
					break
				}
			}
			time.Sleep(time.Second)
		}
	}

	c.Assert(targetReady, Equals, true)
}

func PrepareDeviceFile(c *C) {
	err := os.RemoveAll(defaultDevicePath)
	c.Assert(err, IsNil)

	f, err := os.Create(defaultDevicePath)
	c.Assert(err, IsNil)
	err = f.Close()
	c.Assert(err, IsNil)

	err = os.Truncate(defaultDevicePath, int64(defaultDeviceSize))
	c.Assert(err, IsNil)
}

func (s *TestSuite) TestSPDKTargetWithHostNamespace(c *C) {
	fmt.Println("Testing SPDK Target With Host Namespace")

	hostProcPath := "/host/proc"
	ne, err := util.NewNamespaceExecutor(util.GetHostNamespacePath(hostProcPath))
	c.Assert(err, IsNil)

	LaunchTestSPDKTarget(c, ne.Execute)
}

func (s *TestSuite) TestSPDKBasic(c *C) {
	fmt.Println("Testing SPDK Basic")
	LaunchTestSPDKTarget(c, util.Execute)
	PrepareDeviceFile(c)
	defer func() {
		os.RemoveAll(defaultDevicePath)
	}()

	spdkCli, err := client.NewClient()
	c.Assert(err, IsNil)

	// Do blindly cleanup
	spdkCli.DeleteDevice(defaultDeviceName, defaultDeviceName)

	bdevAioName, lvsName, lvsUUID, err := spdkCli.AddDevice(defaultDevicePath, defaultDeviceName, types.MiB)
	c.Assert(err, IsNil)
	defer func() {
		err := spdkCli.DeleteDevice(bdevAioName, lvsName)
		c.Assert(err, IsNil)
	}()
	c.Assert(bdevAioName, Equals, defaultDeviceName)
	c.Assert(lvsName, Equals, defaultDeviceName)
	c.Assert(lvsUUID, Not(Equals), "")

	bdevAioInfoList, err := spdkCli.BdevAioGet(bdevAioName, 0)
	c.Assert(err, IsNil)
	c.Assert(len(bdevAioInfoList), Equals, 1)
	bdevAio := bdevAioInfoList[0]
	c.Assert(bdevAio.Name, Equals, bdevAioName)
	c.Assert(uint64(bdevAio.BlockSize)*bdevAio.NumBlocks, Equals, defaultDeviceSize)

	lvsList, err := spdkCli.BdevLvolGetLvstore(lvsName, "")
	c.Assert(err, IsNil)
	c.Assert(len(lvsList), Equals, 1)
	lvs := lvsList[0]
	c.Assert(lvs.UUID, Equals, lvsUUID)
	c.Assert(lvs.BaseBdev, Equals, bdevAio.Name)
	c.Assert(int(lvs.BlockSize), Equals, int(bdevAio.BlockSize))
	c.Assert(lvs.ClusterSize*(lvs.TotalDataClusters+1), Equals, defaultDeviceSize)

	lvolName1, lvolName2 := "test-lvol1", "test-lvol2"
	lvolUUID1, err := spdkCli.BdevLvolCreate(lvsName, lvolName1, "", defaultLvolSizeInMiB, "", true)
	c.Assert(err, IsNil)
	defer func() {
		deleted, err := spdkCli.BdevLvolDelete(lvolUUID1)
		c.Assert(err, IsNil)
		c.Assert(deleted, Equals, true)
	}()
	lvolUUID2, err := spdkCli.BdevLvolCreate(lvsName, lvolName2, "", defaultLvolSizeInMiB, "", true)
	c.Assert(err, IsNil)
	defer func() {
		deleted, err := spdkCli.BdevLvolDelete(lvolUUID2)
		c.Assert(err, IsNil)
		c.Assert(deleted, Equals, true)
	}()

	lvolList, err := spdkCli.BdevLvolGet("", 0)
	c.Assert(err, IsNil)
	c.Assert(len(lvolList), Equals, 2)
	for _, lvol := range lvolList {
		c.Assert(len(lvol.Aliases), Equals, 1)
		c.Assert(uint64(lvol.BlockSize)*lvol.NumBlocks, Equals, defaultLvolSizeInMiB*types.MiB)
		c.Assert(lvol.DriverSpecific.Lvol, NotNil)
		c.Assert(lvol.DriverSpecific.Lvol.ThinProvision, Equals, true)
		c.Assert(lvol.DriverSpecific.Lvol.BaseBdev, Equals, defaultDeviceName)
		c.Assert(lvol.DriverSpecific.Lvol.Snapshot, Equals, false)
		c.Assert(lvol.DriverSpecific.Lvol.Clone, Equals, false)
		if lvol.UUID == lvolUUID1 {
			c.Assert(lvol.Aliases[0], Equals, fmt.Sprintf("%s/%s", lvsName, lvolName1))
		}
		if lvol.UUID == lvolUUID2 {
			c.Assert(lvol.Aliases[0], Equals, fmt.Sprintf("%s/%s", lvsName, lvolName2))
		}
	}

	snapLvolUUID1, err := spdkCli.BdevLvolSnapshot(lvolUUID1, "snap11")
	c.Assert(err, IsNil)
	defer func() {
		deleted, err := spdkCli.BdevLvolDelete(snapLvolUUID1)
		c.Assert(err, IsNil)
		c.Assert(deleted, Equals, true)
	}()
	lvolList, err = spdkCli.BdevLvolGet(snapLvolUUID1, 0)
	c.Assert(err, IsNil)
	c.Assert(len(lvolList), Equals, 1)
	snapLvol1 := lvolList[0]
	c.Assert(snapLvol1.DriverSpecific.Lvol, NotNil)
	c.Assert(snapLvol1.DriverSpecific.Lvol.Snapshot, Equals, true)
	c.Assert(snapLvol1.DriverSpecific.Lvol.Clone, Equals, false)

	cloneLvolUUID1, err := spdkCli.BdevLvolClone(snapLvolUUID1, "clone111")
	c.Assert(err, IsNil)
	defer func() {
		deleted, err := spdkCli.BdevLvolDelete(cloneLvolUUID1)
		c.Assert(err, IsNil)
		c.Assert(deleted, Equals, true)
	}()
	lvolList, err = spdkCli.BdevLvolGet(cloneLvolUUID1, 0)
	c.Assert(err, IsNil)
	c.Assert(len(lvolList), Equals, 1)
	cloneLvol1 := lvolList[0]
	c.Assert(cloneLvol1.DriverSpecific.Lvol, NotNil)
	c.Assert(cloneLvol1.DriverSpecific.Lvol.Snapshot, Equals, false)
	c.Assert(cloneLvol1.DriverSpecific.Lvol.Clone, Equals, true)

	decoupled, err := spdkCli.BdevLvolDecoupleParent(cloneLvolUUID1)
	c.Assert(err, IsNil)
	c.Assert(decoupled, Equals, true)
	lvolList, err = spdkCli.BdevLvolGet(cloneLvolUUID1, 0)
	c.Assert(err, IsNil)
	c.Assert(len(lvolList), Equals, 1)
	decoupledCloneLvol1 := lvolList[0]
	c.Assert(decoupledCloneLvol1.DriverSpecific.Lvol, NotNil)
	c.Assert(decoupledCloneLvol1.DriverSpecific.Lvol.Snapshot, Equals, false)
	c.Assert(decoupledCloneLvol1.DriverSpecific.Lvol.Clone, Equals, false)

	raidName := "test-raid"
	created, err := spdkCli.BdevRaidCreate(raidName, spdktypes.BdevRaidLevelRaid1, 0, []string{lvolUUID1, lvolUUID2})
	c.Assert(err, IsNil)
	c.Assert(created, Equals, true)
	defer func() {
		deleted, err := spdkCli.BdevRaidDelete(raidName)
		c.Assert(err, IsNil)
		c.Assert(deleted, Equals, true)
	}()

	raidList, err := spdkCli.BdevRaidGetBdevs(spdktypes.BdevRaidCategoryOnline)
	c.Assert(err, IsNil)
	c.Assert(len(raidList), Equals, 1)
	c.Assert(raidList[0].Name, Equals, raidName)
	c.Assert(int(raidList[0].NumBaseBdevs), Equals, 2)
	c.Assert(int(raidList[0].NumBaseBdevsDiscovered), Equals, 2)
	c.Assert(len(raidList[0].BaseBdevsList), Equals, 2)
	if raidList[0].BaseBdevsList[0] != lvolUUID1 {
		c.Assert(raidList[0].BaseBdevsList[0], Equals, lvolUUID2)
		c.Assert(raidList[0].BaseBdevsList[1], Equals, lvolUUID1)
	} else {
		c.Assert(raidList[0].BaseBdevsList[0], Equals, lvolUUID1)
		c.Assert(raidList[0].BaseBdevsList[1], Equals, lvolUUID2)
	}

	nqn := types.GetNQN(raidName)
	err = spdkCli.StartExposeBdev(nqn, raidName, types.LocalIP, defaultPort1)
	c.Assert(err, IsNil)
	defer func() {
		err = spdkCli.StopExposeBdev(nqn)
		c.Assert(err, IsNil)
	}()

	transportList, err := spdkCli.NvmfGetTransports(spdktypes.NvmeTransportTypeTCP, "")
	c.Assert(err, IsNil)
	c.Assert(len(transportList), Equals, 1)

	subsystemList, err := spdkCli.NvmfGetSubsystems("")
	c.Assert(err, IsNil)
	c.Assert(len(transportList) >= 1, Equals, true)
	found := false
	for _, subsys := range subsystemList {
		if subsys.Nqn == nqn {
			found = true
			break
		}
	}
	c.Assert(found, Equals, true)

	nsList, err := spdkCli.NvmfSubsystemsGetNss(nqn, "", 0)
	c.Assert(err, IsNil)
	c.Assert(len(nsList), Equals, 1)
	c.Assert(nsList[0].BdevName, Equals, raidName)
}
