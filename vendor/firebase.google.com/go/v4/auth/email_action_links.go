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
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

// ActionCodeSettings specifies the required continue/state URL with optional Android and iOS settings. Used when
// invoking the email action link generation APIs.
type ActionCodeSettings struct {
	URL                   string `json:"continueUrl"`
	HandleCodeInApp       bool   `json:"canHandleCodeInApp"`
	IOSBundleID           string `json:"iOSBundleId,omitempty"`
	AndroidPackageName    string `json:"androidPackageName,omitempty"`
	AndroidMinimumVersion string `json:"androidMinimumVersion,omitempty"`
	AndroidInstallApp     bool   `json:"androidInstallApp,omitempty"`
	LinkDomain            string `json:"linkDomain,omitempty"`
	// Deprecated: Use LinkDomain instead.
	DynamicLinkDomain string `json:"dynamicLinkDomain,omitempty"`
}

func (settings *ActionCodeSettings) toMap() (map[string]interface{}, error) {
	if settings.URL == "" {
		return nil, errors.New("URL must not be empty")
	}

	url, err := url.Parse(settings.URL)
	if err != nil || url.Scheme == "" || url.Host == "" {
		return nil, fmt.Errorf("malformed url string: %q", settings.URL)
	}

	if settings.AndroidMinimumVersion != "" || settings.AndroidInstallApp {
		if settings.AndroidPackageName == "" {
			return nil, errors.New("Android package name is required when specifying other Android settings")
		}
	}

	b, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	return result, nil
}

type linkType string

const (
	emailLinkSignIn   linkType = "EMAIL_SIGNIN"
	emailVerification linkType = "VERIFY_EMAIL"
	passwordReset     linkType = "PASSWORD_RESET"
)

// EmailVerificationLink generates the out-of-band email action link for email verification flows for the specified
// email address.
func (c *baseClient) EmailVerificationLink(ctx context.Context, email string) (string, error) {
	return c.EmailVerificationLinkWithSettings(ctx, email, nil)
}

// EmailVerificationLinkWithSettings generates the out-of-band email action link for email verification flows for the
// specified email address, using the action code settings provided.
func (c *baseClient) EmailVerificationLinkWithSettings(
	ctx context.Context, email string, settings *ActionCodeSettings) (string, error) {
	return c.generateEmailActionLink(ctx, emailVerification, email, settings)
}

// PasswordResetLink generates the out-of-band email action link for password reset flows for the specified email
// address.
func (c *baseClient) PasswordResetLink(ctx context.Context, email string) (string, error) {
	return c.PasswordResetLinkWithSettings(ctx, email, nil)
}

// PasswordResetLinkWithSettings generates the out-of-band email action link for password reset flows for the
// specified email address, using the action code settings provided.
func (c *baseClient) PasswordResetLinkWithSettings(
	ctx context.Context, email string, settings *ActionCodeSettings) (string, error) {
	return c.generateEmailActionLink(ctx, passwordReset, email, settings)
}

// EmailSignInLink generates the out-of-band email action link for email link sign-in flows, using the action
// code settings provided.
func (c *baseClient) EmailSignInLink(
	ctx context.Context, email string, settings *ActionCodeSettings) (string, error) {
	return c.generateEmailActionLink(ctx, emailLinkSignIn, email, settings)
}

func (c *baseClient) generateEmailActionLink(
	ctx context.Context, linkType linkType, email string, settings *ActionCodeSettings) (string, error) {

	if email == "" {
		return "", errors.New("email must not be empty")
	}

	if linkType == emailLinkSignIn && settings == nil {
		return "", errors.New("ActionCodeSettings must not be nil when generating sign-in links")
	}

	payload := map[string]interface{}{
		"requestType":   linkType,
		"email":         email,
		"returnOobLink": true,
	}
	if settings != nil {
		settingsMap, err := settings.toMap()
		if err != nil {
			return "", err
		}
		for k, v := range settingsMap {
			payload[k] = v
		}
	}

	var result struct {
		OOBLink string `json:"oobLink"`
	}
	_, err := c.post(ctx, "/accounts:sendOobCode", payload, &result)
	return result.OOBLink, err
}
