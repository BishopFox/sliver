// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package oauth

import (
	"fmt"
	"strings"

	"go.mau.fi/util/exslices"
	"go.mau.fi/util/jsontime"

	"maunium.net/go/mautrix/id"
)

type AccountManagementAction string

const (
	AccountManagementActionProfile           AccountManagementAction = "org.matrix.profile"
	AccountManagementActionDeviceList        AccountManagementAction = "org.matrix.devices_list"
	AccountManagementActionDeviceView        AccountManagementAction = "org.matrix.device_view"
	AccountManagementActionDeviceDelete      AccountManagementAction = "org.matrix.device_delete"
	AccountManagementActionCrossSigningReset AccountManagementAction = "org.matrix.cross_signing_reset"
	AccountManagementActionAccountDeactivate AccountManagementAction = "org.matrix.account_deactivate"
)

type PromptValue string

const (
	PromptValueLogin  PromptValue = "login"
	PromptValueCreate PromptValue = "create"
)

type ResponseMode string

const (
	ResponseModeQuery    ResponseMode = "query"
	ResponseModeFragment ResponseMode = "fragment"
	ResponseModeFormPost ResponseMode = "form_post"
)

type ResponseType string

const (
	ResponseTypeCode    ResponseType = "code"
	ResponseTypeIDToken ResponseType = "id_token"
)

type GrantType string

const (
	GrantTypeAuthorizationCode GrantType = "authorization_code"
	GrantTypeRefreshToken      GrantType = "refresh_token"
	GrantTypeClientCredentials GrantType = "client_credentials"
	GrantTypeDeviceCode        GrantType = "urn:ietf:params:oauth:grant-type:device_code"
)

type Scope string

const (
	ScopeOpenID    Scope = "openid"
	ScopeEmail     Scope = "email"
	ScopeClientAPI Scope = "urn:matrix:client:api:*"
)

func ScopeDevice(deviceID id.DeviceID) Scope {
	return Scope(fmt.Sprintf("urn:matrix:client:device:%s", deviceID))
}

type CodeChallengeMethod string

const (
	CodeChallengeMethodS256  CodeChallengeMethod = "S256"
	CodeChallengeMethodPlain CodeChallengeMethod = "plain"
)

type AuthMethod string

const (
	AuthMethodClientSecretPost  AuthMethod = "client_secret_post"
	AuthMethodClientSecretBasic AuthMethod = "client_secret_basic"
	AuthMethodClientSecretJWT   AuthMethod = "client_secret_jwt"
	AuthMethodPrivateKeyJWT     AuthMethod = "private_key_jwt"
	AuthMethodNone              AuthMethod = "none"
)

type ServerMetadata struct {
	Issuer                      string `json:"issuer"`
	AuthorizationEndpoint       string `json:"authorization_endpoint"`
	RegistrationEndpoint        string `json:"registration_endpoint"`
	DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint,omitempty"`
	RevocationEndpoint          string `json:"revocation_endpoint"`
	TokenEndpoint               string `json:"token_endpoint"`

	ResponseTypesSupported        []ResponseType        `json:"response_types_supported"`
	ResponseModesSupported        []ResponseMode        `json:"response_modes_supported"`
	GrantTypesSupported           []GrantType           `json:"grant_types_supported"`
	CodeChallengeMethodsSupported []CodeChallengeMethod `json:"code_challenge_methods_supported"`
	PromptValuesSupported         []PromptValue         `json:"prompt_values_supported,omitempty"`

	AccountManagementURI              string                    `json:"account_management_uri,omitempty"`
	AccountManagementActionsSupported []AccountManagementAction `json:"account_management_actions_supported,omitempty"`

	Unrecognized map[string]any `json:",unknown"`
}

type ApplicationType string

const (
	ApplicationTypeNative ApplicationType = "native"
	ApplicationTypeWeb    ApplicationType = "web"
)

type ClientMetadata struct {
	// This is only set in the response by the server
	ClientID string `json:"client_id,omitempty"`

	ApplicationType ApplicationType `json:"application_type,omitempty"`

	ClientName string `json:"client_name,omitempty"`
	ClientURI  string `json:"client_uri"`
	LogoURI    string `json:"logo_uri,omitempty"`
	PolicyURI  string `json:"policy_uri,omitempty"`
	TOSURI     string `json:"tos_uri,omitempty"`

	GrantTypes    []GrantType    `json:"grant_types,omitempty"`
	RedirectURIs  []string       `json:"redirect_uris,omitempty"`
	ResponseTypes []ResponseType `json:"response_types,omitempty"`

	TokenEndpointAuthMethod AuthMethod `json:"token_endpoint_auth_method,omitempty"`

	Extra map[string]any `json:",unknown"`
}

type ScopeGroup string

func (sg ScopeGroup) Split() []Scope {
	return exslices.CastToString[Scope](strings.Split(string(sg), " "))
}

type TokenResponse struct {
	AccessToken  string           `json:"access_token"`
	TokenType    string           `json:"token_type"`
	ExpiresIn    jsontime.Seconds `json:"expires_in"`
	RefreshToken string           `json:"refresh_token,omitempty"`
	Scope        ScopeGroup       `json:"scope,omitempty"`
}

type ScopeList []Scope

func (sl ScopeList) String() string {
	return strings.Join(exslices.CastToString[string](sl), " ")
}

type GetAuthorizationURLParams struct {
	RedirectURI  string       `json:"redirect_uri"`
	Scopes       ScopeList    `json:"scopes"`
	UserIDHint   id.UserID    `json:"user_id_hint,omitempty"`
	ClientID     string       `json:"client_id,omitempty"`
	ResponseMode ResponseMode `json:"response_mode,omitempty"`
}

type GenerateDeviceCodeParams struct {
	Scopes     ScopeList `json:"scopes"`
	UserIDHint id.UserID `json:"user_id_hint,omitempty"`
	ClientID   string    `json:"client_id,omitempty"`
}

type AuthorizationCodeResponse struct {
	State        string `json:"state"`
	CodeVerifier string `json:"code_verifier"`
	URL          string `json:"url"`
}

type ExchangeTokenParams struct {
	CodeVerifier string `json:"code_verifier"`
	RedirectURI  string `json:"redirect_uri"`
	Code         string `json:"code"`
	ClientID     string `json:"client_id,omitempty"`

	StoreCredentials bool `json:"-"`
}

type DeviceCodeResponse struct {
	DeviceCode              string           `json:"device_code"`
	UserCode                string           `json:"user_code"`
	VerificationURI         string           `json:"verification_uri"`
	VerificationURIComplete string           `json:"verification_uri_complete,omitempty"`
	ExpiresIn               jsontime.Seconds `json:"expires_in"`
	Interval                jsontime.Seconds `json:"interval,omitzero"`
}

type PollDeviceCodeParams struct {
	DeviceCode       string `json:"device_code"`
	ClientID         string `json:"client_id,omitempty"`
	StoreCredentials bool   `json:"-"`
}
