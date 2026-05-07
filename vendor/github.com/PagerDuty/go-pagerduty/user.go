package pagerduty

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
)

// NotificationRule is a rule for notifying the user.
type NotificationRule struct {
	ID                  string        `json:"id,omitempty"`
	Type                string        `json:"type,omitempty"`
	Summary             string        `json:"summary,omitempty"`
	Self                string        `json:"self,omitempty"`
	HTMLURL             string        `json:"html_url,omitempty"`
	StartDelayInMinutes uint          `json:"start_delay_in_minutes"`
	CreatedAt           string        `json:"created_at"`
	ContactMethod       ContactMethod `json:"contact_method"`
	Urgency             string        `json:"urgency"`
}

// User is a member of a PagerDuty account that has the ability to interact with incidents and other data on the account.
type User struct {
	APIObject
	Name              string             `json:"name"`
	Email             string             `json:"email"`
	Timezone          string             `json:"time_zone,omitempty"`
	Color             string             `json:"color,omitempty"`
	Role              string             `json:"role,omitempty"`
	AvatarURL         string             `json:"avatar_url,omitempty"`
	Description       string             `json:"description,omitempty"`
	InvitationSent    bool               `json:"invitation_sent,omitempty"`
	ContactMethods    []ContactMethod    `json:"contact_methods,omitempty"`
	NotificationRules []NotificationRule `json:"notification_rules,omitempty"`
	JobTitle          string             `json:"job_title,omitempty"`
	Teams             []Team             `json:"teams,omitempty"`
}

// ContactMethod is a way of contacting the user.
type ContactMethod struct {
	ID             string `json:"id,omitempty"`
	Type           string `json:"type,omitempty"`
	Summary        string `json:"summary,omitempty"`
	Self           string `json:"self,omitempty"`
	HTMLURL        string `json:"html_url,omitempty"`
	Label          string `json:"label"`
	Address        string `json:"address"`
	SendShortEmail bool   `json:"send_short_email,omitempty"`
	SendHTMLEmail  bool   `json:"send_html_email,omitempty"`
	Blacklisted    bool   `json:"blacklisted,omitempty"`
	CountryCode    int    `json:"country_code,omitempty"`
	Enabled        bool   `json:"enabled,omitempty"`
}

// ListUsersResponse is the data structure returned from calling the ListUsers API endpoint.
type ListUsersResponse struct {
	APIListObject
	Users []User `json:"users"`
}

// ListUsersOptions is the data structure used when calling the ListUsers API endpoint.
type ListUsersOptions struct {
	// Limit is the pagination parameter that limits the number of results per
	// page. PagerDuty defaults this value to 25 if omitted, and sets an upper
	// bound of 100.
	Limit uint `url:"limit,omitempty"`

	// Offset is the pagination parameter that specifies the offset at which to
	// start pagination results. When trying to request the next page of
	// results, the new Offset value should be currentOffset + Limit.
	Offset uint `url:"offset,omitempty"`

	// Total is the pagination parameter to request that the API return the
	// total count of items in the response. If this field is omitted or set to
	// false, the total number of results will not be sent back from the PagerDuty API.
	//
	// Setting this to true will slow down the API response times, and so it's
	// recommended to omit it unless you've a specific reason for wanting the
	// total count of items in the collection.
	Total bool `url:"total,omitempty"`

	Query    string   `url:"query,omitempty"`
	TeamIDs  []string `url:"team_ids,omitempty,brackets"`
	Includes []string `url:"include,omitempty,brackets"`
}

// ListContactMethodsResponse is the data structure returned from calling the GetUserContactMethod API endpoint.
type ListContactMethodsResponse struct {
	APIListObject
	ContactMethods []ContactMethod `json:"contact_methods"`
}

// ListUserNotificationRulesResponse the data structure returned from calling the ListNotificationRules API endpoint.
type ListUserNotificationRulesResponse struct {
	APIListObject
	NotificationRules []NotificationRule `json:"notification_rules"`
}

// GetUserOptions is the data structure used when calling the GetUser API endpoint.
type GetUserOptions struct {
	Includes []string `url:"include,omitempty,brackets"`
}

// GetCurrentUserOptions is the data structure used when calling the GetCurrentUser API endpoint.
type GetCurrentUserOptions struct {
	Includes []string `url:"include,omitempty,brackets"`
}

// ListUsers lists users of your PagerDuty account, optionally filtered by a
// search query.
//
// Deprecated: Use ListUsersWithContext instead.
func (c *Client) ListUsers(o ListUsersOptions) (*ListUsersResponse, error) {
	return c.ListUsersWithContext(context.Background(), o)
}

// ListUsersWithContext lists users of your PagerDuty account, optionally filtered by a search query.
func (c *Client) ListUsersWithContext(ctx context.Context, o ListUsersOptions) (*ListUsersResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/users?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result ListUsersResponse
	if err := c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateUser creates a new user.
//
// Deprecated: Use CreateUserWithContext instead.
func (c *Client) CreateUser(u User) (*User, error) {
	return c.CreateUserWithContext(context.Background(), u)
}

// CreateUserWithContext creates a new user.
func (c *Client) CreateUserWithContext(ctx context.Context, u User) (*User, error) {
	d := map[string]User{
		"user": u,
	}

	resp, err := c.post(ctx, "/users", d, nil)
	return getUserFromResponse(c, resp, err)
}

// DeleteUser deletes a user.
//
// Deprecated: Use DeleteUserWithContext instead.
func (c *Client) DeleteUser(id string) error {
	return c.DeleteUserWithContext(context.Background(), id)
}

// DeleteUserWithContext deletes a user.
func (c *Client) DeleteUserWithContext(ctx context.Context, id string) error {
	_, err := c.delete(ctx, "/users/"+id)
	return err
}

// GetUser gets details about an existing user.
//
// Deprecated: Use GetUserWithContext instead.
func (c *Client) GetUser(id string, o GetUserOptions) (*User, error) {
	return c.GetUserWithContext(context.Background(), id, o)
}

// GetUserWithContext gets details about an existing user.
func (c *Client) GetUserWithContext(ctx context.Context, id string, o GetUserOptions) (*User, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/users/"+id+"?"+v.Encode(), nil)
	return getUserFromResponse(c, resp, err)
}

// UpdateUser updates an existing user.
//
// Deprecated: Use UpdateUserWithContext instead.
func (c *Client) UpdateUser(u User) (*User, error) {
	return c.UpdateUserWithContext(context.Background(), u)
}

// UpdateUserWithContext updates an existing user.
func (c *Client) UpdateUserWithContext(ctx context.Context, u User) (*User, error) {
	d := map[string]User{
		"user": u,
	}

	resp, err := c.put(ctx, "/users/"+u.ID, d, nil)
	return getUserFromResponse(c, resp, err)
}

// GetCurrentUser gets details about the authenticated user when using a
// user-level API key or OAuth token.
//
// Deprecated: Use GetCurrentUserWithContext instead.
func (c *Client) GetCurrentUser(o GetCurrentUserOptions) (*User, error) {
	return c.GetCurrentUserWithContext(context.Background(), o)
}

// GetCurrentUserWithContext gets details about the authenticated user when
// using a user-level API key or OAuth token.
func (c *Client) GetCurrentUserWithContext(ctx context.Context, o GetCurrentUserOptions) (*User, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/users/me?"+v.Encode(), nil)
	return getUserFromResponse(c, resp, err)
}

func getUserFromResponse(c *Client, resp *http.Response, err error) (*User, error) {
	if err != nil {
		return nil, err
	}

	var target map[string]User
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}

	const rootNode = "user"

	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &t, nil
}

// ListUserContactMethods fetches contact methods of the existing user.
//
// Deprecated: Use ListUserContactMethodsWithContext instead.
func (c *Client) ListUserContactMethods(userID string) (*ListContactMethodsResponse, error) {
	return c.ListUserContactMethodsWithContext(context.Background(), userID)
}

// ListUserContactMethodsWithContext fetches contact methods of the existing user.
func (c *Client) ListUserContactMethodsWithContext(ctx context.Context, userID string) (*ListContactMethodsResponse, error) {
	resp, err := c.get(ctx, "/users/"+userID+"/contact_methods", nil)
	if err != nil {
		return nil, err
	}

	var result ListContactMethodsResponse
	if err := c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetUserContactMethod gets details about a contact method.
//
// Deprecated: Use GetUserContactMethodWithContext instead.
func (c *Client) GetUserContactMethod(userID, contactMethodID string) (*ContactMethod, error) {
	return c.GetUserContactMethodWithContext(context.Background(), userID, contactMethodID)
}

// GetUserContactMethodWithContext gets details about a contact method.
func (c *Client) GetUserContactMethodWithContext(ctx context.Context, userID, contactMethodID string) (*ContactMethod, error) {
	resp, err := c.get(ctx, "/users/"+userID+"/contact_methods/"+contactMethodID, nil)
	return getContactMethodFromResponse(c, resp, err)
}

// DeleteUserContactMethod deletes a user.
//
// Deprecated: Use DeleteUserContactMethodWithContext instead.
func (c *Client) DeleteUserContactMethod(userID, contactMethodID string) error {
	return c.DeleteUserContactMethodWithContext(context.Background(), userID, contactMethodID)
}

// DeleteUserContactMethodWithContext deletes a user.
func (c *Client) DeleteUserContactMethodWithContext(ctx context.Context, userID, contactMethodID string) error {
	_, err := c.delete(ctx, "/users/"+userID+"/contact_methods/"+contactMethodID)
	return err
}

// CreateUserContactMethod creates a new contact method for user.
//
// Deprecated: Use CreateUserContactMethodWithContext instead.
func (c *Client) CreateUserContactMethod(userID string, cm ContactMethod) (*ContactMethod, error) {
	return c.CreateUserContactMethodWithContext(context.Background(), userID, cm)
}

// CreateUserContactMethodWithContext creates a new contact method for user.
func (c *Client) CreateUserContactMethodWithContext(ctx context.Context, userID string, cm ContactMethod) (*ContactMethod, error) {
	d := map[string]ContactMethod{
		"contact_method": cm,
	}

	resp, err := c.post(ctx, "/users/"+userID+"/contact_methods", d, nil)
	return getContactMethodFromResponse(c, resp, err)
}

// UpdateUserContactMethod updates an existing user. It's recommended to use
// UpdateUserContactMethodWithContext instead.
func (c *Client) UpdateUserContactMethod(userID string, cm ContactMethod) (*ContactMethod, error) {
	return c.UpdateUserContactMethodWthContext(context.Background(), userID, cm)
}

// UpdateUserContactMethodWthContext updates an existing user.
func (c *Client) UpdateUserContactMethodWthContext(ctx context.Context, userID string, cm ContactMethod) (*ContactMethod, error) {
	d := map[string]ContactMethod{
		"contact_method": cm,
	}

	resp, err := c.put(ctx, "/users/"+userID+"/contact_methods/"+cm.ID, d, nil)
	return getContactMethodFromResponse(c, resp, err)
}

func getContactMethodFromResponse(c *Client, resp *http.Response, err error) (*ContactMethod, error) {
	if err != nil {
		return nil, err
	}

	var target map[string]ContactMethod
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}

	const rootNode = "contact_method"

	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &t, nil
}

// GetUserNotificationRule gets details about a notification rule.
//
// Deprecated: Use GetUserNotificationRuleWithContext instead.
func (c *Client) GetUserNotificationRule(userID, ruleID string) (*NotificationRule, error) {
	return c.GetUserNotificationRuleWithContext(context.Background(), userID, ruleID)
}

// GetUserNotificationRuleWithContext gets details about a notification rule.
func (c *Client) GetUserNotificationRuleWithContext(ctx context.Context, userID, ruleID string) (*NotificationRule, error) {
	resp, err := c.get(ctx, "/users/"+userID+"/notification_rules/"+ruleID, nil)
	return getUserNotificationRuleFromResponse(c, resp, err)
}

// CreateUserNotificationRule creates a new notification rule for a user.
//
// Deprecated: Use CreateUserNotificationRuleWithContext instead.
func (c *Client) CreateUserNotificationRule(userID string, rule NotificationRule) (*NotificationRule, error) {
	return c.CreateUserNotificationRuleWithContext(context.Background(), userID, rule)
}

// CreateUserNotificationRuleWithContext creates a new notification rule for a user.
func (c *Client) CreateUserNotificationRuleWithContext(ctx context.Context, userID string, rule NotificationRule) (*NotificationRule, error) {
	d := map[string]NotificationRule{
		"notification_rule": rule,
	}

	resp, err := c.post(ctx, "/users/"+userID+"/notification_rules", d, nil)
	return getUserNotificationRuleFromResponse(c, resp, err)
}

// UpdateUserNotificationRule updates a notification rule for a user.
//
// Deprecated: Use UpdateUserNotificationRuleWithContext instead.
func (c *Client) UpdateUserNotificationRule(userID string, rule NotificationRule) (*NotificationRule, error) {
	return c.UpdateUserNotificationRuleWithContext(context.Background(), userID, rule)
}

// UpdateUserNotificationRuleWithContext updates a notification rule for a user.
func (c *Client) UpdateUserNotificationRuleWithContext(ctx context.Context, userID string, rule NotificationRule) (*NotificationRule, error) {
	d := map[string]NotificationRule{
		"notification_rule": rule,
	}

	resp, err := c.put(ctx, "/users/"+userID+"/notification_rules/"+rule.ID, d, nil)
	return getUserNotificationRuleFromResponse(c, resp, err)
}

// DeleteUserNotificationRule deletes a notification rule for a user.
//
// Deprecated: Use DeleteUserNotificationRuleWithContext instead.
func (c *Client) DeleteUserNotificationRule(userID, ruleID string) error {
	return c.DeleteUserNotificationRuleWithContext(context.Background(), userID, ruleID)
}

// DeleteUserNotificationRuleWithContext deletes a notification rule for a user.
func (c *Client) DeleteUserNotificationRuleWithContext(ctx context.Context, userID, ruleID string) error {
	_, err := c.delete(ctx, "/users/"+userID+"/notification_rules/"+ruleID)
	return err
}

// ListUserNotificationRules fetches notification rules of the existing user.
//
// Deprecated: Use ListUserNotificationRulesWithContext instead.
func (c *Client) ListUserNotificationRules(userID string) (*ListUserNotificationRulesResponse, error) {
	return c.ListUserNotificationRulesWithContext(context.Background(), userID)
}

// ListUserNotificationRulesWithContext fetches notification rules of the existing user.
func (c *Client) ListUserNotificationRulesWithContext(ctx context.Context, userID string) (*ListUserNotificationRulesResponse, error) {
	resp, err := c.get(ctx, "/users/"+userID+"/notification_rules", nil)
	if err != nil {
		return nil, err
	}

	var result ListUserNotificationRulesResponse
	if err := c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func getUserNotificationRuleFromResponse(c *Client, resp *http.Response, err error) (*NotificationRule, error) {
	if err != nil {
		return nil, err
	}

	var target map[string]NotificationRule
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}

	const rootNode = "notification_rule"

	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &t, nil
}
