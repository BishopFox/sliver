package slack

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

// AssistantThreadSetStatusParameters are the parameters for AssistantThreadSetStatus
type AssistantThreadsSetStatusParameters struct {
	ChannelID       string   `json:"channel_id"`
	Status          string   `json:"status"`
	ThreadTS        string   `json:"thread_ts"`
	LoadingMessages []string `json:"loading_messages,omitempty"`
	Username        string   `json:"username,omitempty"`
	IconURL         string   `json:"icon_url,omitempty"`
	IconEmoji       string   `json:"icon_emoji,omitempty"`
}

// AssistantThreadSetTitleParameters are the parameters for AssistantThreadSetTitle
type AssistantThreadsSetTitleParameters struct {
	ChannelID string `json:"channel_id"`
	ThreadTS  string `json:"thread_ts"`
	Title     string `json:"title"`
}

// AssistantThreadSetSuggestedPromptsParameters are the parameters for AssistantThreadSetSuggestedPrompts
type AssistantThreadsSetSuggestedPromptsParameters struct {
	Title     string                   `json:"title"`
	ChannelID string                   `json:"channel_id"`
	ThreadTS  string                   `json:"thread_ts"`
	Prompts   []AssistantThreadsPrompt `json:"prompts"`
}

// AssistantThreadPrompt is a suggested prompt for a thread
type AssistantThreadsPrompt struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

// AssistantSearchContextParameters are the parameters for AssistantSearchContext
type AssistantSearchContextParameters struct {
	Query                   string   `json:"query"`
	ActionToken             string   `json:"action_token,omitempty"`
	ChannelTypes            []string `json:"channel_types,omitempty"`
	ContentTypes            []string `json:"content_types,omitempty"`
	ContextChannelID        string   `json:"context_channel_id,omitempty"`
	Cursor                  string   `json:"cursor,omitempty"`
	IncludeBots             bool     `json:"include_bots,omitempty"`
	Limit                   int      `json:"limit,omitempty"`
	IncludeDeletedUsers     bool     `json:"include_deleted_users,omitempty"`
	Before                  int64    `json:"before,omitempty"`
	After                   int64    `json:"after,omitempty"`
	IncludeContextMessages  bool     `json:"include_context_messages,omitempty"`
	Sort                    string   `json:"sort,omitempty"`
	SortDir                 string   `json:"sort_dir,omitempty"`
	IncludeMessageBlocks    bool     `json:"include_message_blocks,omitempty"`
	Highlight               bool     `json:"highlight,omitempty"`
	TermClauses             []string `json:"term_clauses,omitempty"`
	Modifiers               string   `json:"modifiers,omitempty"`
	IncludeArchivedChannels bool     `json:"include_archived_channels,omitempty"`
	DisableSemanticSearch   bool     `json:"disable_semantic_search,omitempty"`
}

// AssistantSearchContextMessage represents a search result message
type AssistantSearchContextMessage struct {
	AuthorUserID    string                                `json:"author_user_id"`
	AuthorName      string                                `json:"author_name,omitempty"`
	TeamID          string                                `json:"team_id"`
	ChannelID       string                                `json:"channel_id"`
	ChannelName     string                                `json:"channel_name,omitempty"`
	MessageTS       string                                `json:"message_ts"`
	Content         string                                `json:"content"`
	IsAuthorBot     bool                                  `json:"is_author_bot"`
	Permalink       string                                `json:"permalink"`
	Blocks          Blocks                                `json:"blocks,omitempty"`
	ContextMessages *AssistantSearchContextMessageContext `json:"context_messages,omitempty"`
}

// AssistantSearchContextMessageContext contains context messages surrounding a search result
type AssistantSearchContextMessageContext struct {
	Before []AssistantSearchContextMessage `json:"before"`
	After  []AssistantSearchContextMessage `json:"after"`
}

// AssistantSearchContextFile represents a search result file
type AssistantSearchContextFile struct {
	UploaderUserID string `json:"uploader_user_id"`
	AuthorUserID   string `json:"author_user_id"`
	AuthorName     string `json:"author_name"`
	TeamID         string `json:"team_id"`
	FileID         string `json:"file_id"`
	DateCreated    int64  `json:"date_created"`
	DateUpdated    int64  `json:"date_updated"`
	Title          string `json:"title"`
	FileType       string `json:"file_type"`
	Permalink      string `json:"permalink"`
	Content        string `json:"content"`
}

// AssistantSearchContextChannel represents a search result channel
type AssistantSearchContextChannel struct {
	TeamID        string `json:"team_id"`
	CreatorUserID string `json:"creator_user_id"`
	CreatorName   string `json:"creator_name"`
	DateCreated   int64  `json:"date_created"`
	DateUpdated   int64  `json:"date_updated"`
	Name          string `json:"name"`
	Topic         string `json:"topic"`
	Purpose       string `json:"purpose"`
	Permalink     string `json:"permalink"`
}

// AssistantSearchContextResults contains the search results
type AssistantSearchContextResults struct {
	Messages []AssistantSearchContextMessage `json:"messages,omitempty"`
	Files    []AssistantSearchContextFile    `json:"files,omitempty"`
	Channels []AssistantSearchContextChannel `json:"channels,omitempty"`
}

// AssistantSearchContextResponse is the response from assistant.search.context
type AssistantSearchContextResponse struct {
	SlackResponse
	Results          AssistantSearchContextResults `json:"results"`
	ResponseMetadata struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
}

// AssistantThreadSetSuggestedPrompts sets the suggested prompts for a thread
func (p *AssistantThreadsSetSuggestedPromptsParameters) AddPrompt(title, message string) {
	p.Prompts = append(p.Prompts, AssistantThreadsPrompt{
		Title:   title,
		Message: message,
	})
}

// SetAssistantThreadsSugesstedPrompts sets the suggested prompts for a thread
// @see https://api.slack.com/methods/assistant.threads.setSuggestedPrompts
func (api *Client) SetAssistantThreadsSuggestedPrompts(params AssistantThreadsSetSuggestedPromptsParameters) (err error) {
	return api.SetAssistantThreadsSuggestedPromptsContext(context.Background(), params)
}

// SetAssistantThreadSuggestedPromptsContext sets the suggested prompts for a thread with a custom context
// @see https://api.slack.com/methods/assistant.threads.setSuggestedPrompts
func (api *Client) SetAssistantThreadsSuggestedPromptsContext(ctx context.Context, params AssistantThreadsSetSuggestedPromptsParameters) (err error) {

	values := url.Values{
		"token": {api.token},
	}

	if params.ThreadTS != "" {
		values.Add("thread_ts", params.ThreadTS)
	}

	values.Add("channel_id", params.ChannelID)

	if params.Title != "" {
		values.Add("title", params.Title)
	}

	// Send Prompts as JSON
	prompts, err := json.Marshal(params.Prompts)
	if err != nil {
		return err
	}

	values.Add("prompts", string(prompts))

	response := struct {
		SlackResponse
	}{}

	err = api.postMethod(ctx, "assistant.threads.setSuggestedPrompts", values, &response)
	if err != nil {
		return
	}

	return response.Err()
}

// SetAssistantThreadsStatus sets the status of a thread.
// This method accepts either the chat:write or assistant:write scope.
// Note: the assistant:write scope is being deprecated in favor of chat:write.
// @see https://api.slack.com/methods/assistant.threads.setStatus
func (api *Client) SetAssistantThreadsStatus(params AssistantThreadsSetStatusParameters) (err error) {
	return api.SetAssistantThreadsStatusContext(context.Background(), params)
}

// SetAssistantThreadsStatusContext sets the status of a thread with a custom context.
// This method accepts either the chat:write or assistant:write scope.
// Note: the assistant:write scope is being deprecated in favor of chat:write.
// @see https://api.slack.com/methods/assistant.threads.setStatus
func (api *Client) SetAssistantThreadsStatusContext(ctx context.Context, params AssistantThreadsSetStatusParameters) (err error) {

	values := url.Values{
		"token": {api.token},
	}

	if params.ThreadTS != "" {
		values.Add("thread_ts", params.ThreadTS)
	}

	values.Add("channel_id", params.ChannelID)

	// Always send the status parameter, if empty, it will clear any existing status
	values.Add("status", params.Status)

	if len(params.LoadingMessages) > 0 {
		values.Add("loading_messages", strings.Join(params.LoadingMessages, ","))
	}

	if params.Username != "" {
		values.Add("username", params.Username)
	}

	if params.IconURL != "" {
		values.Add("icon_url", params.IconURL)
	}

	if params.IconEmoji != "" {
		values.Add("icon_emoji", params.IconEmoji)
	}

	response := struct {
		SlackResponse
	}{}

	err = api.postMethod(ctx, "assistant.threads.setStatus", values, &response)
	if err != nil {
		return
	}

	return response.Err()
}

// SetAssistantThreadsTitle sets the title of a thread
// @see https://api.slack.com/methods/assistant.threads.setTitle
func (api *Client) SetAssistantThreadsTitle(params AssistantThreadsSetTitleParameters) (err error) {
	return api.SetAssistantThreadsTitleContext(context.Background(), params)
}

// SetAssistantThreadsTitleContext sets the title of a thread with a custom context
// @see https://api.slack.com/methods/assistant.threads.setTitle
func (api *Client) SetAssistantThreadsTitleContext(ctx context.Context, params AssistantThreadsSetTitleParameters) (err error) {

	values := url.Values{
		"token": {api.token},
	}

	if params.ChannelID != "" {
		values.Add("channel_id", params.ChannelID)
	}

	if params.ThreadTS != "" {
		values.Add("thread_ts", params.ThreadTS)
	}

	if params.Title != "" {
		values.Add("title", params.Title)
	}

	response := struct {
		SlackResponse
	}{}

	err = api.postMethod(ctx, "assistant.threads.setTitle", values, &response)
	if err != nil {
		return
	}

	return response.Err()

}

// SearchAssistantContext searches messages across the Slack organization
// @see https://api.slack.com/methods/assistant.search.context
func (api *Client) SearchAssistantContext(params AssistantSearchContextParameters) (*AssistantSearchContextResponse, error) {
	return api.SearchAssistantContextContext(context.Background(), params)
}

// SearchAssistantContextContext searches messages across the Slack organization with a custom context
// @see https://api.slack.com/methods/assistant.search.context
func (api *Client) SearchAssistantContextContext(ctx context.Context, params AssistantSearchContextParameters) (*AssistantSearchContextResponse, error) {
	values := url.Values{
		"token": {api.token},
	}

	values.Add("query", params.Query)

	if params.ActionToken != "" {
		values.Add("action_token", params.ActionToken)
	}

	if len(params.ChannelTypes) > 0 {
		values.Add("channel_types", strings.Join(params.ChannelTypes, ","))
	}

	if len(params.ContentTypes) > 0 {
		values.Add("content_types", strings.Join(params.ContentTypes, ","))
	}

	if params.ContextChannelID != "" {
		values.Add("context_channel_id", params.ContextChannelID)
	}

	if params.Cursor != "" {
		values.Add("cursor", params.Cursor)
	}

	if params.IncludeBots {
		values.Add("include_bots", "true")
	}

	if params.Limit > 0 {
		values.Add("limit", strconv.Itoa(params.Limit))
	}

	if params.IncludeDeletedUsers {
		values.Add("include_deleted_users", "true")
	}

	if params.Before > 0 {
		values.Add("before", strconv.FormatInt(params.Before, 10))
	}

	if params.After > 0 {
		values.Add("after", strconv.FormatInt(params.After, 10))
	}

	if params.IncludeContextMessages {
		values.Add("include_context_messages", "true")
	}

	if params.Sort != "" {
		values.Add("sort", params.Sort)
	}

	if params.SortDir != "" {
		values.Add("sort_dir", params.SortDir)
	}

	if params.IncludeMessageBlocks {
		values.Add("include_message_blocks", "true")
	}

	if params.Highlight {
		values.Add("highlight", "true")
	}

	if len(params.TermClauses) > 0 {
		values.Add("term_clauses", strings.Join(params.TermClauses, ","))
	}

	if params.Modifiers != "" {
		values.Add("modifiers", params.Modifiers)
	}

	if params.IncludeArchivedChannels {
		values.Add("include_archived_channels", "true")
	}

	if params.DisableSemanticSearch {
		values.Add("disable_semantic_search", "true")
	}

	response := &AssistantSearchContextResponse{}

	err := api.postMethod(ctx, "assistant.search.context", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}
