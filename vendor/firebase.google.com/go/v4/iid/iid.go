// Copyright 2017 Google Inc. All Rights Reserved.
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

// Package iid contains functions for deleting instance IDs from Firebase projects.
package iid

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/errorutils"
	"firebase.google.com/go/v4/internal"
)

const iidEndpoint = "https://console.firebase.google.com/v1"

var errorMessages = map[int]string{
	http.StatusBadRequest:          "malformed instance id argument",
	http.StatusUnauthorized:        "request not authorized",
	http.StatusForbidden:           "project does not match instance ID or the client does not have sufficient privileges",
	http.StatusNotFound:            "failed to find the instance id",
	http.StatusConflict:            "already deleted",
	http.StatusTooManyRequests:     "request throttled out by the backend server",
	http.StatusInternalServerError: "internal server error",
	http.StatusServiceUnavailable:  "backend servers are over capacity",
}

// IsInvalidArgument checks if the given error was due to an invalid instance ID argument.
//
// Deprecated: Use errorutils.IsInvalidArgument() function instead.
func IsInvalidArgument(err error) bool {
	return errorutils.IsInvalidArgument(err)
}

// IsInsufficientPermission checks if the given error was due to the request not having the
// required authorization. This could be due to the client not having the required permission
// or the specified instance ID not matching the target Firebase project.
//
// Deprecated: Use errorutils.IsUnauthenticated() or errorutils.IsPermissionDenied() instead.
func IsInsufficientPermission(err error) bool {
	return errorutils.IsUnauthenticated(err) || errorutils.IsPermissionDenied(err)
}

// IsNotFound checks if the given error was due to a non existing instance ID.
func IsNotFound(err error) bool {
	return errorutils.IsNotFound(err)
}

// IsAlreadyDeleted checks if the given error was due to the instance ID being already deleted from
// the project.
//
// Deprecated: Use errorutils.IsConflict() function instead.
func IsAlreadyDeleted(err error) bool {
	return errorutils.IsConflict(err)
}

// IsTooManyRequests checks if the given error was due to the client sending too many requests
// causing a server quota to exceed.
//
// Deprecated: Use errorutils.IsResourceExhausted() function instead.
func IsTooManyRequests(err error) bool {
	return errorutils.IsResourceExhausted(err)
}

// IsInternal checks if the given error was due to an internal server error.
//
// Deprecated: Use errorutils.IsInternal() function instead.
func IsInternal(err error) bool {
	return errorutils.IsInternal(err)
}

// IsServerUnavailable checks if the given error was due to the backend server being temporarily
// unavailable.
//
// Deprecated: Use errorutils.IsUnavailable() function instead.
func IsServerUnavailable(err error) bool {
	return errorutils.IsUnavailable(err)
}

// IsUnknown checks if the given error was due to unknown error returned by the backend server.
//
// Deprecated: Use errorutils.IsUnknown() function instead.
func IsUnknown(err error) bool {
	return errorutils.IsUnknown(err)
}

// Client is the interface for the Firebase Instance ID service.
type Client struct {
	// To enable testing against arbitrary endpoints.
	endpoint string
	client   *internal.HTTPClient
	project  string
}

// NewClient creates a new instance of the Firebase instance ID Client.
//
// This function can only be invoked from within the SDK. Client applications should access the
// the instance ID service through firebase.App.
func NewClient(ctx context.Context, c *internal.InstanceIDConfig) (*Client, error) {
	if c.ProjectID == "" {
		return nil, errors.New("project id is required to access instance id client")
	}

	hc, _, err := internal.NewHTTPClient(ctx, c.Opts...)
	if err != nil {
		return nil, err
	}
	hc.Opts = []internal.HTTPOption{
		internal.WithHeader("x-goog-api-client", internal.GetMetricsHeader(c.Version)),
	}

	hc.CreateErrFn = createError
	return &Client{
		endpoint: iidEndpoint,
		client:   hc,
		project:  c.ProjectID,
	}, nil
}

// DeleteInstanceID deletes the specified instance ID and the associated data from Firebase..
//
// Note that Google Analytics for Firebase uses its own form of Instance ID to keep track of
// analytics data. Therefore deleting a regular instance ID does not delete Analytics data.
// See https://firebase.google.com/support/privacy/manage-iids#delete_an_instance_id for
// more information.
func (c *Client) DeleteInstanceID(ctx context.Context, iid string) error {
	if iid == "" {
		return errors.New("instance id must not be empty")
	}

	url := fmt.Sprintf("%s/project/%s/instanceId/%s", c.endpoint, c.project, iid)
	_, err := c.client.Do(ctx, &internal.Request{Method: http.MethodDelete, URL: url})
	return err
}

func createError(resp *internal.Response) error {
	err := internal.NewFirebaseError(resp)
	if msg, ok := errorMessages[resp.Status]; ok {
		requestPath := resp.LowLevelResponse().Request.URL.Path
		idx := strings.LastIndex(requestPath, "/")
		err.String = fmt.Sprintf("instance id %q: %s", requestPath[idx+1:], msg)
	}

	return err
}
