package slack

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

type adminConversationsEKMListOriginalConnectedChannelInfoParams struct {
	channelIDs []string
	teamIDs    []string
	cursor     string
	limit      int
}

// AdminConversationsEKMListOriginalConnectedChannelInfoOption is an option for
// AdminConversationsEKMListOriginalConnectedChannelInfo.
type AdminConversationsEKMListOriginalConnectedChannelInfoOption func(*adminConversationsEKMListOriginalConnectedChannelInfoParams)

// AdminConversationsEKMListOriginalConnectedChannelInfoOptionChannelIDs filters results to specific channels.
func AdminConversationsEKMListOriginalConnectedChannelInfoOptionChannelIDs(channelIDs []string) AdminConversationsEKMListOriginalConnectedChannelInfoOption {
	return func(params *adminConversationsEKMListOriginalConnectedChannelInfoParams) {
		params.channelIDs = channelIDs
	}
}

// AdminConversationsEKMListOriginalConnectedChannelInfoOptionTeamIDs filters results to specific teams.
func AdminConversationsEKMListOriginalConnectedChannelInfoOptionTeamIDs(teamIDs []string) AdminConversationsEKMListOriginalConnectedChannelInfoOption {
	return func(params *adminConversationsEKMListOriginalConnectedChannelInfoParams) {
		params.teamIDs = teamIDs
	}
}

// AdminConversationsEKMListOriginalConnectedChannelInfoOptionCursor sets the cursor for pagination.
func AdminConversationsEKMListOriginalConnectedChannelInfoOptionCursor(cursor string) AdminConversationsEKMListOriginalConnectedChannelInfoOption {
	return func(params *adminConversationsEKMListOriginalConnectedChannelInfoParams) {
		params.cursor = cursor
	}
}

// AdminConversationsEKMListOriginalConnectedChannelInfoOptionLimit sets the maximum number of results to return.
func AdminConversationsEKMListOriginalConnectedChannelInfoOptionLimit(limit int) AdminConversationsEKMListOriginalConnectedChannelInfoOption {
	return func(params *adminConversationsEKMListOriginalConnectedChannelInfoParams) {
		params.limit = limit
	}
}

// AdminConversationsEKMOriginalConnectedChannelInfo represents channel info for EKM response.
type AdminConversationsEKMOriginalConnectedChannelInfo struct {
	ID                         string   `json:"id"`
	OriginalConnectedHostID    string   `json:"original_connected_host_id"`
	OriginalConnectedChannelID string   `json:"original_connected_channel_id"`
	InternalTeamIDs            []string `json:"internal_team_ids_count"`
}

// AdminConversationsEKMListOriginalConnectedChannelInfoResponse represents the response from
// admin.conversations.ekm.listOriginalConnectedChannelInfo.
type AdminConversationsEKMListOriginalConnectedChannelInfoResponse struct {
	SlackResponse
	Channels []AdminConversationsEKMOriginalConnectedChannelInfo `json:"channels"`
}

// AdminConversationsEKMListOriginalConnectedChannelInfo lists the original connected channel
// information for Slack Connect channels.
// For more information see the admin.conversations.ekm.listOriginalConnectedChannelInfo docs:
// https://api.slack.com/methods/admin.conversations.ekm.listOriginalConnectedChannelInfo
func (api *Client) AdminConversationsEKMListOriginalConnectedChannelInfo(ctx context.Context, options ...AdminConversationsEKMListOriginalConnectedChannelInfoOption) (*AdminConversationsEKMListOriginalConnectedChannelInfoResponse, error) {
	params := adminConversationsEKMListOriginalConnectedChannelInfoParams{}
	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token": {api.token},
	}

	if len(params.channelIDs) > 0 {
		values.Add("channel_ids", strings.Join(params.channelIDs, ","))
	}

	if len(params.teamIDs) > 0 {
		values.Add("team_ids", strings.Join(params.teamIDs, ","))
	}

	if params.cursor != "" {
		values.Add("cursor", params.cursor)
	}

	if params.limit > 0 {
		values.Add("limit", strconv.Itoa(params.limit))
	}

	response := &AdminConversationsEKMListOriginalConnectedChannelInfoResponse{}
	err := api.postMethod(ctx, "admin.conversations.ekm.listOriginalConnectedChannelInfo", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}
