package plivo

type EndpointService struct {
	client *Client
}

type Endpoint struct {
	Alias       string `json:"alias,omitempty" url:"alias,omitempty"`
	EndpointID  string `json:"endpoint_id,omitempty" url:"endpoint_id,omitempty"`
	Password    string `json:"password,omitempty" url:"password,omitempty"`
	ResourceURI string `json:"resource_uri,omitempty" url:"resource_uri,omitempty"`
	SIPURI      string `json:"sip_uri,omitempty" url:"sip_uri,omitempty"`
	Username    string `json:"username,omitempty" url:"username,omitempty"`
	// Optional field for Create call.
	AppID string `json:"app_id,omitempty" url:"app_id,omitempty"`
}

type EndpointUpdateParams struct {
	Alias    string `json:"alias,omitempty" url:"alias,omitempty"`
	Password string `json:"password,omitempty" url:"password,omitempty"`
	AppID    string `json:"app_id,omitempty" url:"app_id,omitempty"`
}

type EndpointCreateParams struct {
	Alias    string `json:"alias,omitempty" url:"alias,omitempty"`
	Password string `json:"password,omitempty" url:"password,omitempty"`
	AppID    string `json:"app_id,omitempty" url:"app_id,omitempty"`
	Username string `json:"username,omitempty" url:"username,omitempty"`
}

type EndpointCreateResponse struct {
	BaseResponse
	Alias      string `json:"alias,omitempty" url:"alias,omitempty"`
	EndpointID string `json:"endpoint_id,omitempty" url:"endpoint_id,omitempty"`
	Username   string `json:"username,omitempty" url:"username,omitempty"`
}

type EndpointListResponse struct {
	BaseListResponse
	Objects []*Endpoint `json:"objects" url:"objects"`
}

type EndpointUpdateResponse BaseResponse

type EndpointListParams struct {
	Limit  int `url:"limit,omitempty"`
	Offset int `url:"offset,omitempty"`
}

func (service *EndpointService) Create(params EndpointCreateParams) (response *EndpointCreateResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "Endpoint")
	if err != nil {
		return
	}
	response = &EndpointCreateResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *EndpointService) Get(endpointId string) (response *Endpoint, err error) {
	req, err := service.client.NewRequest("GET", nil, "Endpoint/%s", endpointId)
	if err != nil {
		return
	}
	response = &Endpoint{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *EndpointService) Delete(endpointId string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Endpoint/%s", endpointId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *EndpointService) Update(endpointId string, params EndpointUpdateParams) (response *EndpointUpdateResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "Endpoint/%s", endpointId)
	if err != nil {
		return
	}
	response = &EndpointUpdateResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *EndpointService) List(params EndpointListParams) (response *EndpointListResponse, err error) {
	req, err := service.client.NewRequest("GET", params, "Endpoint")
	if err != nil {
		return
	}
	response = &EndpointListResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}
