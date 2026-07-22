package types

import "math"

// SPDK bdev_ec on-disk metadata constants, mirrored from longhorn/spdk. The front
// reservation recomputes ec_compute_geometry, so these must stay in sync with SPDK.
const (
	ecWibHeaderBytes     = 24   // sizeof(struct ec_wib_header)
	ecBitmapHeaderBytes  = 40   // sizeof(struct ec_bitmap_header)
	ecCRCBytes           = 4    // uint32 crc trailer
	ecWibRegionStripes   = 1024 // EC_WIB_REGION_STRIPES
	ecWibStrips          = 2    // double-buffered WIB, reserved on every disk
	ecCommitRecordStrips = 2    // commit record (stamp): one strip per double-buffer copy
)

// EcFrontReservationStrips returns the strips SPDK bdev_ec keeps at the front of each base
// disk for metadata (the double-buffered unmapped bitmap, the commit record, and the WIB)
// before user data, recomputing SPDK's ec_compute_geometry. Assumes a valid strip size
// (power of two, 4..1024 KiB); a smaller value underflows the WIB-payload subtraction.
func EcFrontReservationStrips(stripSizeKB uint32) uint64 {
	stripBytes := uint64(stripSizeKB) * 1024
	wibPayload := stripBytes - ecWibHeaderBytes - ecCRCBytes
	maxStripes := (wibPayload / 8) * 64 * ecWibRegionStripes
	blobBytes := ecBitmapHeaderBytes + ((maxStripes+63)/64)*8
	slotStrips := (blobBytes + ecCRCBytes + stripBytes - 1) / stripBytes
	return slotStrips*2 + ecWibStrips + ecCommitRecordStrips
}

// EcFrontReservationBytes returns the per-disk front reservation, in bytes, to add on
// top of ceil(volumeSize/k) when sizing an EC shard lvol.
func EcFrontReservationBytes(stripSizeKB uint32) uint64 {
	return EcFrontReservationStrips(stripSizeKB) * uint64(stripSizeKB) * 1024
}

// Lvstore parameters the engine pins at creation, so shard sizing works from
// known values instead of SPDK defaults.
//
// SPDK sizes the lvstore metadata region once, at creation. Growing the
// lvstore later does not extend it; see bs_load_try_to_grow:
// https://github.com/longhorn/spdk/blob/8d8b790728eb210141aab70fba0e30d5ba291ed3/lib/blob/blobstore.c#L10762
//
// The md-pages ratio therefore sets both the metadata budget and the in-place
// expansion ceiling:
//
//	max growable size = (ratio / 100) x creation size
//
// Ratio 100 is one 4 KiB md page per cluster: enough for the head lvol and
// snapshots, but no room to grow. Multiplying by EcLvstoreMaxGrowthFactor
// lets the lvstore grow to 10x its creation size, at a metadata cost of
// about 1% of the creation size. Growing past the ceiling requires a
// shard-group rebuild.
const (
	EcLvstoreClusterSize = 4 * 1024 * 1024

	// ecLvstoreMdRatioBase is SPDK's scale for num_md_pages_per_cluster_ratio:
	// ratio 100 means one metadata page per cluster.
	ecLvstoreMdRatioBase = 100

	// EcLvstoreMaxGrowthFactor is how far a lvstore can grow in place, as a
	// multiple of its creation size.
	EcLvstoreMaxGrowthFactor = 10

	EcLvstoreMdPagesPerClusterRatio = ecLvstoreMdRatioBase * EcLvstoreMaxGrowthFactor

	// EcLvstoreMaxCreationSize is the largest device SPDK can size lvstore
	// metadata for at the pinned ratio. setup_lvs_opts (lib/lvol/lvol.c in
	// longhorn/spdk) rejects lvstore creation with -EINVAL when the metadata
	// page count, ratio x clusters / 100, exceeds UINT32_MAX.
	// Callers must reject creating an EC volume above this size.
	EcLvstoreMaxCreationSize = (math.MaxUint32 / (EcLvstoreMdPagesPerClusterRatio / ecLvstoreMdRatioBase)) * EcLvstoreClusterSize

	ecLvstoreMdPageSize      = 4096
	ecLvstorePagesPerCluster = EcLvstoreClusterSize / ecLvstoreMdPageSize
	ecBsMdMaskHeaderBytes    = 8               // sizeof(struct spdk_bs_md_mask), padded
	ecShardSizeAlignment     = 2 * 1024 * 1024 // matches longhorn-manager util.SizeAlignment
)

// lvstoreMaskPages returns the md pages one blobstore allocation mask occupies:
// the mask header plus 1 bit per tracked item, in whole pages.
func lvstoreMaskPages(bits uint64) uint64 {
	return (ecBsMdMaskHeaderBytes + (bits+7)/8 + ecLvstoreMdPageSize - 1) / ecLvstoreMdPageSize
}

// lvstoreMetadataBytesFor returns the metadata SPDK carves out of a device of
// deviceBytes, in whole clusters. It mirrors spdk_bs_init: one super block
// page, the used_pages, used_clusters, and used_blobids masks, then the
// md_len metadata pages. The used_clusters mask region is sized for the
// larger of clusters and md_len. It rounds up where SPDK rounds down, so the
// result is an upper bound. See
// https://github.com/longhorn/spdk/blob/8d8b790728eb210141aab70fba0e30d5ba291ed3/lib/blob/blobstore.c#L5847-L5896
func lvstoreMetadataBytesFor(deviceBytes uint64) uint64 {
	clusters := (deviceBytes + EcLvstoreClusterSize - 1) / EcLvstoreClusterSize
	mdLen := (clusters*EcLvstoreMdPagesPerClusterRatio + ecLvstoreMdRatioBase - 1) / ecLvstoreMdRatioBase
	mdPages := 1 + lvstoreMaskPages(mdLen) + // 1 page for the super block
		max(lvstoreMaskPages(clusters), lvstoreMaskPages(mdLen)) +
		lvstoreMaskPages(mdLen) + mdLen
	mdClusters := (mdPages + ecLvstorePagesPerCluster - 1) / ecLvstorePagesPerCluster
	return mdClusters * EcLvstoreClusterSize
}

// lvstoreMetadataBytes bounds the metadata a lvstore holding dataBytes carves
// out. The metadata itself takes device space, so iterate until the estimate
// stops growing, then keep one spare cluster.
func lvstoreMetadataBytes(dataBytes uint64) uint64 {
	meta := lvstoreMetadataBytesFor(dataBytes)
	for {
		next := lvstoreMetadataBytesFor(dataBytes + meta)
		if next <= meta {
			break
		}
		meta = next
	}
	return meta + EcLvstoreClusterSize
}

// ComputeShardSize returns the size in bytes of each EC shard lvol so the
// lvstore built on the EC bdev can hold the full volume spec. The manager
// schedules with this; the engine checks it against the real lvstore at
// creation.
//
// stripSizeKB must be a valid strip size (see EcFrontReservationStrips).
// Non-positive volumeSize or k returns volumeSize unchanged.
func ComputeShardSize(volumeSize int64, k, stripSizeKB int) int64 {
	if volumeSize <= 0 || k <= 0 {
		return volumeSize
	}
	backing := uint64(volumeSize) + lvstoreMetadataBytes(uint64(volumeSize))
	perDisk := (backing+uint64(k)-1)/uint64(k) + EcFrontReservationBytes(uint32(stripSizeKB))
	return int64((perDisk + ecShardSizeAlignment - 1) / ecShardSizeAlignment * ecShardSizeAlignment)
}
