package slack

import (
	"context"
	"net/url"
)

// SocketModeConnection contains various details about the SocketMode connection.
// It is returned by an "apps.connections.open" API call.
type SocketModeConnection struct {
	URL  string                 `json:"url,omitempty"`
	Data map[string]interface{} `json:"-"`
}

type openResponseFull struct {
	SlackResponse
	SocketModeConnection
}

// StartSocketModeContext calls the "apps.connections.open" endpoint and returns the provided URL and the full Info block with a custom context.
//
// To have a fully managed Socket Mode connection, use `socketmode.New()`, and call `Run()` on it.
func (api *Client) StartSocketModeContext(ctx context.Context) (info *SocketModeConnection, websocketURL string, err error) {
	response := &openResponseFull{}
	err = postJSON(ctx, api.httpclient, api.endpoint+"apps.connections.open", api.appLevelToken, nil, response, api)
	if err != nil {
		return nil, "", err
	}

	if response.Err() == nil {
		api.Debugln("Using URL:", response.SocketModeConnection.URL)
	}

	// According to the API documentation at https://api.slack.com/apis/socket-mode, we
	// can add a query parameter `debug_reconnects=true` to the URL to make the connection
	// time significantly shorter (360 seconds).
	if api.debug {
		u, _ := url.Parse(response.SocketModeConnection.URL)
		u.Query().Add("debug_reconnects", "true")
		response.SocketModeConnection.URL = u.String()
	}
	return &response.SocketModeConnection, response.SocketModeConnection.URL, response.Err()
}
