package types

import "fmt"

const (
	DefaultJSONServerNetwork    = "unix"
	DefaultUnixDomainSocketPath = "/var/tmp/spdk.sock"

	MiB = 1 << 20
)

func GetNQN(name string) string {
	return fmt.Sprintf("nqn.2023-01.io.longhorn.spdk:%s", name)
}
