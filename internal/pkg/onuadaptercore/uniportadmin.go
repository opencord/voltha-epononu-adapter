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
	"time"

	"github.com/looplab/fsm"

	"github.com/opencord/omci-lib-go"
	me "github.com/opencord/omci-lib-go/generated"
	"github.com/opencord/voltha-lib-go/v3/pkg/log"
)

//lockStateFsm defines the structure for the state machine to lock/unlock the ONU UNI ports via OMCI
type lockStateFsm struct {
	pOmciCC                  *omciCC
	adminState               bool
	requestEvent             OnuDeviceEvent
	omciLockResponseReceived chan bool //seperate channel needed for checking UNI port OMCi message responses
	pAdaptFsm                *AdapterFsm
}

const (
	uniEvStart         = "uniEvStart"
	uniEvStartAdmin    = "uniEvStartAdmin"
	uniEvRxUnisResp    = "uniEvRxUnisResp"
	uniEvRxOnugResp    = "uniEvRxOnugResp"
	uniEvTimeoutSimple = "uniEvTimeoutSimple"
	uniEvTimeoutUnis   = "uniEvTimeoutUnis"
	uniEvReset         = "uniEvReset"
	uniEvRestart       = "uniEvRestart"
)
const (
	uniStDisabled    = "uniStDisabled"
	uniStStarting    = "uniStStarting"
	uniStSettingUnis = "uniStSettingUnis"
	uniStSettingOnuG = "uniStSettingOnuG"
	uniStAdminDone   = "uniStAdminDone"
	uniStResetting   = "uniStResetting"
)

//newLockStateFsm is the 'constructor' for the state machine to lock/unlock the ONU UNI ports via OMCI
func newLockStateFsm(apDevOmciCC *omciCC, aAdminState bool, aRequestEvent OnuDeviceEvent,
	aName string, aDeviceID string, aCommChannel chan Message) *lockStateFsm {
	instFsm := &lockStateFsm{
		pOmciCC:      apDevOmciCC,
		adminState:   aAdminState,
		requestEvent: aRequestEvent,
	}
	instFsm.pAdaptFsm = NewAdapterFsm(aName, aDeviceID, aCommChannel)
	if instFsm.pAdaptFsm == nil {
		logger.Errorw("LockStateFsm's AdapterFsm could not be instantiated!!", log.Fields{
			"device-id": aDeviceID})
		return nil
	}
	if aAdminState { //port locking requested
		instFsm.pAdaptFsm.pFsm = fsm.NewFSM(
			uniStDisabled,
			fsm.Events{

				{Name: uniEvStart, Src: []string{uniStDisabled}, Dst: uniStStarting},

				{Name: uniEvStartAdmin, Src: []string{uniStStarting}, Dst: uniStSettingUnis},
				// the settingUnis state is used for multi ME config for all UNI related ports
				// maybe such could be reflected in the state machine as well (port number parametrized)
				// but that looks not straightforward here - so we keep it simple here for the beginning(?)
				{Name: uniEvRxUnisResp, Src: []string{uniStSettingUnis}, Dst: uniStSettingOnuG},
				{Name: uniEvRxOnugResp, Src: []string{uniStSettingOnuG}, Dst: uniStAdminDone},

				{Name: uniEvTimeoutSimple, Src: []string{uniStSettingOnuG}, Dst: uniStStarting},
				{Name: uniEvTimeoutUnis, Src: []string{uniStSettingUnis}, Dst: uniStStarting},

				{Name: uniEvReset, Src: []string{uniStStarting, uniStSettingOnuG, uniStSettingUnis,
					uniStAdminDone}, Dst: uniStResetting},
				// exceptional treatment for all states except uniStResetting
				{Name: uniEvRestart, Src: []string{uniStStarting, uniStSettingOnuG, uniStSettingUnis,
					uniStAdminDone, uniStResetting}, Dst: uniStDisabled},
			},

			fsm.Callbacks{
				"enter_state":                 func(e *fsm.Event) { instFsm.pAdaptFsm.logFsmStateChange(e) },
				("enter_" + uniStStarting):    func(e *fsm.Event) { instFsm.enterAdminStartingState(e) },
				("enter_" + uniStSettingOnuG): func(e *fsm.Event) { instFsm.enterSettingOnuGState(e) },
				("enter_" + uniStSettingUnis): func(e *fsm.Event) { instFsm.enterSettingUnisState(e) },
				("enter_" + uniStAdminDone):   func(e *fsm.Event) { instFsm.enterAdminDoneState(e) },
				("enter_" + uniStResetting):   func(e *fsm.Event) { instFsm.enterResettingState(e) },
			},
		)
	} else { //port unlocking requested
		instFsm.pAdaptFsm.pFsm = fsm.NewFSM(
			uniStDisabled,
			fsm.Events{

				{Name: uniEvStart, Src: []string{uniStDisabled}, Dst: uniStStarting},

				{Name: uniEvStartAdmin, Src: []string{uniStStarting}, Dst: uniStSettingOnuG},
				{Name: uniEvRxOnugResp, Src: []string{uniStSettingOnuG}, Dst: uniStSettingUnis},
				// the settingUnis state is used for multi ME config for all UNI related ports
				// maybe such could be reflected in the state machine as well (port number parametrized)
				// but that looks not straightforward here - so we keep it simple here for the beginning(?)
				{Name: uniEvRxUnisResp, Src: []string{uniStSettingUnis}, Dst: uniStAdminDone},

				{Name: uniEvTimeoutSimple, Src: []string{uniStSettingOnuG}, Dst: uniStStarting},
				{Name: uniEvTimeoutUnis, Src: []string{uniStSettingUnis}, Dst: uniStStarting},

				{Name: uniEvReset, Src: []string{uniStStarting, uniStSettingOnuG, uniStSettingUnis,
					uniStAdminDone}, Dst: uniStResetting},
				// exceptional treatment for all states except uniStResetting
				{Name: uniEvRestart, Src: []string{uniStStarting, uniStSettingOnuG, uniStSettingUnis,
					uniStAdminDone, uniStResetting}, Dst: uniStDisabled},
			},

			fsm.Callbacks{
				"enter_state":                 func(e *fsm.Event) { instFsm.pAdaptFsm.logFsmStateChange(e) },
				("enter_" + uniStStarting):    func(e *fsm.Event) { instFsm.enterAdminStartingState(e) },
				("enter_" + uniStSettingOnuG): func(e *fsm.Event) { instFsm.enterSettingOnuGState(e) },
				("enter_" + uniStSettingUnis): func(e *fsm.Event) { instFsm.enterSettingUnisState(e) },
				("enter_" + uniStAdminDone):   func(e *fsm.Event) { instFsm.enterAdminDoneState(e) },
				("enter_" + uniStResetting):   func(e *fsm.Event) { instFsm.enterResettingState(e) },
			},
		)
	}
	if instFsm.pAdaptFsm.pFsm == nil {
		logger.Errorw("LockStateFsm's Base FSM could not be instantiated!!", log.Fields{
			"device-id": aDeviceID})
		return nil
	}

	logger.Infow("LockStateFsm created", log.Fields{"device-id": aDeviceID})
	return instFsm
}

//setSuccessEvent modifies the requested event notified on success
//assumption is that this is only called in the disabled (idle) state of the FSM, hence no sem protection required
func (oFsm *lockStateFsm) setSuccessEvent(aEvent OnuDeviceEvent) {
	oFsm.requestEvent = aEvent
}

func (oFsm *lockStateFsm) enterAdminStartingState(e *fsm.Event) {
	logger.Debugw("LockStateFSM start", log.Fields{"in state": e.FSM.Current(),
		"device-id": oFsm.pAdaptFsm.deviceID})
	if oFsm.omciLockResponseReceived == nil {
		oFsm.omciLockResponseReceived = make(chan bool)
		logger.Debug("LockStateFSM - OMCI UniLock RxChannel defined")
	} else {
		// as we may 're-use' this instance of FSM and the connected channel
		// make sure there is no 'lingering' request in the already existing channel:
		// (simple loop sufficient as we are the only receiver)
		for len(oFsm.omciLockResponseReceived) > 0 {
			<-oFsm.omciLockResponseReceived
		}
	}
	go oFsm.processOmciLockMessages()

	pLockStateAFsm := oFsm.pAdaptFsm
	if pLockStateAFsm != nil {
		// obviously calling some FSM event here directly does not work - so trying to decouple it ...
		go func(a_pAFsm *AdapterFsm) {
			if a_pAFsm != nil && a_pAFsm.pFsm != nil {
				_ = a_pAFsm.pFsm.Event(uniEvStartAdmin)
			}
		}(pLockStateAFsm)
	}
}

func (oFsm *lockStateFsm) enterSettingOnuGState(e *fsm.Event) {
	var omciAdminState uint8 = 1 //default locked
	if !oFsm.adminState {
		omciAdminState = 0
	}
	logger.Debugw("LockStateFSM Tx Set::ONU-G:admin", log.Fields{
		"omciAdmin": omciAdminState, "in state": e.FSM.Current(), "device-id": oFsm.pAdaptFsm.deviceID})
	requestedAttributes := me.AttributeValueMap{"AdministrativeState": omciAdminState}
	meInstance := oFsm.pOmciCC.sendSetOnuGLS(context.TODO(), ConstDefaultOmciTimeout, true,
		requestedAttributes, oFsm.pAdaptFsm.commChan)
	oFsm.pOmciCC.pLastTxMeInstance = meInstance
}

func (oFsm *lockStateFsm) enterSettingUnisState(e *fsm.Event) {
	logger.Infow("LockStateFSM - starting PPTP config loop", log.Fields{
		"in state": e.FSM.Current(), "device-id": oFsm.pAdaptFsm.deviceID, "LockState": oFsm.adminState})
	go oFsm.performUniPortAdminSet()
}

func (oFsm *lockStateFsm) enterAdminDoneState(e *fsm.Event) {
	logger.Debugw("LockStateFSM", log.Fields{"send notification to core in State": e.FSM.Current(), "device-id": oFsm.pAdaptFsm.deviceID})
	oFsm.pOmciCC.pBaseDeviceHandler.deviceProcStatusUpdate(oFsm.requestEvent)
	pLockStateAFsm := oFsm.pAdaptFsm
	if pLockStateAFsm != nil {
		// obviously calling some FSM event here directly does not work - so trying to decouple it ...
		go func(a_pAFsm *AdapterFsm) {
			if a_pAFsm != nil && a_pAFsm.pFsm != nil {
				_ = a_pAFsm.pFsm.Event(uniEvReset)
			}
		}(pLockStateAFsm)
	}
}

func (oFsm *lockStateFsm) enterResettingState(e *fsm.Event) {
	logger.Debugw("LockStateFSM resetting", log.Fields{"device-id": oFsm.pAdaptFsm.deviceID})
	pLockStateAFsm := oFsm.pAdaptFsm
	if pLockStateAFsm != nil {
		// abort running message processing
		fsmAbortMsg := Message{
			Type: TestMsg,
			Data: TestMessage{
				TestMessageVal: AbortMessageProcessing,
			},
		}
		pLockStateAFsm.commChan <- fsmAbortMsg

		//try to restart the FSM to 'disabled'
		// see DownloadedState: decouple event transfer
		go func(a_pAFsm *AdapterFsm) {
			if a_pAFsm != nil && a_pAFsm.pFsm != nil {
				_ = a_pAFsm.pFsm.Event(uniEvRestart)
			}
		}(pLockStateAFsm)
	}
}

func (oFsm *lockStateFsm) processOmciLockMessages( /*ctx context.Context*/ ) {
	logger.Debugw("Start LockStateFsm Msg processing", log.Fields{"for device-id": oFsm.pAdaptFsm.deviceID})
loop:
	for {
		// case <-ctx.Done():
		// 	logger.Info("MibSync Msg", log.Fields{"Message handling canceled via context for device-id": oFsm.pAdaptFsm.deviceID})
		// 	break loop
		message, ok := <-oFsm.pAdaptFsm.commChan
		if !ok {
			logger.Info("LockStateFsm Rx Msg - could not read from channel", log.Fields{"device-id": oFsm.pAdaptFsm.deviceID})
			// but then we have to ensure a restart of the FSM as well - as exceptional procedure
			_ = oFsm.pAdaptFsm.pFsm.Event(uniEvRestart)
			break loop
		}
		logger.Debugw("LockStateFsm Rx Msg", log.Fields{"device-id": oFsm.pAdaptFsm.deviceID})

		switch message.Type {
		case TestMsg:
			msg, _ := message.Data.(TestMessage)
			if msg.TestMessageVal == AbortMessageProcessing {
				logger.Infow("LockStateFsm abort ProcessMsg", log.Fields{"for device-id": oFsm.pAdaptFsm.deviceID})
				break loop
			}
			logger.Warnw("LockStateFsm unknown TestMessage", log.Fields{"device-id": oFsm.pAdaptFsm.deviceID, "MessageVal": msg.TestMessageVal})
		case OMCI:
			msg, _ := message.Data.(OmciMessage)
			oFsm.handleOmciLockStateMessage(msg)
		default:
			logger.Warn("LockStateFsm Rx unknown message", log.Fields{"device-id": oFsm.pAdaptFsm.deviceID,
				"message.Type": message.Type})
		}
	}
	logger.Infow("End LockStateFsm Msg processing", log.Fields{"device-id": oFsm.pAdaptFsm.deviceID})
}

func (oFsm *lockStateFsm) handleOmciLockStateMessage(msg OmciMessage) {
	logger.Debugw("Rx OMCI LockStateFsm Msg", log.Fields{"device-id": oFsm.pAdaptFsm.deviceID,
		"msgType": msg.OmciMsg.MessageType})

	if msg.OmciMsg.MessageType == omci.SetResponseType {
		msgLayer := (*msg.OmciPacket).Layer(omci.LayerTypeSetResponse)
		if msgLayer == nil {
			logger.Error("LockStateFsm - Omci Msg layer could not be detected for SetResponse")
			return
		}
		msgObj, msgOk := msgLayer.(*omci.SetResponse)
		if !msgOk {
			logger.Error("LockStateFsm - Omci Msg layer could not be assigned for SetResponse")
			return
		}
		logger.Debugw("LockStateFsm SetResponse Data", log.Fields{"device-id": oFsm.pAdaptFsm.deviceID, "data-fields": msgObj})
		if msgObj.Result != me.Success {
			logger.Errorw("LockStateFsm - Omci SetResponse Error - later: drive FSM to abort state ?", log.Fields{"Error": msgObj.Result})
			// possibly force FSM into abort or ignore some errors for some messages? store error for mgmt display?
			return
		}
		// compare comments above for CreateResponse (apply also here ...)
		if msgObj.EntityClass == oFsm.pOmciCC.pLastTxMeInstance.GetClassID() &&
			msgObj.EntityInstance == oFsm.pOmciCC.pLastTxMeInstance.GetEntityID() {
			//store the created ME into DB //TODO??? obviously the Python code does not store the config ...
			// if, then something like:
			//oFsm.pOnuDB.StoreMe(msgObj)

			switch oFsm.pOmciCC.pLastTxMeInstance.GetName() {
			case "OnuG":
				{ // let the FSM proceed ...
					_ = oFsm.pAdaptFsm.pFsm.Event(uniEvRxOnugResp)
				}
			case "UniG", "VEIP":
				{ // let the PPTP init proceed by stopping the wait function
					oFsm.omciLockResponseReceived <- true
				}
			}
		}
	} else {
		logger.Errorw("LockStateFsm - Rx OMCI unhandled MsgType", log.Fields{"omciMsgType": msg.OmciMsg.MessageType})
		return
	}
}

func (oFsm *lockStateFsm) performUniPortAdminSet() {
	var omciAdminState uint8 = 1 //default locked
	if !oFsm.adminState {
		omciAdminState = 0
	}
	requestedAttributes := me.AttributeValueMap{"AdministrativeState": omciAdminState}

	for uniNo, uniPort := range oFsm.pOmciCC.pBaseDeviceHandler.uniEntityMap {
		logger.Debugw("Setting PPTP admin state", log.Fields{
			"device-id": oFsm.pAdaptFsm.deviceID, "for PortNo": uniNo})

		var meInstance *me.ManagedEntity
		if uniPort.portType == uniPPTP {
			meInstance = oFsm.pOmciCC.sendSetUniGLS(context.TODO(), uniPort.entityID, ConstDefaultOmciTimeout,
				true, requestedAttributes, oFsm.pAdaptFsm.commChan)
			oFsm.pOmciCC.pLastTxMeInstance = meInstance
		} else if uniPort.portType == uniVEIP {
			meInstance = oFsm.pOmciCC.sendSetVeipLS(context.TODO(), uniPort.entityID, ConstDefaultOmciTimeout,
				true, requestedAttributes, oFsm.pAdaptFsm.commChan)
			oFsm.pOmciCC.pLastTxMeInstance = meInstance
		} else {
			logger.Warnw("Unsupported PPTP type - skip",
				log.Fields{"device-id": oFsm.pAdaptFsm.deviceID, "Port": uniNo})
			continue
		}

		//verify response
		err := oFsm.waitforOmciResponse(meInstance)
		if err != nil {
			logger.Errorw("PPTP Admin State set failed, aborting LockState set!",
				log.Fields{"device-id": oFsm.pAdaptFsm.deviceID, "Port": uniNo})
			_ = oFsm.pAdaptFsm.pFsm.Event(uniEvReset)
			return
		}
	} //for all UNI ports
	logger.Infow("PPTP config loop finished", log.Fields{"device-id": oFsm.pAdaptFsm.deviceID})
	_ = oFsm.pAdaptFsm.pFsm.Event(uniEvRxUnisResp)
}

func (oFsm *lockStateFsm) waitforOmciResponse(apMeInstance *me.ManagedEntity) error {
	select {
	case <-time.After(30 * time.Second): //3s was detected to be to less in 8*8 bbsim test with debug Info/Debug
		logger.Warnw("LockStateFSM uni-set timeout", log.Fields{"for device-id": oFsm.pAdaptFsm.deviceID})
		return errors.New("lockStateFsm uni-set timeout")
	case success := <-oFsm.omciLockResponseReceived:
		if success {
			logger.Debug("LockStateFSM uni-set response received")
			return nil
		}
		// should not happen so far
		logger.Warnw("LockStateFSM uni-set response error", log.Fields{"for device-id": oFsm.pAdaptFsm.deviceID})
		return errors.New("lockStateFsm uni-set responseError")
	}
}
