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
	"fmt"
	"time"

	"github.com/opencord/voltha-lib-go/v3/pkg/adapters/adapterif"
	"github.com/opencord/voltha-protos/v3/go/common"
)

var startWatchStatus bool = false

//WatchStatus ...
func WatchStatus(ctx context.Context, cp adapterif.CoreProxy) {
	if startWatchStatus {
		return
	}
	startWatchStatus = true
	logger.Debug(ctx, fmt.Sprintf("Start WatchStatus()"))
	ticker := time.NewTicker(1000 * time.Millisecond)
	prevOnuList := make(map[string]OnuStatus)
	for range ticker.C {
		onuList, err := ReadOnuStatusList()
		if err == nil {
			for _, onu := range onuList {
				current, ok := prevOnuList[onu.MacAddress]
				if !ok || onu.OpeState != current.OpeState || onu.ConnectState != current.ConnectState {
					con := common.ConnectStatus_Types(common.ConnectStatus_Types_value[onu.ConnectState])
					ope := common.OperStatus_Types(common.OperStatus_Types_value[onu.OpeState])
					err2 := cp.DeviceStateUpdate(ctx, onu.ID, con, ope)
					logger.Debug(ctx, fmt.Sprintf("WatchStatus() update: %s, OpeState: %v, ConnectState: %v, error: %v",
						onu.ID, onu.OpeState, onu.ConnectState, err2))
					prevOnuList[onu.MacAddress] = onu
				}
			}
		}
	}
}
