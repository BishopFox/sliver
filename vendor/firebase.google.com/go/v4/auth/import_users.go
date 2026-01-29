// Copyright 2019 Google Inc. All Rights Reserved.
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

package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"firebase.google.com/go/v4/internal"
)

const maxImportUsers = 1000

// UserImportOption is an option for the ImportUsers() function.
type UserImportOption interface {
	applyTo(req map[string]interface{}) error
}

// UserImportResult represents the result of an ImportUsers() call.
type UserImportResult struct {
	SuccessCount int
	FailureCount int
	Errors       []*ErrorInfo
}

// ErrorInfo represents an error encountered while importing a single user account.
//
// The Index field corresponds to the index of the failed user in the users array that was passed
// to ImportUsers().
type ErrorInfo struct {
	Index  int
	Reason string
}

// ImportUsers imports an array of users to Firebase Auth.
//
// No more than 1000 users can be imported in a single call. If at least one user specifies a
// password, a UserImportHash must be specified as an option.
func (c *baseClient) ImportUsers(
	ctx context.Context, users []*UserToImport, opts ...UserImportOption) (*UserImportResult, error) {

	if len(users) == 0 {
		return nil, errors.New("users list must not be empty")
	}
	if len(users) > maxImportUsers {
		return nil, fmt.Errorf("users list must not contain more than %d elements", maxImportUsers)
	}

	var validatedUsers []map[string]interface{}
	hashRequired := false
	for _, u := range users {
		vu, err := u.validatedUserInfo()
		if err != nil {
			return nil, err
		}
		if pw, ok := vu["passwordHash"]; ok && pw != "" {
			hashRequired = true
		}
		validatedUsers = append(validatedUsers, vu)
	}

	req := map[string]interface{}{
		"users": validatedUsers,
	}
	for _, opt := range opts {
		if err := opt.applyTo(req); err != nil {
			return nil, err
		}
	}
	if hashRequired {
		if algo, ok := req["hashAlgorithm"]; !ok || algo == "" {
			return nil, errors.New("hash algorithm option is required to import users with passwords")
		}
	}

	var parsed struct {
		Error []struct {
			Index   int    `json:"index"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	_, err := c.post(ctx, "/accounts:batchCreate", req, &parsed)
	if err != nil {
		return nil, err
	}

	result := &UserImportResult{
		SuccessCount: len(users) - len(parsed.Error),
		FailureCount: len(parsed.Error),
	}
	for _, e := range parsed.Error {
		result.Errors = append(result.Errors, &ErrorInfo{
			Index:  int(e.Index),
			Reason: e.Message,
		})
	}
	return result, nil
}

// UserToImport represents a user account that can be bulk imported into Firebase Auth.
type UserToImport struct {
	params map[string]interface{}
}

// UID setter. This field is required.
func (u *UserToImport) UID(uid string) *UserToImport {
	return u.set("localId", uid)
}

// Email setter.
func (u *UserToImport) Email(email string) *UserToImport {
	return u.set("email", email)
}

// DisplayName setter.
func (u *UserToImport) DisplayName(displayName string) *UserToImport {
	return u.set("displayName", displayName)
}

// PhotoURL setter.
func (u *UserToImport) PhotoURL(url string) *UserToImport {
	return u.set("photoUrl", url)
}

// PhoneNumber setter.
func (u *UserToImport) PhoneNumber(phoneNumber string) *UserToImport {
	return u.set("phoneNumber", phoneNumber)
}

// Metadata setter.
func (u *UserToImport) Metadata(metadata *UserMetadata) *UserToImport {
	if metadata.CreationTimestamp != 0 {
		u.set("createdAt", metadata.CreationTimestamp)
	}

	if metadata.LastLogInTimestamp != 0 {
		u.set("lastLoginAt", metadata.LastLogInTimestamp)
	}

	return u
}

// CustomClaims setter.
func (u *UserToImport) CustomClaims(claims map[string]interface{}) *UserToImport {
	return u.set("customClaims", claims)
}

// Disabled setter.
func (u *UserToImport) Disabled(disabled bool) *UserToImport {
	return u.set("disabled", disabled)
}

// EmailVerified setter.
func (u *UserToImport) EmailVerified(emailVerified bool) *UserToImport {
	return u.set("emailVerified", emailVerified)
}

// PasswordHash setter. When set, a UserImportHash must be specified as an option to call
// ImportUsers().
func (u *UserToImport) PasswordHash(password []byte) *UserToImport {
	return u.set("passwordHash", base64.RawURLEncoding.EncodeToString(password))
}

// PasswordSalt setter.
func (u *UserToImport) PasswordSalt(salt []byte) *UserToImport {
	return u.set("salt", base64.RawURLEncoding.EncodeToString(salt))
}

func (u *UserToImport) set(key string, value interface{}) *UserToImport {
	if u.params == nil {
		u.params = make(map[string]interface{})
	}
	u.params[key] = value
	return u
}

// UserProvider represents a user identity provider.
//
// One or more user providers can be specified for each user when importing in bulk.
// See UserToImport type.
type UserProvider struct {
	UID         string `json:"rawId"`
	ProviderID  string `json:"providerId"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	PhotoURL    string `json:"photoUrl,omitempty"`
}

// ProviderData setter.
func (u *UserToImport) ProviderData(providers []*UserProvider) *UserToImport {
	return u.set("providerUserInfo", providers)
}

func (u *UserToImport) validatedUserInfo() (map[string]interface{}, error) {
	if len(u.params) == 0 {
		return nil, fmt.Errorf("no parameters are set on the user to import")
	}

	info := make(map[string]interface{})
	for k, v := range u.params {
		info[k] = v
	}

	if err := validateUID(info["localId"].(string)); err != nil {
		return nil, err
	}
	if email, ok := info["email"]; ok {
		if err := validateEmail(email.(string)); err != nil {
			return nil, err
		}
	}
	if phone, ok := info["phoneNumber"]; ok {
		if err := validatePhone(phone.(string)); err != nil {
			return nil, err
		}
	}

	if claims, ok := info["customClaims"]; ok {
		claimsMap := claims.(map[string]interface{})
		if len(claimsMap) > 0 {
			cc, err := marshalCustomClaims(claimsMap)
			if err != nil {
				return nil, err
			}
			info["customAttributes"] = cc
		}
		delete(info, "customClaims")
	}

	if providers, ok := info["providerUserInfo"]; ok {
		for _, p := range providers.([]*UserProvider) {
			if err := validateProviderUserInfo(p); err != nil {
				return nil, err
			}
		}
	}
	return info, nil
}

// WithHash returns a UserImportOption that specifies a hash configuration.
func WithHash(hash UserImportHash) UserImportOption {
	return withHash{hash}
}

// UserImportHash represents a hash algorithm and the associated configuration that can be used to
// hash user passwords.
//
// A UserImportHash must be specified in the form of a UserImportOption when importing users with
// passwords. See ImportUsers() and WithHash() functions.
type UserImportHash interface {
	Config() (internal.HashConfig, error)
}

type withHash struct {
	hash UserImportHash
}

func (w withHash) applyTo(req map[string]interface{}) error {
	conf, err := w.hash.Config()
	if err != nil {
		return err
	}

	for k, v := range conf {
		req[k] = v
	}
	return nil
}
