package cloudflare

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

// AccessIdentityProvider is the structure of the provider object.
type AccessIdentityProvider struct {
	ID     string                              `json:"id,omitemtpy"`
	Name   string                              `json:"name"`
	Type   string                              `json:"type"`
	Config AccessIdentityProviderConfiguration `json:"config"`
}

// AccessIdentityProviderConfiguration is the combined structure of *all*
// identity provider configuration fields. This is done to simplify the use of
// Access products and their relationship to each other.
//
// API reference: https://developers.cloudflare.com/access/configuring-identity-providers/
type AccessIdentityProviderConfiguration struct {
	AppsDomain         string   `json:"apps_domain,omitempty"`
	Attributes         []string `json:"attributes,omitempty"`
	AuthURL            string   `json:"auth_url,omitempty"`
	CentrifyAccount    string   `json:"centrify_account,omitempty"`
	CentrifyAppID      string   `json:"centrify_app_id,omitempty"`
	CertsURL           string   `json:"certs_url,omitempty"`
	ClientID           string   `json:"client_id,omitempty"`
	ClientSecret       string   `json:"client_secret,omitempty"`
	DirectoryID        string   `json:"directory_id,omitempty"`
	EmailAttributeName string   `json:"email_attribute_name,omitempty"`
	IdpPublicCert      string   `json:"idp_public_cert,omitempty"`
	IssuerURL          string   `json:"issuer_url,omitempty"`
	OktaAccount        string   `json:"okta_account,omitempty"`
	OneloginAccount    string   `json:"onelogin_account,omitempty"`
	RedirectURL        string   `json:"redirect_url,omitempty"`
	SignRequest        bool     `json:"sign_request,omitempty"`
	SsoTargetURL       string   `json:"sso_target_url,omitempty"`
	SupportGroups      bool     `json:"support_groups,omitempty"`
	TokenURL           string   `json:"token_url,omitempty"`
}

// AccessIdentityProvidersListResponse is the API response for multiple
// Access Identity Providers.
type AccessIdentityProvidersListResponse struct {
	Response
	Result   []AccessIdentityProvider         `json:"result"`
}

// AccessIdentityProviderListResponse is the API response for a single
// Access Identity Provider.
type AccessIdentityProviderListResponse struct {
	Response
	Result   AccessIdentityProvider           `json:"result"`
}

// AccessIdentityProviders returns all Access Identity Providers for an
// account.
//
// API reference: https://api.cloudflare.com/#access-identity-providers-list-access-identity-providers
func (api *API) AccessIdentityProviders(accountID string) ([]AccessIdentityProvider, error) {
	uri := "/accounts/" + accountID + "/access/identity_providers"

	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return []AccessIdentityProvider{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessIdentityProviderResponse AccessIdentityProvidersListResponse
	err = json.Unmarshal(res, &accessIdentityProviderResponse)
	if err != nil {
		return []AccessIdentityProvider{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessIdentityProviderResponse.Result, nil
}

// AccessIdentityProviderDetails returns a single Access Identity
// Provider for an account.
//
// API reference: https://api.cloudflare.com/#access-identity-providers-access-identity-providers-details
func (api *API) AccessIdentityProviderDetails(accountID, identityProviderID string) (AccessIdentityProvider, error) {
	uri := fmt.Sprintf(
		"/accounts/%s/access/identity_providers/%s",
		accountID,
		identityProviderID,
	)

	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return AccessIdentityProvider{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessIdentityProviderResponse AccessIdentityProviderListResponse
	err = json.Unmarshal(res, &accessIdentityProviderResponse)
	if err != nil {
		return AccessIdentityProvider{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessIdentityProviderResponse.Result, nil
}

// CreateAccessIdentityProvider creates a new Access Identity Provider.
//
// API reference: https://api.cloudflare.com/#access-identity-providers-create-access-identity-provider
func (api *API) CreateAccessIdentityProvider(accountID string, identityProviderConfiguration AccessIdentityProvider) (AccessIdentityProvider, error) {
	uri := "/accounts/" + accountID + "/access/identity_providers"

	res, err := api.makeRequest("POST", uri, identityProviderConfiguration)
	if err != nil {
		return AccessIdentityProvider{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessIdentityProviderResponse AccessIdentityProviderListResponse
	err = json.Unmarshal(res, &accessIdentityProviderResponse)
	if err != nil {
		return AccessIdentityProvider{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessIdentityProviderResponse.Result, nil
}

// UpdateAccessIdentityProvider updates an existing Access Identity
// Provider.
//
// API reference: https://api.cloudflare.com/#access-identity-providers-create-access-identity-provider
func (api *API) UpdateAccessIdentityProvider(accountID, identityProviderUUID string, identityProviderConfiguration AccessIdentityProvider) (AccessIdentityProvider, error) {
	uri := fmt.Sprintf(
		"/accounts/%s/access/identity_providers/%s",
		accountID,
		identityProviderUUID,
	)

	res, err := api.makeRequest("PUT", uri, identityProviderConfiguration)
	if err != nil {
		return AccessIdentityProvider{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessIdentityProviderResponse AccessIdentityProviderListResponse
	err = json.Unmarshal(res, &accessIdentityProviderResponse)
	if err != nil {
		return AccessIdentityProvider{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessIdentityProviderResponse.Result, nil
}

// DeleteAccessIdentityProvider deletes an Access Identity Provider.
//
// API reference: https://api.cloudflare.com/#access-identity-providers-create-access-identity-provider
func (api *API) DeleteAccessIdentityProvider(accountID, identityProviderUUID string) (AccessIdentityProvider, error) {
	uri := fmt.Sprintf(
		"/accounts/%s/access/identity_providers/%s",
		accountID,
		identityProviderUUID,
	)

	res, err := api.makeRequest("DELETE", uri, nil)
	if err != nil {
		return AccessIdentityProvider{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessIdentityProviderResponse AccessIdentityProviderListResponse
	err = json.Unmarshal(res, &accessIdentityProviderResponse)
	if err != nil {
		return AccessIdentityProvider{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessIdentityProviderResponse.Result, nil
}
