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

// Represents a Remote Config condition in the dataplane.
// A condition targets a specific group of users. A list of these conditions
// comprises part of a Remote Config template.
type namedCondition struct {
	// A non-empty and unique name of this condition.
	Name string `json:"name,omitempty"`

	// The logic of this condition.
	// See the documentation on https://firebase.google.com/docs/remote-config/condition-reference
	// for the expected syntax of this field.
	Condition *oneOfCondition `json:"condition,omitempty"`
}

// Represents a condition that may be one of several types.
// Only the first defined field will be processed.
type oneOfCondition struct {
	// Makes this condition an OR condition.
	OrCondition *orCondition `json:"orCondition,omitempty"`

	// Makes this condition an AND condition.
	AndCondition *andCondition `json:"andCondition,omitempty"`

	// Makes this condition a percent condition.
	Percent *percentCondition `json:"percent,omitempty"`

	// Makes this condition a custom signal condition.
	CustomSignal *customSignalCondition `json:"customSignal,omitempty"`

	// Added for the purpose of testing.
	Boolean *bool `json:"boolean,omitempty"`
}

// Represents a collection of conditions that evaluate to true if any are true.
type orCondition struct {
	Conditions []oneOfCondition `json:"conditions,omitempty"`
}

// Represents a collection of conditions that evaluate to true if all are true.
type andCondition struct {
	Conditions []oneOfCondition `json:"conditions,omitempty"`
}

// Represents a condition that compares the instance pseudo-random percentile to a given limit.
type percentCondition struct {
	//  The choice of percent operator to determine how to compare targets to percent(s).
	PercentOperator string `json:"percentOperator,omitempty"`

	// The seed used when evaluating the hash function to map an instance to
	// a value in the hash space. This is a string which can have 0 - 32
	// characters and can contain ASCII characters [-_.0-9a-zA-Z].The string is case-sensitive.
	Seed string `json:"seed,omitempty"`

	// The limit of percentiles to target in micro-percents when
	// using the LESS_OR_EQUAL and GREATER_THAN operators. The value must
	// be in the range [0 and 100_000_000].
	MicroPercent uint32 `json:"microPercent,omitempty"`

	// The micro-percent interval to be used with the BETWEEN operator.
	MicroPercentRange microPercentRange `json:"microPercentRange,omitempty"`
}

// Represents the limit of percentiles to target in micro-percents.
// The value must be in the range [0 and 100_000_000].
type microPercentRange struct {
	// The lower limit of percentiles to target in micro-percents.
	// The value must be in the range [0 and 100_000_000].
	MicroPercentLowerBound uint32 `json:"microPercentLowerBound"`

	// The upper limit of percentiles to target in micro-percents.
	// The value must be in the range [0 and 100_000_000].
	MicroPercentUpperBound uint32 `json:"microPercentUpperBound"`
}

// Represents a condition that compares provided signals against a target value.
type customSignalCondition struct {
	// The choice of custom signal operator to determine how to compare targets
	// to value(s).
	CustomSignalOperator string `json:"customSignalOperator,omitempty"`

	// The key of the signal set in the EvaluationContext.
	CustomSignalKey string `json:"customSignalKey,omitempty"`

	// A list of at most 100 target custom signal values. For numeric and semantic version operators, this will have exactly ONE target value.
	TargetCustomSignalValues []string `json:"targetCustomSignalValues,omitempty"`
}

// Structure representing a Remote Config parameter.
// At minimum, a `defaultValue` or a `conditionalValues` entry must be present for the parameter to have any effect.
type parameter struct {
	// The value to set the parameter to, when none of the named conditions evaluate to `true`.
	DefaultValue parameterValue `json:"defaultValue,omitempty"`

	// A `(condition name, value)` map. The condition name of the highest priority
	// (the one listed first in the Remote Config template's conditions list) determines the value of this parameter.
	ConditionalValues map[string]parameterValue `json:"conditionalValues,omitempty"`

	// A description for this parameter. Should not be over 100 characters and may contain any Unicode characters.
	Description string `json:"description,omitempty"`

	// The data type for all values of this parameter in the current version of the template.
	// It can be a string, number, boolean or JSON, and defaults to type string if unspecified.
	ValueType string `json:"valueType,omitempty"`
}

// Represents a Remote Config parameter value
// that could be either an explicit parameter value or an in-app default value.
type parameterValue struct {
	// The `string` value that the parameter is set to when it is an explicit parameter value.
	Value *string `json:"value,omitempty"`

	// If true, indicates that the in-app default value is to be used for the parameter.
	UseInAppDefault *bool `json:"useInAppDefault,omitempty"`
}

// Structure representing a Remote Config template version.
// Output only, except for the version description. Contains metadata about a particular
// version of the Remote Config template. All fields are set at the time the specified Remote Config template is published.
type version struct {
	// The version number of a Remote Config template.
	VersionNumber string `json:"versionNumber,omitempty"`

	// The timestamp of when this version of the Remote Config template was written to the
	// Remote Config backend.
	UpdateTime string `json:"updateTime,omitempty"`

	// The origin of the template update action.
	UpdateOrigin string `json:"updateOrigin,omitempty"`

	// The type of the template update action.
	UpdateType string `json:"updateType,omitempty"`

	// Aggregation of all metadata fields about the account that performed the update.
	UpdateUser *remoteConfigUser `json:"updateUser,omitempty"`

	// The user-provided description of the corresponding Remote Config template.
	Description string `json:"description,omitempty"`

	// The version number of the Remote Config template that has become the current version
	// due to a rollback. Only present if this version is the result of a rollback.
	RollbackSource string `json:"rollbackSource,omitempty"`

	// Indicates whether this Remote Config template was published before version history was supported.
	IsLegacy bool `json:"isLegacy,omitempty"`
}

// Represents a Remote Config user.
type remoteConfigUser struct {
	// Email address. Output only.
	Email string `json:"email,omitempty"`

	// Display name. Output only.
	Name string `json:"name,omitempty"`

	// Image URL. Output only.
	ImageURL string `json:"imageUrl,omitempty"`
}
