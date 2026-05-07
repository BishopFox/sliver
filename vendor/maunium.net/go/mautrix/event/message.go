// Copyright (c) 2023 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package event

import (
	"encoding/json"
	"html"
	"slices"
	"strconv"
	"strings"

	"maunium.net/go/mautrix/crypto/attachment"
	"maunium.net/go/mautrix/id"
)

// MessageType is the sub-type of a m.room.message event.
// https://spec.matrix.org/v1.2/client-server-api/#mroommessage-msgtypes
type MessageType string

func (mt MessageType) IsText() bool {
	switch mt {
	case MsgText, MsgNotice, MsgEmote:
		return true
	default:
		return false
	}
}

func (mt MessageType) IsMedia() bool {
	switch mt {
	case MsgImage, MsgVideo, MsgAudio, MsgFile, CapMsgSticker:
		return true
	default:
		return false
	}
}

// Msgtypes
const (
	MsgText     MessageType = "m.text"
	MsgEmote    MessageType = "m.emote"
	MsgNotice   MessageType = "m.notice"
	MsgImage    MessageType = "m.image"
	MsgLocation MessageType = "m.location"
	MsgVideo    MessageType = "m.video"
	MsgAudio    MessageType = "m.audio"
	MsgFile     MessageType = "m.file"

	MsgVerificationRequest MessageType = "m.key.verification.request"

	MsgBeeperGallery MessageType = "com.beeper.gallery"
)

// Format specifies the format of the formatted_body in m.room.message events.
// https://spec.matrix.org/v1.2/client-server-api/#mroommessage-msgtypes
type Format string

// Message formats
const (
	FormatHTML Format = "org.matrix.custom.html"
)

// RedactionEventContent represents the content of a m.room.redaction message event.
//
// https://spec.matrix.org/v1.8/client-server-api/#mroomredaction
type RedactionEventContent struct {
	Reason string `json:"reason,omitempty"`

	// The event ID is here as of room v11. In old servers it may only be at the top level.
	Redacts id.EventID `json:"redacts,omitempty"`
}

// ReactionEventContent represents the content of a m.reaction message event.
// This is not yet in a spec release, see https://github.com/matrix-org/matrix-doc/pull/1849
type ReactionEventContent struct {
	RelatesTo RelatesTo `json:"m.relates_to"`
}

func (content *ReactionEventContent) GetRelatesTo() *RelatesTo {
	return &content.RelatesTo
}

func (content *ReactionEventContent) OptionalGetRelatesTo() *RelatesTo {
	return &content.RelatesTo
}

func (content *ReactionEventContent) SetRelatesTo(rel *RelatesTo) {
	content.RelatesTo = *rel
}

// MessageEventContent represents the content of a m.room.message event.
//
// It is also used to represent m.sticker events, as they are equivalent to m.room.message
// with the exception of the msgtype field.
//
// https://spec.matrix.org/v1.2/client-server-api/#mroommessage
type MessageEventContent struct {
	// Base m.room.message fields
	MsgType MessageType `json:"msgtype,omitempty"`
	Body    string      `json:"body"`

	// Extra fields for text types
	Format        Format `json:"format,omitempty"`
	FormattedBody string `json:"formatted_body,omitempty"`

	// Extra field for m.location
	GeoURI string `json:"geo_uri,omitempty"`

	// Extra fields for media types
	URL  id.ContentURIString `json:"url,omitempty"`
	Info *FileInfo           `json:"info,omitempty"`
	File *EncryptedFileInfo  `json:"file,omitempty"`

	FileName string `json:"filename,omitempty"`

	Mentions *Mentions `json:"m.mentions,omitempty"`

	// Edits and relations
	NewContent *MessageEventContent `json:"m.new_content,omitempty"`
	RelatesTo  *RelatesTo           `json:"m.relates_to,omitempty"`

	// In-room verification
	To         id.UserID            `json:"to,omitempty"`
	FromDevice id.DeviceID          `json:"from_device,omitempty"`
	Methods    []VerificationMethod `json:"methods,omitempty"`

	replyFallbackRemoved bool

	MessageSendRetry         *BeeperRetryMetadata     `json:"com.beeper.message_send_retry,omitempty"`
	BeeperGalleryImages      []*MessageEventContent   `json:"com.beeper.gallery.images,omitempty"`
	BeeperGalleryCaption     string                   `json:"com.beeper.gallery.caption,omitempty"`
	BeeperGalleryCaptionHTML string                   `json:"com.beeper.gallery.caption_html,omitempty"`
	BeeperPerMessageProfile  *BeeperPerMessageProfile `json:"com.beeper.per_message_profile,omitempty"`

	BeeperLinkPreviews []*BeeperLinkPreview `json:"com.beeper.linkpreviews,omitempty"`

	BeeperDisappearingTimer *BeeperDisappearingTimer `json:"com.beeper.disappearing_timer,omitempty"`

	MSC1767Audio *MSC1767Audio `json:"org.matrix.msc1767.audio,omitempty"`
	MSC3245Voice *MSC3245Voice `json:"org.matrix.msc3245.voice,omitempty"`

	MSC4332BotCommand *BotCommandInput `json:"org.matrix.msc4332.command,omitempty"`
}

func (content *MessageEventContent) GetCapMsgType() CapabilityMsgType {
	switch content.MsgType {
	case CapMsgSticker:
		return CapMsgSticker
	case "":
		if content.URL != "" || content.File != nil {
			return CapMsgSticker
		}
	case MsgImage:
		return MsgImage
	case MsgAudio:
		if content.MSC3245Voice != nil {
			return CapMsgVoice
		}
		return MsgAudio
	case MsgVideo:
		if content.Info != nil && content.Info.MauGIF {
			return CapMsgGIF
		}
		return MsgVideo
	case MsgFile:
		return MsgFile
	}
	return ""
}

func (content *MessageEventContent) GetFileName() string {
	if content.FileName != "" {
		return content.FileName
	}
	return content.Body
}

func (content *MessageEventContent) GetCaption() string {
	if content.FileName != "" && content.Body != "" && content.Body != content.FileName {
		return content.Body
	}
	return ""
}

func (content *MessageEventContent) GetFormattedCaption() string {
	if content.Format == FormatHTML && content.FormattedBody != "" {
		return content.FormattedBody
	}
	return ""
}

func (content *MessageEventContent) GetRelatesTo() *RelatesTo {
	if content.RelatesTo == nil {
		content.RelatesTo = &RelatesTo{}
	}
	return content.RelatesTo
}

func (content *MessageEventContent) OptionalGetRelatesTo() *RelatesTo {
	return content.RelatesTo
}

func (content *MessageEventContent) SetRelatesTo(rel *RelatesTo) {
	content.RelatesTo = rel
}

func (content *MessageEventContent) SetEdit(original id.EventID) {
	newContent := *content
	content.NewContent = &newContent
	content.RelatesTo = (&RelatesTo{}).SetReplace(original)
	if content.MsgType == MsgText || content.MsgType == MsgNotice {
		content.Body = "* " + content.Body
		content.Mentions = &Mentions{}
		if content.Format == FormatHTML && len(content.FormattedBody) > 0 {
			content.FormattedBody = "* " + content.FormattedBody
		}
		// If the message is long, remove most of the useless edit fallback to avoid event size issues.
		if len(content.Body) > 10000 {
			content.FormattedBody = ""
			content.Format = ""
			content.Body = content.Body[:50] + "[edit fallback cut…]"
		}
	}
}

// TextToHTML converts the given text to a HTML-safe representation by escaping HTML characters
// and replacing newlines with <br/> tags.
func TextToHTML(text string) string {
	return strings.ReplaceAll(html.EscapeString(text), "\n", "<br/>")
}

// ReverseTextToHTML reverses the modifications made by TextToHTML, i.e. replaces <br/> tags with newlines
// and unescapes HTML escape codes. For actually parsing HTML, use the format package instead.
func ReverseTextToHTML(input string) string {
	return html.UnescapeString(strings.ReplaceAll(input, "<br/>", "\n"))
}

func (content *MessageEventContent) EnsureHasHTML() {
	if len(content.FormattedBody) == 0 || content.Format != FormatHTML {
		content.FormattedBody = TextToHTML(content.Body)
		content.Format = FormatHTML
	}
}

func (content *MessageEventContent) GetFile() *EncryptedFileInfo {
	if content.File == nil {
		content.File = &EncryptedFileInfo{}
	}
	return content.File
}

func (content *MessageEventContent) GetInfo() *FileInfo {
	if content.Info == nil {
		content.Info = &FileInfo{}
	}
	return content.Info
}

type Mentions struct {
	UserIDs []id.UserID `json:"user_ids,omitempty"`
	Room    bool        `json:"room,omitempty"`
}

func (m *Mentions) Add(userID id.UserID) {
	if userID != "" && !slices.Contains(m.UserIDs, userID) {
		m.UserIDs = append(m.UserIDs, userID)
	}
}

func (m *Mentions) Has(userID id.UserID) bool {
	return m != nil && slices.Contains(m.UserIDs, userID)
}

func (m *Mentions) Merge(other *Mentions) *Mentions {
	if m == nil {
		return other
	} else if other == nil {
		return m
	}
	return &Mentions{
		UserIDs: slices.Concat(m.UserIDs, other.UserIDs),
		Room:    m.Room || other.Room,
	}
}

type EncryptedFileInfo struct {
	attachment.EncryptedFile
	URL id.ContentURIString `json:"url"`
}

type FileInfo struct {
	MimeType      string
	ThumbnailInfo *FileInfo
	ThumbnailURL  id.ContentURIString
	ThumbnailFile *EncryptedFileInfo

	Blurhash     string
	AnoaBlurhash string

	MauGIF     bool
	IsAnimated bool

	Width    int
	Height   int
	Duration int
	Size     int
}

type serializableFileInfo struct {
	MimeType      string                `json:"mimetype,omitempty"`
	ThumbnailInfo *serializableFileInfo `json:"thumbnail_info,omitempty"`
	ThumbnailURL  id.ContentURIString   `json:"thumbnail_url,omitempty"`
	ThumbnailFile *EncryptedFileInfo    `json:"thumbnail_file,omitempty"`

	Blurhash     string `json:"blurhash,omitempty"`
	AnoaBlurhash string `json:"xyz.amorgan.blurhash,omitempty"`

	MauGIF     bool `json:"fi.mau.gif,omitempty"`
	IsAnimated bool `json:"is_animated,omitempty"`

	Width    json.Number `json:"w,omitempty"`
	Height   json.Number `json:"h,omitempty"`
	Duration json.Number `json:"duration,omitempty"`
	Size     json.Number `json:"size,omitempty"`
}

func (sfi *serializableFileInfo) CopyFrom(fileInfo *FileInfo) *serializableFileInfo {
	if fileInfo == nil {
		return nil
	}
	*sfi = serializableFileInfo{
		MimeType:      fileInfo.MimeType,
		ThumbnailURL:  fileInfo.ThumbnailURL,
		ThumbnailInfo: (&serializableFileInfo{}).CopyFrom(fileInfo.ThumbnailInfo),
		ThumbnailFile: fileInfo.ThumbnailFile,

		MauGIF:     fileInfo.MauGIF,
		IsAnimated: fileInfo.IsAnimated,

		Blurhash:     fileInfo.Blurhash,
		AnoaBlurhash: fileInfo.AnoaBlurhash,
	}
	if fileInfo.Width > 0 {
		sfi.Width = json.Number(strconv.Itoa(fileInfo.Width))
	}
	if fileInfo.Height > 0 {
		sfi.Height = json.Number(strconv.Itoa(fileInfo.Height))
	}
	if fileInfo.Size > 0 {
		sfi.Size = json.Number(strconv.Itoa(fileInfo.Size))

	}
	if fileInfo.Duration > 0 {
		sfi.Duration = json.Number(strconv.Itoa(int(fileInfo.Duration)))
	}
	return sfi
}

func (sfi *serializableFileInfo) CopyTo(fileInfo *FileInfo) {
	*fileInfo = FileInfo{
		Width:         numberToInt(sfi.Width),
		Height:        numberToInt(sfi.Height),
		Size:          numberToInt(sfi.Size),
		Duration:      numberToInt(sfi.Duration),
		MimeType:      sfi.MimeType,
		ThumbnailURL:  sfi.ThumbnailURL,
		ThumbnailFile: sfi.ThumbnailFile,
		MauGIF:        sfi.MauGIF,
		IsAnimated:    sfi.IsAnimated,
		Blurhash:      sfi.Blurhash,
		AnoaBlurhash:  sfi.AnoaBlurhash,
	}
	if sfi.ThumbnailInfo != nil {
		fileInfo.ThumbnailInfo = &FileInfo{}
		sfi.ThumbnailInfo.CopyTo(fileInfo.ThumbnailInfo)
	}
}

func (fileInfo *FileInfo) UnmarshalJSON(data []byte) error {
	sfi := &serializableFileInfo{}
	if err := json.Unmarshal(data, sfi); err != nil {
		return err
	}
	sfi.CopyTo(fileInfo)
	return nil
}

func (fileInfo *FileInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal((&serializableFileInfo{}).CopyFrom(fileInfo))
}

func numberToInt(val json.Number) int {
	f64, _ := val.Float64()
	if f64 > 0 {
		return int(f64)
	}
	return 0
}

func (fileInfo *FileInfo) GetThumbnailInfo() *FileInfo {
	if fileInfo.ThumbnailInfo == nil {
		fileInfo.ThumbnailInfo = &FileInfo{}
	}
	return fileInfo.ThumbnailInfo
}
