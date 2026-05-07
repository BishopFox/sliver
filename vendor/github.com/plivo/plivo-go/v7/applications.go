package plivo

type ApplicationService struct {
	client *Client
}

type Application struct {
	FallbackMethod      string `json:"fallback_method,omitempty" url:"fallback_method,omitempty"`
	DefaultApp          bool   `json:"default_app,omitempty" url:"default_app,omitempty"`
	AppName             string `json:"app_name,omitempty" url:"app_name,omitempty"`
	ProductionApp       bool   `json:"production_app,omitempty" url:"production_app,omitempty"`
	AppID               string `json:"app_id,omitempty" url:"app_id,omitempty"`
	HangupURL           string `json:"hangup_url,omitempty" url:"hangup_url,omitempty"`
	AnswerURL           string `json:"answer_url,omitempty" url:"answer_url,omitempty"`
	MessageURL          string `json:"message_url,omitempty" url:"message_url,omitempty"`
	ResourceURI         string `json:"resource_uri,omitempty" url:"resource_uri,omitempty"`
	HangupMethod        string `json:"hangup_method,omitempty" url:"hangup_method,omitempty"`
	MessageMethod       string `json:"message_method,omitempty" url:"message_method,omitempty"`
	FallbackAnswerURL   string `json:"fallback_answer_url,omitempty" url:"fallback_answer_url,omitempty"`
	AnswerMethod        string `json:"answer_method,omitempty" url:"answer_method,omitempty"`
	ApiID               string `json:"api_id,omitempty" url:"api_id,omitempty"`
	LogIncomingMessages bool   `json:"log_incoming_messages,omitempty" url:"log_incoming_messages,omitempty"`
	PublicURI           bool   `json:"public_uri,omitempty" url:"public_uri,omitempty"`

	// Additional fields for Modify calls
	DefaultNumberApp   bool `json:"default_number_app,omitempty" url:"default_number_app,omitempty"`
	DefaultEndpointApp bool `json:"default_endpoint_app,omitempty" url:"default_endpoint_app,omitempty"`
}

//TODO Verify against docs
type ApplicationCreateParams struct {
	FallbackMethod      string `json:"fallback_method,omitempty" url:"fallback_method,omitempty"`
	DefaultApp          bool   `json:"default_app,omitempty" url:"default_app,omitempty"`
	AppName             string `json:"app_name,omitempty" url:"app_name,omitempty"`
	ProductionApp       bool   `json:"production_app,omitempty" url:"production_app,omitempty"`
	AppID               string `json:"app_id,omitempty" url:"app_id,omitempty"`
	HangupURL           string `json:"hangup_url,omitempty" url:"hangup_url,omitempty"`
	AnswerURL           string `json:"answer_url,omitempty" url:"answer_url,omitempty"`
	MessageURL          string `json:"message_url,omitempty" url:"message_url,omitempty"`
	ResourceURI         string `json:"resource_uri,omitempty" url:"resource_uri,omitempty"`
	HangupMethod        string `json:"hangup_method,omitempty" url:"hangup_method,omitempty"`
	MessageMethod       string `json:"message_method,omitempty" url:"message_method,omitempty"`
	FallbackAnswerURL   string `json:"fallback_answer_url,omitempty" url:"fallback_answer_url,omitempty"`
	AnswerMethod        string `json:"answer_method,omitempty" url:"answer_method,omitempty"`
	ApiID               string `json:"api_id,omitempty" url:"api_id,omitempty"`
	LogIncomingMessages bool   `json:"log_incoming_messages,omitempty" url:"log_incoming_messages,omitempty"`
	PublicURI           bool   `json:"public_uri,omitempty" url:"public_uri,omitempty"`

	// Additional fields for Modify calls
	DefaultNumberApp   bool `json:"default_number_app,omitempty" url:"default_number_app,omitempty"`
	DefaultEndpointApp bool `json:"default_endpoint_app,omitempty" url:"default_endpoint_app,omitempty"`
}

// TODO Check against docs
type ApplicationUpdateParams ApplicationCreateParams

// Stores response for Create call
type ApplicationCreateResponseBody struct {
	Message string `json:"message" url:"message"`
	ApiID   string `json:"api_id" url:"api_id"`
	AppID   string `json:"app_id" url:"app_id"`
}

type ApplicationListParams struct {
	Subaccount string `url:"subaccount,omitempty"`
	AppName    string `url:"app_name,omitempty"`
	Limit      int    `url:"limit,omitempty"`
	Offset     int    `url:"offset,omitempty"`
}

type ApplicationList struct {
	BaseListResponse
	Objects []Application `json:"objects" url:"objects"`
}

type ApplicationDeleteParams struct {
	Cascade                bool   `json:"cascade" url:"cascade"` // Specify if the Application should be cascade deleted or not. Takes a value of True or False
	NewEndpointApplication string `json:"new_endpoint_application,omitempty" url:"new_endpoint_application,omitempty"`
}

type ApplicationUpdateResponse BaseResponse

func (service *ApplicationService) Create(params ApplicationCreateParams) (response *ApplicationCreateResponseBody, err error) {
	request, err := service.client.NewRequest("POST", params, "Application")
	if err != nil {
		return
	}
	response = &ApplicationCreateResponseBody{}
	err = service.client.ExecuteRequest(request, response, isVoiceRequest())
	return
}

func (service *ApplicationService) List(params ApplicationListParams) (response *ApplicationList, err error) {
	request, err := service.client.NewRequest("GET", params, "Application")
	if err != nil {
		return
	}
	response = &ApplicationList{}
	err = service.client.ExecuteRequest(request, response, isVoiceRequest())
	return
}

func (service *ApplicationService) Get(appId string) (response *Application, err error) {
	request, err := service.client.NewRequest("GET", nil, "Application/%s", appId)
	if err != nil {
		return
	}
	response = &Application{}
	err = service.client.ExecuteRequest(request, response, isVoiceRequest())
	return
}

func (service *ApplicationService) Update(appId string, params ApplicationUpdateParams) (response *ApplicationUpdateResponse, err error) {
	request, err := service.client.NewRequest("POST", params, "Application/%s", appId)
	if err != nil {
		return
	}
	response = &ApplicationUpdateResponse{}
	err = service.client.ExecuteRequest(request, response, isVoiceRequest())
	return
}

func (service *ApplicationService) Delete(appId string, data ...ApplicationDeleteParams) (err error) {
	var optionalParams interface{}
	if data != nil {
		optionalParams = data[0]
	}
	request, err := service.client.NewRequest("DELETE", optionalParams, "Application/%s", appId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(request, nil, isVoiceRequest())
	return
}
