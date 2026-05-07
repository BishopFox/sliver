package lark

// Msg Types
const (
	MsgText        = "text"
	MsgPost        = "post"
	MsgInteractive = "interactive"
	MsgImage       = "image"
	MsgShareCard   = "share_chat"
	MsgShareUser   = "share_user"
	MsgAudio       = "audio"
	MsgMedia       = "media"
	MsgFile        = "file"
	MsgSticker     = "sticker"
)

// OutcomingMessage struct of an outcoming message
type OutcomingMessage struct {
	MsgType string         `json:"msg_type"`
	Content MessageContent `json:"content"`
	Card    CardContent    `json:"card"`
	// ID for user
	UIDType string `json:"-"`
	OpenID  string `json:"open_id,omitempty"`
	Email   string `json:"email,omitempty"`
	UserID  string `json:"user_id,omitempty"`
	ChatID  string `json:"chat_id,omitempty"`
	UnionID string `json:"-"`
	// For reply
	RootID        string `json:"root_id,omitempty"`
	ReplyInThread bool   `json:"reply_in_thread,omitempty"`
	// Sign for notification bot
	Sign string `json:"sign"`
	// Timestamp for sign
	Timestamp int64 `json:"timestamp"`
	// UUID for idempotency
	UUID string `json:"uuid"`
}

// CardContent struct of card content
type CardContent map[string]interface{}

// MessageContent struct of message content
type MessageContent struct {
	Text      *TextContent      `json:"text,omitempty"`
	Image     *ImageContent     `json:"image,omitempty"`
	Post      *PostContent      `json:"post,omitempty"`
	Card      *CardContent      `json:"card,omitempty"`
	ShareChat *ShareChatContent `json:"share_chat,omitempty"`
	ShareUser *ShareUserContent `json:"share_user,omitempty"`
	Audio     *AudioContent     `json:"audio,omitempty"`
	Media     *MediaContent     `json:"media,omitempty"`
	File      *FileContent      `json:"file,omitempty"`
	Sticker   *StickerContent   `json:"sticker,omitempty"`
	Template  *TemplateContent  `json:"template,omitempty"`
}

// TextContent .
type TextContent struct {
	Text string `json:"text"`
}

// ImageContent .
type ImageContent struct {
	ImageKey string `json:"image_key"`
}

// ShareChatContent .
type ShareChatContent struct {
	ChatID string `json:"chat_id"`
}

// ShareUserContent .
type ShareUserContent struct {
	UserID string `json:"user_id"`
}

// AudioContent .
type AudioContent struct {
	FileKey string `json:"file_key"`
}

// MediaContent .
type MediaContent struct {
	FileName string `json:"file_name,omitempty"`
	FileKey  string `json:"file_key"`
	ImageKey string `json:"image_key"`
	Duration int    `json:"duration,omitempty"`
}

// FileContent .
type FileContent struct {
	FileName string `json:"file_name,omitempty"`
	FileKey  string `json:"file_key"`
}

// StickerContent .
type StickerContent struct {
	FileKey string `json:"file_key"`
}

// TemplateContent .
type TemplateContent struct {
	Type string       `json:"type"`
	Data templateData `json:"data,omitempty"`
}

type templateData struct {
	TemplateID          string                 `json:"template_id"`
	TemplateVersionName string                 `json:"template_version_name,omitempty"`
	TemplateVariable    map[string]interface{} `json:"template_variable,omitempty"`
}
