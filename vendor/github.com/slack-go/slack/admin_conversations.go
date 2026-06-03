package slack

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

// AdminConversationsInviteParams contains arguments for AdminConversationsInvite method call.
type AdminConversationsInviteParams struct {
	ChannelID string
	UserIDs   []string
}

// AdminConversationsInvite invites users to a channel.
// For more information see the admin.conversations.invite docs:
// https://api.slack.com/methods/admin.conversations.invite
func (api *Client) AdminConversationsInvite(ctx context.Context, params AdminConversationsInviteParams) error {
	values := url.Values{
		"token":      {api.token},
		"channel_id": {params.ChannelID},
		"user_ids":   {strings.Join(params.UserIDs, ",")},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.conversations.invite", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}

// AdminConversationsArchive archives a public or private channel.
// For more information see the admin.conversations.archive docs:
// https://api.slack.com/methods/admin.conversations.archive
func (api *Client) AdminConversationsArchive(ctx context.Context, channelID string) error {
	values := url.Values{
		"token":      {api.token},
		"channel_id": {channelID},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.conversations.archive", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}

// AdminConversationsUnarchive unarchives a public or private channel.
// For more information see the admin.conversations.unarchive docs:
// https://api.slack.com/methods/admin.conversations.unarchive
func (api *Client) AdminConversationsUnarchive(ctx context.Context, channelID string) error {
	values := url.Values{
		"token":      {api.token},
		"channel_id": {channelID},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.conversations.unarchive", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}

// AdminConversationsRename renames a public or private channel.
// For more information see the admin.conversations.rename docs:
// https://api.slack.com/methods/admin.conversations.rename
func (api *Client) AdminConversationsRename(ctx context.Context, channelID, name string) error {
	values := url.Values{
		"token":      {api.token},
		"channel_id": {channelID},
		"name":       {name},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.conversations.rename", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}

// AdminConversationsDelete deletes a public or private channel.
// For more information see the admin.conversations.delete docs:
// https://api.slack.com/methods/admin.conversations.delete
func (api *Client) AdminConversationsDelete(ctx context.Context, channelID string) error {
	values := url.Values{
		"token":      {api.token},
		"channel_id": {channelID},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.conversations.delete", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}

type adminConversationsDisconnectSharedParams struct {
	leavingTeamIDs []string
}

// AdminConversationsDisconnectSharedOption is an option for AdminConversationsDisconnectShared.
type AdminConversationsDisconnectSharedOption func(*adminConversationsDisconnectSharedParams)

// AdminConversationsDisconnectSharedOptionLeavingTeamIDs sets the team IDs of the workspaces to disconnect.
func AdminConversationsDisconnectSharedOptionLeavingTeamIDs(teamIDs []string) AdminConversationsDisconnectSharedOption {
	return func(params *adminConversationsDisconnectSharedParams) {
		params.leavingTeamIDs = teamIDs
	}
}

// AdminConversationsDisconnectShared disconnects a connected channel from one or more workspaces.
// For more information see the admin.conversations.disconnectShared docs:
// https://api.slack.com/methods/admin.conversations.disconnectShared
func (api *Client) AdminConversationsDisconnectShared(ctx context.Context, channelID string, options ...AdminConversationsDisconnectSharedOption) error {
	params := adminConversationsDisconnectSharedParams{}
	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token":      {api.token},
		"channel_id": {channelID},
	}

	if len(params.leavingTeamIDs) > 0 {
		values.Add("leaving_team_ids", strings.Join(params.leavingTeamIDs, ","))
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.conversations.disconnectShared", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}

type adminConversationsCreateParams struct {
	description string
	orgWide     bool
	teamID      string
}

// AdminConversationsCreateOption is an option for AdminConversationsCreate.
type AdminConversationsCreateOption func(*adminConversationsCreateParams)

// AdminConversationsCreateOptionDescription sets the description of the channel.
func AdminConversationsCreateOptionDescription(description string) AdminConversationsCreateOption {
	return func(params *adminConversationsCreateParams) {
		params.description = description
	}
}

// AdminConversationsCreateOptionOrgWide sets whether the channel should be org-wide.
func AdminConversationsCreateOptionOrgWide(orgWide bool) AdminConversationsCreateOption {
	return func(params *adminConversationsCreateParams) {
		params.orgWide = orgWide
	}
}

// AdminConversationsCreateOptionTeamID sets the team ID where the channel should be created.
func AdminConversationsCreateOptionTeamID(teamID string) AdminConversationsCreateOption {
	return func(params *adminConversationsCreateParams) {
		params.teamID = teamID
	}
}

// AdminConversationsCreateResponse represents the response from admin.conversations.create.
type AdminConversationsCreateResponse struct {
	SlackResponse
	ChannelID string `json:"channel_id"`
}

// AdminConversationsCreate creates a public or private channel-based conversation.
// For more information see the admin.conversations.create docs:
// https://api.slack.com/methods/admin.conversations.create
func (api *Client) AdminConversationsCreate(ctx context.Context, name string, isPrivate bool, options ...AdminConversationsCreateOption) (string, error) {
	params := adminConversationsCreateParams{}
	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token":      {api.token},
		"is_private": {strconv.FormatBool(isPrivate)},
		"name":       {name},
	}

	if params.description != "" {
		values.Add("description", params.description)
	}

	if params.orgWide {
		values.Add("org_wide", "true")
	}

	if params.teamID != "" {
		values.Add("team_id", params.teamID)
	}

	response := &AdminConversationsCreateResponse{}
	err := api.postMethod(ctx, "admin.conversations.create", values, response)
	if err != nil {
		return "", err
	}

	return response.ChannelID, response.Err()
}

// AdminConversationsGetTeamsParams contains arguments for AdminConversationsGetTeams method call.
type AdminConversationsGetTeamsParams struct {
	ChannelID string
	Cursor    string
	Limit     int
}

// AdminConversationsGetTeamsResponse represents the response from admin.conversations.getTeams.
type AdminConversationsGetTeamsResponse struct {
	SlackResponse
	TeamIDs []string `json:"team_ids"`
}

// AdminConversationsGetTeams gets all the workspaces a given public or private channel is connected to within this Enterprise org.
// For more information see the admin.conversations.getTeams docs:
// https://api.slack.com/methods/admin.conversations.getTeams
func (api *Client) AdminConversationsGetTeams(ctx context.Context, params AdminConversationsGetTeamsParams) ([]string, string, error) {
	values := url.Values{
		"token":      {api.token},
		"channel_id": {params.ChannelID},
	}

	if params.Cursor != "" {
		values.Add("cursor", params.Cursor)
	}

	if params.Limit > 0 {
		values.Add("limit", strconv.Itoa(params.Limit))
	}

	response := &AdminConversationsGetTeamsResponse{}
	err := api.postMethod(ctx, "admin.conversations.getTeams", values, response)
	if err != nil {
		return nil, "", err
	}

	return response.TeamIDs, response.ResponseMetadata.Cursor, response.Err()
}

type adminConversationsSearchParams struct {
	cursor            string
	limit             int
	query             string
	searchChannelType []string
	sort              string
	sortDir           string
	teamIDs           []string
	connectedTeamIDs  []string
	totalCountOnly    bool
}

// AdminConversationsSearchOption is an option for AdminConversationsSearch.
type AdminConversationsSearchOption func(*adminConversationsSearchParams)

// AdminConversationsSearchOptionCursor sets the cursor for pagination.
func AdminConversationsSearchOptionCursor(cursor string) AdminConversationsSearchOption {
	return func(params *adminConversationsSearchParams) {
		params.cursor = cursor
	}
}

// AdminConversationsSearchOptionLimit sets the maximum number of results to return.
func AdminConversationsSearchOptionLimit(limit int) AdminConversationsSearchOption {
	return func(params *adminConversationsSearchParams) {
		params.limit = limit
	}
}

// AdminConversationsSearchOptionQuery sets the search query.
func AdminConversationsSearchOptionQuery(query string) AdminConversationsSearchOption {
	return func(params *adminConversationsSearchParams) {
		params.query = query
	}
}

// AdminConversationsSearchOptionSearchChannelTypes sets the channel types to search.
// Valid values: "private", "public", "private_exclude", "multi_workspace", "org_wide", "external_shared_exclude", "external_shared"
func AdminConversationsSearchOptionSearchChannelTypes(types []string) AdminConversationsSearchOption {
	return func(params *adminConversationsSearchParams) {
		params.searchChannelType = types
	}
}

// AdminConversationsSearchOptionSort sets the sort field.
// Valid values: "name", "member_count", "created"
func AdminConversationsSearchOptionSort(sort string) AdminConversationsSearchOption {
	return func(params *adminConversationsSearchParams) {
		params.sort = sort
	}
}

// AdminConversationsSearchOptionSortDir sets the sort direction.
// Valid values: "asc", "desc"
func AdminConversationsSearchOptionSortDir(sortDir string) AdminConversationsSearchOption {
	return func(params *adminConversationsSearchParams) {
		params.sortDir = sortDir
	}
}

// AdminConversationsSearchOptionTeamIDs filters results to channels in the specified teams.
func AdminConversationsSearchOptionTeamIDs(teamIDs []string) AdminConversationsSearchOption {
	return func(params *adminConversationsSearchParams) {
		params.teamIDs = teamIDs
	}
}

// AdminConversationsSearchOptionConnectedTeamIDs filters results to channels connected to the specified teams.
func AdminConversationsSearchOptionConnectedTeamIDs(teamIDs []string) AdminConversationsSearchOption {
	return func(params *adminConversationsSearchParams) {
		params.connectedTeamIDs = teamIDs
	}
}

// AdminConversationsSearchOptionTotalCountOnly when true, only returns the total count of matching channels.
func AdminConversationsSearchOptionTotalCountOnly(totalCountOnly bool) AdminConversationsSearchOption {
	return func(params *adminConversationsSearchParams) {
		params.totalCountOnly = totalCountOnly
	}
}

// ChannelEmailAddress represents an email address associated with a channel.
type ChannelEmailAddress struct {
	Address   string `json:"address"`
	CreatorID string `json:"creator_id"`
	TeamID    string `json:"team_id"`
}

// AdminConversationOwnershipDetail represents ownership details for lists/canvas.
type AdminConversationOwnershipDetail struct {
	Count  int    `json:"count,omitempty"`
	TeamID string `json:"team_id,omitempty"`
}

// AdminConversationLists represents lists/canvas information in admin conversations.
type AdminConversationLists struct {
	OwnershipDetails []AdminConversationOwnershipDetail `json:"ownership_details,omitempty"`
	TotalCount       int                                `json:"total_count,omitempty"`
}

// AdminConversation represents a conversation in admin API responses.
type AdminConversation struct {
	ID                        string                  `json:"id,omitempty"`
	Name                      string                  `json:"name,omitempty"`
	Purpose                   string                  `json:"purpose,omitempty"`
	MemberCount               int                     `json:"member_count,omitempty"`
	Created                   int64                   `json:"created,omitempty"`
	CreatorID                 string                  `json:"creator_id,omitempty"`
	IsPrivate                 bool                    `json:"is_private,omitempty"`
	IsArchived                bool                    `json:"is_archived,omitempty"`
	IsGeneral                 bool                    `json:"is_general,omitempty"`
	LastActivityTimestamp     int64                   `json:"last_activity_ts,omitempty"`
	IsFrozen                  bool                    `json:"is_frozen,omitempty"`
	IsOrgDefault              bool                    `json:"is_org_default,omitempty"`
	IsOrgMandatory            bool                    `json:"is_org_mandatory,omitempty"`
	IsOrgShared               bool                    `json:"is_org_shared,omitempty"`
	IsExtShared               bool                    `json:"is_ext_shared,omitempty"`
	IsGlobalShared            bool                    `json:"is_global_shared,omitempty"`
	IsPendingExtShared        bool                    `json:"is_pending_ext_shared,omitempty"`
	IsDisconnectInProgress    bool                    `json:"is_disconnect_in_progress,omitempty"`
	ConnectedTeamIDs          []string                `json:"connected_team_ids,omitempty"`
	ConnectedLimitedTeamIDs   []string                `json:"connected_limited_team_ids,omitempty"`
	PendingConnectedTeamIDs   []string                `json:"pending_connected_team_ids,omitempty"`
	InternalTeamIDs           []string                `json:"internal_team_ids,omitempty"`
	InternalTeamIDsCount      int                     `json:"internal_team_ids_count,omitempty"`
	InternalTeamIDsSampleTeam string                  `json:"internal_team_ids_sample_team,omitempty"`
	ContextTeamID             string                  `json:"context_team_id,omitempty"`
	ConversationHostID        string                  `json:"conversation_host_id,omitempty"`
	ChannelEmailAddresses     []ChannelEmailAddress   `json:"channel_email_addresses,omitempty"`
	ChannelManagerCount       int                     `json:"channel_manager_count,omitempty"`
	ExternalUserCount         int                     `json:"external_user_count,omitempty"`
	Canvas                    *AdminConversationLists `json:"canvas,omitempty"`
	Lists                     *AdminConversationLists `json:"lists,omitempty"`
	Properties                *Properties             `json:"properties,omitempty"`
}

// AdminConversationsSearchResponse represents the response from admin.conversations.search.
type AdminConversationsSearchResponse struct {
	SlackResponse
	Conversations []AdminConversation `json:"conversations"`
	TotalCount    int                 `json:"total_count"`
	NextCursor    string              `json:"next_cursor"`
}

// AdminConversationsSearch searches for public or private channels in an Enterprise organization.
// For more information see the admin.conversations.search docs:
// https://api.slack.com/methods/admin.conversations.search
func (api *Client) AdminConversationsSearch(ctx context.Context, options ...AdminConversationsSearchOption) (*AdminConversationsSearchResponse, error) {
	params := adminConversationsSearchParams{}
	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token": {api.token},
	}

	if params.cursor != "" {
		values.Add("cursor", params.cursor)
	}

	if params.limit > 0 {
		values.Add("limit", strconv.Itoa(params.limit))
	}

	if params.query != "" {
		values.Add("query", params.query)
	}

	if len(params.searchChannelType) > 0 {
		values.Add("search_channel_types", strings.Join(params.searchChannelType, ","))
	}

	if params.sort != "" {
		values.Add("sort", params.sort)
	}

	if params.sortDir != "" {
		values.Add("sort_dir", params.sortDir)
	}

	if len(params.teamIDs) > 0 {
		values.Add("team_ids", strings.Join(params.teamIDs, ","))
	}

	if len(params.connectedTeamIDs) > 0 {
		values.Add("connected_team_ids", strings.Join(params.connectedTeamIDs, ","))
	}

	if params.totalCountOnly {
		values.Add("total_count_only", "true")
	}

	response := &AdminConversationsSearchResponse{}
	err := api.postMethod(ctx, "admin.conversations.search", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

type adminConversationsLookupParams struct {
	cursor         string
	limit          int
	maxMemberCount int
}

// AdminConversationsLookupOption is an option for AdminConversationsLookup.
type AdminConversationsLookupOption func(*adminConversationsLookupParams)

// AdminConversationsLookupOptionCursor sets the cursor for pagination.
func AdminConversationsLookupOptionCursor(cursor string) AdminConversationsLookupOption {
	return func(params *adminConversationsLookupParams) {
		params.cursor = cursor
	}
}

// AdminConversationsLookupOptionLimit sets the maximum number of results to return.
func AdminConversationsLookupOptionLimit(limit int) AdminConversationsLookupOption {
	return func(params *adminConversationsLookupParams) {
		params.limit = limit
	}
}

// AdminConversationsLookupOptionMaxMemberCount filters to channels with at most this many members.
func AdminConversationsLookupOptionMaxMemberCount(maxMemberCount int) AdminConversationsLookupOption {
	return func(params *adminConversationsLookupParams) {
		params.maxMemberCount = maxMemberCount
	}
}

// AdminConversationsLookupResponse represents the response from admin.conversations.lookup.
type AdminConversationsLookupResponse struct {
	SlackResponse
	Channels []string `json:"channels"`
}

// AdminConversationsLookup returns channels on the given team matching the specified filters.
// For more information see the admin.conversations.lookup docs:
// https://api.slack.com/methods/admin.conversations.lookup
func (api *Client) AdminConversationsLookup(ctx context.Context, teamIDs []string, lastMessageActivityBefore int64, options ...AdminConversationsLookupOption) ([]string, string, error) {
	params := adminConversationsLookupParams{}
	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token":                        {api.token},
		"last_message_activity_before": {strconv.FormatInt(lastMessageActivityBefore, 10)},
		"team_ids":                     {strings.Join(teamIDs, ",")},
	}

	if params.cursor != "" {
		values.Add("cursor", params.cursor)
	}

	if params.limit > 0 {
		values.Add("limit", strconv.Itoa(params.limit))
	}

	if params.maxMemberCount > 0 {
		values.Add("max_member_count", strconv.Itoa(params.maxMemberCount))
	}

	response := &AdminConversationsLookupResponse{}
	err := api.postMethod(ctx, "admin.conversations.lookup", values, response)
	if err != nil {
		return nil, "", err
	}

	return response.Channels, response.ResponseMetadata.Cursor, response.Err()
}

// AdminConversationsBulkArchive archives public or private channels in bulk.
// For more information see the admin.conversations.bulkArchive docs:
// https://api.slack.com/methods/admin.conversations.bulkArchive
func (api *Client) AdminConversationsBulkArchive(ctx context.Context, channelIDs []string) error {
	values := url.Values{
		"token":       {api.token},
		"channel_ids": {strings.Join(channelIDs, ",")},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.conversations.bulkArchive", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}

// AdminConversationsBulkDelete deletes public or private channels in bulk.
// For more information see the admin.conversations.bulkDelete docs:
// https://api.slack.com/methods/admin.conversations.bulkDelete
func (api *Client) AdminConversationsBulkDelete(ctx context.Context, channelIDs []string) error {
	values := url.Values{
		"token":       {api.token},
		"channel_ids": {strings.Join(channelIDs, ",")},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.conversations.bulkDelete", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}

// AdminConversationsBulkMoveParams contains arguments for AdminConversationsBulkMove method call.
type AdminConversationsBulkMoveParams struct {
	ChannelIDs   []string
	TargetTeamID string
}

// AdminConversationsBulkMove moves public or private channels in bulk.
// For more information see the admin.conversations.bulkMove docs:
// https://api.slack.com/methods/admin.conversations.bulkMove
func (api *Client) AdminConversationsBulkMove(ctx context.Context, params AdminConversationsBulkMoveParams) error {
	values := url.Values{
		"token":          {api.token},
		"channel_ids":    {strings.Join(params.ChannelIDs, ",")},
		"target_team_id": {params.TargetTeamID},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.conversations.bulkMove", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}

// AdminConversationPrefs represents conversation preferences.
type AdminConversationPrefs struct {
	WhoCanPost      *AdminConversationPref        `json:"who_can_post,omitempty"`
	CanThread       *AdminConversationPref        `json:"can_thread,omitempty"`
	CanHuddle       *AdminConversationPref        `json:"can_huddle,omitempty"`
	EnableAtHere    *AdminConversationPrefEnabled `json:"enable_at_here,omitempty"`
	EnableAtChannel *AdminConversationPrefEnabled `json:"enable_at_channel,omitempty"`
}

// AdminConversationPrefEnabled represents an enabled/disabled preference.
type AdminConversationPrefEnabled struct {
	Enabled bool `json:"enabled"`
}

// AdminConversationPref represents a single conversation preference.
type AdminConversationPref struct {
	Type []string `json:"type,omitempty"`
	User []string `json:"user,omitempty"`
}

// AdminConversationsGetConversationPrefsResponse represents the response from admin.conversations.getConversationPrefs.
type AdminConversationsGetConversationPrefsResponse struct {
	SlackResponse
	Prefs AdminConversationPrefs `json:"prefs"`
}

// AdminConversationsGetConversationPrefs gets conversation preferences for a public or private channel.
// For more information see the admin.conversations.getConversationPrefs docs:
// https://api.slack.com/methods/admin.conversations.getConversationPrefs
func (api *Client) AdminConversationsGetConversationPrefs(ctx context.Context, channelID string) (*AdminConversationPrefs, error) {
	values := url.Values{
		"token":      {api.token},
		"channel_id": {channelID},
	}

	response := &AdminConversationsGetConversationPrefsResponse{}
	err := api.postMethod(ctx, "admin.conversations.getConversationPrefs", values, response)
	if err != nil {
		return nil, err
	}

	return &response.Prefs, response.Err()
}

// AdminConversationsSetConversationPrefsParams contains arguments for AdminConversationsSetConversationPrefs method call.
type AdminConversationsSetConversationPrefsParams struct {
	ChannelID string
	Prefs     AdminConversationPrefs
}

// AdminConversationsSetConversationPrefs sets conversation preferences for a public or private channel.
// For more information see the admin.conversations.setConversationPrefs docs:
// https://api.slack.com/methods/admin.conversations.setConversationPrefs
func (api *Client) AdminConversationsSetConversationPrefs(ctx context.Context, params AdminConversationsSetConversationPrefsParams) error {
	prefsJSON, err := json.Marshal(params.Prefs)
	if err != nil {
		return err
	}

	values := url.Values{
		"token":      {api.token},
		"channel_id": {params.ChannelID},
		"prefs":      {string(prefsJSON)},
	}

	response := &SlackResponse{}
	err = api.postMethod(ctx, "admin.conversations.setConversationPrefs", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}

// AdminConversationsGetCustomRetentionResponse represents the response from admin.conversations.getCustomRetention.
type AdminConversationsGetCustomRetentionResponse struct {
	SlackResponse
	DurationDays    int  `json:"duration_days"`
	IsPolicyEnabled bool `json:"is_policy_enabled"`
}

// AdminConversationsGetCustomRetention gets a conversation's custom retention policy.
// For more information see the admin.conversations.getCustomRetention docs:
// https://api.slack.com/methods/admin.conversations.getCustomRetention
func (api *Client) AdminConversationsGetCustomRetention(ctx context.Context, channelID string) (*AdminConversationsGetCustomRetentionResponse, error) {
	values := url.Values{
		"token":      {api.token},
		"channel_id": {channelID},
	}

	response := &AdminConversationsGetCustomRetentionResponse{}
	err := api.postMethod(ctx, "admin.conversations.getCustomRetention", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// AdminConversationsSetCustomRetention sets a conversation's custom retention policy.
// For more information see the admin.conversations.setCustomRetention docs:
// https://api.slack.com/methods/admin.conversations.setCustomRetention
func (api *Client) AdminConversationsSetCustomRetention(ctx context.Context, channelID string, durationDays int) error {
	values := url.Values{
		"token":         {api.token},
		"channel_id":    {channelID},
		"duration_days": {strconv.Itoa(durationDays)},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.conversations.setCustomRetention", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}

// AdminConversationsRemoveCustomRetention removes a conversation's custom retention policy.
// For more information see the admin.conversations.removeCustomRetention docs:
// https://api.slack.com/methods/admin.conversations.removeCustomRetention
func (api *Client) AdminConversationsRemoveCustomRetention(ctx context.Context, channelID string) error {
	values := url.Values{
		"token":      {api.token},
		"channel_id": {channelID},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.conversations.removeCustomRetention", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}

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
