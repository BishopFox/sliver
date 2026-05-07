package slack

import (
	"context"
	"net/url"
)

// Bot contains information about a bot
type Bot struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Deleted bool     `json:"deleted"`
	UserID  string   `json:"user_id"`
	AppID   string   `json:"app_id"`
	Updated JSONTime `json:"updated"`
	Icons   Icons    `json:"icons"`
}

type botResponseFull struct {
	Bot `json:"bot,omitempty"` // GetBotInfo
	SlackResponse
}

func (api *Client) botRequest(ctx context.Context, path string, values url.Values) (*botResponseFull, error) {
	response := &botResponseFull{}
	err := api.postMethod(ctx, path, values, response)
	if err != nil {
		return nil, err
	}

	if err := response.Err(); err != nil {
		return nil, err
	}

	return response, nil
}

type GetBotInfoParameters struct {
	Bot    string
	TeamID string
}

// GetBotInfo will retrieve the complete bot information.
// For more details, see GetBotInfoContext documentation.
func (api *Client) GetBotInfo(parameters GetBotInfoParameters) (*Bot, error) {
	return api.GetBotInfoContext(context.Background(), parameters)
}

// GetBotInfoContext will retrieve the complete bot information using a custom context.
// Slack API docs: https://api.slack.com/methods/bots.info
func (api *Client) GetBotInfoContext(ctx context.Context, parameters GetBotInfoParameters) (*Bot, error) {
	values := url.Values{
		"token": {api.token},
	}

	if parameters.Bot != "" {
		values.Add("bot", parameters.Bot)
	}

	if parameters.TeamID != "" {
		values.Add("team_id", parameters.TeamID)
	}

	response, err := api.botRequest(ctx, "bots.info", values)
	if err != nil {
		return nil, err
	}
	return &response.Bot, nil
}
