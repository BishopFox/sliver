package slack

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"
)

type Call struct {
	ID                string            `json:"id"`
	Title             string            `json:"title"`
	DateStart         JSONTime          `json:"date_start"`
	DateEnd           JSONTime          `json:"date_end"`
	ExternalUniqueID  string            `json:"external_unique_id"`
	JoinURL           string            `json:"join_url"`
	DesktopAppJoinURL string            `json:"desktop_app_join_url"`
	ExternalDisplayID string            `json:"external_display_id"`
	Participants      []CallParticipant `json:"users"`
	Channels          []string          `json:"channels"`
}

// CallParticipant is a thin user representation which has a SlackID, ExternalID, or both.
//
// See: https://api.slack.com/apis/calls#users
type CallParticipant struct {
	SlackID     string `json:"slack_id,omitempty"`
	ExternalID  string `json:"external_id,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
}

// Valid checks if the CallUser has a is valid with a SlackID or ExternalID or both.
func (u CallParticipant) Valid() bool {
	return u.SlackID != "" || u.ExternalID != ""
}

type AddCallParameters struct {
	JoinURL           string // Required
	ExternalUniqueID  string // Required
	CreatedBy         string // Required if using a bot token
	Title             string
	DesktopAppJoinURL string
	ExternalDisplayID string
	DateStart         JSONTime
	Participants      []CallParticipant
}

type UpdateCallParameters struct {
	Title             string
	DesktopAppJoinURL string
	JoinURL           string
}

type EndCallParameters struct {
	// Duration is the duration of the call in seconds. Omitted if 0.
	Duration time.Duration
}

type callResponse struct {
	Call Call `json:"call"`
	SlackResponse
}

// AddCall adds a new Call to the Slack API.
func (api *Client) AddCall(params AddCallParameters) (Call, error) {
	return api.AddCallContext(context.Background(), params)
}

// AddCallContext adds a new Call to the Slack API.
func (api *Client) AddCallContext(ctx context.Context, params AddCallParameters) (Call, error) {
	values := url.Values{
		"token":              {api.token},
		"join_url":           {params.JoinURL},
		"external_unique_id": {params.ExternalUniqueID},
	}
	if params.CreatedBy != "" {
		values.Set("created_by", params.CreatedBy)
	}
	if params.DateStart != 0 {
		values.Set("date_start", strconv.FormatInt(int64(params.DateStart), 10))
	}
	if params.DesktopAppJoinURL != "" {
		values.Set("desktop_app_join_url", params.DesktopAppJoinURL)
	}
	if params.ExternalDisplayID != "" {
		values.Set("external_display_id", params.ExternalDisplayID)
	}
	if params.Title != "" {
		values.Set("title", params.Title)
	}
	if len(params.Participants) > 0 {
		data, err := json.Marshal(params.Participants)
		if err != nil {
			return Call{}, err
		}
		values.Set("users", string(data))
	}

	response := &callResponse{}
	if err := api.postMethod(ctx, "calls.add", values, response); err != nil {
		return Call{}, err
	}

	return response.Call, response.Err()
}

// GetCallInfo returns information about a Call.
func (api *Client) GetCall(callID string) (Call, error) {
	return api.GetCallContext(context.Background(), callID)
}

// GetCallInfoContext returns information about a Call.
func (api *Client) GetCallContext(ctx context.Context, callID string) (Call, error) {
	values := url.Values{
		"token": {api.token},
		"id":    {callID},
	}

	response := &callResponse{}
	if err := api.postMethod(ctx, "calls.info", values, response); err != nil {
		return Call{}, err
	}
	return response.Call, response.Err()
}

func (api *Client) UpdateCall(callID string, params UpdateCallParameters) (Call, error) {
	return api.UpdateCallContext(context.Background(), callID, params)
}

// UpdateCallContext updates a Call with the given parameters.
func (api *Client) UpdateCallContext(ctx context.Context, callID string, params UpdateCallParameters) (Call, error) {
	values := url.Values{
		"token": {api.token},
		"id":    {callID},
	}

	if params.DesktopAppJoinURL != "" {
		values.Set("desktop_app_join_url", params.DesktopAppJoinURL)
	}
	if params.JoinURL != "" {
		values.Set("join_url", params.JoinURL)
	}
	if params.Title != "" {
		values.Set("title", params.Title)
	}

	response := &callResponse{}
	if err := api.postMethod(ctx, "calls.update", values, response); err != nil {
		return Call{}, err
	}
	return response.Call, response.Err()
}

// EndCall ends a Call.
func (api *Client) EndCall(callID string, params EndCallParameters) error {
	return api.EndCallContext(context.Background(), callID, params)
}

// EndCallContext ends a Call.
func (api *Client) EndCallContext(ctx context.Context, callID string, params EndCallParameters) error {
	values := url.Values{
		"token": {api.token},
		"id":    {callID},
	}

	if params.Duration != 0 {
		values.Set("duration", strconv.FormatInt(int64(params.Duration.Seconds()), 10))
	}

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "calls.end", values, response); err != nil {
		return err
	}
	return response.Err()
}

// CallAddParticipants adds users to a Call.
func (api *Client) CallAddParticipants(callID string, participants []CallParticipant) error {
	return api.CallAddParticipantsContext(context.Background(), callID, participants)
}

// CallAddParticipantsContext adds users to a Call.
func (api *Client) CallAddParticipantsContext(ctx context.Context, callID string, participants []CallParticipant) error {
	return api.setCallParticipants(ctx, "calls.participants.add", callID, participants)
}

// CallRemoveParticipants removes users from a Call.
func (api *Client) CallRemoveParticipants(callID string, participants []CallParticipant) error {
	return api.CallRemoveParticipantsContext(context.Background(), callID, participants)
}

// CallRemoveParticipantsContext removes users from a Call.
func (api *Client) CallRemoveParticipantsContext(ctx context.Context, callID string, participants []CallParticipant) error {
	return api.setCallParticipants(ctx, "calls.participants.remove", callID, participants)
}

func (api *Client) setCallParticipants(ctx context.Context, method, callID string, participants []CallParticipant) error {
	values := url.Values{
		"token": {api.token},
		"id":    {callID},
	}

	data, err := json.Marshal(participants)
	if err != nil {
		return err
	}
	values.Set("users", string(data))

	response := &SlackResponse{}
	if err := api.postMethod(ctx, method, values, response); err != nil {
		return err
	}
	return response.Err()
}
