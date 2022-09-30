package slack

import (
	"context"
	"encoding/json"
)

type (
	WorkflowStepCompletedRequest struct {
		WorkflowStepExecuteID string            `json:"workflow_step_execute_id"`
		Outputs               map[string]string `json:"outputs"`
	}

	WorkflowStepFailedRequest struct {
		WorkflowStepExecuteID string `json:"workflow_step_execute_id"`
		Error                 struct {
			Message string `json:"message"`
		} `json:"error"`
	}
)

type WorkflowStepCompletedRequestOption func(opt WorkflowStepCompletedRequest) error

func WorkflowStepCompletedRequestOptionOutput(outputs map[string]string) WorkflowStepCompletedRequestOption {
	return func(opt WorkflowStepCompletedRequest) error {
		if len(outputs) > 0 {
			opt.Outputs = outputs
		}
		return nil
	}
}

// WorkflowStepCompleted indicates step is completed
func (api *Client) WorkflowStepCompleted(workflowStepExecuteID string, options ...WorkflowStepCompletedRequestOption) error {
	// More information: https://api.slack.com/methods/workflows.stepCompleted
	r := WorkflowStepCompletedRequest{
		WorkflowStepExecuteID: workflowStepExecuteID,
	}
	for _, option := range options {
		option(r)
	}

	endpoint := api.endpoint + "workflows.stepCompleted"
	jsonData, err := json.Marshal(r)
	if err != nil {
		return err
	}

	response := &SlackResponse{}
	if err := postJSON(context.Background(), api.httpclient, endpoint, api.token, jsonData, response, api); err != nil {
		return err
	}

	if !response.Ok {
		return response.Err()
	}

	return nil
}

// WorkflowStepFailed indicates step is failed
func (api *Client) WorkflowStepFailed(workflowStepExecuteID string, errorMessage string) error {
	// More information: https://api.slack.com/methods/workflows.stepFailed
	r := WorkflowStepFailedRequest{
		WorkflowStepExecuteID: workflowStepExecuteID,
	}
	r.Error.Message = errorMessage

	endpoint := api.endpoint + "workflows.stepFailed"
	jsonData, err := json.Marshal(r)
	if err != nil {
		return err
	}

	response := &SlackResponse{}
	if err := postJSON(context.Background(), api.httpclient, endpoint, api.token, jsonData, response, api); err != nil {
		return err
	}

	if !response.Ok {
		return response.Err()
	}

	return nil
}
