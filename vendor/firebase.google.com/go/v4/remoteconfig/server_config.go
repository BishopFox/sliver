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
	"slices"
	"strconv"
	"strings"
)

// ValueSource represents the source of a value.
type ValueSource int

// Constants for value source.
const (
	sourceUnspecified ValueSource = iota
	Static                        // Static represents a statically defined value.
	Remote                        // Remote represents a value fetched from a remote source.
	Default                       // Default represents a default value.
)

// Value defines the interface for configuration values.
type value struct {
	source ValueSource
	value  string
}

// Default values for different parameter types.
const (
	DefaultValueForBoolean = false
	DefaultValueForString  = ""
	DefaultValueForNumber  = 0
)

var booleanTruthyValues = []string{"1", "true", "t", "yes", "y", "on"}

// ServerConfig is the implementation of the ServerConfig interface.
type ServerConfig struct {
	configValues map[string]value
}

// NewServerConfig creates a new ServerConfig instance.
func newServerConfig(configValues map[string]value) *ServerConfig {
	return &ServerConfig{configValues: configValues}
}

// GetBoolean returns the boolean value associated with the given key.
//
// It returns true if the string value is "1", "true", "t", "yes", "y", or "on" (case-insensitive).
// Otherwise, or if the key is not found, it returns the default boolean value (false).
func (s *ServerConfig) GetBoolean(key string) bool {
	return s.getValue(key).asBoolean()
}

// GetInt returns the integer value associated with the given key.
//
// If the parameter value cannot be parsed as an integer, or if the key is not found,
// it returns the default numeric value (0).
func (s *ServerConfig) GetInt(key string) int {
	return s.getValue(key).asInt()
}

// GetFloat returns the float value associated with the given key.
//
// If the parameter value cannot be parsed as a float64, or if the key is not found,
// it returns the default float value (0).
func (s *ServerConfig) GetFloat(key string) float64 {
	return s.getValue(key).asFloat()
}

// GetString returns the string value associated with the given key.
//
// If the key is not found, it returns the default string value ("").
func (s *ServerConfig) GetString(key string) string {
	return s.getValue(key).asString()
}

// GetValueSource returns the source of the value.
func (s *ServerConfig) GetValueSource(key string) ValueSource {
	return s.getValue(key).source
}

// getValue returns the value associated with the given key.
func (s *ServerConfig) getValue(key string) *value {
	if val, ok := s.configValues[key]; ok {
		return &val
	}
	return newValue(Static, DefaultValueForString)
}

// newValue creates a new value instance.
func newValue(source ValueSource, customValue string) *value {
	if customValue == "" {
		customValue = DefaultValueForString
	}
	return &value{source: source, value: customValue}
}

// asString returns the value as a string.
func (v *value) asString() string {
	return v.value
}

// asBoolean returns the value as a boolean.
func (v *value) asBoolean() bool {
	if v.source == Static {
		return DefaultValueForBoolean
	}

	return slices.Contains(booleanTruthyValues, strings.ToLower(v.value))
}

// asInt returns the value as an integer.
func (v *value) asInt() int {
	if v.source == Static {
		return DefaultValueForNumber
	}
	num, err := strconv.Atoi(v.value)

	if err != nil {
		return DefaultValueForNumber
	}

	return num
}

// asFloat returns the value as a float.
func (v *value) asFloat() float64 {
	if v.source == Static {
		return DefaultValueForNumber
	}
	num, err := strconv.ParseFloat(v.value, doublePrecision)

	if err != nil {
		return DefaultValueForNumber
	}

	return num
}
