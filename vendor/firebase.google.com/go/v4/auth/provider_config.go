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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"firebase.google.com/go/v4/internal"
	"google.golang.org/api/iterator"
)

const (
	maxConfigs = 100

	idpEntityIDKey = "idpConfig.idpEntityId"
	ssoURLKey      = "idpConfig.ssoUrl"
	signRequestKey = "idpConfig.signRequest"
	idpCertsKey    = "idpConfig.idpCertificates"

	spEntityIDKey  = "spConfig.spEntityId"
	callbackURIKey = "spConfig.callbackUri"

	clientIDKey     = "clientId"
	clientSecretKey = "clientSecret"
	issuerKey       = "issuer"

	displayNameKey = "displayName"
	enabledKey     = "enabled"

	idTokenResponseTypeKey = "responseType.idToken"
	codeResponseTypeKey    = "responseType.code"
)

type nestedMap map[string]interface{}

func (nm nestedMap) Get(key string) (interface{}, bool) {
	segments := strings.Split(key, ".")
	curr := map[string]interface{}(nm)
	for idx, segment := range segments {
		val, ok := curr[segment]
		if idx == len(segments)-1 || !ok {
			return val, ok
		}

		curr = val.(map[string]interface{})
	}

	return nil, false
}

func (nm nestedMap) GetString(key string) (string, bool) {
	if val, ok := nm.Get(key); ok {
		return val.(string), true
	}

	return "", false
}

func (nm nestedMap) Set(key string, value interface{}) {
	segments := strings.Split(key, ".")
	curr := map[string]interface{}(nm)
	for idx, segment := range segments {
		if idx == len(segments)-1 {
			curr[segment] = value
			return
		}

		child, ok := curr[segment]
		if ok {
			curr = child.(map[string]interface{})
			continue
		}
		newChild := make(map[string]interface{})
		curr[segment] = newChild
		curr = newChild
	}
}

func (nm nestedMap) UpdateMask() []string {
	return buildMask(nm)
}

func buildMask(data map[string]interface{}) []string {
	var mask []string
	for k, v := range data {
		if child, ok := v.(map[string]interface{}); ok {
			childMask := buildMask(child)
			for _, item := range childMask {
				mask = append(mask, fmt.Sprintf("%s.%s", k, item))
			}
		} else {
			mask = append(mask, k)
		}
	}

	return mask
}

// OIDCProviderConfig is the OIDC auth provider configuration.
// See https://openid.net/specs/openid-connect-core-1_0-final.html.
type OIDCProviderConfig struct {
	ID                  string
	DisplayName         string
	Enabled             bool
	ClientID            string
	Issuer              string
	ClientSecret        string
	CodeResponseType    bool
	IDTokenResponseType bool
}

// OIDCProviderConfigToCreate represents the options used to create a new OIDCProviderConfig.
type OIDCProviderConfigToCreate struct {
	id     string
	params nestedMap
}

// ID sets the provider ID of the new config.
func (config *OIDCProviderConfigToCreate) ID(id string) *OIDCProviderConfigToCreate {
	config.id = id
	return config
}

// ClientID sets the client ID of the new config.
func (config *OIDCProviderConfigToCreate) ClientID(clientID string) *OIDCProviderConfigToCreate {
	return config.set(clientIDKey, clientID)
}

// Issuer sets the issuer of the new config.
func (config *OIDCProviderConfigToCreate) Issuer(issuer string) *OIDCProviderConfigToCreate {
	return config.set(issuerKey, issuer)
}

// DisplayName sets the DisplayName field of the new config.
func (config *OIDCProviderConfigToCreate) DisplayName(name string) *OIDCProviderConfigToCreate {
	return config.set(displayNameKey, name)
}

// Enabled enables or disables the new config.
func (config *OIDCProviderConfigToCreate) Enabled(enabled bool) *OIDCProviderConfigToCreate {
	return config.set(enabledKey, enabled)
}

// ClientSecret sets the client secret for the new provider.
// This is required for the code flow.
func (config *OIDCProviderConfigToCreate) ClientSecret(secret string) *OIDCProviderConfigToCreate {
	return config.set(clientSecretKey, secret)
}

// IDTokenResponseType sets whether to enable the ID token response flow for the new provider.
// By default, this is enabled if no response type is specified.
// Having both the code and ID token response flows is currently not supported.
func (config *OIDCProviderConfigToCreate) IDTokenResponseType(enabled bool) *OIDCProviderConfigToCreate {
	return config.set(idTokenResponseTypeKey, enabled)
}

// CodeResponseType sets whether to enable the code response flow for the new provider.
// By default, this is not enabled if no response type is specified.
// A client secret must be set for this response type.
// Having both the code and ID token response flows is currently not supported.
func (config *OIDCProviderConfigToCreate) CodeResponseType(enabled bool) *OIDCProviderConfigToCreate {
	return config.set(codeResponseTypeKey, enabled)
}

func (config *OIDCProviderConfigToCreate) set(key string, value interface{}) *OIDCProviderConfigToCreate {
	if config.params == nil {
		config.params = make(nestedMap)
	}

	config.params.Set(key, value)
	return config
}

func (config *OIDCProviderConfigToCreate) buildRequest() (nestedMap, string, error) {
	if err := validateOIDCConfigID(config.id); err != nil {
		return nil, "", err
	}

	if len(config.params) == 0 {
		return nil, "", errors.New("no parameters specified in the create request")
	}

	if val, ok := config.params.GetString(clientIDKey); !ok || val == "" {
		return nil, "", errors.New("ClientID must not be empty")
	}

	if val, ok := config.params.GetString(issuerKey); !ok || val == "" {
		return nil, "", errors.New("Issuer must not be empty")
	} else if _, err := url.ParseRequestURI(val); err != nil {
		return nil, "", fmt.Errorf("failed to parse Issuer: %v", err)
	}

	if val, ok := config.params.Get(codeResponseTypeKey); ok && val.(bool) {
		if val, ok := config.params.GetString(clientSecretKey); !ok || val == "" {
			return nil, "", errors.New("Client Secret must not be empty for Code Response Type")
		}
		if val, ok := config.params.Get(idTokenResponseTypeKey); ok && val.(bool) {
			return nil, "", errors.New("Only one response type may be chosen")
		}
	} else if ok && !val.(bool) {
		if val, ok := config.params.Get(idTokenResponseTypeKey); ok && !val.(bool) {
			return nil, "", errors.New("At least one response type must be returned")
		}
	}

	return config.params, config.id, nil
}

// OIDCProviderConfigToUpdate represents the options used to update an existing OIDCProviderConfig.
type OIDCProviderConfigToUpdate struct {
	params nestedMap
}

// ClientID updates the client ID of the config.
func (config *OIDCProviderConfigToUpdate) ClientID(clientID string) *OIDCProviderConfigToUpdate {
	return config.set(clientIDKey, clientID)
}

// Issuer updates the issuer of the config.
func (config *OIDCProviderConfigToUpdate) Issuer(issuer string) *OIDCProviderConfigToUpdate {
	return config.set(issuerKey, issuer)
}

// DisplayName updates the DisplayName field of the config.
func (config *OIDCProviderConfigToUpdate) DisplayName(name string) *OIDCProviderConfigToUpdate {
	var nameOrNil interface{}
	if name != "" {
		nameOrNil = name
	}

	return config.set(displayNameKey, nameOrNil)
}

// Enabled enables or disables the config.
func (config *OIDCProviderConfigToUpdate) Enabled(enabled bool) *OIDCProviderConfigToUpdate {
	return config.set(enabledKey, enabled)
}

// ClientSecret sets the client secret for the provider.
// This is required for the code flow.
func (config *OIDCProviderConfigToUpdate) ClientSecret(secret string) *OIDCProviderConfigToUpdate {
	return config.set(clientSecretKey, secret)
}

// IDTokenResponseType sets whether to enable the ID token response flow for the provider.
// By default, this is enabled if no response type is specified.
// Having both the code and ID token response flows is currently not supported.
func (config *OIDCProviderConfigToUpdate) IDTokenResponseType(enabled bool) *OIDCProviderConfigToUpdate {
	return config.set(idTokenResponseTypeKey, enabled)
}

// CodeResponseType sets whether to enable the code response flow for the new provider.
// By default, this is not enabled if no response type is specified.
// A client secret must be set for this response type.
// Having both the code and ID token response flows is currently not supported.
func (config *OIDCProviderConfigToUpdate) CodeResponseType(enabled bool) *OIDCProviderConfigToUpdate {
	return config.set(codeResponseTypeKey, enabled)
}

func (config *OIDCProviderConfigToUpdate) set(key string, value interface{}) *OIDCProviderConfigToUpdate {
	if config.params == nil {
		config.params = make(nestedMap)
	}

	config.params.Set(key, value)
	return config
}

func (config *OIDCProviderConfigToUpdate) buildRequest() (nestedMap, error) {
	if len(config.params) == 0 {
		return nil, errors.New("no parameters specified in the update request")
	}

	if val, ok := config.params.GetString(clientIDKey); ok && val == "" {
		return nil, errors.New("ClientID must not be empty")
	}

	if val, ok := config.params.GetString(issuerKey); ok {
		if val == "" {
			return nil, errors.New("Issuer must not be empty")
		}
		if _, err := url.ParseRequestURI(val); err != nil {
			return nil, fmt.Errorf("failed to parse Issuer: %v", err)
		}
	}

	if val, ok := config.params.Get(codeResponseTypeKey); ok && val.(bool) {
		if val, ok := config.params.GetString(clientSecretKey); !ok || val == "" {
			return nil, errors.New("Client Secret must not be empty for Code Response Type")
		}
		if val, ok := config.params.Get(idTokenResponseTypeKey); ok && val.(bool) {
			return nil, errors.New("Only one response type may be chosen")
		}
	} else if ok && !val.(bool) {
		if val, ok := config.params.Get(idTokenResponseTypeKey); ok && !val.(bool) {
			return nil, errors.New("At least one response type must be returned")
		}
	}

	return config.params, nil
}

// OIDCProviderConfigIterator is an iterator over OIDC provider configurations.
type OIDCProviderConfigIterator struct {
	client   *baseClient
	ctx      context.Context
	nextFunc func() error
	pageInfo *iterator.PageInfo
	configs  []*OIDCProviderConfig
}

// PageInfo supports pagination.
func (it *OIDCProviderConfigIterator) PageInfo() *iterator.PageInfo {
	return it.pageInfo
}

// Next returns the next OIDCProviderConfig. The error value of [iterator.Done] is
// returned if there are no more results. Once Next returns [iterator.Done], all
// subsequent calls will return [iterator.Done].
func (it *OIDCProviderConfigIterator) Next() (*OIDCProviderConfig, error) {
	if err := it.nextFunc(); err != nil {
		return nil, err
	}

	config := it.configs[0]
	it.configs = it.configs[1:]
	return config, nil
}

func (it *OIDCProviderConfigIterator) fetch(pageSize int, pageToken string) (string, error) {
	params := map[string]string{
		"pageSize": strconv.Itoa(pageSize),
	}
	if pageToken != "" {
		params["pageToken"] = pageToken
	}

	req := &internal.Request{
		Method: http.MethodGet,
		URL:    "/oauthIdpConfigs",
		Opts: []internal.HTTPOption{
			internal.WithQueryParams(params),
		},
	}

	var result struct {
		Configs       []oidcProviderConfigDAO `json:"oauthIdpConfigs"`
		NextPageToken string                  `json:"nextPageToken"`
	}
	if _, err := it.client.makeRequest(it.ctx, req, &result); err != nil {
		return "", err
	}

	for _, config := range result.Configs {
		it.configs = append(it.configs, config.toOIDCProviderConfig())
	}

	it.pageInfo.Token = result.NextPageToken
	return result.NextPageToken, nil
}

// SAMLProviderConfig is the SAML auth provider configuration.
// See http://docs.oasis-open.org/security/saml/Post2.0/sstc-saml-tech-overview-2.0.html.
type SAMLProviderConfig struct {
	ID                    string
	DisplayName           string
	Enabled               bool
	IDPEntityID           string
	SSOURL                string
	RequestSigningEnabled bool
	X509Certificates      []string
	RPEntityID            string
	CallbackURL           string
}

// SAMLProviderConfigToCreate represents the options used to create a new SAMLProviderConfig.
type SAMLProviderConfigToCreate struct {
	id     string
	params nestedMap
}

// ID sets the provider ID of the new config.
func (config *SAMLProviderConfigToCreate) ID(id string) *SAMLProviderConfigToCreate {
	config.id = id
	return config
}

// IDPEntityID sets the IDPEntityID field of the new config.
func (config *SAMLProviderConfigToCreate) IDPEntityID(entityID string) *SAMLProviderConfigToCreate {
	return config.set(idpEntityIDKey, entityID)
}

// SSOURL sets the SSOURL field of the new config.
func (config *SAMLProviderConfigToCreate) SSOURL(url string) *SAMLProviderConfigToCreate {
	return config.set(ssoURLKey, url)
}

// RequestSigningEnabled enables or disables the request signing support.
func (config *SAMLProviderConfigToCreate) RequestSigningEnabled(enabled bool) *SAMLProviderConfigToCreate {
	return config.set(signRequestKey, enabled)
}

// X509Certificates sets the certificates for the new config.
func (config *SAMLProviderConfigToCreate) X509Certificates(certs []string) *SAMLProviderConfigToCreate {
	var result []idpCertificate
	for _, cert := range certs {
		result = append(result, idpCertificate{cert})
	}

	return config.set(idpCertsKey, result)
}

// RPEntityID sets the RPEntityID field of the new config.
func (config *SAMLProviderConfigToCreate) RPEntityID(entityID string) *SAMLProviderConfigToCreate {
	return config.set(spEntityIDKey, entityID)
}

// CallbackURL sets the CallbackURL field of the new config.
func (config *SAMLProviderConfigToCreate) CallbackURL(url string) *SAMLProviderConfigToCreate {
	return config.set(callbackURIKey, url)
}

// DisplayName sets the DisplayName field of the new config.
func (config *SAMLProviderConfigToCreate) DisplayName(name string) *SAMLProviderConfigToCreate {
	return config.set(displayNameKey, name)
}

// Enabled enables or disables the new config.
func (config *SAMLProviderConfigToCreate) Enabled(enabled bool) *SAMLProviderConfigToCreate {
	return config.set(enabledKey, enabled)
}

func (config *SAMLProviderConfigToCreate) set(key string, value interface{}) *SAMLProviderConfigToCreate {
	if config.params == nil {
		config.params = make(nestedMap)
	}

	config.params.Set(key, value)
	return config
}

func (config *SAMLProviderConfigToCreate) buildRequest() (nestedMap, string, error) {
	if err := validateSAMLConfigID(config.id); err != nil {
		return nil, "", err
	}

	if len(config.params) == 0 {
		return nil, "", errors.New("no parameters specified in the create request")
	}

	if val, ok := config.params.GetString(idpEntityIDKey); !ok || val == "" {
		return nil, "", errors.New("IDPEntityID must not be empty")
	}

	if val, ok := config.params.GetString(ssoURLKey); !ok || val == "" {
		return nil, "", errors.New("SSOURL must not be empty")
	} else if _, err := url.ParseRequestURI(val); err != nil {
		return nil, "", fmt.Errorf("failed to parse SSOURL: %v", err)
	}

	var certs interface{}
	var ok bool
	if certs, ok = config.params.Get(idpCertsKey); !ok || len(certs.([]idpCertificate)) == 0 {
		return nil, "", errors.New("X509Certificates must not be empty")
	}
	for _, cert := range certs.([]idpCertificate) {
		if cert.X509Certificate == "" {
			return nil, "", errors.New("X509Certificates must not contain empty strings")
		}
	}

	if val, ok := config.params.GetString(spEntityIDKey); !ok || val == "" {
		return nil, "", errors.New("RPEntityID must not be empty")
	}

	if val, ok := config.params.GetString(callbackURIKey); !ok || val == "" {
		return nil, "", errors.New("CallbackURL must not be empty")
	} else if _, err := url.ParseRequestURI(val); err != nil {
		return nil, "", fmt.Errorf("failed to parse CallbackURL: %v", err)
	}

	return config.params, config.id, nil
}

// SAMLProviderConfigToUpdate represents the options used to update an existing SAMLProviderConfig.
type SAMLProviderConfigToUpdate struct {
	params nestedMap
}

// IDPEntityID the IDPEntityID field of the config.
func (config *SAMLProviderConfigToUpdate) IDPEntityID(entityID string) *SAMLProviderConfigToUpdate {
	return config.set(idpEntityIDKey, entityID)
}

// SSOURL updates the SSOURL field of the config.
func (config *SAMLProviderConfigToUpdate) SSOURL(url string) *SAMLProviderConfigToUpdate {
	return config.set(ssoURLKey, url)
}

// RequestSigningEnabled enables or disables the request signing support.
func (config *SAMLProviderConfigToUpdate) RequestSigningEnabled(enabled bool) *SAMLProviderConfigToUpdate {
	return config.set(signRequestKey, enabled)
}

// X509Certificates updates the certificates of the config.
func (config *SAMLProviderConfigToUpdate) X509Certificates(certs []string) *SAMLProviderConfigToUpdate {
	var result []idpCertificate
	for _, cert := range certs {
		result = append(result, idpCertificate{cert})
	}

	return config.set(idpCertsKey, result)
}

// RPEntityID updates the RPEntityID field of the config.
func (config *SAMLProviderConfigToUpdate) RPEntityID(entityID string) *SAMLProviderConfigToUpdate {
	return config.set(spEntityIDKey, entityID)
}

// CallbackURL updates the CallbackURL field of the config.
func (config *SAMLProviderConfigToUpdate) CallbackURL(url string) *SAMLProviderConfigToUpdate {
	return config.set(callbackURIKey, url)
}

// DisplayName updates the DisplayName field of the config.
func (config *SAMLProviderConfigToUpdate) DisplayName(name string) *SAMLProviderConfigToUpdate {
	var nameOrNil interface{}
	if name != "" {
		nameOrNil = name
	}

	return config.set(displayNameKey, nameOrNil)
}

// Enabled enables or disables the config.
func (config *SAMLProviderConfigToUpdate) Enabled(enabled bool) *SAMLProviderConfigToUpdate {
	return config.set(enabledKey, enabled)
}

func (config *SAMLProviderConfigToUpdate) set(key string, value interface{}) *SAMLProviderConfigToUpdate {
	if config.params == nil {
		config.params = make(nestedMap)
	}

	config.params.Set(key, value)
	return config
}

func (config *SAMLProviderConfigToUpdate) buildRequest() (nestedMap, error) {
	if len(config.params) == 0 {
		return nil, errors.New("no parameters specified in the update request")
	}

	if val, ok := config.params.GetString(idpEntityIDKey); ok && val == "" {
		return nil, errors.New("IDPEntityID must not be empty")
	}

	if val, ok := config.params.GetString(ssoURLKey); ok {
		if val == "" {
			return nil, errors.New("SSOURL must not be empty")
		}
		if _, err := url.ParseRequestURI(val); err != nil {
			return nil, fmt.Errorf("failed to parse SSOURL: %v", err)
		}
	}

	if val, ok := config.params.Get(idpCertsKey); ok {
		if len(val.([]idpCertificate)) == 0 {
			return nil, errors.New("X509Certificates must not be empty")
		}
		for _, cert := range val.([]idpCertificate) {
			if cert.X509Certificate == "" {
				return nil, errors.New("X509Certificates must not contain empty strings")
			}
		}
	}

	if val, ok := config.params.GetString(spEntityIDKey); ok && val == "" {
		return nil, errors.New("RPEntityID must not be empty")
	}

	if val, ok := config.params.GetString(callbackURIKey); ok {
		if val == "" {
			return nil, errors.New("CallbackURL must not be empty")
		}
		if _, err := url.ParseRequestURI(val); err != nil {
			return nil, fmt.Errorf("failed to parse CallbackURL: %v", err)
		}
	}

	return config.params, nil
}

// SAMLProviderConfigIterator is an iterator over SAML provider configurations.
type SAMLProviderConfigIterator struct {
	client   *baseClient
	ctx      context.Context
	nextFunc func() error
	pageInfo *iterator.PageInfo
	configs  []*SAMLProviderConfig
}

// PageInfo supports pagination.
func (it *SAMLProviderConfigIterator) PageInfo() *iterator.PageInfo {
	return it.pageInfo
}

// Next returns the next SAMLProviderConfig. The error value of [iterator.Done] is
// returned if there are no more results. Once Next returns [iterator.Done], all
// subsequent calls will return [iterator.Done].
func (it *SAMLProviderConfigIterator) Next() (*SAMLProviderConfig, error) {
	if err := it.nextFunc(); err != nil {
		return nil, err
	}

	config := it.configs[0]
	it.configs = it.configs[1:]
	return config, nil
}

func (it *SAMLProviderConfigIterator) fetch(pageSize int, pageToken string) (string, error) {
	params := map[string]string{
		"pageSize": strconv.Itoa(pageSize),
	}
	if pageToken != "" {
		params["pageToken"] = pageToken
	}

	req := &internal.Request{
		Method: http.MethodGet,
		URL:    "/inboundSamlConfigs",
		Opts: []internal.HTTPOption{
			internal.WithQueryParams(params),
		},
	}

	var result struct {
		Configs       []samlProviderConfigDAO `json:"inboundSamlConfigs"`
		NextPageToken string                  `json:"nextPageToken"`
	}
	if _, err := it.client.makeRequest(it.ctx, req, &result); err != nil {
		return "", err
	}

	for _, config := range result.Configs {
		it.configs = append(it.configs, config.toSAMLProviderConfig())
	}

	it.pageInfo.Token = result.NextPageToken
	return result.NextPageToken, nil
}

// OIDCProviderConfig returns the OIDCProviderConfig with the given ID.
func (c *baseClient) OIDCProviderConfig(ctx context.Context, id string) (*OIDCProviderConfig, error) {
	if err := validateOIDCConfigID(id); err != nil {
		return nil, err
	}

	req := &internal.Request{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("/oauthIdpConfigs/%s", id),
	}
	var result oidcProviderConfigDAO
	if _, err := c.makeRequest(ctx, req, &result); err != nil {
		return nil, err
	}

	return result.toOIDCProviderConfig(), nil
}

// CreateOIDCProviderConfig creates a new OIDC provider config from the given parameters.
func (c *baseClient) CreateOIDCProviderConfig(ctx context.Context, config *OIDCProviderConfigToCreate) (*OIDCProviderConfig, error) {
	if config == nil {
		return nil, errors.New("config must not be nil")
	}

	body, id, err := config.buildRequest()
	if err != nil {
		return nil, err
	}

	req := &internal.Request{
		Method: http.MethodPost,
		URL:    "/oauthIdpConfigs",
		Body:   internal.NewJSONEntity(body),
		Opts: []internal.HTTPOption{
			internal.WithQueryParam("oauthIdpConfigId", id),
		},
	}
	var result oidcProviderConfigDAO
	if _, err := c.makeRequest(ctx, req, &result); err != nil {
		return nil, err
	}

	return result.toOIDCProviderConfig(), nil
}

// UpdateOIDCProviderConfig updates an existing OIDC provider config with the given parameters.
func (c *baseClient) UpdateOIDCProviderConfig(ctx context.Context, id string, config *OIDCProviderConfigToUpdate) (*OIDCProviderConfig, error) {
	if err := validateOIDCConfigID(id); err != nil {
		return nil, err
	}
	if config == nil {
		return nil, errors.New("config must not be nil")
	}

	body, err := config.buildRequest()
	if err != nil {
		return nil, err
	}

	mask := body.UpdateMask()
	req := &internal.Request{
		Method: http.MethodPatch,
		URL:    fmt.Sprintf("/oauthIdpConfigs/%s", id),
		Body:   internal.NewJSONEntity(body),
		Opts: []internal.HTTPOption{
			internal.WithQueryParam("updateMask", strings.Join(mask, ",")),
		},
	}
	var result oidcProviderConfigDAO
	if _, err := c.makeRequest(ctx, req, &result); err != nil {
		return nil, err
	}

	return result.toOIDCProviderConfig(), nil
}

// DeleteOIDCProviderConfig deletes the OIDCProviderConfig with the given ID.
func (c *baseClient) DeleteOIDCProviderConfig(ctx context.Context, id string) error {
	if err := validateOIDCConfigID(id); err != nil {
		return err
	}

	req := &internal.Request{
		Method: http.MethodDelete,
		URL:    fmt.Sprintf("/oauthIdpConfigs/%s", id),
	}
	_, err := c.makeRequest(ctx, req, nil)
	return err
}

// OIDCProviderConfigs returns an iterator over OIDC provider configurations.
//
// If nextPageToken is empty, the iterator will start at the beginning. Otherwise,
// iterator starts after the token.
func (c *baseClient) OIDCProviderConfigs(ctx context.Context, nextPageToken string) *OIDCProviderConfigIterator {
	it := &OIDCProviderConfigIterator{
		ctx:    ctx,
		client: c,
	}
	it.pageInfo, it.nextFunc = iterator.NewPageInfo(
		it.fetch,
		func() int { return len(it.configs) },
		func() interface{} { b := it.configs; it.configs = nil; return b })
	it.pageInfo.MaxSize = maxConfigs
	it.pageInfo.Token = nextPageToken
	return it
}

// SAMLProviderConfig returns the SAMLProviderConfig with the given ID.
func (c *baseClient) SAMLProviderConfig(ctx context.Context, id string) (*SAMLProviderConfig, error) {
	if err := validateSAMLConfigID(id); err != nil {
		return nil, err
	}

	req := &internal.Request{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("/inboundSamlConfigs/%s", id),
	}
	var result samlProviderConfigDAO
	if _, err := c.makeRequest(ctx, req, &result); err != nil {
		return nil, err
	}

	return result.toSAMLProviderConfig(), nil
}

// CreateSAMLProviderConfig creates a new SAML provider config from the given parameters.
func (c *baseClient) CreateSAMLProviderConfig(ctx context.Context, config *SAMLProviderConfigToCreate) (*SAMLProviderConfig, error) {
	if config == nil {
		return nil, errors.New("config must not be nil")
	}

	body, id, err := config.buildRequest()
	if err != nil {
		return nil, err
	}

	req := &internal.Request{
		Method: http.MethodPost,
		URL:    "/inboundSamlConfigs",
		Body:   internal.NewJSONEntity(body),
		Opts: []internal.HTTPOption{
			internal.WithQueryParam("inboundSamlConfigId", id),
		},
	}
	var result samlProviderConfigDAO
	if _, err := c.makeRequest(ctx, req, &result); err != nil {
		return nil, err
	}

	return result.toSAMLProviderConfig(), nil
}

// UpdateSAMLProviderConfig updates an existing SAML provider config with the given parameters.
func (c *baseClient) UpdateSAMLProviderConfig(ctx context.Context, id string, config *SAMLProviderConfigToUpdate) (*SAMLProviderConfig, error) {
	if err := validateSAMLConfigID(id); err != nil {
		return nil, err
	}
	if config == nil {
		return nil, errors.New("config must not be nil")
	}

	body, err := config.buildRequest()
	if err != nil {
		return nil, err
	}

	mask := body.UpdateMask()
	req := &internal.Request{
		Method: http.MethodPatch,
		URL:    fmt.Sprintf("/inboundSamlConfigs/%s", id),
		Body:   internal.NewJSONEntity(body),
		Opts: []internal.HTTPOption{
			internal.WithQueryParam("updateMask", strings.Join(mask, ",")),
		},
	}
	var result samlProviderConfigDAO
	if _, err := c.makeRequest(ctx, req, &result); err != nil {
		return nil, err
	}

	return result.toSAMLProviderConfig(), nil
}

// DeleteSAMLProviderConfig deletes the SAMLProviderConfig with the given ID.
func (c *baseClient) DeleteSAMLProviderConfig(ctx context.Context, id string) error {
	if err := validateSAMLConfigID(id); err != nil {
		return err
	}

	req := &internal.Request{
		Method: http.MethodDelete,
		URL:    fmt.Sprintf("/inboundSamlConfigs/%s", id),
	}
	_, err := c.makeRequest(ctx, req, nil)
	return err
}

// SAMLProviderConfigs returns an iterator over SAML provider configurations.
//
// If nextPageToken is empty, the iterator will start at the beginning. Otherwise,
// iterator starts after the token.
func (c *baseClient) SAMLProviderConfigs(ctx context.Context, nextPageToken string) *SAMLProviderConfigIterator {
	it := &SAMLProviderConfigIterator{
		ctx:    ctx,
		client: c,
	}
	it.pageInfo, it.nextFunc = iterator.NewPageInfo(
		it.fetch,
		func() int { return len(it.configs) },
		func() interface{} { b := it.configs; it.configs = nil; return b })
	it.pageInfo.MaxSize = maxConfigs
	it.pageInfo.Token = nextPageToken
	return it
}

func (c *baseClient) makeRequest(
	ctx context.Context, req *internal.Request, v interface{}) (*internal.Response, error) {

	if c.projectID == "" {
		return nil, errors.New("project id not available")
	}

	if c.tenantID != "" {
		req.URL = fmt.Sprintf("%s/projects/%s/tenants/%s%s", c.providerConfigEndpoint, c.projectID, c.tenantID, req.URL)
	} else {
		req.URL = fmt.Sprintf("%s/projects/%s%s", c.providerConfigEndpoint, c.projectID, req.URL)
	}

	return c.httpClient.DoAndUnmarshal(ctx, req, v)
}

type oidcProviderConfigDAO struct {
	Name         string                   `json:"name"`
	ClientID     string                   `json:"clientId"`
	Issuer       string                   `json:"issuer"`
	DisplayName  string                   `json:"displayName"`
	Enabled      bool                     `json:"enabled"`
	ClientSecret string                   `json:"clientSecret"`
	ResponseType oidcProviderResponseType `json:"responseType"`
}

type oidcProviderResponseType struct {
	Code    bool `json:"code"`
	IDToken bool `json:"idToken"`
}

func (dao *oidcProviderConfigDAO) toOIDCProviderConfig() *OIDCProviderConfig {
	return &OIDCProviderConfig{
		ID:                  extractResourceID(dao.Name),
		DisplayName:         dao.DisplayName,
		Enabled:             dao.Enabled,
		ClientID:            dao.ClientID,
		Issuer:              dao.Issuer,
		ClientSecret:        dao.ClientSecret,
		CodeResponseType:    dao.ResponseType.Code,
		IDTokenResponseType: dao.ResponseType.IDToken,
	}
}

type idpCertificate struct {
	X509Certificate string `json:"x509Certificate"`
}

type samlProviderConfigDAO struct {
	Name      string `json:"name"`
	IDPConfig struct {
		IDPEntityID     string           `json:"idpEntityId"`
		SSOURL          string           `json:"ssoUrl"`
		IDPCertificates []idpCertificate `json:"idpCertificates"`
		SignRequest     bool             `json:"signRequest"`
	} `json:"idpConfig"`
	SPConfig struct {
		SPEntityID  string `json:"spEntityId"`
		CallbackURI string `json:"callbackUri"`
	} `json:"spConfig"`
	DisplayName string `json:"displayName"`
	Enabled     bool   `json:"enabled"`
}

func (dao *samlProviderConfigDAO) toSAMLProviderConfig() *SAMLProviderConfig {
	var certs []string
	for _, cert := range dao.IDPConfig.IDPCertificates {
		certs = append(certs, cert.X509Certificate)
	}

	return &SAMLProviderConfig{
		ID:                    extractResourceID(dao.Name),
		DisplayName:           dao.DisplayName,
		Enabled:               dao.Enabled,
		IDPEntityID:           dao.IDPConfig.IDPEntityID,
		SSOURL:                dao.IDPConfig.SSOURL,
		RequestSigningEnabled: dao.IDPConfig.SignRequest,
		X509Certificates:      certs,
		RPEntityID:            dao.SPConfig.SPEntityID,
		CallbackURL:           dao.SPConfig.CallbackURI,
	}
}

func validateOIDCConfigID(id string) error {
	if !strings.HasPrefix(id, "oidc.") {
		return fmt.Errorf("invalid OIDC provider id: %q", id)
	}

	return nil
}

func validateSAMLConfigID(id string) error {
	if !strings.HasPrefix(id, "saml.") {
		return fmt.Errorf("invalid SAML provider id: %q", id)
	}

	return nil
}

func extractResourceID(name string) string {
	// name format: "projects/project-id/resource/resource-id"
	segments := strings.Split(name, "/")
	return segments[len(segments)-1]
}
