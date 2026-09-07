package plivo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
)

type PhoneNumberComplianceService struct {
	client *Client
}

// ── Request types ────────────────────────────────────────────────────────────

type PhoneNumberComplianceCreateParams struct {
	CountryISO     string                                `json:"country_iso"`
	NumberType     string                                `json:"number_type"`
	Alias          string                                `json:"alias"`
	EndUser        PhoneNumberComplianceEndUserParams     `json:"end_user"`
	Documents      []PhoneNumberComplianceDocumentParams  `json:"documents"`
	CallbackURL    string                                `json:"callback_url,omitempty"`
	CallbackMethod string                                `json:"callback_method,omitempty"`
}

type PhoneNumberComplianceEndUserParams struct {
	Type               string `json:"type"`
	Name               string `json:"name"`
	LastName           string `json:"last_name,omitempty"`
	Email              string `json:"email,omitempty"`
	AddressLine1       string `json:"address_line1,omitempty"`
	AddressLine2       string `json:"address_line2,omitempty"`
	City               string `json:"city,omitempty"`
	State              string `json:"state,omitempty"`
	PostalCode         string `json:"postal_code,omitempty"`
	Country            string `json:"country,omitempty"`
	RegistrationNumber string `json:"registration_number,omitempty"`
}

type PhoneNumberComplianceDocumentParams struct {
	DocumentTypeID string            `json:"document_type_id"`
	DataFields     map[string]string `json:"data_fields,omitempty"`
	File           string            `json:"-"`
}

type PhoneNumberComplianceUpdateParams struct {
	Alias          string                                     `json:"alias,omitempty"`
	EndUser        *PhoneNumberComplianceEndUserUpdateParams   `json:"end_user,omitempty"`
	Documents      []PhoneNumberComplianceDocumentParams       `json:"documents,omitempty"`
	CallbackURL    string                                     `json:"callback_url,omitempty"`
	CallbackMethod string                                     `json:"callback_method,omitempty"`
}

type PhoneNumberComplianceEndUserUpdateParams struct {
	Name               string `json:"name,omitempty"`
	LastName           string `json:"last_name,omitempty"`
	Email              string `json:"email,omitempty"`
	AddressLine1       string `json:"address_line1,omitempty"`
	AddressLine2       string `json:"address_line2,omitempty"`
	City               string `json:"city,omitempty"`
	State              string `json:"state,omitempty"`
	PostalCode         string `json:"postal_code,omitempty"`
	Country            string `json:"country,omitempty"`
	RegistrationNumber string `json:"registration_number,omitempty"`
}

type PhoneNumberComplianceListParams struct {
	Limit      int    `json:"limit,omitempty" url:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty" url:"offset,omitempty"`
	Status     string `json:"status,omitempty" url:"status,omitempty"`
	CountryISO string `json:"country_iso,omitempty" url:"country_iso,omitempty"`
	NumberType string `json:"number_type,omitempty" url:"number_type,omitempty"`
	UserType   string `json:"user_type,omitempty" url:"user_type,omitempty"`
	Alias      string `json:"alias,omitempty" url:"alias,omitempty"`
	Expand     string `json:"expand,omitempty" url:"expand,omitempty"`
}

type PhoneNumberComplianceGetParams struct {
	Expand string `json:"expand,omitempty" url:"expand,omitempty"`
}

// ── Response types ───────────────────────────────────────────────────────────

type PhoneNumberCompliance struct {
	ComplianceID    string                              `json:"compliance_id"`
	Alias           string                              `json:"alias"`
	Status          string                              `json:"status"`
	CountryISO      string                              `json:"country_iso"`
	NumberType      string                              `json:"number_type"`
	UserType        string                              `json:"user_type"`
	CallbackURL     string                              `json:"callback_url,omitempty"`
	CallbackMethod  string                              `json:"callback_method,omitempty"`
	RejectionReason string                              `json:"rejection_reason,omitempty"`
	CreatedAt       string                              `json:"created_at"`
	UpdatedAt       string                              `json:"updated_at"`
	EndUser         *PhoneNumberComplianceEndUser        `json:"end_user,omitempty"`
	Documents       []PhoneNumberComplianceDocument      `json:"documents,omitempty"`
	LinkedNumbers   []PhoneNumberComplianceLinkedNumber   `json:"linked_numbers,omitempty"`
}

type PhoneNumberComplianceEndUser struct {
	EndUserID          string `json:"end_user_id"`
	Type               string `json:"type"`
	Name               string `json:"name"`
	LastName           string `json:"last_name"`
	Email              string `json:"email"`
	AddressLine1       string `json:"address_line1"`
	AddressLine2       string `json:"address_line2"`
	City               string `json:"city"`
	State              string `json:"state"`
	PostalCode         string `json:"postal_code"`
	Country            string `json:"country"`
	RegistrationNumber string `json:"registration_number,omitempty"`
}

type PhoneNumberComplianceDocument struct {
	DocumentID     string                 `json:"document_id"`
	DocumentTypeID string                 `json:"document_type_id"`
	DocumentName   string                 `json:"document_name"`
	FileName       string                 `json:"file_name,omitempty"`
	MetaInformation map[string]interface{} `json:"meta_information,omitempty"`
	DownloadURL    string                 `json:"download_url,omitempty"`
	CreatedAt      string                 `json:"created_at"`
}

type PhoneNumberComplianceLinkedNumber struct {
	Number     string `json:"number"`
	NumberType string `json:"number_type"`
}

type PhoneNumberComplianceCreateResponse struct {
	ApiID        string `json:"api_id"`
	ComplianceID string `json:"compliance_id"`
	Message      string `json:"message"`
}

type PhoneNumberComplianceGetResponse struct {
	ApiID      string                 `json:"api_id"`
	Compliance PhoneNumberCompliance  `json:"compliance"`
}

type PhoneNumberComplianceListResponse struct {
	ApiID       string                  `json:"api_id"`
	Meta        PhoneNumberComplianceMeta `json:"meta"`
	Compliances []PhoneNumberCompliance  `json:"compliances"`
}

type PhoneNumberComplianceMeta struct {
	Limit      int     `json:"limit"`
	Offset     int     `json:"offset"`
	TotalCount int     `json:"total_count"`
	Next       *string `json:"next"`
	Previous   *string `json:"previous"`
}

type PhoneNumberComplianceUpdateResponse struct {
	ApiID      string                `json:"api_id"`
	Message    string                `json:"message"`
	Compliance PhoneNumberCompliance `json:"compliance"`
}

type PhoneNumberComplianceDeleteResponse struct {
	ApiID        string `json:"api_id"`
	ComplianceID string `json:"compliance_id"`
	Message      string `json:"message"`
}

// ── Multipart helper ─────────────────────────────────────────────────────────

func newComplianceMultipartRequest(method, uri string, data interface{}, documents []PhoneNumberComplianceDocumentParams) (*http.Request, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("data", string(jsonBytes)); err != nil {
		return nil, fmt.Errorf("failed to write data field: %w", err)
	}

	for i, doc := range documents {
		if doc.File == "" {
			continue
		}
		file, err := os.Open(doc.File)
		if err != nil {
			return nil, fmt.Errorf("failed to open file for documents[%d]: %w", i, err)
		}
		fileContents, err := ioutil.ReadAll(file)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to read file for documents[%d]: %w", i, err)
		}
		fi, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to stat file for documents[%d]: %w", i, err)
		}
		file.Close()

		fieldName := fmt.Sprintf("documents[%d].file", i)
		part, err := writer.CreateFormFile(fieldName, fi.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to create form file for documents[%d]: %w", i, err)
		}
		if _, err := part.Write(fileContents); err != nil {
			return nil, fmt.Errorf("failed to write file content for documents[%d]: %w", i, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	request, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())

	return request, nil
}

// ── Service methods ──────────────────────────────────────────────────────────

func (service *PhoneNumberComplianceService) Create(params PhoneNumberComplianceCreateParams) (response *PhoneNumberComplianceCreateResponse, err error) {
	requestUrl := *service.client.BaseUrl
	requestUrl.Path = fmt.Sprintf(baseRequestString, fmt.Sprintf("%s/PhoneNumber/Compliance", service.client.AuthId))

	request, err := newComplianceMultipartRequest("POST", requestUrl.String(), params, params.Documents)
	if err != nil {
		return
	}
	request.SetBasicAuth(service.client.AuthId, service.client.AuthToken)

	response = &PhoneNumberComplianceCreateResponse{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func (service *PhoneNumberComplianceService) Get(complianceId string, params PhoneNumberComplianceGetParams) (response *PhoneNumberComplianceGetResponse, err error) {
	req, err := service.client.NewRequest("GET", params, "PhoneNumber/Compliance/%s", complianceId)
	if err != nil {
		return
	}
	response = &PhoneNumberComplianceGetResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *PhoneNumberComplianceService) List(params PhoneNumberComplianceListParams) (response *PhoneNumberComplianceListResponse, err error) {
	req, err := service.client.NewRequest("GET", params, "PhoneNumber/Compliance")
	if err != nil {
		return
	}
	response = &PhoneNumberComplianceListResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *PhoneNumberComplianceService) Update(complianceId string, params PhoneNumberComplianceUpdateParams) (response *PhoneNumberComplianceUpdateResponse, err error) {
	requestUrl := *service.client.BaseUrl
	requestUrl.Path = fmt.Sprintf(baseRequestString, fmt.Sprintf("%s/PhoneNumber/Compliance/%s", service.client.AuthId, complianceId))

	request, err := newComplianceMultipartRequest("PATCH", requestUrl.String(), params, params.Documents)
	if err != nil {
		return
	}
	request.SetBasicAuth(service.client.AuthId, service.client.AuthToken)

	response = &PhoneNumberComplianceUpdateResponse{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func (service *PhoneNumberComplianceService) Delete(complianceId string) (response *PhoneNumberComplianceDeleteResponse, err error) {
	req, err := service.client.NewRequest("DELETE", nil, "PhoneNumber/Compliance/%s", complianceId)
	if err != nil {
		return
	}
	response = &PhoneNumberComplianceDeleteResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}
