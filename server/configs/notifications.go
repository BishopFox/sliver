package configs

// NotificationsConfig - Notification configuration for the server.
type NotificationsConfig struct {
	Enabled   bool                                   `json:"enabled" yaml:"enabled"`
	Events    []string                               `json:"events,omitempty" yaml:"events,omitempty"`
	Templates map[string]*NotificationTemplateConfig `json:"templates,omitempty" yaml:"templates,omitempty"`
	Services  *NotificationsServicesConfig           `json:"services,omitempty" yaml:"services,omitempty"`
}

// NotificationServiceConfig - Shared settings for notification services.
type NotificationServiceConfig struct {
	Enabled bool     `json:"enabled" yaml:"enabled"`
	Events  []string `json:"events,omitempty" yaml:"events,omitempty"`
}

// NotificationTemplateConfig - Per-event template configuration.
type NotificationTemplateConfig struct {
	Type     string `json:"type,omitempty" yaml:"type,omitempty"`
	Template string `json:"template" yaml:"template"`
}

// NotificationsServicesConfig - Available notification services.
type NotificationsServicesConfig struct {
	AmazonSES     *AmazonSESConfig     `json:"amazon_ses,omitempty" yaml:"amazon_ses,omitempty"`
	AmazonSNS     *AmazonSNSConfig     `json:"amazon_sns,omitempty" yaml:"amazon_sns,omitempty"`
	Bark          *BarkConfig          `json:"bark,omitempty" yaml:"bark,omitempty"`
	DingDing      *DingDingConfig      `json:"dingding,omitempty" yaml:"dingding,omitempty"`
	Discord       *DiscordConfig       `json:"discord,omitempty" yaml:"discord,omitempty"`
	FCM           *FCMConfig           `json:"fcm,omitempty" yaml:"fcm,omitempty"`
	GoogleChat    *GoogleChatConfig    `json:"google_chat,omitempty" yaml:"google_chat,omitempty"`
	HTTP          *HTTPConfig          `json:"http,omitempty" yaml:"http,omitempty"`
	Lark          *LarkConfig          `json:"lark,omitempty" yaml:"lark,omitempty"`
	Line          *LineConfig          `json:"line,omitempty" yaml:"line,omitempty"`
	LineNotify    *LineNotifyConfig    `json:"line_notify,omitempty" yaml:"line_notify,omitempty"`
	Mail          *MailConfig          `json:"mail,omitempty" yaml:"mail,omitempty"`
	Mailgun       *MailgunConfig       `json:"mailgun,omitempty" yaml:"mailgun,omitempty"`
	Matrix        *MatrixConfig        `json:"matrix,omitempty" yaml:"matrix,omitempty"`
	Mattermost    *MattermostConfig    `json:"mattermost,omitempty" yaml:"mattermost,omitempty"`
	MSTeams       *MSTeamsConfig       `json:"msteams,omitempty" yaml:"msteams,omitempty"`
	PagerDuty     *PagerDutyConfig     `json:"pagerduty,omitempty" yaml:"pagerduty,omitempty"`
	Plivo         *PlivoConfig         `json:"plivo,omitempty" yaml:"plivo,omitempty"`
	Pushbullet    *PushbulletConfig    `json:"pushbullet,omitempty" yaml:"pushbullet,omitempty"`
	PushbulletSMS *PushbulletSMSConfig `json:"pushbullet_sms,omitempty" yaml:"pushbullet_sms,omitempty"`
	Pushover      *PushoverConfig      `json:"pushover,omitempty" yaml:"pushover,omitempty"`
	Reddit        *RedditConfig        `json:"reddit,omitempty" yaml:"reddit,omitempty"`
	RocketChat    *RocketChatConfig    `json:"rocketchat,omitempty" yaml:"rocketchat,omitempty"`
	SendGrid      *SendGridConfig      `json:"sendgrid,omitempty" yaml:"sendgrid,omitempty"`
	Slack         *SlackConfig         `json:"slack,omitempty" yaml:"slack,omitempty"`
	Syslog        *SyslogConfig        `json:"syslog,omitempty" yaml:"syslog,omitempty"`
	Telegram      *TelegramConfig      `json:"telegram,omitempty" yaml:"telegram,omitempty"`
	TextMagic     *TextMagicConfig     `json:"textmagic,omitempty" yaml:"textmagic,omitempty"`
	Twilio        *TwilioConfig        `json:"twilio,omitempty" yaml:"twilio,omitempty"`
	Twitter       *TwitterConfig       `json:"twitter,omitempty" yaml:"twitter,omitempty"`
	Viber         *ViberConfig         `json:"viber,omitempty" yaml:"viber,omitempty"`
	WebPush       *WebPushConfig       `json:"webpush,omitempty" yaml:"webpush,omitempty"`
	WeChat        *WeChatConfig        `json:"wechat,omitempty" yaml:"wechat,omitempty"`
	WhatsApp      *WhatsAppConfig      `json:"whatsapp,omitempty" yaml:"whatsapp,omitempty"`
}

// AmazonSESConfig - Amazon SES notifications.
type AmazonSESConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	AccessKeyID               string   `json:"access_key_id" yaml:"access_key_id"`
	SecretKey                 string   `json:"secret_key" yaml:"secret_key"`
	Region                    string   `json:"region" yaml:"region"`
	SenderAddress             string   `json:"sender_address" yaml:"sender_address"`
	Receivers                 []string `json:"receivers" yaml:"receivers"`
}

// AmazonSNSConfig - Amazon SNS notifications.
type AmazonSNSConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	AccessKeyID               string   `json:"access_key_id" yaml:"access_key_id"`
	SecretKey                 string   `json:"secret_key" yaml:"secret_key"`
	Region                    string   `json:"region" yaml:"region"`
	Receivers                 []string `json:"receivers" yaml:"receivers"`
}

// BarkConfig - Bark notifications.
type BarkConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	DeviceKey                 string   `json:"device_key" yaml:"device_key"`
	Servers                   []string `json:"servers,omitempty" yaml:"servers,omitempty"`
}

// DingDingConfig - DingTalk notifications.
type DingDingConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	Token                     string `json:"token" yaml:"token"`
	Secret                    string `json:"secret" yaml:"secret"`
}

// DiscordConfig - Discord notifications.
type DiscordConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	Token                     string   `json:"token" yaml:"token"`
	TokenType                 string   `json:"token_type,omitempty" yaml:"token_type,omitempty"`
	Channels                  []string `json:"channels" yaml:"channels"`
}

// FCMConfig - Firebase Cloud Messaging notifications.
type FCMConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	CredentialsFile           string   `json:"credentials_file,omitempty" yaml:"credentials_file,omitempty"`
	ProjectID                 string   `json:"project_id,omitempty" yaml:"project_id,omitempty"`
	DeviceTokens              []string `json:"device_tokens" yaml:"device_tokens"`
}

// GoogleChatConfig - Google Chat notifications.
type GoogleChatConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	CredentialsFile           string   `json:"credentials_file,omitempty" yaml:"credentials_file,omitempty"`
	CredentialsJSON           string   `json:"credentials_json,omitempty" yaml:"credentials_json,omitempty"`
	Spaces                    []string `json:"spaces" yaml:"spaces"`
}

// HTTPConfig - HTTP webhook notifications.
type HTTPConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	URLs                      []string            `json:"urls,omitempty" yaml:"urls,omitempty"`
	Webhooks                  []HTTPWebhookConfig `json:"webhooks,omitempty" yaml:"webhooks,omitempty"`
}

// HTTPWebhookConfig - Detailed webhook configuration.
type HTTPWebhookConfig struct {
	URL         string            `json:"url" yaml:"url"`
	Method      string            `json:"method,omitempty" yaml:"method,omitempty"`
	ContentType string            `json:"content_type,omitempty" yaml:"content_type,omitempty"`
	Headers     map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// LarkConfig - Lark notifications.
type LarkConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	Webhook                   *LarkWebhookConfig   `json:"webhook,omitempty" yaml:"webhook,omitempty"`
	CustomApp                 *LarkCustomAppConfig `json:"custom_app,omitempty" yaml:"custom_app,omitempty"`
}

// LarkWebhookConfig - Lark webhook notifications.
type LarkWebhookConfig struct {
	URL string `json:"url" yaml:"url"`
}

// LarkCustomAppConfig - Lark custom app notifications.
type LarkCustomAppConfig struct {
	AppID     string               `json:"app_id" yaml:"app_id"`
	AppSecret string               `json:"app_secret" yaml:"app_secret"`
	Receivers []LarkReceiverConfig `json:"receivers" yaml:"receivers"`
}

// LarkReceiverConfig - Lark receiver IDs.
type LarkReceiverConfig struct {
	Type string `json:"type" yaml:"type"`
	ID   string `json:"id" yaml:"id"`
}

// LineConfig - Line bot notifications.
type LineConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	ChannelSecret             string   `json:"channel_secret" yaml:"channel_secret"`
	ChannelAccessToken        string   `json:"channel_access_token" yaml:"channel_access_token"`
	Receivers                 []string `json:"receivers" yaml:"receivers"`
}

// LineNotifyConfig - Line notify token notifications.
type LineNotifyConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	Receivers                 []string `json:"receivers" yaml:"receivers"`
}

// MailConfig - SMTP mail notifications.
type MailConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	SenderAddress             string   `json:"sender_address" yaml:"sender_address"`
	SMTPHost                  string   `json:"smtp_host" yaml:"smtp_host"`
	SMTPIdentity              string   `json:"smtp_identity,omitempty" yaml:"smtp_identity,omitempty"`
	SMTPUsername              string   `json:"smtp_username,omitempty" yaml:"smtp_username,omitempty"`
	SMTPPassword              string   `json:"smtp_password,omitempty" yaml:"smtp_password,omitempty"`
	SMTPAuthHost              string   `json:"smtp_auth_host,omitempty" yaml:"smtp_auth_host,omitempty"`
	BodyType                  string   `json:"body_type,omitempty" yaml:"body_type,omitempty"`
	Receivers                 []string `json:"receivers" yaml:"receivers"`
}

// MailgunConfig - Mailgun notifications.
type MailgunConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	Domain                    string   `json:"domain" yaml:"domain"`
	APIKey                    string   `json:"api_key" yaml:"api_key"`
	SenderAddress             string   `json:"sender_address" yaml:"sender_address"`
	Receivers                 []string `json:"receivers" yaml:"receivers"`
}

// MatrixConfig - Matrix notifications.
type MatrixConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	UserID                    string `json:"user_id" yaml:"user_id"`
	RoomID                    string `json:"room_id" yaml:"room_id"`
	HomeServer                string `json:"home_server" yaml:"home_server"`
	AccessToken               string `json:"access_token" yaml:"access_token"`
}

// MattermostConfig - Mattermost notifications.
type MattermostConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	URL                       string   `json:"url" yaml:"url"`
	LoginID                   string   `json:"login_id,omitempty" yaml:"login_id,omitempty"`
	Password                  string   `json:"password,omitempty" yaml:"password,omitempty"`
	Channels                  []string `json:"channels" yaml:"channels"`
}

// MSTeamsConfig - Microsoft Teams notifications.
type MSTeamsConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	Webhooks                  []string `json:"webhooks" yaml:"webhooks"`
	WrapText                  bool     `json:"wrap_text,omitempty" yaml:"wrap_text,omitempty"`
	DisableWebhookValidation  bool     `json:"disable_webhook_validation,omitempty" yaml:"disable_webhook_validation,omitempty"`
	UserAgent                 string   `json:"user_agent,omitempty" yaml:"user_agent,omitempty"`
}

// PagerDutyConfig - PagerDuty notifications.
type PagerDutyConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	Token                     string   `json:"token" yaml:"token"`
	FromAddress               string   `json:"from_address" yaml:"from_address"`
	Receivers                 []string `json:"receivers" yaml:"receivers"`
	NotificationType          string   `json:"notification_type,omitempty" yaml:"notification_type,omitempty"`
	Urgency                   string   `json:"urgency,omitempty" yaml:"urgency,omitempty"`
	PriorityID                string   `json:"priority_id,omitempty" yaml:"priority_id,omitempty"`
}

// PlivoConfig - Plivo notifications.
type PlivoConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	AuthID                    string   `json:"auth_id" yaml:"auth_id"`
	AuthToken                 string   `json:"auth_token" yaml:"auth_token"`
	Source                    string   `json:"source" yaml:"source"`
	CallbackURL               string   `json:"callback_url,omitempty" yaml:"callback_url,omitempty"`
	CallbackMethod            string   `json:"callback_method,omitempty" yaml:"callback_method,omitempty"`
	Receivers                 []string `json:"receivers" yaml:"receivers"`
}

// PushbulletConfig - Pushbullet notifications.
type PushbulletConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	APIToken                  string   `json:"api_token" yaml:"api_token"`
	DeviceNicknames           []string `json:"device_nicknames" yaml:"device_nicknames"`
}

// PushbulletSMSConfig - Pushbullet SMS notifications.
type PushbulletSMSConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	APIToken                  string   `json:"api_token" yaml:"api_token"`
	DeviceNickname            string   `json:"device_nickname" yaml:"device_nickname"`
	PhoneNumbers              []string `json:"phone_numbers" yaml:"phone_numbers"`
}

// PushoverConfig - Pushover notifications.
type PushoverConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	AppToken                  string   `json:"app_token" yaml:"app_token"`
	Recipients                []string `json:"recipients" yaml:"recipients"`
}

// RedditConfig - Reddit notifications.
type RedditConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	ClientID                  string   `json:"client_id" yaml:"client_id"`
	ClientSecret              string   `json:"client_secret" yaml:"client_secret"`
	Username                  string   `json:"username" yaml:"username"`
	Password                  string   `json:"password" yaml:"password"`
	Recipients                []string `json:"recipients" yaml:"recipients"`
}

// RocketChatConfig - Rocket.Chat notifications.
type RocketChatConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	ServerURL                 string   `json:"server_url" yaml:"server_url"`
	Scheme                    string   `json:"scheme" yaml:"scheme"`
	UserID                    string   `json:"user_id" yaml:"user_id"`
	Token                     string   `json:"token" yaml:"token"`
	Channels                  []string `json:"channels" yaml:"channels"`
}

// SendGridConfig - SendGrid notifications.
type SendGridConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	APIKey                    string   `json:"api_key" yaml:"api_key"`
	SenderAddress             string   `json:"sender_address" yaml:"sender_address"`
	SenderName                string   `json:"sender_name" yaml:"sender_name"`
	Receivers                 []string `json:"receivers" yaml:"receivers"`
}

// SlackConfig - Slack notifications.
type SlackConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	APIToken                  string   `json:"api_token" yaml:"api_token"`
	Channels                  []string `json:"channels" yaml:"channels"`
}

// SyslogConfig - Syslog notifications.
type SyslogConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	Priority                  string `json:"priority,omitempty" yaml:"priority,omitempty"`
	Network                   string `json:"network,omitempty" yaml:"network,omitempty"`
	Address                   string `json:"address,omitempty" yaml:"address,omitempty"`
	Tag                       string `json:"tag,omitempty" yaml:"tag,omitempty"`
}

// TelegramConfig - Telegram notifications.
type TelegramConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	APIToken                  string   `json:"api_token" yaml:"api_token"`
	ChatIDs                   []string `json:"chat_ids" yaml:"chat_ids"`
	ParseMode                 string   `json:"parse_mode,omitempty" yaml:"parse_mode,omitempty"`
}

// TextMagicConfig - TextMagic notifications.
type TextMagicConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	Username                  string   `json:"username" yaml:"username"`
	APIKey                    string   `json:"api_key" yaml:"api_key"`
	PhoneNumbers              []string `json:"phone_numbers" yaml:"phone_numbers"`
}

// TwilioConfig - Twilio notifications.
type TwilioConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	AccountSID                string   `json:"account_sid" yaml:"account_sid"`
	AuthToken                 string   `json:"auth_token" yaml:"auth_token"`
	FromNumber                string   `json:"from_number" yaml:"from_number"`
	PhoneNumbers              []string `json:"phone_numbers" yaml:"phone_numbers"`
}

// TwitterConfig - Twitter notifications.
type TwitterConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	ConsumerKey               string   `json:"consumer_key" yaml:"consumer_key"`
	ConsumerSecret            string   `json:"consumer_secret" yaml:"consumer_secret"`
	AccessToken               string   `json:"access_token" yaml:"access_token"`
	AccessTokenSecret         string   `json:"access_token_secret" yaml:"access_token_secret"`
	Recipients                []string `json:"recipients" yaml:"recipients"`
}

// ViberConfig - Viber notifications.
type ViberConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	AppKey                    string   `json:"app_key" yaml:"app_key"`
	SenderName                string   `json:"sender_name" yaml:"sender_name"`
	SenderAvatar              string   `json:"sender_avatar,omitempty" yaml:"sender_avatar,omitempty"`
	Receivers                 []string `json:"receivers" yaml:"receivers"`
	WebhookURL                string   `json:"webhook_url,omitempty" yaml:"webhook_url,omitempty"`
}

// WebPushConfig - Webpush notifications.
type WebPushConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	VAPIDPublicKey            string                      `json:"vapid_public_key" yaml:"vapid_public_key"`
	VAPIDPrivateKey           string                      `json:"vapid_private_key" yaml:"vapid_private_key"`
	Subscriptions             []WebPushSubscriptionConfig `json:"subscriptions" yaml:"subscriptions"`
}

// WebPushSubscriptionConfig - Webpush subscription data.
type WebPushSubscriptionConfig struct {
	Endpoint string            `json:"endpoint" yaml:"endpoint"`
	Keys     WebPushKeysConfig `json:"keys" yaml:"keys"`
}

// WebPushKeysConfig - Webpush subscription keys.
type WebPushKeysConfig struct {
	Auth   string `json:"auth" yaml:"auth"`
	P256DH string `json:"p256dh" yaml:"p256dh"`
}

// WeChatConfig - WeChat notifications.
type WeChatConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	AppID                     string   `json:"app_id" yaml:"app_id"`
	AppSecret                 string   `json:"app_secret" yaml:"app_secret"`
	Token                     string   `json:"token" yaml:"token"`
	EncodingAESKey            string   `json:"encoding_aes_key" yaml:"encoding_aes_key"`
	Receivers                 []string `json:"receivers" yaml:"receivers"`
}

// WhatsAppConfig - WhatsApp notifications.
type WhatsAppConfig struct {
	NotificationServiceConfig `json:",inline" yaml:",inline"`
	Receivers                 []string `json:"receivers,omitempty" yaml:"receivers,omitempty"`
}

func defaultNotificationsConfig() *NotificationsConfig {
	return &NotificationsConfig{
		Enabled:   false,
		Events:    []string{},
		Templates: map[string]*NotificationTemplateConfig{},
		Services:  &NotificationsServicesConfig{},
	}
}
