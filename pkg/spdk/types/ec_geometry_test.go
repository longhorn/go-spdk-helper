package types

import "testing"

// Golden values for the EC front reservation across every valid strip size. These pin
// the formula's arithmetic against frozen results (it dips below the 2052 plateau for
// the three smallest strips). They cannot catch drift from SPDK's actual carve, since
// this repo has no SPDK source; the engine create-time capacity guard is that backstop.
func TestEcFrontReservation(t *testing.T) {
	tests := map[string]struct {
		stripSizeKB    uint32
		expectedStrips uint64
		expectedBytes  uint64
	}{
		"4KiB":    {stripSizeKB: 4, expectedStrips: 2038, expectedBytes: 8347648},
		"8KiB":    {stripSizeKB: 8, expectedStrips: 2046, expectedBytes: 16760832},
		"16KiB":   {stripSizeKB: 16, expectedStrips: 2050, expectedBytes: 33587200},
		"32KiB":   {stripSizeKB: 32, expectedStrips: 2052, expectedBytes: 67239936},
		"64KiB":   {stripSizeKB: 64, expectedStrips: 2052, expectedBytes: 134479872},
		"128KiB":  {stripSizeKB: 128, expectedStrips: 2052, expectedBytes: 268959744},
		"256KiB":  {stripSizeKB: 256, expectedStrips: 2052, expectedBytes: 537919488},
		"512KiB":  {stripSizeKB: 512, expectedStrips: 2052, expectedBytes: 1075838976},
		"1024KiB": {stripSizeKB: 1024, expectedStrips: 2052, expectedBytes: 2151677952},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := EcFrontReservationStrips(test.stripSizeKB); got != test.expectedStrips {
				t.Fatalf("EcFrontReservationStrips(%d) = %d, want %d", test.stripSizeKB, got, test.expectedStrips)
			}
			if got := EcFrontReservationBytes(test.stripSizeKB); got != test.expectedBytes {
				t.Fatalf("EcFrontReservationBytes(%d) = %d, want %d", test.stripSizeKB, got, test.expectedBytes)
			}
		})
	}
}
