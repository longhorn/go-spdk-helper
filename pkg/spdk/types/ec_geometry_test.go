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

func TestComputeShardSizeGolden(t *testing.T) {
	// The production failure geometry: 33 Gi, k=1, strip 64 KiB.
	if got := lvstoreMetadataBytes(33 << 30); got != 41943040 {
		t.Fatalf("lvstoreMetadataBytes(33Gi) = %d, want 41943040", got)
	}
	if got := ComputeShardSize(33<<30, 1, 64); got != 35611738112 {
		t.Fatalf("ComputeShardSize(33Gi,1,64) = %d, want 35611738112", got)
	}
}

// TestComputeShardSizeBacksSpec replays what SPDK does with the computed shard
// size (EC geometry, then bs_init metadata) and checks the lvstore can always
// hold the full spec. It cannot catch changes in SPDK itself; the engine's
// create-time check covers that.
func TestComputeShardSizeBacksSpec(t *testing.T) {
	const gib = int64(1) << 30
	specs := []int64{gib, 33 * gib, 100 * gib, 1024 * gib, 10 * 1024 * gib}
	strips := []int{4, 8, 16, 32, 64, 128, 256, 512, 1024}
	ks := []int{1, 2, 4, 8, 16}

	const pagesPerCluster = EcLvstoreClusterSize / ecLvstoreMdPageSize
	// One allocation mask: 8-byte spdk_bs_md_mask header + 1 bit per tracked
	// item, rounded up to whole md pages.
	maskPages := func(bits uint64) uint64 {
		return (8 + (bits+7)/8 + ecLvstoreMdPageSize - 1) / ecLvstoreMdPageSize
	}

	for _, spec := range specs {
		for _, strip := range strips {
			for _, k := range ks {
				shard := uint64(ComputeShardSize(spec, k, strip))
				stripBytes := uint64(strip) * 1024
				// EC bdev usable capacity, per ec_compute_geometry.
				numStripes := shard/stripBytes - EcFrontReservationStrips(uint32(strip))
				ecUsable := numStripes * uint64(k) * stripBytes
				// The bs_init carve; mdLen rounds down as SPDK does.
				clusters := ecUsable / EcLvstoreClusterSize
				mdLen := clusters * EcLvstoreMdPagesPerClusterRatio / 100
				mdPages := 1 + maskPages(mdLen) + max(maskPages(clusters), maskPages(mdLen)) + maskPages(mdLen) + mdLen
				mdClusters := (mdPages + pagesPerCluster - 1) / pagesPerCluster
				dataClusters := clusters - mdClusters
				headNeeds := (uint64(spec) + EcLvstoreClusterSize - 1) / EcLvstoreClusterSize
				if dataClusters < headNeeds {
					t.Errorf("spec=%d k=%d strip=%dKiB: %d data clusters < %d needed (shard=%d)",
						spec, k, strip, dataClusters, headNeeds, shard)
				}
			}
		}
	}
}
