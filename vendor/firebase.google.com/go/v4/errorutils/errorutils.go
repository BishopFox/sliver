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

// Package errorutils provides functions for checking and handling error conditions.
package errorutils

import (
	"net/http"

	"firebase.google.com/go/v4/internal"
)

// IsInvalidArgument checks if the given error was due to an invalid client argument.
func IsInvalidArgument(err error) bool {
	return internal.HasPlatformErrorCode(err, internal.InvalidArgument)
}

// IsFailedPrecondition checks if the given error was because a request could not be executed
// in the current system state, such as deleting a non-empty directory.
func IsFailedPrecondition(err error) bool {
	return internal.HasPlatformErrorCode(err, internal.FailedPrecondition)
}

// IsOutOfRange checks if the given error due to an invalid range specified by the client.
func IsOutOfRange(err error) bool {
	return internal.HasPlatformErrorCode(err, internal.OutOfRange)
}

// IsUnauthenticated checks if the given error was caused by an unauthenticated request.
//
// Unauthenticated requests are due to missing, invalid, or expired OAuth token.
func IsUnauthenticated(err error) bool {
	return internal.HasPlatformErrorCode(err, internal.Unauthenticated)
}

// IsPermissionDenied checks if the given error was due to a client not having suffificient
// permissions.
//
// This can happen because the OAuth token does not have the right scopes, the client doesn't have
// permission, or the API has not been enabled for the client project.
func IsPermissionDenied(err error) bool {
	return internal.HasPlatformErrorCode(err, internal.PermissionDenied)
}

// IsNotFound checks if the given error was due to a specified resource being not found.
//
// This may also occur when the request is rejected by undisclosed reasons, such as whitelisting.
func IsNotFound(err error) bool {
	return internal.HasPlatformErrorCode(err, internal.NotFound)
}

// IsConflict checks if the given error was due to a concurrency conflict, such as a
// read-modify-write conflict.
//
// This represents an HTTP 409 Conflict status code, without additional information to distinguish
// between ABORTED or ALREADY_EXISTS error conditions.
func IsConflict(err error) bool {
	return internal.HasPlatformErrorCode(err, internal.Conflict)
}

// IsAborted checks if the given error was due to a concurrency conflict, such as a
// read-modify-write conflict.
func IsAborted(err error) bool {
	return internal.HasPlatformErrorCode(err, internal.Aborted)
}

// IsAlreadyExists checks if the given error was because a resource that a client tried to create
// already exists.
func IsAlreadyExists(err error) bool {
	return internal.HasPlatformErrorCode(err, internal.AlreadyExists)
}

// IsResourceExhausted checks if the given error was caused by either running out of a quota or
// reaching a rate limit.
func IsResourceExhausted(err error) bool {
	return internal.HasPlatformErrorCode(err, internal.ResourceExhausted)
}

// IsCancelled checks if the given error was due to the client cancelling a request.
func IsCancelled(err error) bool {
	return internal.HasPlatformErrorCode(err, internal.Cancelled)
}

// IsDataLoss checks if the given error was due to an unrecoverable data loss or corruption.
//
// The client should report such errors to the end user.
func IsDataLoss(err error) bool {
	return internal.HasPlatformErrorCode(err, internal.DataLoss)
}

// IsUnknown checks if the given error was cuased by an unknown server error.
//
// This typically indicates a server bug.
func IsUnknown(err error) bool {
	return internal.HasPlatformErrorCode(err, internal.Unknown)
}

// IsInternal checks if the given error was due to an internal server error.
//
// This typically indicates a server bug.
func IsInternal(err error) bool {
	return internal.HasPlatformErrorCode(err, internal.Internal)
}

// IsUnavailable checks if the given error was caused by an unavailable service.
//
// This typically indicates that the target service is temporarily down.
func IsUnavailable(err error) bool {
	return internal.HasPlatformErrorCode(err, internal.Unavailable)
}

// IsDeadlineExceeded checks if the given error was due a request exceeding a deadline.
//
// This will happen only if the caller sets a deadline that is shorter than the method's default
// deadline (i.e. requested deadline is not enough for the server to process the request) and the
// request did not finish within the deadline.
func IsDeadlineExceeded(err error) bool {
	return internal.HasPlatformErrorCode(err, internal.DeadlineExceeded)
}

// HTTPResponse returns the http.Response instance that caused the given error.
//
// If the error was not caused by an HTTP error response, returns nil.
//
// Returns a buffered copy of the original response received from the network stack. It is safe to
// read the response content from the returned http.Response.
func HTTPResponse(err error) *http.Response {
	fe, ok := err.(*internal.FirebaseError)
	if ok {
		return fe.Response
	}

	return nil
}
