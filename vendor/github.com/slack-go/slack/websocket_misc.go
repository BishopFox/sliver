package slack

import (
	"encoding/json"
	"fmt"
)

// AckMessage is used for messages received in reply to other messages
type AckMessage struct {
	ReplyTo   int    `json:"reply_to"`
	Timestamp string `json:"ts"`
	Text      string `json:"text"`
	RTMResponse
}

// RTMResponse encapsulates response details as returned by the Slack API
type RTMResponse struct {
	Ok    bool      `json:"ok"`
	Error *RTMError `json:"error"`
}

// RTMError encapsulates error information as returned by the Slack API
type RTMError struct {
	Code int
	Msg  string
}

func (s RTMError) Error() string {
	return fmt.Sprintf("Code %d - %s", s.Code, s.Msg)
}

// MessageEvent represents a Slack Message (used as the event type for an incoming message)
type MessageEvent Message

// RTMEvent is the main wrapper. You will find all the other messages attached
type RTMEvent struct {
	Type string
	Data interface{}
}

// HelloEvent represents the hello event
type HelloEvent struct{}

// PresenceChangeEvent represents the presence change event
type PresenceChangeEvent struct {
	Type     string   `json:"type"`
	Presence string   `json:"presence"`
	User     string   `json:"user"`
	Users    []string `json:"users"`
}

// UserTypingEvent represents the user typing event
type UserTypingEvent struct {
	Type    string `json:"type"`
	User    string `json:"user"`
	Channel string `json:"channel"`
}

// PrefChangeEvent represents a user preferences change event
type PrefChangeEvent struct {
	Type  string          `json:"type"`
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}

// ManualPresenceChangeEvent represents the manual presence change event
type ManualPresenceChangeEvent struct {
	Type     string `json:"type"`
	Presence string `json:"presence"`
}

// UserChangeEvent represents the user change event
type UserChangeEvent struct {
	Type    string `json:"type"`
	User    User   `json:"user"`
	CacheTS int64  `json:"cache_ts"`
	EventTS string `json:"event_ts"`
}

// UserStatusChangedEvent represents the user status changed event
type UserStatusChangedEvent struct {
	Type    string `json:"type"`
	User    User   `json:"user"`
	CacheTS int64  `json:"cache_ts"`
	EventTS string `json:"event_ts"`
}

// UserHuddleChangedEvent represents the user huddle changed event
type UserHuddleChangedEvent struct {
	Type    string `json:"type"`
	User    User   `json:"user"`
	CacheTS int64  `json:"cache_ts"`
	EventTS string `json:"event_ts"`
}

// UserProfileChangedEvent represents the user profile changed event
type UserProfileChangedEvent struct {
	Type    string `json:"type"`
	User    User   `json:"user"`
	CacheTS int64  `json:"cache_ts"`
	EventTS string `json:"event_ts"`
}

// EmojiChangedEvent represents the emoji changed event
type EmojiChangedEvent struct {
	Type           string   `json:"type"`
	SubType        string   `json:"subtype"`
	Name           string   `json:"name"`
	Names          []string `json:"names"`
	Value          string   `json:"value"`
	EventTimestamp string   `json:"event_ts"`
}

// CommandsChangedEvent represents the commands changed event
type CommandsChangedEvent struct {
	Type           string `json:"type"`
	EventTimestamp string `json:"event_ts"`
}

// EmailDomainChangedEvent represents the email domain changed event
type EmailDomainChangedEvent struct {
	Type           string `json:"type"`
	EventTimestamp string `json:"event_ts"`
	EmailDomain    string `json:"email_domain"`
}

// BotAddedEvent represents the bot added event
type BotAddedEvent struct {
	Type string `json:"type"`
	Bot  Bot    `json:"bot"`
}

// BotChangedEvent represents the bot changed event
type BotChangedEvent struct {
	Type string `json:"type"`
	Bot  Bot    `json:"bot"`
}

// AccountsChangedEvent represents the accounts changed event
type AccountsChangedEvent struct {
	Type string `json:"type"`
}

// ReconnectUrlEvent represents the receiving reconnect url event
type ReconnectUrlEvent struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// MemberJoinedChannelEvent, a user joined a public or private channel
type MemberJoinedChannelEvent struct {
	Type        string `json:"type"`
	User        string `json:"user"`
	Channel     string `json:"channel"`
	ChannelType string `json:"channel_type"`
	Team        string `json:"team"`
	Inviter     string `json:"inviter"`
}

// MemberLeftChannelEvent a user left a public or private channel
type MemberLeftChannelEvent struct {
	Type        string `json:"type"`
	User        string `json:"user"`
	Channel     string `json:"channel"`
	ChannelType string `json:"channel_type"`
	Team        string `json:"team"`
}

// ChannelUpdatedEvent is fired when a channel's properties are updated (tabs, meeting
// notes, etc.).
type ChannelUpdatedEvent struct {
	Type     string         `json:"type"`
	Updates  map[string]any `json:"updates"`
	Channel  string         `json:"channel"`
	Channels []string       `json:"channels"`
	EventTS  string         `json:"event_ts"`
	TS       string         `json:"ts"`
}

// SHRoomRecording holds recording metadata for a Slack Call/Huddle room.
type SHRoomRecording struct {
	CanRecordSummary string `json:"can_record_summary,omitempty"`
}

// SHRoom represents a Slack Huddle/Call room.
type SHRoom struct {
	ID                         string                    `json:"id"`
	Name                       *string                   `json:"name"` // nullable in Slack's response
	MediaServer                string                    `json:"media_server"`
	CreatedBy                  string                    `json:"created_by"`
	DateStart                  int64                     `json:"date_start"`
	DateEnd                    int64                     `json:"date_end"`
	Participants               []string                  `json:"participants"`
	ParticipantHistory         []string                  `json:"participant_history"`
	ParticipantsEvents         map[string]map[string]any `json:"participants_events,omitempty"`
	ParticipantsCameraOn       []string                  `json:"participants_camera_on"`
	ParticipantsCameraOff      []string                  `json:"participants_camera_off"`
	ParticipantsScreenshareOn  []string                  `json:"participants_screenshare_on"`
	ParticipantsScreenshareOff []string                  `json:"participants_screenshare_off"`
	CanvasThreadTS             string                    `json:"canvas_thread_ts,omitempty"`
	ThreadRootTS               string                    `json:"thread_root_ts,omitempty"`
	Channels                   []string                  `json:"channels"`
	IsDMCall                   bool                      `json:"is_dm_call"`
	WasRejected                bool                      `json:"was_rejected"`
	WasMissed                  bool                      `json:"was_missed"`
	WasAccepted                bool                      `json:"was_accepted"`
	HasEnded                   bool                      `json:"has_ended"`
	BackgroundID               string                    `json:"background_id,omitempty"`
	CanvasBackground           string                    `json:"canvas_background,omitempty"`
	IsPrewarmed                bool                      `json:"is_prewarmed,omitempty"`
	IsScheduled                bool                      `json:"is_scheduled,omitempty"`
	Recording                  *SHRoomRecording          `json:"recording,omitempty"`
	Locale                     string                    `json:"locale,omitempty"`
	AttachedFileIDs            []string                  `json:"attached_file_ids,omitempty"`
	MediaBackendType           string                    `json:"media_backend_type"`
	DisplayID                  string                    `json:"display_id,omitempty"`
	ExternalUniqueID           string                    `json:"external_unique_id"`
	AppID                      string                    `json:"app_id"`
	CallFamily                 string                    `json:"call_family,omitempty"`
	HuddleLink                 string                    `json:"huddle_link,omitempty"`
}

// SHRoomHuddle holds the huddle-specific metadata on sh_room events.
type SHRoomHuddle struct {
	ChannelID string `json:"channel_id"`
}

// SHRoomJoinEvent is fired when a user joins a Slack Call/Huddle room.
type SHRoomJoinEvent struct {
	Type    string        `json:"type"`
	Room    SHRoom        `json:"room"`
	User    string        `json:"user"`
	Huddle  *SHRoomHuddle `json:"huddle,omitempty"`
	EventTS string        `json:"event_ts"`
	TS      string        `json:"ts"`
}

// SHRoomLeaveEvent is fired when a user leaves a Slack Call/Huddle room.
type SHRoomLeaveEvent struct {
	Type    string        `json:"type"`
	Room    SHRoom        `json:"room"`
	User    string        `json:"user"`
	Huddle  *SHRoomHuddle `json:"huddle,omitempty"`
	EventTS string        `json:"event_ts"`
	TS      string        `json:"ts"`
}

// SHRoomUpdateEvent is fired when a Slack Call/Huddle room is updated.
type SHRoomUpdateEvent struct {
	Type    string        `json:"type"`
	Room    SHRoom        `json:"room"`
	User    string        `json:"user"`
	Huddle  *SHRoomHuddle `json:"huddle,omitempty"`
	EventTS string        `json:"event_ts"`
	TS      string        `json:"ts"`
}

// AppsUninstalledEvent represents the apps_uninstalled event sent via RTM
// when one or more apps are uninstalled from the workspace.
type AppsUninstalledEvent struct {
	Type string `json:"type"`
}

// ActivityEvent represents the activity event sent via RTM. This is an
// internal Slack event that fires during normal workspace usage (e.g. new
// messages, bundle updates).
type ActivityEvent struct {
	Type           string          `json:"type"`
	SubType        string          `json:"subtype"`
	Key            string          `json:"key"`
	Entry          json.RawMessage `json:"entry"`
	EventTimestamp string          `json:"event_ts"`
}

// BadgeCountsUpdatedEvent represents the badge_counts_updated event sent via
// RTM when notification badge counts change.
type BadgeCountsUpdatedEvent struct {
	Type string `json:"type"`
}
