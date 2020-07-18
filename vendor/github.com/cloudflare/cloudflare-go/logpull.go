package cloudflare

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// LogpullRentionConfiguration describes a the structure of a Logpull Rention
// payload.
type LogpullRentionConfiguration struct {
	Flag bool `json:"flag"`
}

// LogpullRentionConfigurationResponse is the API response, containing the
// Logpull rentention result.
type LogpullRentionConfigurationResponse struct {
	Response
	Result LogpullRentionConfiguration `json:"result"`
}

// GetLogpullRentionFlag gets the current setting flag.
//
// API reference: https://developers.cloudflare.com/logs/logpull-api/enabling-log-retention/
func (api *API) GetLogpullRentionFlag(zoneID string) (*LogpullRentionConfiguration, error) {
	uri := "/zones/" + zoneID + "/logs/control/retention/flag"
	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return &LogpullRentionConfiguration{}, errors.Wrap(err, errMakeRequestError)
	}
	var r LogpullRentionConfigurationResponse
	err = json.Unmarshal(res, &r)
	if err != nil {
		return nil, errors.Wrap(err, errUnmarshalError)
	}
	return &r.Result, nil
}

// SetLogpullRentionFlag updates the retention flag to the defined boolean.
//
// API reference: https://developers.cloudflare.com/logs/logpull-api/enabling-log-retention/
func (api *API) SetLogpullRentionFlag(zoneID string, enabled bool) (*LogpullRentionConfiguration, error) {
	uri := "/zones/" + zoneID + "/logs/control/retention/flag"
	flagPayload := LogpullRentionConfiguration{Flag: enabled}

	res, err := api.makeRequest("POST", uri, flagPayload)
	if err != nil {
		return &LogpullRentionConfiguration{}, errors.Wrap(err, errMakeRequestError)
	}
	var r LogpullRentionConfigurationResponse
	err = json.Unmarshal(res, &r)
	if err != nil {
		return &LogpullRentionConfiguration{}, errors.Wrap(err, errMakeRequestError)
	}
	return &r.Result, nil
}
