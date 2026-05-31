package slack

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// AdminRolesAddAssignmentsParams contains arguments for AdminRolesAddAssignments method call.
type AdminRolesAddAssignmentsParams struct {
	RoleID    string
	EntityIDs []string
	UserIDs   []string
}

// AdminRolesRejectedUser represents a user that could not be assigned a role.
type AdminRolesRejectedUser struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

// AdminRolesRejectedEntity represents an entity that could not be assigned a role.
type AdminRolesRejectedEntity struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

// AdminRolesAddAssignmentsResponse represents the response from admin.roles.addAssignments.
type AdminRolesAddAssignmentsResponse struct {
	SlackResponse
	RejectedUsers    []AdminRolesRejectedUser   `json:"rejected_users"`
	RejectedEntities []AdminRolesRejectedEntity `json:"rejected_entities"`
}

// AdminRolesAddAssignments adds members to a specified role.
// For more information see the admin.roles.addAssignments docs:
// https://api.slack.com/methods/admin.roles.addAssignments
func (api *Client) AdminRolesAddAssignments(ctx context.Context, params AdminRolesAddAssignmentsParams) (*AdminRolesAddAssignmentsResponse, error) {
	values := url.Values{
		"token":   {api.token},
		"role_id": {params.RoleID},
	}

	if len(params.EntityIDs) > 0 {
		values.Add("entity_ids", strings.Join(params.EntityIDs, ","))
	}

	if len(params.UserIDs) > 0 {
		values.Add("user_ids", strings.Join(params.UserIDs, ","))
	}

	response := &AdminRolesAddAssignmentsResponse{}
	err := api.postMethod(ctx, "admin.roles.addAssignments", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

type adminRolesListAssignmentsParams struct {
	roleIDs       []string
	entityIDs     []string
	limit         int
	cursor        string
	sortDirection string
}

// AdminRolesListAssignmentsOption is an option for AdminRolesListAssignments.
type AdminRolesListAssignmentsOption func(*adminRolesListAssignmentsParams)

// AdminRolesListAssignmentsOptionRoleIDs filters results to the specified role IDs.
func AdminRolesListAssignmentsOptionRoleIDs(roleIDs []string) AdminRolesListAssignmentsOption {
	return func(params *adminRolesListAssignmentsParams) {
		params.roleIDs = roleIDs
	}
}

// AdminRolesListAssignmentsOptionEntityIDs filters results to the specified entity IDs.
func AdminRolesListAssignmentsOptionEntityIDs(entityIDs []string) AdminRolesListAssignmentsOption {
	return func(params *adminRolesListAssignmentsParams) {
		params.entityIDs = entityIDs
	}
}

// AdminRolesListAssignmentsOptionLimit sets the maximum number of results to return.
func AdminRolesListAssignmentsOptionLimit(limit int) AdminRolesListAssignmentsOption {
	return func(params *adminRolesListAssignmentsParams) {
		params.limit = limit
	}
}

// AdminRolesListAssignmentsOptionCursor sets the cursor for pagination.
func AdminRolesListAssignmentsOptionCursor(cursor string) AdminRolesListAssignmentsOption {
	return func(params *adminRolesListAssignmentsParams) {
		params.cursor = cursor
	}
}

// AdminRolesListAssignmentsOptionSortDir sets the sort direction.
// Valid values: "asc", "desc".
func AdminRolesListAssignmentsOptionSortDir(sortDir string) AdminRolesListAssignmentsOption {
	return func(params *adminRolesListAssignmentsParams) {
		params.sortDirection = sortDir
	}
}

// RoleAssignment represents a single role assignment.
type RoleAssignment struct {
	RoleID     string `json:"role_id"`
	EntityID   string `json:"entity_id,omitempty"`
	UserID     string `json:"user_id,omitempty"`
	DateCreate int64  `json:"date_create,omitempty"`
}

// AdminRolesListAssignmentsResponse represents the response from admin.roles.listAssignments.
type AdminRolesListAssignmentsResponse struct {
	SlackResponse
	RoleAssignments  []RoleAssignment `json:"role_assignments"`
	ResponseMetadata ResponseMetadata `json:"response_metadata"`
}

// AdminRolesListAssignments lists assignments for roles.
// For more information see the admin.roles.listAssignments docs:
// https://api.slack.com/methods/admin.roles.listAssignments
func (api *Client) AdminRolesListAssignments(ctx context.Context, options ...AdminRolesListAssignmentsOption) (*AdminRolesListAssignmentsResponse, error) {
	params := adminRolesListAssignmentsParams{}
	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token": {api.token},
	}

	if len(params.roleIDs) > 0 {
		values.Add("role_ids", strings.Join(params.roleIDs, ","))
	}

	if len(params.entityIDs) > 0 {
		values.Add("entity_ids", strings.Join(params.entityIDs, ","))
	}

	if params.limit > 0 {
		values.Add("limit", strconv.Itoa(params.limit))
	}

	if params.cursor != "" {
		values.Add("cursor", params.cursor)
	}

	if params.sortDirection != "" {
		values.Add("sort_dir", params.sortDirection)
	}

	response := &AdminRolesListAssignmentsResponse{}
	err := api.postMethod(ctx, "admin.roles.listAssignments", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// AdminRolesRemoveAssignmentsParams contains arguments for AdminRolesRemoveAssignments method call.
type AdminRolesRemoveAssignmentsParams struct {
	RoleID    string
	EntityIDs []string
	UserIDs   []string
}

// AdminRolesRemoveAssignmentsResponse represents the response from admin.roles.removeAssignments.
type AdminRolesRemoveAssignmentsResponse struct {
	SlackResponse
	RejectedUsers    []AdminRolesRejectedUser   `json:"rejected_users"`
	RejectedEntities []AdminRolesRejectedEntity `json:"rejected_entities"`
}

// AdminRolesRemoveAssignments removes members from a specified role.
// For more information see the admin.roles.removeAssignments docs:
// https://api.slack.com/methods/admin.roles.removeAssignments
func (api *Client) AdminRolesRemoveAssignments(ctx context.Context, params AdminRolesRemoveAssignmentsParams) (*AdminRolesRemoveAssignmentsResponse, error) {
	values := url.Values{
		"token":   {api.token},
		"role_id": {params.RoleID},
	}

	if len(params.EntityIDs) > 0 {
		values.Add("entity_ids", strings.Join(params.EntityIDs, ","))
	}

	if len(params.UserIDs) > 0 {
		values.Add("user_ids", strings.Join(params.UserIDs, ","))
	}

	response := &AdminRolesRemoveAssignmentsResponse{}
	err := api.postMethod(ctx, "admin.roles.removeAssignments", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}
