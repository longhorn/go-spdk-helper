package nvme

import (
	"fmt"
	"strings"
	"testing"

	execlib "github.com/longhorn/go-common-libs/exec"
	nslib "github.com/longhorn/go-common-libs/ns"
	typeslib "github.com/longhorn/go-common-libs/types"

	. "gopkg.in/check.v1"
)

const (
	TestHostNQN       = "hostnqn"
	TestNQN           = "testnqn"
	TestTransportType = "loop"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})

func ProbeNvmeModules() error {
	namespaces := []typeslib.Namespace{typeslib.NamespaceMnt, typeslib.NamespaceIpc, typeslib.NamespaceNet}
	ne, err := nslib.NewNamespaceExecutor(typeslib.ProcessNone, typeslib.HostProcDirectory, namespaces)
	if err != nil {
		return err
	}
	_, err = ne.Execute("modprobe", []string{"nvme", "nvme_loop", "nvmet", "nvme_fabrics"}, typeslib.ExecuteDefaultTimeout)
	return err
}

func (s *TestSuite) TestNvmeVersion(c *C) {
	fmt.Println("Testing nvme-version With Host Namespace")

	namespaces := []typeslib.Namespace{typeslib.NamespaceMnt, typeslib.NamespaceIpc, typeslib.NamespaceNet}
	ne, err := nslib.NewNamespaceExecutor(typeslib.ProcessNone, typeslib.ProcDirectory, namespaces)
	c.Assert(err, IsNil)

	major, minor, err := cliVersion(ne)
	c.Assert(err, IsNil)
	nvmeVersion := fmt.Sprintf("%d.%d", major, minor)

	e := execlib.NewExecutor()
	output, err := e.Execute(nil, "nvme", []string{"--version"}, typeslib.ExecuteDefaultTimeout)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(output, nvmeVersion), Equals, true)
}

func (s *TestSuite) TestNvmeConnectionAndDisconnection(c *C) {
	fmt.Println("Testing nvme connect and disconnect")

	err := ProbeNvmeModules()
	c.Assert(err, IsNil)

	namespaces := []typeslib.Namespace{typeslib.NamespaceMnt, typeslib.NamespaceIpc, typeslib.NamespaceNet}
	ne, err := nslib.NewNamespaceExecutor(typeslib.ProcessNone, typeslib.ProcDirectory, namespaces)
	c.Assert(err, IsNil)

	// Connect to a nvme target
	_, err = connect(TestHostNQN, TestNQN, TestTransportType, "", "", ne)
	c.Assert(err, IsNil)
	defer func() {
		err := disconnect(TestNQN, ne)
		c.Assert(err, IsNil)
	}()

	// Check device is connected
	devices, err := GetDevices("", "", "", ne)
	c.Assert(err, IsNil)
	found := false
	for _, device := range devices {
		if device.SubsystemNQN == TestNQN {
			found = true
			break
		}
	}
	c.Assert(found, Equals, true)

	// List subsystems
	subsystems, err := listSubsystems("", ne)
	c.Assert(err, IsNil)
	found = false
	for _, subsystem := range subsystems {
		if subsystem.NQN == TestNQN {
			found = true
			break
		}
	}
	c.Assert(found, Equals, true)
}
