/*
 * Copyright 2020-present Open Networking Foundation

 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at

 * http://www.apache.org/licenses/LICENSE-2.0

 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package adaptercoreonu

import (
	"context"
	"encoding/json"

	"github.com/opencord/voltha-lib-go/v3/pkg/log"
	ic "github.com/opencord/voltha-protos/v3/go/inter_container"
	oop "github.com/opencord/voltha-protos/v3/go/openolt"
)

//AdditionalMessage ...
type AdditionalMessage struct {
}

/*
func (dh *deviceHandler) receivedMsgFromOlt(msg *ic.InterAdapterMessage) error {
	msgBody := msg.GetBody()
	ind := &oop.OnuIndication{}

	err := ptypes.UnmarshalAny(msgBody, ind)
	if err != nil {
		logger.Debugw("cannot-unmarshal-onu-indication-body", log.Fields{"error": err})
	} else {
		var info AdditionalMessage
		err = json.Unmarshal(ind.SerialNumber.VendorSpecific, &info)
		if err != nil {
			logger.Debugw("cannot-unmarshal-additional-message", log.Fields{"error": err})
		}
	}

	return err
}
*/
func (dh *deviceHandler) sendMsgToOlt(ctx context.Context, msg string, option interface{}) error {
	bytes, _ := json.Marshal(option)
	info := oop.OnuIndication{
		SerialNumber: &oop.SerialNumber{
			VendorSpecific: bytes,
		},
	}

	err := dh.AdapterProxy.SendInterAdapterMessage(ctx, &info,
		ic.InterAdapterMessageType_ONU_IND_REQUEST,
		dh.DeviceType, dh.ProxyAddressType,
		dh.deviceID, dh.parentID, msg)

	if err != nil {
		logger.Debugw("err-sending-message", log.Fields{"device": dh})
	}
	return err
}
