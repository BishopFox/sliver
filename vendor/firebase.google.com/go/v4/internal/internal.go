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

// Package internal contains functionality that is only accessible from within the Admin SDK.
package internal

import (
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// FirebaseScopes is the set of OAuth2 scopes used by the Admin SDK.
var FirebaseScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/datastore",
	"https://www.googleapis.com/auth/devstorage.full_control",
	"https://www.googleapis.com/auth/firebase",
	"https://www.googleapis.com/auth/identitytoolkit",
	"https://www.googleapis.com/auth/userinfo.email",
}

// SystemClock is a clock that returns local time of the system.
var SystemClock = &systemClock{}

// AuthConfig represents the configuration of Firebase Auth service.
type AuthConfig struct {
	Opts             []option.ClientOption
	ProjectID        string
	ServiceAccountID string
	Version          string
}

// HashConfig represents a hash algorithm configuration used to generate password hashes.
type HashConfig map[string]interface{}

// InstanceIDConfig represents the configuration of Firebase Instance ID service.
type InstanceIDConfig struct {
	Opts      []option.ClientOption
	ProjectID string
	Version   string
}

// DatabaseConfig represents the configuration of Firebase Database service.
type DatabaseConfig struct {
	Opts         []option.ClientOption
	URL          string
	Version      string
	AuthOverride map[string]interface{}
}

// StorageConfig represents the configuration of Google Cloud Storage service.
type StorageConfig struct {
	Opts   []option.ClientOption
	Bucket string
}

// MessagingConfig represents the configuration of Firebase Cloud Messaging service.
type MessagingConfig struct {
	Opts      []option.ClientOption
	ProjectID string
	Version   string
}

// RemoteConfigClientConfig represents the configuration of Firebase Remote Config
type RemoteConfigClientConfig struct {
	Opts      []option.ClientOption
	ProjectID string
	Version   string
}

// AppCheckConfig represents the configuration of App Check service.
type AppCheckConfig struct {
	ProjectID string
}

// MockTokenSource is a TokenSource implementation that can be used for testing.
type MockTokenSource struct {
	AccessToken string
}

// Token returns the test token associated with the TokenSource.
func (ts *MockTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: ts.AccessToken}, nil
}

// Clock is used to query the current local time.
type Clock interface {
	Now() time.Time
}

// systemClock returns the current system time.
type systemClock struct{}

// Now returns the current system time by calling time.Now().
func (s *systemClock) Now() time.Time {
	return time.Now()
}

// MockClock can be used to mock current time during tests.
type MockClock struct {
	Timestamp time.Time
}

// Now returns the timestamp set in the MockClock.
func (m *MockClock) Now() time.Time {
	return m.Timestamp
}
