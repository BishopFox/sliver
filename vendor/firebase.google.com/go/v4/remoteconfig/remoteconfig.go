// Copyright 2025 Google Inc. All Rights Reserved.
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

// Package remoteconfig provides functions to fetch and evaluate a server-side Remote Config template.
package remoteconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"firebase.google.com/go/v4/internal"
)

const (
	defaultBaseURL       = "https://firebaseremoteconfig.googleapis.com"
	firebaseClientHeader = "X-Firebase-Client"
)

// Client is the interface for the Remote Config Cloud service.
type Client struct {
	*rcClient
}

// NewClient initializes a RemoteConfigClient with app-specific detail and returns a
// client to be used by the user.
func NewClient(ctx context.Context, c *internal.RemoteConfigClientConfig) (*Client, error) {
	if c.ProjectID == "" {
		return nil, errors.New("project ID is required to access Remote Conifg")
	}

	hc, _, err := internal.NewHTTPClient(ctx, c.Opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		rcClient: newRcClient(hc, c),
	}, nil
}

// RemoteConfigClient facilitates requests to the Firebase Remote Config backend.
type rcClient struct {
	httpClient *internal.HTTPClient
	project    string
	rcBaseURL  string
	version    string
}

func newRcClient(client *internal.HTTPClient, conf *internal.RemoteConfigClientConfig) *rcClient {
	version := fmt.Sprintf("fire-admin-go/%s", conf.Version)
	client.Opts = []internal.HTTPOption{
		internal.WithHeader(firebaseClientHeader, version),
		internal.WithHeader("X-Firebase-ETag", "true"),
		internal.WithHeader("x-goog-api-client", internal.GetMetricsHeader(conf.Version)),
	}

	// Handles errors for non-success HTTP status codes from Remote Config servers.
	client.CreateErrFn = handleRemoteConfigError

	return &rcClient{
		rcBaseURL:  defaultBaseURL,
		project:    conf.ProjectID,
		version:    version,
		httpClient: client,
	}
}

// GetServerTemplate initializes a new ServerTemplate instance and fetches the server template.
func (c *rcClient) GetServerTemplate(ctx context.Context,
	defaultConfig map[string]any) (*ServerTemplate, error) {
	template, err := c.InitServerTemplate(defaultConfig, "")

	if err != nil {
		return nil, err
	}

	err = template.Load(ctx)
	return template, err
}

// InitServerTemplate initializes a new ServerTemplate with the default config and
// an optional template data json.
func (c *rcClient) InitServerTemplate(defaultConfig map[string]any,
	templateDataJSON string) (*ServerTemplate, error) {
	template, err := newServerTemplate(c, defaultConfig)

	if templateDataJSON != "" && err == nil {
		err = template.Set(templateDataJSON)
	}

	return template, err
}

func handleRemoteConfigError(resp *internal.Response) error {
	err := internal.NewFirebaseError(resp)
	var p struct {
		Error string `json:"error"`
	}
	json.Unmarshal(resp.Body, &p)
	if p.Error != "" {
		err.String = fmt.Sprintf("http error status: %d; reason: %s", resp.Status, p.Error)
	}

	return err
}
