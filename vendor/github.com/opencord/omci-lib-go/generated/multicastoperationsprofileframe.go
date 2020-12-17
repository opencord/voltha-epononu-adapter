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

const MulticastOperationsProfileClassID ClassID = ClassID(309)

var multicastoperationsprofileME *ManagedEntityDefinition

type MulticastOperationsProfile struct{
	ManagedEntityDefinition
	Attributes AttributeValueMap
}

func init(){
	multicastoperationsprofileME = &ManagedEntityDefinition{
		Name: "MulticastOperationsProfile",
		ClassID: 309,
		MessageTypes: mapset.NewSetWith(
			Create,
			Delete,
			Get,
			Set,
			GetNext,
			),
		AllowedAttributeMask: 0xffff,
		AttributeDefinitions: AttributeDefinitionMap{
			0: Uint16Field("ManagedEntityId", PointerAttributeType, 0x0000, 0, mapset.NewSetWith(Read, SetByCreate), false,false,false,0),
			1: ByteField("IgmpVersion", EnumerationAttributeType, 0x8000, 0, mapset.NewSetWith(Read, SetByCreate, Write), false, false, false, 1),
			2: ByteField("IgmpFunction", EnumerationAttributeType, 0x4000, 0, mapset.NewSetWith(Read, SetByCreate, Write), false, false, false, 2),
			3: ByteField("ImmediateLeave", EnumerationAttributeType, 0x2000, 0, mapset.NewSetWith(Read, SetByCreate, Write), false, false, false, 3),
			4: Uint16Field("USIgmpTci", PointerAttributeType, 0x1000, 0, mapset.NewSetWith(Read, SetByCreate, Write), false, true, false, 4),
            5: ByteField("USIgmpTagCtrl", EnumerationAttributeType, 0x0800, 0, mapset.NewSetWith(Read, SetByCreate, Write), false, true, false, 5),
			6: Uint32Field("USIgmpRate", CounterAttributeType, 0x0400, 0, mapset.NewSetWith(Read), false, true, false, 6),
			7: MultiByteField("DynamicAccessControlListTable", StringAttributeType, 0x0200, 24, toOctets("AAAAAAAAAAAAAAAAAAAAAAAAAAA="), mapset.NewSetWith(Read), false, false, false, 7),
			8: MultiByteField("StaticAccessControlListTable", StringAttributeType, 0x0100, 24, toOctets("AAAAAAAAAAAAAAAAAAAAAAAAAAA="), mapset.NewSetWith(Read), false, true, false, 8),
			9: MultiByteField("LostGroupsListTable", StringAttributeType, 0x0080, 10, toOctets("AAAAAAAAAAAAAAAAAAAAAAAAAAA="), mapset.NewSetWith(Read), false, true, false, 9),
			10: ByteField("Robustness", EnumerationAttributeType, 0x0040, 0, mapset.NewSetWith(Read, SetByCreate, Write), false, false, false, 10),
			11: Uint32Field("QuerierIp", CounterAttributeType, 0x0020, 0, mapset.NewSetWith(Read), false, false, false, 11),
			12: Uint32Field("QueryInterval", CounterAttributeType, 0x0010, 0, mapset.NewSetWith(Read), false, false, false, 12),
			13: Uint32Field("QuerierMaxResponseTime", CounterAttributeType, 0x0008, 0, mapset.NewSetWith(Read), false, false, false, 13),
			14: Uint32Field("LastMemberResponseTime", CounterAttributeType, 0x0004, 0, mapset.NewSetWith(Read), false, false, false, 14),
			15: ByteField("UnauthorizedJoinBehaviour", EnumerationAttributeType, 0x0002, 0, mapset.NewSetWith(Read, SetByCreate, Write), false, false, false, 15),
			16: MultiByteField("DSIgmpMcastTci", StringAttributeType, 0x0001, 10, toOctets("AAAAAAAAAAAAAAAAAAAAAAAAAAA="), mapset.NewSetWith(Read), false, true, false, 16),
		},
		Access: CreatedByOlt,
		Support: UnknownSupport,
	}
}

// NewMulticastOperationsProfilePoint (class ID 309) creates the basic
// Managed Entity definition that is used to validate an ME of this type that
// is received from or transmitted to the OMCC.
func NewMulticastOperationsProfile(params ...ParamData) (*ManagedEntity, OmciErrors){
	return NewManagedEntity(*multicastoperationsprofileME, params...)
}