package plivo

type TokenService struct {
	client *Client
}

type TokenCreateParams struct {
	// Required parameters.
	Iss string `json:"iss,omitempty" url:"iss,omitempty"`
	// Optional parameters.
	Exp           int64       `json:"exp,omitempty" url:"Exp,omitempty"`
	Nbf           int64       `json:"nbf,omitempty" url:"Nbf,omitempty"`
	IncomingAllow interface{} `json:"incoming_allow,omitempty" url:"incoming_allow,omitempty"`
	OutgoingAllow interface{} `json:"outgoing_allow,omitempty" url:"outgoing_allow,omitempty"`
	Per           interface{} `json:"per,omitempty" url:"per,omitempty"`
	App           string      `json:"app,omitempty" url:"App,omitempty"`
	Sub           string      `json:"sub,omitempty" url:"Sub,omitempty"`
}

// Stores response for creating a token.

type TokenCreateResponse struct {
	Token string `json:"token" url:"token"`
	ApiID string `json:"api_id" url:"api_id"`
}

func (service *TokenService) Create(params TokenCreateParams) (response *TokenCreateResponse, err error) {

	if params.IncomingAllow != nil || params.OutgoingAllow != nil {
		voicemap := make(map[string]interface{})
		if params.IncomingAllow != nil {
			voicemap["incoming_allow"] = params.IncomingAllow
		}
		if params.OutgoingAllow != nil {
			voicemap["outgoing_allow"] = params.OutgoingAllow
		}
		if len(voicemap) > 0 {
			permissionsMap := map[string]interface{}{"voice": voicemap}
			params.Per = permissionsMap
		}
	}

	req, err := service.client.NewRequest("POST", params, "JWT/Token")
	if err != nil {
		return
	}

	response = &TokenCreateResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}
