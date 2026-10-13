package types

import "testing"

func TestKnownQpairState(t *testing.T) {
	known := []BdevNvmeQpairState{
		BdevNvmeQpairStateDisconnected,
		BdevNvmeQpairStateDisconnecting,
		BdevNvmeQpairStateConnecting,
		BdevNvmeQpairStateConnected,
		BdevNvmeQpairStateEnabling,
		BdevNvmeQpairStateEnabled,
		BdevNvmeQpairStateDestroying,
	}
	for _, state := range known {
		if !KnownQpairState(state) {
			t.Fatalf("expected state %q to be known", state)
		}
	}

	unknown := []BdevNvmeQpairState{"", "bogus", "connected", "UNKNOWN"}
	for _, state := range unknown {
		if KnownQpairState(state) {
			t.Fatalf("expected state %q to be unknown", state)
		}
	}
}
