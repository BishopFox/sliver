package slack

import (
	"context"
	"encoding/json"
	"fmt"
)

type (
	WorkflowsTriggersPermissionsAddInput struct {
		TriggerId  string   `json:"trigger_id"`
		ChannelIds []string `json:"channel_ids,omitempty"`
		OrgIds     []string `json:"org_ids,omitempty"`
		TeamIds    []string `json:"team_ids,omitempty"`
		UserIds    []string `json:"user_ids,omitempty"`
	}

	WorkflowsTriggersPermissionsAddOutput struct {
		PermissionType string   `json:"permission_type"`
		ChannelIds     []string `json:"channel_ids,omitempty"`
		OrgIds         []string `json:"org_ids,omitempty"`
		TeamIds        []string `json:"team_ids,omitempty"`
		UserIds        []string `json:"user_ids,omitempty"`
	}

	WorkflowsTriggersPermissionsListInput struct {
		TriggerId string `json:"trigger_id"`
	}

	WorkflowsTriggersPermissionsListOutput struct {
		PermissionType string   `json:"permission_type"`
		ChannelIds     []string `json:"channel_ids,omitempty"`
		OrgIds         []string `json:"org_ids,omitempty"`
		TeamIds        []string `json:"team_ids,omitempty"`
		UserIds        []string `json:"user_ids,omitempty"`
	}

	WorkflowsTriggersPermissionsRemoveInput struct {
		TriggerId  string   `json:"trigger_id"`
		ChannelIds []string `json:"channel_ids,omitempty"`
		OrgIds     []string `json:"org_ids,omitempty"`
		TeamIds    []string `json:"team_ids,omitempty"`
		UserIds    []string `json:"user_ids,omitempty"`
	}

	WorkflowsTriggersPermissionsRemoveOutput struct {
		PermissionType string   `json:"permission_type"`
		ChannelIds     []string `json:"channel_ids,omitempty"`
		OrgIds         []string `json:"org_ids,omitempty"`
		TeamIds        []string `json:"team_ids,omitempty"`
		UserIds        []string `json:"user_ids,omitempty"`
	}

	WorkflowsTriggersPermissionsSetInput struct {
		PermissionType string   `json:"permission_type"`
		TriggerId      string   `json:"trigger_id"`
		ChannelIds     []string `json:"channel_ids,omitempty"`
		OrgIds         []string `json:"org_ids,omitempty"`
		TeamIds        []string `json:"team_ids,omitempty"`
		UserIds        []string `json:"user_ids,omitempty"`
	}

	WorkflowsTriggersPermissionsSetOutput struct {
		PermissionType string   `json:"permission_type"`
		ChannelIds     []string `json:"channel_ids,omitempty"`
		OrgIds         []string `json:"org_ids,omitempty"`
		TeamIds        []string `json:"team_ids,omitempty"`
		UserIds        []string `json:"user_ids,omitempty"`
	}
)

// WorkflowsTriggersPermissionsAdd allows users to run a trigger that has its permission
// type set to named_entities.
//
// Slack API Docs:https://api.slack.com/methods/workflows.triggers.permissions.add
func (api *Client) WorkflowsTriggersPermissionsAdd(ctx context.Context, input *WorkflowsTriggersPermissionsAddInput) (*WorkflowsTriggersPermissionsAddOutput, error) {
	response := struct {
		SlackResponse
		*WorkflowsTriggersPermissionsAddOutput
	}{}

	jsonPayload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal WorkflowsTriggersPermissionsAddInput: %w", err)
	}

	err = postJSON(ctx, api.httpclient, api.endpoint+"workflows.triggers.permissions.add", api.token, jsonPayload, &response, api)
	if err != nil {
		return nil, err
	}

	if err := response.Err(); err != nil {
		return nil, err
	}

	return response.WorkflowsTriggersPermissionsAddOutput, nil
}

// WorkflowsTriggersPermissionsList returns the permission type of a trigger and if
// applicable, includes the entities that have been granted access.
//
// Slack API Docs:https://api.slack.com/methods/workflows.triggers.permissions.list
func (api *Client) WorkflowsTriggersPermissionsList(ctx context.Context, input *WorkflowsTriggersPermissionsListInput) (*WorkflowsTriggersPermissionsListOutput, error) {
	response := struct {
		SlackResponse
		*WorkflowsTriggersPermissionsListOutput
	}{}

	jsonPayload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal WorkflowsTriggersPermissionsListInput: %w", err)
	}

	err = postJSON(ctx, api.httpclient, api.endpoint+"workflows.triggers.permissions.list", api.token, jsonPayload, &response, api)
	if err != nil {
		return nil, err
	}

	if err := response.Err(); err != nil {
		return nil, err
	}

	return response.WorkflowsTriggersPermissionsListOutput, nil
}

// WorkflowsTriggersPermissionsRemove revoke an entity's access to a trigger that has its
// permission type set to named_entities.
//
// Slack API Docs:https://api.slack.com/methods/workflows.triggers.permissions.remove
func (api *Client) WorkflowsTriggersPermissionsRemove(ctx context.Context, input *WorkflowsTriggersPermissionsRemoveInput) (*WorkflowsTriggersPermissionsRemoveOutput, error) {
	response := struct {
		SlackResponse
		*WorkflowsTriggersPermissionsRemoveOutput
	}{}

	jsonPayload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal WorkflowsTriggersPermissionsRemoveInput: %w", err)
	}

	err = postJSON(ctx, api.httpclient, api.endpoint+"workflows.triggers.permissions.remove", api.token, jsonPayload, &response, api)
	if err != nil {
		return nil, err
	}

	if err := response.Err(); err != nil {
		return nil, err
	}

	return response.WorkflowsTriggersPermissionsRemoveOutput, nil
}

// WorkflowsTriggersPermissionsSet sets the permission type for who can run a trigger.
//
// Slack API Docs:https://api.slack.com/methods/workflows.triggers.permissions.set
func (api *Client) WorkflowsTriggersPermissionsSet(ctx context.Context, input *WorkflowsTriggersPermissionsSetInput) (*WorkflowsTriggersPermissionsSetOutput, error) {
	response := struct {
		SlackResponse
		*WorkflowsTriggersPermissionsSetOutput
	}{}

	jsonPayload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal WorkflowsTriggersPermissionsSetInput: %w", err)
	}

	err = postJSON(ctx, api.httpclient, api.endpoint+"workflows.triggers.permissions.set", api.token, jsonPayload, &response, api)
	if err != nil {
		return nil, err
	}

	if err := response.Err(); err != nil {
		return nil, err
	}

	return response.WorkflowsTriggersPermissionsSetOutput, nil
}
