package cloudflare

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

// PerHostnameAuthenticatedOriginPullsCertificateDetails represents the metadata for a Per Hostname AuthenticatedOriginPulls certificate.
type PerHostnameAuthenticatedOriginPullsCertificateDetails struct {
	ID           string    `json:"id"`
	Certificate  string    `json:"certificate"`
	Issuer       string    `json:"issuer"`
	Signature    string    `json:"signature"`
	SerialNumber string    `json:"serial_number"`
	ExpiresOn    time.Time `json:"expires_on"`
	Status       string    `json:"status"`
	UploadedOn   time.Time `json:"uploaded_on"`
}

// PerHostnameAuthenticatedOriginPullsCertificateResponse represents the response from endpoints relating to creating and deleting a Per Hostname AuthenticatedOriginPulls certificate.
type PerHostnameAuthenticatedOriginPullsCertificateResponse struct {
	Response
	Result PerHostnameAuthenticatedOriginPullsCertificateDetails `json:"result"`
}

// PerHostnameAuthenticatedOriginPullsDetails contains metadata about the Per Hostname AuthenticatedOriginPulls configuration on a hostname.
type PerHostnameAuthenticatedOriginPullsDetails struct {
	Hostname       string    `json:"hostname"`
	CertID         string    `json:"cert_id"`
	Enabled        bool      `json:"enabled"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CertStatus     string    `json:"cert_status"`
	Issuer         string    `json:"issuer"`
	Signature      string    `json:"signature"`
	SerialNumber   string    `json:"serial_number"`
	Certificate    string    `json:"certificate"`
	CertUploadedOn time.Time `json:"cert_uploaded_on"`
	CertUpdatedAt  time.Time `json:"cert_updated_at"`
	ExpiresOn      time.Time `json:"expires_on"`
}

// PerHostnameAuthenticatedOriginPullsDetailsResponse represents Per Hostname AuthenticatedOriginPulls configuration metadata for a single hostname.
type PerHostnameAuthenticatedOriginPullsDetailsResponse struct {
	Response
	Result PerHostnameAuthenticatedOriginPullsDetails `json:"result"`
}

// PerHostnamesAuthenticatedOriginPullsDetailsResponse represents Per Hostname AuthenticatedOriginPulls configuration metadata for multiple hostnames.
type PerHostnamesAuthenticatedOriginPullsDetailsResponse struct {
	Response
	Result []PerHostnameAuthenticatedOriginPullsDetails `json:"result"`
}

// PerHostnameAuthenticatedOriginPullsCertificateParams represents the required data related to the client certificate being uploaded to be used in Per Hostname AuthenticatedOriginPulls.
type PerHostnameAuthenticatedOriginPullsCertificateParams struct {
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
}

// PerHostnameAuthenticatedOriginPullsConfig represents the config state for Per Hostname AuthenticatedOriginPulls applied on a hostname.
type PerHostnameAuthenticatedOriginPullsConfig struct {
	Hostname string `json:"hostname"`
	CertID   string `json:"cert_id"`
	Enabled  bool   `json:"enabled"`
}

// PerHostnameAuthenticatedOriginPullsConfigParams represents the expected config param format for Per Hostname AuthenticatedOriginPulls applied on a hostname.
type PerHostnameAuthenticatedOriginPullsConfigParams struct {
	Config []PerHostnameAuthenticatedOriginPullsConfig `json:"config"`
}

// UploadPerHostnameAuthenticatedOriginPullsCertificate will upload the provided certificate and private key to the edge under Per Hostname AuthenticatedOriginPulls.
//
// API reference: https://api.cloudflare.com/#per-hostname-authenticated-origin-pull-upload-a-hostname-client-certificate
func (api *API) UploadPerHostnameAuthenticatedOriginPullsCertificate(zoneID string, params PerHostnameAuthenticatedOriginPullsCertificateParams) (PerHostnameAuthenticatedOriginPullsCertificateDetails, error) {
	uri := "/zones/" + zoneID + "/origin_tls_client_auth/hostnames/certificates"
	res, err := api.makeRequest("POST", uri, params)
	if err != nil {
		return PerHostnameAuthenticatedOriginPullsCertificateDetails{}, errors.Wrap(err, errMakeRequestError)
	}
	var r PerHostnameAuthenticatedOriginPullsCertificateResponse
	if err := json.Unmarshal(res, &r); err != nil {
		return PerHostnameAuthenticatedOriginPullsCertificateDetails{}, errors.Wrap(err, errUnmarshalError)
	}
	return r.Result, nil
}

// GetPerHostnameAuthenticatedOriginPullsCertificate retrieves certificate metadata about the requested Per Hostname certificate.
//
// API reference: https://api.cloudflare.com/#per-hostname-authenticated-origin-pull-get-the-hostname-client-certificate
func (api *API) GetPerHostnameAuthenticatedOriginPullsCertificate(zoneID, certificateID string) (PerHostnameAuthenticatedOriginPullsCertificateDetails, error) {
	uri := "/zones/" + zoneID + "/origin_tls_client_auth/hostnames/certificates/" + certificateID
	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return PerHostnameAuthenticatedOriginPullsCertificateDetails{}, errors.Wrap(err, errMakeRequestError)
	}
	var r PerHostnameAuthenticatedOriginPullsCertificateResponse
	if err := json.Unmarshal(res, &r); err != nil {
		return PerHostnameAuthenticatedOriginPullsCertificateDetails{}, errors.Wrap(err, errUnmarshalError)
	}
	return r.Result, nil
}

// DeletePerHostnameAuthenticatedOriginPullsCertificate will remove the requested Per Hostname certificate from the edge.
//
// API reference: https://api.cloudflare.com/#per-hostname-authenticated-origin-pull-delete-hostname-client-certificate
func (api *API) DeletePerHostnameAuthenticatedOriginPullsCertificate(zoneID, certificateID string) (PerHostnameAuthenticatedOriginPullsCertificateDetails, error) {
	uri := "/zones/" + zoneID + "/origin_tls_client_auth/hostnames/certificates/" + certificateID
	res, err := api.makeRequest("DELETE", uri, nil)
	if err != nil {
		return PerHostnameAuthenticatedOriginPullsCertificateDetails{}, errors.Wrap(err, errMakeRequestError)
	}
	var r PerHostnameAuthenticatedOriginPullsCertificateResponse
	if err := json.Unmarshal(res, &r); err != nil {
		return PerHostnameAuthenticatedOriginPullsCertificateDetails{}, errors.Wrap(err, errUnmarshalError)
	}
	return r.Result, nil
}

// EditPerHostnameAuthenticatedOriginPullsConfig applies the supplied Per Hostname AuthenticatedOriginPulls config onto a hostname(s) in the edge.
//
// API reference: https://api.cloudflare.com/#per-hostname-authenticated-origin-pull-enable-or-disable-a-hostname-for-client-authentication
func (api *API) EditPerHostnameAuthenticatedOriginPullsConfig(zoneID string, config []PerHostnameAuthenticatedOriginPullsConfig) ([]PerHostnameAuthenticatedOriginPullsDetails, error) {
	uri := "/zones/" + zoneID + "/origin_tls_client_auth/hostnames"
	conf := PerHostnameAuthenticatedOriginPullsConfigParams{
		Config: config,
	}
	res, err := api.makeRequest("PUT", uri, conf)
	if err != nil {
		return []PerHostnameAuthenticatedOriginPullsDetails{}, errors.Wrap(err, errMakeRequestError)
	}
	var r PerHostnamesAuthenticatedOriginPullsDetailsResponse
	if err := json.Unmarshal(res, &r); err != nil {
		return []PerHostnameAuthenticatedOriginPullsDetails{}, errors.Wrap(err, errUnmarshalError)
	}
	return r.Result, nil
}

// GetPerHostnameAuthenticatedOriginPullsConfig returns the config state of Per Hostname AuthenticatedOriginPulls of the provided hostname within a zone.
//
// API reference: https://api.cloudflare.com/#per-hostname-authenticated-origin-pull-get-the-hostname-status-for-client-authentication
func (api *API) GetPerHostnameAuthenticatedOriginPullsConfig(zoneID, hostname string) (PerHostnameAuthenticatedOriginPullsDetails, error) {
	uri := "/zones/" + zoneID + "/origin_tls_client_auth/hostnames/" + hostname
	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return PerHostnameAuthenticatedOriginPullsDetails{}, errors.Wrap(err, errMakeRequestError)
	}
	var r PerHostnameAuthenticatedOriginPullsDetailsResponse
	if err := json.Unmarshal(res, &r); err != nil {
		return PerHostnameAuthenticatedOriginPullsDetails{}, errors.Wrap(err, errUnmarshalError)
	}
	return r.Result, nil
}
