package client

import (
	"encoding/json"
	"fmt"

	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
)

// BdevEcCreate creates an EC bdev backed by k+m base bdevs.
//
//	"name": Required. Name for the new EC bdev.
//	"dataChunks": Required. Number of data chunks per stripe (Reed-Solomon k).
//	"parityChunks": Required. Number of parity chunks per stripe (Reed-Solomon m).
//	"stripSizeKB": Required. Chunk size in KiB (e.g. 64).
//	"baseBdevs": Required. Ordered list of k+m base bdev names.
//	"salvageRequested": Optional. When true, SPDK refuses to fresh-zero a
//	    torn on-disk unmapped bitmap and surfaces the failure so the
//	    operator can decide. Set on operator-driven recovery; leave false
//	    on normal create.
func (c *Client) BdevEcCreate(name string, dataChunks, parityChunks, stripSizeKB uint32, baseBdevs []string, salvageRequested bool) (bdevName string, err error) {
	req := spdktypes.BdevEcCreateRequest{
		Name:             name,
		DataChunks:       dataChunks,
		ParityChunks:     parityChunks,
		StripSizeKB:      stripSizeKB,
		BaseBdevs:        baseBdevs,
		SalvageRequested: salvageRequested,
	}

	cmdOutput, err := c.jsonCli.SendCommand("bdev_ec_create", req)
	if err != nil {
		return "", err
	}

	// bdev_ec_create returns true on success, not the bdev name.
	var created bool
	if err := json.Unmarshal(cmdOutput, &created); err != nil {
		return "", err
	}
	if !created {
		return "", fmt.Errorf("bdev_ec_create returned false for %s", name)
	}
	return name, nil
}

// BdevEcDelete deletes an EC bdev by name.
func (c *Client) BdevEcDelete(name string) (deleted bool, err error) {
	req := spdktypes.BdevEcDeleteRequest{
		Name: name,
	}

	cmdOutput, err := c.jsonCli.SendCommand("bdev_ec_delete", req)
	if err != nil {
		return false, err
	}

	return deleted, json.Unmarshal(cmdOutput, &deleted)
}

// BdevEcGetBdevs lists EC bdevs. If name is empty, all EC bdevs are returned.
// The State field of each BdevEcInfo is derived from FailedCount and Offline
// after unmarshaling, since SPDK does not return it directly.
func (c *Client) BdevEcGetBdevs(name string) (bdevEcInfoList []spdktypes.BdevEcInfo, err error) {
	req := spdktypes.BdevEcGetBdevsRequest{
		Name: name,
	}

	cmdOutput, err := c.jsonCli.SendCommand("bdev_ec_get_bdevs", req)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(cmdOutput, &bdevEcInfoList); err != nil {
		return nil, err
	}

	for i := range bdevEcInfoList {
		bdevEcInfoList[i].State = bdevEcState(&bdevEcInfoList[i])
		if bdevEcInfoList[i].RebuildInProgress && bdevEcInfoList[i].RebuildProgress != nil {
			if bdevEcInfoList[i].RebuildProgress.PercentComplete == 100 {
				bdevEcInfoList[i].RebuildProgress.RebuildState = spdktypes.BdevEcRebuildStateDone
			} else {
				bdevEcInfoList[i].RebuildProgress.RebuildState = spdktypes.BdevEcRebuildStateRunning
			}
		}
	}

	return bdevEcInfoList, nil
}

// bdevEcState derives the BdevEcState from the FailedCount and Offline fields.
func bdevEcState(info *spdktypes.BdevEcInfo) spdktypes.BdevEcState {
	if info.Offline {
		return spdktypes.BdevEcStateOffline
	}
	if info.FailedCount > 0 {
		return spdktypes.BdevEcStateDegraded
	}
	return spdktypes.BdevEcStateOnline
}
