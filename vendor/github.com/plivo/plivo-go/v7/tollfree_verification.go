package plivo

import "time"

// TollfreeVerificationService - TF verification service struct
type TollfreeVerificationService struct {
	client *Client
}

// TollfreeVerificationResponse - Default response
type TollfreeVerificationResponse struct {
	APIID   string `json:"api_id" url:"api_id"`
	Message string `json:"message,omitempty" url:"message,omitempty"`
}

// TollfreeVerificationCreateResponse - Default response
type TollfreeVerificationCreateResponse struct {
	APIID   string `json:"api_id" url:"api_id"`
	Message string `json:"message,omitempty" url:"message,omitempty"`
	UUID    string `json:"uuid" url:"uuid"`
}

// TollfreeVerificationCreateParams - List of params to create a TF verification request
type TollfreeVerificationCreateParams struct {
	ProfileUUID           string `json:"profile_uuid,omitempty" url:"profile_uuid,omitempty"`
	Usecase               string `json:"usecase,omitempty" url:"usecase,omitempty"`
	UsecaseSummary        string `json:"usecase_summary,omitempty" url:"usecase_summary,omitempty"`
	MessageSample         string `json:"message_sample,omitempty" url:"message_sample,omitempty"`
	OptInImageURL         string `json:"optin_image_url,omitempty" url:"optin_image_url,omitempty"`
	OptInType             string `json:"optin_type,omitempty" url:"optin_type,omitempty"`
	Volume                string `json:"volume,omitempty" url:"volume,omitempty"`
	AdditionalInformation string `json:"additional_information,omitempty" url:"additional_information,omitempty"`
	ExtraData             string `json:"extra_data,omitempty" url:"extra_data,omitempty"`
	Number                string `json:"number,omitempty" url:"number,omitempty"`
	CallbackURL           string `json:"callback_url,omitempty" url:"callback_url,omitempty"`
	CallbackMethod        string `json:"callback_method,omitempty" url:"callback_method,omitempty"`
}

// TollfreeVerificationUpdateParams - List of update params to update in TF verification request
type TollfreeVerificationUpdateParams struct {
	ProfileUUID           string `json:"profile_uuid,omitempty" url:"profile_uuid,omitempty"`
	Usecase               string `json:"usecase,omitempty" url:"usecase,omitempty"`
	UsecaseSummary        string `json:"usecase_summary,omitempty" url:"usecase_summary,omitempty"`
	MessageSample         string `json:"message_sample,omitempty" url:"message_sample,omitempty"`
	OptInImageURL         string `json:"optin_image_url,omitempty" url:"optin_image_url,omitempty"`
	OptInType             string `json:"optin_type,omitempty" url:"optin_type,omitempty"`
	Volume                string `json:"volume,omitempty" url:"volume,omitempty"`
	AdditionalInformation string `json:"additional_information,omitempty" url:"additional_information,omitempty"`
	ExtraData             string `json:"extra_data,omitempty" url:"extra_data,omitempty"`
	CallbackURL           string `json:"callback_url,omitempty" url:"callback_url,omitempty"`
	CallbackMethod        string `json:"callback_method,omitempty" url:"callback_method,omitempty"`
}

// TollfreeVerificationListParams - List of params to search in list API
type TollfreeVerificationListParams struct {
	Number      string `json:"number,omitempty"  url:"number,omitempty"`
	Status      string `json:"status,omitempty"  url:"status,omitempty"`
	ProfileUUID string `json:"profile_uuid,omitempty" url:"profile_uuid,omitempty"`
	CreatedGT   string `json:"created__gt,omitempty" url:"created__gt,omitempty"`
	CreatedGTE  string `json:"created__gte,omitempty" url:"created__gte,omitempty"`
	CreatedLT   string `json:"created__lt,omitempty" url:"created__lt,omitempty"`
	CreatedLTE  string `json:"created__lte,omitempty" url:"created__lte,omitempty"`
	Usecase     string `json:"usecase,omitempty" url:"usecase,omitempty"`
	Limit       int64  `json:"limit,omitempty" url:"limit,omitempty"`
	Offset      int64  `json:"offset,omitempty" url:"offset,omitempty"`
}

// TollfreeVerification struct
type TollfreeVerification struct {
	UUID                  string    `json:"uuid"  url:"uuid"`
	ProfileUUID           string    `json:"profile_uuid" url:"profile_uuid"`
	Number                string    `json:"number" url:"number"`
	Usecase               string    `json:"usecase" url:"usecase"`
	UsecaseSummary        string    `json:"usecase_summary" url:"usecase_summary"`
	MessageSample         string    `json:"message_sample" url:"message_sample"`
	OptinImageURL         *string   `json:"optin_image_url" url:"optin_image_url"`
	OptinType             string    `json:"optin_type" url:"optin_type"`
	Volume                string    `json:"volume" url:"volume"`
	AdditionalInformation string    `json:"additional_information" url:"additional_information"`
	ExtraData             string    `json:"extra_data" url:"extra_data"`
	CallbackURL           string    `json:"callback_url" url:"callback_url"`
	CallbackMethod        string    `json:"callback_method" url:"callback_method"`
	Status                string    `json:"status" url:"status"`
	ErrorMessage          string    `json:"error_message" url:"error_message"`
	Created               time.Time `json:"created" url:"created"`
	LastModified          time.Time `json:"last_modified" url:"last_modified"`
}

// TollfreeVerificationListResponse - list API response struct
type TollfreeVerificationListResponse struct {
	APIID   string                 `json:"api_id" url:"api_id"`
	Meta    *Meta                  `json:"meta,omitempty" url:"meta,omitempty"`
	Objects []TollfreeVerification `json:"objects,omitempty" url:"objects,omitempty"`
}

// Create - create API for Tollfree Verification Request
func (service *TollfreeVerificationService) Create(params TollfreeVerificationCreateParams) (response *TollfreeVerificationCreateResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "TollfreeVerification")
	if err != nil {
		return
	}
	response = &TollfreeVerificationCreateResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

// Update - Update API for Tollfree Verification Request
func (service *TollfreeVerificationService) Update(UUID string, params TollfreeVerificationUpdateParams) (response *TollfreeVerificationResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "TollfreeVerification/%s", UUID)
	if err != nil {
		return
	}
	response = &TollfreeVerificationResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

// Get - Get API for Tollfree Verification Request
func (service *TollfreeVerificationService) Get(UUID string) (response *TollfreeVerification, err error) {
	req, err := service.client.NewRequest("GET", nil, "TollfreeVerification/%s", UUID)
	if err != nil {
		return
	}
	response = &TollfreeVerification{}
	err = service.client.ExecuteRequest(req, response)
	return
}

// List - List API for Tollfree Verification Request
func (service *TollfreeVerificationService) List(params TollfreeVerificationListParams) (response *TollfreeVerificationListResponse, err error) {
	req, err := service.client.NewRequest("GET", params, "TollfreeVerification")
	if err != nil {
		return
	}
	response = &TollfreeVerificationListResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

// Delete - Delete API for Tollfree Verification Request
func (service *TollfreeVerificationService) Delete(UUID string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "TollfreeVerification/%s", UUID)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil)
	return
}
