package slack

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// DNDOptionTeamID sets the team_id parameter for DND methods. Required after
// workspace migration when the API returns missing_argument: team_id.
func DNDOptionTeamID(teamID string) ParamOption {
	return func(v *url.Values) {
		v.Set("team_id", teamID)
	}
}

type SnoozeDebug struct {
	SnoozeEndDate string `json:"snooze_end_date"`
}

type SnoozeInfo struct {
	SnoozeEnabled   bool        `json:"snooze_enabled,omitempty"`
	SnoozeEndTime   int         `json:"snooze_endtime,omitempty"`
	SnoozeRemaining int         `json:"snooze_remaining,omitempty"`
	SnoozeDebug     SnoozeDebug `json:"snooze_debug,omitempty"`
}

type DNDStatus struct {
	Enabled            bool `json:"dnd_enabled"`
	NextStartTimestamp int  `json:"next_dnd_start_ts"`
	NextEndTimestamp   int  `json:"next_dnd_end_ts"`
	SnoozeInfo
}

type dndResponseFull struct {
	DNDStatus
	SlackResponse
}

type dndTeamInfoResponse struct {
	Users map[string]DNDStatus `json:"users"`
	SlackResponse
}

func (api *Client) dndRequest(ctx context.Context, path string, values url.Values) (*dndResponseFull, error) {
	response := &dndResponseFull{}
	err := api.postMethod(ctx, path, values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// EndDND ends the user's scheduled Do Not Disturb session.
// For more information see the EndDNDContext documentation.
func (api *Client) EndDND() error {
	return api.EndDNDContext(context.Background())
}

// EndDNDContext ends the user's scheduled Do Not Disturb session with a custom context.
// Slack API docs: https://docs.slack.dev/reference/methods/dnd.endDnd
func (api *Client) EndDNDContext(ctx context.Context) error {
	values := url.Values{
		"token": {api.token},
	}

	response := &SlackResponse{}

	if err := api.postMethod(ctx, "dnd.endDnd", values, response); err != nil {
		return err
	}

	return response.Err()
}

// EndSnooze ends the current user's snooze mode.
// For more information see the EndSnoozeContext documentation.
func (api *Client) EndSnooze() (*DNDStatus, error) {
	return api.EndSnoozeContext(context.Background())
}

// EndSnoozeContext ends the current user's snooze mode with a custom context.
// Slack API docs: https://docs.slack.dev/reference/methods/dnd.endSnooze
func (api *Client) EndSnoozeContext(ctx context.Context) (*DNDStatus, error) {
	values := url.Values{
		"token": {api.token},
	}

	response, err := api.dndRequest(ctx, "dnd.endSnooze", values)
	if err != nil {
		return nil, err
	}
	return &response.DNDStatus, nil
}

// GetDNDInfo provides information about a user's current Do Not Disturb settings.
// For more information see the GetDNDInfoContext documentation.
func (api *Client) GetDNDInfo(user *string, options ...ParamOption) (*DNDStatus, error) {
	return api.GetDNDInfoContext(context.Background(), user, options...)
}

// GetDNDInfoContext provides information about a user's current Do Not Disturb settings with a custom context.
// Slack API docs: https://docs.slack.dev/reference/methods/dnd.info/
func (api *Client) GetDNDInfoContext(ctx context.Context, user *string, options ...ParamOption) (*DNDStatus, error) {
	values := url.Values{
		"token": {api.token},
	}
	if user != nil {
		values.Set("user", *user)
	}
	for _, opt := range options {
		opt(&values)
	}

	response, err := api.dndRequest(ctx, "dnd.info", values)
	if err != nil {
		return nil, err
	}
	return &response.DNDStatus, nil
}

// GetDNDTeamInfo provides information about a user's current Do Not Disturb settings.
// For more information see the GetDNDTeamInfoContext documentation.
func (api *Client) GetDNDTeamInfo(users []string, options ...ParamOption) (map[string]DNDStatus, error) {
	return api.GetDNDTeamInfoContext(context.Background(), users, options...)
}

// GetDNDTeamInfoContext provides information about a user's current Do Not Disturb settings with a custom context.
// Slack API docs: https://docs.slack.dev/reference/methods/dnd.teamInfo
func (api *Client) GetDNDTeamInfoContext(ctx context.Context, users []string, options ...ParamOption) (map[string]DNDStatus, error) {
	values := url.Values{
		"token": {api.token},
		"users": {strings.Join(users, ",")},
	}
	for _, opt := range options {
		opt(&values)
	}
	response := &dndTeamInfoResponse{}

	if err := api.postMethod(ctx, "dnd.teamInfo", values, response); err != nil {
		return nil, err
	}

	if response.Err() != nil {
		return nil, response.Err()
	}

	return response.Users, nil
}

// SetSnooze adjusts the snooze duration for a user's Do Not Disturb settings.
// For more information see the SetSnoozeContext documentation.
func (api *Client) SetSnooze(minutes int) (*DNDStatus, error) {
	return api.SetSnoozeContext(context.Background(), minutes)
}

// SetSnoozeContext adjusts the snooze duration for a user's Do Not Disturb settings.
// If a snooze session is not already active for the user, invoking this method will
// begin one for the specified duration.
// Slack API docs: https://docs.slack.dev/reference/methods/dnd.setSnooze
func (api *Client) SetSnoozeContext(ctx context.Context, minutes int) (*DNDStatus, error) {
	values := url.Values{
		"token":       {api.token},
		"num_minutes": {strconv.Itoa(minutes)},
	}

	response, err := api.dndRequest(ctx, "dnd.setSnooze", values)
	if err != nil {
		return nil, err
	}
	return &response.DNDStatus, nil
}
