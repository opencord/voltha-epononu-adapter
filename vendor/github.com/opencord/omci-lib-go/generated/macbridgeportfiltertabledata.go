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

// MacBridgePortFilterTableDataClassID is the 16-bit ID for the OMCI
// Managed entity MAC bridge port filter table data
const MacBridgePortFilterTableDataClassID ClassID = ClassID(49)

var macbridgeportfiltertabledataBME *ManagedEntityDefinition

// MacBridgePortFilterTableData (class ID #49)
//	This ME organizes data associated with a bridge port. The ONU automatically creates or deletes
//	an instance of this ME upon the creation or deletion of a MAC bridge port configuration data ME.
//
//	NOTE - The OLT should disable the learning mode in the MAC bridge service profile before writing
//	to the MAC filter table.
//
//	Relationships
//		An instance of this ME is associated with an instance of a MAC bridge port configuration data
//		ME.
//
//	Attributes
//		Managed Entity Id
//			Managed entity ID: This attribute uniquely identifies each instance of this ME. Through an
//			identical ID, this ME is implicitly linked to an instance of the MAC bridge port configuration
//			data ME. (R) (mandatory) (2-bytes)
//
//		Mac Filter Table
//			(R,-W) (Mandatory) (8N bytes, where N is the number of entries in the list)
//
type MacBridgePortFilterTableData struct {
	ManagedEntityDefinition
	Attributes AttributeValueMap
}

func init() {
	macbridgeportfiltertabledataBME = &ManagedEntityDefinition{
		Name:    "MacBridgePortFilterTableData",
		ClassID: 49,
		MessageTypes: mapset.NewSetWith(
			Get,
			GetNext,
			Set,
		),
		AllowedAttributeMask: 0x8000,
		AttributeDefinitions: AttributeDefinitionMap{
			0: Uint16Field("ManagedEntityId", PointerAttributeType, 0x0000, 0, mapset.NewSetWith(Read), false, false, false, 0),
			1: TableField("MacFilterTable", TableAttributeType, 0x8000, TableInfo{nil, 8}, mapset.NewSetWith(Read, Write), false, false, false, 1),
		},
		Access:  CreatedByOnu,
		Support: UnknownSupport,
	}
}

// NewMacBridgePortFilterTableData (class ID 49) creates the basic
// Managed Entity definition that is used to validate an ME of this type that
// is received from or transmitted to the OMCC.
func NewMacBridgePortFilterTableData(params ...ParamData) (*ManagedEntity, OmciErrors) {
	return NewManagedEntity(*macbridgeportfiltertabledataBME, params...)
}
