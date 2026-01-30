// Copyright 2018 Google Inc. All Rights Reserved.
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
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"firebase.google.com/go/v4/internal"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
)

const (
	idTokenCertURL            = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"
	idTokenIssuerPrefix       = "https://securetoken.google.com/"
	sessionCookieCertURL      = "https://www.googleapis.com/identitytoolkit/v3/relyingparty/publicKeys"
	sessionCookieIssuerPrefix = "https://session.firebase.google.com/"
	clockSkewSeconds          = 300
	certificateFetchFailed    = "CERTIFICATE_FETCH_FAILED"
	idTokenExpired            = "ID_TOKEN_EXPIRED"
	idTokenInvalid            = "ID_TOKEN_INVALID"
	sessionCookieExpired      = "SESSION_COOKIE_EXPIRED"
	sessionCookieInvalid      = "SESSION_COOKIE_INVALID"
)

// IsCertificateFetchFailed checks if the given error was caused by a failure to fetch public key
// certificates required to verify a JWT.
func IsCertificateFetchFailed(err error) bool {
	return hasAuthErrorCode(err, certificateFetchFailed)
}

// IsIDTokenExpired checks if the given error was due to an expired ID token.
//
// When IsIDTokenExpired returns true, IsIDTokenInvalid is guranteed to return true.
func IsIDTokenExpired(err error) bool {
	return hasAuthErrorCode(err, idTokenExpired)
}

// IsIDTokenInvalid checks if the given error was due to an invalid ID token.
//
// An ID token is considered invalid when it is malformed (i.e. contains incorrect data), expired
// or revoked.
func IsIDTokenInvalid(err error) bool {
	return hasAuthErrorCode(err, idTokenInvalid) || IsIDTokenExpired(err) || IsIDTokenRevoked(err) || IsUserDisabled(err)
}

// IsSessionCookieExpired checks if the given error was due to an expired session cookie.
//
// When IsSessionCookieExpired returns true, IsSessionCookieInvalid is guranteed to return true.
func IsSessionCookieExpired(err error) bool {
	return hasAuthErrorCode(err, sessionCookieExpired)
}

// IsSessionCookieInvalid checks if the given error was due to an invalid session cookie.
//
// A session cookie is considered invalid when it is malformed (i.e. contains incorrect data),
// expired or revoked.
func IsSessionCookieInvalid(err error) bool {
	return hasAuthErrorCode(err, sessionCookieInvalid) || IsSessionCookieExpired(err) ||
		IsSessionCookieRevoked(err) || IsUserDisabled(err)
}

// tokenVerifier verifies different types of Firebase token strings, including ID tokens and
// session cookies.
type tokenVerifier struct {
	shortName         string
	articledShortName string
	docURL            string
	projectID         string
	issuerPrefix      string
	invalidTokenCode  string
	expiredTokenCode  string
	keySource         keySource
	clock             internal.Clock
}

func newIDTokenVerifier(ctx context.Context, projectID string) (*tokenVerifier, error) {
	noAuthHTTPClient, _, err := transport.NewHTTPClient(ctx, option.WithoutAuthentication())
	if err != nil {
		return nil, err
	}

	return &tokenVerifier{
		shortName:         "ID token",
		articledShortName: "an ID token",
		docURL:            "https://firebase.google.com/docs/auth/admin/verify-id-tokens",
		projectID:         projectID,
		issuerPrefix:      idTokenIssuerPrefix,
		invalidTokenCode:  idTokenInvalid,
		expiredTokenCode:  idTokenExpired,
		keySource:         newHTTPKeySource(idTokenCertURL, noAuthHTTPClient),
		clock:             internal.SystemClock,
	}, nil
}

func newSessionCookieVerifier(ctx context.Context, projectID string) (*tokenVerifier, error) {
	noAuthHTTPClient, _, err := transport.NewHTTPClient(ctx, option.WithoutAuthentication())
	if err != nil {
		return nil, err
	}

	return &tokenVerifier{
		shortName:         "session cookie",
		articledShortName: "a session cookie",
		docURL:            "https://firebase.google.com/docs/auth/admin/manage-cookies",
		projectID:         projectID,
		issuerPrefix:      sessionCookieIssuerPrefix,
		invalidTokenCode:  sessionCookieInvalid,
		expiredTokenCode:  sessionCookieExpired,
		keySource:         newHTTPKeySource(sessionCookieCertURL, noAuthHTTPClient),
		clock:             internal.SystemClock,
	}, nil
}

// VerifyToken Verifies that the given token string is a valid Firebase JWT.
//
// VerifyToken considers a token string to be valid if all the following conditions are met:
//   - The token string is a valid RS256 JWT.
//   - The JWT contains a valid key ID (kid) claim.
//   - The JWT contains valid issuer (iss) and audience (aud) claims that match the issuerPrefix
//     and projectID of the tokenVerifier.
//   - The JWT contains a valid subject (sub) claim.
//   - The JWT is not expired, and it has been issued some time in the past.
//   - The JWT is signed by a Firebase Auth backend server as determined by the keySource.
//
// If any of the above conditions are not met, an error is returned. Otherwise a pointer to a
// decoded Token is returned.
func (tv *tokenVerifier) VerifyToken(ctx context.Context, token string, isEmulator bool) (*Token, error) {
	if tv.projectID == "" {
		// Configuration error.
		return nil, errors.New("project id not available")
	}

	// Validate the token content first. This is fast and cheap.
	payload, err := tv.verifyContent(token, isEmulator)
	if err != nil {
		return nil, err
	}

	if err := tv.verifyTimestamps(payload); err != nil {
		return nil, err
	}

	// In emulator mode, skip signature verification
	if isEmulator {
		return payload, nil
	}

	// Verifying the signature requires synchronized access to a key cache and
	// potentially issues an http request. Therefore we do it last.
	if err := tv.verifySignature(ctx, token); err != nil {
		return nil, err
	}

	return payload, nil
}

func (tv *tokenVerifier) verifyContent(token string, isEmulator bool) (*Token, error) {
	if token == "" {
		return nil, &internal.FirebaseError{
			ErrorCode: internal.InvalidArgument,
			String:    fmt.Sprintf("%s must be a non-empty string", tv.shortName),
			Ext:       map[string]interface{}{authErrorCode: tv.invalidTokenCode},
		}
	}

	payload, err := tv.verifyHeaderAndBody(token, isEmulator)
	if err != nil {
		return nil, &internal.FirebaseError{
			ErrorCode: internal.InvalidArgument,
			String: fmt.Sprintf(
				"%s; see %s for details on how to retrieve a valid %s",
				err.Error(), tv.docURL, tv.shortName),
			Ext: map[string]interface{}{authErrorCode: tv.invalidTokenCode},
		}
	}

	return payload, nil
}

func (tv *tokenVerifier) verifyTimestamps(payload *Token) error {
	if (payload.IssuedAt - clockSkewSeconds) > tv.clock.Now().Unix() {
		return &internal.FirebaseError{
			ErrorCode: internal.InvalidArgument,
			String:    fmt.Sprintf("%s issued at future timestamp: %d", tv.shortName, payload.IssuedAt),
			Ext:       map[string]interface{}{authErrorCode: tv.invalidTokenCode},
		}
	}

	if (payload.Expires + clockSkewSeconds) < tv.clock.Now().Unix() {
		return &internal.FirebaseError{
			ErrorCode: internal.InvalidArgument,
			String:    fmt.Sprintf("%s has expired at: %d", tv.shortName, payload.Expires),
			Ext:       map[string]interface{}{authErrorCode: tv.expiredTokenCode},
		}
	}

	return nil
}

func (tv *tokenVerifier) verifySignature(ctx context.Context, token string) error {
	keys, err := tv.keySource.Keys(ctx)
	if err != nil {
		return &internal.FirebaseError{
			ErrorCode: internal.Unknown,
			String:    err.Error(),
			Ext:       map[string]interface{}{authErrorCode: certificateFetchFailed},
		}
	}

	if !tv.verifySignatureWithKeys(ctx, token, keys) {
		return &internal.FirebaseError{
			ErrorCode: internal.InvalidArgument,
			String:    "failed to verify token signature",
			Ext:       map[string]interface{}{authErrorCode: tv.invalidTokenCode},
		}
	}

	return nil
}

func (tv *tokenVerifier) verifyHeaderAndBody(token string, isEmulator bool) (*Token, error) {
	var (
		header  jwtHeader
		payload Token
	)

	segments := strings.Split(token, ".")
	if len(segments) != 3 {
		return nil, errors.New("incorrect number of segments")
	}

	if err := decode(segments[0], &header); err != nil {
		return nil, err
	}

	if err := decode(segments[1], &payload); err != nil {
		return nil, err
	}

	issuer := tv.issuerPrefix + tv.projectID
	if !isEmulator && header.KeyID == "" {
		if payload.Audience == firebaseAudience {
			return nil, fmt.Errorf("expected %s but got a custom token", tv.articledShortName)
		}
		return nil, fmt.Errorf("%s has no 'kid' header", tv.shortName)
	}
	if !isEmulator && header.Algorithm != "RS256" {
		return nil, fmt.Errorf("%s has invalid algorithm; expected 'RS256' but got %q",
			tv.shortName, header.Algorithm)
	}
	if payload.Audience != tv.projectID {
		return nil, fmt.Errorf("%s has invalid 'aud' (audience) claim; expected %q but got %q; %s",
			tv.shortName, tv.projectID, payload.Audience, tv.getProjectIDMatchMessage())
	}
	if payload.Issuer != issuer {
		return nil, fmt.Errorf("%s has invalid 'iss' (issuer) claim; expected %q but got %q; %s",
			tv.shortName, issuer, payload.Issuer, tv.getProjectIDMatchMessage())
	}
	if payload.Subject == "" {
		return nil, fmt.Errorf("%s has empty 'sub' (subject) claim", tv.shortName)
	}
	if len(payload.Subject) > 128 {
		return nil, fmt.Errorf("%s has a 'sub' (subject) claim longer than 128 characters",
			tv.shortName)
	}

	payload.UID = payload.Subject

	var customClaims map[string]interface{}
	if err := decode(segments[1], &customClaims); err != nil {
		return nil, err
	}
	for _, standardClaim := range []string{"iss", "aud", "exp", "iat", "sub", "uid"} {
		delete(customClaims, standardClaim)
	}
	payload.Claims = customClaims

	return &payload, nil
}

func (tv *tokenVerifier) verifySignatureWithKeys(ctx context.Context, token string, keys []*publicKey) bool {
	segments := strings.Split(token, ".")
	var h jwtHeader
	decode(segments[0], &h)

	verified := false
	for _, k := range keys {
		if h.KeyID == "" || h.KeyID == k.Kid {
			if verifyJWTSignature(segments, k) == nil {
				verified = true
				break
			}
		}
	}

	return verified
}

func (tv *tokenVerifier) getProjectIDMatchMessage() string {
	return fmt.Sprintf(
		"make sure the %s comes from the same Firebase project as the credential used to"+
			" authenticate this SDK", tv.shortName)
}

// decode accepts a JWT segment, and decodes it into the given interface.
func decode(segment string, i interface{}) error {
	decoded, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		return err
	}
	return json.NewDecoder(bytes.NewBuffer(decoded)).Decode(i)
}

func verifyJWTSignature(parts []string, k *publicKey) error {
	content := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return err
	}

	h := sha256.New()
	h.Write([]byte(content))
	return rsa.VerifyPKCS1v15(k.Key, crypto.SHA256, h.Sum(nil), []byte(signature))
}

// publicKey represents a parsed RSA public key along with its unique key ID.
type publicKey struct {
	Kid string
	Key *rsa.PublicKey
}

// keySource is used to obtain a set of public keys, which can be used to verify cryptographic
// signatures.
type keySource interface {
	Keys(context.Context) ([]*publicKey, error)
}

// httpKeySource fetches RSA public keys from a remote HTTP server, and caches them in
// memory. It also handles cache! invalidation and refresh based on the standard HTTP
// cache-control headers.
type httpKeySource struct {
	KeyURI     string
	HTTPClient *http.Client
	CachedKeys []*publicKey
	ExpiryTime time.Time
	Clock      internal.Clock
	Mutex      *sync.Mutex
}

func newHTTPKeySource(uri string, hc *http.Client) *httpKeySource {
	return &httpKeySource{
		KeyURI:     uri,
		HTTPClient: hc,
		Clock:      internal.SystemClock,
		Mutex:      &sync.Mutex{},
	}
}

// Keys returns the RSA Public Keys hosted at this key source's URI. Refreshes the data if
// the cache is stale.
func (k *httpKeySource) Keys(ctx context.Context) ([]*publicKey, error) {
	k.Mutex.Lock()
	defer k.Mutex.Unlock()
	if len(k.CachedKeys) == 0 || k.hasExpired() {
		err := k.refreshKeys(ctx)
		if err != nil && len(k.CachedKeys) == 0 {
			return nil, err
		}
	}
	return k.CachedKeys, nil
}

// hasExpired indicates whether the cache has expired.
func (k *httpKeySource) hasExpired() bool {
	return k.Clock.Now().After(k.ExpiryTime)
}

func (k *httpKeySource) refreshKeys(ctx context.Context) error {
	k.CachedKeys = nil
	req, err := http.NewRequest(http.MethodGet, k.KeyURI, nil)
	if err != nil {
		return err
	}

	resp, err := k.HTTPClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid response (%d) while retrieving public keys: %s",
			resp.StatusCode, string(contents))
	}

	newKeys, err := parsePublicKeys(contents)
	if err != nil {
		return err
	}

	maxAge := findMaxAge(resp)

	k.CachedKeys = append([]*publicKey(nil), newKeys...)
	k.ExpiryTime = k.Clock.Now().Add(*maxAge)
	return nil
}

func parsePublicKeys(keys []byte) ([]*publicKey, error) {
	m := make(map[string]string)
	err := json.Unmarshal(keys, &m)
	if err != nil {
		return nil, err
	}

	var result []*publicKey
	for kid, key := range m {
		pubKey, err := parsePublicKey(kid, []byte(key))
		if err != nil {
			return nil, err
		}
		result = append(result, pubKey)
	}
	return result, nil
}

func parsePublicKey(kid string, key []byte) (*publicKey, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, errors.New("failed to decode the certificate as PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	pk, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("certificate is not an RSA key")
	}
	return &publicKey{kid, pk}, nil
}

func findMaxAge(resp *http.Response) *time.Duration {
	cc := resp.Header.Get("cache-control")
	for _, value := range strings.Split(cc, ",") {
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "max-age=") {
			sep := strings.Index(value, "=")
			seconds, err := strconv.ParseInt(value[sep+1:], 10, 64)
			if err != nil {
				seconds = 0
			}
			duration := time.Duration(seconds) * time.Second
			return &duration
		}
	}
	defaultDuration := time.Duration(0) * time.Second
	return &defaultDuration
}
