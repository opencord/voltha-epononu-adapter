/*
 * Copyright 2020-present Open Networking Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

//Package adaptercoreonu provides the utility for onu devices, flows and statistics
package adaptercoreonu

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"


	"github.com/opencord/voltha-lib-go/v3/pkg/log"
	vc "github.com/opencord/voltha-protos/v3/go/common"
	of "github.com/opencord/voltha-protos/v3/go/openflow_13"
	"github.com/opencord/voltha-protos/v3/go/voltha"
)

type uniPortType uint8

// UniPPTP Interface type - re-use values from G.988 TP type definition (directly used in OMCI!)
const (
	uniPPTP uniPortType = 1 // relates to PPTP
	uniVEIP uniPortType = 11 // relates to VEIP
)

//onuUniPort structure holds information about the ONU attached Uni Ports
type onuUniPort struct {
	enabled    bool
	name       string
	portNo     uint32
	portType   uniPortType
	ofpPortNo  string
	uniID      uint8
	macBpNo    uint8
	entityID   uint16
	adminState vc.AdminState_Types
	operState  vc.OperStatus_Types
	pPort      *voltha.Port
}

//newOnuUniPort returns a new instance of a OnuUniPort
func newOnuUniPort(aUniID uint8, aPortNo uint32, aInstNo uint16,
	aPortType uniPortType) *onuUniPort {
	logger.Infow("init-onuUniPort", log.Fields{"uniID": aUniID,
		"portNo": aPortNo, "InstNo": aInstNo, "type": aPortType})
	var onuUniPort onuUniPort
	onuUniPort.enabled = false
	onuUniPort.name = "uni-" + strconv.FormatUint(uint64(aPortNo), 10)
	onuUniPort.portNo = aPortNo
	onuUniPort.portType = aPortType
	onuUniPort.ofpPortNo = onuUniPort.name
	onuUniPort.uniID = aUniID
	onuUniPort.macBpNo = aUniID + 1 //ensure >0 instanceNo
	onuUniPort.entityID = aInstNo
	onuUniPort.adminState = vc.AdminState_ENABLED //enabled per create
	onuUniPort.operState = vc.OperStatus_UNKNOWN
	onuUniPort.pPort = nil // to be set on create
	return &onuUniPort
}

//createVolthaPort creates the Voltha port based on ONU UNI Port and informs the core about it
func (oo *onuUniPort) createVolthaPort(apDeviceHandler *deviceHandler) error {
	logger.Debugw("creating-voltha-uni-port", log.Fields{
		"device-id": apDeviceHandler.device.Id, "portNo": oo.portNo})
	name := apDeviceHandler.device.SerialNumber + "-" + strconv.FormatUint(uint64(oo.macBpNo), 10)

	var macOctets [6]uint8
	macOctets[5] = 0x08
	macOctets[4] = uint8(apDeviceHandler.ponPortNumber >> 8)
	macOctets[3] = uint8(apDeviceHandler.ponPortNumber)
	macOctets[2] = uint8(oo.portNo >> 16)
	macOctets[1] = uint8(oo.portNo >> 8)
	macOctets[0] = uint8(oo.portNo)
	onuEntry := apDeviceHandler.getOnuDeviceEntry(false)
	if onuEntry == nil {
		logger.Debugw("not-found-onu-device-entry", log.Fields{"error": "not found"})
	} else {
		if len(onuEntry.macAddress) == 12 {
			for idx := 0; idx < 6; idx++ {
				if v, err := strconv.ParseInt(onuEntry.macAddress[idx*2:(idx*2)+2], 16, 8); err == nil {
					macOctets[5-idx] = uint8(v)
				}
			}
		}
	}
	hwAddr := genMacFromOctets(macOctets)
	ofHwAddr := macAddressToUint32Array(hwAddr)
	capacity := uint32(of.OfpPortFeatures_OFPPF_1GB_FD | of.OfpPortFeatures_OFPPF_FIBER)
	ofUniPortState := of.OfpPortState_OFPPS_LINK_DOWN
	/* as the VOLTHA port create is only called directly after Uni Port create
	   the OfPortOperState is always Down
	   Note: this way the OfPortOperState won't ever change (directly in adapter)
	   maybe that was already always the case, but looks a bit weird - to be kept in mind ...
		if pUniPort.operState == vc.OperStatus_ACTIVE {
			ofUniPortState = of.OfpPortState_OFPPS_LIVE
		}
	*/
	logger.Debugw("ofPort values", log.Fields{
		"forUniPortName": oo.name, "forMacBase": hwAddr,
		"name": name, "hwAddr": ofHwAddr, "OperState": ofUniPortState})

	pUniPort := &voltha.Port{
		PortNo:     oo.portNo,
		Label:      oo.name,
		Type:       voltha.Port_ETHERNET_UNI,
		AdminState: oo.adminState,
		OperStatus: oo.operState,
		// obviously empty peer setting
		OfpPort: &of.OfpPort{
			Name:       name,
			HwAddr:     ofHwAddr,
			Config:     0,
			State:      uint32(ofUniPortState),
			Curr:       capacity,
			Advertised: capacity,
			Peer:       capacity,
			CurrSpeed:  uint32(of.OfpPortFeatures_OFPPF_1GB_FD),
			MaxSpeed:   uint32(of.OfpPortFeatures_OFPPF_1GB_FD),
		},
	}
	if pUniPort != nil {
		if err := apDeviceHandler.coreProxy.PortCreated(context.TODO(),
			apDeviceHandler.deviceID, pUniPort); err != nil {
			logger.Fatalf("adding-uni-port: create-VOLTHA-Port-failed-%s", err)
			return err
		}
		logger.Infow("Voltha onuUniPort-added", log.Fields{
			"device-id": apDeviceHandler.device.Id, "PortNo": oo.portNo})
		oo.pPort = pUniPort
		oo.operState = vc.OperStatus_DISCOVERED
	} else {
		logger.Warnw("could not create Voltha UniPort", log.Fields{
			"device-id": apDeviceHandler.device.Id, "PortNo": oo.portNo})
		return errors.New("create Voltha UniPort failed")
	}
	return nil
}

//setOperState modifies OperState of the the UniPort
func (oo *onuUniPort) setOperState(aNewOperState vc.OperStatus_Types) {
	oo.operState = aNewOperState
}

// uni port related utility functions (so far only used here)
func genMacFromOctets(aOctets [6]uint8) string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		aOctets[5], aOctets[4], aOctets[3],
		aOctets[2], aOctets[1], aOctets[0])
}

//copied from OLT Adapter: unify centrally ?
func macAddressToUint32Array(mac string) []uint32 {
	slist := strings.Split(mac, ":")
	result := make([]uint32, len(slist))
	var err error
	var tmp int64
	for index, val := range slist {
		if tmp, err = strconv.ParseInt(val, 16, 32); err != nil {
			return []uint32{1, 2, 3, 4, 5, 6}
		}
		result[index] = uint32(tmp)
	}
	return result
}
