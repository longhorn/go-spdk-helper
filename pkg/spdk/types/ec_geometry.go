package types

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
// known values instead of SPDK defaults. Ratio 100 gives one 4 KiB md page per
// cluster, enough metadata for the head lvol plus about 500 snapshots.
const (
	EcLvstoreClusterSize            = 4 * 1024 * 1024
	EcLvstoreMdPagesPerClusterRatio = 100

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

// lvstoreMetadataBytesFor returns the metadata SPDK bs_init carves out of a
// device of deviceBytes, in whole clusters. It rounds up where SPDK rounds
// down, so the result is an upper bound.
func lvstoreMetadataBytesFor(deviceBytes uint64) uint64 {
	clusters := (deviceBytes + EcLvstoreClusterSize - 1) / EcLvstoreClusterSize
	mdLen := (clusters*EcLvstoreMdPagesPerClusterRatio + 99) / 100
	mdPages := 1 + lvstoreMaskPages(mdLen) +
		max(lvstoreMaskPages(clusters), lvstoreMaskPages(mdLen)) +
		lvstoreMaskPages(mdLen) + mdLen
	mdClusters := (mdPages + ecLvstorePagesPerCluster - 1) / ecLvstorePagesPerCluster
	return mdClusters * EcLvstoreClusterSize
}

// lvstoreMetadataBytes bounds the metadata a lvstore holding dataBytes carves
// out. The metadata itself grows the device, so compute it once more on the
// grown size and keep one spare cluster for the remainder.
func lvstoreMetadataBytes(dataBytes uint64) uint64 {
	meta := lvstoreMetadataBytesFor(dataBytes)
	meta = lvstoreMetadataBytesFor(dataBytes + meta)
	return meta + EcLvstoreClusterSize
}

// ComputeShardSize returns the size in bytes of each EC shard lvol so that the
// lvstore built on the EC bdev can hold the full volume spec. The manager
// schedules with this; the engine checks it against the real lvstore at
// creation.
func ComputeShardSize(volumeSize int64, k, stripSizeKB int) int64 {
	if k <= 0 {
		return volumeSize
	}
	backing := uint64(volumeSize) + lvstoreMetadataBytes(uint64(volumeSize))
	perDisk := (backing+uint64(k)-1)/uint64(k) + EcFrontReservationBytes(uint32(stripSizeKB))
	return int64((perDisk + ecShardSizeAlignment - 1) / ecShardSizeAlignment * ecShardSizeAlignment)
}
