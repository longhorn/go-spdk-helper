package client

import (
	"net"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"

	"github.com/longhorn/go-spdk-helper/pkg/jsonrpc"

	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
	spdkutil "github.com/longhorn/go-spdk-helper/pkg/util"
)

// sharedTransportIobufCacheSize returns per poll group iobuf cache sizes that
// reserve a quarter of the small and large pools across all poll groups.
func sharedTransportIobufCacheSize(opts *spdktypes.IobufOptions, pollGroupCount int) (smallCacheSize, largeCacheSize uint32) {
	if pollGroupCount < 1 {
		pollGroupCount = 1
	}
	smallCacheSize = uint32(max(opts.SmallPoolCount/4/uint64(pollGroupCount), 1))
	largeCacheSize = uint32(max(opts.LargePoolCount/4/uint64(pollGroupCount), 1))
	return smallCacheSize, largeCacheSize
}

// AddDevice adds a device with the given device path, name, and cluster size.
func (c *Client) AddDevice(devicePath, name string, clusterSize uint32) (bdevAioName, lvsName, lvsUUID string, err error) {
	// Use the file name as aio name and lvs name if name is not specified.
	if name == "" {
		name = filepath.Base(devicePath)
	}

	if _, err := c.BdevAioCreate(devicePath, name, 4096); err != nil {
		return "", "", "", err
	}

	lvsList, err := c.BdevLvolGetLvstore("", "")
	if err != nil {
		return "", "", "", err
	}
	lvsCreated := false
	for _, lvsInfo := range lvsList {
		if lvsInfo.BaseBdev == name {
			lvsCreated = true
			lvsUUID = lvsInfo.UUID
			break
		}
	}
	if !lvsCreated {
		if lvsUUID, err = c.BdevLvolCreateLvstore(name, name, clusterSize); err != nil {
			return "", "", "", err
		}
	}

	return name, name, lvsUUID, nil
}

// DeleteDevice deletes the device with the given bdevAioName and lvsName.
func (c *Client) DeleteDevice(bdevAioName, lvsName string) (err error) {
	if _, err := c.BdevLvolDeleteLvstore(lvsName, ""); err != nil {
		return err
	}

	if _, err := c.BdevAioDelete(bdevAioName); err != nil {
		return err
	}

	return nil
}

// DetectAddressFamily returns the NVMe address family for the given IP.
// Exported so downstream repos (longhorn-spdk-engine) can reuse it.
func DetectAddressFamily(ip string) spdktypes.NvmeAddressFamily {
	if ip == "" {
		return spdktypes.NvmeAddressFamilyIPv4
	}

	normalized := spdkutil.NormalizeNvmeAddr(ip)
	parsedIP := net.ParseIP(normalized)
	if parsedIP == nil {
		logrus.Warnf("Failed to parse IP %q, defaulting to IPv4", ip)
		return spdktypes.NvmeAddressFamilyIPv4
	}

	if parsedIP.To4() == nil {
		return spdktypes.NvmeAddressFamilyIPv6
	}
	return spdktypes.NvmeAddressFamilyIPv4
}

// nvmfTransportMu serializes the check-and-create of the NVMe-oF transports.
// spdk_tgt only rejects a duplicate transport after the first one has been added to every poll group,
// so concurrent nvmf_create_transport calls can all succeed and leave several transports. Each transport
// reserves half of the iobuf pools for its per-poll-group caches, so a second one makes the iobuf channel
// creation of the bdev layer fail, which later crashes spdk_tgt.
var nvmfTransportMu sync.Mutex

// EnsureNvmfTransport creates the TCP transport if spdk_tgt has no TCP transport yet.
// It is safe to call concurrently from many goroutines of the same process.
func (c *Client) EnsureNvmfTransport() error {
	return c.EnsureNvmfTransportType(spdktypes.NvmeTransportTypeTCP)
}

// EnsureNvmfTransportType creates an NVMe-oF transport of the requested type if spdk_tgt
// does not have one of that type yet. Unlike a "create only if the transport list is empty"
// check, this lets an RDMA transport be added to a target that already has a TCP transport
// (and vice versa). It is safe to call concurrently from many goroutines of the same process.
func (c *Client) EnsureNvmfTransportType(trtype spdktypes.NvmeTransportType) error {
	// hasTransport also returns the number of transports of any type, which decides how the
	// iobuf caches of a new transport are sized.
	hasTransport := func() (bool, int, error) {
		nvmfTransportList, err := c.NvmfGetTransports("", "")
		if err != nil {
			return false, 0, err
		}
		for _, t := range nvmfTransportList {
			if strings.EqualFold(string(t.Trtype), string(trtype)) {
				return true, len(nvmfTransportList), nil
			}
		}
		return false, len(nvmfTransportList), nil
	}

	if exists, _, err := hasTransport(); err != nil || exists {
		return err
	}

	nvmfTransportMu.Lock()
	defer nvmfTransportMu.Unlock()

	// Check again: nvmf_create_transport responds only after the transport is visible in
	// nvmf_get_transports, so a caller that waited for the lock sees the transport created by another one.
	exists, transportCount, err := hasTransport()
	if err != nil || exists {
		return err
	}

	if err := c.createNvmfTransport(trtype, transportCount); err != nil {
		if jsonrpc.IsJSONRPCRespErrorTransportTypeAlreadyExists(err) {
			return nil
		}
		// A duplicate that spdk_tgt only detects after the transport setup is reported as a generic
		// "Failed to add transport to tgt.(-17)" error instead, so verify by listing the transports.
		if exists, _, listErr := hasTransport(); listErr == nil && exists {
			return nil
		}
		return err
	}
	return nil
}

// createNvmfTransport creates a transport of the given type in a target that has existingCount transports.
//
// By default SPDK lets the poll groups of each transport reserve half of the
// shared iobuf pools, so a second transport in the same target would leave no
// buffers for I/O and its own poll group caches would be partially populated.
// A transport added next to an existing one, and the RDMA transport, which is
// always used next to the TCP frontend on engine nodes, are therefore created
// with explicit per poll group caches sized to a quarter of the pools. TCP and
// RDMA together then reserve the same half of the pools as TCP alone.
func (c *Client) createNvmfTransport(trtype spdktypes.NvmeTransportType, existingCount int) error {
	if existingCount == 0 && !strings.EqualFold(string(trtype), string(spdktypes.NvmeTransportTypeRDMA)) {
		logrus.Infof("Creating transport with type %v", trtype)
		_, err := c.NvmfCreateTransport(trtype)
		return err
	}

	iobufOpts, err := c.IobufGetOptions()
	if err != nil {
		return errors.Wrap(err, "failed to get iobuf options")
	}
	pollGroupCount, err := c.NvmfGetPollGroupCount()
	if err != nil {
		return errors.Wrap(err, "failed to get NVMe-oF poll group count")
	}
	smallCacheSize, largeCacheSize := sharedTransportIobufCacheSize(iobufOpts, pollGroupCount)

	logrus.Infof("Creating transport with type %v and iobuf small/large cache size per poll group %v/%v, %v existing transport(s)",
		trtype, smallCacheSize, largeCacheSize, existingCount)
	_, err = c.NvmfCreateTransportWithIobufCacheSize(trtype, smallCacheSize, largeCacheSize)
	return err
}

// StartExposeBdev exposes the bdev with the given nqn, bdevName, nguid, ip, port,
// and NVMe-oF transport type (trtype). Pass spdktypes.NvmeTransportTypeTCP for the
// historical TCP behavior or NvmeTransportTypeRDMA for RoCEv2/InfiniBand.
// If allowedHostNQNs is not empty, only those hosts can connect and the subsystem is
// hidden from other hosts' discovery log pages.
func (c *Client) StartExposeBdev(nqn, bdevName, nguid, ip, port string, trtype spdktypes.NvmeTransportType, allowedHostNQNs ...string) error {
	ip = spdkutil.NormalizeNvmeAddr(ip)

	logrus.Infof("Exposing bdev with nqn %v, bdevName %v, nguid %v, ip %v, port %v, trtype %v, allowedHostNQNs %v", nqn, bdevName, nguid, ip, port, trtype, allowedHostNQNs)

	if err := c.EnsureNvmfTransportType(trtype); err != nil {
		return err
	}

	logrus.Infof("Creating subsystem with nqn %v", nqn)
	if _, err := c.NvmfCreateSubsystem(nqn, len(allowedHostNQNs) == 0); err != nil {
		return err
	}

	for _, hostNQN := range allowedHostNQNs {
		logrus.Infof("Adding allowed host %v to subsystem with nqn %v", hostNQN, nqn)
		if _, err := c.NvmfSubsystemAddHost(nqn, hostNQN); err != nil {
			return err
		}
	}

	logrus.Infof("Adding NVMe namespace with bdev name %v and nguid %v to subsystem with nqn %v", bdevName, nguid, nqn)
	if _, err := c.NvmfSubsystemAddNs(nqn, bdevName, nguid); err != nil {
		return err
	}

	adrfam := DetectAddressFamily(ip)

	logrus.Infof("Adding listener with transport address %v, transport service id %v, transport type %v, address family %v to subsystem with nqn %v", ip, port, trtype, adrfam, nqn)
	if _, err := c.NvmfSubsystemAddListener(nqn, ip, port, trtype, adrfam); err != nil {
		return err
	}

	return nil
}

// StartExposeBdevWithANAState exposes the bdev with the given nqn, bdevName,
// nguid, nsUUID, ip, port, NVMe-oF transport type (trtype), initial ANA state,
// and optional CNTLID range. Pass spdktypes.NvmeTransportTypeTCP for the
// historical TCP behavior or NvmeTransportTypeRDMA for RoCEv2/InfiniBand.
// nsUUID sets a stable namespace UUID so the Linux kernel can aggregate
// controllers into the same NVMe multipath group. minCntlid/maxCntlid assign
// a unique controller-ID range per engine to avoid "Duplicate cntlid" errors
// when multiple targets share one subsystem NQN. Pass 0 for defaults.
func (c *Client) StartExposeBdevWithANAState(nqn, bdevName, nguid, nsUUID, ip, port string, trtype spdktypes.NvmeTransportType, anaState spdktypes.NvmfSubsystemListenerAnaState, minCntlid, maxCntlid uint16) error {
	ip = spdkutil.NormalizeNvmeAddr(ip)

	logrus.Infof("Exposing bdev with nqn %v, bdevName %v, nguid %v, nsUUID %v, ip %v, port %v, trtype %v, anaState %v, minCntlid %v, maxCntlid %v",
		nqn, bdevName, nguid, nsUUID, ip, port, trtype, anaState, minCntlid, maxCntlid)

	if err := c.EnsureNvmfTransportType(trtype); err != nil {
		return err
	}

	logrus.Infof("Creating subsystem with nqn %v, minCntlid %v, maxCntlid %v", nqn, minCntlid, maxCntlid)
	if _, err := c.NvmfCreateSubsystemWithCntlid(nqn, minCntlid, maxCntlid); err != nil {
		return err
	}

	logrus.Infof("Adding NVMe namespace with bdev name %v, nguid %v, uuid %v to subsystem with nqn %v", bdevName, nguid, nsUUID, nqn)
	if _, err := c.NvmfSubsystemAddNsWithUUID(nqn, bdevName, nguid, nsUUID); err != nil {
		return err
	}

	adrfam := DetectAddressFamily(ip)

	logrus.Infof("Adding listener with transport address %v, transport service id %v, transport type %v, address family %v to subsystem with nqn %v", ip, port, trtype, adrfam, nqn)
	if _, err := c.NvmfSubsystemAddListener(nqn, ip, port, trtype, adrfam); err != nil {
		return err
	}

	logrus.Infof("Setting listener ANA state to %v for subsystem with nqn %v", anaState, nqn)
	if _, err := c.NvmfSubsystemListenerSetANAState(nqn, ip, port, trtype,
		adrfam, anaState, spdktypes.DefaultNvmfANAGroupID); err != nil {
		return err
	}

	return nil
}

// StopExposeBdev stops exposing the bdev with the given nqn.
func (c *Client) StopExposeBdev(nqn string) error {
	logrus.Infof("Stopping exposing bdev with nqn %v", nqn)

	var subsystem *spdktypes.NvmfSubsystem
	subsystemList, err := c.NvmfGetSubsystems("", "")
	if err != nil {
		return err
	}
	for _, s := range subsystemList {
		if s.Nqn != nqn {
			continue
		}
		subsystem = &s
		break
	}
	if subsystem == nil {
		return nil
	}

	listenerList, err := c.NvmfSubsystemGetListeners(nqn, "")
	if err != nil {
		return err
	}
	for _, l := range listenerList {
		logrus.Infof("Removing listener with transport address %v, transport service id %v, transport type %v, address family %v", l.Address.Traddr, l.Address.Trsvcid, l.Address.Trtype, l.Address.Adrfam)
		if _, err := c.NvmfSubsystemRemoveListener(nqn, l.Address.Traddr, l.Address.Trsvcid, l.Address.Trtype, l.Address.Adrfam); err != nil {
			return err
		}
	}

	for _, ns := range subsystem.Namespaces {
		logrus.Infof("Removing namespace with NSID %v", ns.Nsid)
		if _, err := c.NvmfSubsystemRemoveNs(nqn, ns.Nsid); err != nil {
			return err
		}
	}

	logrus.Infof("Deleting subsystem with nqn %v", nqn)
	if _, err := c.NvmfDeleteSubsystem(nqn, ""); err != nil {
		return err
	}

	return nil
}
