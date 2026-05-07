package notifications

import (
	"testing"

	"github.com/bishopfox/sliver/server/configs"
)

func TestResolveEnvString(t *testing.T) {
	t.Setenv("SLIVER_NOTIFY_TEST", "secret")

	got, ok, found := resolveEnvString("$SLIVER_NOTIFY_TEST")
	if !ok || !found || got != "secret" {
		t.Fatalf("expected env replacement, got %q (ok=%v)", got, ok)
	}

	got, ok, found = resolveEnvString("${SLIVER_NOTIFY_TEST}")
	if !ok || !found || got != "secret" {
		t.Fatalf("expected env replacement with braces, got %q (ok=%v)", got, ok)
	}

	got, ok, found = resolveEnvString("plain")
	if ok || found || got != "plain" {
		t.Fatalf("expected literal value, got %q (ok=%v)", got, ok)
	}

	got, ok, found = resolveEnvString("$SLIVER_NOTIFY_MISSING")
	if !ok || found || got != "" {
		t.Fatalf("expected empty string for missing env, got %q (ok=%v)", got, ok)
	}
}

func TestExpandEnvConfig(t *testing.T) {
	t.Setenv("SLIVER_NOTIFY_SLACK_TOKEN", "slack-secret")
	t.Setenv("SLIVER_NOTIFY_SLACK_CHANNEL", "C123")
	t.Setenv("SLIVER_NOTIFY_WEBHOOK", "https://example.com/hook")
	t.Setenv("SLIVER_NOTIFY_HEADER", "Bearer 123")

	cfg := &configs.NotificationsConfig{
		Enabled: true,
		Services: &configs.NotificationsServicesConfig{
			Slack: &configs.SlackConfig{
				NotificationServiceConfig: configs.NotificationServiceConfig{Enabled: true},
				APIToken:                  "$SLIVER_NOTIFY_SLACK_TOKEN",
				Channels:                  []string{"$SLIVER_NOTIFY_SLACK_CHANNEL"},
			},
			HTTP: &configs.HTTPConfig{
				NotificationServiceConfig: configs.NotificationServiceConfig{Enabled: true},
				Webhooks: []configs.HTTPWebhookConfig{
					{
						URL: "$SLIVER_NOTIFY_WEBHOOK",
						Headers: map[string]string{
							"Authorization": "$SLIVER_NOTIFY_HEADER",
						},
					},
				},
			},
		},
	}

	expandEnv(cfg)

	if cfg.Services.Slack.APIToken != "slack-secret" {
		t.Fatalf("expected slack token to expand, got %q", cfg.Services.Slack.APIToken)
	}
	if len(cfg.Services.Slack.Channels) != 1 || cfg.Services.Slack.Channels[0] != "C123" {
		t.Fatalf("unexpected slack channels: %v", cfg.Services.Slack.Channels)
	}
	if cfg.Services.HTTP.Webhooks[0].URL != "https://example.com/hook" {
		t.Fatalf("expected webhook url to expand, got %q", cfg.Services.HTTP.Webhooks[0].URL)
	}
	if cfg.Services.HTTP.Webhooks[0].Headers["Authorization"] != "Bearer 123" {
		t.Fatalf("expected header to expand, got %q", cfg.Services.HTTP.Webhooks[0].Headers["Authorization"])
	}
}
