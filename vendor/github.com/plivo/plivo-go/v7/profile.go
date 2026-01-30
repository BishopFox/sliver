package plivo

type ProfileService struct {
	client *Client
}

type CreateProfileRequestParams struct {
	ProfileAlias      string             `json:"profile_alias" validate:"required"`
	CustomerType      string             `json:"customer_type" validate:"oneof= DIRECT RESELLER"`
	EntityType        string             `json:"entity_type" validate:"oneof= PRIVATE PUBLIC NON_PROFIT GOVERNMENT INDIVIDUAL"`
	CompanyName       string             `json:"company_name" validate:"required,max=100"`
	Ein               string             `json:"ein" validate:"max=100"`
	EinIssuingCountry string             `json:"ein_issuing_country" validate:"max=2"`
	Address           *Address           `json:"address" validate:"required"`
	StockSymbol       string             `json:"stock_symbol" validate:"required_if=EntityType PUBLIC,max=10"`
	StockExchange     string             `json:"stock_exchange" validate:"required_if=EntityType PUBLIC,oneof= NASDAQ NYSE AMEX AMX ASX B3 BME BSE FRA ICEX JPX JSE KRX LON NSE OMX SEHK SGX SSE STO SWX SZSE TSX TWSE VSE OTHER ''"`
	Website           string             `json:"website" validate:"max=100"`
	Vertical          string             `json:"vertical" validate:"oneof= PROFESSIONAL REAL_ESTATE HEALTHCARE HUMAN_RESOURCES ENERGY ENTERTAINMENT RETAIL TRANSPORTATION AGRICULTURE INSURANCE POSTAL EDUCATION HOSPITALITY FINANCIAL POLITICAL GAMBLING LEGAL CONSTRUCTION NGO MANUFACTURING GOVERNMENT TECHNOLOGY COMMUNICATION"`
	AltBusinessID     string             `json:"alt_business_id" validate:"max=50"`
	AltBusinessidType string             `json:"alt_business_id_type" validate:"oneof= DUNS LEI GIIN NONE ''"`
	PlivoSubaccount   string             `json:"plivo_subaccount" validate:"max=20"`
	AuthorizedContact *AuthorizedContact `json:"authorized_contact"`
}

type CreateProfileResponse struct {
	ApiID       string `json:"api_id"`
	ProfileUUID string `json:"profile_uuid,omitempty"`
	Message     string `json:"message,omitempty"`
}

type ProfileGetResponse struct {
	ApiID   string  `json:"api_id"`
	Profile Profile `json:"profile"`
}

type ProfileListParams struct {
	Limit  int `url:"limit,omitempty"`
	Offset int `url:"offset,omitempty"`
}

type ProfileListResponse struct {
	ApiID string `json:"api_id"`
	Meta  struct {
		Previous *string
		Next     *string
		Offset   int64
		Limit    int64
	} `json:"meta"`
	ProfileResponse []Profile `json:"profiles"`
}

type DeleteProfileResponse struct {
	ApiID   string `json:"api_id"`
	Message string `json:"message,omitempty"`
}

type UpdateProfileRequestParams struct {
	EntityType        string             `json:"entity_type" validate:"oneof= PRIVATE PUBLIC NON_PROFIT GOVERNMENT INDIVIDUAL"`
	CompanyName       string             `json:"company_name" validate:"required,max=100"`
	Address           *Address           `json:"address" validate:"required"`
	Website           string             `json:"website" validate:"max=100"`
	Vertical          string             `json:"vertical" validate:"oneof= PROFESSIONAL REAL_ESTATE HEALTHCARE HUMAN_RESOURCES ENERGY ENTERTAINMENT RETAIL TRANSPORTATION AGRICULTURE INSURANCE POSTAL EDUCATION HOSPITALITY FINANCIAL POLITICAL GAMBLING LEGAL CONSTRUCTION NGO MANUFACTURING GOVERNMENT TECHNOLOGY COMMUNICATION"`
	AuthorizedContact *AuthorizedContact `json:"authorized_contact"`
}

type Profile struct {
	ProfileUUID       string            `json:"profile_uuid,omitempty"`
	ProfileAlias      string            `json:"profile_alias,omitempty"`
	ProfileType       string            `json:"profile_type,omitempty"`
	PrimaryProfile    string            `json:"primary_profile,omitempty"`
	CustomerType      string            `json:"customer_type,omitempty"`
	EntityType        string            `json:"entity_type,omitempty"`
	CompanyName       string            `json:"company_name,omitempty"`
	Ein               string            `json:"ein,omitempty"`
	EinIssuingCountry string            `json:"ein_issuing_country,omitempty"`
	Address           Address           `json:"address,omitempty"`
	StockSymbol       string            `json:"stock_symbol,omitempty"`
	StockExchange     string            `json:"stock_exchange,omitempty"`
	Website           string            `json:"website,omitempty"`
	Vertical          string            `json:"vertical,omitempty"`
	AltBusinessID     string            `json:"alt_business_id,omitempty"`
	AltBusinessidType string            `json:"alt_business_id_type,omitempty"`
	PlivoSubaccount   string            `json:"plivo_subaccount,omitempty"`
	AuthorizedContact AuthorizedContact `json:"authorized_contact,omitempty"`
	CreatedAt         string            `json:"created_at,omitempty"`
}

func (service *ProfileService) List(param ProfileListParams) (response *ProfileListResponse, err error) {
	req, err := service.client.NewRequest("GET", param, "Profile")
	if err != nil {
		return
	}
	response = &ProfileListResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *ProfileService) Get(ProfileUUID string) (response *ProfileGetResponse, err error) {
	req, err := service.client.NewRequest("GET", nil, "Profile/%s", ProfileUUID)
	if err != nil {
		return
	}
	response = &ProfileGetResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *ProfileService) Create(params CreateProfileRequestParams) (response *CreateProfileResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "Profile")
	if err != nil {
		return
	}
	response = &CreateProfileResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *ProfileService) Delete(profileUUID string) (response *DeleteProfileResponse, err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Profile/%s", profileUUID)
	if err != nil {
		return
	}
	response = &DeleteProfileResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *ProfileService) Update(profileUUID string, params UpdateProfileRequestParams) (response *ProfileGetResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "Profile/%s", profileUUID)
	if err != nil {
		return
	}
	response = &ProfileGetResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}
