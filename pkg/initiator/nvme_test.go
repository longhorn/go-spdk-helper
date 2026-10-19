package initiator

import (
	"os"
	"path/filepath"
	"strings"

	. "gopkg.in/check.v1"
)

const (
	testSubsystemNQN      = "nqn.2023-01.io.longhorn.spdk:volume-test"
	testOtherSubsystemNQN = "nqn.2023-01.io.longhorn.spdk:volume-other"
)

const (
	testDeletingPath = `{"Name":"nvme0","Transport":"tcp","Address":"traddr=10.0.0.1,trsvcid=20006","State":"deleting"}`
	testLivePath     = `{"Name":"nvme1","Transport":"tcp","Address":"traddr=10.0.0.2,trsvcid=20343","State":"live"}`
	testUnknownPath  = `{"Name":"nvme2","Transport":"tcp","Address":"traddr=10.0.0.3,trsvcid=20344"}`
	// testSecondLivePath is a second usable path of the same subsystem, which is what
	// native multipath leaves behind after a switchover.
	testSecondLivePath = `{"Name":"nvme3","Transport":"tcp","Address":"traddr=10.0.0.4,trsvcid=20345","State":"live"}`
)

// A path the kernel is still failing must be left to ctrl_loss_tmo: disconnecting it
// re-arms the I/O requeueing that failfast had just stopped.
func (s *InitiatorTestSuite) TestDisconnectUsableTargetPathsSkipsUnusablePaths(c *C) {
	disconnected := filepath.Join(c.MkDir(), "disconnected")
	script := `#!/bin/sh
case "$1" in
	--version) echo "nvme version 1.16" ;;
	list-subsys) echo '{"Subsystems":[{"NQN":"` + testSubsystemNQN + `","Paths":[` + testDeletingPath + `,` + testUnknownPath + `,` + testLivePath + `]}]}' ;;
	disconnect) echo "$3" >> ` + disconnected + ` ;;
esac
exit 0
`
	restorePath := setupFakeCommandPath(c, map[string]string{"nvme": script})
	defer restorePath()

	executor, err := newExecutorWithoutNamespace()
	c.Assert(err, IsNil)

	c.Assert(DisconnectUsableTargetPaths(testSubsystemNQN, executor), IsNil)

	recorded, err := os.ReadFile(disconnected)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(string(recorded), "/dev/nvme1"), Equals, true)
	c.Assert(strings.Contains(string(recorded), "/dev/nvme0"), Equals, false)
	// A state we cannot read is not evidence that the path is safe to delete.
	c.Assert(strings.Contains(string(recorded), "/dev/nvme2"), Equals, false)
}

// Every path that refused to disconnect has to reach the caller. Keeping only the
// last one hides how much of the subsystem is still connected.
func (s *InitiatorTestSuite) TestDisconnectUsableTargetPathsReportsEveryFailure(c *C) {
	script := `#!/bin/sh
case "$1" in
	--version) echo "nvme version 1.16" ;;
	list-subsys) echo '{"Subsystems":[{"NQN":"` + testSubsystemNQN + `","Paths":[` + testLivePath + `,` + testSecondLivePath + `]}]}' ;;
	disconnect) echo "disconnect refused" >&2; exit 1 ;;
esac
exit 0
`
	restorePath := setupFakeCommandPath(c, map[string]string{"nvme": script})
	defer restorePath()

	executor, err := newExecutorWithoutNamespace()
	c.Assert(err, IsNil)

	err = DisconnectUsableTargetPaths(testSubsystemNQN, executor)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "nvme1"), Equals, true)
	c.Assert(strings.Contains(err.Error(), "nvme3"), Equals, true)
}
