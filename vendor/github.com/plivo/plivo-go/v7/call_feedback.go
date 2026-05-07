package plivo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

const CallInsightsParams = "call_insights_params"
const CallInsightsBaseURL = "stats.plivo.com"
const CallInsightsFeedbackPath = "v1/Call/%s/Feedback/"
const CallInsightsRequestPath = "call_insights_feedback_path"

type CallFeedbackService struct {
	client *Client
}

type CallFeedbackParams struct {
	CallUUID string      `json:"call_uuid"`
	Notes    string      `json:"notes"`
	Rating   interface{} `json:"rating"`
	Issues   []string    `json:"issues"`
}

type CallFeedbackCreateResponse struct {
	ApiID   string `json:"api_id"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

func (service *CallFeedbackService) Create(params CallFeedbackParams) (response *CallFeedbackCreateResponse, err error) {
	if service.client == nil {
		err = errors.New("client cannot be nil")
		return
	}
	if params.CallUUID == "" {
		err = errors.New("CallUUID cannot be nil")
		return
	}
	if params.Rating == nil {
		err = errors.New("rating cannot be nil")
		return
	}
	var buffer = new(bytes.Buffer)
	if err = json.NewEncoder(buffer).Encode(params); err != nil {
		return
	}
	formatParams := map[string]interface{}{}
	formatParams[CallInsightsParams] = make(map[string]interface{})
	feedbackPath := fmt.Sprintf(CallInsightsFeedbackPath, params.CallUUID)
	formatParams[CallInsightsParams].(map[string]interface{})[CallInsightsRequestPath] = feedbackPath
	req, err := service.client.NewRequest("POST", params, "Call", formatParams)
	if err != nil {
		return
	}
	response = &CallFeedbackCreateResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}
