package linenotify

import "net/http"

// Client calls LINE Notify API. refs https://notify-bot.line.me/doc/
type Client struct {
	HTTPClient *http.Client
}

// NewClient returns *Client
func NewClient() *Client {
	return &Client{HTTPClient: http.DefaultClient}
}
