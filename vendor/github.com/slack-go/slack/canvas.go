package slack

import (
	"context"
	"encoding/json"
	"net/url"
)

type CanvasDetails struct {
	CanvasID string `json:"canvas_id"`
}

type DocumentContent struct {
	Type     string `json:"type"`
	Markdown string `json:"markdown,omitempty"`
}

type CanvasChange struct {
	Operation       string          `json:"operation"`
	SectionID       string          `json:"section_id,omitempty"`
	DocumentContent DocumentContent `json:"document_content"`
}

type EditCanvasParams struct {
	CanvasID string         `json:"canvas_id"`
	Changes  []CanvasChange `json:"changes"`
}

type SetCanvasAccessParams struct {
	CanvasID    string   `json:"canvas_id"`
	AccessLevel string   `json:"access_level"`
	ChannelIDs  []string `json:"channel_ids,omitempty"`
	UserIDs     []string `json:"user_ids,omitempty"`
}

type DeleteCanvasAccessParams struct {
	CanvasID   string   `json:"canvas_id"`
	ChannelIDs []string `json:"channel_ids,omitempty"`
	UserIDs    []string `json:"user_ids,omitempty"`
}

type LookupCanvasSectionsCriteria struct {
	SectionTypes []string `json:"section_types,omitempty"`
	ContainsText string   `json:"contains_text,omitempty"`
}

type LookupCanvasSectionsParams struct {
	CanvasID string                       `json:"canvas_id"`
	Criteria LookupCanvasSectionsCriteria `json:"criteria"`
}

type CanvasSection struct {
	ID string `json:"id"`
}

type LookupCanvasSectionsResponse struct {
	SlackResponse
	Sections []CanvasSection `json:"sections"`
}

// CreateCanvas creates a new canvas.
// For more details, see CreateCanvasContext documentation.
func (api *Client) CreateCanvas(title string, documentContent DocumentContent) (string, error) {
	return api.CreateCanvasContext(context.Background(), title, documentContent)
}

// CreateCanvasContext creates a new canvas with a custom context.
// Slack API docs: https://api.slack.com/methods/canvases.create
func (api *Client) CreateCanvasContext(ctx context.Context, title string, documentContent DocumentContent) (string, error) {
	values := url.Values{
		"token": {api.token},
	}
	if title != "" {
		values.Add("title", title)
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

	err := api.postMethod(ctx, "canvases.create", values, &response)
	if err != nil {
		return "", err
	}

	return response.CanvasID, response.Err()
}

// DeleteCanvas deletes an existing canvas.
// For more details, see DeleteCanvasContext documentation.
func (api *Client) DeleteCanvas(canvasID string) error {
	return api.DeleteCanvasContext(context.Background(), canvasID)
}

// DeleteCanvasContext deletes an existing canvas with a custom context.
// Slack API docs: https://api.slack.com/methods/canvases.delete
func (api *Client) DeleteCanvasContext(ctx context.Context, canvasID string) error {
	values := url.Values{
		"token":     {api.token},
		"canvas_id": {canvasID},
	}

	response := struct {
		SlackResponse
	}{}

	err := api.postMethod(ctx, "canvases.delete", values, &response)
	if err != nil {
		return err
	}

	return response.Err()
}

// EditCanvas edits an existing canvas.
// For more details, see EditCanvasContext documentation.
func (api *Client) EditCanvas(params EditCanvasParams) error {
	return api.EditCanvasContext(context.Background(), params)
}

// EditCanvasContext edits an existing canvas with a custom context.
// Slack API docs: https://api.slack.com/methods/canvases.edit
func (api *Client) EditCanvasContext(ctx context.Context, params EditCanvasParams) error {
	values := url.Values{
		"token":     {api.token},
		"canvas_id": {params.CanvasID},
	}

	changesJSON, err := json.Marshal(params.Changes)
	if err != nil {
		return err
	}
	values.Add("changes", string(changesJSON))

	response := struct {
		SlackResponse
	}{}

	err = api.postMethod(ctx, "canvases.edit", values, &response)
	if err != nil {
		return err
	}

	return response.Err()
}

// SetCanvasAccess sets the access level to a canvas for specified entities.
// For more details, see SetCanvasAccessContext documentation.
func (api *Client) SetCanvasAccess(params SetCanvasAccessParams) error {
	return api.SetCanvasAccessContext(context.Background(), params)
}

// SetCanvasAccessContext sets the access level to a canvas for specified entities with a custom context.
// Slack API docs: https://api.slack.com/methods/canvases.access.set
func (api *Client) SetCanvasAccessContext(ctx context.Context, params SetCanvasAccessParams) error {
	values := url.Values{
		"token":        {api.token},
		"canvas_id":    {params.CanvasID},
		"access_level": {params.AccessLevel},
	}
	if len(params.ChannelIDs) > 0 {
		channelIDsJSON, err := json.Marshal(params.ChannelIDs)
		if err != nil {
			return err
		}
		values.Add("channel_ids", string(channelIDsJSON))
	}
	if len(params.UserIDs) > 0 {
		userIDsJSON, err := json.Marshal(params.UserIDs)
		if err != nil {
			return err
		}
		values.Add("user_ids", string(userIDsJSON))
	}

	response := struct {
		SlackResponse
	}{}

	err := api.postMethod(ctx, "canvases.access.set", values, &response)
	if err != nil {
		return err
	}

	return response.Err()
}

// DeleteCanvasAccess removes access to a canvas for specified entities.
// For more details, see DeleteCanvasAccessContext documentation.
func (api *Client) DeleteCanvasAccess(params DeleteCanvasAccessParams) error {
	return api.DeleteCanvasAccessContext(context.Background(), params)
}

// DeleteCanvasAccessContext removes access to a canvas for specified entities with a custom context.
// Slack API docs: https://api.slack.com/methods/canvases.access.delete
func (api *Client) DeleteCanvasAccessContext(ctx context.Context, params DeleteCanvasAccessParams) error {
	values := url.Values{
		"token":     {api.token},
		"canvas_id": {params.CanvasID},
	}
	if len(params.ChannelIDs) > 0 {
		channelIDsJSON, err := json.Marshal(params.ChannelIDs)
		if err != nil {
			return err
		}
		values.Add("channel_ids", string(channelIDsJSON))
	}
	if len(params.UserIDs) > 0 {
		userIDsJSON, err := json.Marshal(params.UserIDs)
		if err != nil {
			return err
		}
		values.Add("user_ids", string(userIDsJSON))
	}

	response := struct {
		SlackResponse
	}{}

	err := api.postMethod(ctx, "canvases.access.delete", values, &response)
	if err != nil {
		return err
	}

	return response.Err()
}

// LookupCanvasSections finds sections matching the provided criteria.
// For more details, see LookupCanvasSectionsContext documentation.
func (api *Client) LookupCanvasSections(params LookupCanvasSectionsParams) ([]CanvasSection, error) {
	return api.LookupCanvasSectionsContext(context.Background(), params)
}

// LookupCanvasSectionsContext finds sections matching the provided criteria with a custom context.
// Slack API docs: https://api.slack.com/methods/canvases.sections.lookup
func (api *Client) LookupCanvasSectionsContext(ctx context.Context, params LookupCanvasSectionsParams) ([]CanvasSection, error) {
	values := url.Values{
		"token":     {api.token},
		"canvas_id": {params.CanvasID},
	}

	criteriaJSON, err := json.Marshal(params.Criteria)
	if err != nil {
		return nil, err
	}
	values.Add("criteria", string(criteriaJSON))

	response := LookupCanvasSectionsResponse{}

	err = api.postMethod(ctx, "canvases.sections.lookup", values, &response)
	if err != nil {
		return nil, err
	}

	return response.Sections, response.Err()
}
