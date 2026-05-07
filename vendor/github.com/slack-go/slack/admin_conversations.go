package slack

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// AdminConversationsSetTeamsParams contains arguments for AdminConversationsSetTeams
// method calls.
type AdminConversationsSetTeamsParams struct {
	ChannelID     string
	OrgChannel    *bool
	TargetTeamIDs []string
	TeamID        *string
}

// Set the workspaces in an Enterprise Grid organisation that connect to a public or
// private channel.
// See: https://api.slack.com/methods/admin.conversations.setTeams
func (api *Client) AdminConversationsSetTeams(ctx context.Context, params AdminConversationsSetTeamsParams) error {
	values := url.Values{
		"token":      {api.token},
		"channel_id": {params.ChannelID},
	}

	if params.OrgChannel != nil {
		values.Add("org_channel", strconv.FormatBool(*params.OrgChannel))
	}

	if len(params.TargetTeamIDs) > 0 {
		values.Add("target_team_ids", strings.Join(params.TargetTeamIDs, ",")) // ["T123", "T456"] - > "T123,T456"
	}

	if params.TeamID != nil {
		values.Add("team_id", *params.TeamID)
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.conversations.setTeams", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}

// ConversationsConvertToPrivate converts a public channel to a private channel. To do
// this, you must have the admin.conversations:write scope. There are other requirements:
// you should read the Slack documentation for more details.
// See: https://api.slack.com/methods/admin.conversations.convertToPrivate
func (api *Client) AdminConversationsConvertToPrivate(ctx context.Context, channelID string) error {
	values := url.Values{
		"token":      []string{api.token},
		"channel_id": []string{channelID},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.conversations.convertToPrivate", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}

// ConversationsConvertToPublic converts a private channel to a public channel. To do
// this, you must have the admin.conversations:write scope. There are other requirements:
// you should read the Slack documentation for more details.
// See: https://api.slack.com/methods/admin.conversations.convertToPublic
func (api *Client) AdminConversationsConvertToPublic(ctx context.Context, channelID string) error {
	values := url.Values{
		"token":      []string{api.token},
		"channel_id": []string{channelID},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.conversations.convertToPublic", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}
