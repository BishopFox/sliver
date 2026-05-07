package plivo

type NumberService struct {
	client *Client
}

type Number struct {
	Alias                              string `json:"alias,omitempty" url:"alias,omitempty"`
	VoiceEnabled                       bool   `json:"voice_enabled,omitempty" url:"voice_enabled,omitempty"`
	SMSEnabled                         bool   `json:"sms_enabled,omitempty" url:"sms_enabled,omitempty"`
	MMSEnabled                         bool   `json:"mms_enabled,omitempty" url:"mms_enabled,omitempty"`
	Description                        string `json:"description,omitempty" url:"description,omitempty"`
	PlivoNumber                        bool   `json:"plivo_number,omitempty" url:"plivo_number,omitempty"`
	City                               string `json:"city,omitempty" url:"city,omitempty"`
	Country                            string `json:"country,omitempty" url:"country,omitempty"`
	Carrier                            string `json:"carrier,omitempty" url:"carrier,omitempty"`
	Number                             string `json:"number,omitempty" url:"number,omitempty"`
	NumberType                         string `json:"number_type,omitempty" url:"number_type,omitempty"`
	MonthlyRentalRate                  string `json:"monthly_rental_rate,omitempty" url:"monthly_rental_rate,omitempty"`
	Application                        string `json:"application,omitempty" url:"application,omitempty"`
	RenewalDate                        string `json:"renewal_date,omitempty" url:"renewal_date,omitempty"`
	CNAMLookup                         string `json:"cnam_lookup,omitempty" url:"cnam_lookup,omitempty"`
	AddedOn                            string `json:"added_on,omitempty" url:"added_on,omitempty"`
	ResourceURI                        string `json:"resource_uri,omitempty" url:"resource_uri,omitempty"`
	VoiceRate                          string `json:"voice_rate,omitempty" url:"voice_rate,omitempty"`
	SMSRate                            string `json:"sms_rate,omitempty" url:"sms_rate,omitempty"`
	MMSRate                            string `json:"mms_rate,omitempty" url:"mms_rate,omitempty"`
	TendlcCampaignID                   string `json:"tendlc_campaign_id,omitempty" url:"tendlc_campaign_id,omitempty"`
	TendlcRegistrationStatus           string `json:"tendlc_registration_status,omitempty" url:"tendlc_registration_status,omitempty"`
	TollFreeSMSVerification            string `json:"toll_free_sms_verification,omitempty" url:"toll_free_sms_verification,omitempty"`
	TollFreeSMSVerificationID          string `json:"toll_free_sms_verification_id,omitempty" url:"toll_free_sms_verification_id,omitempty"`
	TollFreeSMSVerificationOrderStatus string `json:"toll_free_sms_verification_order_status,omitempty" url:"toll_free_sms_verification_order_status,omitempty"`
}

type NumberCreateParams struct {
	Numbers    string `json:"numbers,omitempty" url:"numbers,omitempty"`
	Carrier    string `json:"carrier,omitempty" url:"carrier,omitempty"`
	Region     string `json:"region,omitempty" url:"region,omitempty"`
	NumberType string `json:"number_type,omitempty" url:"number_type,omitempty"`
	AppID      string `json:"app_id,omitempty" url:"app_id,omitempty"`
	Subaccount string `json:"subaccount,omitempty" url:"subaccount,omitempty"`
}

type NumberCreateResponse BaseResponse
type NumberUpdateResponse BaseResponse

type NumberUpdateParams struct {
	AppID      string `json:"app_id,omitempty" url:"app_id,omitempty"`
	Subaccount string `json:"subaccount,omitempty" url:"subaccount,omitempty"`
	Alias      string `json:"alias,omitempty" url:"alias,omitempty"`
	CNAMLookup string `json:"cnam_lookup,omitempty" url:"cnam_lookup,omitempty"`
}

type NumberListParams struct {
	NumberType                         string `json:"number_type,omitempty" url:"number_type,omitempty"`
	NumberStartsWith                   string `json:"number_startswith,omitempty" url:"number_startswith,omitempty"`
	Subaccount                         string `json:"subaccount,omitempty" url:"subaccount,omitempty"`
	RenewalDate                        string `json:"renewal_date,omitempty" url:"renewal_date,omitempty"`
	RenewalDateLt                      string `json:"renewal_date__lt,omitempty" url:"renewal_date__lt,omitempty"`
	RenewalDateLte                     string `json:"renewal_date__lte,omitempty" url:"renewal_date__lte,omitempty"`
	RenewalDateGt                      string `json:"renewal_date__gt,omitempty" url:"renewal_date__gt,omitempty"`
	RenewalDateGte                     string `json:"renewal_date__gte,omitempty" url:"renewal_date__gte,omitempty"`
	Services                           string `json:"services,omitempty" url:"services,omitempty"`
	Alias                              string `json:"alias,omitempty" url:"alias,omitempty"`
	Limit                              int64  `json:"limit,omitempty" url:"limit,omitempty"`
	Offset                             int64  `json:"offset,omitempty" url:"offset,omitempty"`
	TendlcCampaignID                   string `json:"tendlc_campaign_id,omitempty" url:"tendlc_campaign_id,omitempty"`
	TendlcRegistrationStatus           string `json:"tendlc_registration_status,omitempty" url:"tendlc_registration_status,omitempty"`
	TollFreeSMSVerification            string `json:"toll_free_sms_verification,omitempty" url:"toll_free_sms_verification,omitempty"`
	CNAMLookup                         string `json:"cnam_lookup,omitempty" url:"cnam_lookup,omitempty"`
	TollFreeSMSVerificationOrderStatus string `json:"toll_free_sms_verification_order_status,omitempty" url:"toll_free_sms_verification_order_status,omitempty"`
}

type NumberListResponse struct {
	ApiID   string    `json:"api_id" url:"api_id"`
	Meta    *Meta     `json:"meta" url:"meta"`
	Objects []*Number `json:"objects" url:"objects"`
}

func (service *NumberService) Create(params NumberCreateParams) (response *NumberCreateResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "Number")
	if err != nil {
		return
	}
	response = &NumberCreateResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *NumberService) Update(NumberId string, params NumberUpdateParams) (response *NumberUpdateResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "Number/%s", NumberId)
	if err != nil {
		return
	}
	response = &NumberUpdateResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *NumberService) List(params NumberListParams) (response *NumberListResponse, err error) {
	req, err := service.client.NewRequest("GET", params, "Number")
	if err != nil {
		return
	}
	response = &NumberListResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *NumberService) Get(NumberId string) (response *Number, err error) {
	req, err := service.client.NewRequest("GET", nil, "Number/%s", NumberId)
	if err != nil {
		return
	}
	response = &Number{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *NumberService) Delete(NumberId string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Number/%s", NumberId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil)
	return
}

type PhoneNumber struct {
	Country           string `json:"country" url:"country"`
	City              string `json:"city" url:"city"`
	Lata              int    `json:"lata" url:"lata"`
	MonthlyRentalRate string `json:"monthly_rental_rate" url:"monthly_rental_rate"`
	Number            string `json:"number" url:"number"`
	Type              string `json:"type" url:"type"`
	Prefix            string `json:"prefix" url:"prefix"`
	RateCenter        string `json:"rate_center" url:"rate_center"`
	Region            string `json:"region" url:"region"`
	ResourceURI       string `json:"resource_uri" url:"resource_uri"`
	Restriction       string `json:"restriction" url:"restriction"`
	RestrictionText   string `json:"restriction_text" url:"restriction_text"`
	SetupRate         string `json:"setup_rate" url:"setup_rate"`
	SmsEnabled        bool   `json:"sms_enabled" url:"sms_enabled"`
	SmsRate           string `json:"sms_rate" url:"sms_rate"`
	MmsEnabled        bool   `json:"mms_enabled" url:"mms_enabled"`
	MmsRate           string `json:"mms_rate" url:"mms_rate"`
	VoiceEnabled      bool   `json:"voice_enabled" url:"voice_enabled"`
	VoiceRate         string `json:"voice_rate" url:"voice_rate"`
}

type PhoneNumberListParams struct {
	CountryISO string `json:"country_iso,omitempty" url:"country_iso,omitempty"`
	Type       string `json:"type,omitempty" url:"type,omitempty"`
	Pattern    string `json:"pattern,omitempty" url:"pattern,omitempty"`
	Region     string `json:"region,omitempty" url:"region,omitempty"`
	Services   string `json:"services,omitempty" url:"services,omitempty"`
	LATA       string `json:"lata,omitempty" url:"lata,omitempty"`
	RateCenter string `json:"rate_center,omitempty" url:"rate_center,omitempty"`
	City       string `json:"city,omitempty" url:"city,omitempty"`
	Limit      int    `json:"limit,omitempty" url:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty" url:"offset,omitempty"`
}

type PhoneNumberCreateParams struct {
	AppID      string `json:"app_id,omitempty" url:"app_id,omitempty"`
	CNAMLookup string `json:"cnam_lookup,omitempty" url:"cnam_lookup,omitempty"`
}

type PhoneNumberService struct {
	client *Client
}

type PhoneNumberCreateResponse struct {
	APIID   string `json:"api_id" url:"api_id"`
	Message string `json:"message" url:"message"`
	Numbers []struct {
		Number string `json:"number" url:"number"`
		Status string `json:"status" url:"status"`
	} `json:"numbers" url:"numbers"`
	Status string `json:"status" url:"status"`
}

type PhoneNumberListResponse struct {
	ApiID   string         `json:"api_id" url:"api_id"`
	Meta    *Meta          `json:"meta" url:"meta"`
	Objects []*PhoneNumber `json:"objects" url:"objects"`
}

func (service *PhoneNumberService) Create(number string, params PhoneNumberCreateParams) (response *PhoneNumberCreateResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "PhoneNumber/%s", number)
	if err != nil {
		return
	}
	response = &PhoneNumberCreateResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *PhoneNumberService) List(params PhoneNumberListParams) (response *PhoneNumberListResponse, err error) {
	req, err := service.client.NewRequest("GET", params, "PhoneNumber")
	if err != nil {
		return
	}
	response = &PhoneNumberListResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}
