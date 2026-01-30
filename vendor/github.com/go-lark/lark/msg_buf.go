package lark

import (
	"encoding/json"
	"fmt"
)

// MsgBuffer stores all the messages attached
// You can call every function, but some of which is only available for specific condition
type MsgBuffer struct {
	// Message type
	msgType string
	// Output
	message OutcomingMessage

	err error
}

// NewMsgBuffer create a message buffer
func NewMsgBuffer(newMsgType string) *MsgBuffer {
	msgBuffer := MsgBuffer{
		message: OutcomingMessage{
			MsgType: newMsgType,
		},
		msgType: newMsgType,
	}
	return &msgBuffer
}

// BindOpenID binds open_id
func (m *MsgBuffer) BindOpenID(openID string) *MsgBuffer {
	m.message.OpenID = openID
	m.message.UIDType = UIDOpenID
	return m
}

// BindEmail binds email
func (m *MsgBuffer) BindEmail(email string) *MsgBuffer {
	m.message.Email = email
	m.message.UIDType = UIDEmail
	return m
}

// BindChatID binds chat_id
func (m *MsgBuffer) BindChatID(chatID string) *MsgBuffer {
	m.message.ChatID = chatID
	m.message.UIDType = UIDChatID
	return m
}

// BindOpenChatID binds open_chat_id
func (m *MsgBuffer) BindOpenChatID(openChatID string) *MsgBuffer {
	m.BindChatID(openChatID)
	m.message.UIDType = UIDChatID
	return m
}

// BindUserID binds open_id
func (m *MsgBuffer) BindUserID(userID string) *MsgBuffer {
	m.message.UserID = userID
	m.message.UIDType = UIDUserID
	return m
}

// BindUnionID binds union_id
func (m *MsgBuffer) BindUnionID(unionID string) *MsgBuffer {
	m.message.UnionID = unionID
	m.message.UIDType = UIDUnionID
	return m
}

// BindReply binds root id for reply
// rootID is OpenMessageID of the message you reply
func (m *MsgBuffer) BindReply(rootID string) *MsgBuffer {
	m.message.RootID = rootID
	return m
}

// ReplyInThread replies message in thread
func (m *MsgBuffer) ReplyInThread(replyInThread bool) *MsgBuffer {
	m.message.ReplyInThread = replyInThread
	return m
}

// WithSign generates sign for notification bot check
func (m *MsgBuffer) WithSign(secret string, ts int64) *MsgBuffer {
	m.message.Sign, _ = GenSign(secret, ts)
	m.message.Timestamp = ts
	return m
}

// WithUUID add UUID to message for idempotency
func (m *MsgBuffer) WithUUID(uuid string) *MsgBuffer {
	m.message.UUID = uuid
	return m
}

func (m MsgBuffer) typeError(funcName string, msgType string) error {
	return fmt.Errorf("`%s` is only available to `%s`", funcName, msgType)
}

// Text attaches text
func (m *MsgBuffer) Text(text string) *MsgBuffer {
	if m.msgType != MsgText {
		m.err = m.typeError("Text", MsgText)
		return m
	}
	m.message.Content.Text = &TextContent{
		Text: text,
	}
	return m
}

// Image attaches image key
// for MsgImage only
func (m *MsgBuffer) Image(imageKey string) *MsgBuffer {
	if m.msgType != MsgImage {
		m.err = m.typeError("Image", MsgImage)
		return m
	}
	m.message.Content.Image = &ImageContent{
		ImageKey: imageKey,
	}
	return m
}

// ShareChat attaches chat id
// for MsgShareChat only
func (m *MsgBuffer) ShareChat(chatID string) *MsgBuffer {
	if m.msgType != MsgShareCard {
		m.err = m.typeError("ShareChat", MsgShareCard)
		return m
	}
	m.message.Content.ShareChat = &ShareChatContent{
		ChatID: chatID,
	}
	return m
}

// ShareUser attaches user id
// for MsgShareUser only
func (m *MsgBuffer) ShareUser(userID string) *MsgBuffer {
	if m.msgType != MsgShareUser {
		m.err = m.typeError("ShareUser", MsgShareUser)
		return m
	}
	m.message.Content.ShareUser = &ShareUserContent{
		UserID: userID,
	}
	return m
}

// File attaches file
// for MsgFile only
func (m *MsgBuffer) File(fileKey string) *MsgBuffer {
	if m.msgType != MsgFile {
		m.err = m.typeError("File", MsgFile)
		return m
	}
	m.message.Content.File = &FileContent{
		FileKey: fileKey,
	}
	return m
}

// Audio attaches audio
// for MsgAudio only
func (m *MsgBuffer) Audio(fileKey string) *MsgBuffer {
	if m.msgType != MsgAudio {
		m.err = m.typeError("Audio", MsgAudio)
		return m
	}
	m.message.Content.Audio = &AudioContent{
		FileKey: fileKey,
	}
	return m
}

// Media attaches media
// for MsgMedia only
func (m *MsgBuffer) Media(fileKey, imageKey string) *MsgBuffer {
	if m.msgType != MsgMedia {
		m.err = m.typeError("Media", MsgMedia)
		return m
	}
	m.message.Content.Media = &MediaContent{
		FileKey:  fileKey,
		ImageKey: imageKey,
	}
	return m
}

// Sticker attaches sticker
// for MsgSticker only
func (m *MsgBuffer) Sticker(fileKey string) *MsgBuffer {
	if m.msgType != MsgSticker {
		m.err = m.typeError("Sticker", MsgSticker)
		return m
	}
	m.message.Content.Sticker = &StickerContent{
		FileKey: fileKey,
	}
	return m
}

// Post sets raw post content
func (m *MsgBuffer) Post(postContent *PostContent) *MsgBuffer {
	if m.msgType != MsgPost {
		m.err = m.typeError("Post", MsgPost)
		return m
	}
	m.message.Content.Post = postContent
	return m
}

// Card binds card content with V4 format
func (m *MsgBuffer) Card(cardContent string) *MsgBuffer {
	if m.msgType != MsgInteractive {
		m.err = m.typeError("Card", MsgInteractive)
		return m
	}
	card := make(CardContent)
	_ = json.Unmarshal([]byte(cardContent), &card)
	m.message.Content.Card = &card
	return m
}

// Template sets raw template content
func (m *MsgBuffer) Template(tempateContent *TemplateContent) *MsgBuffer {
	if m.msgType != MsgInteractive {
		m.err = m.typeError("Template", MsgInteractive)
		return m
	}
	m.message.Content.Template = tempateContent
	return m
}

// Build message and return message body
func (m *MsgBuffer) Build() OutcomingMessage {
	return m.message
}

// Error returns last error
func (m *MsgBuffer) Error() error {
	return m.err
}

// Clear message in buffer
func (m *MsgBuffer) Clear() *MsgBuffer {
	m.message = OutcomingMessage{
		MsgType: m.msgType,
	}
	m.err = nil
	return m
}
