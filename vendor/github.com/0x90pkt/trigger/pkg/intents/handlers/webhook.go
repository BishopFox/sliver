// SPDX-License-Identifier: GPL-3.0-or-later

package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/0x90pkt/trigger/pkg/intents"
)

// WebhookSignatureHeader is the HTTP header carrying the hex-encoded
// HMAC-SHA256 of the request body. The webhook receiver verifies it
// using its copy of the per-webhook secret.
const WebhookSignatureHeader = "X-Trigger-Signature"

// WebhookConfig is the construction-time configuration for a Webhook
// handler.
//
// Security notes:
//
//   - URL must use http:// or https://. Schemes other than these are
//     rejected at construction time (no file://, ftp://, gopher://, ...).
//
//   - The destination host is locked at construction time. There is no
//     runtime URL templating — operators cannot exfiltrate intent
//     context to attacker-controlled URLs via crafted client IDs.
//
//   - TLS verification is ON by default. InsecureSkipVerify must be
//     explicitly set in config and is logged at WARN at construction.
//
//   - The HMAC secret is per-webhook, separate from the listener's
//     trigger shared secret. Operators provide it via SecretEnv (an
//     env-var name); the handler reads it at construction and holds
//     a private copy.
type WebhookConfig struct {
	// URL is the absolute http(s) URL to POST to. Locked at
	// construction; runtime intent fire only POSTs here.
	URL string
	// Secret is the per-webhook HMAC key used to sign request bodies.
	// Either Secret or SecretEnv must be set; SecretEnv wins if both.
	Secret string
	// SecretEnv is the name of an environment variable holding the
	// per-webhook HMAC key. Resolved at construction.
	SecretEnv string
	// Timeout is the per-request total deadline (connect + send +
	// receive). Default 5s if zero. Capped by the listener's per-
	// handler ctx deadline.
	Timeout time.Duration
	// InsecureSkipVerify, if true, disables TLS certificate
	// verification. EXPLICIT opt-in; logs WARN at construction.
	InsecureSkipVerify bool
	// Logger receives lifecycle and transport logs. nil → slog.Default().
	Logger *slog.Logger
}

// Webhook posts a signed JSON body to a fixed URL on intent fire. The
// body shape is: {intent, client_id, source_ip, nonce, timestamp}.
type Webhook struct {
	name    string
	url     string
	secret  []byte
	client  *http.Client
	logger  *slog.Logger
	timeout time.Duration
}

// NewWebhook constructs a Webhook handler from the given config.
func NewWebhook(name string, cfg WebhookConfig) (*Webhook, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("webhook handler: name must be set")
	}
	if cfg.URL == "" {
		return nil, errors.New("webhook handler: URL must be set")
	}
	parsed, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("webhook handler: invalid URL %q: %w", cfg.URL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("webhook handler: scheme %q not allowed (http or https only)", parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("webhook handler: URL %q missing host", cfg.URL)
	}

	secret := cfg.Secret
	if cfg.SecretEnv != "" {
		v, ok := os.LookupEnv(cfg.SecretEnv)
		if !ok || v == "" {
			return nil, fmt.Errorf("webhook handler: env %q not set or empty", cfg.SecretEnv)
		}
		secret = v
	}
	if secret == "" {
		return nil, errors.New("webhook handler: Secret or SecretEnv must be set")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.InsecureSkipVerify {
		logger.Warn("webhook handler created with InsecureSkipVerify=true",
			slog.String("intent", name),
			slog.String("url", cfg.URL),
		)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec // explicit operator opt-in
			MinVersion:         tls.VersionTLS12,
		},
		MaxIdleConns:        4,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     30 * time.Second,
	}
	return &Webhook{
		name:    name,
		url:     cfg.URL,
		secret:  []byte(secret),
		client:  &http.Client{Timeout: timeout, Transport: transport},
		logger:  logger,
		timeout: timeout,
	}, nil
}

// Name implements intents.Handler.
func (h *Webhook) Name() string { return h.name }

// webhookBody is the JSON document POSTed to the configured URL. Field
// order in the marshaled output is deterministic (Go encodes struct
// fields in declaration order).
type webhookBody struct {
	Intent    string `json:"intent"`
	ClientID  string `json:"client_id"`
	SourceIP  string `json:"source_ip"`
	Nonce     string `json:"nonce"`
	Timestamp string `json:"timestamp"`
}

// Execute implements intents.Handler. Synchronously POSTs the signed
// body and returns nil on 2xx response, error otherwise. The parent
// ctx deadline caps the wait.
func (h *Webhook) Execute(ctx context.Context, evt intents.Event) error {
	body := webhookBody{
		Intent:    evt.Intent,
		ClientID:  evt.ClientID,
		SourceIP:  evt.SourceIP,
		Nonce:     evt.Nonce,
		Timestamp: evt.Timestamp.UTC().Format(time.RFC3339Nano),
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("webhook %s: marshal: %w", h.name, err)
	}

	mac := hmac.New(sha256.New, h.secret)
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("webhook %s: build request: %w", h.name, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(WebhookSignatureHeader, signature)
	req.Header.Set("User-Agent", "trigger-webhook/1")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook %s: POST %s: %w", h.name, h.url, err)
	}
	defer func() {
		// Drain a small amount of the body to allow connection reuse,
		// then close.
		_, _ = io.CopyN(io.Discard, resp.Body, 4096)
		_ = resp.Body.Close()
	}()

	h.logger.Info("webhook handler complete",
		slog.String("intent", h.name),
		slog.String("client_id", evt.ClientID),
		slog.String("url", h.url),
		slog.Int("status", resp.StatusCode),
	)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook %s: %s returned status %d", h.name, h.url, resp.StatusCode)
	}
	return nil
}
