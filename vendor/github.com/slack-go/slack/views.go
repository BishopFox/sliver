package slack

import (
	"context"
	"encoding/json"
)

const (
	VTModal   ViewType = "modal"
	VTHomeTab ViewType = "home"
)

type ViewType string

type ViewState struct {
	Values map[string]map[string]BlockAction `json:"values"`
}

type View struct {
	SlackResponse
	ID                 string           `json:"id"`
	TeamID             string           `json:"team_id"`
	Type               ViewType         `json:"type"`
	Title              *TextBlockObject `json:"title"`
	Close              *TextBlockObject `json:"close"`
	Submit             *TextBlockObject `json:"submit"`
	Blocks             Blocks           `json:"blocks"`
	PrivateMetadata    string           `json:"private_metadata"`
	CallbackID         string           `json:"callback_id"`
	State              *ViewState       `json:"state"`
	Hash               string           `json:"hash"`
	ClearOnClose       bool             `json:"clear_on_close"`
	NotifyOnClose      bool             `json:"notify_on_close"`
	RootViewID         string           `json:"root_view_id"`
	PreviousViewID     string           `json:"previous_view_id"`
	AppID              string           `json:"app_id"`
	ExternalID         string           `json:"external_id"`
	BotID              string           `json:"bot_id"`
	AppInstalledTeamID string           `json:"app_installed_team_id"`
}

type ViewSubmissionCallbackResponseURL struct {
	BlockID     string `json:"block_id"`
	ActionID    string `json:"action_id"`
	ChannelID   string `json:"channel_id"`
	ResponseURL string `json:"response_url"`
}

type ViewSubmissionCallback struct {
	Hash         string                              `json:"hash"`
	ResponseURLs []ViewSubmissionCallbackResponseURL `json:"response_urls,omitempty"`
}

type ViewClosedCallback struct {
	IsCleared bool `json:"is_cleared"`
}

const (
	RAClear  ViewResponseAction = "clear"
	RAUpdate ViewResponseAction = "update"
	RAPush   ViewResponseAction = "push"
	RAErrors ViewResponseAction = "errors"
)

type ViewResponseAction string

type ViewSubmissionResponse struct {
	ResponseAction ViewResponseAction `json:"response_action"`
	View           *ModalViewRequest  `json:"view,omitempty"`
	Errors         map[string]string  `json:"errors,omitempty"`
}

// NewClearViewSubmissionResponse closes all open modals in the current stack.
//
// For HTTP-based apps, marshal this to JSON and write it as the HTTP response
// body. The response is not sent until the handler returns, so start any slow
// work in a goroutine and return promptly.
//
// For Socket Mode apps, pass this as the payload argument to Ack().
//
// See https://docs.slack.dev/surfaces/modals#closing_views
func NewClearViewSubmissionResponse() *ViewSubmissionResponse {
	return &ViewSubmissionResponse{
		ResponseAction: RAClear,
	}
}

// NewUpdateViewSubmissionResponse replaces the current modal with a new view.
//
// For HTTP-based apps, marshal this to JSON and write it as the HTTP response
// body. The response is not sent until the handler returns, so start any slow
// work in a goroutine and return promptly.
//
// For Socket Mode apps, pass this as the payload argument to Ack().
//
// See https://docs.slack.dev/surfaces/modals#updating_views
func NewUpdateViewSubmissionResponse(view *ModalViewRequest) *ViewSubmissionResponse {
	return &ViewSubmissionResponse{
		ResponseAction: RAUpdate,
		View:           view,
	}
}

// NewPushViewSubmissionResponse pushes a new view onto the modal stack.
//
// For HTTP-based apps, marshal this to JSON and write it as the HTTP response
// body. The response is not sent until the handler returns, so start any slow
// work in a goroutine and return promptly.
//
// For Socket Mode apps, pass this as the payload argument to Ack().
//
// See https://docs.slack.dev/surfaces/modals#pushing_views
func NewPushViewSubmissionResponse(view *ModalViewRequest) *ViewSubmissionResponse {
	return &ViewSubmissionResponse{
		ResponseAction: RAPush,
		View:           view,
	}
}

// NewErrorsViewSubmissionResponse displays validation errors on form fields.
//
// The errors map keys must be the BlockID of an InputBlock in the view. Keys
// that reference other block types (e.g. SectionBlock) are silently ignored
// by Slack, which shows a generic "trouble connecting" error instead.
//
// For HTTP-based apps, marshal this to JSON and write it as the HTTP response
// body. The response is not sent until the handler returns, so start any slow
// work in a goroutine and return promptly.
//
// For Socket Mode apps, pass this as the payload argument to Ack().
//
// See https://docs.slack.dev/surfaces/modals/#displaying_errors
func NewErrorsViewSubmissionResponse(errors map[string]string) *ViewSubmissionResponse {
	return &ViewSubmissionResponse{
		ResponseAction: RAErrors,
		Errors:         errors,
	}
}

type ModalViewRequest struct {
	Type            ViewType         `json:"type"`
	Title           *TextBlockObject `json:"title,omitempty"`
	Blocks          Blocks           `json:"blocks"`
	Close           *TextBlockObject `json:"close,omitempty"`
	Submit          *TextBlockObject `json:"submit,omitempty"`
	PrivateMetadata string           `json:"private_metadata,omitempty"`
	CallbackID      string           `json:"callback_id,omitempty"`
	ClearOnClose    bool             `json:"clear_on_close,omitempty"`
	NotifyOnClose   bool             `json:"notify_on_close,omitempty"`
	ExternalID      string           `json:"external_id,omitempty"`
}

type PublishViewContextRequest struct {
	UserID string             `json:"user_id"`
	View   HomeTabViewRequest `json:"view"`
	Hash   *string            `json:"hash,omitempty"`
}

func (v *ModalViewRequest) ViewType() ViewType {
	return v.Type
}

type HomeTabViewRequest struct {
	Type            ViewType `json:"type"`
	Blocks          Blocks   `json:"blocks"`
	PrivateMetadata string   `json:"private_metadata,omitempty"`
	CallbackID      string   `json:"callback_id,omitempty"`
	ExternalID      string   `json:"external_id,omitempty"`
}

func (v *HomeTabViewRequest) ViewType() ViewType {
	return v.Type
}

type openViewRequest struct {
	TriggerID string           `json:"trigger_id"`
	View      ModalViewRequest `json:"view"`
}

type pushViewRequest struct {
	TriggerID string           `json:"trigger_id"`
	View      ModalViewRequest `json:"view"`
}

type updateViewRequest struct {
	View       ModalViewRequest `json:"view"`
	ExternalID string           `json:"external_id,omitempty"`
	Hash       string           `json:"hash,omitempty"`
	ViewID     string           `json:"view_id,omitempty"`
}

type ViewResponse struct {
	SlackResponse
	View `json:"view"`
}

// OpenView opens a view for a user.
// For more information see the OpenViewContext documentation.
func (api *Client) OpenView(triggerID string, view ModalViewRequest) (*ViewResponse, error) {
	return api.OpenViewContext(context.Background(), triggerID, view)
}

// ValidateUniqueBlockID will verify if each input block has a unique block ID if set
func ValidateUniqueBlockID(view ModalViewRequest) bool {

	uniqueBlockID := map[string]bool{}

	for _, b := range view.Blocks.BlockSet {
		if inputBlock, ok := b.(*InputBlock); ok {
			if inputBlock.BlockID == "" {
				continue
			}
			if _, ok := uniqueBlockID[inputBlock.BlockID]; ok {
				return false
			}
			uniqueBlockID[inputBlock.BlockID] = true
		}
	}

	return true
}

// OpenViewContext opens a view for a user with a custom context.
// Slack API docs: https://docs.slack.dev/reference/methods/views.open
func (api *Client) OpenViewContext(
	ctx context.Context,
	triggerID string,
	view ModalViewRequest,
) (*ViewResponse, error) {
	if triggerID == "" {
		return nil, ErrParametersMissing
	}

	if !ValidateUniqueBlockID(view) {
		return nil, ErrBlockIDNotUnique
	}

	req := openViewRequest{
		TriggerID: triggerID,
		View:      view,
	}
	encoded, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp := &ViewResponse{}
	err = api.postJSONMethod(ctx, "views.open", api.token, encoded, resp)
	if err != nil {
		return nil, err
	}
	return resp, resp.Err()
}

// PublishView publishes a static view for a user.
// For more information see the PublishViewContext documentation.
func (api *Client) PublishView(userID string, view HomeTabViewRequest, hash string) (*ViewResponse, error) {
	var hashPtr *string
	if hash != "" {
		hashPtr = &hash
	}
	return api.PublishViewContext(context.Background(), PublishViewContextRequest{UserID: userID, View: view, Hash: hashPtr})
}

// PublishViewContext publishes a static view for a user with a custom context.
// Slack API docs: https://docs.slack.dev/reference/methods/views.publish
func (api *Client) PublishViewContext(
	ctx context.Context,
	req PublishViewContextRequest,
) (*ViewResponse, error) {
	if req.UserID == "" {
		return nil, ErrParametersMissing
	}
	encoded, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp := &ViewResponse{}
	err = api.postJSONMethod(ctx, "views.publish", api.token, encoded, resp)
	if err != nil {
		return nil, err
	}
	return resp, resp.Err()
}

// PushView pushes a view onto the stack of a root view.
// For more information see the PushViewContext documentation.
func (api *Client) PushView(triggerID string, view ModalViewRequest) (*ViewResponse, error) {
	return api.PushViewContext(context.Background(), triggerID, view)
}

// PushViewContext pushes a view onto the stack of a root view with a custom context.
// Slack API docs: https://docs.slack.dev/reference/methods/views.push
func (api *Client) PushViewContext(
	ctx context.Context,
	triggerID string,
	view ModalViewRequest,
) (*ViewResponse, error) {
	if triggerID == "" {
		return nil, ErrParametersMissing
	}
	req := pushViewRequest{
		TriggerID: triggerID,
		View:      view,
	}
	encoded, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp := &ViewResponse{}
	err = api.postJSONMethod(ctx, "views.push", api.token, encoded, resp)
	if err != nil {
		return nil, err
	}
	return resp, resp.Err()
}

// UpdateView updates an existing view.
// For more information see the UpdateViewContext documentation.
func (api *Client) UpdateView(view ModalViewRequest, externalID, hash, viewID string) (*ViewResponse, error) {
	return api.UpdateViewContext(context.Background(), view, externalID, hash, viewID)
}

// UpdateViewContext updates an existing view with a custom context.
// Slack API docs: https://docs.slack.dev/reference/methods/views.update
func (api *Client) UpdateViewContext(
	ctx context.Context,
	view ModalViewRequest,
	externalID, hash,
	viewID string,
) (*ViewResponse, error) {
	if externalID == "" && viewID == "" {
		return nil, ErrParametersMissing
	}
	req := updateViewRequest{
		View:       view,
		ExternalID: externalID,
		Hash:       hash,
		ViewID:     viewID,
	}
	encoded, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp := &ViewResponse{}
	err = api.postJSONMethod(ctx, "views.update", api.token, encoded, resp)
	if err != nil {
		return nil, err
	}
	return resp, resp.Err()
}
