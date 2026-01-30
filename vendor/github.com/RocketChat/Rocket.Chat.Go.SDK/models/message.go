package models

import "time"

type Message struct {
	ID       string `json:"_id"`
	RoomID   string `json:"rid"`
	Msg      string `json:"msg"`
	EditedBy string `json:"editedBy,omitempty"`
	Type     string `json:"t,omitempty"`

	Groupable bool `json:"groupable,omitempty"`

	EditedAt  *time.Time `json:"editedAt,omitempty"`
	Timestamp *time.Time `json:"ts,omitempty"`
	UpdatedAt *time.Time `json:"_updatedAt,omitempty"`

	Mentions []User `json:"mentions,omitempty"`
	User     *User  `json:"u,omitempty"`
	PostMessage

	// Bot         interface{}  `json:"bot"`
	// CustomFields interface{} `json:"customFields"`
	// Channels           []interface{} `json:"channels"`
	// SandstormSessionID interface{} `json:"sandstormSessionId"`
}

// UpdateMessage is payload for chat.update
//
// https://developer.rocket.chat/reference/api/rest-api/endpoints/core-endpoints/chat-endpoints/message-update
type UpdateMessage struct {
	RoomID string `json:"roomId"`
	MsgID  string `json:"msgId"`
	Text   string `json:"text"`
}

// DeleteMessage is payload for chat.delete endpoint
//
// https://developer.rocket.chat/reference/api/rest-api/endpoints/core-endpoints/chat-endpoints/delete
type DeleteMessage struct {
	RoomID string `json:"roomId"`
	MsgID  string `json:"msgId"`
	AsUser bool   `json:"asUser"`
}

// PostMessage Payload for postmessage rest API
//
// https://rocket.chat/docs/developer-guides/rest-api/chat/postmessage/
type PostMessage struct {
	RoomID      string       `json:"roomId,omitempty"`
	Channel     string       `json:"channel,omitempty"`
	Text        string       `json:"text,omitempty"`
	ParseUrls   bool         `json:"parseUrls,omitempty"`
	Alias       string       `json:"alias,omitempty"`
	Emoji       string       `json:"emoji,omitempty"`
	Avatar      string       `json:"avatar,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment Payload for postmessage rest API
//
// https://rocket.chat/docs/developer-guides/rest-api/chat/postmessage/
type Attachment struct {
	Color       string `json:"color,omitempty"`
	Text        string `json:"text,omitempty"`
	Timestamp   string `json:"ts,omitempty"`
	ThumbURL    string `json:"thumb_url,omitempty"`
	MessageLink string `json:"message_link,omitempty"`
	Collapsed   bool   `json:"collapsed"`

	AuthorName string `json:"author_name,omitempty"`
	AuthorLink string `json:"author_link,omitempty"`
	AuthorIcon string `json:"author_icon,omitempty"`

	Title             string `json:"title,omitempty"`
	TitleLink         string `json:"title_link,omitempty"`
	TitleLinkDownload string `json:"title_link_download,omitempty"`

	ImageURL string `json:"image_url,omitempty"`

	AudioURL string `json:"audio_url,omitempty"`
	VideoURL string `json:"video_url,omitempty"`

	Actions                []AttachmentAction               `json:"actions,omitempty"`
	ActionButtonsAlignment AttachmentActionButtonsAlignment `json:"button_alignment,omitempty"`

	Fields []AttachmentField `json:"fields,omitempty"`
}

// AttachmentField Payload for postmessage rest API
//
// https://rocket.chat/docs/developer-guides/rest-api/chat/postmessage/
type AttachmentField struct {
	Short bool   `json:"short"`
	Title string `json:"title,omitempty"`
	Value string `json:"value,omitempty"`
}

type AttachmentActionType string

const (
	AttachmentActionTypeButton AttachmentActionType = "button"
)

// AttachmentAction are action buttons on message attachments
type AttachmentAction struct {
	Type               AttachmentActionType  `json:"type,omitempty"`
	Text               string                `json:"text,omitempty"`
	Url                string                `json:"url,omitempty"`
	ImageURL           string                `json:"image_url,omitempty"`
	IsWebView          bool                  `json:"is_webview"`
	WebviewHeightRatio string                `json:"webview_height_ratio,omitempty"`
	Msg                string                `json:"msg,omitempty"`
	MsgInChatWindow    bool                  `json:"msg_in_chat_window"`
	MsgProcessingType  MessageProcessingType `json:"msg_processing_type,omitempty"`
}

// AttachmentActionButtonAlignment configures how the actions buttons will be aligned
type AttachmentActionButtonsAlignment string

const (
	ActionButtonAlignVertical   AttachmentActionButtonsAlignment = "vertical"
	ActionButtonAlignHorizontal AttachmentActionButtonsAlignment = "horizontal"
)

type MessageProcessingType string

const (
	ProcessingTypeSendMessage        MessageProcessingType = "sendMessage"
	ProcessingTypeRespondWithMessage MessageProcessingType = "respondWithMessage"
)
