package cloudflare

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// AccessPolicy defines a policy for allowing or disallowing access to
// one or more Access applications.
type AccessPolicy struct {
	ID         string     `json:"id,omitempty"`
	Precedence int        `json:"precedence"`
	Decision   string     `json:"decision"`
	CreatedAt  *time.Time `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at"`
	Name       string     `json:"name"`

	// The include policy works like an OR logical operator. The user must
	// satisfy one of the rules.
	Include []interface{} `json:"include"`

	// The exclude policy works like a NOT logical operator. The user must
	// not satisfy all of the rules in exclude.
	Exclude []interface{} `json:"exclude"`

	// The require policy works like a AND logical operator. The user must
	// satisfy all of the rules in require.
	Require []interface{} `json:"require"`
}

// AccessPolicyListResponse represents the response from the list
// access policies endpoint.
type AccessPolicyListResponse struct {
	Result []AccessPolicy `json:"result"`
	Response
	ResultInfo `json:"result_info"`
}

// AccessPolicyDetailResponse is the API response, containing a single
// access policy.
type AccessPolicyDetailResponse struct {
	Success  bool         `json:"success"`
	Errors   []string     `json:"errors"`
	Messages []string     `json:"messages"`
	Result   AccessPolicy `json:"result"`
}

// AccessPolicies returns all access policies for an access application.
//
// API reference: https://api.cloudflare.com/#access-policy-list-access-policies
func (api *API) AccessPolicies(accountID, applicationID string, pageOpts PaginationOptions) ([]AccessPolicy, ResultInfo, error) {
	return api.accessPolicies(accountID, applicationID, pageOpts, AccountRouteRoot)
}

// ZoneLevelAccessPolicies returns all zone level access policies for an access application.
//
// API reference: https://api.cloudflare.com/#zone-level-access-policy-list-access-policies
func (api *API) ZoneLevelAccessPolicies(zoneID, applicationID string, pageOpts PaginationOptions) ([]AccessPolicy, ResultInfo, error) {
	return api.accessPolicies(zoneID, applicationID, pageOpts, ZoneRouteRoot)
}

func (api *API) accessPolicies(id string, applicationID string, pageOpts PaginationOptions, routeRoot RouteRoot) ([]AccessPolicy, ResultInfo, error) {
	v := url.Values{}
	if pageOpts.PerPage > 0 {
		v.Set("per_page", strconv.Itoa(pageOpts.PerPage))
	}
	if pageOpts.Page > 0 {
		v.Set("page", strconv.Itoa(pageOpts.Page))
	}

	uri := fmt.Sprintf(
		"/%s/%s/access/apps/%s/policies",
		routeRoot,
		id,
		applicationID,
	)

	if len(v) > 0 {
		uri = uri + "?" + v.Encode()
	}

	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return []AccessPolicy{}, ResultInfo{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessPolicyListResponse AccessPolicyListResponse
	err = json.Unmarshal(res, &accessPolicyListResponse)
	if err != nil {
		return []AccessPolicy{}, ResultInfo{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessPolicyListResponse.Result, accessPolicyListResponse.ResultInfo, nil
}

// AccessPolicy returns a single policy based on the policy ID.
//
// API reference: https://api.cloudflare.com/#access-policy-access-policy-details
func (api *API) AccessPolicy(accountID, applicationID, policyID string) (AccessPolicy, error) {
	return api.accessPolicy(accountID, applicationID, policyID, AccountRouteRoot)
}

// ZoneLevelAccessPolicy returns a single zone level policy based on the policy ID.
//
// API reference: https://api.cloudflare.com/#zone-level-access-policy-access-policy-details
func (api *API) ZoneLevelAccessPolicy(zoneID, applicationID, policyID string) (AccessPolicy, error) {
	return api.accessPolicy(zoneID, applicationID, policyID, ZoneRouteRoot)
}

func (api *API) accessPolicy(id string, applicationID string, policyID string, routeRoot RouteRoot) (AccessPolicy, error) {
	uri := fmt.Sprintf(
		"/%s/%s/access/apps/%s/policies/%s",
		routeRoot,
		id,
		applicationID,
		policyID,
	)

	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return AccessPolicy{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessPolicyDetailResponse AccessPolicyDetailResponse
	err = json.Unmarshal(res, &accessPolicyDetailResponse)
	if err != nil {
		return AccessPolicy{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessPolicyDetailResponse.Result, nil
}

// CreateAccessPolicy creates a new access policy.
//
// API reference: https://api.cloudflare.com/#access-policy-create-access-policy
func (api *API) CreateAccessPolicy(accountID, applicationID string, accessPolicy AccessPolicy) (AccessPolicy, error) {
	return api.createAccessPolicy(accountID, applicationID, accessPolicy, AccountRouteRoot)
}

// CreateZoneLevelAccessPolicy creates a new zone level access policy.
//
// API reference: https://api.cloudflare.com/#zone-level-access-policy-create-access-policy
func (api *API) CreateZoneLevelAccessPolicy(zoneID, applicationID string, accessPolicy AccessPolicy) (AccessPolicy, error) {
	return api.createAccessPolicy(zoneID, applicationID, accessPolicy, ZoneRouteRoot)
}

func (api *API) createAccessPolicy(id, applicationID string, accessPolicy AccessPolicy, routeRoot RouteRoot) (AccessPolicy, error) {
	uri := fmt.Sprintf(
		"/%s/%s/access/apps/%s/policies",
		routeRoot,
		id,
		applicationID,
	)

	res, err := api.makeRequest("POST", uri, accessPolicy)
	if err != nil {
		return AccessPolicy{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessPolicyDetailResponse AccessPolicyDetailResponse
	err = json.Unmarshal(res, &accessPolicyDetailResponse)
	if err != nil {
		return AccessPolicy{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessPolicyDetailResponse.Result, nil
}

// UpdateAccessPolicy updates an existing access policy.
//
// API reference: https://api.cloudflare.com/#access-policy-update-access-policy
func (api *API) UpdateAccessPolicy(accountID, applicationID string, accessPolicy AccessPolicy) (AccessPolicy, error) {
	return api.updateAccessPolicy(accountID, applicationID, accessPolicy, AccountRouteRoot)
}

// UpdateZoneLevelAccessPolicy updates an existing zone level access policy.
//
// API reference: https://api.cloudflare.com/#zone-level-access-policy-update-access-policy
func (api *API) UpdateZoneLevelAccessPolicy(zoneID, applicationID string, accessPolicy AccessPolicy) (AccessPolicy, error) {
	return api.updateAccessPolicy(zoneID, applicationID, accessPolicy, ZoneRouteRoot)
}

func (api *API) updateAccessPolicy(id, applicationID string, accessPolicy AccessPolicy, routeRoot RouteRoot) (AccessPolicy, error) {
	if accessPolicy.ID == "" {
		return AccessPolicy{}, errors.Errorf("access policy ID cannot be empty")
	}
	uri := fmt.Sprintf(
		"/%s/%s/access/apps/%s/policies/%s",
		routeRoot,
		id,
		applicationID,
		accessPolicy.ID,
	)

	res, err := api.makeRequest("PUT", uri, accessPolicy)
	if err != nil {
		return AccessPolicy{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessPolicyDetailResponse AccessPolicyDetailResponse
	err = json.Unmarshal(res, &accessPolicyDetailResponse)
	if err != nil {
		return AccessPolicy{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessPolicyDetailResponse.Result, nil
}

// DeleteAccessPolicy deletes an access policy.
//
// API reference: https://api.cloudflare.com/#access-policy-update-access-policy
func (api *API) DeleteAccessPolicy(accountID, applicationID, accessPolicyID string) error {
	return api.deleteAccessPolicy(accountID, applicationID, accessPolicyID, AccountRouteRoot)
}

// DeleteZoneLevelAccessPolicy deletes a zone level access policy.
//
// API reference: https://api.cloudflare.com/#zone-level-access-policy-delete-access-policy
func (api *API) DeleteZoneLevelAccessPolicy(zoneID, applicationID, accessPolicyID string) error {
	return api.deleteAccessPolicy(zoneID, applicationID, accessPolicyID, ZoneRouteRoot)
}

func (api *API) deleteAccessPolicy(id, applicationID, accessPolicyID string, routeRoot RouteRoot) error {
	uri := fmt.Sprintf(
		"/%s/%s/access/apps/%s/policies/%s",
		routeRoot,
		id,
		applicationID,
		accessPolicyID,
	)

	_, err := api.makeRequest("DELETE", uri, nil)
	if err != nil {
		return errors.Wrap(err, errMakeRequestError)
	}

	return nil
}
