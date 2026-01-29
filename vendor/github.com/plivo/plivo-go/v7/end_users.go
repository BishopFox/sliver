package plivo

import "time"

type EndUserService struct {
	client *Client
}

type EndUserGetResponse struct {
	CreatedAt   time.Time `json:"created_at"`
	EndUserID   string    `json:"end_user_id"`
	Name        string    `json:"name"`
	LastName    string    `json:"last_name"`
	EndUserType string    `json:"end_user_type"`
}

type EndUserListParams struct {
	Limit       int    `json:"limit,omitempty" url:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty" url:"offset,omitempty"`
	Name        string `json:"name,omitempty" url:"name,omitempty"`
	LastName    string `json:"last_name,omitempty" url:"last_name,omitempty"`
	EndUserType string `json:"end_user_type,omitempty" url:"end_user_type,omitempty"`
}

type EndUserListResponse struct {
	BaseListResponse
	Objects []EndUserGetResponse `json:"objects" url:"objects"`
}

type CreateEndUserResponse struct {
	CreatedAt   time.Time `json:"created_at"`
	EndUserID   string    `json:"end_user_id"`
	Name        string    `json:"name"`
	LastName    string    `json:"last_name"`
	EndUserType string    `json:"end_user_type"`
	APIID       string    `json:"api_id"`
	Message     string    `json:"message"`
}

type EndUserParams struct {
	Name        string `json:"name,omitempty" url:"name,omitempty"`
	LastName    string `json:"last_name,omitempty" url:"last_name,omitempty"`
	EndUserType string `json:"end_user_type,omitempty" url:"end_user_type,omitempty"`
}

type UpdateEndUserParams struct {
	EndUserParams
	EndUserID string `json:"end_user_id"`
}

type UpdateEndUserResponse BaseResponse

func (service *EndUserService) Get(endUserId string) (response *EndUserGetResponse, err error) {
	req, err := service.client.NewRequest("GET", nil, "EndUser/%s", endUserId)
	if err != nil {
		return nil, err
	}
	response = &EndUserGetResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *EndUserService) List(params EndUserListParams) (response *EndUserListResponse, err error) {
	request, err := service.client.NewRequest("GET", params, "EndUser")
	if err != nil {
		return
	}
	response = &EndUserListResponse{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func (service *EndUserService) Create(params EndUserParams) (response *CreateEndUserResponse, err error) {
	request, err := service.client.NewRequest("POST", params, "EndUser")
	if err != nil {
		return
	}
	response = &CreateEndUserResponse{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func (service *EndUserService) Update(params UpdateEndUserParams) (response *UpdateEndUserResponse, err error) {
	request, err := service.client.NewRequest("POST", params, "EndUser/%s", params.EndUserID)
	if err != nil {
		return
	}
	response = &UpdateEndUserResponse{}
	err = service.client.ExecuteRequest(request, response)
	return
}

func (service *EndUserService) Delete(endUserId string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "EndUser/%s", endUserId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil)
	return
}
