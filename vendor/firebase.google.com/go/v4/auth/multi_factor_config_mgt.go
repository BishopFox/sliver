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
	"fmt"
)

// ProviderConfig represents a multi-factor auth provider configuration.
// Currently, only TOTP is supported.
type ProviderConfig struct {
	// The state of multi-factor configuration, whether it's enabled or disabled.
	State MultiFactorConfigState `json:"state"`
	// TOTPProviderConfig holds the TOTP (time-based one-time password) configuration that is used in second factor authentication.
	TOTPProviderConfig *TOTPProviderConfig `json:"totpProviderConfig,omitempty"`
}

// TOTPProviderConfig represents configuration settings for TOTP second factor auth.
type TOTPProviderConfig struct {
	// The number of adjacent intervals used by TOTP.
	AdjacentIntervals int `json:"adjacentIntervals,omitempty"`
}

// MultiFactorConfigState represents whether the multi-factor configuration is enabled or disabled.
type MultiFactorConfigState string

// These constants represent the possible values for the MultiFactorConfigState type.
const (
	Enabled  MultiFactorConfigState = "ENABLED"
	Disabled MultiFactorConfigState = "DISABLED"
)

// MultiFactorConfig represents a multi-factor configuration for a tenant or project.
// This can be used to define whether multi-factor authentication is enabled or disabled and the list of second factor challenges that are supported.
type MultiFactorConfig struct {
	// A slice of pointers to ProviderConfig structs, each outlining the specific second factor authorization method.
	ProviderConfigs []*ProviderConfig `json:"providerConfigs,omitempty"`
}

func (mfa *MultiFactorConfig) validate() error {
	if mfa == nil {
		return nil
	}
	if len(mfa.ProviderConfigs) == 0 {
		return fmt.Errorf("\"ProviderConfigs\" must be a non-empty array of type \"ProviderConfig\"s")
	}
	for _, providerConfig := range mfa.ProviderConfigs {
		if providerConfig == nil {
			return fmt.Errorf("\"ProviderConfigs\" must be a non-empty array of type \"ProviderConfig\"s")
		}
		if err := providerConfig.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (pvc *ProviderConfig) validate() error {
	if pvc.State == "" && pvc.TOTPProviderConfig == nil {
		return fmt.Errorf("\"ProviderConfig\" must be defined")
	}
	state := string(pvc.State)
	if state != string(Enabled) && state != string(Disabled) {
		return fmt.Errorf("\"ProviderConfig.State\" must be 'Enabled' or 'Disabled'")
	}
	return pvc.TOTPProviderConfig.validate()
}

func (tpvc *TOTPProviderConfig) validate() error {
	if tpvc == nil {
		return fmt.Errorf("\"TOTPProviderConfig\" must be defined")
	}
	if !(tpvc.AdjacentIntervals >= 1 && tpvc.AdjacentIntervals <= 10) {
		return fmt.Errorf("\"AdjacentIntervals\" must be an integer between 1 and 10 (inclusive)")
	}
	return nil
}
