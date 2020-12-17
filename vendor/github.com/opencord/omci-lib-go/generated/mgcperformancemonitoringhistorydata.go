/*
 * Copyright (c) 2018 - present.  Boling Consulting Solutions (bcsw.net)
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
 /*
 * NOTE: This file was generated, manual edits will be overwritten!
 *
 * Generated by 'goCodeGenerator.py':
 *              https://github.com/cboling/OMCI-parser/README.md
 */

package generated

import "github.com/deckarep/golang-set"

// MgcPerformanceMonitoringHistoryDataClassID is the 16-bit ID for the OMCI
// Managed entity MGC performance monitoring history data
const MgcPerformanceMonitoringHistoryDataClassID ClassID = ClassID(156)

var mgcperformancemonitoringhistorydataBME *ManagedEntityDefinition

// MgcPerformanceMonitoringHistoryData (class ID #156)
//	The MGC monitoring data ME provides run-time statistics for an active MGC association. Instances
//	of this ME are created and deleted by the OLT.
//
//	For a complete discussion of generic PM architecture, refer to clause I.4.
//
//	Relationships
//		An instance of this ME is associated with an instance of the MGC config data or MGC config
//		portal ME.
//
//	Attributes
//		Managed Entity Id
//			Managed entity ID: This attribute uniquely identifies each instance of this ME. Through an
//			identical ID, this ME is implicitly linked to an instance of the associated MGC config data or
//			to the MGC config portal ME. If a non-OMCI configuration method is used for VoIP, there can be
//			only one live ME instance, associated with the MGC config portal, and with ME ID 0. (R,
//			setbycreate) (mandatory) (2-bytes)
//
//		Interval End Time
//			Interval end time: This attribute identifies the most recently finished 15-min interval. (R)
//			(mandatory) (1-byte)
//
//		Threshold Data 1_2 Id
//			Threshold data 1/2 ID: This attribute points to an instance of the threshold data 1 ME that
//			contains PM threshold values. Since no threshold value attribute number exceeds 7, a threshold
//			data 2 ME is optional. (R,-W, setbycreate) (mandatory) (2-bytes)
//
//		Received Messages
//			Received messages: This attribute counts the number of received Megaco messages on this
//			association, as defined by [ITUT H.341]. (R) (mandatory) (4-bytes)
//
//		Received Octets
//			Received octets: This attribute counts the total number of octets received on this association,
//			as defined by [ITU-T H.341]. (R) (mandatory) (4-bytes)
//
//		Sent Messages
//			Sent messages: This attribute counts the total number of Megaco messages sent over this
//			association, as defined by [ITU-T H.341]. (R) (mandatory) (4-bytes)
//
//		Sent Octets
//			Sent octets:	This attribute counts the total number of octets sent over this association, as
//			defined by [ITU-T H.341]. (R) (mandatory) (4-bytes)
//
//		Protocol Errors
//			(R) (mandatory) (4-bytes)
//
//		Transport Losses
//			Transport losses: This attribute counts the total number of transport losses (e.g., socket
//			problems) detected on this association. A link loss is defined as loss of communication with the
//			remote entity due to hardware/transient problems, or problems in related software. (R)
//			(mandatory) (4-bytes)
//
//		Last Detected Event
//			(R) (mandatory) (1-byte)
//
//		Last Detected Event Time
//			Last detected event time: This attribute reports the time in seconds since the last event on
//			this association was detected, as defined by [ITU-T H.341]. (R) (mandatory) (4-bytes)
//
//		Last Detected Reset Time
//			Last detected reset time: This attribute reports the time in seconds since these statistics were
//			last reset, as defined by [ITU-T H.341]. Under normal circumstances, a get action on this
//			attribute would return 900-s to indicate a completed 15-min interval. (R) (mandatory) (4-bytes)
//
type MgcPerformanceMonitoringHistoryData struct {
	ManagedEntityDefinition
	Attributes AttributeValueMap
}

func init() {
	mgcperformancemonitoringhistorydataBME = &ManagedEntityDefinition{
		Name:    "MgcPerformanceMonitoringHistoryData",
		ClassID: 156,
		MessageTypes: mapset.NewSetWith(
			Create,
			Delete,
			Get,
			Set,
		),
		AllowedAttributeMask: 0xffe0,
		AttributeDefinitions: AttributeDefinitionMap{
			0:  Uint16Field("ManagedEntityId", PointerAttributeType, 0x0000, 0, mapset.NewSetWith(Read, SetByCreate), false, false, false, 0),
			1:  ByteField("IntervalEndTime", UnsignedIntegerAttributeType, 0x8000, 0, mapset.NewSetWith(Read), false, false, false, 1),
			2:  Uint16Field("ThresholdData12Id", UnsignedIntegerAttributeType, 0x4000, 0, mapset.NewSetWith(Read, SetByCreate, Write), false, false, false, 2),
			3:  Uint32Field("ReceivedMessages", CounterAttributeType, 0x2000, 0, mapset.NewSetWith(Read), false, false, false, 3),
			4:  Uint32Field("ReceivedOctets", CounterAttributeType, 0x1000, 0, mapset.NewSetWith(Read), false, false, false, 4),
			5:  Uint32Field("SentMessages", CounterAttributeType, 0x0800, 0, mapset.NewSetWith(Read), false, false, false, 5),
			6:  Uint32Field("SentOctets", CounterAttributeType, 0x0400, 0, mapset.NewSetWith(Read), false, false, false, 6),
			7:  Uint32Field("ProtocolErrors", CounterAttributeType, 0x0200, 0, mapset.NewSetWith(Read), false, false, false, 7),
			8:  Uint32Field("TransportLosses", CounterAttributeType, 0x0100, 0, mapset.NewSetWith(Read), false, false, false, 8),
			9:  ByteField("LastDetectedEvent", CounterAttributeType, 0x0080, 0, mapset.NewSetWith(Read), false, false, false, 9),
			10: Uint32Field("LastDetectedEventTime", CounterAttributeType, 0x0040, 0, mapset.NewSetWith(Read), false, false, false, 10),
			11: Uint32Field("LastDetectedResetTime", CounterAttributeType, 0x0020, 0, mapset.NewSetWith(Read), false, false, false, 11),
		},
		Access:  CreatedByOlt,
		Support: UnknownSupport,
	}
}

// NewMgcPerformanceMonitoringHistoryData (class ID 156) creates the basic
// Managed Entity definition that is used to validate an ME of this type that
// is received from or transmitted to the OMCC.
func NewMgcPerformanceMonitoringHistoryData(params ...ParamData) (*ManagedEntity, OmciErrors) {
	return NewManagedEntity(*mgcperformancemonitoringhistorydataBME, params...)
}
