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

	if namespace.Anagrpid != NvmfANAGroupID("1") {
		t.Fatalf("expected namespace anagrpid %q, got %q", NvmfANAGroupID("1"), namespace.Anagrpid)
	}
}
