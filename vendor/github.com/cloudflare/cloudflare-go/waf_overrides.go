package cloudflare

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// WAFOverridesResponse represents the response form the WAF overrides endpoint.
type WAFOverridesResponse struct {
	Response
	Result     []WAFOverride `json:"result"`
	ResultInfo ResultInfo    `json:"result_info"`
}

// WAFOverrideResponse represents the response form the WAF override endpoint.
type WAFOverrideResponse struct {
	Response
	Result     WAFOverride `json:"result"`
	ResultInfo ResultInfo  `json:"result_info"`
}

// WAFOverride represents a WAF override.
type WAFOverride struct {
	ID            string            `json:"id,omitempty"`
	Description   string            `json:"description"`
	URLs          []string          `json:"urls"`
	Priority      int               `json:"priority"`
	Groups        map[string]string `json:"groups"`
	RewriteAction map[string]string `json:"rewrite_action"`
	Rules         map[string]string `json:"rules"`
	Paused        bool              `json:"paused"`
}

// ListWAFOverrides returns a slice of the WAF overrides.
//
// API Reference: https://api.cloudflare.com/#waf-overrides-list-uri-controlled-waf-configurations
func (api *API) ListWAFOverrides(zoneID string) ([]WAFOverride, error) {
	var overrides []WAFOverride
	var res []byte
	var err error

	uri := "/zones/" + zoneID + "/firewall/waf/overrides"
	res, err = api.makeRequest("GET", uri, nil)
	if err != nil {
		return []WAFOverride{}, errors.Wrap(err, errMakeRequestError)
	}

	var r WAFOverridesResponse
	err = json.Unmarshal(res, &r)
	if err != nil {
		return []WAFOverride{}, errors.Wrap(err, errUnmarshalError)
	}

	if !r.Success {
		// TODO: Provide an actual error message instead of always returning nil
		return []WAFOverride{}, err
	}

	for ri := range r.Result {
		overrides = append(overrides, r.Result[ri])
	}
	return overrides, nil
}

// WAFOverride returns a WAF override from the given override ID.
//
// API Reference: https://api.cloudflare.com/#waf-overrides-uri-controlled-waf-configuration-details
func (api *API) WAFOverride(zoneID, overrideID string) (WAFOverride, error) {
	uri := "/zones/" + zoneID + "/firewall/waf/overrides/" + overrideID
	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return WAFOverride{}, errors.Wrap(err, errMakeRequestError)
	}

	var r WAFOverrideResponse
	err = json.Unmarshal(res, &r)
	if err != nil {
		return WAFOverride{}, errors.Wrap(err, errUnmarshalError)
	}

	return r.Result, nil
}

// CreateWAFOverride creates a new WAF override.
//
// API reference: https://api.cloudflare.com/#waf-overrides-create-a-uri-controlled-waf-configuration
func (api *API) CreateWAFOverride(zoneID string, override WAFOverride) (WAFOverride, error) {
	uri := "/zones/" + zoneID + "/firewall/waf/overrides"
	res, err := api.makeRequest("POST", uri, override)
	if err != nil {
		return WAFOverride{}, errors.Wrap(err, errMakeRequestError)
	}
	var r WAFOverrideResponse
	if err := json.Unmarshal(res, &r); err != nil {
		return WAFOverride{}, errors.Wrap(err, errUnmarshalError)
	}
	return r.Result, nil
}

// UpdateWAFOverride updates an existing WAF override.
//
// API reference: https://api.cloudflare.com/#waf-overrides-update-uri-controlled-waf-configuration
func (api *API) UpdateWAFOverride(zoneID, overrideID string, override WAFOverride) (WAFOverride, error) {
	uri := "/zones/" + zoneID + "/firewall/waf/overrides/" + overrideID

	res, err := api.makeRequest("PUT", uri, override)
	if err != nil {
		return WAFOverride{}, errors.Wrap(err, errMakeRequestError)
	}

	var r WAFOverrideResponse
	err = json.Unmarshal(res, &r)
	if err != nil {
		return WAFOverride{}, errors.Wrap(err, errUnmarshalError)
	}

	return r.Result, nil
}

// DeleteWAFOverride deletes a WAF override for a zone.
//
// API reference: https://api.cloudflare.com/#waf-overrides-delete-lockdown-rule
func (api *API) DeleteWAFOverride(zoneID, overrideID string) error {
	uri := "/zones/" + zoneID + "/firewall/waf/overrides/" + overrideID
	res, err := api.makeRequest("DELETE", uri, nil)
	if err != nil {
		return errors.Wrap(err, errMakeRequestError)
	}
	var r WAFOverrideResponse
	err = json.Unmarshal(res, &r)
	if err != nil {
		return errors.Wrap(err, errUnmarshalError)
	}
	return nil
}
