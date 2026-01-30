// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package event

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"mime"
	"slices"
	"strings"

	"go.mau.fi/util/exerrors"
	"go.mau.fi/util/jsontime"
	"go.mau.fi/util/ptr"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
)

type RoomFeatures struct {
	ID string `json:"id,omitempty"`

	// N.B. New fields need to be added to the Hash function to be included in the deduplication hash.

	Formatting    FormattingFeatureMap `json:"formatting,omitempty"`
	File          FileFeatureMap       `json:"file,omitempty"`
	State         StateFeatureMap      `json:"state,omitempty"`
	MemberActions MemberFeatureMap     `json:"member_actions,omitempty"`

	MaxTextLength int `json:"max_text_length,omitempty"`

	LocationMessage CapabilitySupportLevel `json:"location_message,omitempty"`
	Poll            CapabilitySupportLevel `json:"poll,omitempty"`
	Thread          CapabilitySupportLevel `json:"thread,omitempty"`
	Reply           CapabilitySupportLevel `json:"reply,omitempty"`

	Edit         CapabilitySupportLevel `json:"edit,omitempty"`
	EditMaxCount int                    `json:"edit_max_count,omitempty"`
	EditMaxAge   *jsontime.Seconds      `json:"edit_max_age,omitempty"`
	Delete       CapabilitySupportLevel `json:"delete,omitempty"`
	DeleteForMe  bool                   `json:"delete_for_me,omitempty"`
	DeleteMaxAge *jsontime.Seconds      `json:"delete_max_age,omitempty"`

	DisappearingTimer *DisappearingTimerCapability `json:"disappearing_timer,omitempty"`

	Reaction             CapabilitySupportLevel `json:"reaction,omitempty"`
	ReactionCount        int                    `json:"reaction_count,omitempty"`
	AllowedReactions     []string               `json:"allowed_reactions,omitempty"`
	CustomEmojiReactions bool                   `json:"custom_emoji_reactions,omitempty"`

	ReadReceipts          bool `json:"read_receipts,omitempty"`
	TypingNotifications   bool `json:"typing_notifications,omitempty"`
	Archive               bool `json:"archive,omitempty"`
	MarkAsUnread          bool `json:"mark_as_unread,omitempty"`
	DeleteChat            bool `json:"delete_chat,omitempty"`
	DeleteChatForEveryone bool `json:"delete_chat_for_everyone,omitempty"`

	PerMessageProfileRelay bool `json:"-"`
}

func (rf *RoomFeatures) GetID() string {
	if rf.ID != "" {
		return rf.ID
	}
	return base64.RawURLEncoding.EncodeToString(rf.Hash())
}

func (rf *RoomFeatures) Clone() *RoomFeatures {
	if rf == nil {
		return nil
	}
	clone := *rf
	clone.File = clone.File.Clone()
	clone.Formatting = maps.Clone(clone.Formatting)
	clone.State = clone.State.Clone()
	clone.MemberActions = clone.MemberActions.Clone()
	clone.EditMaxAge = ptr.Clone(clone.EditMaxAge)
	clone.DeleteMaxAge = ptr.Clone(clone.DeleteMaxAge)
	clone.DisappearingTimer = clone.DisappearingTimer.Clone()
	clone.AllowedReactions = slices.Clone(clone.AllowedReactions)
	return &clone
}

type MemberFeatureMap map[MemberAction]CapabilitySupportLevel

func (mfm MemberFeatureMap) Clone() MemberFeatureMap {
	return maps.Clone(mfm)
}

type MemberAction string

const (
	MemberActionBan          MemberAction = "ban"
	MemberActionKick         MemberAction = "kick"
	MemberActionLeave        MemberAction = "leave"
	MemberActionRevokeInvite MemberAction = "revoke_invite"
	MemberActionInvite       MemberAction = "invite"
)

type StateFeatureMap map[string]*StateFeatures

func (sfm StateFeatureMap) Clone() StateFeatureMap {
	dup := maps.Clone(sfm)
	for key, value := range dup {
		dup[key] = value.Clone()
	}
	return dup
}

type StateFeatures struct {
	Level CapabilitySupportLevel `json:"level"`
}

func (sf *StateFeatures) Clone() *StateFeatures {
	if sf == nil {
		return nil
	}
	clone := *sf
	return &clone
}

func (sf *StateFeatures) Hash() []byte {
	return sf.Level.Hash()
}

type FormattingFeatureMap map[FormattingFeature]CapabilitySupportLevel

type FileFeatureMap map[CapabilityMsgType]*FileFeatures

func (ffm FileFeatureMap) Clone() FileFeatureMap {
	dup := maps.Clone(ffm)
	for key, value := range dup {
		dup[key] = value.Clone()
	}
	return dup
}

type DisappearingTimerCapability struct {
	Types  []DisappearingType      `json:"types"`
	Timers []jsontime.Milliseconds `json:"timers,omitempty"`

	OmitEmptyTimer bool `json:"omit_empty_timer,omitempty"`
}

func (dtc *DisappearingTimerCapability) Clone() *DisappearingTimerCapability {
	if dtc == nil {
		return nil
	}
	clone := *dtc
	clone.Types = slices.Clone(clone.Types)
	clone.Timers = slices.Clone(clone.Timers)
	return &clone
}

func (dtc *DisappearingTimerCapability) Supports(content *BeeperDisappearingTimer) bool {
	if dtc == nil || content == nil || content.Type == DisappearingTypeNone {
		return true
	}
	return slices.Contains(dtc.Types, content.Type) && (dtc.Timers == nil || slices.Contains(dtc.Timers, content.Timer))
}

type CapabilityMsgType = MessageType

// Message types which are used for event capability signaling, but aren't real values for the msgtype field.
const (
	CapMsgVoice   CapabilityMsgType = "org.matrix.msc3245.voice"
	CapMsgGIF     CapabilityMsgType = "fi.mau.gif"
	CapMsgSticker CapabilityMsgType = "m.sticker"
)

type CapabilitySupportLevel int

func (csl CapabilitySupportLevel) Partial() bool {
	return csl >= CapLevelPartialSupport
}

func (csl CapabilitySupportLevel) Full() bool {
	return csl >= CapLevelFullySupported
}

func (csl CapabilitySupportLevel) Reject() bool {
	return csl <= CapLevelRejected
}

const (
	CapLevelRejected       CapabilitySupportLevel = -2 // The feature is unsupported and messages using it will be rejected.
	CapLevelDropped        CapabilitySupportLevel = -1 // The feature is unsupported and has no fallback. The message will go through, but data may be lost.
	CapLevelUnsupported    CapabilitySupportLevel = 0  // The feature is unsupported, but may have a fallback.
	CapLevelPartialSupport CapabilitySupportLevel = 1  // The feature is partially supported (e.g. it may be converted to a different format).
	CapLevelFullySupported CapabilitySupportLevel = 2  // The feature is fully supported and can be safely used.
)

type FormattingFeature string

const (
	FmtBold                FormattingFeature = "bold"                           // strong, b
	FmtItalic              FormattingFeature = "italic"                         // em, i
	FmtUnderline           FormattingFeature = "underline"                      // u
	FmtStrikethrough       FormattingFeature = "strikethrough"                  // del, s
	FmtInlineCode          FormattingFeature = "inline_code"                    // code
	FmtCodeBlock           FormattingFeature = "code_block"                     // pre + code
	FmtSyntaxHighlighting  FormattingFeature = "code_block.syntax_highlighting" // <pre><code class="language-...">
	FmtBlockquote          FormattingFeature = "blockquote"                     // blockquote
	FmtInlineLink          FormattingFeature = "inline_link"                    // a
	FmtUserLink            FormattingFeature = "user_link"                      // <a href="https://matrix.to/#/@...">
	FmtRoomLink            FormattingFeature = "room_link"                      // <a href="https://matrix.to/#/#...">
	FmtEventLink           FormattingFeature = "event_link"                     // <a href="https://matrix.to/#/!.../$...">
	FmtAtRoomMention       FormattingFeature = "at_room_mention"                // @room (no html tag)
	FmtUnorderedList       FormattingFeature = "unordered_list"                 // ul + li
	FmtOrderedList         FormattingFeature = "ordered_list"                   // ol + li
	FmtListStart           FormattingFeature = "ordered_list.start"             // <ol start="N">
	FmtListJumpValue       FormattingFeature = "ordered_list.jump_value"        // <li value="N">
	FmtCustomEmoji         FormattingFeature = "custom_emoji"                   // <img data-mx-emoticon>
	FmtSpoiler             FormattingFeature = "spoiler"                        // <span data-mx-spoiler>
	FmtSpoilerReason       FormattingFeature = "spoiler.reason"                 // <span data-mx-spoiler="...">
	FmtTextForegroundColor FormattingFeature = "color.foreground"               // <span data-mx-color="#...">
	FmtTextBackgroundColor FormattingFeature = "color.background"               // <span data-mx-bg-color="#...">
	FmtHorizontalLine      FormattingFeature = "horizontal_line"                // hr
	FmtHeaders             FormattingFeature = "headers"                        // h1, h2, h3, h4, h5, h6
	FmtSuperscript         FormattingFeature = "superscript"                    // sup
	FmtSubscript           FormattingFeature = "subscript"                      // sub
	FmtMath                FormattingFeature = "math"                           // <span data-mx-maths="...">
	FmtDetailsSummary      FormattingFeature = "details_summary"                // <details><summary>...</summary>...</details>
	FmtTable               FormattingFeature = "table"                          // table, thead, tbody, tr, th, td
)

type FileFeatures struct {
	// N.B. New fields need to be added to the Hash function to be included in the deduplication hash.

	MimeTypes map[string]CapabilitySupportLevel `json:"mime_types"`

	Caption          CapabilitySupportLevel `json:"caption,omitempty"`
	MaxCaptionLength int                    `json:"max_caption_length,omitempty"`

	MaxSize     int64             `json:"max_size,omitempty"`
	MaxWidth    int               `json:"max_width,omitempty"`
	MaxHeight   int               `json:"max_height,omitempty"`
	MaxDuration *jsontime.Seconds `json:"max_duration,omitempty"`

	ViewOnce bool `json:"view_once,omitempty"`
}

func (ff *FileFeatures) GetMimeSupport(inputType string) CapabilitySupportLevel {
	match, ok := ff.MimeTypes[inputType]
	if ok {
		return match
	}
	if strings.IndexByte(inputType, ';') != -1 {
		plainMime, _, _ := mime.ParseMediaType(inputType)
		if plainMime != "" {
			if match, ok = ff.MimeTypes[plainMime]; ok {
				return match
			}
		}
	}
	if slash := strings.IndexByte(inputType, '/'); slash > 0 {
		generalType := fmt.Sprintf("%s/*", inputType[:slash])
		if match, ok = ff.MimeTypes[generalType]; ok {
			return match
		}
	}
	match, ok = ff.MimeTypes["*/*"]
	if ok {
		return match
	}
	return CapLevelRejected
}

type hashable interface {
	Hash() []byte
}

func hashMap[Key ~string, Value hashable](w io.Writer, name string, data map[Key]Value) {
	keys := maps.Keys(data)
	slices.Sort(keys)
	exerrors.Must(w.Write([]byte(name)))
	for _, key := range keys {
		exerrors.Must(w.Write([]byte(key)))
		exerrors.Must(w.Write(data[key].Hash()))
		exerrors.Must(w.Write([]byte{0}))
	}
}

func hashValue(w io.Writer, name string, data hashable) {
	exerrors.Must(w.Write([]byte(name)))
	exerrors.Must(w.Write(data.Hash()))
}

func hashInt[T constraints.Integer](w io.Writer, name string, data T) {
	exerrors.Must(w.Write(binary.BigEndian.AppendUint64([]byte(name), uint64(data))))
}

func hashBool[T ~bool](w io.Writer, name string, data T) {
	exerrors.Must(w.Write([]byte(name)))
	if data {
		exerrors.Must(w.Write([]byte{1}))
	} else {
		exerrors.Must(w.Write([]byte{0}))
	}
}

func (csl CapabilitySupportLevel) Hash() []byte {
	return []byte{byte(csl + 128)}
}

func (rf *RoomFeatures) Hash() []byte {
	hasher := sha256.New()

	hashMap(hasher, "formatting", rf.Formatting)
	hashMap(hasher, "file", rf.File)
	hashMap(hasher, "state", rf.State)
	hashMap(hasher, "member_actions", rf.MemberActions)

	hashInt(hasher, "max_text_length", rf.MaxTextLength)

	hashValue(hasher, "location_message", rf.LocationMessage)
	hashValue(hasher, "poll", rf.Poll)
	hashValue(hasher, "thread", rf.Thread)
	hashValue(hasher, "reply", rf.Reply)

	hashValue(hasher, "edit", rf.Edit)
	hashInt(hasher, "edit_max_count", rf.EditMaxCount)
	hashInt(hasher, "edit_max_age", rf.EditMaxAge.Get())

	hashValue(hasher, "delete", rf.Delete)
	hashBool(hasher, "delete_for_me", rf.DeleteForMe)
	hashInt(hasher, "delete_max_age", rf.DeleteMaxAge.Get())
	hashValue(hasher, "disappearing_timer", rf.DisappearingTimer)

	hashValue(hasher, "reaction", rf.Reaction)
	hashInt(hasher, "reaction_count", rf.ReactionCount)
	hasher.Write([]byte("allowed_reactions"))
	for _, reaction := range rf.AllowedReactions {
		hasher.Write([]byte(reaction))
	}
	hashBool(hasher, "custom_emoji_reactions", rf.CustomEmojiReactions)

	hashBool(hasher, "read_receipts", rf.ReadReceipts)
	hashBool(hasher, "typing_notifications", rf.TypingNotifications)
	hashBool(hasher, "archive", rf.Archive)
	hashBool(hasher, "mark_as_unread", rf.MarkAsUnread)
	hashBool(hasher, "delete_chat", rf.DeleteChat)
	hashBool(hasher, "delete_chat_for_everyone", rf.DeleteChatForEveryone)

	return hasher.Sum(nil)
}

func (dtc *DisappearingTimerCapability) Hash() []byte {
	if dtc == nil {
		return nil
	}
	hasher := sha256.New()
	hasher.Write([]byte("types"))
	for _, t := range dtc.Types {
		hasher.Write([]byte(t))
	}
	hasher.Write([]byte("timers"))
	for _, timer := range dtc.Timers {
		hashInt(hasher, "", timer.Milliseconds())
	}
	return hasher.Sum(nil)
}

func (ff *FileFeatures) Hash() []byte {
	hasher := sha256.New()
	hashMap(hasher, "mime_types", ff.MimeTypes)
	hashValue(hasher, "caption", ff.Caption)
	hashInt(hasher, "max_caption_length", ff.MaxCaptionLength)
	hashInt(hasher, "max_size", ff.MaxSize)
	hashInt(hasher, "max_width", ff.MaxWidth)
	hashInt(hasher, "max_height", ff.MaxHeight)
	hashInt(hasher, "max_duration", ff.MaxDuration.Get())
	hashBool(hasher, "view_once", ff.ViewOnce)
	return hasher.Sum(nil)
}

func (ff *FileFeatures) Clone() *FileFeatures {
	if ff == nil {
		return nil
	}
	clone := *ff
	clone.MimeTypes = maps.Clone(clone.MimeTypes)
	clone.MaxDuration = ptr.Clone(clone.MaxDuration)
	return &clone
}
