package plivo

import (
	"time"
)

type VerifyCallerIdService struct {
	client *Client
}

type InitiateVerify struct {
	PhoneNumber string `json:"phone_number"`
	Alias       string `json:"alias"`
	Channel     string `json:"channel"`
	Country     string `json:"country"`
	SubAccount  string `json:"subaccount"`
	AccountID   int64  `json:"account_id"`
	AuthID      string `json:"auth_id"`
	AuthToken   string `json:"auth_token"`
}

type InitiateVerifyResponse struct {
	ApiID            string `json:"api_id,omitempty" url:"api_id,omitempty"`
	Message          string `json:"message,omitempty" url:"message,omitempty"`
	VerificationUUID string `json:"verification_uuid,omitempty" url:"verification_uuid,omitempty"`
}

type ListVerifiedCallerIdParams struct {
	Country    string `json:"country,omitempty" url:"country,omitempty"`
	SubAccount string `json:"subaccount,omitempty" url:"subaccount,omitempty"`
	Alias      string `json:"alias,omitempty" url:"alias,omitempty"`
	Limit      int64  `json:"limit,omitempty" url:"limit,omitempty"`
	Offset     int64  `json:"offset,omitempty" url:"offset,omitempty"`
}

type VerifyResponse struct {
	Alias            string    `json:"alias,omitempty"`
	ApiID            string    `json:"api_id,omitempty" url:"api_id,omitempty"`
	Channel          string    `json:"channel"`
	Country          string    `json:"country"`
	CreatedAt        time.Time `json:"created_at"`
	PhoneNumber      string    `json:"phone_number"`
	VerificationUUID string    `json:"verification_uuid"`
	SubAccount       string    `json:"subaccount,omitempty"`
}

type UpdateVerifiedCallerIDParams struct {
	Alias      string `json:"alias,omitempty"`
	SubAccount string `json:"subaccount,omitempty"`
}

type GetVerifyResponse struct {
	Alias            string    `json:"alias,omitempty"`
	ApiID            string    `json:"api_id,omitempty" url:"api_id,omitempty"`
	Country          string    `json:"country"`
	CreatedAt        time.Time `json:"created_at"`
	ModifiedAt       time.Time `json:"modified_at"`
	PhoneNumber      string    `json:"phone_number"`
	SubAccount       string    `json:"subaccount,omitempty" url:"subaccount,omitempty"`
	VerificationUUID string    `json:"verification_uuid"`
}

type ListVerifyResponse struct {
	Alias            string    `json:"alias,omitempty"`
	Country          string    `json:"country"`
	CreatedAt        time.Time `json:"created_at"`
	ModifiedAt       time.Time `json:"modified_at"`
	PhoneNumber      string    `json:"phone_number"`
	ResourceUri      string    `json:"resource_uri,omitempty"`
	SubAccount       string    `json:"subaccount,omitempty"`
	VerificationUUID string    `json:"verification_uuid"`
}

type ListVerifiedCallerIDResponse struct {
	ApiID   string               `json:"api_id,omitempty" url:"api_id,omitempty"`
	Meta    Meta                 `json:"meta" url:"meta"`
	Objects []ListVerifyResponse `json:"objects" url:"objects"`
}

func (service *VerifyCallerIdService) InitiateVerify(params InitiateVerify) (response *InitiateVerifyResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "VerifiedCallerId")
	if err != nil {
		return
	}
	response = &InitiateVerifyResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *VerifyCallerIdService) VerifyCallerID(verificationUuid string, otp string) (response *VerifyResponse, err error) {
	req, err := service.client.NewRequest("POST", map[string]string{"otp": otp}, "VerifiedCallerId/Verification/%s", verificationUuid)
	if err != nil {
		return
	}
	response = &VerifyResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *VerifyCallerIdService) DeleteVerifiedCallerID(phoneNumber string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "VerifiedCallerId/%s", phoneNumber)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())

	return
}

func (service *VerifyCallerIdService) UpdateVerifiedCallerID(phoneNumber string, params UpdateVerifiedCallerIDParams) (response *GetVerifyResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "VerifiedCallerId/%s", phoneNumber)
	if err != nil {
		return
	}
	response = &GetVerifyResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *VerifyCallerIdService) GetVerifiedCallerID(phoneNumber string) (response *GetVerifyResponse, err error) {
	req, err := service.client.NewRequest("GET", nil, "VerifiedCallerId/%s", phoneNumber)
	if err != nil {
		return
	}
	response = &GetVerifyResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *VerifyCallerIdService) ListVerifiedCallerID(params ListVerifiedCallerIdParams) (response *ListVerifiedCallerIDResponse, err error) {
	req, err := service.client.NewRequest("GET", params, "VerifiedCallerId")
	if err != nil {
		return
	}
	response = &ListVerifiedCallerIDResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}
