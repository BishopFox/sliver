package slack

import (
	"context"
	"net/url"
)

// AdminConversationsRestrictAccessAddGroup

type adminConversationsRestrictAccessAddGroupParams struct {
	teamID string
}

// AdminConversationsRestrictAccessAddGroupOption is an option for AdminConversationsRestrictAccessAddGroup.
type AdminConversationsRestrictAccessAddGroupOption func(*adminConversationsRestrictAccessAddGroupParams)

// AdminConversationsRestrictAccessAddGroupOptionTeamID sets the workspace where the channel exists.
// Required if using an org token.
func AdminConversationsRestrictAccessAddGroupOptionTeamID(teamID string) AdminConversationsRestrictAccessAddGroupOption {
	return func(params *adminConversationsRestrictAccessAddGroupParams) {
		params.teamID = teamID
	}
}

// AdminConversationsRestrictAccessAddGroup adds an allowlist of IDP groups
// for accessing a channel.
// For more information see the admin.conversations.restrictAccess.addGroup docs:
// https://api.slack.com/methods/admin.conversations.restrictAccess.addGroup
func (api *Client) AdminConversationsRestrictAccessAddGroup(ctx context.Context, channelID, groupID string, options ...AdminConversationsRestrictAccessAddGroupOption) error {
	params := adminConversationsRestrictAccessAddGroupParams{}
	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token":      {api.token},
		"channel_id": {channelID},
		"group_id":   {groupID},
	}

	if params.teamID != "" {
		values.Add("team_id", params.teamID)
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.conversations.restrictAccess.addGroup", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}

// AdminConversationsRestrictAccessListGroups

type adminConversationsRestrictAccessListGroupsParams struct {
	teamID string
}

// AdminConversationsRestrictAccessListGroupsOption is an option for AdminConversationsRestrictAccessListGroups.
type AdminConversationsRestrictAccessListGroupsOption func(*adminConversationsRestrictAccessListGroupsParams)

// AdminConversationsRestrictAccessListGroupsOptionTeamID sets the workspace where the channel exists.
// Required if using an org token.
func AdminConversationsRestrictAccessListGroupsOptionTeamID(teamID string) AdminConversationsRestrictAccessListGroupsOption {
	return func(params *adminConversationsRestrictAccessListGroupsParams) {
		params.teamID = teamID
	}
}

// AdminConversationsRestrictAccessListGroupsResponse represents the response from
// admin.conversations.restrictAccess.listGroups.
type AdminConversationsRestrictAccessListGroupsResponse struct {
	SlackResponse
	GroupIDs []string `json:"group_ids"`
}

// AdminConversationsRestrictAccessListGroups lists the allowlist of IDP groups
// for a private channel.
// For more information see the admin.conversations.restrictAccess.listGroups docs:
// https://api.slack.com/methods/admin.conversations.restrictAccess.listGroups
func (api *Client) AdminConversationsRestrictAccessListGroups(ctx context.Context, channelID string, options ...AdminConversationsRestrictAccessListGroupsOption) ([]string, error) {
	params := adminConversationsRestrictAccessListGroupsParams{}
	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token":      {api.token},
		"channel_id": {channelID},
	}

	if params.teamID != "" {
		values.Add("team_id", params.teamID)
	}

	response := &AdminConversationsRestrictAccessListGroupsResponse{}
	err := api.postMethod(ctx, "admin.conversations.restrictAccess.listGroups", values, response)
	if err != nil {
		return nil, err
	}

	return response.GroupIDs, response.Err()
}

// AdminConversationsRestrictAccessRemoveGroup

type adminConversationsRestrictAccessRemoveGroupParams struct {
	teamID string
}

// AdminConversationsRestrictAccessRemoveGroupOption is an option for AdminConversationsRestrictAccessRemoveGroup.
type AdminConversationsRestrictAccessRemoveGroupOption func(*adminConversationsRestrictAccessRemoveGroupParams)

// AdminConversationsRestrictAccessRemoveGroupOptionTeamID sets the workspace where the channel exists.
// Required if using an org token.
func AdminConversationsRestrictAccessRemoveGroupOptionTeamID(teamID string) AdminConversationsRestrictAccessRemoveGroupOption {
	return func(params *adminConversationsRestrictAccessRemoveGroupParams) {
		params.teamID = teamID
	}
}

// AdminConversationsRestrictAccessRemoveGroup removes an IDP group from the
// allowlist of a private channel.
// For more information see the admin.conversations.restrictAccess.removeGroup docs:
// https://api.slack.com/methods/admin.conversations.restrictAccess.removeGroup
func (api *Client) AdminConversationsRestrictAccessRemoveGroup(ctx context.Context, channelID, groupID string, options ...AdminConversationsRestrictAccessRemoveGroupOption) error {
	params := adminConversationsRestrictAccessRemoveGroupParams{}
	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token":      {api.token},
		"channel_id": {channelID},
		"group_id":   {groupID},
	}

	if params.teamID != "" {
		values.Add("team_id", params.teamID)
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.conversations.restrictAccess.removeGroup", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}
