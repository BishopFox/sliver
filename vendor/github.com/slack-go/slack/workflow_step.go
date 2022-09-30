package slack

import (
	"context"
	"encoding/json"
)

const VTWorkflowStep ViewType = "workflow_step"

type (
	ConfigurationModalRequest struct {
		ModalViewRequest
	}

	WorkflowStepCompleteResponse struct {
		WorkflowStepEditID string                `json:"workflow_step_edit_id"`
		Inputs             *WorkflowStepInputs   `json:"inputs,omitempty"`
		Outputs            *[]WorkflowStepOutput `json:"outputs,omitempty"`
	}

	WorkflowStepInputElement struct {
		Value                   string `json:"value"`
		SkipVariableReplacement bool   `json:"skip_variable_replacement"`
	}

	WorkflowStepInputs map[string]WorkflowStepInputElement

	WorkflowStepOutput struct {
		Name  string `json:"name"`
		Type  string `json:"type"`
		Label string `json:"label"`
	}
)

func NewConfigurationModalRequest(blocks Blocks, privateMetaData string, externalID string) *ConfigurationModalRequest {
	return &ConfigurationModalRequest{
		ModalViewRequest{
			Type:            VTWorkflowStep,
			Title:           nil, // slack configuration modal must not have a title!
			Blocks:          blocks,
			PrivateMetadata: privateMetaData,
			ExternalID:      externalID,
		},
	}
}

func (api *Client) SaveWorkflowStepConfiguration(workflowStepEditID string, inputs *WorkflowStepInputs, outputs *[]WorkflowStepOutput) error {
	return api.SaveWorkflowStepConfigurationContext(context.Background(), workflowStepEditID, inputs, outputs)
}

func (api *Client) SaveWorkflowStepConfigurationContext(ctx context.Context, workflowStepEditID string, inputs *WorkflowStepInputs, outputs *[]WorkflowStepOutput) error {
	// More information: https://api.slack.com/methods/workflows.updateStep
	wscr := WorkflowStepCompleteResponse{
		WorkflowStepEditID: workflowStepEditID,
		Inputs:             inputs,
		Outputs:            outputs,
	}

	endpoint := api.endpoint + "workflows.updateStep"
	jsonData, err := json.Marshal(wscr)
	if err != nil {
		return err
	}

	response := &SlackResponse{}
	if err := postJSON(ctx, api.httpclient, endpoint, api.token, jsonData, response, api); err != nil {
		return err
	}

	if !response.Ok {
		return response.Err()
	}

	return nil
}

func GetInitialOptionFromWorkflowStepInput(selection *SelectBlockElement, inputs *WorkflowStepInputs, options []*OptionBlockObject) (*OptionBlockObject, bool) {
	if len(*inputs) == 0 {
		return &OptionBlockObject{}, false
	}
	if len(options) == 0 {
		return &OptionBlockObject{}, false
	}

	if val, ok := (*inputs)[selection.ActionID]; ok {
		if val.SkipVariableReplacement {
			return &OptionBlockObject{}, false
		}

		for _, option := range options {
			if option.Value == val.Value {
				return option, true
			}
		}
	}

	return &OptionBlockObject{}, false
}
