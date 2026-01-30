package slack

import (
	"context"
	"encoding/json"
	"net/url"
)

// AssistantThreadSetStatusParameters are the parameters for AssistantThreadSetStatus
type AssistantThreadsSetStatusParameters struct {
	ChannelID string `json:"channel_id"`
	Status    string `json:"status"`
	ThreadTS  string `json:"thread_ts"`
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

// SetAssistantThreadStatus sets the status of a thread
// @see https://api.slack.com/methods/assistant.threads.setStatus
func (api *Client) SetAssistantThreadsStatus(params AssistantThreadsSetStatusParameters) (err error) {
	return api.SetAssistantThreadsStatusContext(context.Background(), params)
}

// SetAssistantThreadStatusContext sets the status of a thread with a custom context
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
