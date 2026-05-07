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

// Package auth contains functions for minting custom authentication tokens, verifying Firebase ID tokens,
// and managing users in a Firebase project.
package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"firebase.google.com/go/v4/internal"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
)

const (
	authErrorCode      = "authErrorCode"
	emulatorHostEnvVar = "FIREBASE_AUTH_EMULATOR_HOST"
	defaultAuthURL     = "https://identitytoolkit.googleapis.com"
	firebaseAudience   = "https://identitytoolkit.googleapis.com/google.identity.identitytoolkit.v1.IdentityToolkit"
	oneHourInSeconds   = 3600

	// SDK-generated error codes
	idTokenRevoked       = "ID_TOKEN_REVOKED"
	userDisabled         = "USER_DISABLED"
	sessionCookieRevoked = "SESSION_COOKIE_REVOKED"
	tenantIDMismatch     = "TENANT_ID_MISMATCH"
)

var reservedClaims = []string{
	"acr", "amr", "at_hash", "aud", "auth_time", "azp", "cnf", "c_hash",
	"exp", "firebase", "iat", "iss", "jti", "nbf", "nonce", "sub",
}

var emulatorToken = &oauth2.Token{
	AccessToken: "owner",
}

// Client is the interface for the Firebase auth service.
//
// Client facilitates generating custom JWT tokens for Firebase clients, and verifying ID tokens issued
// by Firebase backend services.
type Client struct {
	*baseClient
	TenantManager *TenantManager
}

// NewClient creates a new instance of the Firebase Auth Client.
//
// This function can only be invoked from within the SDK. Client applications should access the
// Auth service through firebase.App.
func NewClient(ctx context.Context, conf *internal.AuthConfig) (*Client, error) {
	var (
		isEmulator bool
		signer     cryptoSigner
		err        error
	)

	authEmulatorHost := os.Getenv(emulatorHostEnvVar)
	if authEmulatorHost != "" {
		isEmulator = true
		signer = emulatedSigner{}
	}

	if signer == nil {
		creds, _ := transport.Creds(ctx, conf.Opts...)

		// Initialize a signer by following the go/firebase-admin-sign protocol.
		if creds != nil && len(creds.JSON) > 0 {
			// If the SDK was initialized with a service account, use it to sign bytes.
			signer, err = signerFromCreds(creds.JSON)
			if err != nil && err != errNotAServiceAcct {
				return nil, err
			}
		}
	}

	if signer == nil {
		if conf.ServiceAccountID != "" {
			// If the SDK was initialized with a service account email, use it with the IAM service
			// to sign bytes.
			signer, err = newIAMSigner(ctx, conf)
			if err != nil {
				return nil, err
			}
		} else {
			// Use GAE signing capabilities if available. Otherwise, obtain a service account email
			// from the local Metadata service, and fallback to the IAM service.
			signer, err = newCryptoSigner(ctx, conf)
			if err != nil {
				return nil, err
			}
		}
	}

	idTokenVerifier, err := newIDTokenVerifier(ctx, conf.ProjectID)
	if err != nil {
		return nil, err
	}

	cookieVerifier, err := newSessionCookieVerifier(ctx, conf.ProjectID)
	if err != nil {
		return nil, err
	}

	var opts []option.ClientOption
	if isEmulator {
		ts := oauth2.StaticTokenSource(emulatorToken)
		opts = append(opts, option.WithTokenSource(ts))
	} else {
		opts = append(opts, conf.Opts...)
	}

	transport, _, err := transport.NewHTTPClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	hc := internal.WithDefaultRetryConfig(transport)
	hc.CreateErrFn = handleHTTPError
	hc.Opts = []internal.HTTPOption{
		internal.WithHeader("X-Client-Version", fmt.Sprintf("Go/Admin/%s", conf.Version)),
		internal.WithHeader("x-goog-api-client", internal.GetMetricsHeader(conf.Version)),
	}

	baseURL := defaultAuthURL
	if isEmulator {
		baseURL = fmt.Sprintf("http://%s/identitytoolkit.googleapis.com", authEmulatorHost)
	}
	idToolkitV1Endpoint := fmt.Sprintf("%s/v1", baseURL)
	idToolkitV2Endpoint := fmt.Sprintf("%s/v2", baseURL)
	userManagementEndpoint := idToolkitV1Endpoint
	providerConfigEndpoint := idToolkitV2Endpoint
	tenantMgtEndpoint := idToolkitV2Endpoint
	projectMgtEndpoint := idToolkitV2Endpoint

	base := &baseClient{
		userManagementEndpoint: userManagementEndpoint,
		providerConfigEndpoint: providerConfigEndpoint,
		tenantMgtEndpoint:      tenantMgtEndpoint,
		projectMgtEndpoint:     projectMgtEndpoint,
		projectID:              conf.ProjectID,
		httpClient:             hc,
		idTokenVerifier:        idTokenVerifier,
		cookieVerifier:         cookieVerifier,
		signer:                 signer,
		clock:                  internal.SystemClock,
		isEmulator:             isEmulator,
	}
	return &Client{
		baseClient:    base,
		TenantManager: newTenantManager(hc, conf, base),
	}, nil
}

// CustomToken creates a signed custom authentication token with the specified user ID.
//
// The resulting JWT can be used in a Firebase client SDK to trigger an authentication flow. See
// https://firebase.google.com/docs/auth/admin/create-custom-tokens#sign_in_using_custom_tokens_on_clients
// for more details on how to use custom tokens for client authentication.
//
// CustomToken follows the protocol outlined below to sign the generated tokens:
//   - If the SDK was initialized with service account credentials, uses the private key present in
//     the credentials to sign tokens locally.
//   - If a service account email was specified during initialization (via firebase.Config struct),
//     calls the IAMCredentials service with that email to sign tokens remotely. See
//     https://cloud.google.com/iam/docs/reference/credentials/rest/v1/projects.serviceAccounts/signBlob.
//   - If the code is deployed in the Google App Engine standard environment, uses the App Identity
//     service to sign tokens. See https://cloud.google.com/appengine/docs/standard/go/reference#SignBytes.
//   - If the code is deployed in a different GCP-managed environment (e.g. Google Compute Engine),
//     uses the local Metadata server to auto discover a service account email. This is used in
//     conjunction with the IAM service to sign tokens remotely.
//
// CustomToken returns an error the SDK fails to discover a viable mechanism for signing tokens.
func (c *baseClient) CustomToken(ctx context.Context, uid string) (string, error) {
	return c.CustomTokenWithClaims(ctx, uid, nil)
}

// CustomTokenWithClaims is similar to CustomToken, but in addition to the user ID, it also encodes
// all the key-value pairs in the provided map as claims in the resulting JWT.
func (c *baseClient) CustomTokenWithClaims(ctx context.Context, uid string, devClaims map[string]interface{}) (string, error) {
	iss, err := c.signer.Email(ctx)
	if err != nil {
		return "", err
	}

	if len(uid) == 0 || len(uid) > 128 {
		return "", errors.New("uid must be non-empty, and not longer than 128 characters")
	}

	var disallowed []string
	for _, k := range reservedClaims {
		if _, contains := devClaims[k]; contains {
			disallowed = append(disallowed, k)
		}
	}
	if len(disallowed) == 1 {
		return "", fmt.Errorf("developer claim %q is reserved and cannot be specified", disallowed[0])
	} else if len(disallowed) > 1 {
		return "", fmt.Errorf("developer claims %q are reserved and cannot be specified", strings.Join(disallowed, ", "))
	}

	now := c.clock.Now().Unix()
	info := &jwtInfo{
		header: jwtHeader{Algorithm: c.signer.Algorithm(), Type: "JWT"},
		payload: &customToken{
			Iss:      iss,
			Sub:      iss,
			Aud:      firebaseAudience,
			UID:      uid,
			Iat:      now,
			Exp:      now + oneHourInSeconds,
			TenantID: c.tenantID,
			Claims:   devClaims,
		},
	}
	return info.Token(ctx, c.signer)
}

// SessionCookie creates a new Firebase session cookie from the given ID token and expiry
// duration. The returned JWT can be set as a server-side session cookie with a custom cookie
// policy. Expiry duration must be at least 5 minutes but may not exceed 14 days.
func (c *Client) SessionCookie(
	ctx context.Context,
	idToken string,
	expiresIn time.Duration,
) (string, error) {
	return c.baseClient.createSessionCookie(ctx, idToken, expiresIn)
}

// Token represents a decoded Firebase ID token.
//
// Token provides typed accessors to the common JWT fields such as Audience (aud) and Expiry (exp).
// Additionally it provides a UID field, which indicates the user ID of the account to which this token
// belongs. Any additional JWT claims can be accessed via the Claims map of Token.
type Token struct {
	AuthTime int64                  `json:"auth_time"`
	Issuer   string                 `json:"iss"`
	Audience string                 `json:"aud"`
	Expires  int64                  `json:"exp"`
	IssuedAt int64                  `json:"iat"`
	Subject  string                 `json:"sub,omitempty"`
	UID      string                 `json:"uid,omitempty"`
	Firebase FirebaseInfo           `json:"firebase"`
	Claims   map[string]interface{} `json:"-"`
}

// FirebaseInfo represents the information about the sign-in event, including which auth provider
// was used and provider-specific identity details.
//
// This data is provided by the Firebase Auth service and is a reserved claim in the ID token.
type FirebaseInfo struct {
	SignInProvider string                 `json:"sign_in_provider"`
	Tenant         string                 `json:"tenant"`
	Identities     map[string]interface{} `json:"identities"`
}

// baseClient exposes the APIs common to both auth.Client and auth.TenantClient.
type baseClient struct {
	userManagementEndpoint string
	providerConfigEndpoint string
	tenantMgtEndpoint      string
	projectMgtEndpoint     string
	projectID              string
	tenantID               string
	httpClient             *internal.HTTPClient
	idTokenVerifier        *tokenVerifier
	cookieVerifier         *tokenVerifier
	signer                 cryptoSigner
	clock                  internal.Clock
	isEmulator             bool
}

func (c *baseClient) withTenantID(tenantID string) *baseClient {
	copy := *c
	copy.tenantID = tenantID
	return &copy
}

// VerifyIDToken verifies the signature	and payload of the provided ID token.
//
// VerifyIDToken accepts a signed JWT token string, and verifies that it is current, issued for the
// correct Firebase project, and signed by the Google Firebase services in the cloud. It returns
// a Token containing the decoded claims in the input JWT. See
// https://firebase.google.com/docs/auth/admin/verify-id-tokens#retrieve_id_tokens_on_clients for
// more details on how to obtain an ID token in a client app.
//
// In non-emulator mode, this function does not make any RPC calls most of the time.
// The only time it makes an RPC call is when Google public keys need to be refreshed.
// These keys get cached up to 24 hours, and therefore the RPC overhead gets amortized
// over many invocations of this function.
//
// This does not check whether or not the token has been revoked or disabled. Use `VerifyIDTokenAndCheckRevoked()`
// when a revocation check is needed.
func (c *baseClient) VerifyIDToken(ctx context.Context, idToken string) (*Token, error) {
	return c.verifyIDToken(ctx, idToken, false)
}

// VerifyIDTokenAndCheckRevoked verifies the provided ID token, and additionally checks that the
// token has not been revoked or disabled.
//
// Unlike `VerifyIDToken()`, this function must make an RPC call to perform the revocation check.
// Developers are advised to take this additional overhead into consideration when including this
// function in an authorization flow that gets executed often.
func (c *baseClient) VerifyIDTokenAndCheckRevoked(ctx context.Context, idToken string) (*Token, error) {
	return c.verifyIDToken(ctx, idToken, true)
}

func (c *baseClient) verifyIDToken(ctx context.Context, idToken string, checkRevokedOrDisabled bool) (*Token, error) {
	decoded, err := c.idTokenVerifier.VerifyToken(ctx, idToken, c.isEmulator)
	if err != nil {
		return nil, err
	}

	if c.tenantID != "" && c.tenantID != decoded.Firebase.Tenant {
		return nil, &internal.FirebaseError{
			ErrorCode: internal.InvalidArgument,
			String:    fmt.Sprintf("invalid tenant id: %q", decoded.Firebase.Tenant),
			Ext: map[string]interface{}{
				authErrorCode: tenantIDMismatch,
			},
		}
	}

	if c.isEmulator || checkRevokedOrDisabled {
		err = c.checkRevokedOrDisabled(ctx, decoded, idTokenRevoked, "ID token has been revoked")
		if err != nil {
			return nil, err
		}
	}

	return decoded, nil
}

// IsTenantIDMismatch checks if the given error was due to a mismatched tenant ID in a JWT.
func IsTenantIDMismatch(err error) bool {
	return hasAuthErrorCode(err, tenantIDMismatch)
}

// IsIDTokenRevoked checks if the given error was due to a revoked ID token.
//
// When IsIDTokenRevoked returns true, IsIDTokenInvalid is guaranteed to return true.
func IsIDTokenRevoked(err error) bool {
	return hasAuthErrorCode(err, idTokenRevoked)
}

// IsUserDisabled checks if the given error was due to a disabled ID token
//
// When IsUserDisabled returns true, IsIDTokenInvalid is guaranteed to return true.
func IsUserDisabled(err error) bool {
	return hasAuthErrorCode(err, userDisabled)
}

// VerifySessionCookie verifies the signature and payload of the provided Firebase session cookie.
//
// VerifySessionCookie accepts a signed JWT token string, and verifies that it is current, issued for the
// correct Firebase project, and signed by the Google Firebase services in the cloud. It returns a Token containing the
// decoded claims in the input JWT. See https://firebase.google.com/docs/auth/admin/manage-cookies for more details on
// how to obtain a session cookie.
//
// In non-emulator mode, this function does not make any RPC calls most of the time.
// The only time it makes an RPC call is when Google public keys need to be refreshed.
// These keys get cached up to 24 hours, and therefore the RPC overhead gets amortized
// over many invocations of this function.
//
// This does not check whether or not the cookie has been revoked. Use `VerifySessionCookieAndCheckRevoked()`
// when a revocation check is needed.
func (c *Client) VerifySessionCookie(ctx context.Context, sessionCookie string) (*Token, error) {
	return c.verifySessionCookie(ctx, sessionCookie, false)
}

// VerifySessionCookieAndCheckRevoked verifies the provided session cookie, and additionally checks that the
// cookie has not been revoked and the user has not been disabled.
//
// Unlike `VerifySessionCookie()`, this function must make an RPC call to perform the revocation check.
// Developers are advised to take this additional overhead into consideration when including this
// function in an authorization flow that gets executed often.
func (c *Client) VerifySessionCookieAndCheckRevoked(ctx context.Context, sessionCookie string) (*Token, error) {
	return c.verifySessionCookie(ctx, sessionCookie, true)
}

func (c *Client) verifySessionCookie(ctx context.Context, sessionCookie string, checkRevokedOrDisabled bool) (*Token, error) {
	decoded, err := c.cookieVerifier.VerifyToken(ctx, sessionCookie, c.isEmulator)
	if err != nil {
		return nil, err
	}

	if c.isEmulator || checkRevokedOrDisabled {
		err := c.checkRevokedOrDisabled(ctx, decoded, sessionCookieRevoked, "session cookie has been revoked")
		if err != nil {
			return nil, err
		}
	}

	return decoded, nil
}

// IsSessionCookieRevoked checks if the given error was due to a revoked session cookie.
//
// When IsSessionCookieRevoked returns true, IsSessionCookieInvalid is guaranteed to return true.
func IsSessionCookieRevoked(err error) bool {
	return hasAuthErrorCode(err, sessionCookieRevoked)
}

// checkRevokedOrDisabled checks whether the input token has been revoked or disabled.
func (c *baseClient) checkRevokedOrDisabled(ctx context.Context, token *Token, errCode string, errMessage string) error {
	user, err := c.GetUser(ctx, token.UID)
	if err != nil {
		return err
	}
	if user.Disabled {
		return &internal.FirebaseError{
			ErrorCode: internal.InvalidArgument,
			String:    "user has been disabled",
			Ext: map[string]interface{}{
				authErrorCode: userDisabled,
			},
		}

	}
	if token.IssuedAt*1000 < user.TokensValidAfterMillis {
		return &internal.FirebaseError{
			ErrorCode: internal.InvalidArgument,
			String:    errMessage,
			Ext: map[string]interface{}{
				authErrorCode: errCode,
			},
		}
	}
	return nil
}

func hasAuthErrorCode(err error, code string) bool {
	fe, ok := err.(*internal.FirebaseError)
	if !ok {
		return false
	}

	got, ok := fe.Ext[authErrorCode]
	return ok && got == code
}
