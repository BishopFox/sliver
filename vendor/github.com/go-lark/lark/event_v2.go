package lark

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

// EventType definitions
const (
	EventTypeMessageReceived        = "im.message.receive_v1"
	EventTypeMessageRead            = "im.message.message_read_v1"
	EventTypeMessageRecalled        = "im.message.recalled_v1"
	EventTypeMessageReactionCreated = "im.message.reaction.created_v1"
	EventTypeMessageReactionDeleted = "im.message.reaction.deleted_v1"
	EventTypeChatDisbanded          = "im.chat.disbanded_v1"
	EventTypeUserAdded              = "im.chat.member.user.added_v1"
	EventTypeUserDeleted            = "im.chat.member.user.deleted_v1"
	EventTypeBotAdded               = "im.chat.member.bot.added_v1"
	EventTypeBotDeleted             = "im.chat.member.bot.deleted_v1"
	// not supported yet
	EventTypeChatUpdated   = "im.chat.updated_v1"
	EventTypeUserWithdrawn = "im.chat.member.user.withdrawn_v1"
)

// EventV2 handles events with v2 schema
type EventV2 struct {
	Schema string        `json:"schema,omitempty"`
	Header EventV2Header `json:"header,omitempty"`

	EventRaw json.RawMessage `json:"event,omitempty"`
	Event    interface{}     `json:"-"`
}

// EventV2Header .
type EventV2Header struct {
	EventID    string `json:"event_id,omitempty"`
	EventType  string `json:"event_type,omitempty"`
	CreateTime string `json:"create_time,omitempty"`
	Token      string `json:"token,omitempty"`
	AppID      string `json:"app_id,omitempty"`
	TenantKey  string `json:"tenant_key,omitempty"`
}

// EventV2UserID .
type EventV2UserID struct {
	UnionID string `json:"union_id,omitempty"`
	UserID  string `json:"user_id,omitempty"`
	OpenID  string `json:"open_id,omitempty"`
}

// PostEvent with event v2 format
// and it's part of EventV2 instead of package method
func (e EventV2) PostEvent(client *http.Client, hookURL string) (*http.Response, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(e)
	if err != nil {
		log.Printf("Encode json failed: %+v\n", err)
		return nil, err
	}
	resp, err := client.Post(hookURL, "application/json; charset=utf-8", buf)
	return resp, err
}

// GetEvent .
func (e EventV2) GetEvent(eventType string, body interface{}) error {
	if e.Header.EventType != eventType {
		return ErrEventTypeNotMatch
	}
	err := json.Unmarshal(e.EventRaw, &body)
	return err
}
