package plivo

type AccountService struct {
	client *Client
}

type SubaccountService struct {
	client *Client
}

type Plan struct {
	VoiceRate           string `json:"voice_rate,omitempty" url:"voice_rate,omitempty"`
	MessagingRate       string `json:"messaging_rate,omitempty" url:"messaging_rate,omitempty"`
	Name                string `json:"name_rate,omitempty" url:"name_rate,omitempty"`
	MonthlyCloudCredits string `json:"monthly_cloud_credits,omitempty" url:"monthly_cloud_credits,omitempty"`
}

type Account struct {
	AccountType  string `json:"account_type,omitempty" url:"account_type,omitempty"` // The type of your Plivo account. All accounts with funds are standard accounts. If your account is on free trial, this attribute will return developer.
	Address      string `json:"address,omitempty" url:"address,omitempty"`           // The postal address of the account which will be reflected in the invoices.
	ApiID        string `json:"api_id,omitempty" url:"api_id,omitempty"`
	AuthID       string `json:"auth_id,omitempty" url:"auth_id,omitempty"`             // The auth id of the account.
	AutoRecharge bool   `json:"auto_recharge,omitempty" url:"auto_recharge,omitempty"` // Auto recharge settings associated with the account. If this value is true, we will recharge your account if the credits fall below a certain threshold.
	BillingMode  string `json:"billing_mode,omitempty" url:"billing_mode,omitempty"`   // The billing mode of the account. Can be prepaid or postpaid.
	CashCredits  string `json:"cash_credits,omitempty" url:"cash_credits,omitempty"`   // Credits of the account.
	City         string `json:"city,omitempty" url:"city,omitempty"`                   // The city of the account.
	Name         string `json:"name,omitempty" url:"name,omitempty"`                   // The name of the account holder.
	ResourceURI  string `json:"resource_uri,omitempty" url:"resource_uri,omitempty"`
	State        string `json:"state,omitempty" url:"state,omitempty"`       // The state of the account holder.
	Timezone     string `json:"timezone,omitempty" url:"timezone,omitempty"` // The timezone of the account.
}

type AccountUpdateParams struct {
	Name    string `json:"name,omitempty" url:"name,omitempty"`       // Name of the account holder or business.
	Address string `json:"address,omitempty" url:"address,omitempty"` // City of the account holder.
	City    string `json:"city,omitempty" url:"city,omitempty"`       // Address of the account holder.
}

type Subaccount struct {
	Account     string `json:"account,omitempty" url:"account,omitempty"`
	ApiID       string `json:"api_id,omitempty" url:"api_id,omitempty"`
	AuthID      string `json:"auth_id,omitempty" url:"auth_id,omitempty"`
	AuthToken   string `json:"auth_token,omitempty" url:"auth_token,omitempty"`
	Created     string `json:"created,omitempty" url:"created,omitempty"`
	Modified    string `json:"modified,omitempty" url:"modified,omitempty"`
	Name        string `json:"name,omitempty" url:"name,omitempty"`
	Enabled     bool   `json:"enabled,omitempty" url:"enabled,omitempty"`
	ResourceURI string `json:"resource_uri,omitempty" url:"resource_uri,omitempty"`
}

type SubaccountCreateParams struct {
	Name    string `json:"name,omitempty" url:"name,omitempty"`       // Name of the subaccount
	Enabled bool   `json:"enabled,omitempty" url:"enabled,omitempty"` // Specify if the subaccount should be enabled or not. Takes a value of True or False. Defaults to False
}

type SubaccountUpdateParams SubaccountCreateParams

type SubaccountDeleteParams struct {
	Cascade bool `json:"cascade,omitempty" url:"cascade,omitempty"` // Specify if the sub account should be cascade deleted or not. Takes a value of True or False. Defaults to False
}

type SubaccountCreateResponse struct {
	BaseResponse
	AuthId    string `json:"auth_id" url:"auth_id"`
	AuthToken string `json:"auth_token" url:"auth_token"`
}

type SubaccountUpdateResponse BaseResponse

type SubaccountListParams struct {
	Limit  int `json:"limit,omitempty" url:"limit,omitempty"`
	Offset int `json:"offset,omitempty" url:"offset,omitempty"`
}

type SubaccountList struct {
	BaseListResponse
	Objects []Subaccount
}

func (service *AccountService) Update(params AccountUpdateParams) (response *SubaccountUpdateResponse, err error) {
	request, err := service.client.NewRequest("POST", params, "")
	if err != nil {
		return
	}
	response = &SubaccountUpdateResponse{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func (service *AccountService) Get() (response *Account, err error) {
	request, err := service.client.NewRequest("GET", nil, "")
	if err != nil {
		return
	}
	response = &Account{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func (service *SubaccountService) Create(params SubaccountCreateParams) (response *SubaccountCreateResponse, err error) {
	request, err := service.client.NewRequest("POST", params, "Subaccount")
	if err != nil {
		return
	}
	response = &SubaccountCreateResponse{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func (service *SubaccountService) Update(subauthId string, params SubaccountUpdateParams) (response *SubaccountUpdateResponse, err error) {
	request, err := service.client.NewRequest("POST", params, "Subaccount/%s", subauthId)
	if err != nil {
		return
	}
	response = &SubaccountUpdateResponse{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func (service *SubaccountService) Get(subauthId string) (response *Subaccount, err error) {
	request, err := service.client.NewRequest("GET", nil, "Subaccount/%s", subauthId)
	if err != nil {
		return
	}
	response = &Subaccount{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func (service *SubaccountService) List(params SubaccountListParams) (response *SubaccountList, err error) {
	request, err := service.client.NewRequest("GET", params, "Subaccount")
	if err != nil {
		return
	}
	response = &SubaccountList{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func (service *SubaccountService) Delete(subauthId string, data ...SubaccountDeleteParams) (err error) {
	var optionalParams interface{}
	if data != nil {
		optionalParams = data[0]
	}
	request, err := service.client.NewRequest("DELETE", optionalParams, "Subaccount/%s", subauthId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(request, nil)
	return
}
