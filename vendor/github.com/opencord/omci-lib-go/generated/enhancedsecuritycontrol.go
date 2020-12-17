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

// EnhancedSecurityControlClassID is the 16-bit ID for the OMCI
// Managed entity Enhanced security control
const EnhancedSecurityControlClassID ClassID = ClassID(332)

var enhancedsecuritycontrolBME *ManagedEntityDefinition

// EnhancedSecurityControl (class ID #332)
//	This ME contains the capabilities, parameters and controls of enhanced GPON security features
//	when they are negotiated via the OMCI (Note). The attributes in this ME are intended to be used
//	to implement a symmetric-key-based three step authentication process as described in the
//	supplemental information section in the following.
//
//	NOTE - If an ITU-T G.987 system uses 802.1X authentication as defined in [ITU-T G.987.3], the
//	only applicable attribute of this ME is the broadcast key table.
//
//	Relationships
//		One instance of this ME is associated with the ONU ME.
//
//	Attributes
//		Managed Entity Id
//			Managed entity ID: This attribute uniquely identifies each instance of this ME. There is only
//			one instance, number 0. (R) (mandatory) (2 bytes)
//
//		Olt Crypto Capabilities
//			(W) (mandatory) (16 bytes)
//
//		Olt Random Challenge Table
//			NOTE - It is assumed that the length of OLT_challenge is always an integer multiple of 16-bytes.
//
//		Olt Challenge Status
//			The ONU initializes this attribute to the value false. (R, W) (mandatory) (1-byte)
//
//		Onu Selected Crypto Capabilities
//			ONU selected crypto capabilities: This attribute specifies the cryptographic capability selected
//			by the ONU in authentication step 2. Its value specifies one of the bit positions that has the
//			value 1 in the OLT crypto capabilities attribute. (R) (mandatory) (1 byte)
//
//		Onu Random Challenge Table
//			ONU random challenge table: This attribute specifies the random challenge ONU_challenge issued
//			by the ONU during authentication step 2. It is structured as a table, with each entry being
//			16-bytes of content. ONU_challenge is the concatenation of all 16-byte content fields in the
//			table. Once the OLT triggers a response to be generated using the OLT challenge status
//			attribute, the ONU generates the response and writes the table (in a single operation). The AVC
//			generated by this attribute signals to the OLT that the challenge is ready, so that the OLT can
//			commence a get/get-next sequence to obtain the table's contents. (R) (mandatory) (16 * P-bytes)
//
//		Onu Authentication Result Table
//			Once the OLT triggers a response to be generated using the OLT challenge status attribute, the
//			ONU generates ONU_result and writes the table (in a single operation). The AVC generated by this
//			attribute signals to the OLT that the response is ready, so that the OLT can commence a get/get-
//			next sequence to obtain the table's contents. (R) (mandatory) (16 * Q-bytes)
//
//		Olt Authentication Result Table
//			This attribute is structured as a table, with each entry being 17 bytes. The first byte is the
//			table row number, starting at 1; the remaining 16 bytes are content. OLT_result is the
//			concatenation of all 16-byte content fields. The OLT writes all entries into the table, and then
//			triggers the ONU's processing of the table using the OLT result status attribute. The number of
//			rows R is implicit in the choice of hash algorithm. The OLT can clear the table with a set
//			operation to row 0. (W) (mandatory) (17 * R-bytes)
//
//		Olt Result Status
//			(R, W) (mandatory) (1 byte)
//
//		Onu Authentication Status
//			(R) (mandatory) (1 byte)
//
//		Master Session Key Name
//			Upon the invalidation of a master session key (e.g., due to an ONU reset or deactivation, or due
//			to an ONU-local decision that the master session key has expired), the ONU sets the master
//			session key name to all zeros. (R) (mandatory) (16 bytes)
//
//		Broadcast Key Table
//			(R, W) (optional) (18N bytes)
//
//		Effective Key Length
//			Effective key length: This attribute specifies the maximum effective length, in bits, of keys
//			generated by the ONU. (R) (optional) (2 bytes)
//
type EnhancedSecurityControl struct {
	ManagedEntityDefinition
	Attributes AttributeValueMap
}

func init() {
	enhancedsecuritycontrolBME = &ManagedEntityDefinition{
		Name:    "EnhancedSecurityControl",
		ClassID: 332,
		MessageTypes: mapset.NewSetWith(
			Get,
			GetNext,
			Set,
		),
		AllowedAttributeMask: 0xfff0,
		AttributeDefinitions: AttributeDefinitionMap{
			0:  Uint16Field("ManagedEntityId", PointerAttributeType, 0x0000, 0, mapset.NewSetWith(Read), false, false, false, 0),
			1:  MultiByteField("OltCryptoCapabilities", OctetsAttributeType, 0x8000, 16, toOctets("AAAAAAAAAAAAAAAAAAAAAA=="), mapset.NewSetWith(Write), false, false, false, 1),
			2:  TableField("OltRandomChallengeTable", TableAttributeType, 0x4000, TableInfo{nil, 17}, mapset.NewSetWith(Read, Write), false, false, false, 2),
			3:  ByteField("OltChallengeStatus", UnsignedIntegerAttributeType, 0x2000, 0, mapset.NewSetWith(Read, Write), false, false, false, 3),
			4:  ByteField("OnuSelectedCryptoCapabilities", UnsignedIntegerAttributeType, 0x1000, 0, mapset.NewSetWith(Read), false, false, false, 4),
			5:  TableField("OnuRandomChallengeTable", TableAttributeType, 0x0800, TableInfo{nil, 16}, mapset.NewSetWith(Read), true, false, false, 5),
			6:  TableField("OnuAuthenticationResultTable", TableAttributeType, 0x0400, TableInfo{nil, 16}, mapset.NewSetWith(Read), true, false, false, 6),
			7:  TableField("OltAuthenticationResultTable", TableAttributeType, 0x0200, TableInfo{nil, 17}, mapset.NewSetWith(Read, Write), false, false, false, 7),
			8:  ByteField("OltResultStatus", UnsignedIntegerAttributeType, 0x0100, 0, mapset.NewSetWith(Read, Write), false, false, false, 8),
			9:  ByteField("OnuAuthenticationStatus", UnsignedIntegerAttributeType, 0x0080, 0, mapset.NewSetWith(Read), true, false, false, 9),
			10: MultiByteField("MasterSessionKeyName", OctetsAttributeType, 0x0040, 16, toOctets("AAAAAAAAAAAAAAAAAAAAAA=="), mapset.NewSetWith(Read), false, false, false, 10),
			11: TableField("BroadcastKeyTable", TableAttributeType, 0x0020, TableInfo{nil, 18}, mapset.NewSetWith(Read, Write), false, true, false, 11),
			12: Uint16Field("EffectiveKeyLength", UnsignedIntegerAttributeType, 0x0010, 0, mapset.NewSetWith(Read), false, true, false, 12),
		},
		Access:  CreatedByOnu,
		Support: UnknownSupport,
	}
}

// NewEnhancedSecurityControl (class ID 332) creates the basic
// Managed Entity definition that is used to validate an ME of this type that
// is received from or transmitted to the OMCC.
func NewEnhancedSecurityControl(params ...ParamData) (*ManagedEntity, OmciErrors) {
	return NewManagedEntity(*enhancedsecuritycontrolBME, params...)
}
