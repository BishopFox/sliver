// Copyright (c) 2020 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package mautrix

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"go.mau.fi/util/exhttp"
	"go.mau.fi/util/exmaps"
	"golang.org/x/exp/maps"
)

// Common error codes from https://matrix.org/docs/spec/client_server/latest#api-standards
//
// Can be used with errors.Is() to check the response code without casting the error:
//
//	err := client.Sync()
//	if errors.Is(err, MUnknownToken) {
//		// logout
//	}
var (
	// Generic error for when the server encounters an error and it does not have a more specific error code.
	// Note that `errors.Is` will check the error message rather than code for M_UNKNOWNs.
	MUnknown = RespError{ErrCode: "M_UNKNOWN", StatusCode: http.StatusInternalServerError}
	// Forbidden access, e.g. joining a room without permission, failed login.
	MForbidden = RespError{ErrCode: "M_FORBIDDEN", StatusCode: http.StatusForbidden}
	// Unrecognized request, e.g. the endpoint does not exist or is not implemented.
	MUnrecognized = RespError{ErrCode: "M_UNRECOGNIZED", StatusCode: http.StatusNotFound}
	// The access token specified was not recognised.
	MUnknownToken = RespError{ErrCode: "M_UNKNOWN_TOKEN", StatusCode: http.StatusUnauthorized}
	// No access token was specified for the request.
	MMissingToken = RespError{ErrCode: "M_MISSING_TOKEN", StatusCode: http.StatusUnauthorized}
	// Request contained valid JSON, but it was malformed in some way, e.g. missing required keys, invalid values for keys.
	MBadJSON = RespError{ErrCode: "M_BAD_JSON", StatusCode: http.StatusBadRequest}
	// Request did not contain valid JSON.
	MNotJSON = RespError{ErrCode: "M_NOT_JSON", StatusCode: http.StatusBadRequest}
	// No resource was found for this request.
	MNotFound = RespError{ErrCode: "M_NOT_FOUND", StatusCode: http.StatusNotFound}
	// Too many requests have been sent in a short period of time. Wait a while then try again.
	MLimitExceeded = RespError{ErrCode: "M_LIMIT_EXCEEDED", StatusCode: http.StatusTooManyRequests}
	// The user ID associated with the request has been deactivated.
	// Typically for endpoints that prove authentication, such as /login.
	MUserDeactivated = RespError{ErrCode: "M_USER_DEACTIVATED"}
	// Encountered when trying to register a user ID which has been taken.
	MUserInUse = RespError{ErrCode: "M_USER_IN_USE", StatusCode: http.StatusBadRequest}
	// Encountered when trying to register a user ID which is not valid.
	MInvalidUsername = RespError{ErrCode: "M_INVALID_USERNAME", StatusCode: http.StatusBadRequest}
	// Sent when the room alias given to the createRoom API is already in use.
	MRoomInUse = RespError{ErrCode: "M_ROOM_IN_USE", StatusCode: http.StatusBadRequest}
	// The state change requested cannot be performed, such as attempting to unban a user who is not banned.
	MBadState = RespError{ErrCode: "M_BAD_STATE"}
	// The request or entity was too large.
	MTooLarge = RespError{ErrCode: "M_TOO_LARGE", StatusCode: http.StatusRequestEntityTooLarge}
	// The resource being requested is reserved by an application service, or the application service making the request has not created the resource.
	MExclusive = RespError{ErrCode: "M_EXCLUSIVE", StatusCode: http.StatusBadRequest}
	// The client's request to create a room used a room version that the server does not support.
	MUnsupportedRoomVersion = RespError{ErrCode: "M_UNSUPPORTED_ROOM_VERSION"}
	// The client attempted to join a room that has a version the server does not support.
	// Inspect the room_version property of the error response for the room's version.
	MIncompatibleRoomVersion = RespError{ErrCode: "M_INCOMPATIBLE_ROOM_VERSION"}
	// The client specified a parameter that has the wrong value.
	MInvalidParam = RespError{ErrCode: "M_INVALID_PARAM", StatusCode: http.StatusBadRequest}
	// The client specified a room key backup version that is not the current room key backup version for the user.
	MWrongRoomKeysVersion = RespError{ErrCode: "M_WRONG_ROOM_KEYS_VERSION", StatusCode: http.StatusForbidden}

	MURLNotSet         = RespError{ErrCode: "M_URL_NOT_SET"}
	MBadStatus         = RespError{ErrCode: "M_BAD_STATUS"}
	MConnectionTimeout = RespError{ErrCode: "M_CONNECTION_TIMEOUT"}
	MConnectionFailed  = RespError{ErrCode: "M_CONNECTION_FAILED"}

	MUnredactedContentDeleted     = RespError{ErrCode: "FI.MAU.MSC2815_UNREDACTED_CONTENT_DELETED"}
	MUnredactedContentNotReceived = RespError{ErrCode: "FI.MAU.MSC2815_UNREDACTED_CONTENT_NOT_RECEIVED"}
)

var (
	ErrOAuthInvalidRequest        = RespError{ErrCode: "invalid_request", OAuth: true}
	ErrOAuthInvalidClient         = RespError{ErrCode: "invalid_client", OAuth: true}
	ErrOAuthInvalidGrant          = RespError{ErrCode: "invalid_grant", OAuth: true}
	ErrOAuthUnauthorizedClient    = RespError{ErrCode: "unauthorized_client", OAuth: true}
	ErrOAuthUnauthorizedGrantType = RespError{ErrCode: "unauthorized_grant_type", OAuth: true}
	ErrOAuthInvalidScope          = RespError{ErrCode: "invalid_scope", OAuth: true}
	ErrOAuthAuthorizationPending  = RespError{ErrCode: "authorization_pending", OAuth: true}
	ErrOAuthSlowDown              = RespError{ErrCode: "slow_down", OAuth: true}
	ErrOAuthExpiredToken          = RespError{ErrCode: "expired_token", OAuth: true}
	ErrOAuthAccessDenied          = RespError{ErrCode: "access_denied", OAuth: true}
)

var (
	ErrClientIsNil           = errors.New("client is nil")
	ErrClientHasNoHomeserver = errors.New("client has no homeserver set")

	ErrFailedToRefreshToken = errors.New("failed to refresh access token")
	ErrClientIDNotSet       = errors.New("client ID not set")

	ErrResponseTooLong      = errors.New("response content length too long")
	ErrBodyReadReachedLimit = errors.New("reached response size limit while reading body")

	// Special error that indicates we should retry canceled contexts. Note that on it's own this
	// is useless, the context itself must also be replaced.
	ErrContextCancelRetry = errors.New("retry canceled context")
)

// HTTPError An HTTP Error response, which may wrap an underlying native Go Error.
type HTTPError struct {
	Request      *http.Request
	Response     *http.Response
	ResponseBody string

	WrappedError error
	RespError    *RespError
	Message      string
}

func (e HTTPError) Is(err error) bool {
	return (e.RespError != nil && errors.Is(e.RespError, err)) || (e.WrappedError != nil && errors.Is(e.WrappedError, err))
}

func (e HTTPError) IsStatus(code int) bool {
	return e.Response != nil && e.Response.StatusCode == code
}

func (e HTTPError) Error() string {
	if e.WrappedError != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.WrappedError)
	} else if e.RespError != nil {
		msg := e.RespError.Err
		if e.RespError.InternalError != "" {
			msg = e.RespError.InternalError
		}
		return fmt.Sprintf("%s (HTTP %d): %s", e.RespError.ErrCode, e.Response.StatusCode, msg)
	} else if e.Response != nil {
		msg := fmt.Sprintf("HTTP %d", e.Response.StatusCode)
		if len(e.ResponseBody) > 0 {
			msg = fmt.Sprintf("%s: %s", msg, e.ResponseBody)
		}
		return msg
	} else if e.Message != "" {
		return e.Message
	}
	return "unexpected empty error"
}

func (e HTTPError) Unwrap() error {
	if e.WrappedError != nil {
		return e.WrappedError
	} else if e.RespError != nil {
		return *e.RespError
	}
	return nil
}

// RespError is the standard JSON error response from Homeservers. It also implements the Golang "error" interface.
// See https://spec.matrix.org/v1.2/client-server-api/#api-standards
type RespError struct {
	ErrCode   string
	Err       string
	ExtraData map[string]any

	StatusCode  int
	ExtraHeader map[string]string

	CanRetry      bool
	OAuth         bool
	InternalError string
}

func (e *RespError) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &e.ExtraData)
	if err != nil {
		return err
	}
	e.ErrCode, _ = e.ExtraData["errcode"].(string)
	e.Err, _ = e.ExtraData["error"].(string)
	e.CanRetry, _ = e.ExtraData["com.beeper.can_retry"].(bool)
	e.InternalError, _ = e.ExtraData["fi.mau.internal_error"].(string)
	return nil
}

func (e *RespError) mutateOAuthError() {
	if e.ErrCode == "" && e.Err != "" {
		e.ErrCode = e.Err
		e.Err, _ = e.ExtraData["error_description"].(string)
		e.OAuth = true
	}
}

func (e *RespError) MarshalJSON() ([]byte, error) {
	data := exmaps.NonNilClone(e.ExtraData)
	if e.OAuth {
		data["error"] = e.ErrCode
		data["error_description"] = e.Err
	} else {
		data["errcode"] = e.ErrCode
		data["error"] = e.Err
	}
	if e.CanRetry {
		data["com.beeper.can_retry"] = e.CanRetry
	} else {
		delete(data, "com.beeper.can_retry")
	}
	if e.InternalError != "" {
		data["fi.mau.internal_error"] = e.InternalError
	} else {
		delete(data, "fi.mau.internal_error")
	}
	return json.Marshal(data)
}

func (e RespError) Write(w http.ResponseWriter) {
	if w == nil {
		return
	}
	statusCode := e.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}
	for key, value := range e.ExtraHeader {
		w.Header().Set(key, value)
	}
	exhttp.WriteJSONResponse(w, statusCode, &e)
}

func (e RespError) WithMessage(msg string, args ...any) RespError {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	e.Err = msg
	return e
}

func (e RespError) WithStatus(status int) RespError {
	e.StatusCode = status
	return e
}

func (e RespError) WithCanRetry(canRetry bool) RespError {
	e.CanRetry = canRetry
	return e
}

func (e RespError) WithInternalError(err error) RespError {
	e.InternalError = err.Error()
	return e
}

func (e RespError) WithExtraData(extraData map[string]any) RespError {
	e.ExtraData = exmaps.NonNilClone(e.ExtraData)
	maps.Copy(e.ExtraData, extraData)
	return e
}

func (e RespError) WithExtraField(key string, value any) RespError {
	e.ExtraData = exmaps.NonNilClone(e.ExtraData)
	e.ExtraData[key] = value
	return e
}

func (e RespError) WithExtraHeader(key, value string) RespError {
	e.ExtraHeader = exmaps.NonNilClone(e.ExtraHeader)
	e.ExtraHeader[key] = value
	return e
}

func (e RespError) WithExtraHeaders(headers map[string]string) RespError {
	e.ExtraHeader = exmaps.NonNilClone(e.ExtraHeader)
	maps.Copy(e.ExtraHeader, headers)
	return e
}

// Error returns the errcode and error message.
func (e RespError) Error() string {
	if e.OAuth {
		prefix := "oauth: "
		if e.Err != "" {
			return prefix + e.ErrCode + ": " + e.Err
		}
		return prefix + e.ErrCode
	}
	return e.ErrCode + ": " + e.Err
}

func (e RespError) Is(err error) bool {
	e2, ok := err.(RespError)
	if !ok {
		return false
	}
	if e.ErrCode == "M_UNKNOWN" && e2.ErrCode == "M_UNKNOWN" {
		return e.Err == e2.Err
	}
	return e2.ErrCode == e.ErrCode
}
