package plivo

import (
	"encoding/json"
)

// NOTE: All of Plivo's APIs are in a single Go package. Unfortunately,
// this imposes the limitation that each struct type has to be unique across
// all of Plivo's product APIs.

type Country struct {
	Name string `json:"name"`
	ISO2 string `json:"iso2"`
	ISO3 string `json:"iso3"`
}

type NumberFormat struct {
	E164          string `json:"e164"`
	National      string `json:"national"`
	International string `json:"international"`
	RFC3966       string `json:"rfc3966"`
}

type Carrier struct {
	MobileCountryCode string `json:"mobile_country_code"`
	MobileNetworkCode string `json:"mobile_network_code"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	Ported            string `json:"ported"`
}

// LookupResponse is the success response returned by Plivo Lookup API.
type LookupResponse struct {
	ApiID       string        `json:"api_id"`
	PhoneNumber string        `json:"phone_number"`
	Country     *Country      `json:"country"`
	Format      *NumberFormat `json:"format"`
	Carrier     *Carrier      `json:"carrier"`
	ResourceURI string        `json:"resource_uri"`
}

type LookupService struct {
	client *Client
}

// LookupParams is the input parameters for Plivo Lookup API.
type LookupParams struct {
	// If empty, Type defaults to "carrier".
	Type string `url:"type"`
}

// Get looks up a phone number using Plivo Lookup API.
func (s *LookupService) Get(number string, params LookupParams) (*LookupResponse, error) {
	if params.Type == "" {
		params.Type = "carrier"
	}

	req, err := s.client.BaseClient.NewRequest("GET", params, "v1/Number/%s", number)
	if err != nil {
		return nil, err
	}

	resp := new(LookupResponse)
	if err := s.client.ExecuteRequest(req, resp, map[string]interface{}{
		"is_lookup_request": true,
	}); err != nil {
		return nil, s.newError(err.Error())
	}

	return resp, nil
}

// LookupError is the error response returned by Plivo Lookup API.
type LookupError struct {
	respBody  string
	ApiID     string `json:"api_id"`
	ErrorCode int    `json:"error_code"`
	Message   string `json:"message"`
}

// Error returns the raw json/text response from Lookup API.
func (e *LookupError) Error() string {
	return e.respBody
}

func (s *LookupService) newError(body string) *LookupError {
	resp := &LookupError{
		respBody: body,
	}

	_ = json.Unmarshal([]byte(body), resp)
	return resp
}
