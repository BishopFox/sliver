package slack

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

// Conversation is the foundation for IM and BaseGroupConversation
type Conversation struct {
	ID                 string   `json:"id"`
	Created            JSONTime `json:"created"`
	IsOpen             bool     `json:"is_open"`
	LastRead           string   `json:"last_read,omitempty"`
	Latest             *Message `json:"latest,omitempty"`
	UnreadCount        int      `json:"unread_count,omitempty"`
	UnreadCountDisplay int      `json:"unread_count_display,omitempty"`
	IsGroup            bool     `json:"is_group"`
	IsShared           bool     `json:"is_shared"`
	IsIM               bool     `json:"is_im"`
	IsExtShared        bool     `json:"is_ext_shared"`
	IsOrgShared        bool     `json:"is_org_shared"`
	IsGlobalShared     bool     `json:"is_global_shared"`
	IsPendingExtShared bool     `json:"is_pending_ext_shared"`
	IsPrivate          bool     `json:"is_private"`
	IsReadOnly         bool     `json:"is_read_only"`
	IsMpIM             bool     `json:"is_mpim"`
	Unlinked           int      `json:"unlinked"`
	NameNormalized     string   `json:"name_normalized"`
	NumMembers         int      `json:"num_members"`
	Priority           float64  `json:"priority"`
	User               string   `json:"user"`
	ConnectedTeamIDs   []string `json:"connected_team_ids,omitempty"`
	SharedTeamIDs      []string `json:"shared_team_ids,omitempty"`
	InternalTeamIDs    []string `json:"internal_team_ids,omitempty"`
	ContextTeamID      string   `json:"context_team_id,omitempty"`
	ConversationHostID string   `json:"conversation_host_id,omitempty"`
	PreviousNames      []string `json:"previous_names,omitempty"`
	PendingShared      []string `json:"pending_shared,omitempty"`
}

// GroupConversation is the foundation for Group and Channel
type GroupConversation struct {
	Conversation
	Name       string   `json:"name"`
	Creator    string   `json:"creator"`
	IsArchived bool     `json:"is_archived"`
	Members    []string `json:"members"`
	Topic      Topic    `json:"topic"`
	Purpose    Purpose  `json:"purpose"`
}

// Topic contains information about the topic
type Topic struct {
	Value   string   `json:"value"`
	Creator string   `json:"creator"`
	LastSet JSONTime `json:"last_set"`
}

// Purpose contains information about the purpose
type Purpose struct {
	Value   string   `json:"value"`
	Creator string   `json:"creator"`
	LastSet JSONTime `json:"last_set"`
}

// Properties contains the Canvas associated to the channel.
type Properties struct {
	Canvas              Canvas       `json:"canvas"`
	PostingRestrictedTo RestrictedTo `json:"posting_restricted_to"`
	Tabs                []Tab        `json:"tabs"`
	ThreadsRestrictedTo RestrictedTo `json:"threads_restricted_to"`
}

type RestrictedTo struct {
	Type []string `json:"type"`
	User []string `json:"user"`
}

type Tab struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

type Canvas struct {
	FileId       string `json:"file_id"`
	IsEmpty      bool   `json:"is_empty"`
	QuipThreadId string `json:"quip_thread_id"`
}

type GetUsersInConversationParameters struct {
	ChannelID string
	Cursor    string
	Limit     int
}

type GetConversationsForUserParameters struct {
	UserID          string
	Cursor          string
	Types           []string
	Limit           int
	ExcludeArchived bool
	TeamID          string
}

type responseMetaData struct {
	NextCursor string `json:"next_cursor"`
}

// GetUsersInConversation returns the list of users in a conversation.
// For more details, see GetUsersInConversationContext documentation.
func (api *Client) GetUsersInConversation(params *GetUsersInConversationParameters) ([]string, string, error) {
	return api.GetUsersInConversationContext(context.Background(), params)
}

// GetUsersInConversationContext returns the list of users in a conversation with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.members
func (api *Client) GetUsersInConversationContext(ctx context.Context, params *GetUsersInConversationParameters) ([]string, string, error) {
	values := url.Values{
		"token":   {api.token},
		"channel": {params.ChannelID},
	}
	if params.Cursor != "" {
		values.Add("cursor", params.Cursor)
	}
	if params.Limit != 0 {
		values.Add("limit", strconv.Itoa(params.Limit))
	}
	response := struct {
		Members          []string         `json:"members"`
		ResponseMetaData responseMetaData `json:"response_metadata"`
		SlackResponse
	}{}

	err := api.postMethod(ctx, "conversations.members", values, &response)
	if err != nil {
		return nil, "", err
	}

	if err := response.Err(); err != nil {
		return nil, "", err
	}

	return response.Members, response.ResponseMetaData.NextCursor, nil
}

// GetConversationsForUser returns the list conversations for a given user.
// For more details, see GetConversationsForUserContext documentation.
func (api *Client) GetConversationsForUser(params *GetConversationsForUserParameters) (channels []Channel, nextCursor string, err error) {
	return api.GetConversationsForUserContext(context.Background(), params)
}

// GetConversationsForUserContext returns the list conversations for a given user with a custom context
// Slack API docs: https://api.slack.com/methods/users.conversations
func (api *Client) GetConversationsForUserContext(ctx context.Context, params *GetConversationsForUserParameters) (channels []Channel, nextCursor string, err error) {
	values := url.Values{
		"token": {api.token},
	}
	if params.UserID != "" {
		values.Add("user", params.UserID)
	}
	if params.Cursor != "" {
		values.Add("cursor", params.Cursor)
	}
	if params.Limit != 0 {
		values.Add("limit", strconv.Itoa(params.Limit))
	}
	if params.Types != nil {
		values.Add("types", strings.Join(params.Types, ","))
	}
	if params.ExcludeArchived {
		values.Add("exclude_archived", "true")
	}
	if params.TeamID != "" {
		values.Add("team_id", params.TeamID)
	}

	response := struct {
		Channels         []Channel        `json:"channels"`
		ResponseMetaData responseMetaData `json:"response_metadata"`
		SlackResponse
	}{}
	err = api.postMethod(ctx, "users.conversations", values, &response)
	if err != nil {
		return nil, "", err
	}

	return response.Channels, response.ResponseMetaData.NextCursor, response.Err()
}

// ArchiveConversation archives a conversation.
// For more details, see ArchiveConversationContext documentation.
func (api *Client) ArchiveConversation(channelID string) error {
	return api.ArchiveConversationContext(context.Background(), channelID)
}

// ArchiveConversationContext archives a conversation with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.archive
func (api *Client) ArchiveConversationContext(ctx context.Context, channelID string) error {
	values := url.Values{
		"token":   {api.token},
		"channel": {channelID},
	}

	response := SlackResponse{}
	err := api.postMethod(ctx, "conversations.archive", values, &response)
	if err != nil {
		return err
	}

	return response.Err()
}

// UnArchiveConversation reverses conversation archival.
// For more details, see UnArchiveConversationContext documentation.
func (api *Client) UnArchiveConversation(channelID string) error {
	return api.UnArchiveConversationContext(context.Background(), channelID)
}

// UnArchiveConversationContext reverses conversation archival with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.unarchive
func (api *Client) UnArchiveConversationContext(ctx context.Context, channelID string) error {
	values := url.Values{
		"token":   {api.token},
		"channel": {channelID},
	}
	response := SlackResponse{}
	err := api.postMethod(ctx, "conversations.unarchive", values, &response)
	if err != nil {
		return err
	}

	return response.Err()
}

// SetTopicOfConversation sets the topic for a conversation.
// For more details, see SetTopicOfConversationContext documentation.
func (api *Client) SetTopicOfConversation(channelID, topic string) (*Channel, error) {
	return api.SetTopicOfConversationContext(context.Background(), channelID, topic)
}

// SetTopicOfConversationContext sets the topic for a conversation with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.setTopic
func (api *Client) SetTopicOfConversationContext(ctx context.Context, channelID, topic string) (*Channel, error) {
	values := url.Values{
		"token":   {api.token},
		"channel": {channelID},
		"topic":   {topic},
	}
	response := struct {
		SlackResponse
		Channel *Channel `json:"channel"`
	}{}
	err := api.postMethod(ctx, "conversations.setTopic", values, &response)
	if err != nil {
		return nil, err
	}

	return response.Channel, response.Err()
}

// SetPurposeOfConversation sets the purpose for a conversation.
// For more details, see SetPurposeOfConversationContext documentation.
func (api *Client) SetPurposeOfConversation(channelID, purpose string) (*Channel, error) {
	return api.SetPurposeOfConversationContext(context.Background(), channelID, purpose)
}

// SetPurposeOfConversationContext sets the purpose for a conversation with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.setPurpose
func (api *Client) SetPurposeOfConversationContext(ctx context.Context, channelID, purpose string) (*Channel, error) {
	values := url.Values{
		"token":   {api.token},
		"channel": {channelID},
		"purpose": {purpose},
	}
	response := struct {
		SlackResponse
		Channel *Channel `json:"channel"`
	}{}

	err := api.postMethod(ctx, "conversations.setPurpose", values, &response)
	if err != nil {
		return nil, err
	}

	return response.Channel, response.Err()
}

// RenameConversation renames a conversation.
// For more details, see RenameConversationContext documentation.
func (api *Client) RenameConversation(channelID, channelName string) (*Channel, error) {
	return api.RenameConversationContext(context.Background(), channelID, channelName)
}

// RenameConversationContext renames a conversation with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.rename
func (api *Client) RenameConversationContext(ctx context.Context, channelID, channelName string) (*Channel, error) {
	values := url.Values{
		"token":   {api.token},
		"channel": {channelID},
		"name":    {channelName},
	}
	response := struct {
		SlackResponse
		Channel *Channel `json:"channel"`
	}{}

	err := api.postMethod(ctx, "conversations.rename", values, &response)
	if err != nil {
		return nil, err
	}

	return response.Channel, response.Err()
}

// InviteUsersToConversation invites users to a channel.
// For more details, see InviteUsersToConversation documentation.
func (api *Client) InviteUsersToConversation(channelID string, users ...string) (*Channel, error) {
	return api.InviteUsersToConversationContext(context.Background(), channelID, users...)
}

// InviteUsersToConversationContext invites users to a channel with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.invite
func (api *Client) InviteUsersToConversationContext(ctx context.Context, channelID string, users ...string) (*Channel, error) {
	values := url.Values{
		"token":   {api.token},
		"channel": {channelID},
		"users":   {strings.Join(users, ",")},
	}
	response := struct {
		SlackResponse
		Channel *Channel `json:"channel"`
	}{}

	err := api.postMethod(ctx, "conversations.invite", values, &response)
	if err != nil {
		return nil, err
	}

	return response.Channel, response.Err()
}

// The following functions are for inviting users to a channel but setting the `force`
// parameter to true. We have added this so that we don't break the existing API.
//
// IMPORTANT: If we ever get here for _another_ parameter, we should consider refactoring
// this to be more flexible.
//
// ForceInviteUsersToConversation invites users to a channel but sets the `force`
// parameter to true.
//
// For more details, see ForceInviteUsersToConversationContext documentation.
func (api *Client) ForceInviteUsersToConversation(channelID string, users ...string) (*Channel, error) {
	return api.ForceInviteUsersToConversationContext(context.Background(), channelID, users...)
}

// ForceInviteUsersToConversationContext invites users to a channel with a custom context
// while setting the `force` argument to true.
//
// Slack API docs: https://api.slack.com/methods/conversations.invite
func (api *Client) ForceInviteUsersToConversationContext(ctx context.Context, channelID string, users ...string) (*Channel, error) {
	values := url.Values{
		"token":   {api.token},
		"channel": {channelID},
		"users":   {strings.Join(users, ",")},
		"force":   {"true"},
	}
	response := struct {
		SlackResponse
		Channel *Channel `json:"channel"`
	}{}

	err := api.postMethod(ctx, "conversations.invite", values, &response)
	if err != nil {
		return nil, err
	}

	return response.Channel, response.Err()
}

// InviteSharedEmailsToConversation invites users to a shared channels by email.
// For more details, see InviteSharedToConversationContext documentation.
func (api *Client) InviteSharedEmailsToConversation(channelID string, emails ...string) (string, bool, error) {
	return api.InviteSharedToConversationContext(context.Background(), InviteSharedToConversationParams{
		ChannelID: channelID,
		Emails:    emails,
	})
}

// InviteSharedEmailsToConversationContext invites users to a shared channels by email using context.
// For more details, see InviteSharedToConversationContext documentation.
func (api *Client) InviteSharedEmailsToConversationContext(ctx context.Context, channelID string, emails ...string) (string, bool, error) {
	return api.InviteSharedToConversationContext(ctx, InviteSharedToConversationParams{
		ChannelID: channelID,
		Emails:    emails,
	})
}

// InviteSharedUserIDsToConversation invites users to a shared channels by user id.
// For more details, see InviteSharedToConversationContext documentation.
func (api *Client) InviteSharedUserIDsToConversation(channelID string, userIDs ...string) (string, bool, error) {
	return api.InviteSharedToConversationContext(context.Background(), InviteSharedToConversationParams{
		ChannelID: channelID,
		UserIDs:   userIDs,
	})
}

// InviteSharedUserIDsToConversationContext invites users to a shared channels by user id with context.
// For more details, see InviteSharedToConversationContext documentation.
func (api *Client) InviteSharedUserIDsToConversationContext(ctx context.Context, channelID string, userIDs ...string) (string, bool, error) {
	return api.InviteSharedToConversationContext(ctx, InviteSharedToConversationParams{
		ChannelID: channelID,
		UserIDs:   userIDs,
	})
}

// InviteSharedToConversationParams defines the parameters for the InviteSharedToConversation and InviteSharedToConversationContext functions.
type InviteSharedToConversationParams struct {
	ChannelID       string
	Emails          []string
	UserIDs         []string
	ExternalLimited *bool
}

// InviteSharedToConversation invites emails or userIDs to a channel.
// For more details, see InviteSharedToConversationContext documentation.
func (api *Client) InviteSharedToConversation(params InviteSharedToConversationParams) (string, bool, error) {
	return api.InviteSharedToConversationContext(context.Background(), params)
}

// InviteSharedToConversationContext invites emails or userIDs to a channel with a custom context.
// This is a helper function for InviteSharedEmailsToConversation and InviteSharedUserIDsToConversation.
// It accepts either emails or userIDs, but not both.
// Slack API docs: https://api.slack.com/methods/conversations.inviteShared
func (api *Client) InviteSharedToConversationContext(ctx context.Context, params InviteSharedToConversationParams) (string, bool, error) {
	values := url.Values{
		"token":   {api.token},
		"channel": {params.ChannelID},
	}
	if len(params.Emails) > 0 {
		values.Add("emails", strings.Join(params.Emails, ","))
	} else if len(params.UserIDs) > 0 {
		values.Add("user_ids", strings.Join(params.UserIDs, ","))
	}
	if params.ExternalLimited != nil {
		values.Add("external_limited", strconv.FormatBool(*params.ExternalLimited))
	}
	response := struct {
		SlackResponse
		InviteID              string `json:"invite_id"`
		IsLegacySharedChannel bool   `json:"is_legacy_shared_channel"`
	}{}

	err := api.postMethod(ctx, "conversations.inviteShared", values, &response)
	if err != nil {
		return "", false, err
	}

	return response.InviteID, response.IsLegacySharedChannel, response.Err()
}

// KickUserFromConversation removes a user from a conversation.
// For more details, see KickUserFromConversationContext documentation.
func (api *Client) KickUserFromConversation(channelID string, user string) error {
	return api.KickUserFromConversationContext(context.Background(), channelID, user)
}

// KickUserFromConversationContext removes a user from a conversation with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.kick
func (api *Client) KickUserFromConversationContext(ctx context.Context, channelID string, user string) error {
	values := url.Values{
		"token":   {api.token},
		"channel": {channelID},
		"user":    {user},
	}

	response := SlackResponse{}
	err := api.postMethod(ctx, "conversations.kick", values, &response)
	if err != nil {
		return err
	}

	return response.Err()
}

// CloseConversation closes a direct message or multi-person direct message.
// For more details, see CloseConversationContext documentation.
func (api *Client) CloseConversation(channelID string) (noOp bool, alreadyClosed bool, err error) {
	return api.CloseConversationContext(context.Background(), channelID)
}

// CloseConversationContext closes a direct message or multi-person direct message with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.close
func (api *Client) CloseConversationContext(ctx context.Context, channelID string) (noOp bool, alreadyClosed bool, err error) {
	values := url.Values{
		"token":   {api.token},
		"channel": {channelID},
	}
	response := struct {
		SlackResponse
		NoOp          bool `json:"no_op"`
		AlreadyClosed bool `json:"already_closed"`
	}{}

	err = api.postMethod(ctx, "conversations.close", values, &response)
	if err != nil {
		return false, false, err
	}

	return response.NoOp, response.AlreadyClosed, response.Err()
}

type CreateConversationParams struct {
	ChannelName string
	IsPrivate   bool
	TeamID      string
}

// CreateConversation initiates a public or private channel-based conversation.
// For more details, see CreateConversationContext documentation.
func (api *Client) CreateConversation(params CreateConversationParams) (*Channel, error) {
	return api.CreateConversationContext(context.Background(), params)
}

// CreateConversationContext initiates a public or private channel-based conversation with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.create
func (api *Client) CreateConversationContext(ctx context.Context, params CreateConversationParams) (*Channel, error) {
	values := url.Values{
		"token":      {api.token},
		"name":       {params.ChannelName},
		"is_private": {strconv.FormatBool(params.IsPrivate)},
	}
	if params.TeamID != "" {
		values.Set("team_id", params.TeamID)
	}
	response, err := api.channelRequest(ctx, "conversations.create", values)
	if err != nil {
		return nil, err
	}

	return &response.Channel, nil
}

// GetConversationInfoInput Defines the parameters of a GetConversationInfo and GetConversationInfoContext function
type GetConversationInfoInput struct {
	ChannelID         string
	IncludeLocale     bool
	IncludeNumMembers bool
}

// GetConversationInfo retrieves information about a conversation.
// For more details, see GetConversationInfoContext documentation.
func (api *Client) GetConversationInfo(input *GetConversationInfoInput) (*Channel, error) {
	return api.GetConversationInfoContext(context.Background(), input)
}

// GetConversationInfoContext retrieves information about a conversation with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.info
func (api *Client) GetConversationInfoContext(ctx context.Context, input *GetConversationInfoInput) (*Channel, error) {
	if input == nil {
		return nil, errors.New("GetConversationInfoInput must not be nil")
	}

	if input.ChannelID == "" {
		return nil, errors.New("ChannelID must be defined")
	}

	values := url.Values{
		"token":               {api.token},
		"channel":             {input.ChannelID},
		"include_locale":      {strconv.FormatBool(input.IncludeLocale)},
		"include_num_members": {strconv.FormatBool(input.IncludeNumMembers)},
	}
	response, err := api.channelRequest(ctx, "conversations.info", values)
	if err != nil {
		return nil, err
	}

	return &response.Channel, response.Err()
}

// LeaveConversation leaves a conversation.
// For more details, see LeaveConversationContext documentation.
func (api *Client) LeaveConversation(channelID string) (bool, error) {
	return api.LeaveConversationContext(context.Background(), channelID)
}

// LeaveConversationContext leaves a conversation with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.leave
func (api *Client) LeaveConversationContext(ctx context.Context, channelID string) (bool, error) {
	values := url.Values{
		"token":   {api.token},
		"channel": {channelID},
	}

	response, err := api.channelRequest(ctx, "conversations.leave", values)
	if err != nil {
		return false, err
	}

	return response.NotInChannel, err
}

type GetConversationRepliesParameters struct {
	ChannelID          string
	Timestamp          string
	Cursor             string
	Inclusive          bool
	Latest             string
	Limit              int
	Oldest             string
	IncludeAllMetadata bool
}

// GetConversationReplies retrieves a thread of messages posted to a conversation.
// For more details, see GetConversationRepliesContext documentation.
func (api *Client) GetConversationReplies(params *GetConversationRepliesParameters) (msgs []Message, hasMore bool, nextCursor string, err error) {
	return api.GetConversationRepliesContext(context.Background(), params)
}

// GetConversationRepliesContext retrieves a thread of messages posted to a conversation with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.replies
func (api *Client) GetConversationRepliesContext(ctx context.Context, params *GetConversationRepliesParameters) (msgs []Message, hasMore bool, nextCursor string, err error) {
	values := url.Values{
		"token":   {api.token},
		"channel": {params.ChannelID},
		"ts":      {params.Timestamp},
	}
	if params.Cursor != "" {
		values.Add("cursor", params.Cursor)
	}
	if params.Latest != "" {
		values.Add("latest", params.Latest)
	}
	if params.Limit != 0 {
		values.Add("limit", strconv.Itoa(params.Limit))
	}
	if params.Oldest != "" {
		values.Add("oldest", params.Oldest)
	}
	if params.Inclusive {
		values.Add("inclusive", "1")
	} else {
		values.Add("inclusive", "0")
	}
	if params.IncludeAllMetadata {
		values.Add("include_all_metadata", "1")
	} else {
		values.Add("include_all_metadata", "0")
	}
	response := struct {
		SlackResponse
		HasMore          bool `json:"has_more"`
		ResponseMetaData struct {
			NextCursor string `json:"next_cursor"`
		} `json:"response_metadata"`
		Messages []Message `json:"messages"`
	}{}

	err = api.postMethod(ctx, "conversations.replies", values, &response)
	if err != nil {
		return nil, false, "", err
	}

	return response.Messages, response.HasMore, response.ResponseMetaData.NextCursor, response.Err()
}

type GetConversationsParameters struct {
	Cursor          string
	ExcludeArchived bool
	Limit           int
	Types           []string
	TeamID          string
}

// GetConversations returns the list of channels in a Slack team.
// For more details, see GetConversationsContext documentation.
func (api *Client) GetConversations(params *GetConversationsParameters) (channels []Channel, nextCursor string, err error) {
	return api.GetConversationsContext(context.Background(), params)
}

// GetConversationsContext returns the list of channels in a Slack team with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.list
func (api *Client) GetConversationsContext(ctx context.Context, params *GetConversationsParameters) (channels []Channel, nextCursor string, err error) {
	values := url.Values{
		"token": {api.token},
	}
	if params.Cursor != "" {
		values.Add("cursor", params.Cursor)
	}
	if params.Limit != 0 {
		values.Add("limit", strconv.Itoa(params.Limit))
	}
	if params.Types != nil {
		values.Add("types", strings.Join(params.Types, ","))
	}
	if params.ExcludeArchived {
		values.Add("exclude_archived", strconv.FormatBool(params.ExcludeArchived))
	}
	if params.TeamID != "" {
		values.Add("team_id", params.TeamID)
	}

	response := struct {
		Channels         []Channel        `json:"channels"`
		ResponseMetaData responseMetaData `json:"response_metadata"`
		SlackResponse
	}{}

	err = api.postMethod(ctx, "conversations.list", values, &response)
	if err != nil {
		return nil, "", err
	}

	return response.Channels, response.ResponseMetaData.NextCursor, response.Err()
}

type OpenConversationParameters struct {
	ChannelID string
	ReturnIM  bool
	Users     []string
}

// OpenConversation opens or resumes a direct message or multi-person direct message.
// For more details, see OpenConversationContext documentation.
func (api *Client) OpenConversation(params *OpenConversationParameters) (*Channel, bool, bool, error) {
	return api.OpenConversationContext(context.Background(), params)
}

// OpenConversationContext opens or resumes a direct message or multi-person direct message with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.open
func (api *Client) OpenConversationContext(ctx context.Context, params *OpenConversationParameters) (*Channel, bool, bool, error) {
	values := url.Values{
		"token":     {api.token},
		"return_im": {strconv.FormatBool(params.ReturnIM)},
	}
	if params.ChannelID != "" {
		values.Add("channel", params.ChannelID)
	}
	if params.Users != nil {
		values.Add("users", strings.Join(params.Users, ","))
	}
	response := struct {
		Channel     *Channel `json:"channel"`
		NoOp        bool     `json:"no_op"`
		AlreadyOpen bool     `json:"already_open"`
		SlackResponse
	}{}

	err := api.postMethod(ctx, "conversations.open", values, &response)
	if err != nil {
		return nil, false, false, err
	}

	return response.Channel, response.NoOp, response.AlreadyOpen, response.Err()
}

// JoinConversation joins an existing conversation.
// For more details, see JoinConversationContext documentation.
func (api *Client) JoinConversation(channelID string) (*Channel, string, []string, error) {
	return api.JoinConversationContext(context.Background(), channelID)
}

// JoinConversationContext joins an existing conversation with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.join
func (api *Client) JoinConversationContext(ctx context.Context, channelID string) (*Channel, string, []string, error) {
	values := url.Values{"token": {api.token}, "channel": {channelID}}
	response := struct {
		Channel          *Channel `json:"channel"`
		Warning          string   `json:"warning"`
		ResponseMetaData *struct {
			Warnings []string `json:"warnings"`
		} `json:"response_metadata"`
		SlackResponse
	}{}

	err := api.postMethod(ctx, "conversations.join", values, &response)
	if err != nil {
		return nil, "", nil, err
	}
	if response.Err() != nil {
		return nil, "", nil, response.Err()
	}
	var warnings []string
	if response.ResponseMetaData != nil {
		warnings = response.ResponseMetaData.Warnings
	}
	return response.Channel, response.Warning, warnings, nil
}

type GetConversationHistoryParameters struct {
	ChannelID          string
	Cursor             string
	Inclusive          bool
	Latest             string
	Limit              int
	Oldest             string
	IncludeAllMetadata bool
}

type GetConversationHistoryResponse struct {
	SlackResponse
	HasMore          bool   `json:"has_more"`
	PinCount         int    `json:"pin_count"`
	Latest           string `json:"latest"`
	ResponseMetaData struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
	Messages []Message `json:"messages"`
}

// GetConversationHistory joins an existing conversation.
// For more details, see GetConversationHistoryContext documentation.
func (api *Client) GetConversationHistory(params *GetConversationHistoryParameters) (*GetConversationHistoryResponse, error) {
	return api.GetConversationHistoryContext(context.Background(), params)
}

// GetConversationHistoryContext joins an existing conversation with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.history
func (api *Client) GetConversationHistoryContext(ctx context.Context, params *GetConversationHistoryParameters) (*GetConversationHistoryResponse, error) {
	values := url.Values{"token": {api.token}, "channel": {params.ChannelID}}
	if params.Cursor != "" {
		values.Add("cursor", params.Cursor)
	}
	if params.Inclusive {
		values.Add("inclusive", "1")
	} else {
		values.Add("inclusive", "0")
	}
	if params.Latest != "" {
		values.Add("latest", params.Latest)
	}
	if params.Limit != 0 {
		values.Add("limit", strconv.Itoa(params.Limit))
	}
	if params.Oldest != "" {
		values.Add("oldest", params.Oldest)
	}
	if params.IncludeAllMetadata {
		values.Add("include_all_metadata", "1")
	} else {
		values.Add("include_all_metadata", "0")
	}

	response := GetConversationHistoryResponse{}

	err := api.postMethod(ctx, "conversations.history", values, &response)
	if err != nil {
		return nil, err
	}

	return &response, response.Err()
}

// MarkConversation sets the read mark of a conversation to a specific point.
// For more details, see MarkConversationContext documentation.
func (api *Client) MarkConversation(channel, ts string) (err error) {
	return api.MarkConversationContext(context.Background(), channel, ts)
}

// MarkConversationContext sets the read mark of a conversation to a specific point with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.mark
func (api *Client) MarkConversationContext(ctx context.Context, channel, ts string) error {
	values := url.Values{
		"token":   {api.token},
		"channel": {channel},
		"ts":      {ts},
	}

	response := &SlackResponse{}

	err := api.postMethod(ctx, "conversations.mark", values, response)
	if err != nil {
		return err
	}
	return response.Err()
}

// CreateChannelCanvas creates a new canvas in a channel.
// For more details, see CreateChannelCanvasContext documentation.
func (api *Client) CreateChannelCanvas(channel string, documentContent DocumentContent) (string, error) {
	return api.CreateChannelCanvasContext(context.Background(), channel, documentContent)
}

// CreateChannelCanvasContext creates a new canvas in a channel with a custom context.
// Slack API docs: https://api.slack.com/methods/conversations.canvases.create
func (api *Client) CreateChannelCanvasContext(ctx context.Context, channel string, documentContent DocumentContent) (string, error) {
	values := url.Values{
		"token":      {api.token},
		"channel_id": {channel},
	}
	if documentContent.Type != "" {
		documentContentJSON, err := json.Marshal(documentContent)
		if err != nil {
			return "", err
		}
		values.Add("document_content", string(documentContentJSON))
	}

	response := struct {
		SlackResponse
		CanvasID string `json:"canvas_id"`
	}{}
	err := api.postMethod(ctx, "conversations.canvases.create", values, &response)
	if err != nil {
		return "", err
	}

	return response.CanvasID, response.Err()
}
