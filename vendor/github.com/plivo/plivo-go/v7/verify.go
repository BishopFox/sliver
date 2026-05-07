package plivo

import (
	"time"
)

type VerifyService struct {
	client *Client
	Session
}

type SessionList struct {
	APIID string `json:"api_id"`
	Meta  struct {
		Limit    int     `json:"limit"`
		Offset   int     `json:"offset"`
		Next     *string `json:"next"`
		Previous *string `json:"previous"`
	} `json:"meta"`
	Sessions []Session `json:"sessions"`
}

type Session struct {
	APIID              string           `json:"api_id,omitempty"`
	SessionUUID        string           `json:"session_uuid,omitempty"`
	AppUUID            string           `json:"app_uuid,omitempty"`
	Alias              string           `json:"alias,omitempty"`
	Recipient          string           `json:"recipient,omitempty"`
	Channel            string           `json:"channel,omitempty"`
	Locale             string           `json:"locale,omitempty"`
	Status             string           `json:"status,omitempty"`
	Count              int              `json:"count,omitempty"`
	RequesterIP        string           `json:"requestor_ip,omitempty"`
	CountryISO         string           `json:"destination_country_iso2,omitempty"`
	DestinationNetwork *string          `json:"destination_network,omitempty"`
	AttemptDetails     []AttemptDetails `json:"attempt_details,omitempty"`
	Charges            Charges          `json:"charges,omitempty"`
	CreatedAt          time.Time        `json:"created_at,omitempty"`
	UpdatedAt          time.Time        `json:"updated_at,omitempty"`
}

type AttemptDetails struct {
	Channel     string    `json:"channel,omitempty"`
	AttemptUUID string    `json:"attempt_uuid,omitempty"`
	Status      string    `json:"status,omitempty"`
	BrandName   string    `json:"brand_name,omitempty"`
	AppHash     string    `json:"app_hash,omitempty"`
	CodeLength  int       `json:"code_length,omitempty"`
	Time        time.Time `json:"time,omitempty"`
	Dtmf        *int      `json:"dtmf,omitempty"`
	FraudCheck  string    `json:"fraud_check,omitempty"`
}

type Charges struct {
	TotalCharge      string           `json:"total_charge,omitempty"`
	ValidationCharge string           `json:"validation_charge,omitempty"`
	AttemptCharges   []AttemptCharges `json:"attempt_charges,omitempty"`
}

type AttemptCharges struct {
	AttemptUUID string `json:"attempt_uuid,omitempty"`
	Channel     string `json:"channel,omitempty"`
	Charge      string `json:"charge,omitempty"`
}

type SessionCreateParams struct {
	Recipient string `json:"recipient,omitempty"`
	// Optional parameters.
	AppUUID    string `json:"app_uuid,omitempty"`
	Channel    string `json:"channel,omitempty"`
	URL        string `json:"url,omitempty"`
	Method     string `json:"method,omitempty"`
	Src        string `json:"src,omitempty"`
	Locale     string `json:"locale,omitempty"`
	BrandName  string `json:"brand_name,omitempty"`
	AppHash    string `json:"app_hash,omitempty"`
	CodeLength int    `json:"code_length,omitempty"`
	Dtmf       *int   `json:"dtmf,omitempty"`
	FraudCheck string `json:"fraud_check,omitempty"`
}

type SessionCreateResponseBody struct {
	APIID       string `json:"api_id,omitempty"`
	Error       string `json:"error,omitempty"`
	Message     string `json:"message,omitempty"`
	SessionUUID string `json:"session_uuid,omitempty"`
}

type SessionValidationParams struct {
	OTP string `json:"otp,omitempty"`
}

type SessionListParams struct {
	Limit                     int    `url:"limit,omitempty"`
	Offset                    int    `url:"offset,omitempty"`
	Status                    string `url:"status,omitempty"`
	Recipient                 string `url:"recipient,omitempty"`
	AppUUID                   string `url:"app_uuid,omitempty"`
	Country                   string `url:"country,omitempty"`
	Alias                     string `url:"alias,omitempty"`
	BrandName                 string `json:"brand_name,omitempty"`
	AppHash                   string `json:"app_hash,omitempty"`
	SessionTime               string `url:"session_time,omitempty"`
	Subaccount                string `url:"subaccount,omitempty"`
	SessionTimeGreaterThan    string `url:"session_time__gt,omitempty"`
	SessionTimeGreaterOrEqual string `url:"session_time__gte,omitempty"`
	SessionTimeLessThan       string `url:"session_time__lt,omitempty"`
	SessionTimeLessOrEqual    string `url:"session_time__lte,omitempty"`
}

func (service *VerifyService) Create(params SessionCreateParams) (response *SessionCreateResponseBody, err error) {
	req, err := service.client.NewRequest("POST", params, "Verify/Session")
	if err != nil {
		return
	}
	response = &SessionCreateResponseBody{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *VerifyService) Get(sessionUUID string) (response *Session, err error) {
	req, err := service.client.NewRequest("GET", nil, "Verify/Session/%s", sessionUUID)
	if err != nil {
		return
	}
	response = &Session{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *VerifyService) List(params SessionListParams) (response *SessionList, err error) {
	req, err := service.client.NewRequest("GET", params, "Verify/Session")
	if err != nil {
		return
	}
	response = &SessionList{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *VerifyService) Validate(params SessionValidationParams, sessionUUID string) (response *SessionCreateResponseBody, err error) {
	req, err := service.client.NewRequest("POST", params, "Verify/Session/%s", sessionUUID)
	if err != nil {
		return
	}
	response = &SessionCreateResponseBody{}
	err = service.client.ExecuteRequest(req, response)
	return
}
