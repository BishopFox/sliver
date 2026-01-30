package plivo

import "time"

type ComplianceApplicationService struct {
	client *Client
}

type SubmitComplianceApplicationResponse struct {
	APIID                   string    `json:"api_id"`
	CreatedAt               time.Time `json:"created_at"`
	ComplianceApplicationID string    `json:"compliance_application_id"`
	Alias                   string    `json:"alias"`
	Status                  string    `json:"status"`
	EndUserType             string    `json:"end_user_type"`
	CountryIso2             string    `json:"country_iso2"`
	NumberType              string    `json:"number_type"`
	EndUserID               string    `json:"end_user_id"`
	ComplianceRequirementID string    `json:"compliance_requirement_id"`
	Documents               []struct {
		DocumentID       string `json:"document_id"`
		Name             string `json:"name"`
		DocumentName     string `json:"document_name,omitempty"`
		DocumentTypeID   string `json:"document_type_id,omitempty"`
		DocumentTypeName string `json:"document_type_name,omitempty"`
		Scope            string `json:"scope,omitempty"`
	} `json:"documents"`
}

type UpdateComplianceApplicationParams struct {
	ComplianceApplicationId string   `json:"compliance_application_id"`
	DocumentIds             []string `json:"document_ids"`
}

type UpdateComplianceApplicationResponse BaseResponse

type CreateComplianceApplicationParams struct {
	ComplianceRequirementId string   `json:"compliance_requirement_id,omitempty" url:"compliance_requirement_id,omitempty"`
	EndUserId               string   `json:"end_user_id,omitempty" url:"end_user_id,omitempty"`
	DocumentIds             []string `json:"document_ids,omitempty" url:"document_ids, omitempty"`
	Alias                   string   `json:"alias,omitempty" url:"alias, omitempty"`
	EndUserType             string   `json:"end_user_type,omitempty" url:"end_user_type, omitempty"`
	CountryIso2             string   `json:"country_iso2,omitempty" url:"country_iso2, omitempty"`
	NumberType              string   `json:"number_type,omitempty" url:"number_type, omitempty"`
}

type ComplianceApplicationResponse struct {
	APIID                   string    `json:"api_id"`
	CreatedAt               time.Time `json:"created_at"`
	ComplianceApplicationID string    `json:"compliance_application_id"`
	Alias                   string    `json:"alias"`
	Status                  string    `json:"status"`
	EndUserType             string    `json:"end_user_type"`
	CountryIso2             string    `json:"country_iso2"`
	NumberType              string    `json:"number_type"`
	EndUserID               string    `json:"end_user_id"`
	ComplianceRequirementID string    `json:"compliance_requirement_id"`
	Documents               []struct {
		DocumentID       string `json:"document_id"`
		Name             string `json:"name"`
		DocumentName     string `json:"document_name,omitempty"`
		DocumentTypeID   string `json:"document_type_id,omitempty"`
		DocumentTypeName string `json:"document_type_name,omitempty"`
		Scope            string `json:"scope,omitempty"`
	} `json:"documents"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}

type ComplianceApplicationListParams struct {
	Limit       int    `json:"limit,omitempty" url:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty" url:"offset,omitempty"`
	EndUserID   string `json:"end_user_id,omitempty" url:"end_user_id,omitempty"`
	Country     string `json:"country,omitempty" url:"country,omitempty"`
	NumberType  string `json:"number_type,omitempty" url:"number_type,omitempty"`
	EndUserType string `json:"end_user_type,omitempty" url:"end_user_type,omitempty"`
	Alias       string `json:"alias,omitempty" url:"alias,omitempty"`
	Status      string `json:"status,omitempty" url:"status,omitempty"`
}

type ListComplianceApplicationResponse struct {
	APIID string `json:"api_id"`
	Meta  struct {
		Limit      int         `json:"limit"`
		Next       string      `json:"next"`
		Offset     int         `json:"offset"`
		Previous   interface{} `json:"previous"`
		TotalCount int         `json:"total_count"`
	} `json:"meta"`
	Objects []struct {
		CreatedAt               time.Time `json:"created_at"`
		ComplianceApplicationID string    `json:"compliance_application_id"`
		Alias                   string    `json:"alias"`
		Status                  string    `json:"status"`
		EndUserType             string    `json:"end_user_type"`
		CountryIso2             string    `json:"country_iso2"`
		NumberType              string    `json:"number_type"`
		EndUserID               string    `json:"end_user_id"`
		ComplianceRequirementID string    `json:"compliance_requirement_id"`
		Documents               []struct {
			DocumentID       string `json:"document_id"`
			Name             string `json:"name"`
			DocumentName     string `json:"document_name"`
			DocumentTypeID   string `json:"document_type_id"`
			DocumentTypeName string `json:"document_type_name"`
			Scope            string `json:"scope"`
		} `json:"documents"`
		RejectionReason string `json:"rejection_reason,omitempty"`
	} `json:"objects"`
}

func (service *ComplianceApplicationService) Get(complianceApplicationId string) (response *ComplianceApplicationResponse, err error) {
	req, err := service.client.NewRequest("GET", nil, "ComplianceApplication/%s", complianceApplicationId)
	if err != nil {
		return
	}
	response = &ComplianceApplicationResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *ComplianceApplicationService) List(params ComplianceApplicationListParams) (response *ListComplianceApplicationResponse, err error) {
	request, err := service.client.NewRequest("GET", params, "ComplianceApplication")
	if err != nil {
		return
	}
	response = &ListComplianceApplicationResponse{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func (service *ComplianceApplicationService) Create(params CreateComplianceApplicationParams) (response *ComplianceApplicationResponse, err error) {
	request, err := service.client.NewRequest("POST", params, "ComplianceApplication")
	if err != nil {
		return
	}
	response = &ComplianceApplicationResponse{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func (service *ComplianceApplicationService) Update(params UpdateComplianceApplicationParams) (response *UpdateComplianceApplicationResponse, err error) {
	request, err := service.client.NewRequest("POST", params, "ComplianceApplication/%s", params.ComplianceApplicationId)
	if err != nil {
		return
	}
	response = &UpdateComplianceApplicationResponse{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func (service *ComplianceApplicationService) Delete(complianceApplicationId string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "ComplianceApplication/%s", complianceApplicationId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil)
	return
}

func (service *ComplianceApplicationService) Submit(complianceApplicationId string) (response *SubmitComplianceApplicationResponse, err error) {
	req, err := service.client.NewRequest("POST", nil, "ComplianceApplication/%s/Submit", complianceApplicationId)
	if err != nil {
		return
	}
	response = &SubmitComplianceApplicationResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}
