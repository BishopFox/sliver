package cloudflare

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// AccessApplication represents an Access application.
type AccessApplication struct {
	ID                     string                        `json:"id,omitempty"`
	CreatedAt              *time.Time                    `json:"created_at,omitempty"`
	UpdatedAt              *time.Time                    `json:"updated_at,omitempty"`
	AUD                    string                        `json:"aud,omitempty"`
	Name                   string                        `json:"name"`
	Domain                 string                        `json:"domain"`
	SessionDuration        string                        `json:"session_duration,omitempty"`
	AutoRedirectToIdentity bool                          `json:"auto_redirect_to_identity,omitempty"`
	EnableBindingCookie    bool                          `json:"enable_binding_cookie,omitempty"`
	AllowedIdps            []string                      `json:"allowed_idps,omitempty"`
	CorsHeaders            *AccessApplicationCorsHeaders `json:"cors_headers,omitempty"`
}

// AccessApplicationCorsHeaders represents the CORS HTTP headers for an Access
// Application.
type AccessApplicationCorsHeaders struct {
	AllowedMethods   []string `json:"allowed_methods,omitempty"`
	AllowedOrigins   []string `json:"allowed_origins,omitempty"`
	AllowedHeaders   []string `json:"allowed_headers,omitempty"`
	AllowAllMethods  bool     `json:"allow_all_methods,omitempty"`
	AllowAllHeaders  bool     `json:"allow_all_headers,omitempty"`
	AllowAllOrigins  bool     `json:"allow_all_origins,omitempty"`
	AllowCredentials bool     `json:"allow_credentials,omitempty"`
	MaxAge           int      `json:"max_age,omitempty"`
}

// AccessApplicationListResponse represents the response from the list
// access applications endpoint.
type AccessApplicationListResponse struct {
	Result []AccessApplication `json:"result"`
	Response
	ResultInfo `json:"result_info"`
}

// AccessApplicationDetailResponse is the API response, containing a single
// access application.
type AccessApplicationDetailResponse struct {
	Success  bool              `json:"success"`
	Errors   []string          `json:"errors"`
	Messages []string          `json:"messages"`
	Result   AccessApplication `json:"result"`
}

// AccessApplications returns all applications within an account.
//
// API reference: https://api.cloudflare.com/#access-applications-list-access-applications
func (api *API) AccessApplications(accountID string, pageOpts PaginationOptions) ([]AccessApplication, ResultInfo, error) {
	return api.accessApplications(accountID, pageOpts, AccountRouteRoot)
}

// ZoneLevelAccessApplications returns all applications within a zone.
//
// API reference: https://api.cloudflare.com/#zone-level-access-applications-list-access-applications
func (api *API) ZoneLevelAccessApplications(zoneID string, pageOpts PaginationOptions) ([]AccessApplication, ResultInfo, error) {
	return api.accessApplications(zoneID, pageOpts, ZoneRouteRoot)
}

func (api *API) accessApplications(id string, pageOpts PaginationOptions, routeRoot RouteRoot) ([]AccessApplication, ResultInfo, error) {
	v := url.Values{}
	if pageOpts.PerPage > 0 {
		v.Set("per_page", strconv.Itoa(pageOpts.PerPage))
	}
	if pageOpts.Page > 0 {
		v.Set("page", strconv.Itoa(pageOpts.Page))
	}

	uri := fmt.Sprintf("/%s/%s/access/apps", routeRoot, id)
	if len(v) > 0 {
		uri = uri + "?" + v.Encode()
	}

	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return []AccessApplication{}, ResultInfo{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessApplicationListResponse AccessApplicationListResponse
	err = json.Unmarshal(res, &accessApplicationListResponse)
	if err != nil {
		return []AccessApplication{}, ResultInfo{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessApplicationListResponse.Result, accessApplicationListResponse.ResultInfo, nil
}

// AccessApplication returns a single application based on the
// application ID.
//
// API reference: https://api.cloudflare.com/#access-applications-access-applications-details
func (api *API) AccessApplication(accountID, applicationID string) (AccessApplication, error) {
	return api.accessApplication(accountID, applicationID, AccountRouteRoot)
}

// ZoneLevelAccessApplication returns a single zone level application based on the
// application ID.
//
// API reference: https://api.cloudflare.com/#zone-level-access-applications-access-applications-details
func (api *API) ZoneLevelAccessApplication(zoneID, applicationID string) (AccessApplication, error) {
	return api.accessApplication(zoneID, applicationID, ZoneRouteRoot)
}

func (api *API) accessApplication(id, applicationID string, routeRoot RouteRoot) (AccessApplication, error) {
	uri := fmt.Sprintf(
		"/%s/%s/access/apps/%s",
		routeRoot,
		id,
		applicationID,
	)

	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return AccessApplication{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessApplicationDetailResponse AccessApplicationDetailResponse
	err = json.Unmarshal(res, &accessApplicationDetailResponse)
	if err != nil {
		return AccessApplication{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessApplicationDetailResponse.Result, nil
}

// CreateAccessApplication creates a new access application.
//
// API reference: https://api.cloudflare.com/#access-applications-create-access-application
func (api *API) CreateAccessApplication(accountID string, accessApplication AccessApplication) (AccessApplication, error) {
	return api.createAccessApplication(accountID, accessApplication, AccountRouteRoot)
}

// CreateZoneLevelAccessApplication creates a new zone level access application.
//
// API reference: https://api.cloudflare.com/#zone-level-access-applications-create-access-application
func (api *API) CreateZoneLevelAccessApplication(zoneID string, accessApplication AccessApplication) (AccessApplication, error) {
	return api.createAccessApplication(zoneID, accessApplication, ZoneRouteRoot)
}

func (api *API) createAccessApplication(id string, accessApplication AccessApplication, routeRoot RouteRoot) (AccessApplication, error) {
	uri := fmt.Sprintf("/%s/%s/access/apps", routeRoot, id)

	res, err := api.makeRequest("POST", uri, accessApplication)
	if err != nil {
		return AccessApplication{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessApplicationDetailResponse AccessApplicationDetailResponse
	err = json.Unmarshal(res, &accessApplicationDetailResponse)
	if err != nil {
		return AccessApplication{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessApplicationDetailResponse.Result, nil
}

// UpdateAccessApplication updates an existing access application.
//
// API reference: https://api.cloudflare.com/#access-applications-update-access-application
func (api *API) UpdateAccessApplication(accountID string, accessApplication AccessApplication) (AccessApplication, error) {
	return api.updateAccessApplication(accountID, accessApplication, AccountRouteRoot)
}

// UpdateZoneLevelAccessApplication updates an existing zone level access application.
//
// API reference: https://api.cloudflare.com/#zone-level-access-applications-update-access-application
func (api *API) UpdateZoneLevelAccessApplication(zoneID string, accessApplication AccessApplication) (AccessApplication, error) {
	return api.updateAccessApplication(zoneID, accessApplication, ZoneRouteRoot)
}

func (api *API) updateAccessApplication(id string, accessApplication AccessApplication, routeRoot RouteRoot) (AccessApplication, error) {
	if accessApplication.ID == "" {
		return AccessApplication{}, errors.Errorf("access application ID cannot be empty")
	}

	uri := fmt.Sprintf(
		"/%s/%s/access/apps/%s",
		routeRoot,
		id,
		accessApplication.ID,
	)

	res, err := api.makeRequest("PUT", uri, accessApplication)
	if err != nil {
		return AccessApplication{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessApplicationDetailResponse AccessApplicationDetailResponse
	err = json.Unmarshal(res, &accessApplicationDetailResponse)
	if err != nil {
		return AccessApplication{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessApplicationDetailResponse.Result, nil
}

// DeleteAccessApplication deletes an access application.
//
// API reference: https://api.cloudflare.com/#access-applications-delete-access-application
func (api *API) DeleteAccessApplication(accountID, applicationID string) error {
	return api.deleteAccessApplication(accountID, applicationID, AccountRouteRoot)
}

// DeleteZoneLevelAccessApplication deletes a zone level access application.
//
// API reference: https://api.cloudflare.com/#zone-level-access-applications-delete-access-application
func (api *API) DeleteZoneLevelAccessApplication(zoneID, applicationID string) error {
	return api.deleteAccessApplication(zoneID, applicationID, ZoneRouteRoot)
}

func (api *API) deleteAccessApplication(id, applicationID string, routeRoot RouteRoot) error {
	uri := fmt.Sprintf(
		"/%s/%s/access/apps/%s",
		routeRoot,
		id,
		applicationID,
	)

	_, err := api.makeRequest("DELETE", uri, nil)
	if err != nil {
		return errors.Wrap(err, errMakeRequestError)
	}

	return nil
}

// RevokeAccessApplicationTokens revokes tokens associated with an
// access application.
//
// API reference: https://api.cloudflare.com/#access-applications-revoke-access-tokens
func (api *API) RevokeAccessApplicationTokens(accountID, applicationID string) error {
	return api.revokeAccessApplicationTokens(accountID, applicationID, AccountRouteRoot)
}

// RevokeZoneLevelAccessApplicationTokens revokes tokens associated with a zone level
// access application.
//
// API reference: https://api.cloudflare.com/#zone-level-access-applications-revoke-access-tokens
func (api *API) RevokeZoneLevelAccessApplicationTokens(zoneID, applicationID string) error {
	return api.revokeAccessApplicationTokens(zoneID, applicationID, ZoneRouteRoot)
}

func (api *API) revokeAccessApplicationTokens(id string, applicationID string, routeRoot RouteRoot) error {
	uri := fmt.Sprintf(
		"/%s/%s/access/apps/%s/revoke-tokens",
		routeRoot,
		id,
		applicationID,
	)

	_, err := api.makeRequest("POST", uri, nil)
	if err != nil {
		return errors.Wrap(err, errMakeRequestError)
	}

	return nil
}
