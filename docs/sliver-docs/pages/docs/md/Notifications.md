### Notifications

#### Table of Contents

- [Overview](#overview)
- [Configuration Overview](#configuration-overview)
- [Global Examples](#global-examples)
- [Service Examples](#service-examples)
  - [Amazon SES](#amazon-ses)
  - [Amazon SNS](#amazon-sns)
  - [Bark](#bark)
  - [DingTalk (DingDing)](#dingtalk-dingding)
  - [Discord](#discord)
  - [FCM](#fcm)
  - [Google Chat](#google-chat)
  - [HTTP Webhooks](#http-webhooks)
  - [Lark](#lark)
  - [Line](#line)
  - [Line Notify](#line-notify)
  - [Mail (SMTP)](#mail-smtp)
  - [Mailgun](#mailgun)
  - [Matrix](#matrix)
  - [Mattermost](#mattermost)
  - [Microsoft Teams](#microsoft-teams)
  - [PagerDuty](#pagerduty)
  - [Plivo](#plivo)
  - [Pushbullet](#pushbullet)
  - [Pushbullet SMS](#pushbullet-sms)
  - [Pushover](#pushover)
  - [Reddit](#reddit)
  - [Rocket.Chat](#rocketchat)
  - [SendGrid](#sendgrid)
  - [Slack](#slack)
  - [Syslog](#syslog)
  - [Telegram](#telegram)
  - [TextMagic](#textmagic)
  - [Twilio](#twilio)
  - [Twitter](#twitter)
  - [Viber](#viber)
  - [Web Push](#web-push)
  - [WeChat](#wechat)
  - [WhatsApp](#whatsapp)
- [Notes](#notes)

#### Overview

Sliver can send server event notifications via the `github.com/nikoksr/notify` services. Notifications are configured in `server.yaml` under the `notifications` block (located at `~/.sliver/configs/server.yaml` by default).

Notifications are disabled by default. Enable them globally, then enable one or more services. You can optionally filter which events are sent globally or per-service.

#### Configuration Overview

- `notifications.enabled` - Master switch for notifications.
- `notifications.events` - Optional list of event types to send. Empty means all events.
- `notifications.services.<service>.enabled` - Enable a specific service.
- `notifications.services.<service>.events` - Optional per-service event filter. If empty, the global list is used.
- Any string field can reference an environment variable by using `$ENV_VAR` or `${ENV_VAR}`.

#### Global Examples

##### Example: Slack + Telegram with Event Filters

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

##### Example: HTTP Webhooks + Syslog

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

#### Service Examples

Each example below shows the service-specific block under `notifications.services`. All examples assume `notifications.enabled: true`.

##### Amazon SES

```yaml
services:
  amazon_ses:
    enabled: true
    access_key_id: "$AWS_ACCESS_KEY_ID"
    secret_key: "$AWS_SECRET_ACCESS_KEY"
    region: "us-east-1"
    sender_address: "alerts@example.com"
    receivers:
      - "ops@example.com"
```

##### Amazon SNS

```yaml
services:
  amazon_sns:
    enabled: true
    access_key_id: "$AWS_ACCESS_KEY_ID"
    secret_key: "$AWS_SECRET_ACCESS_KEY"
    region: "us-east-1"
    receivers:
      - "https://sns.us-east-1.amazonaws.com/123456789012/my-topic"
```

##### Bark

```yaml
services:
  bark:
    enabled: true
    device_key: "$BARK_DEVICE_KEY"
    servers:
      - "https://api.day.app/"
```

##### DingTalk (DingDing)

```yaml
services:
  dingding:
    enabled: true
    token: "$DINGDING_TOKEN"
    secret: "$DINGDING_SECRET"
```

##### Discord

```yaml
services:
  discord:
    enabled: true
    token: "$DISCORD_BOT_TOKEN"
    token_type: "bot"  # bot (default) or oauth
    channels:
      - "123456789012345678"
```

##### FCM

```yaml
services:
  fcm:
    enabled: true
    credentials_file: "/path/to/firebase.json"
    project_id: "my-firebase-project"
    device_tokens:
      - "$FCM_DEVICE_TOKEN"
```

##### Google Chat

```yaml
services:
  google_chat:
    enabled: true
    credentials_file: "/path/to/google-chat.json"
    spaces:
      - "AAAA1234567"
```

##### HTTP Webhooks

```yaml
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
```

##### Lark

```yaml
services:
  lark:
    enabled: true
    webhook:
      url: "$LARK_WEBHOOK_URL"
    custom_app:
      app_id: "$LARK_APP_ID"
      app_secret: "$LARK_APP_SECRET"
      receivers:
        - type: "open_id"
          id: "ou_1234567890abcdef"
```

##### Line

```yaml
services:
  line:
    enabled: true
    channel_secret: "$LINE_CHANNEL_SECRET"
    channel_access_token: "$LINE_CHANNEL_ACCESS_TOKEN"
    receivers:
      - "U1234567890abcdef"
```

##### Line Notify

```yaml
services:
  line_notify:
    enabled: true
    receivers:
      - "$LINE_NOTIFY_TOKEN"
```

##### Mail (SMTP)

```yaml
services:
  mail:
    enabled: true
    sender_address: "alerts@example.com"
    smtp_host: "smtp.example.com:587"
    smtp_identity: ""
    smtp_username: "$SMTP_USER"
    smtp_password: "$SMTP_PASSWORD"
    smtp_auth_host: "smtp.example.com"
    body_type: "plain"  # plain or html
    receivers:
      - "ops@example.com"
```

##### Mailgun

```yaml
services:
  mailgun:
    enabled: true
    domain: "mg.example.com"
    api_key: "$MAILGUN_API_KEY"
    sender_address: "alerts@mg.example.com"
    receivers:
      - "ops@example.com"
```

##### Matrix

```yaml
services:
  matrix:
    enabled: true
    user_id: "@bot:matrix.example"
    room_id: "!roomid:matrix.example"
    home_server: "https://matrix.example"
    access_token: "$MATRIX_ACCESS_TOKEN"
```

##### Mattermost

```yaml
services:
  mattermost:
    enabled: true
    url: "https://mattermost.example.com"
    login_id: "$MM_LOGIN_ID"
    password: "$MM_PASSWORD"
    channels:
      - "channel-id"
```

##### Microsoft Teams

```yaml
services:
  msteams:
    enabled: true
    webhooks:
      - "$MSTEAMS_WEBHOOK"
    wrap_text: true
    disable_webhook_validation: false
    user_agent: "sliver-notify"
```

##### PagerDuty

```yaml
services:
  pagerduty:
    enabled: true
    token: "$PAGERDUTY_TOKEN"
    from_address: "alerts@example.com"
    receivers:
      - "P123456"
    notification_type: "incident"
    urgency: "high"
    priority_id: "PQ123456"
```

##### Plivo

```yaml
services:
  plivo:
    enabled: true
    auth_id: "$PLIVO_AUTH_ID"
    auth_token: "$PLIVO_AUTH_TOKEN"
    source: "+15551234567"
    callback_url: "https://example.com/plivo"
    callback_method: "POST"
    receivers:
      - "+15557654321"
```

##### Pushbullet

```yaml
services:
  pushbullet:
    enabled: true
    api_token: "$PUSHBULLET_TOKEN"
    device_nicknames:
      - "MyPhone"
```

##### Pushbullet SMS

```yaml
services:
  pushbullet_sms:
    enabled: true
    api_token: "$PUSHBULLET_TOKEN"
    device_nickname: "MyPhone"
    phone_numbers:
      - "+15557654321"
```

##### Pushover

```yaml
services:
  pushover:
    enabled: true
    app_token: "$PUSHOVER_APP_TOKEN"
    recipients:
      - "$PUSHOVER_USER_KEY"
```

##### Reddit

```yaml
services:
  reddit:
    enabled: true
    client_id: "$REDDIT_CLIENT_ID"
    client_secret: "$REDDIT_CLIENT_SECRET"
    username: "$REDDIT_USERNAME"
    password: "$REDDIT_PASSWORD"
    recipients:
      - "target_user"
```

##### Rocket.Chat

```yaml
services:
  rocketchat:
    enabled: true
    server_url: "chat.example.com"
    scheme: "https"
    user_id: "$ROCKETCHAT_USER_ID"
    token: "$ROCKETCHAT_TOKEN"
    channels:
      - "#ops"
```

##### SendGrid

```yaml
services:
  sendgrid:
    enabled: true
    api_key: "$SENDGRID_API_KEY"
    sender_address: "alerts@example.com"
    sender_name: "Sliver"
    receivers:
      - "ops@example.com"
```

##### Slack

```yaml
services:
  slack:
    enabled: true
    api_token: "$SLIVER_SLACK_TOKEN"
    channels:
      - "C0123456789"
```

##### Syslog

```yaml
services:
  syslog:
    enabled: true
    priority: "info"
    network: "udp"
    address: "127.0.0.1:514"
    tag: "sliver"
```

##### Telegram

```yaml
services:
  telegram:
    enabled: true
    api_token: "$SLIVER_TELEGRAM_TOKEN"
    chat_ids:
      - "123456789"
    parse_mode: "HTML"
```

##### TextMagic

```yaml
services:
  textmagic:
    enabled: true
    username: "$TEXTMAGIC_USER"
    api_key: "$TEXTMAGIC_API_KEY"
    phone_numbers:
      - "+15557654321"
```

##### Twilio

```yaml
services:
  twilio:
    enabled: true
    account_sid: "$TWILIO_ACCOUNT_SID"
    auth_token: "$TWILIO_AUTH_TOKEN"
    from_number: "+15551234567"
    phone_numbers:
      - "+15557654321"
```

##### Twitter

```yaml
services:
  twitter:
    enabled: true
    consumer_key: "$TWITTER_CONSUMER_KEY"
    consumer_secret: "$TWITTER_CONSUMER_SECRET"
    access_token: "$TWITTER_ACCESS_TOKEN"
    access_token_secret: "$TWITTER_ACCESS_TOKEN_SECRET"
    recipients:
      - "1234567890"
```

##### Viber

```yaml
services:
  viber:
    enabled: true
    app_key: "$VIBER_APP_KEY"
    sender_name: "Sliver"
    sender_avatar: "https://example.com/avatar.png"
    webhook_url: "https://example.com/viber-webhook"
    receivers:
      - "viber-user-id"
```

##### Web Push

```yaml
services:
  webpush:
    enabled: true
    vapid_public_key: "$WEBPUSH_PUBLIC_KEY"
    vapid_private_key: "$WEBPUSH_PRIVATE_KEY"
    subscriptions:
      - endpoint: "https://push.example.com/123"
        keys:
          p256dh: "$WEBPUSH_P256DH"
          auth: "$WEBPUSH_AUTH"
```

##### WeChat

```yaml
services:
  wechat:
    enabled: true
    app_id: "$WECHAT_APP_ID"
    app_secret: "$WECHAT_APP_SECRET"
    token: "$WECHAT_TOKEN"
    encoding_aes_key: "$WECHAT_AES_KEY"
    receivers:
      - "wechat-user-id"
```

##### WhatsApp

```yaml
services:
  whatsapp:
    enabled: true
    receivers:
      - "whatsapp-contact"
```

#### Notes

- Environment variable expansion only applies to string fields (API keys, tokens, URLs, etc.).
- If a service configuration is incomplete, that service is skipped and the server continues without it.
