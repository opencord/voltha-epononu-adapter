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

import (
	"errors"
	"fmt"
)

// Custom Go Error messages for common OMCI errors
//
// Response Status code related errors
type OmciErrors interface {
	Error() string
	StatusCode() Results
	GetError() error
	GetFailureMask() uint16
	GetUnsupporteMask() uint16
}

type OmciError struct {
	err             string
	statusCode      Results
	failureMask     uint16
	unsupportedMask uint16
}

func (e *OmciError) GetError() error {
	if e.statusCode == Success {
		return nil
	}
	return errors.New(e.err)
}

func (e *OmciError) Error() string {
	return e.err
}

func (e *OmciError) StatusCode() Results {
	return e.statusCode
}

func (e *OmciError) GetFailureMask() uint16 {
	return e.failureMask
}

func (e *OmciError) GetUnsupporteMask() uint16 {
	return e.unsupportedMask
}

// NewOmciSuccess is used to convey a successful request. For Set/Get responses,
// this indicates that all attributes were successfully set/retrieved.
//
// For Set/Get requests that have unsupported/failed attributes (code 1001), use the
// NewAttributeFailureError() function to convey the proper status (AttributeFailure).
//
// For Create requests that have parameter errors (code 0011), use the NewParameterError()
// function to signal which attributes were in error
func NewOmciSuccess() OmciErrors {
	return &OmciError{
		statusCode: Success,
	}
}

// NewNonStatusError is for processing errors that do not involve
// frame processing status & results
func NewNonStatusError(args ...interface{}) OmciErrors {
	defaultValue := "command processing error"
	return &OmciProcessingError{
		OmciError: OmciError{
			err: genMessage(defaultValue, args...),
		},
	}
}

type OmciProcessingError struct {
	OmciError
}

// NewProcessingError means the command processing failed at the ONU
// for reasons not described by one of the more specific error codes.
func NewProcessingError(args ...interface{}) OmciErrors {
	defaultValue := "command processing error"
	return &OmciProcessingError{
		OmciError: OmciError{
			err:        genMessage(defaultValue, args...),
			statusCode: ProcessingError,
		},
	}
}

// NotSupportedError means that the message type indicated in byte 3 is
// not supported by the ONU.
type NotSupportedError struct {
	OmciError
}

// NewNotSupportedError creates a NotSupportedError
func NewNotSupportedError(args ...interface{}) OmciErrors {
	defaultValue := "command not supported"
	return &NotSupportedError{
		OmciError: OmciError{
			err:        genMessage(defaultValue, args...),
			statusCode: NotSupported,
		},
	}
}

// ParamError means that the command message received by the
// ONU was errored. It would be appropriate if an attribute mask
// were out of range, for example. In practice, this result code is
// frequently used interchangeably with code 1001. However, the
// optional attribute and attribute execution masks in the reply
// messages are only defined for code 1001.
type ParamError struct {
	OmciError
}

// NewParameterError creates a ParamError
//
// For Set/Get requests that have unsupported/failed attributes (code 1001), use the
// NewAttributeFailureError() function to convey the proper status (AttributeFailure).
func NewParameterError(mask uint16, args ...interface{}) OmciErrors {
	if mask == 0 {
		panic("invalid attribute mask specified")
	}
	defaultValue := "parameter error"
	err := &ParamError{
		OmciError: OmciError{
			err:         genMessage(defaultValue, args...),
			statusCode:  ParameterError,
			failureMask: mask,
		},
	}
	return err
}

// UnknownEntityError means that the managed entity class
// (bytes 5..6) is not supported by the ONU.
type UnknownEntityError struct {
	OmciError
}

// NewUnknownEntityError creates an UnknownEntityError
func NewUnknownEntityError(args ...interface{}) OmciErrors {
	defaultValue := "unknown managed entity"
	return &UnknownEntityError{
		OmciError: OmciError{
			err:        genMessage(defaultValue, args...),
			statusCode: UnknownEntity,
		},
	}
}

// UnknownInstanceError means that the managed entity instance (bytes 7..8)
// does not exist in the ONU.
type UnknownInstanceError struct {
	OmciError
}

// NewUnknownInstanceError creates an UnknownInstanceError
func NewUnknownInstanceError(args ...interface{}) OmciErrors {
	defaultValue := "unknown managed entity instance"
	return &UnknownInstanceError{
		OmciError: OmciError{
			err:        genMessage(defaultValue, args...),
			statusCode: UnknownInstance,
		},
	}
}

// DeviceBusyError means that the command could not be processed due
// to process-related congestion at the ONU. This result code may
// also be used as a pause indication to the OLT while the ONU
// conducts a time-consuming operation such as storage of a
// software image into non-volatile memory.
type DeviceBusyError struct {
	OmciError
}

// NewDeviceBusyError creates a DeviceBusyError
func NewDeviceBusyError(args ...interface{}) OmciErrors {
	defaultValue := "device busy"
	return &DeviceBusyError{
		OmciError: OmciError{
			err:        genMessage(defaultValue, args...),
			statusCode: DeviceBusy,
		},
	}
}

// InstanceExistsError means that the ONU already has a managed entity instance
// that corresponds to the one the OLT is attempting to create.
type InstanceExistsError struct {
	OmciError
}

// NewInstanceExistsError
func NewInstanceExistsError(args ...interface{}) OmciErrors {
	defaultValue := "instance exists"
	return &InstanceExistsError{
		OmciError: OmciError{
			err:        genMessage(defaultValue, args...),
			statusCode: InstanceExists,
		},
	}
}

// AttributeFailureError is used to encode failed attributes for Get/Set Requests
//
// For Get requests, the failed mask is used to report attributes that could not be
// retrieved (most likely no space available to serialize) and could not be returned
// to the caller. The unsupported mask reports attributes the ONU does not support.
//
// For Set requests, the failed mask is used to report attributes that have errors
// (possibly constraints) and could not be set/saved. The unsupported mask reports
// attributes the ONU does not support.
//
// For Create requests that have parameter errors (code 0011), use the NewParameterError()
// function to signal which attributes were in error
type AttributeFailureError struct {
	OmciError
}

// NewAttributeFailureError is used to ceeate an AttributeFailure error status for
// Get/Set requests
func NewAttributeFailureError(failedMask uint16, unsupportedMask uint16, args ...interface{}) OmciErrors {
	defaultValue := "attribute(s) failed or unknown"

	err := &AttributeFailureError{
		OmciError: OmciError{
			err:             genMessage(defaultValue, args...),
			statusCode:      AttributeFailure,
			failureMask:     failedMask,
			unsupportedMask: unsupportedMask,
		},
	}
	return err
}

// MessageTruncatedError means that the requested attributes could not
// be added to the frame due to size limitations. This is typically an OMCI Error
// returned internally by support functions in the OMCI library and used by the
// frame encoding routines to eventually return an AttributeFailureError
// result (code 1001)
type MessageTruncatedError struct {
	OmciError
}

// NewMessageTruncatedError creates a MessageTruncatedError message
func NewMessageTruncatedError(args ...interface{}) OmciErrors {
	defaultValue := "out-of-space. Cannot fit attribute into message"
	return &MessageTruncatedError{
		OmciError: OmciError{
			err:        genMessage(defaultValue, args...),
			statusCode: ProcessingError,
		},
	}
}

func genMessage(defaultValue string, args ...interface{}) string {
	switch len(args) {
	case 0:
		return defaultValue

	case 1:
		switch first := args[0].(type) {
		case string:
			// Assume a simple, pre-formatted string
			return args[0].(string)

		case func() string:
			// Assume a closure with no other arguments used
			return first()

		default:
			panic("Unsupported parameter type")
		}
	}
	return fmt.Sprintf(args[0].(string), args[1:]...)
}
