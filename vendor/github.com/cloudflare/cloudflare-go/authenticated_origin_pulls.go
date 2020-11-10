package cloudflare

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

// AuthenticatedOriginPulls represents global AuthenticatedOriginPulls (tls_client_auth) metadata.
type AuthenticatedOriginPulls struct {
	ID         string    `json:"id"`
	Value      string    `json:"value"`
	Editable   bool      `json:"editable"`
	ModifiedOn time.Time `json:"modified_on"`
}

// AuthenticatedOriginPullsResponse represents the response from the global AuthenticatedOriginPulls (tls_client_auth) details endpoint.
type AuthenticatedOriginPullsResponse struct {
	Response
	Result AuthenticatedOriginPulls `json:"result"`
}

// GetAuthenticatedOriginPullsStatus returns the configuration details for global AuthenticatedOriginPulls (tls_client_auth).
//
// API reference: https://api.cloudflare.com/#zone-settings-get-tls-client-auth-setting
func (api *API) GetAuthenticatedOriginPullsStatus(zoneID string) (AuthenticatedOriginPulls, error) {
	uri := "/zones/" + zoneID + "/settings/tls_client_auth"
	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return AuthenticatedOriginPulls{}, errors.Wrap(err, errMakeRequestError)
	}
	var r AuthenticatedOriginPullsResponse
	if err := json.Unmarshal(res, &r); err != nil {
		return AuthenticatedOriginPulls{}, errors.Wrap(err, errUnmarshalError)
	}
	return r.Result, nil
}

// SetAuthenticatedOriginPullsStatus toggles whether global AuthenticatedOriginPulls is enabled for the zone.
//
// API reference: https://api.cloudflare.com/#zone-settings-change-tls-client-auth-setting
func (api *API) SetAuthenticatedOriginPullsStatus(zoneID string, enable bool) (AuthenticatedOriginPulls, error) {
	uri := "/zones/" + zoneID + "/settings/tls_client_auth"
	var val string
	if enable {
		val = "on"
	} else {
		val = "off"
	}
	params := struct {
		Value string `json:"value"`
	}{
		Value: val,
	}
	res, err := api.makeRequest("PATCH", uri, params)
	if err != nil {
		return AuthenticatedOriginPulls{}, errors.Wrap(err, errMakeRequestError)
	}
	var r AuthenticatedOriginPullsResponse
	if err := json.Unmarshal(res, &r); err != nil {
		return AuthenticatedOriginPulls{}, errors.Wrap(err, errUnmarshalError)
	}
	return r.Result, nil
}
