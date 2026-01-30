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
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"firebase.google.com/go/v4/internal"
)

const (
	algorithmNone  = "none"
	algorithmRS256 = "RS256"

	emulatorEmail = "firebase-auth-emulator@example.com"
)

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
	KeyID     string `json:"kid,omitempty"`
}

type customToken struct {
	Iss      string                 `json:"iss"`
	Aud      string                 `json:"aud"`
	Exp      int64                  `json:"exp"`
	Iat      int64                  `json:"iat"`
	Sub      string                 `json:"sub,omitempty"`
	UID      string                 `json:"uid,omitempty"`
	TenantID string                 `json:"tenant_id,omitempty"`
	Claims   map[string]interface{} `json:"claims,omitempty"`
}

type jwtInfo struct {
	header  jwtHeader
	payload interface{}
}

// Token encodes the data in the jwtInfo into a signed JSON web token.
func (info *jwtInfo) Token(ctx context.Context, signer cryptoSigner) (string, error) {
	encode := func(i interface{}) (string, error) {
		b, err := json.Marshal(i)
		if err != nil {
			return "", err
		}
		return base64.RawURLEncoding.EncodeToString(b), nil
	}
	header, err := encode(info.header)
	if err != nil {
		return "", err
	}
	payload, err := encode(info.payload)
	if err != nil {
		return "", err
	}

	tokenData := fmt.Sprintf("%s.%s", header, payload)
	sig, err := signer.Sign(ctx, []byte(tokenData))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s", tokenData, base64.RawURLEncoding.EncodeToString(sig)), nil
}

type serviceAccount struct {
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
}

// cryptoSigner is used to cryptographically sign data, and query the identity of the signer.
type cryptoSigner interface {
	Algorithm() string
	Sign(context.Context, []byte) ([]byte, error)
	Email(context.Context) (string, error)
}

// serviceAccountSigner is a cryptoSigner that signs data using service account credentials.
type serviceAccountSigner struct {
	privateKey  *rsa.PrivateKey
	clientEmail string
}

var errNotAServiceAcct = errors.New("credentials json is not a service account")

func signerFromCreds(creds []byte) (cryptoSigner, error) {
	var sa serviceAccount
	if err := json.Unmarshal(creds, &sa); err != nil {
		return nil, err
	}
	if sa.PrivateKey != "" && sa.ClientEmail != "" {
		return newServiceAccountSigner(sa)
	}
	return nil, errNotAServiceAcct
}

func newServiceAccountSigner(sa serviceAccount) (*serviceAccountSigner, error) {
	block, _ := pem.Decode([]byte(sa.PrivateKey))
	if block == nil {
		return nil, fmt.Errorf("no private key data found in: %q", sa.PrivateKey)
	}
	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		parsedKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("private key should be a PEM or plain PKCS1 or PKCS8; parse error: %v", err)
		}
	}
	rsaKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not an RSA key")
	}
	return &serviceAccountSigner{
		privateKey:  rsaKey,
		clientEmail: sa.ClientEmail,
	}, nil
}

func (s serviceAccountSigner) Algorithm() string {
	return algorithmRS256
}

func (s serviceAccountSigner) Sign(ctx context.Context, b []byte) ([]byte, error) {
	hash := sha256.New()
	hash.Write(b)
	return rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, hash.Sum(nil))
}

func (s serviceAccountSigner) Email(ctx context.Context) (string, error) {
	return s.clientEmail, nil
}

// iamSigner is a cryptoSigner that signs data by sending them to the IAMCredentials service. See
// https://cloud.google.com/iam/docs/reference/credentials/rest/v1/projects.serviceAccounts/signBlob
// for details regarding the REST API.
//
// IAMCredentials requires the identity of a service account. This can be specified explicitly
// at initialization. If not specified iamSigner attempts to discover a service account identity by
// calling the local metadata service (works in environments like Google Compute Engine).
type iamSigner struct {
	mutex        *sync.Mutex
	httpClient   *internal.HTTPClient
	serviceAcct  string
	metadataHost string
	iamHost      string
}

func newIAMSigner(ctx context.Context, config *internal.AuthConfig) (*iamSigner, error) {
	hc, _, err := internal.NewHTTPClient(ctx, config.Opts...)
	if err != nil {
		return nil, err
	}
	hc.Opts = []internal.HTTPOption{
		internal.WithHeader("x-goog-api-client", internal.GetMetricsHeader(config.Version)),
	}

	return &iamSigner{
		mutex:        &sync.Mutex{},
		httpClient:   hc,
		serviceAcct:  config.ServiceAccountID,
		metadataHost: "http://metadata.google.internal",
		iamHost:      "https://iamcredentials.googleapis.com",
	}, nil
}

func (s iamSigner) Algorithm() string {
	return algorithmRS256
}

func (s iamSigner) Sign(ctx context.Context, b []byte) ([]byte, error) {
	account, err := s.Email(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/v1/projects/-/serviceAccounts/%s:signBlob", s.iamHost, account)
	body := map[string]interface{}{
		"payload": base64.StdEncoding.EncodeToString(b),
	}
	req := &internal.Request{
		Method: http.MethodPost,
		URL:    url,
		Body:   internal.NewJSONEntity(body),
	}
	var signResponse struct {
		Signature string `json:"signedBlob"`
	}
	if _, err := s.httpClient.DoAndUnmarshal(ctx, req, &signResponse); err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(signResponse.Signature)
}

func (s iamSigner) Email(ctx context.Context) (string, error) {
	if s.serviceAcct != "" {
		return s.serviceAcct, nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	result, err := s.callMetadataService(ctx)
	if err != nil {
		msg := "failed to determine service account: %v; initialize the SDK with service " +
			"account credentials or specify a service account with iam.serviceAccounts.signBlob " +
			"permission; refer to https://firebase.google.com/docs/auth/admin/create-custom-tokens " +
			"for more details on creating custom tokens"
		return "", fmt.Errorf(msg, err)
	}

	s.serviceAcct = result
	return result, nil
}

func (s iamSigner) callMetadataService(ctx context.Context) (string, error) {
	// Use the built-in default client without request authorization or retries for this call.
	noAuthClient := &internal.HTTPClient{
		Client: http.DefaultClient,
	}

	url := fmt.Sprintf("%s/computeMetadata/v1/instance/service-accounts/default/email", s.metadataHost)
	req := &internal.Request{
		Method: http.MethodGet,
		URL:    url,
		Opts: []internal.HTTPOption{
			internal.WithHeader("Metadata-Flavor", "Google"),
		},
	}

	resp, err := noAuthClient.Do(ctx, req)
	if err != nil {
		return "", err
	}

	result := strings.TrimSpace(string(resp.Body))
	if result == "" {
		return "", errors.New("unexpected response from metadata service")
	}

	return result, nil
}

type emulatedSigner struct{}

func (s emulatedSigner) Algorithm() string {
	return algorithmNone
}

func (s emulatedSigner) Email(context.Context) (string, error) {
	return emulatorEmail, nil
}

func (s emulatedSigner) Sign(context.Context, []byte) ([]byte, error) {
	return []byte(""), nil
}
