package mtypes

import (
	"github.com/mailgun/mailgun-go/v5/events"
)

type UrlOrUrls struct {
	Urls []string `json:"urls"`
	Url  string   `json:"url"`
}

type WebHooksListResponse struct {
	Webhooks map[string]UrlOrUrls `json:"webhooks"`
}

type WebHookResponse struct {
	Webhook UrlOrUrls `json:"webhook"`
}

// Signature represents the signature portion of the webhook POST body
type Signature struct {
	TimeStamp string `json:"timestamp"`
	Token     string `json:"token"`
	Signature string `json:"signature"`
}

// WebhookPayload represents the JSON payload provided when a Webhook is called by mailgun
type WebhookPayload struct {
	Signature Signature      `json:"signature"`
	EventData events.RawJSON `json:"event-data"`
}
