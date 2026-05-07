// Copyright 2016 LINE Corporation
//
// LINE Corporation licenses this file to you under the Apache License,
// version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at:
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package linebot

import (
	"encoding/json"
)

// MessageType type
type MessageType string

// MessageType constants
const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeVideo    MessageType = "video"
	MessageTypeAudio    MessageType = "audio"
	MessageTypeFile     MessageType = "file"
	MessageTypeLocation MessageType = "location"
	MessageTypeSticker  MessageType = "sticker"
	MessageTypeTemplate MessageType = "template"
	MessageTypeImagemap MessageType = "imagemap"
	MessageTypeFlex     MessageType = "flex"
)

// Message interface
type Message interface {
	Message()
}

// SendingMessage interface
type SendingMessage interface {
	Message
	WithQuickReplies(*QuickReplyItems) SendingMessage
	WithSender(*Sender) SendingMessage
	AddEmoji(emoji *Emoji) SendingMessage
}

// TextMessage type
type TextMessage struct {
	ID     string
	Text   string
	Emojis []*Emoji

	quickReplyitems *QuickReplyItems
	sender          *Sender

	Mention *Mention
}

// MarshalJSON method of TextMessage
func (m *TextMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type       MessageType      `json:"type"`
		Text       string           `json:"text"`
		QuickReply *QuickReplyItems `json:"quickReply,omitempty"`
		Sender     *Sender          `json:"sender,omitempty"`
		Emojis     []*Emoji         `json:"emojis,omitempty"`
	}{
		Type:       MessageTypeText,
		Text:       m.Text,
		QuickReply: m.quickReplyitems,
		Sender:     m.sender,
		Emojis:     m.Emojis,
	})
}

// WithQuickReplies method of TextMessage
func (m *TextMessage) WithQuickReplies(items *QuickReplyItems) SendingMessage {
	m.quickReplyitems = items
	return m
}

// WithSender method of TextMessage
func (m *TextMessage) WithSender(sender *Sender) SendingMessage {
	m.sender = sender
	return m
}

// AddEmoji method of TextMessage
func (m *TextMessage) AddEmoji(emoji *Emoji) SendingMessage {
	m.Emojis = append(m.Emojis, emoji)
	return m
}

// ImageMessage type
type ImageMessage struct {
	ID                 string
	OriginalContentURL string
	PreviewImageURL    string

	quickReplyitems *QuickReplyItems
	sender          *Sender
}

// MarshalJSON method of ImageMessage
func (m *ImageMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type               MessageType      `json:"type"`
		OriginalContentURL string           `json:"originalContentUrl"`
		PreviewImageURL    string           `json:"previewImageUrl"`
		QuickReply         *QuickReplyItems `json:"quickReply,omitempty"`
		Sender             *Sender          `json:"sender,omitempty"`
	}{
		Type:               MessageTypeImage,
		OriginalContentURL: m.OriginalContentURL,
		PreviewImageURL:    m.PreviewImageURL,
		QuickReply:         m.quickReplyitems,
		Sender:             m.sender,
	})
}

// WithQuickReplies method of ImageMessage
func (m *ImageMessage) WithQuickReplies(items *QuickReplyItems) SendingMessage {
	m.quickReplyitems = items
	return m
}

// WithSender method of ImageMessage
func (m *ImageMessage) WithSender(sender *Sender) SendingMessage {
	m.sender = sender
	return m
}

// AddEmoji method of ImageMessage
func (m *ImageMessage) AddEmoji(emoji *Emoji) SendingMessage {
	return m
}

// VideoMessage type
type VideoMessage struct {
	ID                 string
	OriginalContentURL string
	PreviewImageURL    string

	quickReplyitems *QuickReplyItems
	sender          *Sender
}

// MarshalJSON method of VideoMessage
func (m *VideoMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type               MessageType      `json:"type"`
		OriginalContentURL string           `json:"originalContentUrl"`
		PreviewImageURL    string           `json:"previewImageUrl"`
		QuickReply         *QuickReplyItems `json:"quickReply,omitempty"`
		Sender             *Sender          `json:"sender,omitempty"`
	}{
		Type:               MessageTypeVideo,
		OriginalContentURL: m.OriginalContentURL,
		PreviewImageURL:    m.PreviewImageURL,
		QuickReply:         m.quickReplyitems,
		Sender:             m.sender,
	})
}

// WithQuickReplies method of VideoMessage
func (m *VideoMessage) WithQuickReplies(items *QuickReplyItems) SendingMessage {
	m.quickReplyitems = items
	return m
}

// WithSender method of VideoMessage
func (m *VideoMessage) WithSender(sender *Sender) SendingMessage {
	m.sender = sender
	return m
}

// AddEmoji method of VideoMessage
func (m *VideoMessage) AddEmoji(emoji *Emoji) SendingMessage {
	return m
}

// AudioMessage type
type AudioMessage struct {
	ID                 string
	OriginalContentURL string
	Duration           int

	quickReplyitems *QuickReplyItems
	sender          *Sender
}

// MarshalJSON method of AudioMessage
func (m *AudioMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type               MessageType      `json:"type"`
		OriginalContentURL string           `json:"originalContentUrl"`
		Duration           int              `json:"duration"`
		QuickReply         *QuickReplyItems `json:"quickReply,omitempty"`
		Sender             *Sender          `json:"sender,omitempty"`
	}{
		Type:               MessageTypeAudio,
		OriginalContentURL: m.OriginalContentURL,
		Duration:           m.Duration,
		QuickReply:         m.quickReplyitems,
		Sender:             m.sender,
	})
}

// WithQuickReplies method of AudioMessage
func (m *AudioMessage) WithQuickReplies(items *QuickReplyItems) SendingMessage {
	m.quickReplyitems = items
	return m
}

// WithSender method of AudioMessage
func (m *AudioMessage) WithSender(sender *Sender) SendingMessage {
	m.sender = sender
	return m
}

// AddEmoji method of AudioMessage
func (m *AudioMessage) AddEmoji(emoji *Emoji) SendingMessage {
	return m
}

// FileMessage type
type FileMessage struct {
	ID       string
	FileName string
	FileSize int
}

// LocationMessage type
type LocationMessage struct {
	ID        string
	Title     string
	Address   string
	Latitude  float64
	Longitude float64

	quickReplyitems *QuickReplyItems
	sender          *Sender
}

// MarshalJSON method of LocationMessage
func (m *LocationMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type       MessageType      `json:"type"`
		Title      string           `json:"title"`
		Address    string           `json:"address"`
		Latitude   float64          `json:"latitude"`
		Longitude  float64          `json:"longitude"`
		QuickReply *QuickReplyItems `json:"quickReply,omitempty"`
		Sender     *Sender          `json:"sender,omitempty"`
	}{
		Type:       MessageTypeLocation,
		Title:      m.Title,
		Address:    m.Address,
		Latitude:   m.Latitude,
		Longitude:  m.Longitude,
		QuickReply: m.quickReplyitems,
		Sender:     m.sender,
	})
}

// WithQuickReplies method of LocationMessage
func (m *LocationMessage) WithQuickReplies(items *QuickReplyItems) SendingMessage {
	m.quickReplyitems = items
	return m
}

// WithSender method of LocationMessage
func (m *LocationMessage) WithSender(sender *Sender) SendingMessage {
	m.sender = sender
	return m
}

// AddEmoji method of LocationMessage
func (m *LocationMessage) AddEmoji(emoji *Emoji) SendingMessage {
	return m
}

// StickerMessage type
type StickerMessage struct {
	ID                  string
	PackageID           string
	StickerID           string
	StickerResourceType StickerResourceType
	Keywords            []string

	quickReplyitems *QuickReplyItems
	sender          *Sender
}

// MarshalJSON method of StickerMessage
func (m *StickerMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type                MessageType         `json:"type"`
		PackageID           string              `json:"packageId"`
		StickerID           string              `json:"stickerId"`
		StickerResourceType StickerResourceType `json:"stickerResourceType,omitempty"`
		Keywords            []string            `json:"keywords,omitempty"`
		QuickReply          *QuickReplyItems    `json:"quickReply,omitempty"`
		Sender              *Sender             `json:"sender,omitempty"`
	}{
		Type:                MessageTypeSticker,
		PackageID:           m.PackageID,
		StickerID:           m.StickerID,
		StickerResourceType: m.StickerResourceType,
		Keywords:            m.Keywords,
		QuickReply:          m.quickReplyitems,
		Sender:              m.sender,
	})
}

// WithQuickReplies method of StickerMessage
func (m *StickerMessage) WithQuickReplies(items *QuickReplyItems) SendingMessage {
	m.quickReplyitems = items
	return m
}

// WithSender method of StickerMessage
func (m *StickerMessage) WithSender(sender *Sender) SendingMessage {
	m.sender = sender
	return m
}

// AddEmoji method of StickerMessage
func (m *StickerMessage) AddEmoji(emoji *Emoji) SendingMessage {
	return m
}

// TemplateMessage type
type TemplateMessage struct {
	AltText  string
	Template Template

	quickReplyitems *QuickReplyItems
	sender          *Sender
}

// MarshalJSON method of TemplateMessage
func (m *TemplateMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type       MessageType      `json:"type"`
		AltText    string           `json:"altText"`
		Template   Template         `json:"template"`
		QuickReply *QuickReplyItems `json:"quickReply,omitempty"`
		Sender     *Sender          `json:"sender,omitempty"`
	}{
		Type:       MessageTypeTemplate,
		AltText:    m.AltText,
		Template:   m.Template,
		QuickReply: m.quickReplyitems,
		Sender:     m.sender,
	})
}

// WithQuickReplies method of TemplateMessage
func (m *TemplateMessage) WithQuickReplies(items *QuickReplyItems) SendingMessage {
	m.quickReplyitems = items
	return m
}

// WithSender method of TemplateMessage
func (m *TemplateMessage) WithSender(sender *Sender) SendingMessage {
	m.sender = sender
	return m
}

// AddEmoji method of TemplateMessage
func (m *TemplateMessage) AddEmoji(emoji *Emoji) SendingMessage {
	return m
}

// ImagemapMessage type
type ImagemapMessage struct {
	BaseURL  string
	AltText  string
	BaseSize ImagemapBaseSize
	Actions  []ImagemapAction
	Video    *ImagemapVideo

	quickReplyitems *QuickReplyItems
	sender          *Sender
}

// MarshalJSON method of ImagemapMessage
func (m *ImagemapMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type       MessageType      `json:"type"`
		BaseURL    string           `json:"baseUrl"`
		AltText    string           `json:"altText"`
		BaseSize   ImagemapBaseSize `json:"baseSize"`
		Actions    []ImagemapAction `json:"actions"`
		Video      *ImagemapVideo   `json:"video,omitempty"`
		QuickReply *QuickReplyItems `json:"quickReply,omitempty"`
		Sender     *Sender          `json:"sender,omitempty"`
	}{
		Type:       MessageTypeImagemap,
		BaseURL:    m.BaseURL,
		AltText:    m.AltText,
		BaseSize:   m.BaseSize,
		Actions:    m.Actions,
		Video:      m.Video,
		QuickReply: m.quickReplyitems,
		Sender:     m.sender,
	})
}

// WithVideo method
func (m *ImagemapMessage) WithVideo(video *ImagemapVideo) *ImagemapMessage {
	m.Video = video
	return m
}

// WithQuickReplies method of ImagemapMessage
func (m *ImagemapMessage) WithQuickReplies(items *QuickReplyItems) SendingMessage {
	m.quickReplyitems = items
	return m
}

// WithSender method of ImagemapMessage
func (m *ImagemapMessage) WithSender(sender *Sender) SendingMessage {
	m.sender = sender
	return m
}

// AddEmoji method of ImagemapMessage
func (m *ImagemapMessage) AddEmoji(emoji *Emoji) SendingMessage {
	return m
}

// FlexMessage type
type FlexMessage struct {
	AltText  string
	Contents FlexContainer

	quickReplyitems *QuickReplyItems
	sender          *Sender
}

// MarshalJSON method of FlexMessage
func (m *FlexMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type       MessageType      `json:"type"`
		AltText    string           `json:"altText"`
		Contents   interface{}      `json:"contents"`
		QuickReply *QuickReplyItems `json:"quickReply,omitempty"`
		Sender     *Sender          `json:"sender,omitempty"`
	}{
		Type:       MessageTypeFlex,
		AltText:    m.AltText,
		Contents:   m.Contents,
		QuickReply: m.quickReplyitems,
		Sender:     m.sender,
	})
}

// WithQuickReplies method of FlexMessage
func (m *FlexMessage) WithQuickReplies(items *QuickReplyItems) SendingMessage {
	m.quickReplyitems = items
	return m
}

// WithSender method of FlexMessage
func (m *FlexMessage) WithSender(sender *Sender) SendingMessage {
	m.sender = sender
	return m
}

// AddEmoji method of FlexMessage
func (m *FlexMessage) AddEmoji(emoji *Emoji) SendingMessage {
	return m
}

// Message implements Message interface
func (*FileMessage) Message() {}

// Message implements Message interface
func (*TextMessage) Message() {}

// Message implements Message interface
func (*ImageMessage) Message() {}

// Message implements Message interface
func (*VideoMessage) Message() {}

// Message implements Message interface
func (*AudioMessage) Message() {}

// Message implements Message interface
func (*LocationMessage) Message() {}

// Message implements Message interface
func (*StickerMessage) Message() {}

// Message implements Message interface
func (*TemplateMessage) Message() {}

// Message implements Message interface
func (*ImagemapMessage) Message() {}

// Message implements Message interface
func (*FlexMessage) Message() {}

// NewTextMessage function
func NewTextMessage(content string) *TextMessage {
	return &TextMessage{
		Text: content,
	}
}

// NewImageMessage function
func NewImageMessage(originalContentURL, previewImageURL string) *ImageMessage {
	return &ImageMessage{
		OriginalContentURL: originalContentURL,
		PreviewImageURL:    previewImageURL,
	}
}

// NewVideoMessage function
func NewVideoMessage(originalContentURL, previewImageURL string) *VideoMessage {
	return &VideoMessage{
		OriginalContentURL: originalContentURL,
		PreviewImageURL:    previewImageURL,
	}
}

// NewAudioMessage function
func NewAudioMessage(originalContentURL string, duration int) *AudioMessage {
	return &AudioMessage{
		OriginalContentURL: originalContentURL,
		Duration:           duration,
	}
}

// NewLocationMessage function
func NewLocationMessage(title, address string, latitude, longitude float64) *LocationMessage {
	return &LocationMessage{
		Title:     title,
		Address:   address,
		Latitude:  latitude,
		Longitude: longitude,
	}
}

// NewStickerMessage function
func NewStickerMessage(packageID, stickerID string) *StickerMessage {
	return &StickerMessage{
		PackageID: packageID,
		StickerID: stickerID,
	}
}

// NewTemplateMessage function
func NewTemplateMessage(altText string, template Template) *TemplateMessage {
	return &TemplateMessage{
		AltText:  altText,
		Template: template,
	}
}

// NewImagemapMessage function
func NewImagemapMessage(baseURL, altText string, baseSize ImagemapBaseSize, actions ...ImagemapAction) *ImagemapMessage {
	return &ImagemapMessage{
		BaseURL:  baseURL,
		AltText:  altText,
		BaseSize: baseSize,
		Actions:  actions,
	}
}

// NewFlexMessage function
func NewFlexMessage(altText string, contents FlexContainer) *FlexMessage {
	return &FlexMessage{
		AltText:  altText,
		Contents: contents,
	}
}
