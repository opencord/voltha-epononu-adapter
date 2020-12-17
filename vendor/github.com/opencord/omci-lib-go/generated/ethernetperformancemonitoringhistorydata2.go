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

// EthernetPerformanceMonitoringHistoryData2ClassID is the 16-bit ID for the OMCI
// Managed entity Ethernet performance monitoring history data 2
const EthernetPerformanceMonitoringHistoryData2ClassID ClassID = ClassID(89)

var ethernetperformancemonitoringhistorydata2BME *ManagedEntityDefinition

// EthernetPerformanceMonitoringHistoryData2 (class ID #89)
//	This ME collects additional PM data for a physical Ethernet interface. Instances of this ME are
//	created and deleted by the OLT.
//
//	For a complete discussion of generic PM architecture, refer to clause I.4.
//
//	Relationships
//		An instance of this Ethernet PM history data 2 ME is associated with an instance of the PPTP
//		Ethernet UNI.
//
//	Attributes
//		Managed Entity Id
//			Managed entity ID: This attribute uniquely identifies each instance of this ME. Through an
//			identical ID, this ME is implicitly linked to an instance of the PPTP Ethernet UNI. (R,
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
//		Pppoe Filtered Frame Counter
//			PPPoE filtered frame counter: This attribute counts the number of frames discarded due to PPPoE
//			filtering. (R) (mandatory) (4-bytes)
//
type EthernetPerformanceMonitoringHistoryData2 struct {
	ManagedEntityDefinition
	Attributes AttributeValueMap
}

func init() {
	ethernetperformancemonitoringhistorydata2BME = &ManagedEntityDefinition{
		Name:    "EthernetPerformanceMonitoringHistoryData2",
		ClassID: 89,
		MessageTypes: mapset.NewSetWith(
			Create,
			Delete,
			Get,
			Set,
		),
		AllowedAttributeMask: 0xe000,
		AttributeDefinitions: AttributeDefinitionMap{
			0: Uint16Field("ManagedEntityId", PointerAttributeType, 0x0000, 0, mapset.NewSetWith(Read, SetByCreate), false, false, false, 0),
			1: ByteField("IntervalEndTime", UnsignedIntegerAttributeType, 0x8000, 0, mapset.NewSetWith(Read), false, false, false, 1),
			2: Uint16Field("ThresholdData12Id", PointerAttributeType, 0x4000, 0, mapset.NewSetWith(Read, SetByCreate, Write), false, false, false, 2),
			3: Uint32Field("PppoeFilteredFrameCounter", CounterAttributeType, 0x2000, 0, mapset.NewSetWith(Read), false, false, false, 3),
		},
		Access:  CreatedByOlt,
		Support: UnknownSupport,
	}
}

// NewEthernetPerformanceMonitoringHistoryData2 (class ID 89) creates the basic
// Managed Entity definition that is used to validate an ME of this type that
// is received from or transmitted to the OMCC.
func NewEthernetPerformanceMonitoringHistoryData2(params ...ParamData) (*ManagedEntity, OmciErrors) {
	return NewManagedEntity(*ethernetperformancemonitoringhistorydata2BME, params...)
}
