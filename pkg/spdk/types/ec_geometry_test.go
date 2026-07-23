package types

import (
	"math"
	"testing"
)

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
	// The production failure geometry: 33 Gi, k=1, strip 64 KiB. Values frozen
	// at ratio 1000 (10x growth budget).
	if got := lvstoreMetadataBytes(33 << 30); got != 356515840 {
		t.Fatalf("lvstoreMetadataBytes(33Gi) = %d, want 356515840", got)
	}
	if got := ComputeShardSize(33<<30, 1, 64); got != 35926310912 {
		t.Fatalf("ComputeShardSize(33Gi,1,64) = %d, want 35926310912", got)
	}
	if got := ComputeShardSize(-1, 1, 64); got != -1 {
		t.Fatalf("ComputeShardSize(-1,1,64) = %d, want -1", got)
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
	// One allocation mask: 8-byte header + 1 bit per tracked item, in whole
	// md pages. Kept separate from lvstoreMaskPages on purpose: this replica
	// rounds down as SPDK does, the implementation rounds up, and the gap is
	// what the test checks. Do not deduplicate.
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
				// The bs_init carve; mdLen comes from setup_lvs_opts,
				// ratio*clusters/100, rounding down.
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

// lvstoreCreateThenGrow replays SPDK's create-then-grow behavior: bs_init on
// a device of createBytes, then grow after the device reaches grownBytes.
// Growth keeps the metadata region fixed and only extends the used_clusters
// mask, which must fit the region reserved at creation; see
// bs_load_try_to_grow:
// https://github.com/longhorn/spdk/blob/8d8b790728eb210141aab70fba0e30d5ba291ed3/lib/blob/blobstore.c#L10762
// Rounding follows SPDK (down), independent of the production helpers.
// Returns whether the grown lvstore can back headBytes.
func lvstoreCreateThenGrow(createBytes, grownBytes, headBytes uint64) bool {
	const pagesPerCluster = EcLvstoreClusterSize / ecLvstoreMdPageSize
	maskPages := func(bits uint64) uint64 {
		return (8 + (bits+7)/8 + ecLvstoreMdPageSize - 1) / ecLvstoreMdPageSize
	}

	// bs_init at creation: metadata region and md_len are frozen here. mdLen
	// comes from setup_lvs_opts, ratio*clusters/100.
	clusters0 := createBytes / EcLvstoreClusterSize
	mdLen := clusters0 * EcLvstoreMdPagesPerClusterRatio / 100
	clusterMaskRegion := max(maskPages(clusters0), maskPages(mdLen))
	mdPages := 1 + maskPages(mdLen) + clusterMaskRegion + maskPages(mdLen) + mdLen
	mdClusters := (mdPages + pagesPerCluster - 1) / pagesPerCluster

	// bs_load_try_to_grow: growth is refused when the new used_clusters mask
	// no longer fits the region reserved at creation.
	clusters1 := grownBytes / EcLvstoreClusterSize
	if maskPages(clusters1) > clusterMaskRegion {
		return false
	}

	headNeeds := (headBytes + EcLvstoreClusterSize - 1) / EcLvstoreClusterSize
	if clusters1-mdClusters < headNeeds {
		return false
	}
	// The fixed md budget must still cover one page per head-lvol cluster.
	return mdLen >= headNeeds
}

// TestLvstoreGrowCeiling checks the advertised expansion ceiling from both
// sides. Positive: an lvstore created at size S grows to
// EcLvstoreMaxGrowthFactor x S - this is the invariant the admission check
// relies on (ceiling <= what the replay allows). Negative: one factor past
// the ceiling must exhaust the fixed md budget. Only the md-budget cap is
// asserted directly: the mask-fit cap floats slightly above the ceiling
// (md_len counts metadata clusters, mask pages round up), so tripping it is
// not guaranteed at factor+1.
func TestLvstoreGrowCeiling(t *testing.T) {
	const gib = int64(1) << 30
	specs := []int64{gib, 33 * gib, 1024 * gib}
	ks := []int{1, 2, 4}
	const strip = 64

	ecUsable := func(spec int64, k int) uint64 {
		shard := uint64(ComputeShardSize(spec, k, strip))
		stripBytes := uint64(strip) * 1024
		numStripes := shard/stripBytes - EcFrontReservationStrips(strip)
		return numStripes * uint64(k) * stripBytes
	}

	for _, spec := range specs {
		for _, k := range ks {
			created := ecUsable(spec, k)

			grownSpec := spec * EcLvstoreMaxGrowthFactor
			if !lvstoreCreateThenGrow(created, ecUsable(grownSpec, k), uint64(grownSpec)) {
				t.Errorf("spec=%d k=%d: lvstore cannot grow to the %dx ceiling",
					spec, k, EcLvstoreMaxGrowthFactor)
			}

			// One factor past the ceiling: the md budget frozen at creation
			// (10x the creation clusters) cannot cover 11x the spec clusters,
			// because the metadata carve is far below one spec of headroom.
			overSpec := spec * (EcLvstoreMaxGrowthFactor + 1)
			mdLen := (created / EcLvstoreClusterSize) * EcLvstoreMdPagesPerClusterRatio / 100
			headNeeds := (uint64(overSpec) + EcLvstoreClusterSize - 1) / EcLvstoreClusterSize
			if mdLen >= headNeeds {
				t.Errorf("spec=%d k=%d: md budget %d covers %d clusters past the ceiling",
					spec, k, mdLen, headNeeds)
			}
			if lvstoreCreateThenGrow(created, ecUsable(overSpec, k), uint64(overSpec)) {
				t.Errorf("spec=%d k=%d: grow past the %dx ceiling unexpectedly fits",
					spec, k, EcLvstoreMaxGrowthFactor)
			}
		}
	}
}

// TestEcLvstoreMaxCreationSize pins the constant to the exact point where
// SPDK's setup_lvs_opts starts rejecting lvstore creation: the metadata page
// count, ratio x clusters / 100, must not exceed UINT32_MAX. At the limit it
// still fits; one cluster more is rejected with -EINVAL.
func TestEcLvstoreMaxCreationSize(t *testing.T) {
	clusters := uint64(EcLvstoreMaxCreationSize) / EcLvstoreClusterSize
	if clusters*EcLvstoreMdPagesPerClusterRatio/100 > math.MaxUint32 {
		t.Fatalf("EcLvstoreMaxCreationSize (%d clusters) already exceeds the md-pages limit", clusters)
	}
	if (clusters+1)*EcLvstoreMdPagesPerClusterRatio/100 <= math.MaxUint32 {
		t.Fatalf("EcLvstoreMaxCreationSize (%d clusters) is not the md-pages boundary", clusters)
	}
}

// TestEcUsableSizeWholeStrips pins the divisibility argument EcUsableSize's
// exactness rests on: every valid strip size divides the 2 MiB-aligned shard
// size, so the per-disk data bytes after the front reservation are whole
// strips and ec_compute_geometry truncates nothing.
func TestEcUsableSizeWholeStrips(t *testing.T) {
	const gib = int64(1) << 30
	for _, spec := range []int64{gib, 33 * gib, 1024 * gib} {
		for _, strip := range []int{4, 8, 16, 32, 64, 128, 256, 512, 1024} {
			for _, k := range []int{1, 2, 4, 8, 16} {
				stripBytes := uint64(strip) * 1024
				perDiskData := uint64(ComputeShardSize(spec, k, strip)) - EcFrontReservationBytes(uint32(strip))
				if perDiskData%stripBytes != 0 {
					t.Errorf("spec=%d k=%d strip=%dKiB: per-disk data %d is not whole strips",
						spec, k, strip, perDiskData)
				}
			}
		}
	}
}

// TestEcUsableSizeGolden pins EcUsableSize against a hand-computed
// ec_compute_geometry result for the production failure geometry (33 Gi,
// k=1, strip 64 KiB): shard 35926310912 minus the 134479872-byte front
// reservation leaves 546140 whole 64 KiB stripes of user data.
func TestEcUsableSizeGolden(t *testing.T) {
	if got := EcUsableSize(33<<30, 1, 64); got != 35791831040 {
		t.Fatalf("EcUsableSize(33Gi,1,64) = %d, want 35791831040", got)
	}
	if got := EcUsableSize(0, 1, 64); got != 0 {
		t.Fatalf("EcUsableSize(0,1,64) = %d, want 0", got)
	}
}

// TestValidateECCreationSize pins the admission boundary to the shard sizing
// formula across representative geometries: the reported maximum passes, one
// byte more fails, and the usable size SPDK sees stays within the md-pages
// cluster limit at the maximum.
func TestValidateECCreationSize(t *testing.T) {
	maxClusters := uint64(EcLvstoreMaxCreationSize) / EcLvstoreClusterSize
	for _, k := range []int{1, 2, 4, 8} {
		for _, strip := range []int{4, 64, 1024} {
			maxSize := MaxECVolumeSizeForCreation(k, strip)
			if err := ValidateECCreationSize(maxSize, k, strip); err != nil {
				t.Errorf("k=%d strip=%dKiB: max %d rejected: %v", k, strip, maxSize, err)
			}
			if err := ValidateECCreationSize(maxSize+1, k, strip); err == nil {
				t.Errorf("k=%d strip=%dKiB: max+1 %d accepted", k, strip, maxSize+1)
			}
			if got := EcUsableSize(maxSize, k, strip) / EcLvstoreClusterSize; got > maxClusters {
				t.Errorf("k=%d strip=%dKiB: usable %d clusters exceeds limit %d", k, strip, got, maxClusters)
			}
			// The usable size must always cover the volume, and the max must
			// sit below the raw cap (the admission gap this helper closes).
			if usable := EcUsableSize(maxSize, k, strip); usable < uint64(maxSize) {
				t.Errorf("k=%d strip=%dKiB: usable %d < volume %d", k, strip, usable, maxSize)
			}
			if maxSize >= EcLvstoreMaxCreationSize {
				t.Errorf("k=%d strip=%dKiB: max %d not below the raw cap", k, strip, maxSize)
			}
		}
	}
}
