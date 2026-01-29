package slack

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// UserGroup contains all the information of a user group
type UserGroup struct {
	ID          string         `json:"id"`
	TeamID      string         `json:"team_id"`
	IsUserGroup bool           `json:"is_usergroup"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Handle      string         `json:"handle"`
	IsExternal  bool           `json:"is_external"`
	DateCreate  JSONTime       `json:"date_create"`
	DateUpdate  JSONTime       `json:"date_update"`
	DateDelete  JSONTime       `json:"date_delete"`
	AutoType    string         `json:"auto_type"`
	CreatedBy   string         `json:"created_by"`
	UpdatedBy   string         `json:"updated_by"`
	DeletedBy   string         `json:"deleted_by"`
	Prefs       UserGroupPrefs `json:"prefs"`
	UserCount   int            `json:"user_count"`
	Users       []string       `json:"users"`
}

// UserGroupPrefs contains default channels and groups (private channels)
type UserGroupPrefs struct {
	Channels []string `json:"channels"`
	Groups   []string `json:"groups"`
}

type userGroupResponseFull struct {
	UserGroups []UserGroup `json:"usergroups"`
	UserGroup  UserGroup   `json:"usergroup"`
	Users      []string    `json:"users"`
	SlackResponse
}

func (api *Client) userGroupRequest(ctx context.Context, path string, values url.Values) (*userGroupResponseFull, error) {
	response := &userGroupResponseFull{}
	err := api.postMethod(ctx, path, values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// createUserGroupParams contains arguments for CreateUserGroup method call
type createUserGroupParams struct {
	enableSection bool
	includeCount  bool
}

// CreateUserGroupOption options for the CreateUserGroup method call.
type CreateUserGroupOption func(*createUserGroupParams)

// CreateUserGroupOptionEnableSection enable the section for the user group (default: false)
func CreateUserGroupOptionEnableSection(enableSection bool) CreateUserGroupOption {
	return func(params *createUserGroupParams) {
		params.enableSection = enableSection
	}
}

// CreateUserGroupOptionIncludeCount include the number of users in each User Group
func CreateUserGroupOptionIncludeCount(includeCount bool) CreateUserGroupOption {
	return func(params *createUserGroupParams) {
		params.includeCount = includeCount
	}
}

// CreateUserGroup creates a new user group.
// For more information see the CreateUserGroupContext documentation.
func (api *Client) CreateUserGroup(userGroup UserGroup, options ...CreateUserGroupOption) (UserGroup, error) {
	return api.CreateUserGroupContext(context.Background(), userGroup, options...)
}

// CreateUserGroupContext creates a new user group with a custom context.
// Slack API docs: https://api.slack.com/methods/usergroups.create
func (api *Client) CreateUserGroupContext(ctx context.Context, userGroup UserGroup, options ...CreateUserGroupOption) (UserGroup, error) {
	params := createUserGroupParams{}

	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token": {api.token},
		"name":  {userGroup.Name},
	}

	if params.enableSection {
		values["enable_section"] = []string{strconv.FormatBool(params.enableSection)}
	}

	if params.includeCount {
		values["include_count"] = []string{strconv.FormatBool(params.includeCount)}
	}

	if userGroup.TeamID != "" {
		values["team_id"] = []string{userGroup.TeamID}
	}

	if userGroup.Handle != "" {
		values["handle"] = []string{userGroup.Handle}
	}

	if userGroup.Description != "" {
		values["description"] = []string{userGroup.Description}
	}

	if len(userGroup.Prefs.Channels) > 0 {
		values["channels"] = []string{strings.Join(userGroup.Prefs.Channels, ",")}
	}

	response, err := api.userGroupRequest(ctx, "usergroups.create", values)
	if err != nil {
		return UserGroup{}, err
	}
	return response.UserGroup, nil
}

// DisableUserGroupParams contains arguments for DisableUserGroup method calls.
type DisableUserGroupParams struct {
	IncludeCount bool
	TeamID       string
}

// DisableUserGroupOption options for the DisableUserGroup method calls.
type DisableUserGroupOption func(*DisableUserGroupParams)

// DisableUserGroupOptionIncludeCount include the count of User Groups (default: false)
func DisableUserGroupOptionIncludeCount(b bool) DisableUserGroupOption {
	return func(params *DisableUserGroupParams) {
		params.IncludeCount = b
	}
}

// DisableUserGroupOptionTeamID include team Id
func DisableUserGroupOptionTeamID(teamID string) DisableUserGroupOption {
	return func(params *DisableUserGroupParams) {
		params.TeamID = teamID
	}
}

// DisableUserGroup disables an existing user group.
// For more information see the DisableUserGroupContext documentation.
func (api *Client) DisableUserGroup(userGroup string, options ...DisableUserGroupOption) (UserGroup, error) {
	return api.DisableUserGroupContext(context.Background(), userGroup, options...)
}

// DisableUserGroupContext disables an existing user group with a custom context.
// Slack API docs: https://api.slack.com/methods/usergroups.disable
func (api *Client) DisableUserGroupContext(ctx context.Context, userGroup string, options ...DisableUserGroupOption) (UserGroup, error) {
	params := DisableUserGroupParams{}

	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token":     {api.token},
		"usergroup": {userGroup},
	}

	if params.IncludeCount {
		values.Add("include_count", "true")
	}

	if params.TeamID != "" {
		values.Add("team_id", params.TeamID)
	}

	response, err := api.userGroupRequest(ctx, "usergroups.disable", values)
	if err != nil {
		return UserGroup{}, err
	}
	return response.UserGroup, nil
}

// EnableUserGroupParams contains arguments for EnableUserGroup method calls.
type EnableUserGroupParams struct {
	IncludeCount bool
	TeamID       string
}

// EnableUserGroupOption options for the EnableUserGroup method calls.
type EnableUserGroupOption func(*EnableUserGroupParams)

// EnableUserGroupOptionIncludeCount include the count of User Groups (default: false)
func EnableUserGroupOptionIncludeCount(b bool) EnableUserGroupOption {
	return func(params *EnableUserGroupParams) {
		params.IncludeCount = b
	}
}

// EnableUserGroupOptionTeamID include team Id
func EnableUserGroupOptionTeamID(teamID string) EnableUserGroupOption {
	return func(params *EnableUserGroupParams) {
		params.TeamID = teamID
	}
}

// EnableUserGroup enables an existing user group.
// For more information see the EnableUserGroupContext documentation.
func (api *Client) EnableUserGroup(userGroup string, options ...EnableUserGroupOption) (UserGroup, error) {
	return api.EnableUserGroupContext(context.Background(), userGroup, options...)
}

// EnableUserGroupContext enables an existing user group with a custom context.
// Slack API docs: https://api.slack.com/methods/usergroups.enable
func (api *Client) EnableUserGroupContext(ctx context.Context, userGroup string, options ...EnableUserGroupOption) (UserGroup, error) {
	params := EnableUserGroupParams{}

	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token":     {api.token},
		"usergroup": {userGroup},
	}

	if params.IncludeCount {
		values.Add("include_count", "true")
	}

	if params.TeamID != "" {
		values.Add("team_id", params.TeamID)
	}

	response, err := api.userGroupRequest(ctx, "usergroups.enable", values)
	if err != nil {
		return UserGroup{}, err
	}
	return response.UserGroup, nil
}

// GetUserGroupsOption options for the GetUserGroups method call.
type GetUserGroupsOption func(*GetUserGroupsParams)

// Deprecated: GetUserGroupsOptionWithTeamID is deprecated, use GetUserGroupsOptionTeamID instead
func GetUserGroupsOptionWithTeamID(teamID string) GetUserGroupsOption {
	return GetUserGroupsOptionTeamID(teamID)
}

func GetUserGroupsOptionTeamID(teamID string) GetUserGroupsOption {
	return func(params *GetUserGroupsParams) {
		params.TeamID = teamID
	}
}

// GetUserGroupsOptionIncludeCount include the number of users in each User Group (default: false)
func GetUserGroupsOptionIncludeCount(b bool) GetUserGroupsOption {
	return func(params *GetUserGroupsParams) {
		params.IncludeCount = b
	}
}

// GetUserGroupsOptionIncludeDisabled include disabled User Groups (default: false)
func GetUserGroupsOptionIncludeDisabled(b bool) GetUserGroupsOption {
	return func(params *GetUserGroupsParams) {
		params.IncludeDisabled = b
	}
}

// GetUserGroupsOptionIncludeUsers include the list of users for each User Group (default: false)
func GetUserGroupsOptionIncludeUsers(b bool) GetUserGroupsOption {
	return func(params *GetUserGroupsParams) {
		params.IncludeUsers = b
	}
}

// GetUserGroupsParams contains arguments for GetUserGroups method call
type GetUserGroupsParams struct {
	TeamID          string
	IncludeCount    bool
	IncludeDisabled bool
	IncludeUsers    bool
}

// GetUserGroups returns a list of user groups for the team.
// For more information see the GetUserGroupsContext documentation.
func (api *Client) GetUserGroups(options ...GetUserGroupsOption) ([]UserGroup, error) {
	return api.GetUserGroupsContext(context.Background(), options...)
}

// GetUserGroupsContext returns a list of user groups for the team with a custom context.
// Slack API docs: https://api.slack.com/methods/usergroups.list
func (api *Client) GetUserGroupsContext(ctx context.Context, options ...GetUserGroupsOption) ([]UserGroup, error) {
	params := GetUserGroupsParams{}

	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token": {api.token},
	}
	if params.TeamID != "" {
		values.Add("team_id", params.TeamID)
	}
	if params.IncludeCount {
		values.Add("include_count", "true")
	}
	if params.IncludeDisabled {
		values.Add("include_disabled", "true")
	}
	if params.IncludeUsers {
		values.Add("include_users", "true")
	}

	response, err := api.userGroupRequest(ctx, "usergroups.list", values)
	if err != nil {
		return nil, err
	}
	return response.UserGroups, nil
}

// UpdateUserGroupsOption options for the UpdateUserGroup method call.
type UpdateUserGroupsOption func(*UpdateUserGroupsParams)

// UpdateUserGroupsOptionName change the name of the User Group (default: empty, so it's no-op)
func UpdateUserGroupsOptionName(name string) UpdateUserGroupsOption {
	return func(params *UpdateUserGroupsParams) {
		params.Name = name
	}
}

// UpdateUserGroupsOptionHandle change the handle of the User Group (default: empty, so it's no-op)
func UpdateUserGroupsOptionHandle(handle string) UpdateUserGroupsOption {
	return func(params *UpdateUserGroupsParams) {
		params.Handle = handle
	}
}

// UpdateUserGroupsOptionDescription change the description of the User Group. (default: nil, so it's no-op)
func UpdateUserGroupsOptionDescription(description *string) UpdateUserGroupsOption {
	return func(params *UpdateUserGroupsParams) {
		params.Description = description
	}
}

// UpdateUserGroupsOptionChannels change the default channels of the User Group. (default: unspecified, so it's no-op)
func UpdateUserGroupsOptionChannels(channels []string) UpdateUserGroupsOption {
	return func(params *UpdateUserGroupsParams) {
		params.Channels = &channels
	}
}

// UpdateUserGroupsOptionEnableSection enable the section for the user group (default: false)
func UpdateUserGroupsOptionEnableSection(enableSection bool) UpdateUserGroupsOption {
	return func(params *UpdateUserGroupsParams) {
		params.EnableSection = enableSection
	}
}

// UpdateUserGroupsOptionTeamID specify the team id for the User Group. (default: nil, so it's no-op)
func UpdateUserGroupsOptionTeamID(teamID string) UpdateUserGroupsOption {
	return func(params *UpdateUserGroupsParams) {
		params.TeamID = teamID
	}
}

// UpdateUserGroupsParams contains arguments for UpdateUserGroup method call
type UpdateUserGroupsParams struct {
	Name          string
	Handle        string
	Description   *string
	Channels      *[]string
	EnableSection bool
	TeamID        string
}

// UpdateUserGroup will update an existing user group.
// For more information see the UpdateUserGroupContext documentation.
func (api *Client) UpdateUserGroup(userGroupID string, options ...UpdateUserGroupsOption) (UserGroup, error) {
	return api.UpdateUserGroupContext(context.Background(), userGroupID, options...)
}

// UpdateUserGroupContext will update an existing user group with a custom context.
// Slack API docs: https://api.slack.com/methods/usergroups.update
func (api *Client) UpdateUserGroupContext(ctx context.Context, userGroupID string, options ...UpdateUserGroupsOption) (UserGroup, error) {
	params := UpdateUserGroupsParams{}

	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token":     {api.token},
		"usergroup": {userGroupID},
	}

	if params.Name != "" {
		values["name"] = []string{params.Name}
	}

	if params.Handle != "" {
		values["handle"] = []string{params.Handle}
	}

	if params.Description != nil {
		values["description"] = []string{*params.Description}
	}

	if params.Channels != nil {
		values["channels"] = []string{strings.Join(*params.Channels, ",")}
	}

	if params.EnableSection {
		values["enable_section"] = []string{strconv.FormatBool(params.EnableSection)}
	}

	if params.TeamID != "" {
		values["team_id"] = []string{params.TeamID}
	}

	response, err := api.userGroupRequest(ctx, "usergroups.update", values)
	if err != nil {
		return UserGroup{}, err
	}
	return response.UserGroup, nil
}

// GetUserGroupMembersOption options for the GetUserGroupMembers method call.
type GetUserGroupMembersOption func(*GetUserGroupMembersParams)

// GetUserGroupMembersParams contains arguments for GetUserGroupMembers method call
type GetUserGroupMembersParams struct {
	IncludeDisabled bool
	TeamID          string
}

// GetUserGroupMembersOptionIncludeDisabled include disabled User Groups (default: false)
func GetUserGroupMembersOptionIncludeDisabled(b bool) GetUserGroupMembersOption {
	return func(params *GetUserGroupMembersParams) {
		params.IncludeDisabled = b
	}
}

// GetUserGroupMembersOptionTeamID include team Id
func GetUserGroupMembersOptionTeamID(teamID string) GetUserGroupMembersOption {
	return func(params *GetUserGroupMembersParams) {
		params.TeamID = teamID
	}
}

// GetUserGroupMembers will retrieve the current list of users in a group.
// For more information see the GetUserGroupMembersContext documentation.
func (api *Client) GetUserGroupMembers(userGroup string, options ...GetUserGroupMembersOption) ([]string, error) {
	return api.GetUserGroupMembersContext(context.Background(), userGroup, options...)
}

// GetUserGroupMembersContext will retrieve the current list of users in a group with a custom context.
// Slack API docs: https://api.slack.com/methods/usergroups.users.list
func (api *Client) GetUserGroupMembersContext(ctx context.Context, userGroup string, options ...GetUserGroupMembersOption) ([]string, error) {
	params := GetUserGroupMembersParams{}

	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token":     {api.token},
		"usergroup": {userGroup},
	}

	if params.IncludeDisabled {
		values.Add("include_disabled", "true")
	}

	if params.TeamID != "" {
		values.Add("team_id", params.TeamID)
	}

	response, err := api.userGroupRequest(ctx, "usergroups.users.list", values)
	if err != nil {
		return []string{}, err
	}
	return response.Users, nil
}

// UpdateUserGroupMembersOption options for the UpdateUserGroupMembers method call.
type UpdateUserGroupMembersOption func(*UpdateUserGroupMembersParams)

// UpdateUserGroupMembersParams contains arguments for UpdateUserGroupMembers method call
type UpdateUserGroupMembersParams struct {
	AdditionalChannels []string
	IncludeCount       bool
	IsShared           bool
	TeamID             string
}

// UpdateUserGroupMembersOptionAdditionalChannels include additional channels
func UpdateUserGroupMembersOptionAdditionalChannels(channels []string) UpdateUserGroupMembersOption {
	return func(params *UpdateUserGroupMembersParams) {
		params.AdditionalChannels = channels
	}
}

// UpdateUserGroupMembersOptionIsShared include the count of User Groups (default: false)
func UpdateUserGroupMembersOptionIsShared(b bool) UpdateUserGroupMembersOption {
	return func(params *UpdateUserGroupMembersParams) {
		params.IsShared = b
	}
}

// UpdateUserGroupMembersOptionIncludeCount include the count of User Groups (default: false)
func UpdateUserGroupMembersOptionIncludeCount(b bool) UpdateUserGroupMembersOption {
	return func(params *UpdateUserGroupMembersParams) {
		params.IncludeCount = b
	}
}

// UpdateUserGroupMembersOptionTeamID include team Id
func UpdateUserGroupMembersOptionTeamID(teamID string) UpdateUserGroupMembersOption {
	return func(params *UpdateUserGroupMembersParams) {
		params.TeamID = teamID
	}
}

// UpdateUserGroupMembers will update the members of an existing user group.
// For more information see the UpdateUserGroupMembersContext documentation.
func (api *Client) UpdateUserGroupMembers(userGroup string, members string, options ...UpdateUserGroupMembersOption) (UserGroup, error) {
	return api.UpdateUserGroupMembersContext(context.Background(), userGroup, members, options...)
}

// UpdateUserGroupMembersContext will update the members of an existing user group with a custom context.
// Slack API docs: https://api.slack.com/methods/usergroups.update
func (api *Client) UpdateUserGroupMembersContext(ctx context.Context, userGroup string, members string, options ...UpdateUserGroupMembersOption) (UserGroup, error) {
	params := UpdateUserGroupMembersParams{}

	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token":     {api.token},
		"usergroup": {userGroup},
		"users":     {members},
	}

	if params.IncludeCount {
		values.Add("include_count", "true")
	}

	if params.IsShared {
		values.Add("is_shared", "true")
	}

	if params.TeamID != "" {
		values.Add("team_id", params.TeamID)
	}

	if len(params.AdditionalChannels) > 0 {
		values["additional_channels"] = []string{strings.Join(params.AdditionalChannels, ",")}
	}

	response, err := api.userGroupRequest(ctx, "usergroups.users.update", values)
	if err != nil {
		return UserGroup{}, err
	}
	return response.UserGroup, nil
}
