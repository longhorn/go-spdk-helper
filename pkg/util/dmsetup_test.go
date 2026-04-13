package util

import "testing"

func TestParseDependentDevicesFromString(t *testing.T) {
	knownDevices := map[string]*LonghornBlockDevice{
		"nvme4n1": {Source: BlockDevice{Name: "nvme4n1", Major: 259, Minor: 6}},
		"nvme4n2": {Source: BlockDevice{Name: "nvme4n2", Major: 259, Minor: 7}},
	}

	tests := map[string]struct {
		input    string
		expected []string
	}{
		"devname output": {
			input:    "1 dependencies  : (nvme4n2)",
			expected: []string{"nvme4n2"},
		},
		"major minor output": {
			input:    "1 dependencies  : (259, 7)",
			expected: []string{"nvme4n2"},
		},
		"mixed output": {
			input:    "2 dependencies  : (nvme4n1) (259, 7)",
			expected: []string{"nvme4n1", "nvme4n2"},
		},
		"unknown major minor": {
			input:    "1 dependencies  : (259, 99)",
			expected: []string{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := parseDependentDevicesFromString(test.input, knownDevices)
			if len(result) != len(test.expected) {
				t.Fatalf("unexpected result length %d, expected %d, result=%v", len(result), len(test.expected), result)
			}
			for index := range test.expected {
				if result[index] != test.expected[index] {
					t.Fatalf("unexpected result at index %d: got %s expected %s", index, result[index], test.expected[index])
				}
			}
		})
	}
}