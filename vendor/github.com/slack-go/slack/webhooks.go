package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type WebhookMessage struct {
	Username        string       `json:"username,omitempty"`
	IconEmoji       string       `json:"icon_emoji,omitempty"`
	IconURL         string       `json:"icon_url,omitempty"`
	Channel         string       `json:"channel,omitempty"`
	ThreadTimestamp string       `json:"thread_ts,omitempty"`
	Text            string       `json:"text,omitempty"`
	Attachments     []Attachment `json:"attachments,omitempty"`
	Parse           string       `json:"parse,omitempty"`
	Blocks          *Blocks      `json:"blocks,omitempty"`
	ResponseType    string       `json:"response_type,omitempty"`
	ReplaceOriginal bool         `json:"replace_original"`
	DeleteOriginal  bool         `json:"delete_original"`
	ReplyBroadcast  bool         `json:"reply_broadcast,omitempty"`
	UnfurlLinks     bool         `json:"unfurl_links,omitempty"`
	UnfurlMedia     bool         `json:"unfurl_media,omitempty"`
}

func PostWebhook(url string, msg *WebhookMessage) error {
	return PostWebhookCustomHTTPContext(context.Background(), url, http.DefaultClient, msg)
}

func PostWebhookContext(ctx context.Context, url string, msg *WebhookMessage) error {
	return PostWebhookCustomHTTPContext(ctx, url, http.DefaultClient, msg)
}

func PostWebhookCustomHTTP(url string, httpClient *http.Client, msg *WebhookMessage) error {
	return PostWebhookCustomHTTPContext(context.Background(), url, httpClient, msg)
}

func PostWebhookCustomHTTPContext(ctx context.Context, url string, httpClient *http.Client, msg *WebhookMessage) error {
	raw, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("failed new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post webhook: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	return checkStatusCode(resp, discard{})
}
