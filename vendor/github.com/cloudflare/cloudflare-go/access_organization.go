package cloudflare

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// AccessOrganization represents an Access organization.
type AccessOrganization struct {
	CreatedAt   *time.Time                    `json:"created_at"`
	UpdatedAt   *time.Time                    `json:"updated_at"`
	Name        string                        `json:"name"`
	AuthDomain  string                        `json:"auth_domain"`
	LoginDesign AccessOrganizationLoginDesign `json:"login_design"`
}

// AccessOrganizationLoginDesign represents the login design options.
type AccessOrganizationLoginDesign struct {
	BackgroundColor string `json:"background_color"`
	TextColor       string `json:"text_color"`
	LogoPath        string `json:"logo_path"`
}

// AccessOrganizationListResponse represents the response from the list
// access organization endpoint.
type AccessOrganizationListResponse struct {
	Result AccessOrganization `json:"result"`
	Response
	ResultInfo `json:"result_info"`
}

// AccessOrganizationDetailResponse is the API response, containing a
// single access organization.
type AccessOrganizationDetailResponse struct {
	Success  bool               `json:"success"`
	Errors   []string           `json:"errors"`
	Messages []string           `json:"messages"`
	Result   AccessOrganization `json:"result"`
}

// AccessOrganization returns the Access organisation details.
//
// API reference: https://api.cloudflare.com/#access-organizations-access-organization-details
func (api *API) AccessOrganization(accountID string) (AccessOrganization, ResultInfo, error) {
	return api.accessOrganization(accountID, AccountRouteRoot)
}

// ZoneLevelAccessOrganization returns the zone level Access organisation details.
//
// API reference: https://api.cloudflare.com/#zone-level-access-organizations-access-organization-details
func (api *API) ZoneLevelAccessOrganization(zoneID string) (AccessOrganization, ResultInfo, error) {
	return api.accessOrganization(zoneID, ZoneRouteRoot)
}

func (api *API) accessOrganization(id string, routeRoot RouteRoot) (AccessOrganization, ResultInfo, error) {
	uri := fmt.Sprintf("/%s/%s/access/organizations", routeRoot, id)

	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return AccessOrganization{}, ResultInfo{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessOrganizationListResponse AccessOrganizationListResponse
	err = json.Unmarshal(res, &accessOrganizationListResponse)
	if err != nil {
		return AccessOrganization{}, ResultInfo{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessOrganizationListResponse.Result, accessOrganizationListResponse.ResultInfo, nil
}

// CreateAccessOrganization creates the Access organisation details.
//
// API reference: https://api.cloudflare.com/#access-organizations-create-access-organization
func (api *API) CreateAccessOrganization(accountID string, accessOrganization AccessOrganization) (AccessOrganization, error) {
	return api.createAccessOrganization(accountID, accessOrganization, AccountRouteRoot)
}

// CreateZoneLevelAccessOrganization creates the zone level Access organisation details.
//
// API reference: https://api.cloudflare.com/#zone-level-access-organizations-create-access-organization
func (api *API) CreateZoneLevelAccessOrganization(zoneID string, accessOrganization AccessOrganization) (AccessOrganization, error) {
	return api.createAccessOrganization(zoneID, accessOrganization, ZoneRouteRoot)
}

func (api *API) createAccessOrganization(id string, accessOrganization AccessOrganization, routeRoot RouteRoot) (AccessOrganization, error) {
	uri := fmt.Sprintf("/%s/%s/access/organizations", routeRoot, id)

	res, err := api.makeRequest("POST", uri, accessOrganization)
	if err != nil {
		return AccessOrganization{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessOrganizationDetailResponse AccessOrganizationDetailResponse
	err = json.Unmarshal(res, &accessOrganizationDetailResponse)
	if err != nil {
		return AccessOrganization{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessOrganizationDetailResponse.Result, nil
}

// UpdateAccessOrganization updates the Access organisation details.
//
// API reference: https://api.cloudflare.com/#access-organizations-update-access-organization
func (api *API) UpdateAccessOrganization(accountID string, accessOrganization AccessOrganization) (AccessOrganization, error) {
	return api.updateAccessOrganization(accountID, accessOrganization, AccountRouteRoot)
}

// UpdateZoneLevelAccessOrganization updates the zone level Access organisation details.
//
// API reference: https://api.cloudflare.com/#zone-level-access-organizations-update-access-organization
func (api *API) UpdateZoneLevelAccessOrganization(zoneID string, accessOrganization AccessOrganization) (AccessOrganization, error) {
	return api.updateAccessOrganization(zoneID, accessOrganization, ZoneRouteRoot)
}

func (api *API) updateAccessOrganization(id string, accessOrganization AccessOrganization, routeRoot RouteRoot) (AccessOrganization, error) {
	uri := fmt.Sprintf("/%s/%s/access/organizations", routeRoot, id)

	res, err := api.makeRequest("PUT", uri, accessOrganization)
	if err != nil {
		return AccessOrganization{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessOrganizationDetailResponse AccessOrganizationDetailResponse
	err = json.Unmarshal(res, &accessOrganizationDetailResponse)
	if err != nil {
		return AccessOrganization{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessOrganizationDetailResponse.Result, nil
}
