// Copyright 2023 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/internal"
)

// ProjectConfig represents the properties to update on the provided project config.
type ProjectConfig struct {
	MultiFactorConfig *MultiFactorConfig `json:"mfa,omitEmpty"`
}

func (base *baseClient) GetProjectConfig(ctx context.Context) (*ProjectConfig, error) {
	req := &internal.Request{
		Method: http.MethodGet,
		URL:    "/config",
	}
	var result ProjectConfig
	if _, err := base.makeRequest(ctx, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (base *baseClient) UpdateProjectConfig(ctx context.Context, projectConfig *ProjectConfigToUpdate) (*ProjectConfig, error) {
	if projectConfig == nil {
		return nil, errors.New("project config must not be nil")
	}
	if err := projectConfig.validate(); err != nil {
		return nil, err
	}
	mask := projectConfig.params.UpdateMask()
	if len(mask) == 0 {
		return nil, errors.New("no parameters specified in the update request")
	}
	req := &internal.Request{
		Method: http.MethodPatch,
		URL:    "/config",
		Body:   internal.NewJSONEntity(projectConfig.params),
		Opts: []internal.HTTPOption{
			internal.WithQueryParam("updateMask", strings.Join(mask, ",")),
		},
	}
	var result ProjectConfig
	if _, err := base.makeRequest(ctx, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ProjectConfigToUpdate represents the options used to update the current project.
type ProjectConfigToUpdate struct {
	params nestedMap
}

const (
	multiFactorConfigProjectKey = "mfa"
)

// MultiFactorConfig configures the project's multi-factor settings
func (pc *ProjectConfigToUpdate) MultiFactorConfig(multiFactorConfig MultiFactorConfig) *ProjectConfigToUpdate {
	return pc.set(multiFactorConfigProjectKey, multiFactorConfig)
}

func (pc *ProjectConfigToUpdate) set(key string, value interface{}) *ProjectConfigToUpdate {
	pc.ensureParams().Set(key, value)
	return pc
}

func (pc *ProjectConfigToUpdate) ensureParams() nestedMap {
	if pc.params == nil {
		pc.params = make(nestedMap)
	}
	return pc.params
}

func (pc *ProjectConfigToUpdate) validate() error {
	req := make(map[string]interface{})
	for k, v := range pc.params {
		req[k] = v
	}
	val, ok := req[multiFactorConfigProjectKey]
	if ok {
		multiFactorConfig, ok := val.(MultiFactorConfig)
		if !ok {
			return fmt.Errorf("invalid type for MultiFactorConfig: %s", req[multiFactorConfigProjectKey])
		}
		if err := multiFactorConfig.validate(); err != nil {
			return err
		}
	}
	return nil
}
