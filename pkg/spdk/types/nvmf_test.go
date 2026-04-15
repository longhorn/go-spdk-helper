package types

import (
	"encoding/json"
	"testing"
)

func TestNvmfANAGroupIDUnmarshalJSON(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected NvmfANAGroupID
	}{
		"string": {
			input:    `"7"`,
			expected: NvmfANAGroupID("7"),
		},
		"number": {
			input:    `7`,
			expected: NvmfANAGroupID("7"),
		},
		"null": {
			input:    `null`,
			expected: NvmfANAGroupID(""),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var groupID NvmfANAGroupID
			if err := json.Unmarshal([]byte(test.input), &groupID); err != nil {
				t.Fatalf("failed to unmarshal %s: %v", test.input, err)
			}

			if groupID != test.expected {
				t.Fatalf("expected ANA group ID %q, got %q", test.expected, groupID)
			}
		})
	}
}

func TestNvmfSubsystemNamespaceUnmarshalNumericANAGroupID(t *testing.T) {
	input := []byte(`{"nsid":1,"bdev_name":"volume-bdev","anagrpid":1}`)

	var namespace NvmfSubsystemNamespace
	if err := json.Unmarshal(input, &namespace); err != nil {
		t.Fatalf("failed to unmarshal namespace: %v", err)
	}

	if namespace.Anagrpid != "1" {
		t.Fatalf("expected namespace anagrpid %q, got %q", "1", namespace.Anagrpid)
	}
}

func TestNvmfSubsystemNamespaceUnmarshalStringANAGroupID(t *testing.T) {
	input := []byte(`{"nsid":1,"bdev_name":"volume-bdev","anagrpid":"3"}`)

	var namespace NvmfSubsystemNamespace
	if err := json.Unmarshal(input, &namespace); err != nil {
		t.Fatalf("failed to unmarshal namespace: %v", err)
	}

	if namespace.Anagrpid != "3" {
		t.Fatalf("expected namespace anagrpid %q, got %q", "3", namespace.Anagrpid)
	}
}

func TestNvmfSubsystemNamespaceBackwardCompatible(t *testing.T) {
	// Verify the field is a plain string so old callers can use it directly
	ns := NvmfSubsystemNamespace{BdevName: "bdev0", Anagrpid: "1"}
	s := ns.Anagrpid
	if s != "1" {
		t.Fatalf("expected %q, got %q", "1", s)
	}
}
