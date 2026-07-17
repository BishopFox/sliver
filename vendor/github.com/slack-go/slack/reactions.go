package slack

import (
	"context"
	"net/url"
	"strconv"
)

// ItemReaction is the reactions that have happened on an item.
type ItemReaction struct {
	Name  string   `json:"name"`
	Count int      `json:"count"`
	Users []string `json:"users"`
}

// ReactedItem is an item that was reacted to, and the details of the
// reactions.
type ReactedItem struct {
	Item
	Reactions []ItemReaction
}

// GetReactionsParameters is the inputs to get reactions to an item.
type GetReactionsParameters struct {
	Full bool
}

// NewGetReactionsParameters initializes the inputs to get reactions to an item.
func NewGetReactionsParameters() GetReactionsParameters {
	return GetReactionsParameters{
		Full: false,
	}
}

type getReactionsResponseFull struct {
	Type    string
	Channel string `json:"channel,omitempty"` // channel is at the root level for message types
	M       struct {
		*Message // message structure already contains reactions
	} `json:"message"`
	F struct {
		*File
		Reactions []ItemReaction
	} `json:"file"`
	FC struct {
		*Comment
		Reactions []ItemReaction
	} `json:"comment"`
	SlackResponse
}

func (res getReactionsResponseFull) extractReactedItem() ReactedItem {
	item := ReactedItem{}
	item.Type = res.Type

	switch item.Type {
	case "message":
		item.Channel = res.Channel
		item.Message = res.M.Message
		item.Reactions = res.M.Reactions
	case "file":
		item.File = res.F.File
		item.Reactions = res.F.Reactions
	case "file_comment":
		item.File = res.F.File
		item.Comment = res.FC.Comment
		item.Reactions = res.FC.Reactions
	}
	return item
}

const (
	DEFAULT_REACTIONS_USER = ""
	DEFAULT_REACTIONS_FULL = false
)

// ListReactionsParameters is the inputs to find all reactions by a user.
type ListReactionsParameters struct {
	User   string
	TeamID string
	Cursor string
	Limit  int
	Full   bool
}

// NewListReactionsParameters initializes the inputs to find all reactions
// performed by a user.
func NewListReactionsParameters() ListReactionsParameters {
	return ListReactionsParameters{
		User: DEFAULT_REACTIONS_USER,
		Full: DEFAULT_REACTIONS_FULL,
	}
}

type listReactionsResponseFull struct {
	Items []struct {
		Type    string
		Channel string
		M       struct {
			*Message
		} `json:"message"`
		F struct {
			*File
			Reactions []ItemReaction
		} `json:"file"`
		FC struct {
			*Comment
			Reactions []ItemReaction
		} `json:"comment"`
	}
	SlackResponse
	ResponseMetadata `json:"response_metadata"`
}

func (res listReactionsResponseFull) extractReactedItems() []ReactedItem {
	items := make([]ReactedItem, len(res.Items))
	for i, input := range res.Items {
		item := ReactedItem{}
		item.Type = input.Type
		switch input.Type {
		case "message":
			item.Channel = input.Channel
			item.Message = input.M.Message
			item.Reactions = input.M.Reactions
		case "file":
			item.File = input.F.File
			item.Reactions = input.F.Reactions
		case "file_comment":
			item.File = input.F.File
			item.Comment = input.FC.Comment
			item.Reactions = input.FC.Reactions
		}
		items[i] = item
	}
	return items
}

// AddReaction adds a reaction emoji to a message, file or file comment.
// For more details, see AddReactionContext documentation.
func (api *Client) AddReaction(name string, item ItemRef) error {
	return api.AddReactionContext(context.Background(), name, item)
}

// AddReactionContext adds a reaction emoji to a message, file or file comment with a custom context.
// Slack API docs: https://api.slack.com/methods/reactions.add
func (api *Client) AddReactionContext(ctx context.Context, name string, item ItemRef) error {
	values := url.Values{
		"token": {api.token},
	}
	if name != "" {
		values.Set("name", name)
	}
	if item.Channel != "" {
		values.Set("channel", item.Channel)
	}
	if item.Timestamp != "" {
		values.Set("timestamp", item.Timestamp)
	}
	if item.File != "" {
		values.Set("file", item.File)
	}
	if item.Comment != "" {
		values.Set("file_comment", item.Comment)
	}

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "reactions.add", values, response); err != nil {
		return err
	}

	return response.Err()
}

// RemoveReaction removes a reaction emoji from a message, file or file comment.
// For more details, see RemoveReactionContext documentation.
func (api *Client) RemoveReaction(name string, item ItemRef) error {
	return api.RemoveReactionContext(context.Background(), name, item)
}

// RemoveReactionContext removes a reaction emoji from a message, file or file comment with a custom context.
// Slack API docs: https://api.slack.com/methods/reactions.remove
func (api *Client) RemoveReactionContext(ctx context.Context, name string, item ItemRef) error {
	values := url.Values{
		"token": {api.token},
	}
	if name != "" {
		values.Set("name", name)
	}
	if item.Channel != "" {
		values.Set("channel", item.Channel)
	}
	if item.Timestamp != "" {
		values.Set("timestamp", item.Timestamp)
	}
	if item.File != "" {
		values.Set("file", item.File)
	}
	if item.Comment != "" {
		values.Set("file_comment", item.Comment)
	}

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "reactions.remove", values, response); err != nil {
		return err
	}

	return response.Err()
}

// GetReactions returns item and details about the reactions on an item.
// For more details, see GetReactionsContext documentation.
func (api *Client) GetReactions(item ItemRef, params GetReactionsParameters) (ReactedItem, error) {
	return api.GetReactionsContext(context.Background(), item, params)
}

// GetReactionsContext returns item and details about the reactions on an item with a custom context.
// Slack API docs: https://api.slack.com/methods/reactions.get
func (api *Client) GetReactionsContext(ctx context.Context, item ItemRef, params GetReactionsParameters) (ReactedItem, error) {
	values := url.Values{
		"token": {api.token},
	}
	if item.Channel != "" {
		values.Set("channel", item.Channel)
	}
	if item.Timestamp != "" {
		values.Set("timestamp", item.Timestamp)
	}
	if item.File != "" {
		values.Set("file", item.File)
	}
	if item.Comment != "" {
		values.Set("file_comment", item.Comment)
	}
	if params.Full {
		values.Set("full", strconv.FormatBool(params.Full))
	}

	response := &getReactionsResponseFull{}
	if err := api.postMethod(ctx, "reactions.get", values, response); err != nil {
		return ReactedItem{}, err
	}

	if err := response.Err(); err != nil {
		return ReactedItem{}, err
	}

	return response.extractReactedItem(), nil
}

// ListReactions returns information about the items a user reacted to.
// For more details, see ListReactionsContext documentation.
func (api *Client) ListReactions(params ListReactionsParameters) ([]ReactedItem, string, error) {
	return api.ListReactionsContext(context.Background(), params)
}

// ListReactionsContext returns information about the items a user reacted to with a custom context.
// Slack API docs: https://api.slack.com/methods/reactions.list
func (api *Client) ListReactionsContext(ctx context.Context, params ListReactionsParameters) ([]ReactedItem, string, error) {
	values := url.Values{
		"token": {api.token},
	}
	if params.User != DEFAULT_REACTIONS_USER {
		values.Add("user", params.User)
	}
	if params.TeamID != "" {
		values.Add("team_id", params.TeamID)
	}
	if params.Cursor != "" {
		values.Add("cursor", params.Cursor)
	}
	if params.Limit != 0 {
		values.Add("limit", strconv.Itoa(params.Limit))
	}
	if params.Full {
		values.Add("full", strconv.FormatBool(params.Full))
	}

	response := &listReactionsResponseFull{}
	err := api.postMethod(ctx, "reactions.list", values, response)
	if err != nil {
		return nil, "", err
	}

	if err := response.Err(); err != nil {
		return nil, "", err
	}

	return response.extractReactedItems(), response.ResponseMetadata.Cursor, nil
}
