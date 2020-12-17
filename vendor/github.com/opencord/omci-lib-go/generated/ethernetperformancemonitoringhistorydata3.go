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

// EthernetPerformanceMonitoringHistoryData3ClassID is the 16-bit ID for the OMCI
// Managed entity Ethernet performance monitoring history data 3
const EthernetPerformanceMonitoringHistoryData3ClassID ClassID = ClassID(296)

var ethernetperformancemonitoringhistorydata3BME *ManagedEntityDefinition

// EthernetPerformanceMonitoringHistoryData3 (class ID #296)
//	This ME collects PM data associated with an Ethernet interface. It includes parameters defined
//	in the Ethernet statistics group of [IETF RFC 2819] that are not already covered by previously
//	defined Ethernet monitoring MEs. The received direction is from the CPE towards the network
//	(upstream).
//
//	NOTE 1 - Several of the same counters are available from the Ethernet frame PM history data MEs,
//	which are associated with MAC bridge ports. MAC bridge port association allows those MEs to be
//	used for any Ethernet flow, in both upstream and downstream directions, while the Ethernet PM
//	history data 3 ME can only be used on a physical IEEE 802.3 port.
//
//	Instances of this ME are created and deleted by the OLT.
//
//	For a complete discussion of generic PM architecture, refer to clause I.4.
//
//	NOTE 2 - Implementers are encouraged to consider the Ethernet frame extended PM ME defined in
//	clause-9.3.32, which collects the same counters in a more generalized way.
//
//	Relationships
//		An instance of this ME is associated with an instance of the PPTP Ethernet UNI.
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
//		Drop Events
//			Drop events:	The total number of events in which packets were dropped due to a lack of
//			resources. This is not necessarily the number of packets dropped; it is the number of times this
//			event was detected. (R) (mandatory) (4-bytes)
//
//		Octets
//			Octets:	The total number of octets received from the CPE, including those in bad packets,
//			excluding framing bytes, but including FCS. (R) (mandatory) (4-bytes)
//
//		Packets
//			Packets:	The total number of packets received, including bad packets, broadcast packets and
//			multicast packets. (R) (mandatory) (4-bytes)
//
//		Broadcast Packets
//			Broadcast packets: The total number of received good packets directed to the broadcast address.
//			This does not include multicast packets. (R) (mandatory) (4-bytes)
//
//		Multicast Packets
//			Multicast packets: The total number of received good packets directed to a multicast address.
//			This does not include broadcast packets. (R) (mandatory) (4-bytes)
//
//		Undersize Packets
//			Undersize packets: The total number of packets received that were less than 64 octets long, but
//			were otherwise well formed (excluding framing bits, but including FCS). (R) (mandatory)
//			(4-bytes)
//
//		Fragments
//			Fragments:	The total number of packets received that were less than 64 octets long, excluding
//			framing bits but including FCS octets, and had either a bad FCS with an integral number of
//			octets (FCS error) or a bad FCS with a non-integral number of octets (alignment error). It is
//			entirely normal for this attribute to increment. This is because it counts both runts (which are
//			normal occurrences due to collisions) and noise hits. (R) (mandatory) (4-bytes)
//
//		Jabbers
//			Jabbers:	The total number of packets received that were longer than 1518 octets, excluding
//			framing bits but including FCS octets, and had either a bad FCS with an integral number of
//			octets (FCS error) or a bad FCS with a non-integral number of octets (alignment error). The
//			range to detect jabber is between 20 ms and 150 ms. (R) (mandatory) (4-bytes)
//
//		Packets 64 Octets
//			Packets 64 octets: The total number of received packets (including bad packets) that were
//			64-octets long, excluding framing bits but including FCS. (R) (mandatory) (4-bytes)
//
//		Packets 65 To 127 Octets
//			Packets 65 to 127 octets: The total number of received packets (including bad packets) that were
//			65..127 octets long, excluding framing bits but including FCS. (R) (mandatory) (4-bytes)
//
//		Packets 128 To 255 Octets
//			Packets 128 to 255 octets: The total number of packets (including bad packets) received that
//			were 128..255 octets long, excluding framing bits but including FCS. (R) (mandatory) (4-bytes)
//
//		Packets 256 To 511 Octets
//			Packets 256 to 511 octets: The total number of packets (including bad packets) received that
//			were 256..511 octets long, excluding framing bits but including FCS. (R) (mandatory) (4-bytes)
//
//		Packets 512 To 1023 Octets
//			Packets 512 to 1023 octets: The total number of packets (including bad packets) received that
//			were 512..1023 octets long, excluding framing bits but including FCS. (R) (mandatory) (4-bytes)
//
//		Packets 1024 To 1518 Octets
//			Packets 1024 to 1518 octets: The total number of packets (including bad packets) received that
//			were 1024..1518 octets long, excluding framing bits but including FCS. (R) (mandatory) (4-bytes)
//
type EthernetPerformanceMonitoringHistoryData3 struct {
	ManagedEntityDefinition
	Attributes AttributeValueMap
}

func init() {
	ethernetperformancemonitoringhistorydata3BME = &ManagedEntityDefinition{
		Name:    "EthernetPerformanceMonitoringHistoryData3",
		ClassID: 296,
		MessageTypes: mapset.NewSetWith(
			Create,
			Delete,
			Get,
			Set,
		),
		AllowedAttributeMask: 0xffff,
		AttributeDefinitions: AttributeDefinitionMap{
			0:  Uint16Field("ManagedEntityId", PointerAttributeType, 0x0000, 0, mapset.NewSetWith(Read, SetByCreate), false, false, false, 0),
			1:  ByteField("IntervalEndTime", UnsignedIntegerAttributeType, 0x8000, 0, mapset.NewSetWith(Read), false, false, false, 1),
			2:  Uint16Field("ThresholdData12Id", PointerAttributeType, 0x4000, 0, mapset.NewSetWith(Read, SetByCreate, Write), false, false, false, 2),
			3:  Uint32Field("DropEvents", CounterAttributeType, 0x2000, 0, mapset.NewSetWith(Read), false, false, false, 3),
			4:  Uint32Field("Octets", CounterAttributeType, 0x1000, 0, mapset.NewSetWith(Read), false, false, false, 4),
			5:  Uint32Field("Packets", CounterAttributeType, 0x0800, 0, mapset.NewSetWith(Read), false, false, false, 5),
			6:  Uint32Field("BroadcastPackets", CounterAttributeType, 0x0400, 0, mapset.NewSetWith(Read), false, false, false, 6),
			7:  Uint32Field("MulticastPackets", CounterAttributeType, 0x0200, 0, mapset.NewSetWith(Read), false, false, false, 7),
			8:  Uint32Field("UndersizePackets", CounterAttributeType, 0x0100, 0, mapset.NewSetWith(Read), false, false, false, 8),
			9:  Uint32Field("Fragments", CounterAttributeType, 0x0080, 0, mapset.NewSetWith(Read), false, false, false, 9),
			10: Uint32Field("Jabbers", CounterAttributeType, 0x0040, 0, mapset.NewSetWith(Read), false, false, false, 10),
			11: Uint32Field("Packets64Octets", CounterAttributeType, 0x0020, 0, mapset.NewSetWith(Read), false, false, false, 11),
			12: Uint32Field("Packets65To127Octets", CounterAttributeType, 0x0010, 0, mapset.NewSetWith(Read), false, false, false, 12),
			13: Uint32Field("Packets128To255Octets", CounterAttributeType, 0x0008, 0, mapset.NewSetWith(Read), false, false, false, 13),
			14: Uint32Field("Packets256To511Octets", CounterAttributeType, 0x0004, 0, mapset.NewSetWith(Read), false, false, false, 14),
			15: Uint32Field("Packets512To1023Octets", CounterAttributeType, 0x0002, 0, mapset.NewSetWith(Read), false, false, false, 15),
			16: Uint32Field("Packets1024To1518Octets", CounterAttributeType, 0x0001, 0, mapset.NewSetWith(Read), false, false, false, 16),
		},
		Access:  CreatedByOlt,
		Support: UnknownSupport,
	}
}

// NewEthernetPerformanceMonitoringHistoryData3 (class ID 296) creates the basic
// Managed Entity definition that is used to validate an ME of this type that
// is received from or transmitted to the OMCC.
func NewEthernetPerformanceMonitoringHistoryData3(params ...ParamData) (*ManagedEntity, OmciErrors) {
	return NewManagedEntity(*ethernetperformancemonitoringhistorydata3BME, params...)
}
