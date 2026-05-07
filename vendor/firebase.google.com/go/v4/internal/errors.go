// Copyright 2020 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
)

// ErrorCode represents the platform-wide error codes that can be raised by
// Admin SDK APIs.
type ErrorCode string

const (
	// InvalidArgument is a OnePlatform error code.
	InvalidArgument ErrorCode = "INVALID_ARGUMENT"

	// FailedPrecondition is a OnePlatform error code.
	FailedPrecondition ErrorCode = "FAILED_PRECONDITION"

	// OutOfRange is a OnePlatform error code.
	OutOfRange ErrorCode = "OUT_OF_RANGE"

	// Unauthenticated is a OnePlatform error code.
	Unauthenticated ErrorCode = "UNAUTHENTICATED"

	// PermissionDenied is a OnePlatform error code.
	PermissionDenied ErrorCode = "PERMISSION_DENIED"

	// NotFound is a OnePlatform error code.
	NotFound ErrorCode = "NOT_FOUND"

	// Conflict is a custom error code that represents HTTP 409 responses.
	//
	// OnePlatform APIs typically respond with ABORTED or ALREADY_EXISTS explicitly. But a few
	// old APIs send HTTP 409 Conflict without any additional details to distinguish between the two
	// cases. For these we currently use this error code. As more APIs adopt OnePlatform conventions
	// this will become less important.
	Conflict ErrorCode = "CONFLICT"

	// Aborted is a OnePlatform error code.
	Aborted ErrorCode = "ABORTED"

	// AlreadyExists is a OnePlatform error code.
	AlreadyExists ErrorCode = "ALREADY_EXISTS"

	// ResourceExhausted is a OnePlatform error code.
	ResourceExhausted ErrorCode = "RESOURCE_EXHAUSTED"

	// Cancelled is a OnePlatform error code.
	Cancelled ErrorCode = "CANCELLED"

	// DataLoss is a OnePlatform error code.
	DataLoss ErrorCode = "DATA_LOSS"

	// Unknown is a OnePlatform error code.
	Unknown ErrorCode = "UNKNOWN"

	// Internal is a OnePlatform error code.
	Internal ErrorCode = "INTERNAL"

	// Unavailable is a OnePlatform error code.
	Unavailable ErrorCode = "UNAVAILABLE"

	// DeadlineExceeded is a OnePlatform error code.
	DeadlineExceeded ErrorCode = "DEADLINE_EXCEEDED"
)

// FirebaseError is an error type containing an error code string.
type FirebaseError struct {
	ErrorCode ErrorCode
	String    string
	Response  *http.Response
	Ext       map[string]interface{}
}

func (fe *FirebaseError) Error() string {
	return fe.String
}

// HasPlatformErrorCode checks if the given error contains a specific error code.
func HasPlatformErrorCode(err error, code ErrorCode) bool {
	fe, ok := err.(*FirebaseError)
	return ok && fe.ErrorCode == code
}

var httpStatusToErrorCodes = map[int]ErrorCode{
	http.StatusBadRequest:          InvalidArgument,
	http.StatusUnauthorized:        Unauthenticated,
	http.StatusForbidden:           PermissionDenied,
	http.StatusNotFound:            NotFound,
	http.StatusConflict:            Conflict,
	http.StatusTooManyRequests:     ResourceExhausted,
	http.StatusInternalServerError: Internal,
	http.StatusServiceUnavailable:  Unavailable,
}

// NewFirebaseError creates a new error from the given HTTP response.
func NewFirebaseError(resp *Response) *FirebaseError {
	code, ok := httpStatusToErrorCodes[resp.Status]
	if !ok {
		code = Unknown
	}

	return &FirebaseError{
		ErrorCode: code,
		String:    fmt.Sprintf("unexpected http response with status: %d\n%s", resp.Status, string(resp.Body)),
		Response:  resp.LowLevelResponse(),
		Ext:       make(map[string]interface{}),
	}
}

// NewFirebaseErrorOnePlatform parses the response payload as a GCP error response
// and create an error from the details extracted.
//
// If the response failes to parse, or otherwise doesn't provide any useful details
// NewFirebaseErrorOnePlatform creates an error with some sensible defaults.
func NewFirebaseErrorOnePlatform(resp *Response) *FirebaseError {
	base := NewFirebaseError(resp)

	var gcpError struct {
		Error struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.Unmarshal(resp.Body, &gcpError) // ignore any json parse errors at this level
	if gcpError.Error.Status != "" {
		base.ErrorCode = ErrorCode(gcpError.Error.Status)
	}

	if gcpError.Error.Message != "" {
		base.String = gcpError.Error.Message
	}

	return base
}

func newFirebaseErrorTransport(err error) *FirebaseError {
	var code ErrorCode
	var msg string
	if os.IsTimeout(err) {
		code = DeadlineExceeded
		msg = fmt.Sprintf("timed out while making an http call: %v", err)
	} else if isConnectionRefused(err) {
		code = Unavailable
		msg = fmt.Sprintf("failed to establish a connection: %v", err)
	} else {
		code = Unknown
		msg = fmt.Sprintf("unknown error while making an http call: %v", err)
	}

	return &FirebaseError{
		ErrorCode: code,
		String:    msg,
		Ext:       make(map[string]interface{}),
	}
}

// isConnectionRefused attempts to determine if the given error was caused by a failure to establish a
// connection.
//
// A net.OpError where the Op field is set to "dial" or "read" is considered a connection refused
// error. Similarly an ECONNREFUSED error code (Linux-specific) is also considered a connection
// refused error.
func isConnectionRefused(err error) bool {
	switch t := err.(type) {
	case *url.Error:
		return isConnectionRefused(t.Err)
	case *net.OpError:
		if t.Op == "dial" || t.Op == "read" {
			return true
		}
		return isConnectionRefused(t.Err)
	case syscall.Errno:
		return t == syscall.ECONNREFUSED
	}

	return false
}
