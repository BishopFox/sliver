package cloudflare

import (
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// CustomHostnameStatus is the enumeration of valid state values in the CustomHostnameSSL
type CustomHostnameStatus string

const (
	// PENDING status represents state of CustomHostname is pending.
	PENDING CustomHostnameStatus = "pending"
	// ACTIVE status represents state of CustomHostname is active.
	ACTIVE CustomHostnameStatus = "active"
	// MOVED status represents state of CustomHostname is moved.
	MOVED CustomHostnameStatus = "moved"
	// DELETED status represents state of CustomHostname is removed.
	DELETED CustomHostnameStatus = "deleted"
)

// CustomHostnameSSLSettings represents the SSL settings for a custom hostname.
type CustomHostnameSSLSettings struct {
	HTTP2         string   `json:"http2,omitempty"`
	TLS13         string   `json:"tls_1_3,omitempty"`
	MinTLSVersion string   `json:"min_tls_version,omitempty"`
	Ciphers       []string `json:"ciphers,omitempty"`
}

//CustomHostnameOwnershipVerification represents ownership verification status of a given custom hostname.
type CustomHostnameOwnershipVerification struct {
	Type  string `json:"type,omitempty"`
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

//CustomHostnameSSLValidationErrors represents errors that occurred during SSL validation.
type CustomHostnameSSLValidationErrors struct {
	Message string `json:"message,omitempty"`
}

// CustomHostnameSSL represents the SSL section in a given custom hostname.
type CustomHostnameSSL struct {
	Status               string                              `json:"status,omitempty"`
	Method               string                              `json:"method,omitempty"`
	Type                 string                              `json:"type,omitempty"`
	CnameTarget          string                              `json:"cname_target,omitempty"`
	CnameName            string                              `json:"cname,omitempty"`
	Wildcard             bool                                `json:"wildcard,omitempty"`
	CustomCertificate    string                              `json:"custom_certificate,omitempty"`
	CustomKey            string                              `json:"custom_key,omitempty"`
	CertificateAuthority string                              `json:"certificate_authority,omitempty"`
	Settings             CustomHostnameSSLSettings           `json:"settings,omitempty"`
	ValidationErrors     []CustomHostnameSSLValidationErrors `json:"validation_errors,omitempty"`
	HTTPUrl              string                              `json:"http_url,omitempty"`
	HTTPBody             string                              `json:"http_body,omitempty"`
}

// CustomMetadata defines custom metadata for the hostname. This requires logic to be implemented by Cloudflare to act on the data provided.
type CustomMetadata map[string]interface{}

// CustomHostname represents a custom hostname in a zone.
type CustomHostname struct {
	ID                        string                                  `json:"id,omitempty"`
	Hostname                  string                                  `json:"hostname,omitempty"`
	CustomOriginServer        string                                  `json:"custom_origin_server,omitempty"`
	SSL                       CustomHostnameSSL                       `json:"ssl,omitempty"`
	CustomMetadata            CustomMetadata                          `json:"custom_metadata,omitempty"`
	Status                    CustomHostnameStatus                    `json:"status,omitempty"`
	VerificationErrors        []string                                `json:"verification_errors,omitempty"`
	OwnershipVerification     CustomHostnameOwnershipVerification     `json:"ownership_verification,omitempty"`
	OwnershipVerificationHTTP CustomHostnameOwnershipVerificationHTTP `json:"ownership_verification_http,omitempty"`
	CreatedAt                 *time.Time                              `json:"created_at,omitempty"`
}

// CustomHostnameOwnershipVerificationHTTP represents a response from the Custom Hostnames endpoints.
type CustomHostnameOwnershipVerificationHTTP struct {
	HTTPUrl  string `json:"http_url,omitempty"`
	HTTPBody string `json:"http_body,omitempty"`
}

// CustomHostnameResponse represents a response from the Custom Hostnames endpoints.
type CustomHostnameResponse struct {
	Result CustomHostname `json:"result"`
	Response
}

// CustomHostnameListResponse represents a response from the Custom Hostnames endpoints.
type CustomHostnameListResponse struct {
	Result []CustomHostname `json:"result"`
	Response
	ResultInfo `json:"result_info"`
}

// CustomHostnameFallbackOrigin represents a Custom Hostnames Fallback Origin
type CustomHostnameFallbackOrigin struct {
	Origin string   `json:"origin,omitempty"`
	Status string   `json:"status,omitempty"`
	Errors []string `json:"errors,omitempty"`
}

// CustomHostnameFallbackOriginResponse represents a response from the Custom Hostnames Fallback Origin endpoint.
type CustomHostnameFallbackOriginResponse struct {
	Result CustomHostnameFallbackOrigin `json:"result"`
	Response
}

// UpdateCustomHostnameSSL modifies SSL configuration for the given custom
// hostname in the given zone.
//
// API reference: https://api.cloudflare.com/#custom-hostname-for-a-zone-update-custom-hostname-configuration
func (api *API) UpdateCustomHostnameSSL(zoneID string, customHostnameID string, ssl CustomHostnameSSL) (*CustomHostnameResponse, error) {
	uri := "/zones/" + zoneID + "/custom_hostnames/" + customHostnameID
	ch := CustomHostname{
		SSL: ssl,
	}
	res, err := api.makeRequest("PATCH", uri, ch)
	if err != nil {
		return nil, errors.Wrap(err, errMakeRequestError)
	}

	var response *CustomHostnameResponse
	err = json.Unmarshal(res, &response)
	if err != nil {
		return nil, errors.Wrap(err, errUnmarshalError)
	}
	return response, nil
}

// UpdateCustomHostname modifies configuration for the given custom
// hostname in the given zone.
//
// API reference: https://api.cloudflare.com/#custom-hostname-for-a-zone-update-custom-hostname-configuration
func (api *API) UpdateCustomHostname(zoneID string, customHostnameID string, ch CustomHostname) (*CustomHostnameResponse, error) {
	uri := "/zones/" + zoneID + "/custom_hostnames/" + customHostnameID
	res, err := api.makeRequest("PATCH", uri, ch)
	if err != nil {
		return nil, errors.Wrap(err, errMakeRequestError)
	}

	var response *CustomHostnameResponse
	err = json.Unmarshal(res, &response)
	if err != nil {
		return nil, errors.Wrap(err, errUnmarshalError)
	}
	return response, nil
}

// DeleteCustomHostname deletes a custom hostname (and any issued SSL
// certificates).
//
// API reference: https://api.cloudflare.com/#custom-hostname-for-a-zone-delete-a-custom-hostname-and-any-issued-ssl-certificates-
func (api *API) DeleteCustomHostname(zoneID string, customHostnameID string) error {
	uri := "/zones/" + zoneID + "/custom_hostnames/" + customHostnameID
	res, err := api.makeRequest("DELETE", uri, nil)
	if err != nil {
		return errors.Wrap(err, errMakeRequestError)
	}

	var response *CustomHostnameResponse
	err = json.Unmarshal(res, &response)
	if err != nil {
		return errors.Wrap(err, errUnmarshalError)
	}

	return nil
}

// CreateCustomHostname creates a new custom hostname and requests that an SSL certificate be issued for it.
//
// API reference: https://api.cloudflare.com/#custom-hostname-for-a-zone-create-custom-hostname
func (api *API) CreateCustomHostname(zoneID string, ch CustomHostname) (*CustomHostnameResponse, error) {
	uri := "/zones/" + zoneID + "/custom_hostnames"
	res, err := api.makeRequest("POST", uri, ch)
	if err != nil {
		return nil, errors.Wrap(err, errMakeRequestError)
	}

	var response *CustomHostnameResponse
	err = json.Unmarshal(res, &response)
	if err != nil {
		return nil, errors.Wrap(err, errUnmarshalError)
	}

	return response, nil
}

// CustomHostnames fetches custom hostnames for the given zone,
// by applying filter.Hostname if not empty and scoping the result to page'th 50 items.
//
// The returned ResultInfo can be used to implement pagination.
//
// API reference: https://api.cloudflare.com/#custom-hostname-for-a-zone-list-custom-hostnames
func (api *API) CustomHostnames(zoneID string, page int, filter CustomHostname) ([]CustomHostname, ResultInfo, error) {
	v := url.Values{}
	v.Set("per_page", "50")
	v.Set("page", strconv.Itoa(page))
	if filter.Hostname != "" {
		v.Set("hostname", filter.Hostname)
	}
	query := "?" + v.Encode()

	uri := "/zones/" + zoneID + "/custom_hostnames" + query
	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return []CustomHostname{}, ResultInfo{}, errors.Wrap(err, errMakeRequestError)
	}
	var customHostnameListResponse CustomHostnameListResponse
	err = json.Unmarshal(res, &customHostnameListResponse)
	if err != nil {
		return []CustomHostname{}, ResultInfo{}, errors.Wrap(err, errMakeRequestError)
	}

	return customHostnameListResponse.Result, customHostnameListResponse.ResultInfo, nil
}

// CustomHostname inspects the given custom hostname in the given zone.
//
// API reference: https://api.cloudflare.com/#custom-hostname-for-a-zone-custom-hostname-configuration-details
func (api *API) CustomHostname(zoneID string, customHostnameID string) (CustomHostname, error) {
	uri := "/zones/" + zoneID + "/custom_hostnames/" + customHostnameID
	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return CustomHostname{}, errors.Wrap(err, errMakeRequestError)
	}

	var response CustomHostnameResponse
	err = json.Unmarshal(res, &response)
	if err != nil {
		return CustomHostname{}, errors.Wrap(err, errUnmarshalError)
	}

	return response.Result, nil
}

// CustomHostnameIDByName retrieves the ID for the given hostname in the given zone.
func (api *API) CustomHostnameIDByName(zoneID string, hostname string) (string, error) {
	customHostnames, _, err := api.CustomHostnames(zoneID, 1, CustomHostname{Hostname: hostname})
	if err != nil {
		return "", errors.Wrap(err, "CustomHostnames command failed")
	}
	for _, ch := range customHostnames {
		if ch.Hostname == hostname {
			return ch.ID, nil
		}
	}
	return "", errors.New("CustomHostname could not be found")
}

// UpdateCustomHostnameFallbackOrigin modifies the Custom Hostname Fallback origin in the given zone.
//
// API reference: https://api.cloudflare.com/#custom-hostname-fallback-origin-for-a-zone-update-fallback-origin-for-custom-hostnames
func (api *API) UpdateCustomHostnameFallbackOrigin(zoneID string, chfo CustomHostnameFallbackOrigin) (*CustomHostnameFallbackOriginResponse, error) {
	uri := "/zones/" + zoneID + "/custom_hostnames/fallback_origin"
	res, err := api.makeRequest("PUT", uri, chfo)
	if err != nil {
		return nil, errors.Wrap(err, errMakeRequestError)
	}

	var response *CustomHostnameFallbackOriginResponse
	err = json.Unmarshal(res, &response)
	if err != nil {
		return nil, errors.Wrap(err, errUnmarshalError)
	}
	return response, nil
}

// DeleteCustomHostnameFallbackOrigin deletes the Custom Hostname Fallback origin in the given zone.
//
// API reference: https://api.cloudflare.com/#custom-hostname-fallback-origin-for-a-zone-delete-fallback-origin-for-custom-hostnames
func (api *API) DeleteCustomHostnameFallbackOrigin(zoneID string) error {
	uri := "/zones/" + zoneID + "/custom_hostnames/fallback_origin"
	res, err := api.makeRequest("DELETE", uri, nil)
	if err != nil {
		return errors.Wrap(err, errMakeRequestError)
	}

	var response *CustomHostnameFallbackOriginResponse
	err = json.Unmarshal(res, &response)
	if err != nil {
		return errors.Wrap(err, errUnmarshalError)
	}
	return nil
}

// CustomHostnameFallbackOrigin inspects the Custom Hostname Fallback origin in the given zone.
//
// API reference: https://api.cloudflare.com/#custom-hostname-fallback-origin-for-a-zone-properties
func (api *API) CustomHostnameFallbackOrigin(zoneID string) (CustomHostnameFallbackOrigin, error) {
	uri := "/zones/" + zoneID + "/custom_hostnames/fallback_origin"
	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return CustomHostnameFallbackOrigin{}, errors.Wrap(err, errMakeRequestError)
	}

	var response CustomHostnameFallbackOriginResponse
	err = json.Unmarshal(res, &response)
	if err != nil {
		return CustomHostnameFallbackOrigin{}, errors.Wrap(err, errUnmarshalError)
	}

	return response.Result, nil
}
