### Notifications

Sliver can send server event notifications via the `github.com/nikoksr/notify` services. Notifications are configured in `server.yaml` under the `notifications` block (located at `~/.sliver/configs/server.yaml` by default).

Notifications are disabled by default. Enable them globally, then enable one or more services. You can optionally filter which events are sent globally or per-service.

#### Configuration Overview

- `notifications.enabled` - Master switch for notifications.
- `notifications.events` - Optional list of event types to send. Empty means all events.
- `notifications.services.<service>.enabled` - Enable a specific service.
- `notifications.services.<service>.events` - Optional per-service event filter. If empty, the global list is used.
- Any string field can reference an environment variable by using `$ENV_VAR` or `${ENV_VAR}`.

#### Example: Slack + Telegram with Event Filters

```yaml
notifications:
  enabled: true
  events:
    - session-connected
    - session-disconnected
  services:
    slack:
      enabled: true
      api_token: "$SLIVER_SLACK_TOKEN"
      channels:
        - "C0123456789"
    telegram:
      enabled: true
      api_token: "$SLIVER_TELEGRAM_TOKEN"
      chat_ids:
        - "123456789"
      events:
        - session-connected
```

#### Example: HTTP Webhooks + Syslog

```yaml
notifications:
  enabled: true
  services:
    http:
      enabled: true
      urls:
        - "https://hooks.example.com/sliver"
      webhooks:
        - url: "https://hooks.example.com/sliver-auth"
          method: "POST"
          content_type: "application/json"
          headers:
            Authorization: "$SLIVER_WEBHOOK_TOKEN"
    syslog:
      enabled: true
      priority: "info"
      network: "udp"
      address: "127.0.0.1:514"
      tag: "sliver"
```

#### Notes

- Environment variable expansion only applies to string fields (API keys, tokens, URLs, etc.).
- If a service configuration is incomplete, that service is skipped and the server continues without it.
