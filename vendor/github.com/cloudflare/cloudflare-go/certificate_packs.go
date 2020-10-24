package cloudflare

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// CertificatePackGeoRestrictions is for the structure of the geographic
// restrictions for a TLS certificate.
type CertificatePackGeoRestrictions struct {
	Label string `json:"label"`
}

// CertificatePackCertificate is the base structure of a TLS certificate that is
// contained within a certificate pack.
type CertificatePackCertificate struct {
	ID              int                            `json:"id"`
	Hosts           []string                       `json:"hosts"`
	Issuer          string                         `json:"issuer"`
	Signature       string                         `json:"signature"`
	Status          string                         `json:"status"`
	BundleMethod    string                         `json:"bundle_method"`
	GeoRestrictions CertificatePackGeoRestrictions `json:"geo_restrictions"`
	ZoneID          string                         `json:"zone_id"`
	UploadedOn      time.Time                      `json:"uploaded_on"`
	ModifiedOn      time.Time                      `json:"modified_on"`
	ExpiresOn       time.Time                      `json:"expires_on"`
	Priority        int                            `json:"priority"`
}

// CertificatePack is the overarching structure of a certificate pack response.
type CertificatePack struct {
	ID                 string                       `json:"id"`
	Type               string                       `json:"type"`
	Hosts              []string                     `json:"hosts"`
	Certificates       []CertificatePackCertificate `json:"certificates"`
	PrimaryCertificate int                          `json:"primary_certificate"`
}

// CertificatePackRequest is used for requesting a new certificate.
type CertificatePackRequest struct {
	Type  string   `json:"type"`
	Hosts []string `json:"hosts"`
}

// CertificatePackAdvancedCertificate is the structure of the advanced
// certificate pack certificate.
type CertificatePackAdvancedCertificate struct {
	ID                   string   `json:"id"`
	Type                 string   `json:"type"`
	Hosts                []string `json:"hosts"`
	ValidationMethod     string   `json:"validation_method"`
	ValidityDays         int      `json:"validity_days"`
	CertificateAuthority string   `json:"certificate_authority"`
	CloudflareBranding   bool     `json:"cloudflare_branding"`
}

// CertificatePacksResponse is for responses where multiple certificates are
// expected.
type CertificatePacksResponse struct {
	Response
	Result []CertificatePack `json:"result"`
}

// CertificatePacksDetailResponse contains a single certificate pack in the
// response.
type CertificatePacksDetailResponse struct {
	Response
	Result CertificatePack `json:"result"`
}

// CertificatePacksAdvancedDetailResponse contains a single advanced certificate
// pack in the response.
type CertificatePacksAdvancedDetailResponse struct {
	Response
	Result CertificatePackAdvancedCertificate `json:"result"`
}

// ListCertificatePacks returns all available TLS certificate packs for a zone.
//
// API Reference: https://api.cloudflare.com/#certificate-packs-list-certificate-packs
func (api *API) ListCertificatePacks(zoneID string) ([]CertificatePack, error) {
	uri := fmt.Sprintf("/zones/%s/ssl/certificate_packs?status=all", zoneID)
	res, err := api.makeRequest(http.MethodGet, uri, nil)
	if err != nil {
		return []CertificatePack{}, errors.Wrap(err, errMakeRequestError)
	}

	var certificatePacksResponse CertificatePacksResponse
	err = json.Unmarshal(res, &certificatePacksResponse)
	if err != nil {
		return []CertificatePack{}, errors.Wrap(err, errUnmarshalError)
	}

	return certificatePacksResponse.Result, nil
}

// CertificatePack returns a single TLS certificate pack on a zone.
//
// API Reference: https://api.cloudflare.com/#certificate-packs-get-certificate-pack
func (api *API) CertificatePack(zoneID, certificatePackID string) (CertificatePack, error) {
	uri := fmt.Sprintf("/zones/%s/ssl/certificate_packs/%s", zoneID, certificatePackID)
	res, err := api.makeRequest(http.MethodGet, uri, nil)
	if err != nil {
		return CertificatePack{}, errors.Wrap(err, errMakeRequestError)
	}

	var certificatePacksDetailResponse CertificatePacksDetailResponse
	err = json.Unmarshal(res, &certificatePacksDetailResponse)
	if err != nil {
		return CertificatePack{}, errors.Wrap(err, errUnmarshalError)
	}

	return certificatePacksDetailResponse.Result, nil
}

// CreateCertificatePack creates a new certificate pack associated with a zone.
//
// API Reference: https://api.cloudflare.com/#certificate-packs-order-certificate-pack
func (api *API) CreateCertificatePack(zoneID string, cert CertificatePackRequest) (CertificatePack, error) {
	uri := fmt.Sprintf("/zones/%s/ssl/certificate_packs", zoneID)
	res, err := api.makeRequest(http.MethodPost, uri, cert)
	if err != nil {
		return CertificatePack{}, errors.Wrap(err, errMakeRequestError)
	}

	var certificatePacksDetailResponse CertificatePacksDetailResponse
	err = json.Unmarshal(res, &certificatePacksDetailResponse)
	if err != nil {
		return CertificatePack{}, errors.Wrap(err, errUnmarshalError)
	}

	return certificatePacksDetailResponse.Result, nil
}

// DeleteCertificatePack removes a certificate pack associated with a zone.
//
// API Reference: https://api.cloudflare.com/#certificate-packs-delete-advanced-certificate-manager-certificate-pack
func (api *API) DeleteCertificatePack(zoneID, certificateID string) error {
	uri := fmt.Sprintf("/zones/%s/ssl/certificate_packs/%s", zoneID, certificateID)
	_, err := api.makeRequest(http.MethodDelete, uri, nil)
	if err != nil {
		return errors.Wrap(err, errMakeRequestError)
	}

	return nil
}

// CreateAdvancedCertificatePack creates a new certificate pack associated with a zone.
//
// API Reference: https://api.cloudflare.com/#certificate-packs-order-certificate-pack
func (api *API) CreateAdvancedCertificatePack(zoneID string, cert CertificatePackAdvancedCertificate) (CertificatePackAdvancedCertificate, error) {
	uri := fmt.Sprintf("/zones/%s/ssl/certificate_packs/order", zoneID)
	res, err := api.makeRequest(http.MethodPost, uri, cert)
	if err != nil {
		return CertificatePackAdvancedCertificate{}, errors.Wrap(err, errMakeRequestError)
	}

	var advancedCertificatePacksDetailResponse CertificatePacksAdvancedDetailResponse
	err = json.Unmarshal(res, &advancedCertificatePacksDetailResponse)
	if err != nil {
		return CertificatePackAdvancedCertificate{}, errors.Wrap(err, errUnmarshalError)
	}

	return advancedCertificatePacksDetailResponse.Result, nil
}

// RestartAdvancedCertificateValidation kicks off the validation process for a
// pending certificate pack.
//
// API Reference: https://api.cloudflare.com/#certificate-packs-restart-validation-for-advanced-certificate-manager-certificate-pack
func (api *API) RestartAdvancedCertificateValidation(zoneID, certificateID string) (CertificatePackAdvancedCertificate, error) {
	uri := fmt.Sprintf("/zones/%s/ssl/certificate_packs/%s", zoneID, certificateID)
	res, err := api.makeRequest(http.MethodPatch, uri, nil)
	if err != nil {
		return CertificatePackAdvancedCertificate{}, errors.Wrap(err, errMakeRequestError)
	}

	var advancedCertificatePacksDetailResponse CertificatePacksAdvancedDetailResponse
	err = json.Unmarshal(res, &advancedCertificatePacksDetailResponse)
	if err != nil {
		return CertificatePackAdvancedCertificate{}, errors.Wrap(err, errUnmarshalError)
	}

	return advancedCertificatePacksDetailResponse.Result, nil
}
