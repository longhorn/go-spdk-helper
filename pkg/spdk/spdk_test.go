package spdk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	nslib "github.com/longhorn/go-common-libs/ns"
	typeslib "github.com/longhorn/go-common-libs/types"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
	"github.com/longhorn/go-spdk-helper/pkg/spdk/target"
	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
	"github.com/longhorn/go-spdk-helper/pkg/types"

	. "gopkg.in/check.v1"
)

var (
	defaultDeviceName = "test-device"
	defaultDevicePath = filepath.Join("/tmp", defaultDeviceName)

	defaultDeviceSize    = uint64(1000 * types.MiB)
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

func LaunchTestSPDKTarget(c *C, execute func(binary string, args []string, timeout time.Duration) (string, error)) {
	targetReady := false
	if spdkCli, err := client.NewClient(context.Background()); err == nil {
		if _, err := spdkCli.BdevGetBdevs("", 0); err == nil {
			targetReady = true
		}
	}

	if !targetReady {
		go func() {
			err := target.StartTarget(GetSPDKDir(), []string{"2>&1 | tee /tmp/spdk_tgt.log"}, execute)
			c.Assert(err, IsNil)
		}()

		for cnt := 0; cnt < 30; cnt++ {
			if spdkCli, err := client.NewClient(context.Background()); err == nil {
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

	namespaces := []typeslib.Namespace{typeslib.NamespaceMnt, typeslib.NamespaceIpc, typeslib.NamespaceNet}
	ne, err := nslib.NewNamespaceExecutor(typeslib.ProcessNone, typeslib.HostProcDirectory, namespaces)
	c.Assert(err, IsNil)

	LaunchTestSPDKTarget(c, ne.Execute)
}

func (s *TestSuite) TestSPDKBasic(c *C) {
	fmt.Println("Testing SPDK Basic")

	namespaces := []typeslib.Namespace{typeslib.NamespaceMnt, typeslib.NamespaceIpc, typeslib.NamespaceNet}
	ne, err := nslib.NewNamespaceExecutor(typeslib.ProcessNone, typeslib.ProcDirectory, namespaces)
	c.Assert(err, IsNil)

	LaunchTestSPDKTarget(c, ne.Execute)
	PrepareDeviceFile(c)
	defer func() {
		os.RemoveAll(defaultDevicePath)
	}()

	spdkCli, err := client.NewClient(context.Background())
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
	// The meta info may take space
	c.Assert(lvs.ClusterSize*(lvs.TotalDataClusters+4), Equals, defaultDeviceSize)

	lvolName1, lvolName2 := "test-lvol1", "test-lvol2"
	lvolUUID1, err := spdkCli.BdevLvolCreate(lvsName, "", lvolName1, defaultLvolSizeInMiB, "", true)
	c.Assert(err, IsNil)
	defer func() {
		deleted, err := spdkCli.BdevLvolDelete(lvolUUID1)
		c.Assert(err, IsNil)
		c.Assert(deleted, Equals, true)
	}()
	lvolUUID2, err := spdkCli.BdevLvolCreate("", lvsUUID, lvolName2, defaultLvolSizeInMiB, "", true)
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
		c.Assert(lvol.DriverSpecific.Lvol.LvolStoreUUID, Equals, lvsUUID)
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

	// Test 2 kinds of raid get APIs
	raidInfoList, err := spdkCli.BdevRaidGetInfoByCategory(spdktypes.BdevRaidCategoryOnline)
	c.Assert(err, IsNil)
	c.Assert(len(raidInfoList), Equals, 1)
	c.Assert(raidInfoList[0].Name, Equals, raidName)
	c.Assert(int(raidInfoList[0].NumBaseBdevs), Equals, 2)
	c.Assert(int(raidInfoList[0].NumBaseBdevsDiscovered), Equals, 2)
	c.Assert(len(raidInfoList[0].BaseBdevsList), Equals, 2)
	if raidInfoList[0].BaseBdevsList[0].UUID != lvolUUID1 {
		c.Assert(raidInfoList[0].BaseBdevsList[0].UUID, Equals, lvolUUID2)
		c.Assert(raidInfoList[0].BaseBdevsList[1].UUID, Equals, lvolUUID1)
	} else {
		c.Assert(raidInfoList[0].BaseBdevsList[0].UUID, Equals, lvolUUID1)
		c.Assert(raidInfoList[0].BaseBdevsList[1].UUID, Equals, lvolUUID2)
	}
	c.Assert(raidInfoList[0].BaseBdevsList[0].IsConfigured, Equals, true)
	c.Assert(raidInfoList[0].BaseBdevsList[1].IsConfigured, Equals, true)
	raidBdevList, err := spdkCli.BdevRaidGet(raidName, 0)
	c.Assert(len(raidBdevList), Equals, 1)
	raidBdev := raidBdevList[0]
	c.Assert(int(raidBdev.DriverSpecific.Raid.NumBaseBdevs), Equals, 2)
	c.Assert(int(raidBdev.DriverSpecific.Raid.NumBaseBdevsDiscovered), Equals, 2)
	c.Assert(int(raidBdev.DriverSpecific.Raid.NumBaseBdevsOperational), Equals, 2)
	c.Assert(len(raidBdev.DriverSpecific.Raid.BaseBdevsList), Equals, 2)
	if raidBdev.DriverSpecific.Raid.BaseBdevsList[0].UUID != lvolUUID1 {
		c.Assert(raidBdev.DriverSpecific.Raid.BaseBdevsList[0].UUID, Equals, lvolUUID2)
		c.Assert(raidBdev.DriverSpecific.Raid.BaseBdevsList[1].UUID, Equals, lvolUUID1)
	} else {
		c.Assert(raidBdev.DriverSpecific.Raid.BaseBdevsList[0].UUID, Equals, lvolUUID1)
		c.Assert(raidBdev.DriverSpecific.Raid.BaseBdevsList[1].UUID, Equals, lvolUUID2)
	}
	c.Assert(raidBdev.DriverSpecific.Raid.BaseBdevsList[0].IsConfigured, Equals, true)
	c.Assert(raidBdev.DriverSpecific.Raid.BaseBdevsList[1].IsConfigured, Equals, true)

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

	subsystemList, err := spdkCli.NvmfGetSubsystems("", "")
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

func (s *TestSuite) TestSPDKClientMultiThread(c *C) {
	fmt.Println("Testing SPDK Client Multi Thread")

	namespaces := []typeslib.Namespace{typeslib.NamespaceMnt, typeslib.NamespaceIpc, typeslib.NamespaceNet}
	ne, err := nslib.NewNamespaceExecutor(typeslib.ProcessNone, typeslib.ProcDirectory, namespaces)
	c.Assert(err, IsNil)

	LaunchTestSPDKTarget(c, ne.Execute)
	PrepareDeviceFile(c)
	defer func() {
		os.RemoveAll(defaultDevicePath)
	}()

	spdkCli, err := client.NewClient(context.Background())
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
	// The meta info may take space
	c.Assert(lvs.ClusterSize*(lvs.TotalDataClusters+4), Equals, defaultDeviceSize)

	threadCount := 100
	repeatCount := 10

	wg := sync.WaitGroup{}
	wg.Add(threadCount)
	for i := 0; i < threadCount; i++ {
		num := i
		go func() {
			defer func() {
				wg.Done()
			}()

			lvolName := fmt.Sprintf("test-lvol-%d", num)
			for j := 0; j < repeatCount; j++ {
				lvolUUID, err := spdkCli.BdevLvolCreate(lvsName, "", lvolName, defaultLvolSizeInMiB, "", true)
				c.Assert(err, IsNil)

				lvolList, err := spdkCli.BdevLvolGet(lvolUUID, 0)
				c.Assert(err, IsNil)
				c.Assert(len(lvolList), Equals, 1)
				lvol := lvolList[0]
				c.Assert(len(lvol.Aliases), Equals, 1)
				c.Assert(uint64(lvol.BlockSize)*lvol.NumBlocks, Equals, defaultLvolSizeInMiB*types.MiB)
				c.Assert(lvol.DriverSpecific.Lvol, NotNil)
				c.Assert(lvol.DriverSpecific.Lvol.ThinProvision, Equals, true)
				c.Assert(lvol.DriverSpecific.Lvol.BaseBdev, Equals, defaultDeviceName)
				c.Assert(lvol.DriverSpecific.Lvol.Snapshot, Equals, false)
				c.Assert(lvol.DriverSpecific.Lvol.Clone, Equals, false)
				c.Assert(lvol.DriverSpecific.Lvol.LvolStoreUUID, Equals, lvsUUID)
				c.Assert(lvol.Aliases[0], Equals, fmt.Sprintf("%s/%s", lvsName, lvolName))

				deleted, err := spdkCli.BdevLvolDelete(lvolUUID)
				c.Assert(err, IsNil)
				c.Assert(deleted, Equals, true)
			}
		}()
	}

	wg.Wait()
}
