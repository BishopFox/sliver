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

package remoteconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"firebase.google.com/go/v4/internal"
)

// serverTemplateData stores the internal representation of the server template.
type serverTemplateData struct {
	// A list of conditions in descending order by priority.
	Parameters map[string]parameter `json:"parameters,omitempty"`

	// Map of parameter keys to their optional default values and optional conditional values.
	Conditions []namedCondition `json:"conditions,omitempty"`

	// Version information for the current Remote Config template.
	Version *version `json:"version,omitempty"`

	// Current Remote Config template ETag.
	ETag string `json:"etag"`
}

// ServerTemplate represents a template with configuration data, cache, and service information.
type ServerTemplate struct {
	rcClient                 *rcClient
	cache                    atomic.Pointer[serverTemplateData]
	stringifiedDefaultConfig map[string]string
}

// newServerTemplate initializes a new ServerTemplate with optional default configuration.
func newServerTemplate(rcClient *rcClient, defaultConfig map[string]any) (*ServerTemplate, error) {
	stringifiedConfig := make(map[string]string, len(defaultConfig)) // Pre-allocate map

	for key, value := range defaultConfig {
		if value == nil {
			stringifiedConfig[key] = ""
			continue
		}

		if stringVal, ok := value.(string); ok {
			stringifiedConfig[key] = stringVal
			continue
		}

		// Marshal the value to JSON bytes.
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("unable to stringify default value for parameter '%s': %w", key, err)
		}

		stringifiedConfig[key] = string(jsonBytes)
	}

	return &ServerTemplate{
		rcClient:                 rcClient,
		stringifiedDefaultConfig: stringifiedConfig,
	}, nil
}

// Load fetches the server template data from the remote config service and caches it.
func (s *ServerTemplate) Load(ctx context.Context) error {
	request := &internal.Request{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("%s/v1/projects/%s/namespaces/firebase-server/serverRemoteConfig", s.rcClient.rcBaseURL, s.rcClient.project),
	}

	templateData := new(serverTemplateData)
	response, err := s.rcClient.httpClient.DoAndUnmarshal(ctx, request, &templateData)

	if err != nil {
		return err
	}

	templateData.ETag = response.Header.Get("etag")
	s.cache.Store(templateData)
	return nil
}

// Set initializes a template using a server template JSON.
func (s *ServerTemplate) Set(templateDataJSON string) error {
	templateData := new(serverTemplateData)
	if err := json.Unmarshal([]byte(templateDataJSON), &templateData); err != nil {
		return fmt.Errorf("error while parsing server template: %v", err)
	}
	s.cache.Store(templateData)
	return nil
}

// ToJSON returns a json representing the cached serverTemplateData.
func (s *ServerTemplate) ToJSON() (string, error) {
	jsonServerTemplate, err := json.Marshal(s.cache.Load())

	if err != nil {
		return "", fmt.Errorf("error while parsing server template: %v", err)
	}

	return string(jsonServerTemplate), nil
}

// Evaluate and processes the cached template data.
func (s *ServerTemplate) Evaluate(context map[string]any) (*ServerConfig, error) {
	if s.cache.Load() == nil {
		return &ServerConfig{}, errors.New("no Remote Config Server template in Cache, call Load() before calling Evaluate()")
	}

	config := make(map[string]value)
	// Initialize config with in-app default values.
	for key, inAppDefault := range s.stringifiedDefaultConfig {
		config[key] = value{source: Default, value: inAppDefault}
	}

	usedConditions := s.cache.Load().filterUsedConditions()
	ce := conditionEvaluator{
		conditions:        usedConditions,
		evaluationContext: context,
	}
	evaluatedConditions := ce.evaluateConditions()

	// Overlays config value objects derived by evaluating the template.
	for key, parameter := range s.cache.Load().Parameters {
		var paramValueWrapper parameterValue
		var matchedConditionName string

		// Iterate through used conditions in decreasing priority order.
		for _, condition := range usedConditions {
			if value, ok := parameter.ConditionalValues[condition.Name]; ok && evaluatedConditions[condition.Name] {
				paramValueWrapper = value
				matchedConditionName = condition.Name
				break
			}
		}

		if paramValueWrapper.UseInAppDefault != nil && *paramValueWrapper.UseInAppDefault {
			log.Printf("Parameter '%s': Condition '%s' uses in-app default.\n", key, matchedConditionName)
		} else if paramValueWrapper.Value != nil {
			config[key] = value{source: Remote, value: *paramValueWrapper.Value}
		} else if parameter.DefaultValue.UseInAppDefault != nil && *parameter.DefaultValue.UseInAppDefault {
			log.Printf("Parameter '%s': Using parameter's in-app default.\n", key)
		} else if parameter.DefaultValue.Value != nil {
			config[key] = value{source: Remote, value: *parameter.DefaultValue.Value}
		}
	}
	return newServerConfig(config), nil
}

// filterUsedConditions identifies conditions that are referenced by parameters and returns them in order of decreasing priority.
func (s *serverTemplateData) filterUsedConditions() []namedCondition {
	usedConditionNames := make(map[string]struct{})
	for _, parameter := range s.Parameters {
		for name := range parameter.ConditionalValues {
			usedConditionNames[name] = struct{}{}
		}
	}

	// Filter the original conditions list, preserving order.
	conditionsToEvaluate := make([]namedCondition, 0, len(usedConditionNames))
	for _, condition := range s.Conditions {
		if _, ok := usedConditionNames[condition.Name]; ok {
			conditionsToEvaluate = append(conditionsToEvaluate, condition)
		}
	}
	return conditionsToEvaluate
}
