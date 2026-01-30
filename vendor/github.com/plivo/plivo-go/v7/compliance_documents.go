package plivo

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"
)

type ComplianceDocumentService struct {
	client *Client
}

type GetComplianceDocumentResponse struct {
	APIID           string `json:"api_id"`
	DocumentID      string `json:"document_id"`
	EndUserID       string `json:"end_user_id"`
	DocumentTypeID  string `json:"document_type_id"`
	Alias           string `json:"alias"`
	FileName        string `json:"file_name,omitempty"`
	MetaInformation struct {
		LastName                     string `json:"last_name,omitempty"`
		FirstName                    string `json:"first_name,omitempty"`
		DateOfBirth                  string `json:"date_of_birth,omitempty"`
		AddressLine1                 string `json:"address_line_1,omitempty"`
		AddressLine2                 string `json:"address_line_2,omitempty"`
		City                         string `json:"city,omitempty"`
		Country                      string `json:"country,omitempty"`
		PostalCode                   string `json:"postal_code,omitempty"`
		UniqueIdentificationNumber   string `json:"unique_identification_number,omitempty"`
		Nationality                  string `json:"nationality,omitempty"`
		PlaceOfBirth                 string `json:"place_of_birth,omitempty"`
		DateOfIssue                  string `json:"date_of_issue,omitempty"`
		DateOfExpiration             string `json:"date_of_expiration,omitempty"`
		TypeOfUtility                string `json:"type_of_utility,omitempty"`
		BillingId                    string `json:"billing_id,omitempty"`
		BillingDate                  string `json:"billing_date,omitempty"`
		BusinessName                 string `json:"business_name,omitempty"`
		TypeOfId                     string `json:"type_of_id,omitempty"`
		SupportPhoneNumber           string `json:"support_phone_number,omitempty"`
		SupportEmail                 string `json:"support_email,omitempty"`
		AuthorizedRepresentativeName string `json:"authorized_representative_name,omitempty"`
		BillDate                     string `json:"bill_date,omitempty"`
		BillId                       string `json:"bill_id,omitempty"`
		UseCaseDescription           string `json:"use_case_description,omitempty"`
	} `json:"meta_information"`
	CreatedAt string `json:"created_at"`
}

type ComplianceDocumentListParams struct {
	Limit          int    `json:"limit,omitempty" url:"limit,omitempty"`
	Offset         int    `json:"offset,omitempty" url:"offset,omitempty"`
	EndUserID      string `json:"end_user_id,omitempty" url:"end_user_id,omitempty"`
	DocumentTypeID string `json:"document_type_id,omitempty" url:"document_type_id,omitempty"`
	Alias          string `json:"alias,omitempty" url:"alias,omitempty"`
}

type ListComplianceDocumentResponse struct {
	APIID string `json:"api_id"`
	Meta  struct {
		Limit      int         `json:"limit"`
		Next       string      `json:"next"`
		Offset     int         `json:"offset"`
		Previous   interface{} `json:"previous"`
		TotalCount int         `json:"total_count"`
	} `json:"meta"`
	Objects []struct {
		CreatedAt            time.Time `json:"created_at"`
		ComplianceDocumentID string    `json:"compliance_document_id"`
		Alias                string    `json:"alias"`
		MetaInformation      struct {
			LastName                     string `json:"last_name,omitempty"`
			FirstName                    string `json:"first_name,omitempty"`
			DateOfBirth                  string `json:"date_of_birth,omitempty"`
			AddressLine1                 string `json:"address_line_1,omitempty"`
			AddressLine2                 string `json:"address_line_2,omitempty"`
			City                         string `json:"city,omitempty"`
			Country                      string `json:"country,omitempty"`
			PostalCode                   string `json:"postal_code,omitempty"`
			UniqueIdentificationNumber   string `json:"unique_identification_number,omitempty"`
			Nationality                  string `json:"nationality,omitempty"`
			PlaceOfBirth                 string `json:"place_of_birth,omitempty"`
			DateOfIssue                  string `json:"date_of_issue,omitempty"`
			DateOfExpiration             string `json:"date_of_expiration,omitempty"`
			TypeOfUtility                string `json:"type_of_utility,omitempty"`
			BillingId                    string `json:"billing_id,omitempty"`
			BillingDate                  string `json:"billing_date,omitempty"`
			BusinessName                 string `json:"business_name,omitempty"`
			TypeOfId                     string `json:"type_of_id,omitempty"`
			SupportPhoneNumber           string `json:"support_phone_number,omitempty"`
			SupportEmail                 string `json:"support_email,omitempty"`
			AuthorizedRepresentativeName string `json:"authorized_representative_name,omitempty"`
			BillDate                     string `json:"bill_date,omitempty"`
			BillId                       string `json:"bill_id,omitempty"`
			UseCaseDescription           string `json:"use_case_description,omitempty"`
		} `json:"meta_information"`
		File           string `json:"file,omitempty"`
		EndUserID      string `json:"end_user_id"`
		DocumentTypeID string `json:"document_type_id"`
	} `json:"objects"`
}

type CreateComplianceDocumentParams struct {
	File                         string `json:"file,omitempty"`
	EndUserID                    string `json:"end_user_id,omitempty"`
	DocumentTypeID               string `json:"document_type_id,omitempty"`
	Alias                        string `json:"alias,omitempty"`
	LastName                     string `json:"last_name,omitempty"`
	FirstName                    string `json:"first_name,omitempty"`
	DateOfBirth                  string `json:"date_of_birth,omitempty"`
	AddressLine1                 string `json:"address_line_1,omitempty"`
	AddressLine2                 string `json:"address_line_2,omitempty"`
	City                         string `json:"city,omitempty"`
	Country                      string `json:"country,omitempty"`
	PostalCode                   string `json:"postal_code,omitempty"`
	UniqueIdentificationNumber   string `json:"unique_identification_number,omitempty"`
	Nationality                  string `json:"nationality,omitempty"`
	PlaceOfBirth                 string `json:"place_of_birth,omitempty"`
	DateOfIssue                  string `json:"date_of_issue,omitempty"`
	DateOfExpiration             string `json:"date_of_expiration,omitempty"`
	TypeOfUtility                string `json:"type_of_utility,omitempty"`
	BillingId                    string `json:"billing_id,omitempty"`
	BillingDate                  string `json:"billing_date,omitempty"`
	BusinessName                 string `json:"business_name,omitempty"`
	TypeOfId                     string `json:"type_of_id,omitempty"`
	SupportPhoneNumber           string `json:"support_phone_number,omitempty"`
	SupportEmail                 string `json:"support_email,omitempty"`
	AuthorizedRepresentativeName string `json:"authorized_representative_name,omitempty"`
	BillDate                     string `json:"bill_date,omitempty"`
	BillId                       string `json:"bill_id,omitempty"`
	UseCaseDescription           string `json:"use_case_description,omitempty"`
}

type UpdateComplianceDocumentParams struct {
	ComplianceDocumentID string `json:"compliance_document_id"`
	CreateComplianceDocumentParams
}

type UpdateComplianceDocumentResponse BaseResponse

func (service *ComplianceDocumentService) Get(complianceDocumentId string) (response *GetComplianceDocumentResponse, err error) {
	req, err := service.client.NewRequest("GET", nil, "ComplianceDocument/%s", complianceDocumentId)
	if err != nil {
		return
	}
	response = &GetComplianceDocumentResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *ComplianceDocumentService) List(params ComplianceDocumentListParams) (response *ListComplianceDocumentResponse, err error) {
	request, err := service.client.NewRequest("GET", params, "ComplianceDocument")
	if err != nil {
		return
	}
	response = &ListComplianceDocumentResponse{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func newfileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	if path != "" {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		fileContents, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
		fi, err := file.Stat()
		if err != nil {
			return nil, err
		}
		file.Close()

		part, err := writer.CreateFormFile(paramName, fi.Name())
		if err != nil {
			return nil, err
		}
		if _, err := part.Write(fileContents); err != nil {
			return nil, err
		}
	}

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err := writer.Close()
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("POST", uri, body)
	if err != nil {
		return request, err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())

	return request, nil
}

func (service *ComplianceDocumentService) Create(params CreateComplianceDocumentParams) (response *GetComplianceDocumentResponse, err error) {
	requestUrl := service.client.BaseUrl
	requestUrl.Path = fmt.Sprintf(baseRequestString, fmt.Sprintf(service.client.AuthId+"/ComplianceDocument"))

	requestParams := make(map[string]string)
	fields := reflect.TypeOf(params)
	values := reflect.ValueOf(params)
	num := fields.NumField()
	for i := 0; i < num; i++ {
		field := strings.Split(fields.Field(i).Tag.Get("json"), ",")[0]
		value := values.Field(i)
		if field != "file" {
			requestParams[field] = value.String()
		}
	}

	request, err := newfileUploadRequest(requestUrl.String(), requestParams, "file", params.File)
	//request, err := service.client.NewRequest("POST", params, "ComplianceDocument/")
	if err != nil {
		return
	}
	request.SetBasicAuth(service.client.AuthId, service.client.AuthToken)
	response = &GetComplianceDocumentResponse{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func (service *ComplianceDocumentService) Update(params UpdateComplianceDocumentParams) (response *UpdateComplianceDocumentResponse, err error) {
	requestUrl := service.client.BaseUrl
	requestUrl.Path = fmt.Sprintf(baseRequestString, fmt.Sprintf(service.client.AuthId+"/ComplianceDocument/"+params.ComplianceDocumentID))

	requestParams := make(map[string]string)

	fields := reflect.TypeOf(params)
	values := reflect.ValueOf(params)
	num := fields.NumField()
	for i := 0; i < num; i++ {
		field := strings.Split(fields.Field(i).Tag.Get("json"), ",")[0]
		value := values.Field(i)
		if field != "file" {
			requestParams[field] = value.String()
		}
	}

	request, err := newfileUploadRequest(requestUrl.String(), requestParams, "file", params.File)
	if err != nil {
		return
	}
	request.SetBasicAuth(service.client.AuthId, service.client.AuthToken)
	response = &UpdateComplianceDocumentResponse{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func (service *ComplianceDocumentService) Delete(complianceDocumentId string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "ComplianceDocument/%s", complianceDocumentId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil)
	return
}
