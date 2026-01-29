package discordgo

// Webhook stores the data for a webhook.
type Webhook struct {
	ID        string      `json:"id"`
	Type      WebhookType `json:"type"`
	GuildID   string      `json:"guild_id"`
	ChannelID string      `json:"channel_id"`
	User      *User       `json:"user"`
	Name      string      `json:"name"`
	Avatar    string      `json:"avatar"`
	Token     string      `json:"token"`

	// ApplicationID is the bot/OAuth2 application that created this webhook
	ApplicationID string `json:"application_id,omitempty"`
}

// WebhookType is the type of Webhook (see WebhookType* consts) in the Webhook struct
// https://discord.com/developers/docs/resources/webhook#webhook-object-webhook-types
type WebhookType int

// Valid WebhookType values
const (
	WebhookTypeIncoming        WebhookType = 1
	WebhookTypeChannelFollower WebhookType = 2
)

// WebhookParams is a struct for webhook params, used in the WebhookExecute command.
type WebhookParams struct {
	Content         string                  `json:"content,omitempty"`
	Username        string                  `json:"username,omitempty"`
	AvatarURL       string                  `json:"avatar_url,omitempty"`
	TTS             bool                    `json:"tts,omitempty"`
	Files           []*File                 `json:"-"`
	Components      []MessageComponent      `json:"components"`
	Embeds          []*MessageEmbed         `json:"embeds,omitempty"`
	Attachments     []*MessageAttachment    `json:"attachments,omitempty"`
	AllowedMentions *MessageAllowedMentions `json:"allowed_mentions,omitempty"`
	// Only MessageFlagsSuppressEmbeds and MessageFlagsEphemeral can be set.
	// MessageFlagsEphemeral can only be set when using Followup Message Create endpoint.
	Flags MessageFlags `json:"flags,omitempty"`
	// Name of the thread to create.
	// NOTE: can only be set if the webhook channel is a forum.
	ThreadName string `json:"thread_name,omitempty"`
}

// WebhookEdit stores data for editing of a webhook message.
type WebhookEdit struct {
	Content         *string                 `json:"content,omitempty"`
	Components      *[]MessageComponent     `json:"components,omitempty"`
	Embeds          *[]*MessageEmbed        `json:"embeds,omitempty"`
	Files           []*File                 `json:"-"`
	Attachments     *[]*MessageAttachment   `json:"attachments,omitempty"`
	AllowedMentions *MessageAllowedMentions `json:"allowed_mentions,omitempty"`
}
