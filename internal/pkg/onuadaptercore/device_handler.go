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
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/looplab/fsm"
	me "github.com/opencord/omci-lib-go/generated"
	"github.com/opencord/voltha-lib-go/v3/pkg/adapters/adapterif"
	"github.com/opencord/voltha-lib-go/v3/pkg/db"
	flow "github.com/opencord/voltha-lib-go/v3/pkg/flows"
	"github.com/opencord/voltha-lib-go/v3/pkg/log"
	vc "github.com/opencord/voltha-protos/v3/go/common"
	ic "github.com/opencord/voltha-protos/v3/go/inter_container"
	"github.com/opencord/voltha-protos/v3/go/openflow_13"
	of "github.com/opencord/voltha-protos/v3/go/openflow_13"
	ofp "github.com/opencord/voltha-protos/v3/go/openflow_13"
	oop "github.com/opencord/voltha-protos/v3/go/openolt"
	"github.com/opencord/voltha-protos/v3/go/voltha"
)

/*
// Constants for number of retries and for timeout
const (
	MaxRetry       = 10
	MaxTimeOutInMs = 500
)
*/

const (
	devEvDeviceInit       = "devEvDeviceInit"
	devEvGrpcConnected    = "devEvGrpcConnected"
	devEvGrpcDisconnected = "devEvGrpcDisconnected"
	devEvDeviceUpInd      = "devEvDeviceUpInd"
	devEvDeviceDownInd    = "devEvDeviceDownInd"
)
const (
	devStNull      = "devStNull"
	devStDown      = "devStDown"
	devStInit      = "devStInit"
	devStConnected = "devStConnected"
	devStUp        = "devStUp"
)

//Event category and subcategory definitions - same as defiend for OLT in eventmgr.go  - should be done more centrally
const (
	pon = voltha.EventSubCategory_PON
	equipment = voltha.EventCategory_EQUIPMENT
)

const (
	cEventObjectType = "ONU"
)
const (
	cOnuActivatedEvent = "ONU_ACTIVATED"
)

//deviceHandler will interact with the ONU ? device.
type deviceHandler struct {
	deviceID         string
	DeviceType       string
	adminState       string
	device           *voltha.Device
	logicalDeviceID  string
	ProxyAddressID   string
	ProxyAddressType string
	parentID         string
	ponPortNumber    uint32

	coreProxy    adapterif.CoreProxy
	AdapterProxy adapterif.AdapterProxy
	EventProxy   adapterif.EventProxy

	pOpenOnuAc      *OpenONUAC
	pDeviceStateFsm *fsm.FSM
	deviceEntrySet  chan bool //channel for DeviceEntry set event
	pOnuOmciDevice  *OnuDeviceEntry
	pOnuTP          *onuUniTechProf
	exitChannel     chan int
	lockDevice      sync.RWMutex
	pOnuIndication  *oop.OnuIndication
	deviceReason    string
	pLockStateFsm   *lockStateFsm
	pUnlockStateFsm *lockStateFsm


	stopCollector       chan bool
	stopHeartbeatCheck  chan bool
	activePorts         sync.Map
	uniEntityMap        map[uint32]*onuUniPort
	UniVlanConfigFsmMap map[uint8]*UniVlanConfigFsm
	reconciling         bool
}

//newDeviceHandler creates a new device handler
func newDeviceHandler(cp adapterif.CoreProxy, ap adapterif.AdapterProxy, ep adapterif.EventProxy, device *voltha.Device, adapter *OpenONUAC) *deviceHandler {
	var dh deviceHandler
	dh.coreProxy = cp
	dh.AdapterProxy = ap
	dh.EventProxy = ep
	cloned := (proto.Clone(device)).(*voltha.Device)
	dh.deviceID = cloned.Id
	dh.DeviceType = cloned.Type
	dh.adminState = "up"
	dh.device = cloned
	dh.pOpenOnuAc = adapter
	dh.exitChannel = make(chan int, 1)
	dh.lockDevice = sync.RWMutex{}
	dh.deviceEntrySet = make(chan bool, 1)
	dh.stopCollector = make(chan bool, 2)
	dh.stopHeartbeatCheck = make(chan bool, 2)
	dh.activePorts = sync.Map{}
	dh.uniEntityMap = make(map[uint32]*onuUniPort)
	dh.UniVlanConfigFsmMap = make(map[uint8]*UniVlanConfigFsm)
	dh.reconciling = false

	dh.pDeviceStateFsm = fsm.NewFSM(
		devStNull,
		fsm.Events{
			{Name: devEvDeviceInit, Src: []string{devStNull, devStDown}, Dst: devStInit},
			{Name: devEvGrpcConnected, Src: []string{devStInit}, Dst: devStConnected},
			{Name: devEvGrpcDisconnected, Src: []string{devStConnected, devStDown}, Dst: devStInit},
			{Name: devEvDeviceUpInd, Src: []string{devStConnected, devStDown}, Dst: devStUp},
			{Name: devEvDeviceDownInd, Src: []string{devStUp}, Dst: devStDown},
		},
		fsm.Callbacks{
			"before_event":                      func(e *fsm.Event) { dh.logStateChange(e) },
			("before_" + devEvDeviceInit):       func(e *fsm.Event) { dh.doStateInit(e) },
			("after_" + devEvDeviceInit):        func(e *fsm.Event) { dh.postInit(e) },
			("before_" + devEvGrpcConnected):    func(e *fsm.Event) { dh.doStateConnected(e) },
			("before_" + devEvGrpcDisconnected): func(e *fsm.Event) { dh.doStateInit(e) },
			("after_" + devEvGrpcDisconnected):  func(e *fsm.Event) { dh.postInit(e) },
			("before_" + devEvDeviceUpInd):      func(e *fsm.Event) { dh.doStateUp(e) },
			("before_" + devEvDeviceDownInd):    func(e *fsm.Event) { dh.doStateDown(e) },
		},
	)

	return &dh
}

// start save the device to the data model
func (dh *deviceHandler) start(ctx context.Context) {
	logger.Debugw("starting-device-handler", log.Fields{"device": dh.device, "device-id": dh.deviceID})
	logger.Debug("device-handler-started")
}

/*
// stop stops the device dh.  Not much to do for now
func (dh *deviceHandler) stop(ctx context.Context) {
	logger.Debug("stopping-device-handler")
	dh.exitChannel <- 1
}
*/

// ##########################################################################################
// deviceHandler methods that implement the adapters interface requests ##### begin #########

//adoptOrReconcileDevice adopts the OLT device
func (dh *deviceHandler) adoptOrReconcileDevice(ctx context.Context, device *voltha.Device) {
	logger.Debugw("Adopt_or_reconcile_device", log.Fields{"device-id": device.Id, "Address": device.GetHostAndPort()})

	logger.Debugw("Device FSM: ", log.Fields{"state": string(dh.pDeviceStateFsm.Current())})
	if dh.pDeviceStateFsm.Is(devStNull) {
		if err := dh.pDeviceStateFsm.Event(devEvDeviceInit); err != nil {
			logger.Errorw("Device FSM: Can't go to state DeviceInit", log.Fields{"err": err})
		}
		logger.Debugw("Device FSM: ", log.Fields{"state": string(dh.pDeviceStateFsm.Current())})
	} else {
		logger.Debugw("AdoptOrReconcileDevice: Agent/device init already done", log.Fields{"device-id": device.Id})
	}

}

func (dh *deviceHandler) processInterAdapterOMCIReqMessage(msg *ic.InterAdapterMessage) error {
	msgBody := msg.GetBody()
	omciMsg := &ic.InterAdapterOmciMessage{}
	if err := ptypes.UnmarshalAny(msgBody, omciMsg); err != nil {
		logger.Warnw("cannot-unmarshal-omci-msg-body", log.Fields{
			"device-id": dh.deviceID, "error": err})
		return err
	}

	logger.Debugw("inter-adapter-recv-omci", log.Fields{
		"device-id": dh.deviceID, "RxOmciMessage": hex.EncodeToString(omciMsg.Message)})
	pDevEntry := dh.getOnuDeviceEntry(true)
	if pDevEntry != nil {
		return pDevEntry.PDevOmciCC.receiveMessage(context.TODO(), omciMsg.Message)
	}
	logger.Errorw("No valid OnuDevice -aborting", log.Fields{"device-id": dh.deviceID})
	return errors.New("no valid OnuDevice")
}

func (dh *deviceHandler) processInterAdapterONUIndReqMessage(msg *ic.InterAdapterMessage) error {
	msgBody := msg.GetBody()
	onuIndication := &oop.OnuIndication{}
	if err := ptypes.UnmarshalAny(msgBody, onuIndication); err != nil {
		logger.Warnw("cannot-unmarshal-onu-indication-msg-body", log.Fields{
			"device-id": dh.deviceID, "error": err})
		return err
	}

	onuOperstate := onuIndication.GetOperState()
	logger.Debugw("inter-adapter-recv-onu-ind", log.Fields{"OnuId": onuIndication.GetOnuId(),
		"AdminState": onuIndication.GetAdminState(), "OperState": onuOperstate,
		"SNR": onuIndication.GetSerialNumber()})

	if onuOperstate == "up" {
		_ = dh.createInterface(onuIndication)
	} else if (onuOperstate == "down") || (onuOperstate == "unreachable") {
		_ = dh.updateInterface(onuIndication)
	} else {
		logger.Errorw("unknown-onu-indication operState", log.Fields{"OnuId": onuIndication.GetOnuId()})
		return errors.New("invalidOperState")
	}
	return nil
}

func (dh *deviceHandler) processInterAdapterTechProfileDownloadReqMessage(
	msg *ic.InterAdapterMessage) error {
	if dh.pOnuTP == nil {
		//should normally not happen ...
		logger.Warnw("onuTechProf instance not set up for DLMsg request - ignoring request",
			log.Fields{"device-id": dh.deviceID})
		return errors.New("techProfile DLMsg request while onuTechProf instance not setup")
	}
	if (dh.deviceReason == "stopping-openomci") || (dh.deviceReason == "omci-admin-lock") {
		// I've seen cases for this request, where the device was already stopped
		logger.Warnw("TechProf stopped: device-unreachable", log.Fields{"device-id": dh.deviceID})
		return errors.New("device-unreachable")
	}

	msgBody := msg.GetBody()
	techProfMsg := &ic.InterAdapterTechProfileDownloadMessage{}
	if err := ptypes.UnmarshalAny(msgBody, techProfMsg); err != nil {
		logger.Warnw("cannot-unmarshal-techprof-msg-body", log.Fields{
			"device-id": dh.deviceID, "error": err})
		return err
	}

	dh.pOnuTP.lockTpProcMutex()
	if bTpModify := dh.pOnuTP.updateOnuUniTpPath(techProfMsg.UniId, techProfMsg.Path); bTpModify {
		//	if there has been some change for some uni TechProfilePath
		//in order to allow concurrent calls to other dh instances we do not wait for execution here
		//but doing so we can not indicate problems to the caller (who does what with that then?)
		//by now we just assume straightforward successful execution
		//TODO!!! Generally: In this scheme it would be good to have some means to indicate
		//  possible problems to the caller later autonomously

		// deadline context to ensure completion of background routines waited for
		//20200721: 10s proved to be less in 8*8 ONU test on local vbox machine with debug, might be further adapted
		deadline := time.Now().Add(30 * time.Second) //allowed run time to finish before execution
		dctx, cancel := context.WithDeadline(context.Background(), deadline)

		dh.pOnuTP.resetProcessingErrorIndication()
		var wg sync.WaitGroup
		wg.Add(2) // for the 2 go routines to finish
		// attention: deadline completion check and wg.Done is to be done in both routines
		go dh.pOnuTP.configureUniTp(dctx, uint8(techProfMsg.UniId), techProfMsg.Path, &wg)
		go dh.pOnuTP.updateOnuTpPathKvStore(dctx, &wg)
		//the wait.. function is responsible for tpProcMutex.Unlock()
		err := dh.pOnuTP.waitForTpCompletion(cancel, &wg) //wait for background process to finish and collect their result
		return err
	}
	dh.pOnuTP.unlockTpProcMutex()
	return nil
}

func (dh *deviceHandler) processInterAdapterDeleteGemPortReqMessage(
	msg *ic.InterAdapterMessage) error {

	if dh.pOnuTP == nil {
		//should normally not happen ...
		logger.Warnw("onuTechProf instance not set up for DelGem request - ignoring request",
			log.Fields{"device-id": dh.deviceID})
		return errors.New("techProfile DelGem request while onuTechProf instance not setup")
	}

	msgBody := msg.GetBody()
	delGemPortMsg := &ic.InterAdapterDeleteGemPortMessage{}
	if err := ptypes.UnmarshalAny(msgBody, delGemPortMsg); err != nil {
		logger.Warnw("cannot-unmarshal-delete-gem-msg-body", log.Fields{
			"device-id": dh.deviceID, "error": err})
		return err
	}

	dh.pOnuTP.lockTpProcMutex()

	deadline := time.Now().Add(10 * time.Second) //allowed run time to finish before execution
	dctx, cancel := context.WithDeadline(context.Background(), deadline)

	dh.pOnuTP.resetProcessingErrorIndication()
	var wg sync.WaitGroup
	wg.Add(1) // for the 1 go routine to finish
	go dh.pOnuTP.deleteTpResource(dctx, delGemPortMsg.UniId, delGemPortMsg.TpPath,
		cResourceGemPort, delGemPortMsg.GemPortId, &wg)
	err := dh.pOnuTP.waitForTpCompletion(cancel, &wg) //let that also run off-line to let the IA messaging return!
	return err
}

func (dh *deviceHandler) processInterAdapterDeleteTcontReqMessage(
	msg *ic.InterAdapterMessage) error {
	if dh.pOnuTP == nil {
		//should normally not happen ...
		logger.Warnw("onuTechProf instance not set up for DelTcont request - ignoring request",
			log.Fields{"device-id": dh.deviceID})
		return errors.New("techProfile DelTcont request while onuTechProf instance not setup")
	}

	msgBody := msg.GetBody()
	delTcontMsg := &ic.InterAdapterDeleteTcontMessage{}
	if err := ptypes.UnmarshalAny(msgBody, delTcontMsg); err != nil {
		logger.Warnw("cannot-unmarshal-delete-tcont-msg-body", log.Fields{
			"device-id": dh.deviceID, "error": err})
		return err
	}

	dh.pOnuTP.lockTpProcMutex()
	if bTpModify := dh.pOnuTP.updateOnuUniTpPath(delTcontMsg.UniId, ""); bTpModify {
		// deadline context to ensure completion of background routines waited for
		deadline := time.Now().Add(10 * time.Second) //allowed run time to finish before execution
		dctx, cancel := context.WithDeadline(context.Background(), deadline)

		dh.pOnuTP.resetProcessingErrorIndication()
		var wg sync.WaitGroup
		wg.Add(2) // for the 2 go routines to finish
		go dh.pOnuTP.deleteTpResource(dctx, delTcontMsg.UniId, delTcontMsg.TpPath,
			cResourceTcont, delTcontMsg.AllocId, &wg)
		// Removal of the tcont/alloc id mapping represents the removal of the tech profile
		go dh.pOnuTP.updateOnuTpPathKvStore(dctx, &wg)
		//the wait.. function is responsible for tpProcMutex.Unlock()
		err := dh.pOnuTP.waitForTpCompletion(cancel, &wg) //let that also run off-line to let the IA messaging return!
		return err
	}
	dh.pOnuTP.unlockTpProcMutex()
	return nil
}

//processInterAdapterMessage sends the proxied messages to the target device
// If the proxy address is not found in the unmarshalled message, it first fetches the onu device for which the message
// is meant, and then send the unmarshalled omci message to this onu
func (dh *deviceHandler) processInterAdapterMessage(msg *ic.InterAdapterMessage) error {
	msgID := msg.Header.Id
	msgType := msg.Header.Type
	fromTopic := msg.Header.FromTopic
	toTopic := msg.Header.ToTopic
	toDeviceID := msg.Header.ToDeviceId
	proxyDeviceID := msg.Header.ProxyDeviceId
	logger.Debugw("InterAdapter message header", log.Fields{"msgID": msgID, "msgType": msgType,
		"fromTopic": fromTopic, "toTopic": toTopic, "toDeviceID": toDeviceID, "proxyDeviceID": proxyDeviceID})

	switch msgType {
	case ic.InterAdapterMessageType_OMCI_REQUEST:
		{
			return dh.processInterAdapterOMCIReqMessage(msg)
		}
	case ic.InterAdapterMessageType_ONU_IND_REQUEST:
		{
			return dh.processInterAdapterONUIndReqMessage(msg)
		}
	case ic.InterAdapterMessageType_TECH_PROFILE_DOWNLOAD_REQUEST:
		{
			return dh.processInterAdapterTechProfileDownloadReqMessage(msg)
		}
	case ic.InterAdapterMessageType_DELETE_GEM_PORT_REQUEST:
		{
			return dh.processInterAdapterDeleteGemPortReqMessage(msg)

		}
	case ic.InterAdapterMessageType_DELETE_TCONT_REQUEST:
		{
			return dh.processInterAdapterDeleteTcontReqMessage(msg)
		}
	default:
		{
			logger.Errorw("inter-adapter-unhandled-type", log.Fields{
				"device-id": dh.deviceID, "msgType": msg.Header.Type})
			return errors.New("unimplemented")
		}
	}
}

//FlowUpdateIncremental removes and/or adds the flow changes on a given device
func (dh *deviceHandler) FlowUpdateIncremental(apOfFlowChanges *openflow_13.FlowChanges,
	apOfGroupChanges *openflow_13.FlowGroupChanges, apFlowMetaData *voltha.FlowMetadata) error {

	if apOfFlowChanges.ToRemove != nil {
		for _, flowItem := range apOfFlowChanges.ToRemove.Items {
			logger.Debugw("incremental flow item remove", log.Fields{"deviceId": dh.deviceID,
				"Item": flowItem})
		}
	}
	if apOfFlowChanges.ToAdd != nil {
		for _, flowItem := range apOfFlowChanges.ToAdd.Items {
			if flowItem.GetCookie() == 0 {
				logger.Debugw("incremental flow add - no cookie - ignore", log.Fields{
					"deviceId": dh.deviceID})
				continue
			}
			flowInPort := flow.GetInPort(flowItem)
			if flowInPort == uint32(of.OfpPortNo_OFPP_INVALID) {
				logger.Errorw("flow inPort invalid", log.Fields{"deviceID": dh.deviceID})
				return errors.New("flow inPort invalid")
			} else if flowInPort == dh.ponPortNumber {
				//this is some downstream flow
				logger.Debugw("incremental flow ignore downstream", log.Fields{
					"deviceId": dh.deviceID, "inPort": flowInPort})
				continue
			} else {
				// this is the relevant upstream flow
				var loUniPort *onuUniPort
				if uniPort, exist := dh.uniEntityMap[flowInPort]; exist {
					loUniPort = uniPort
				} else {
					logger.Errorw("flow inPort not found in UniPorts",
						log.Fields{"deviceID": dh.deviceID, "inPort": flowInPort})
					return fmt.Errorf("flow-parameter inPort %d not found in internal UniPorts", flowInPort)
				}
				flowOutPort := flow.GetOutPort(flowItem)
				logger.Debugw("incremental flow-add port indications", log.Fields{
					"deviceId": dh.deviceID, "inPort": flowInPort, "outPort": flowOutPort,
					"uniPortName": loUniPort.name})
				err := dh.addFlowItemToUniPort(flowItem, loUniPort)
				//abort processing in error case
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

//disableDevice locks the ONU and its UNI/VEIP ports (admin lock via OMCI)
// TODO!!! Clarify usage of this method, it is for sure not used within ONOS (OLT) device disable
//         maybe it is obsolete by now
func (dh *deviceHandler) disableDevice(device *voltha.Device) {
	logger.Debugw("disable-device", log.Fields{"device-id": device.Id, "SerialNumber": device.SerialNumber})

	if dh.deviceReason != "omci-admin-lock" {
		// disable UNI ports/ONU
		// *** should generate UniAdminStateDone event - unrelated to DeviceProcStatusUpdate!!
		//     here the result of the processing is not checked (trusted in background) *****
		if dh.pLockStateFsm == nil {
			dh.createUniLockFsm(true, UniAdminStateDone)
		} else { //LockStateFSM already init
			dh.pLockStateFsm.setSuccessEvent(UniAdminStateDone)
			dh.runUniLockFsm(true)
		}

		if err := dh.coreProxy.DeviceReasonUpdate(context.TODO(), dh.deviceID, "omci-admin-lock"); err != nil {
			//TODO with VOL-3045/VOL-3046: return the error and stop further processing
			logger.Errorw("error-updating-reason-state", log.Fields{"device-id": dh.deviceID, "error": err})
		}
		dh.deviceReason = "omci-admin-lock"
		//200604: ConnState improved to 'unreachable' (was not set in python-code), OperState 'unknown' seems to be best choice
		if err := dh.coreProxy.DeviceStateUpdate(context.TODO(), dh.deviceID, voltha.ConnectStatus_UNREACHABLE,
			voltha.OperStatus_UNKNOWN); err != nil {
			//TODO with VOL-3045/VOL-3046: return the error and stop further processing
			logger.Errorw("error-updating-device-state", log.Fields{"device-id": dh.deviceID, "error": err})
		}

		// update json file
		// onustat := &OnuStatus{
		// 	Id:           dh.deviceID,
		// 	AdminState:   vc.AdminState_Types_name[int32(dh.device.AdminState)],
		// 	OpeState:     vc.OperStatus_Types_name[int32(dh.device.OperStatus)],
		// 	ConnectState: vc.ConnectStatus_Types_name[int32(dh.device.ConnectStatus)],
		// 	MacAddress:   dh.device.MacAddress,
		// }
		// logger.Debugw("UpdateOnu", log.Fields{"Id": dh.deviceID})
		// err := UpdateOnu(onustat)
		// logger.Debugw("UpdateOnu", log.Fields{"err": err})
	}
}

//reEnableDevice unlocks the ONU and its UNI/VEIP ports (admin unlock via OMCI)
// TODO!!! Clarify usage of this method, compare above DisableDevice, usage may clarify resulting states
//         maybe it is obsolete by now
func (dh *deviceHandler) reEnableDevice(device *voltha.Device) {
	logger.Debugw("reenable-device", log.Fields{"device-id": device.Id, "SerialNumber": device.SerialNumber})

	if err := dh.coreProxy.DeviceStateUpdate(context.TODO(), dh.deviceID, voltha.ConnectStatus_REACHABLE,
		voltha.OperStatus_ACTIVE); err != nil {
		//TODO with VOL-3045/VOL-3046: return the error and stop further processing
		logger.Errorw("error-updating-device-state", log.Fields{"device-id": dh.deviceID, "error": err})
	}

	if err := dh.coreProxy.DeviceReasonUpdate(context.TODO(), dh.deviceID, "initial-mib-downloaded"); err != nil {
		//TODO with VOL-3045/VOL-3046: return the error and stop further processing
		logger.Errorw("error-updating-reason-state", log.Fields{"device-id": dh.deviceID, "error": err})
	}
	dh.deviceReason = "initial-mib-downloaded"


	if dh.pUnlockStateFsm == nil {
		dh.createUniLockFsm(false, UniAdminStateDone)
	} else { //UnlockStateFSM already init
		dh.pUnlockStateFsm.setSuccessEvent(UniAdminStateDone)
		dh.runUniLockFsm(false)
	}
}

func (dh *deviceHandler) reconcileDeviceOnuInd() {
	logger.Debugw("reconciling - simulate onu indication", log.Fields{"device-id": dh.deviceID})

	if err := dh.pOnuTP.restoreFromOnuTpPathKvStore(context.TODO()); err != nil {
		logger.Errorw("reconciling - restoring OnuTp-data failed - abort", log.Fields{"err": err, "device-id": dh.deviceID})
		dh.reconciling = false
		return
	}
	var onuIndication oop.OnuIndication
	onuIndication.IntfId = dh.pOnuTP.sOnuPersistentData.PersIntfID
	onuIndication.OnuId = dh.pOnuTP.sOnuPersistentData.PersOnuID
	onuIndication.OperState = dh.pOnuTP.sOnuPersistentData.PersOperState
	onuIndication.AdminState = dh.pOnuTP.sOnuPersistentData.PersAdminState
	_ = dh.createInterface(&onuIndication)
}

func (dh *deviceHandler) reconcileDeviceTechProf() {
	logger.Debugw("reconciling - trigger tech profile config", log.Fields{"device-id": dh.deviceID})

	dh.pOnuTP.lockTpProcMutex()
	for _, uniData := range dh.pOnuTP.sOnuPersistentData.PersUniTpPath {
		//In order to allow concurrent calls to other dh instances we do not wait for execution here
		//but doing so we can not indicate problems to the caller (who does what with that then?)
		//by now we just assume straightforward successful execution
		//TODO!!! Generally: In this scheme it would be good to have some means to indicate
		//  possible problems to the caller later autonomously

		// deadline context to ensure completion of background routines waited for
		//20200721: 10s proved to be less in 8*8 ONU test on local vbox machine with debug, might be further adapted
		deadline := time.Now().Add(30 * time.Second) //allowed run time to finish before execution
		dctx, cancel := context.WithDeadline(context.Background(), deadline)

		dh.pOnuTP.resetProcessingErrorIndication()
		var wg sync.WaitGroup
		wg.Add(1) // for the 1 go routines to finish
		// attention: deadline completion check and wg.Done is to be done in both routines
		go dh.pOnuTP.configureUniTp(dctx, uint8(uniData.PersUniID), uniData.PersTpPath, &wg)
		//the wait.. function is responsible for tpProcMutex.Unlock()
		_ = dh.pOnuTP.waitForTpCompletion(cancel, &wg) //wait for background process to finish and collect their result
		return
	}
	dh.pOnuTP.unlockTpProcMutex()
	dh.reconciling = false
}

func (dh *deviceHandler) deleteDevice(device *voltha.Device) error {
	logger.Debugw("delete-device", log.Fields{"device-id": device.Id, "SerialNumber": device.SerialNumber})
	if err := dh.pOnuTP.deleteOnuTpPathKvStore(context.TODO()); err != nil {
		return err
	}
	return nil
}

func (dh *deviceHandler) rebootDevice(device *voltha.Device) error {
	logger.Debugw("reboot-device", log.Fields{"device-id": device.Id, "SerialNumber": device.SerialNumber})
	if device.ConnectStatus != voltha.ConnectStatus_REACHABLE {
		logger.Errorw("device-unreachable", log.Fields{"device-id": device.Id, "SerialNumber": device.SerialNumber})
		return errors.New("device-unreachable")
	}
	if err := dh.coreProxy.DeviceStateUpdate(context.TODO(), dh.deviceID, voltha.ConnectStatus_UNREACHABLE,
		voltha.OperStatus_DISCOVERED); err != nil {
		//TODO with VOL-3045/VOL-3046: return the error and stop further processing
		logger.Errorw("error-updating-device-state", log.Fields{"device-id": dh.deviceID, "error": err})
		return err
	}
	if err := dh.coreProxy.DeviceReasonUpdate(context.TODO(), dh.deviceID, "rebooting-onu"); err != nil {
		//TODO with VOL-3045/VOL-3046: return the error and stop further processing
		logger.Errorw("error-updating-reason-state", log.Fields{"device-id": dh.deviceID, "error": err})
		return err
	}
	dh.deviceReason = "rebooting-onu"

	if err := dh.sendMsgToOlt(context.TODO(), "reboot", nil); err != nil {
		logger.Debugw("error", log.Fields{"error": err})
	}

	return nil
}

//  deviceHandler methods that implement the adapters interface requests## end #########
// #####################################################################################

// ################  to be updated acc. needs of ONU Device ########################
// deviceHandler StateMachine related state transition methods ##### begin #########

func (dh *deviceHandler) logStateChange(e *fsm.Event) {
	logger.Debugw("Device FSM: ", log.Fields{"event name": string(e.Event), "src state": string(e.Src), "dst state": string(e.Dst), "device-id": dh.deviceID})
}

// doStateInit provides the device update to the core
func (dh *deviceHandler) doStateInit(e *fsm.Event) {

	logger.Debug("doStateInit-started")
	var err error

	dh.device.Root = false
	dh.device.Vendor = "OpenONU"
	dh.device.Model = "go"
	dh.device.Reason = "activating-onu"
	dh.deviceReason = "activating-onu"

	dh.logicalDeviceID = dh.deviceID // really needed - what for ??? //TODO!!!

	if !dh.reconciling {
		_ = dh.coreProxy.DeviceUpdate(context.TODO(), dh.device)
	} else {
		logger.Debugw("reconciling - don't notify core about DeviceUpdate",
			log.Fields{"device-id": dh.deviceID})
	}

	dh.parentID = dh.device.ParentId
	dh.ponPortNumber = dh.device.ParentPortNo

	dh.ProxyAddressID = dh.device.ProxyAddress.GetDeviceId()
	dh.ProxyAddressType = dh.device.ProxyAddress.GetDeviceType()
	logger.Debugw("device-updated", log.Fields{"device-id": dh.deviceID, "proxyAddressID": dh.ProxyAddressID,
		"proxyAddressType": dh.ProxyAddressType, "SNR": dh.device.SerialNumber,
		"ParentId": dh.parentID, "ParentPortNo": dh.ponPortNumber})

	/*
		self._pon = PonPort.create(self, self._pon_port_number)
		self._pon.add_peer(self.parent_id, self._pon_port_number)
		self.logger.debug('adding-pon-port-to-agent',
				   type=self._pon.get_port().type,
				   admin_state=self._pon.get_port().admin_state,
				   oper_status=self._pon.get_port().oper_status,
				   )
	*/
	if !dh.reconciling {
		logger.Debugw("adding-pon-port", log.Fields{"deviceID": dh.deviceID, "ponPortNo": dh.ponPortNumber})
		var ponPortNo uint32 = 1
		if dh.ponPortNumber != 0 {
			ponPortNo = dh.ponPortNumber
		}

		pPonPort := &voltha.Port{
			PortNo: ponPortNo,
			//Label:      fmt.Sprintf("pon-%d", ponPortNo),
			Label:      "PON port",
			Type:       voltha.Port_PON_ONU,
			OperStatus: voltha.OperStatus_ACTIVE,
			Peers: []*voltha.Port_PeerPort{{DeviceId: dh.parentID, // Peer device  is OLT
				PortNo: ponPortNo}}, // Peer port is parent's port number
		}
		if err = dh.coreProxy.PortCreated(context.TODO(), dh.deviceID, pPonPort); err != nil {
			logger.Fatalf("Device FSM: PortCreated-failed-%s", err)
			e.Cancel(err)
			return
		}
	} else {
		logger.Debugw("reconciling - pon-port already added", log.Fields{"device-id": dh.deviceID})
	}
	logger.Debug("doStateInit-done")
}

// postInit setups the DeviceEntry for the conerned device
func (dh *deviceHandler) postInit(e *fsm.Event) {

	logger.Debug("postInit-started")
	var err error
	/*
		dh.Client = oop.NewOpenoltClient(dh.clientCon)
		dh.pTransitionMap.Handle(ctx, GrpcConnected)
		return nil
	*/
	if err = dh.addOnuDeviceEntry(context.TODO()); err != nil {
		logger.Fatalf("Device FSM: addOnuDeviceEntry-failed-%s", err)
		e.Cancel(err)
		return
	}

	if dh.reconciling {
		go dh.reconcileDeviceOnuInd()
		// reconcilement will be continued after mib download is done
	}
	/*
			############################################################################
			# Setup Alarm handler
			self.events = AdapterEvents(self.core_proxy, device.id, self.logical_device_id,
										device.serial_number)
			############################################################################
			# Setup PM configuration for this device
			# Pass in ONU specific options
			kwargs = {
				OnuPmMetrics.DEFAULT_FREQUENCY_KEY: OnuPmMetrics.DEFAULT_ONU_COLLECTION_FREQUENCY,
				'heartbeat': self.heartbeat,
				OnuOmciPmMetrics.OMCI_DEV_KEY: self._onu_omci_device
			}
			self.logger.debug('create-pm-metrics', device_id=device.id, serial_number=device.serial_number)
			self._pm_metrics = OnuPmMetrics(self.events, self.core_proxy, self.device_id,
										   self.logical_device_id, device.serial_number,
										   grouped=True, freq_override=False, **kwargs)
			pm_config = self._pm_metrics.make_proto()
			self._onu_omci_device.set_pm_config(self._pm_metrics.omci_pm.openomci_interval_pm)
			self.logger.info("initial-pm-config", device_id=device.id, serial_number=device.serial_number)
			yield self.core_proxy.device_pm_config_update(pm_config, init=True)

			# Note, ONU ID and UNI intf set in add_uni_port method
			self._onu_omci_device.alarm_synchronizer.set_alarm_params(mgr=self.events,
																	  ani_ports=[self._pon])

			# Code to Run OMCI Test Action
			kwargs_omci_test_action = {
				OmciTestRequest.DEFAULT_FREQUENCY_KEY:
					OmciTestRequest.DEFAULT_COLLECTION_FREQUENCY
			}
			serial_number = device.serial_number
			self._test_request = OmciTestRequest(self.core_proxy,
										   self.omci_agent, self.device_id,
										   AniG, serial_number,
										   self.logical_device_id,
										   exclusive=False,
										   **kwargs_omci_test_action)

			self.enabled = True
		else:
			self.logger.info('onu-already-activated')
	*/
	logger.Debug("postInit-done")
}

// doStateConnected get the device info and update to voltha core
// for comparison of the original method (not that easy to uncomment): compare here:
//  voltha-openolt-adapter/adaptercore/device_handler.go
//  -> this one obviously initiates all communication interfaces of the device ...?
func (dh *deviceHandler) doStateConnected(e *fsm.Event) {

	logger.Debug("doStateConnected-started")
	err := errors.New("device FSM: function not implemented yet")
	e.Cancel(err)
	logger.Debug("doStateConnected-done")
}

// doStateUp handle the onu up indication and update to voltha core
func (dh *deviceHandler) doStateUp(e *fsm.Event) {

	logger.Debug("doStateUp-started")
	err := errors.New("device FSM: function not implemented yet")
	e.Cancel(err)
	logger.Debug("doStateUp-done")

	/*
		// Synchronous call to update device state - this method is run in its own go routine
		if err := dh.coreProxy.DeviceStateUpdate(ctx, dh.device.Id, voltha.ConnectStatus_REACHABLE,
			voltha.OperStatus_ACTIVE); err != nil {
			logger.Errorw("Failed to update device with OLT UP indication", log.Fields{"deviceID": dh.device.Id, "error": err})
			return err
		}
		return nil
	*/
}

// doStateDown handle the onu down indication
func (dh *deviceHandler) doStateDown(e *fsm.Event) {

	logger.Debug("doStateDown-started")
	var err error

	device := dh.device
	if device == nil {
		/*TODO: needs to handle error scenarios */
		logger.Error("Failed to fetch handler device")
		e.Cancel(err)
		return
	}

	cloned := proto.Clone(device).(*voltha.Device)
	logger.Debugw("do-state-down", log.Fields{"ClonedDeviceID": cloned.Id})
	/*
		// Update the all ports state on that device to disable
		if er := dh.coreProxy.PortsStateUpdate(ctx, cloned.Id, voltha.OperStatus_UNKNOWN); er != nil {
			logger.Errorw("updating-ports-failed", log.Fields{"deviceID": device.Id, "error": er})
			return er
		}

		//Update the device oper state and connection status
		cloned.OperStatus = voltha.OperStatus_UNKNOWN
		cloned.ConnectStatus = common.ConnectStatus_UNREACHABLE
		dh.device = cloned

		if er := dh.coreProxy.DeviceStateUpdate(ctx, cloned.Id, cloned.ConnectStatus, cloned.OperStatus); er != nil {
			logger.Errorw("error-updating-device-state", log.Fields{"deviceID": device.Id, "error": er})
			return er
		}

		//get the child device for the parent device
		onuDevices, err := dh.coreProxy.GetChildDevices(ctx, dh.device.Id)
		if err != nil {
			logger.Errorw("failed to get child devices information", log.Fields{"deviceID": dh.device.Id, "error": err})
			return err
		}
		for _, onuDevice := range onuDevices.Items {

			// Update onu state as down in onu adapter
			onuInd := oop.OnuIndication{}
			onuInd.OperState = "down"
			er := dh.AdapterProxy.SendInterAdapterMessage(ctx, &onuInd, ic.InterAdapterMessageType_ONU_IND_REQUEST,
				"openolt", onuDevice.Type, onuDevice.Id, onuDevice.ProxyAddress.DeviceId, "")
			if er != nil {
				logger.Errorw("Failed to send inter-adapter-message", log.Fields{"OnuInd": onuInd,
					"From Adapter": "openolt", "DevieType": onuDevice.Type, "DeviceID": onuDevice.Id})
				//Do not return here and continue to process other ONUs
			}
		}
		// * Discovered ONUs entries need to be cleared , since after OLT
		//   is up, it starts sending discovery indications again* /
		dh.discOnus = sync.Map{}
		logger.Debugw("do-state-down-end", log.Fields{"deviceID": device.Id})
		return nil
	*/
	err = errors.New("device FSM: function not implemented yet")
	e.Cancel(err)
	logger.Debug("doStateDown-done")
}

// deviceHandler StateMachine related state transition methods ##### end #########
// #################################################################################

// ###################################################
// deviceHandler utility methods ##### begin #########

//getOnuDeviceEntry getsthe  ONU device entry and may wait until its value is defined
func (dh *deviceHandler) getOnuDeviceEntry(aWait bool) *OnuDeviceEntry {
	dh.lockDevice.RLock()
	pOnuDeviceEntry := dh.pOnuOmciDevice
	if aWait && pOnuDeviceEntry == nil {
		//keep the read sema short to allow for subsequent write
		dh.lockDevice.RUnlock()
		logger.Debugw("Waiting for DeviceEntry to be set ...", log.Fields{"device-id": dh.deviceID})
		// based on concurrent processing the deviceEntry setup may not yet be finished at his point
		// so it might be needed to wait here for that event with some timeout
		select {
		case <-time.After(60 * time.Second): //timer may be discussed ...
			logger.Errorw("No valid DeviceEntry set after maxTime", log.Fields{"device-id": dh.deviceID})
			return nil
		case <-dh.deviceEntrySet:
			logger.Debugw("devicEntry ready now - continue", log.Fields{"device-id": dh.deviceID})
			// if written now, we can return the written value without sema
			return dh.pOnuOmciDevice
		}
	}
	dh.lockDevice.RUnlock()
	return pOnuDeviceEntry
}

//setOnuDeviceEntry sets the ONU device entry within the handler
func (dh *deviceHandler) setOnuDeviceEntry(
	apDeviceEntry *OnuDeviceEntry, apOnuTp *onuUniTechProf) {
	dh.lockDevice.Lock()
	defer dh.lockDevice.Unlock()
	dh.pOnuOmciDevice = apDeviceEntry
	dh.pOnuTP = apOnuTp
}

//addOnuDeviceEntry creates a new ONU device or returns the existing
func (dh *deviceHandler) addOnuDeviceEntry(ctx context.Context) error {
	logger.Debugw("adding-deviceEntry", log.Fields{"device-id": dh.deviceID})

	deviceEntry := dh.getOnuDeviceEntry(false)
	if deviceEntry == nil {
		/* costum_me_map in python code seems always to be None,
		   we omit that here first (declaration unclear) -> todo at Adapter specialization ...*/
		/* also no 'clock' argument - usage open ...*/
		/* and no alarm_db yet (oo.alarm_db)  */
		deviceEntry = newOnuDeviceEntry(ctx, dh.deviceID, dh.pOpenOnuAc.KVStoreHost,
			dh.pOpenOnuAc.KVStorePort, dh.pOpenOnuAc.KVStoreType,
			dh, dh.coreProxy, dh.AdapterProxy,
			dh.pOpenOnuAc.pSupportedFsms) //nil as FSM pointer would yield deviceEntry internal defaults ...
		onuTechProfProc := newOnuUniTechProf(ctx, dh.deviceID, dh)
		//error treatment possible //TODO!!!
		dh.setOnuDeviceEntry(deviceEntry, onuTechProfProc)
		// fire deviceEntry ready event to spread to possibly waiting processing
		dh.deviceEntrySet <- true

		//dh.sendMsgToOlt(ctx, "adopt", nil)
		// // update json file
		// onustat := &OnuStatus{
		// 	Id:           dh.deviceID,
		// 	AdminState:   vc.AdminState_Types_name[int32(dh.device.AdminState)],
		// 	OpeState:     vc.OperStatus_Types_name[int32(dh.device.OperStatus)],
		// 	ConnectState: vc.ConnectStatus_Types_name[int32(dh.device.ConnectStatus)],
		// 	MacAddress:   dh.device.MacAddress,
		// }
		// logger.Debugw("AddOnu", log.Fields{"Id": onustat.Id, "Admin": onustat.AdminState, "Ope": onustat.OpeState, "Con": onustat.ConnectState, "Mac": onustat.MacAddress})
		// err := AddOnu(onustat)
		// logger.Debugw("AddOnu", log.Fields{"err": err})
		// //
		// go WatchStatus(ctx, dh.coreProxy)

		// dh.device.Root = true
		// dh.coreProxy.DeviceUpdate(context.TODO(), dh.device)

		logger.Infow("onuDeviceEntry-added", log.Fields{"device-id": dh.deviceID})
	} else {
		logger.Infow("onuDeviceEntry-add: Device already exists", log.Fields{"device-id": dh.deviceID})
	}
	return nil
}

// doStateInit provides the device update to the core
func (dh *deviceHandler) createInterface(onuind *oop.OnuIndication) error {
	logger.Debugw("create_interface-started", log.Fields{"OnuId": onuind.GetOnuId(),
		"OnuIntfId": onuind.GetIntfId(), "OnuSerialNumber": onuind.GetSerialNumber()})

	dh.pOnuIndication = onuind // let's revise if storing the pointer is sufficient...

	if !dh.reconciling {
		if err := dh.coreProxy.DeviceStateUpdate(context.TODO(), dh.deviceID,
			voltha.ConnectStatus_REACHABLE, voltha.OperStatus_ACTIVATING); err != nil {
			//TODO with VOL-3045/VOL-3046: return the error and stop further processing
			logger.Errorw("error-updating-device-state", log.Fields{"device-id": dh.deviceID, "error": err})
		}
	} else {
		logger.Debugw("reconciling - don't notify core about DeviceStateUpdate to ACTIVATING",
			log.Fields{"device-id": dh.deviceID})
	}
	/*
			device, err := dh.coreProxy.GetDevice(context.TODO(), dh.device.Id, dh.device.Id)
		    if err != nil || device == nil {
					//TODO: needs to handle error scenarios
				logger.Errorw("Failed to fetch device device at creating If", log.Fields{"err": err})
				return errors.New("Voltha Device not found")
			}
	*/

	pDevEntry := dh.getOnuDeviceEntry(true)
	if pDevEntry != nil {
		if err := pDevEntry.start(context.TODO()); err != nil {
			return err
		}
	} else {
		logger.Errorw("No valid OnuDevice -aborting", log.Fields{"device-id": dh.deviceID})
		return errors.New("no valid OnuDevice")
	}
	if !dh.reconciling {
		if err := dh.coreProxy.DeviceReasonUpdate(context.TODO(), dh.deviceID, "starting-openomci"); err != nil {
			//TODO with VOL-3045/VOL-3046: return the error and stop further processing
			logger.Errorw("error-DeviceReasonUpdate to starting-openomci", log.Fields{"device-id": dh.deviceID, "error": err})
		}
	} else {
		logger.Debugw("reconciling - don't notify core about DeviceReasonUpdate to starting-openomci",
			log.Fields{"device-id": dh.deviceID})
	}
	dh.deviceReason = "starting-openomci"

	/* this might be a good time for Omci Verify message?  */
	verifyExec := make(chan bool)
	omciVerify := newOmciTestRequest(context.TODO(),
		dh.device.Id, pDevEntry.PDevOmciCC,
		true, true) //eclusive and allowFailure (anyway not yet checked)
	omciVerify.performOmciTest(context.TODO(), verifyExec)

	/* 	give the handler some time here to wait for the OMCi verification result
	after Timeout start and try MibUpload FSM anyway
	(to prevent stopping on just not supported OMCI verification from ONU) */
	select {
	case <-time.After(2 * time.Second):
		logger.Warn("omci start-verification timed out (continue normal)")
	case testresult := <-verifyExec:
		logger.Infow("Omci start verification done", log.Fields{"result": testresult})
	}

	/* In py code it looks earlier (on activate ..)
			# Code to Run OMCI Test Action
			kwargs_omci_test_action = {
				OmciTestRequest.DEFAULT_FREQUENCY_KEY:
					OmciTestRequest.DEFAULT_COLLECTION_FREQUENCY
			}
			serial_number = device.serial_number
			self._test_request = OmciTestRequest(self.core_proxy,
											self.omci_agent, self.device_id,
											AniG, serial_number,
											self.logical_device_id,
											exclusive=False,
											**kwargs_omci_test_action)
	...
	                    # Start test requests after a brief pause
	                    if not self._test_request_started:
	                        self._test_request_started = True
	                        tststart = _STARTUP_RETRY_WAIT * (random.randint(1, 5))
	                        reactor.callLater(tststart, self._test_request.start_collector)

	*/
	/* which is then: in omci_test_request.py : */
	/*
	   def start_collector(self, callback=None):
	       """
	               Start the collection loop for an adapter if the frequency > 0

	               :param callback: (callable) Function to call to collect PM data
	       """
	       self.logger.info("starting-pm-collection", device_name=self.name, default_freq=self.default_freq)
	       if callback is None:
	           callback = self.perform_test_omci

	       if self.lc is None:
	           self.lc = LoopingCall(callback)

	       if self.default_freq > 0:
	           self.lc.start(interval=self.default_freq / 10)

	   def perform_test_omci(self):
	       """
	       Perform the initial test request
	       """
	       ani_g_entities = self._device.configuration.ani_g_entities
	       ani_g_entities_ids = list(ani_g_entities.keys()) if ani_g_entities \
	                                                     is not None else None
	       self._entity_id = ani_g_entities_ids[0]
	       self.logger.info('perform-test', entity_class=self._entity_class,
	                     entity_id=self._entity_id)
	       try:
	           frame = MEFrame(self._entity_class, self._entity_id, []).test()
	           result = yield self._device.omci_cc.send(frame)
	           if not result.fields['omci_message'].fields['success_code']:
	               self.logger.info('Self-Test Submitted Successfully',
	                             code=result.fields[
	                                 'omci_message'].fields['success_code'])
	           else:
	               raise TestFailure('Test Failure: {}'.format(
	                   result.fields['omci_message'].fields['success_code']))
	       except TimeoutError as e:
	           self.deferred.errback(failure.Failure(e))

	       except Exception as e:
	           self.logger.exception('perform-test-Error', e=e,
	                              class_id=self._entity_class,
	                              entity_id=self._entity_id)
	           self.deferred.errback(failure.Failure(e))

	*/


	/* Note: Even though FSM calls look 'synchronous' here, FSM is running in background with the effect that possible errors
	 * 	 within the MibUpload are not notified in the OnuIndication response, this might be acceptable here,
	 *   as further OltAdapter processing may rely on the deviceReason event 'MibUploadDone' as a result of the FSM processing
	 *   otherwise some processing synchronization would be required - cmp. e.g TechProfile processing
	 */
	pMibUlFsm := pDevEntry.pMibUploadFsm.pFsm
	if pMibUlFsm != nil {
		if pMibUlFsm.Is(ulStDisabled) {
			if err := pMibUlFsm.Event(ulEvStart); err != nil {
				logger.Errorw("MibSyncFsm: Can't go to state starting", log.Fields{"err": err})
				return errors.New("can't go to state starting")
			}
			logger.Debugw("MibSyncFsm", log.Fields{"state": string(pMibUlFsm.Current())})
			//Determine ONU status and start/re-start MIB Synchronization tasks
			//Determine if this ONU has ever synchronized
			if true { //TODO: insert valid check
				if err := pMibUlFsm.Event(ulEvResetMib); err != nil {
					logger.Errorw("MibSyncFsm: Can't go to state resetting_mib", log.Fields{"err": err})
					return errors.New("can't go to state resetting_mib")
				}
			} else {
				if err := pMibUlFsm.Event(ulEvExamineMds); err != nil {
					logger.Errorw("MibSyncFsm: Can't go to state examine_mds", log.Fields{"err": err})
					return errors.New("can't go to examine_mds")
				}
				logger.Debugw("state of MibSyncFsm", log.Fields{"state": string(pMibUlFsm.Current())})
				//Examine the MIB Data Sync
				// callbacks to be handled:
				// Event(ulEvSuccess)
				// Event(ulEvTimeout)
				// Event(ulEvMismatch)
			}
		} else {
			logger.Errorw("wrong state of MibSyncFsm - want: disabled", log.Fields{"have": string(pMibUlFsm.Current())})
			return errors.New("wrong state of MibSyncFsm")
		}
	} else {
		logger.Errorw("MibSyncFsm invalid - cannot be executed!!", log.Fields{"device-id": dh.deviceID})
		return errors.New("cannot execut MibSync")
	}
	return nil
}

func (dh *deviceHandler) updateInterface(onuind *oop.OnuIndication) error {
	if dh.deviceReason != "stopping-openomci" {
		logger.Debugw("updateInterface-started - stopping-device", log.Fields{"device-id": dh.deviceID})
		//stop all running SM processing - make use of the DH-state as mirrored in the deviceReason
		pDevEntry := dh.getOnuDeviceEntry(false)
		if pDevEntry == nil {
			logger.Errorw("No valid OnuDevice -aborting", log.Fields{"device-id": dh.deviceID})
			return errors.New("no valid OnuDevice")
		}

		switch dh.deviceReason {
		case "starting-openomci":
			{ //MIBSync FSM may run
				pMibUlFsm := pDevEntry.pMibUploadFsm.pFsm
				if pMibUlFsm != nil {
					_ = pMibUlFsm.Event(ulEvStop) //TODO!! verify if MibSyncFsm stop-processing is sufficient (to allow it again afterwards)
				}
			}
		case "discovery-mibsync-complete":
			{ //MibDownload may run
				pMibDlFsm := pDevEntry.pMibDownloadFsm.pFsm
				if pMibDlFsm != nil {
					_ = pMibDlFsm.Event(dlEvReset)
				}
			}
		default:
			{
				//port lock/unlock FSM's may be active
				if dh.pUnlockStateFsm != nil {
					_ = dh.pUnlockStateFsm.pAdaptFsm.pFsm.Event(uniEvReset)
				}
				if dh.pLockStateFsm != nil {
					_ = dh.pLockStateFsm.pAdaptFsm.pFsm.Event(uniEvReset)
				}
				//techProfile related PonAniConfigFsm FSM may be active
				// maybe encapsulated as OnuTP method - perhaps later in context of module splitting
				if dh.pOnuTP.pAniConfigFsm != nil {
					_ = dh.pOnuTP.pAniConfigFsm.pAdaptFsm.pFsm.Event(aniEvReset)
				}
				for _, uniPort := range dh.uniEntityMap {
					//reset the TechProfileConfig Done state for all (active) UNI's
					dh.pOnuTP.setConfigDone(uniPort.uniID, false)
					// reset the possibly existing VlanConfigFsm
					if pVlanFilterFsm, exist := dh.UniVlanConfigFsmMap[uniPort.uniID]; exist {
						//VlanFilterFsm exists and was already started
						pVlanFilterStatemachine := pVlanFilterFsm.pAdaptFsm.pFsm
						if pVlanFilterStatemachine != nil {
							_ = pVlanFilterStatemachine.Event(vlanEvReset)
						}
					}
				}
			}
			//TODO!!! care about PM/Alarm processing once started
		}
		//TODO: from here the deviceHandler FSM itself may be stuck in some of the initial states
		//  (mainly the still separate 'Event states')
		//  so it is questionable, how this is resolved after some possible re-enable
		//  assumption there is obviously, that the system may continue with some 'after "mib-download-done" state'

		//stop/remove(?) the device entry
		_ = pDevEntry.stop(context.TODO()) //maybe some more sophisticated context treatment should be used here?

		//TODO!!! remove existing traffic profiles
		/* from py code, if TP's exist, remove them - not yet implemented
		self._tp = dict()
		# Let TP download happen again
		for uni_id in self._tp_service_specific_task:
			self._tp_service_specific_task[uni_id].clear()
		for uni_id in self._tech_profile_download_done:
			self._tech_profile_download_done[uni_id].clear()
		*/

		dh.disableUniPortStateUpdate()

		if err := dh.coreProxy.DeviceReasonUpdate(context.TODO(), dh.deviceID, "stopping-openomci"); err != nil {
			//TODO with VOL-3045/VOL-3046: return the error and stop further processing
			logger.Errorw("error-DeviceReasonUpdate to 'stopping-openomci'",
				log.Fields{"device-id": dh.deviceID, "error": err})
			// abort: system behavior is just unstable ...
			return err
		}
		dh.deviceReason = "stopping-openomci"

		if err := dh.coreProxy.DeviceStateUpdate(context.TODO(), dh.deviceID,
			voltha.ConnectStatus_UNREACHABLE, voltha.OperStatus_DISCOVERED); err != nil {
			//TODO with VOL-3045/VOL-3046: return the error and stop further processing
			logger.Errorw("error-updating-device-state unreachable-discovered",
				log.Fields{"device-id": dh.deviceID, "error": err})
			// abort: system behavior is just unstable ...
			return err
		}
	} else {
		logger.Debugw("updateInterface - device already stopped", log.Fields{"device-id": dh.deviceID})
	}
	return nil
}

func (dh *deviceHandler) processMibDatabaseSyncEvent(devEvent OnuDeviceEvent) {
	logger.Debugw("MibInSync event received", log.Fields{"device-id": dh.deviceID})
	if !dh.reconciling {
		//initiate DevStateUpdate
		if err := dh.coreProxy.DeviceReasonUpdate(context.TODO(), dh.deviceID, "discovery-mibsync-complete"); err != nil {
			//TODO with VOL-3045/VOL-3046: return the error and stop further processing
			logger.Errorw("error-DeviceReasonUpdate to 'mibsync-complete'", log.Fields{
				"device-id": dh.deviceID, "error": err})
		} else {
			logger.Infow("dev reason updated to 'MibSync complete'", log.Fields{"deviceID": dh.deviceID})
		}
	} else {
		logger.Debugw("reconciling - don't notify core about DeviceReasonUpdate to mibsync-complete",
			log.Fields{"device-id": dh.deviceID})
	}
	dh.deviceReason = "discovery-mibsync-complete"

	i := uint8(0) //UNI Port limit: see MaxUnisPerOnu (by now 16) (OMCI supports max 255 p.b.)
	pDevEntry := dh.getOnuDeviceEntry(false)
	if unigInstKeys := pDevEntry.pOnuDB.getSortedInstKeys(me.UniGClassID); len(unigInstKeys) > 0 {
		for _, mgmtEntityID := range unigInstKeys {
			logger.Debugw("Add UNI port for stored UniG instance:", log.Fields{
				"device-id": dh.deviceID, "UnigMe EntityID": mgmtEntityID})
			dh.addUniPort(mgmtEntityID, i, uniPPTP)
			i++
		}
	} else {
		logger.Debugw("No UniG instances found", log.Fields{"device-id": dh.deviceID})
	}
	if veipInstKeys := pDevEntry.pOnuDB.getSortedInstKeys(me.VirtualEthernetInterfacePointClassID); len(veipInstKeys) > 0 {
		for _, mgmtEntityID := range veipInstKeys {
			logger.Debugw("Add VEIP acc. to stored VEIP instance:", log.Fields{
				"device-id": dh.deviceID, "VEIP EntityID": mgmtEntityID})
			dh.addUniPort(mgmtEntityID, i, uniVEIP)
			i++
		}
	} else {
		logger.Debugw("No VEIP instances found", log.Fields{"device-id": dh.deviceID})
	}
	if i == 0 {
		logger.Warnw("No PPTP instances found", log.Fields{"device-id": dh.deviceID})
	}

	/* 200605: lock processing after initial MIBUpload removed now as the ONU should be in the lock state per default here
	 *  left the code here as comment in case such processing should prove needed unexpectedly
			// Init Uni Ports to Admin locked state
			// maybe not really needed here as UNI ports should be locked by default, but still left as available in python code
			// *** should generate UniLockStateDone event *****
			if dh.pLockStateFsm == nil {
				dh.createUniLockFsm(true, UniLockStateDone)
			} else { //LockStateFSM already init
				dh.pLockStateFsm.SetSuccessEvent(UniLockStateDone)
				dh.runUniLockFsm(true)
			}
		}
	case UniLockStateDone:
		{
			logger.Infow("UniLockStateDone event: Starting MIB download", log.Fields{"device-id": dh.deviceID})
	* lockState processing commented out
	*/
	/*  Mib download procedure -
	***** should run over 'downloaded' state and generate MibDownloadDone event *****
	 */
	pMibDlFsm := pDevEntry.pMibDownloadFsm.pFsm
	if pMibDlFsm != nil {
		if pMibDlFsm.Is(dlStDisabled) {
			if err := pMibDlFsm.Event(dlEvStart); err != nil {
				logger.Errorw("MibDownloadFsm: Can't go to state starting", log.Fields{"err": err})
				// maybe try a FSM reset and then again ... - TODO!!!
			} else {
				logger.Debugw("MibDownloadFsm", log.Fields{"state": string(pMibDlFsm.Current())})
				// maybe use more specific states here for the specific download steps ...
				if err := pMibDlFsm.Event(dlEvCreateGal); err != nil {
					logger.Errorw("MibDownloadFsm: Can't start CreateGal", log.Fields{"err": err})
				} else {
					logger.Debugw("state of MibDownloadFsm", log.Fields{"state": string(pMibDlFsm.Current())})
					//Begin MIB data download (running autonomously)
				}
			}
		} else {
			logger.Errorw("wrong state of MibDownloadFsm - want: disabled", log.Fields{"have": string(pMibDlFsm.Current())})
			// maybe try a FSM reset and then again ... - TODO!!!
		}
		/***** Mib download started */
	} else {
		logger.Errorw("MibDownloadFsm invalid - cannot be executed!!", log.Fields{"device-id": dh.deviceID})
	}
}

func (dh *deviceHandler) processMibDownloadDoneEvent(devEvent OnuDeviceEvent) {
	logger.Debugw("MibDownloadDone event received", log.Fields{"device-id": dh.deviceID})
	if !dh.reconciling {
		if err := dh.coreProxy.DeviceStateUpdate(context.TODO(), dh.deviceID,
			voltha.ConnectStatus_REACHABLE, voltha.OperStatus_ACTIVE); err != nil {
			//TODO with VOL-3045/VOL-3046: return the error and stop further processing
			logger.Errorw("error-updating-device-state", log.Fields{"device-id": dh.deviceID, "error": err})
		} else {
			logger.Debugw("dev state updated to 'Oper.Active'", log.Fields{"device-id": dh.deviceID})
		}
	} else {
		logger.Debugw("reconciling - don't notify core about DeviceStateUpdate to ACTIVE",
			log.Fields{"device-id": dh.deviceID})
	}
	if !dh.reconciling {
		if err := dh.coreProxy.DeviceReasonUpdate(context.TODO(), dh.deviceID, "initial-mib-downloaded"); err != nil {
			//TODO with VOL-3045/VOL-3046: return the error and stop further processing
			logger.Errorw("error-DeviceReasonUpdate to 'initial-mib-downloaded'",
				log.Fields{"device-id": dh.deviceID, "error": err})
		} else {
			logger.Infow("dev reason updated to 'initial-mib-downloaded'", log.Fields{"device-id": dh.deviceID})
		}
	} else {
		logger.Debugw("reconciling - don't notify core about DeviceReasonUpdate to initial-mib-downloaded",
			log.Fields{"device-id": dh.deviceID})
	}
	dh.deviceReason = "initial-mib-downloaded"
	if dh.pUnlockStateFsm == nil {
		dh.createUniLockFsm(false, UniUnlockStateDone)
	} else { //UnlockStateFSM already init
		dh.pUnlockStateFsm.setSuccessEvent(UniUnlockStateDone)
		dh.runUniLockFsm(false)
	}
}

func (dh *deviceHandler) processUniUnlockStateDoneEvent(devEvent OnuDeviceEvent) {
	go dh.enableUniPortStateUpdate() //cmp python yield self.enable_ports()

	if !dh.reconciling {
		logger.Infow("UniUnlockStateDone event: Sending OnuUp event", log.Fields{"device-id": dh.deviceID})
		raisedTs := time.Now().UnixNano()
		go dh.sendOnuOperStateEvent(voltha.OperStatus_ACTIVE, dh.deviceID, raisedTs) //cmp python onu_active_event
	} else {
		logger.Debugw("reconciling - don't notify core that onu went to active but trigger tech profile config",
			log.Fields{"device-id": dh.deviceID})
		go dh.reconcileDeviceTechProf()
		//TODO: further actions e.g. restore flows, metrics, ...
	}
}

func (dh *deviceHandler) processOmciAniConfigDoneEvent(devEvent OnuDeviceEvent) {
	logger.Debugw("OmciAniConfigDone event received", log.Fields{"device-id": dh.deviceID})
	if dh.deviceReason != "tech-profile-config-download-success" {
		// which may be the case from some previous actvity on another UNI Port of the ONU
		if !dh.reconciling {
			if err := dh.coreProxy.DeviceReasonUpdate(context.TODO(), dh.deviceID, "tech-profile-config-download-success"); err != nil {
				//TODO with VOL-3045/VOL-3046: return the error and stop further processing
				logger.Errorw("error-DeviceReasonUpdate to 'tech-profile-config-download-success'",
					log.Fields{"device-id": dh.deviceID, "error": err})
			} else {
				logger.Infow("update dev reason to 'tech-profile-config-download-success'",
					log.Fields{"device-id": dh.deviceID})
			}
		} else {
			logger.Debugw("reconciling - don't notify core about DeviceReasonUpdate to tech-profile-config-download-success",
				log.Fields{"device-id": dh.deviceID})
		}
		//set internal state anyway - as it was done
		dh.deviceReason = "tech-profile-config-download-success"
	}
}

func (dh *deviceHandler) processOmciVlanFilterDoneEvent(devEvent OnuDeviceEvent) {
	logger.Debugw("OmciVlanFilterDone event received",
		log.Fields{"device-id": dh.deviceID})

	if dh.deviceReason != "omci-flows-pushed" {
		// which may be the case from some previous actvity on another UNI Port of the ONU
		// or even some previous flow add activity on the same port
		if err := dh.coreProxy.DeviceReasonUpdate(context.TODO(), dh.deviceID, "omci-flows-pushed"); err != nil {
			logger.Errorw("error-DeviceReasonUpdate to 'omci-flows-pushed'",
				log.Fields{"device-id": dh.deviceID, "error": err})
		} else {
			logger.Infow("updated dev reason to ''omci-flows-pushed'",
				log.Fields{"device-id": dh.deviceID})
		}
		//set internal state anyway - as it was done
		dh.deviceReason = "omci-flows-pushed"
	}
}

//deviceProcStatusUpdate evaluates possible processing events and initiates according next activities
func (dh *deviceHandler) deviceProcStatusUpdate(devEvent OnuDeviceEvent) {
	switch devEvent {
	case MibDatabaseSync:
		{
			dh.processMibDatabaseSyncEvent(devEvent)
		}
	case MibDownloadDone:
		{
			dh.processMibDownloadDoneEvent(devEvent)

		}
	case UniUnlockStateDone:
		{
			dh.processUniUnlockStateDoneEvent(devEvent)

		}
	case OmciAniConfigDone:
		{
			dh.processOmciAniConfigDoneEvent(devEvent)

		}
	case OmciVlanFilterDone:
		{
			dh.processOmciVlanFilterDoneEvent(devEvent)

		}
	default:
		{
			logger.Warnw("unhandled-device-event", log.Fields{"device-id": dh.deviceID, "event": devEvent})
		}
	} //switch
}

func (dh *deviceHandler) addUniPort(aUniInstNo uint16, aUniID uint8, aPortType uniPortType) {
	uniNo := mkUniPortNum(dh.pOnuIndication.GetIntfId(), dh.pOnuIndication.GetOnuId(),
		uint32(aUniID))
	if _, present := dh.uniEntityMap[uniNo]; present {
		logger.Warnw("onuUniPort-add: Port already exists", log.Fields{"for InstanceId": aUniInstNo})
	} else {
		//with arguments aUniID, a_portNo, aPortType
		pUniPort := newOnuUniPort(aUniID, uniNo, aUniInstNo, aPortType)
		if pUniPort == nil {
			logger.Warnw("onuUniPort-add: Could not create Port", log.Fields{"for InstanceId": aUniInstNo})
		} else {
			//store UniPort with the System-PortNumber key
			dh.uniEntityMap[uniNo] = pUniPort
			if !dh.reconciling {
				// create announce the UniPort to the core as VOLTHA Port object
				if err := pUniPort.createVolthaPort(dh); err == nil {
					logger.Infow("onuUniPort-added", log.Fields{"for PortNo": uniNo})
				} //error logging already within UniPort method
			} else {
				logger.Debugw("reconciling - onuUniPort already added", log.Fields{"for PortNo": uniNo, "device-id": dh.deviceID})
			}
		}
	}
}

// enableUniPortStateUpdate enables UniPortState and update core port state accordingly
func (dh *deviceHandler) enableUniPortStateUpdate() {


	for uniNo, uniPort := range dh.uniEntityMap {
		// only if this port is validated for operState transfer
		if (1<<uniPort.uniID)&activeUniPortStateUpdateMask == (1 << uniPort.uniID) {
			logger.Infow("onuUniPort-forced-OperState-ACTIVE", log.Fields{"for PortNo": uniNo})
			uniPort.setOperState(vc.OperStatus_ACTIVE)
			if !dh.reconciling {
				//maybe also use getter functions on uniPort - perhaps later ...
				go dh.coreProxy.PortStateUpdate(context.TODO(), dh.deviceID, voltha.Port_ETHERNET_UNI, uniPort.portNo, uniPort.operState)
			} else {
				logger.Debugw("reconciling - don't notify core about PortStateUpdate", log.Fields{"device-id": dh.deviceID})
			}
		}
	}
}

// Disable UniPortState and update core port state accordingly
func (dh *deviceHandler) disableUniPortStateUpdate() {
	for uniNo, uniPort := range dh.uniEntityMap {
		// only if this port is validated for operState transfer
		if (1<<uniPort.uniID)&activeUniPortStateUpdateMask == (1 << uniPort.uniID) {
			logger.Infow("onuUniPort-forced-OperState-UNKNOWN", log.Fields{"for PortNo": uniNo})
			uniPort.setOperState(vc.OperStatus_UNKNOWN)
			//maybe also use getter functions on uniPort - perhaps later ...
			go dh.coreProxy.PortStateUpdate(context.TODO(), dh.deviceID, voltha.Port_ETHERNET_UNI, uniPort.portNo, uniPort.operState)
		}
	}
}

// ONU_Active/Inactive announcement on system KAFKA bus
// tried to re-use procedure of oltUpDownIndication from openolt_eventmgr.go with used values from Py code
func (dh *deviceHandler) sendOnuOperStateEvent(aOperState vc.OperStatus_Types, aDeviceID string, raisedTs int64) {
	var de voltha.DeviceEvent
	eventContext := make(map[string]string)
	parentDevice, err := dh.coreProxy.GetDevice(context.TODO(), dh.parentID, dh.parentID)
	if err != nil || parentDevice == nil {
		logger.Errorw("Failed to fetch parent device for OnuEvent",
			log.Fields{"parentID": dh.parentID, "err": err})
	}
	oltSerialNumber := parentDevice.SerialNumber

	eventContext["pon-id"] = strconv.FormatUint(uint64(dh.pOnuIndication.IntfId), 10)
	eventContext["onu-id"] = strconv.FormatUint(uint64(dh.pOnuIndication.OnuId), 10)
	eventContext["serial-number"] = dh.device.SerialNumber
	eventContext["olt_serial_number"] = oltSerialNumber
	eventContext["device_id"] = aDeviceID
	eventContext["registration_id"] = aDeviceID //py: string(device_id)??
	logger.Debugw("prepare ONU_ACTIVATED event",
		log.Fields{"DeviceId": aDeviceID, "EventContext": eventContext})

	/* Populating device event body */
	de.Context = eventContext
	de.ResourceId = aDeviceID
	if aOperState == voltha.OperStatus_ACTIVE {
		de.DeviceEventName = fmt.Sprintf("%s_%s", cOnuActivatedEvent, "RAISE_EVENT")
		de.Description = fmt.Sprintf("%s Event - %s - %s",
			cEventObjectType, cOnuActivatedEvent, "Raised")
	} else {
		de.DeviceEventName = fmt.Sprintf("%s_%s", cOnuActivatedEvent, "CLEAR_EVENT")
		de.Description = fmt.Sprintf("%s Event - %s - %s",
			cEventObjectType, cOnuActivatedEvent, "Cleared")
	}
	/* Send event to KAFKA */
	if err := dh.EventProxy.SendDeviceEvent(&de, equipment, pon, raisedTs); err != nil {
		logger.Warnw("could not send ONU_ACTIVATED event",
			log.Fields{"device-id": aDeviceID, "error": err})
	}
	logger.Debugw("ONU_ACTIVATED event sent to KAFKA",
		log.Fields{"device-id": aDeviceID, "with-EventName": de.DeviceEventName})
}

// createUniLockFsm initializes and runs the UniLock FSM to transfer the OMCI related commands for port lock/unlock
func (dh *deviceHandler) createUniLockFsm(aAdminState bool, devEvent OnuDeviceEvent) {
	chLSFsm := make(chan Message, 2048)
	var sFsmName string
	if aAdminState {
		logger.Infow("createLockStateFSM", log.Fields{"device-id": dh.deviceID})
		sFsmName = "LockStateFSM"
	} else {
		logger.Infow("createUnlockStateFSM", log.Fields{"device-id": dh.deviceID})
		sFsmName = "UnLockStateFSM"
	}

	pDevEntry := dh.getOnuDeviceEntry(true)
	if pDevEntry == nil {
		logger.Errorw("No valid OnuDevice -aborting", log.Fields{"device-id": dh.deviceID})
		return
	}
	pLSFsm := newLockStateFsm(pDevEntry.PDevOmciCC, aAdminState, devEvent,
		sFsmName, dh.deviceID, chLSFsm)
	if pLSFsm != nil {
		if aAdminState {
			dh.pLockStateFsm = pLSFsm
		} else {
			dh.pUnlockStateFsm = pLSFsm
		}
		dh.runUniLockFsm(aAdminState)
	} else {
		logger.Errorw("LockStateFSM could not be created - abort!!", log.Fields{"device-id": dh.deviceID})
	}
}

// runUniLockFsm starts the UniLock FSM to transfer the OMCI related commands for port lock/unlock
func (dh *deviceHandler) runUniLockFsm(aAdminState bool) {
	/*  Uni Port lock/unlock procedure -
	 ***** should run via 'adminDone' state and generate the argument requested event *****
	 */
	var pLSStatemachine *fsm.FSM
	if aAdminState {
		pLSStatemachine = dh.pLockStateFsm.pAdaptFsm.pFsm
		//make sure the opposite FSM is not running and if so, terminate it as not relevant anymore
		if (dh.pUnlockStateFsm != nil) &&
			(dh.pUnlockStateFsm.pAdaptFsm.pFsm.Current() != uniStDisabled) {
			_ = dh.pUnlockStateFsm.pAdaptFsm.pFsm.Event(uniEvReset)
		}
	} else {
		pLSStatemachine = dh.pUnlockStateFsm.pAdaptFsm.pFsm
		//make sure the opposite FSM is not running and if so, terminate it as not relevant anymore
		if (dh.pLockStateFsm != nil) &&
			(dh.pLockStateFsm.pAdaptFsm.pFsm.Current() != uniStDisabled) {
			_ = dh.pLockStateFsm.pAdaptFsm.pFsm.Event(uniEvReset)
		}
	}
	if pLSStatemachine != nil {
		if pLSStatemachine.Is(uniStDisabled) {
			if err := pLSStatemachine.Event(uniEvStart); err != nil {
				logger.Warnw("LockStateFSM: can't start", log.Fields{"err": err})
				// maybe try a FSM reset and then again ... - TODO!!!
			} else {
				/***** LockStateFSM started */
				logger.Debugw("LockStateFSM started", log.Fields{
					"state": pLSStatemachine.Current(), "device-id": dh.deviceID})
			}
		} else {
			logger.Warnw("wrong state of LockStateFSM - want: disabled", log.Fields{
				"have": pLSStatemachine.Current(), "device-id": dh.deviceID})
			// maybe try a FSM reset and then again ... - TODO!!!
		}
	} else {
		logger.Errorw("LockStateFSM StateMachine invalid - cannot be executed!!", log.Fields{"device-id": dh.deviceID})
		// maybe try a FSM reset and then again ... - TODO!!!
	}
}

//setBackend provides a DB backend for the specified path on the existing KV client
func (dh *deviceHandler) setBackend(aBasePathKvStore string) *db.Backend {
	addr := dh.pOpenOnuAc.KVStoreHost + ":" + strconv.Itoa(dh.pOpenOnuAc.KVStorePort)
	logger.Debugw("SetKVStoreBackend", log.Fields{"IpTarget": addr,
		"BasePathKvStore": aBasePathKvStore, "device-id": dh.deviceID})
	kvbackend := &db.Backend{
		Client:    dh.pOpenOnuAc.kvClient,
		StoreType: dh.pOpenOnuAc.KVStoreType,
		/* address config update acc. to [VOL-2736] */
		Address:    addr,
		Timeout:    dh.pOpenOnuAc.KVStoreTimeout,
		PathPrefix: aBasePathKvStore}

	return kvbackend
}
func (dh *deviceHandler) getFlowOfbFields(apFlowItem *ofp.OfpFlowStats, loMatchVlan *uint16,
	loAddPcp *uint8, loIPProto *uint32) {

	for _, field := range flow.GetOfbFields(apFlowItem) {
		switch field.Type {
		case of.OxmOfbFieldTypes_OFPXMT_OFB_ETH_TYPE:
			{
				logger.Debugw("FlowAdd type EthType", log.Fields{"device-id": dh.deviceID,
					"EthType": strconv.FormatInt(int64(field.GetEthType()), 16)})
			}
		case of.OxmOfbFieldTypes_OFPXMT_OFB_IP_PROTO:
			{
				*loIPProto = field.GetIpProto()
				logger.Debugw("FlowAdd type IpProto", log.Fields{"device-id": dh.deviceID,
					"IpProto": strconv.FormatInt(int64(*loIPProto), 16)})
				if *loIPProto == 2 {
					// some workaround for TT workflow at proto == 2 (IGMP trap) -> ignore the flow
					// avoids installing invalid EVTOCD rule
					logger.Debugw("FlowAdd type IpProto 2: TT workaround: ignore flow",
						log.Fields{"device-id": dh.deviceID,
							"IpProto": strconv.FormatInt(int64(*loIPProto), 16)})
					return
				}
			}
		case of.OxmOfbFieldTypes_OFPXMT_OFB_VLAN_VID:
			{
				*loMatchVlan = uint16(field.GetVlanVid())
				loMatchVlanMask := uint16(field.GetVlanVidMask())
				if !(*loMatchVlan == uint16(of.OfpVlanId_OFPVID_PRESENT) &&
					loMatchVlanMask == uint16(of.OfpVlanId_OFPVID_PRESENT)) {
					*loMatchVlan = *loMatchVlan & 0xFFF // not transparent: copy only ID bits
				}
				logger.Debugw("FlowAdd field type", log.Fields{"device-id": dh.deviceID,
					"VID": strconv.FormatInt(int64(*loMatchVlan), 16)})
			}
		case of.OxmOfbFieldTypes_OFPXMT_OFB_VLAN_PCP:
			{
				*loAddPcp = uint8(field.GetVlanPcp())
				logger.Debugw("FlowAdd field type", log.Fields{"device-id": dh.deviceID,
					"PCP": loAddPcp})
			}
		case of.OxmOfbFieldTypes_OFPXMT_OFB_UDP_DST:
			{
				logger.Debugw("FlowAdd field type", log.Fields{"device-id": dh.deviceID,
					"UDP-DST": strconv.FormatInt(int64(field.GetUdpDst()), 16)})
			}
		case of.OxmOfbFieldTypes_OFPXMT_OFB_UDP_SRC:
			{
				logger.Debugw("FlowAdd field type", log.Fields{"device-id": dh.deviceID,
					"UDP-SRC": strconv.FormatInt(int64(field.GetUdpSrc()), 16)})
			}
		case of.OxmOfbFieldTypes_OFPXMT_OFB_IPV4_DST:
			{
				logger.Debugw("FlowAdd field type", log.Fields{"device-id": dh.deviceID,
					"IPv4-DST": field.GetIpv4Dst()})
			}
		case of.OxmOfbFieldTypes_OFPXMT_OFB_IPV4_SRC:
			{
				logger.Debugw("FlowAdd field type", log.Fields{"device-id": dh.deviceID,
					"IPv4-SRC": field.GetIpv4Src()})
			}
		case of.OxmOfbFieldTypes_OFPXMT_OFB_METADATA:
			{
				logger.Debugw("FlowAdd field type", log.Fields{"device-id": dh.deviceID,
					"Metadata": field.GetTableMetadata()})
			}
			/*
				default:
					{
						//all other entires ignored
					}
			*/
		}
	} //for all OfbFields
}

func (dh *deviceHandler) getFlowActions(apFlowItem *ofp.OfpFlowStats, loSetPcp *uint8, loSetVlan *uint16) {
	for _, action := range flow.GetActions(apFlowItem) {
		switch action.Type {
		/* not used:
		case of.OfpActionType_OFPAT_OUTPUT:
			{
				logger.Debugw("FlowAdd action type", log.Fields{"device-id": dh.deviceID,
					"Output": action.GetOutput()})
			}
		*/
		case of.OfpActionType_OFPAT_PUSH_VLAN:
			{
				logger.Debugw("FlowAdd action type", log.Fields{"device-id": dh.deviceID,
					"PushEthType": strconv.FormatInt(int64(action.GetPush().Ethertype), 16)})
			}
		case of.OfpActionType_OFPAT_SET_FIELD:
			{
				pActionSetField := action.GetSetField()
				if pActionSetField.Field.OxmClass != of.OfpOxmClass_OFPXMC_OPENFLOW_BASIC {
					logger.Warnw("FlowAdd action SetField invalid OxmClass (ignored)", log.Fields{"device-id": dh.deviceID,
						"OxcmClass": pActionSetField.Field.OxmClass})
				}
				if pActionSetField.Field.GetOfbField().Type == of.OxmOfbFieldTypes_OFPXMT_OFB_VLAN_VID {
					*loSetVlan = uint16(pActionSetField.Field.GetOfbField().GetVlanVid())
					logger.Debugw("FlowAdd Set VLAN from SetField action", log.Fields{"device-id": dh.deviceID,
						"SetVlan": strconv.FormatInt(int64(*loSetVlan), 16)})
				} else if pActionSetField.Field.GetOfbField().Type == of.OxmOfbFieldTypes_OFPXMT_OFB_VLAN_PCP {
					*loSetPcp = uint8(pActionSetField.Field.GetOfbField().GetVlanPcp())
					logger.Debugw("FlowAdd Set PCP from SetField action", log.Fields{"device-id": dh.deviceID,
						"SetPcp": *loSetPcp})
				} else {
					logger.Warnw("FlowAdd action SetField invalid FieldType", log.Fields{"device-id": dh.deviceID,
						"Type": pActionSetField.Field.GetOfbField().Type})
				}
			}
			/*
				default:
					{
						//all other entires ignored
					}
			*/
		}
	} //for all Actions
}

//addFlowItemToUniPort parses the actual flow item to add it to the UniPort
func (dh *deviceHandler) addFlowItemToUniPort(apFlowItem *ofp.OfpFlowStats, apUniPort *onuUniPort) error {
	var loSetVlan uint16 = uint16(of.OfpVlanId_OFPVID_NONE)      //noValidEntry
	var loMatchVlan uint16 = uint16(of.OfpVlanId_OFPVID_PRESENT) //reserved VLANID entry
	var loAddPcp, loSetPcp uint8
	var loIPProto uint32
	/* the TechProfileId is part of the flow Metadata - compare also comment within
	 * OLT-Adapter:openolt_flowmgr.go
	 *     Metadata 8 bytes:
	 *	   Most Significant 2 Bytes = Inner VLAN
	 *	   Next 2 Bytes = Tech Profile ID(TPID)
	 *	   Least Significant 4 Bytes = Port ID
	 *     Flow Metadata carries Tech-Profile (TP) ID and is mandatory in all
	 *     subscriber related flows.
	 */

	metadata := flow.GetMetadataFromWriteMetadataAction(apFlowItem)
	if metadata == 0 {
		logger.Debugw("FlowAdd invalid metadata - abort",
			log.Fields{"device-id": dh.deviceID})
		return errors.New("flowAdd invalid metadata")
	}
	loTpID := flow.GetTechProfileIDFromWriteMetaData(metadata)
	logger.Debugw("FlowAdd TechProfileId", log.Fields{"device-id": dh.deviceID, "TP-Id": loTpID})

	dh.getFlowOfbFields(apFlowItem, &loMatchVlan, &loAddPcp, &loIPProto)
	if loIPProto == 2 {
		// some workaround for TT workflow at proto == 2 (IGMP trap) -> ignore the flow
		// avoids installing invalid EVTOCD rule
		logger.Debugw("FlowAdd type IpProto 2: TT workaround: ignore flow",
			log.Fields{"device-id": dh.deviceID,
				"IpProto": strconv.FormatInt(int64(loIPProto), 16)})
		return nil
	}
	dh.getFlowActions(apFlowItem, &loSetPcp, &loSetVlan)

	if loSetVlan == uint16(of.OfpVlanId_OFPVID_NONE) && loMatchVlan != uint16(of.OfpVlanId_OFPVID_PRESENT) {
		logger.Errorw("FlowAdd aborted - SetVlanId undefined, but MatchVid set", log.Fields{
			"device-id": dh.deviceID, "UniPort": apUniPort.portNo,
			"set_vid":   strconv.FormatInt(int64(loSetVlan), 16),
			"match_vid": strconv.FormatInt(int64(loMatchVlan), 16)})
		//TODO!!: Use DeviceId within the error response to rwCore
		//  likewise also in other error response cases to calling components as requested in [VOL-3458]
		return errors.New("flowAdd Set/Match VlanId inconsistent")
	}
	if loSetVlan == uint16(of.OfpVlanId_OFPVID_NONE) && loMatchVlan == uint16(of.OfpVlanId_OFPVID_PRESENT) {
		logger.Debugw("FlowAdd vlan-any/copy", log.Fields{"device-id": dh.deviceID})
		loSetVlan = loMatchVlan //both 'transparent' (copy any)
	} else {
		//looks like OMCI value 4097 (copyFromOuter - for Uni double tagged) is not supported here
		if loSetVlan != uint16(of.OfpVlanId_OFPVID_PRESENT) {
			// not set to transparent
			loSetVlan &= 0x0FFF //mask VID bits as prerequisite for vlanConfigFsm
		}
		logger.Debugw("FlowAdd vlan-set", log.Fields{"device-id": dh.deviceID})
	}
	if _, exist := dh.UniVlanConfigFsmMap[apUniPort.uniID]; exist {
		logger.Errorw("FlowAdd aborted - FSM already running", log.Fields{
			"device-id": dh.deviceID, "UniPort": apUniPort.portNo})
		return errors.New("flowAdd FSM already running")
	}
	return dh.createVlanFilterFsm(apUniPort,
		loTpID, loMatchVlan, loSetVlan, loSetPcp, OmciVlanFilterDone)
}

// createVlanFilterFsm initializes and runs the VlanFilter FSM to transfer OMCI related VLAN config
func (dh *deviceHandler) createVlanFilterFsm(apUniPort *onuUniPort,
	aTpID uint16, aMatchVlan uint16, aSetVlan uint16, aSetPcp uint8, aDevEvent OnuDeviceEvent) error {
	chVlanFilterFsm := make(chan Message, 2048)

	pDevEntry := dh.getOnuDeviceEntry(true)
	if pDevEntry == nil {
		logger.Errorw("No valid OnuDevice -aborting", log.Fields{"device-id": dh.deviceID})
		return fmt.Errorf("no valid OnuDevice for device-id %x - aborting", dh.deviceID)
	}

	pVlanFilterFsm := NewUniVlanConfigFsm(dh, pDevEntry.PDevOmciCC, apUniPort, dh.pOnuTP,
		pDevEntry.pOnuDB, aTpID, aDevEvent, "UniVlanConfigFsm", dh.deviceID, chVlanFilterFsm,
		dh.pOpenOnuAc.AcceptIncrementalEvto, aMatchVlan, aSetVlan, aSetPcp)
	if pVlanFilterFsm != nil {
		dh.UniVlanConfigFsmMap[apUniPort.uniID] = pVlanFilterFsm
		pVlanFilterStatemachine := pVlanFilterFsm.pAdaptFsm.pFsm
		if pVlanFilterStatemachine != nil {
			if pVlanFilterStatemachine.Is(vlanStDisabled) {
				if err := pVlanFilterStatemachine.Event(vlanEvStart); err != nil {
					logger.Warnw("UniVlanConfigFsm: can't start", log.Fields{"err": err})
					return fmt.Errorf("can't start UniVlanConfigFsm for device-id %x", dh.deviceID)
				}
				/***** UniVlanConfigFsm started */
				logger.Debugw("UniVlanConfigFsm started", log.Fields{
					"state": pVlanFilterStatemachine.Current(), "device-id": dh.deviceID,
					"UniPort": apUniPort.portNo})
			} else {
				logger.Warnw("wrong state of UniVlanConfigFsm - want: disabled", log.Fields{
					"have": pVlanFilterStatemachine.Current(), "device-id": dh.deviceID})
				return fmt.Errorf("uniVlanConfigFsm not in expected disabled state for device-id %x", dh.deviceID)
			}
		} else {
			logger.Errorw("UniVlanConfigFsm StateMachine invalid - cannot be executed!!", log.Fields{
				"device-id": dh.deviceID})
			return fmt.Errorf("uniVlanConfigFsm invalid for device-id %x", dh.deviceID)
		}
	} else {
		logger.Errorw("UniVlanConfigFsm could not be created - abort!!", log.Fields{
			"device-id": dh.deviceID, "UniPort": apUniPort.portNo})
		return fmt.Errorf("uniVlanConfigFsm could not be created for device-id %x", dh.deviceID)
	}
	return nil
}

//verifyUniVlanConfigRequest checks on existence of flow configuration and starts it accordingly
func (dh *deviceHandler) verifyUniVlanConfigRequest(apUniPort *onuUniPort) {
	if pVlanFilterFsm, exist := dh.UniVlanConfigFsmMap[apUniPort.uniID]; exist {
		//VlanFilterFsm exists and was already started (assumed to wait for TechProfile execution here)
		pVlanFilterStatemachine := pVlanFilterFsm.pAdaptFsm.pFsm
		if pVlanFilterStatemachine != nil {
			if pVlanFilterStatemachine.Is(vlanStWaitingTechProf) {
				if err := pVlanFilterStatemachine.Event(vlanEvContinueConfig); err != nil {
					logger.Warnw("UniVlanConfigFsm: can't continue processing", log.Fields{"err": err})
				} else {
					/***** UniVlanConfigFsm continued */
					logger.Debugw("UniVlanConfigFsm continued", log.Fields{
						"state": pVlanFilterStatemachine.Current(), "device-id": dh.deviceID,
						"UniPort": apUniPort.portNo})
				}
			} else {
				logger.Debugw("no state of UniVlanConfigFsm to be continued", log.Fields{
					"have": pVlanFilterStatemachine.Current(), "device-id": dh.deviceID})
			}
		} else {
			logger.Debugw("UniVlanConfigFsm StateMachine does not exist, no flow processing", log.Fields{
				"device-id": dh.deviceID})
		}

	} // else: nothing to do
}

//RemoveVlanFilterFsm deletes the stored pointer to the VlanConfigFsm
// intention is to provide this method to be called from VlanConfigFsm itself, when resources (and methods!) are cleaned up
func (dh *deviceHandler) RemoveVlanFilterFsm(apUniPort *onuUniPort) {
	logger.Debugw("remove UniVlanConfigFsm StateMachine", log.Fields{
		"device-id": dh.deviceID, "uniPort": apUniPort.portNo})
	delete(dh.UniVlanConfigFsmMap, apUniPort.uniID)
}
