package plivo

type BrandService struct {
	client *Client
}

type BrandCreationParams struct {
	BrandAlias       string  `json:"brand_alias" url:"brand_alias" validate:"required"`
	Type             string  `json:"brand_type" url:"brand_type" validate:"oneof= STARTER STANDARD ''"`
	ProfileUUID      string  `json:"profile_uuid" url:"profile_uuid" validate:"required,max=36"`
	SecondaryVetting *string `json:"secondary_vetting,omitempty" url:"secondary_vetting,omitempty"`
	URL              string  `json:"url,omitempty" url:"url,omitempty"`
	Method           string  `json:"method,omitempty" url:"method,omitempty"`
}
type BrandCreationResponse struct {
	ApiID   string `json:"api_id,omitempty"`
	BrandID string `json:"brand_id,omitempty"`
	Message string `json:"message,omitempty"`
}

type BrandUsecaseResponse struct {
	ApiID    string    `json:"api_id,omitempty"`
	Usecases []Usecase `json:"use_cases"`
	BrandID  string    `json:"brand_id"`
}

type Usecase struct {
	Name    string `json:"name"`
	Code    string `json:"code"`
	Details string `json:"details"`
}

type BrandListResponse struct {
	ApiID string `json:"api_id,omitempty"`
	Meta  struct {
		Previous   *string
		Next       *string
		Offset     int64
		Limit      int64
		TotalCount int64 `json:"total_count"`
	} `json:"meta"`
	BrandResponse []Brand `json:"brands,omitempty"`
}

type BrandGetResponse struct {
	ApiID string `json:"api_id,omitempty"`
	Brand Brand  `json:"brand,omitempty"`
}
type Brand struct {
	BrandAlias         string            `json:"brand_alias,omitempty"`
	EntityType         string            `json:"entity_type,omitempty"`
	BrandID            string            `json:"brand_id,omitempty"`
	ProfileUUID        string            `json:"profile_uuid,omitempty"`
	FirstName          string            `json:"first_name,omitempty"`
	LastName           string            `json:"last_name,omitempty"`
	Name               string            `json:"name,omitempty"`
	CompanyName        string            `json:"company_name,omitempty"`
	BrandType          string            `json:"brand_type,omitempty"`
	Ein                string            `json:"ein,omitempty"`
	EinIssuingCountry  string            `json:"ein_issuing_country,omitempty"`
	StockSymbol        string            `json:"stock_symbol,omitempty"`
	StockExchange      string            `json:"stock_exchange,omitempty"`
	Website            string            `json:"website,omitempty"`
	Vertical           string            `json:"vertical,omitempty"`
	AltBusinessID      string            `json:"alt_business_id,omitempty"`
	AltBusinessidType  string            `json:"alt_business_id_type,omitempty"`
	RegistrationStatus string            `json:"registration_status,omitempty"`
	VettingStatus      string            `json:"vetting_status,omitempty"`
	VettingScore       int64             `json:"vetting_score,omitempty"`
	Address            Address           `json:"address,omitempty"`
	AuthorizedContact  AuthorizedContact `json:"authorized_contact,omitempty"`
	DeclinedReasons    []TCRErrorDetail  `json:"declined_reasons,omitempty"`
	CreatedAt          string            `json:"created_at,omitempty"`
}

type BrandDeleteResponse struct {
	ApiID   string `json:"api_id,omitempty"`
	BrandID string `json:"brand_id,omitempty"`
	Message string `json:"message,omitempty"`
}

type BrandListParams struct {
	Type   *string `json:"type,omitempty"`
	Status *string `json:"status,omitempty"`
	Limit  int     `url:"limit,omitempty"`
	Offset int     `url:"offset,omitempty"`
}

type Address struct {
	Street     string `json:"street" validate:"max=100"`
	City       string `json:"city" validate:"max=100"`
	State      string `json:"state" validate:"max=20"`
	PostalCode string `json:"postal_code" validate:"max=10"`
	Country    string `json:"country" validate:"max=2"`
}

type AuthorizedContact struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Phone     string `json:"phone,omitempty" validate:"max=16"`
	Email     string `json:"email,omitempty" validate:"max=100"`
	Title     string `json:"title,omitempty"`
	Seniority string `json:"seniority,omitempty"`
}

type TCRErrorDetail struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func (service *BrandService) List(params BrandListParams) (response *BrandListResponse, err error) {
	req, err := service.client.NewRequest("GET", params, "10dlc/Brand")
	if err != nil {
		return
	}
	response = &BrandListResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *BrandService) Get(brandID string) (response *BrandGetResponse, err error) {
	req, err := service.client.NewRequest("GET", nil, "10dlc/Brand/%s", brandID)
	if err != nil {
		return
	}
	response = &BrandGetResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *BrandService) Delete(brandID string) (response *BrandDeleteResponse, err error) {
	req, err := service.client.NewRequest("DELETE", nil, "10dlc/Brand/%s", brandID)
	if err != nil {
		return
	}
	response = &BrandDeleteResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *BrandService) Create(params BrandCreationParams) (response *BrandCreationResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "10dlc/Brand")
	if err != nil {
		return
	}
	response = &BrandCreationResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *BrandService) Usecases(brandID string) (response *BrandUsecaseResponse, err error) {
	req, err := service.client.NewRequest("GET", nil, "10dlc/Brand/%s/usecases", brandID)
	if err != nil {
		return
	}
	response = &BrandUsecaseResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}
