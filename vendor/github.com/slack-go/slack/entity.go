package slack

import (
	"context"
	"encoding/json"
	"net/url"
)

// EntityPresentDetailsParameters contains the parameters for entity.presentDetails API method
type EntityPresentDetailsParameters struct {
	TriggerID        string                 `json:"trigger_id"`
	Metadata         *EntityDetailsMetadata `json:"metadata,omitempty"`
	Error            *EntityDetailsError    `json:"error,omitempty"`
	UserAuthRequired bool                   `json:"user_auth_required,omitempty"`
	UserAuthURL      string                 `json:"user_auth_url,omitempty"`
	UserAuthMessage  string                 `json:"user_auth_message,omitempty"`
}

// EntityDetailsMetadata represents the metadata for entity details
type EntityDetailsMetadata struct {
	EntityType    string                 `json:"entity_type"`
	URL           string                 `json:"url,omitempty"`
	ExternalRef   WorkObjectExternalRef  `json:"external_ref,omitempty"`
	EntityPayload map[string]interface{} `json:"entity_payload"`
}

// EntityDetailsError represents an error response for entity details
type EntityDetailsError struct {
	Status        string                `json:"status"`
	CustomTitle   string                `json:"custom_title,omitempty"`
	CustomMessage string                `json:"custom_message,omitempty"`
	MessageFormat string                `json:"message_format,omitempty"`
	Actions       []EntityDetailsAction `json:"actions,omitempty"`
}

// EntityDetailsAction represents an action button in entity details error
type EntityDetailsAction struct {
	Text            string                        `json:"text"`
	ActionID        string                        `json:"action_id"`
	Value           string                        `json:"value,omitempty"`
	Style           string                        `json:"style,omitempty"`
	URL             string                        `json:"url,omitempty"`
	ProcessingState *EntityDetailsProcessingState `json:"processing_state,omitempty"`
}

// EntityDetailsProcessingState represents the processing state of an action
type EntityDetailsProcessingState struct {
	Enabled bool `json:"enabled"`
}

// EntityPresentDetailsResponse represents the response from entity.presentDetails
type EntityPresentDetailsResponse struct {
	SlackResponse
}

// EntityPresentDetails presents entity details in the flexpane
// For more details, see EntityPresentDetailsContext documentation.
func (api *Client) EntityPresentDetails(params EntityPresentDetailsParameters) error {
	return api.EntityPresentDetailsContext(context.Background(), params)
}

// EntityPresentDetailsContext presents entity details in the flexpane with a custom context.
// Slack API docs: https://docs.slack.dev/reference/methods/entity.presentDetails
func (api *Client) EntityPresentDetailsContext(ctx context.Context, params EntityPresentDetailsParameters) error {
	values := url.Values{
		"token":      {api.token},
		"trigger_id": {params.TriggerID},
	}

	// Add metadata if provided
	if params.Metadata != nil {
		metadataJSON, err := json.Marshal(params.Metadata)
		if err != nil {
			return err
		}
		values.Set("metadata", string(metadataJSON))
	}

	// Add error if provided
	if params.Error != nil {
		errorJSON, err := json.Marshal(params.Error)
		if err != nil {
			return err
		}
		values.Set("error", string(errorJSON))
	}

	// Add user auth parameters if provided
	if params.UserAuthRequired {
		values.Set("user_auth_required", "true")
	}
	if params.UserAuthURL != "" {
		values.Set("user_auth_url", params.UserAuthURL)
	}
	if params.UserAuthMessage != "" {
		values.Set("user_auth_message", params.UserAuthMessage)
	}

	response := &EntityPresentDetailsResponse{}
	err := api.postMethod(ctx, "entity.presentDetails", values, response)
	if err != nil {
		return err
	}

	return response.Err()
}

// EntityPresentDetailsWithMetadata is a convenience method for presenting entity details with metadata
func (api *Client) EntityPresentDetailsWithMetadata(triggerID string, metadata EntityDetailsMetadata) error {
	return api.EntityPresentDetailsContext(context.Background(), EntityPresentDetailsParameters{
		TriggerID: triggerID,
		Metadata:  &metadata,
	})
}

// EntityPresentDetailsWithError is a convenience method for presenting entity details with an error
func (api *Client) EntityPresentDetailsWithError(triggerID string, errPayload EntityDetailsError) error {
	return api.EntityPresentDetailsContext(context.Background(), EntityPresentDetailsParameters{
		TriggerID: triggerID,
		Error:     &errPayload,
	})
}

// EntityPresentDetailsWithAuth is a convenience method for presenting entity details with authentication required
func (api *Client) EntityPresentDetailsWithAuth(triggerID, authURL, authMessage string) error {
	return api.EntityPresentDetailsContext(context.Background(), EntityPresentDetailsParameters{
		TriggerID:        triggerID,
		UserAuthRequired: true,
		UserAuthURL:      authURL,
		UserAuthMessage:  authMessage,
	})
}
