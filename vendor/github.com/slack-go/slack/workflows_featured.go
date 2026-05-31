package slack

import (
	"context"
	"encoding/json"
	"fmt"
)

type (
	FeaturedWorkflowTrigger struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}

	FeaturedWorkflow struct {
		ChannelID string                    `json:"channel_id"`
		Triggers  []FeaturedWorkflowTrigger `json:"triggers"`
	}

	WorkflowsFeaturedAddInput struct {
		ChannelID  string   `json:"channel_id"`
		TriggerIDs []string `json:"trigger_ids"`
	}

	WorkflowsFeaturedListInput struct {
		ChannelIDs []string `json:"channel_ids"`
	}

	WorkflowsFeaturedListOutput struct {
		FeaturedWorkflows []FeaturedWorkflow `json:"featured_workflows"`
	}

	WorkflowsFeaturedRemoveInput struct {
		ChannelID  string   `json:"channel_id"`
		TriggerIDs []string `json:"trigger_ids"`
	}

	WorkflowsFeaturedSetInput struct {
		ChannelID  string   `json:"channel_id"`
		TriggerIDs []string `json:"trigger_ids"`
	}
)

// WorkflowsFeaturedAdd adds featured workflows to a channel.
//
// Slack API Docs:https://api.slack.com/methods/workflows.featured.add
func (api *Client) WorkflowsFeaturedAdd(ctx context.Context, input *WorkflowsFeaturedAddInput) error {
	response := struct {
		SlackResponse
	}{}

	jsonPayload, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal WorkflowsFeaturedAddInput: %w", err)
	}

	err = api.postJSONMethod(ctx, "workflows.featured.add", api.token, jsonPayload, &response)
	if err != nil {
		return err
	}

	if err := response.Err(); err != nil {
		return err
	}

	return nil
}

// WorkflowsFeaturedList lists featured workflows for the given channels.
//
// Slack API Docs:https://api.slack.com/methods/workflows.featured.list
func (api *Client) WorkflowsFeaturedList(ctx context.Context, input *WorkflowsFeaturedListInput) (*WorkflowsFeaturedListOutput, error) {
	response := struct {
		SlackResponse
		*WorkflowsFeaturedListOutput
	}{}

	jsonPayload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal WorkflowsFeaturedListInput: %w", err)
	}

	err = api.postJSONMethod(ctx, "workflows.featured.list", api.token, jsonPayload, &response)
	if err != nil {
		return nil, err
	}

	if err := response.Err(); err != nil {
		return nil, err
	}

	return response.WorkflowsFeaturedListOutput, nil
}

// WorkflowsFeaturedRemove removes featured workflows from a channel.
//
// Slack API Docs:https://api.slack.com/methods/workflows.featured.remove
func (api *Client) WorkflowsFeaturedRemove(ctx context.Context, input *WorkflowsFeaturedRemoveInput) error {
	response := struct {
		SlackResponse
	}{}

	jsonPayload, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal WorkflowsFeaturedRemoveInput: %w", err)
	}

	err = api.postJSONMethod(ctx, "workflows.featured.remove", api.token, jsonPayload, &response)
	if err != nil {
		return err
	}

	if err := response.Err(); err != nil {
		return err
	}

	return nil
}

// WorkflowsFeaturedSet replaces all featured workflows in a channel with the given triggers.
//
// Slack API Docs:https://api.slack.com/methods/workflows.featured.set
func (api *Client) WorkflowsFeaturedSet(ctx context.Context, input *WorkflowsFeaturedSetInput) error {
	response := struct {
		SlackResponse
	}{}

	jsonPayload, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal WorkflowsFeaturedSetInput: %w", err)
	}

	err = api.postJSONMethod(ctx, "workflows.featured.set", api.token, jsonPayload, &response)
	if err != nil {
		return err
	}

	if err := response.Err(); err != nil {
		return err
	}

	return nil
}
