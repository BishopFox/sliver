package lark

import "encoding/json"

// EventChallenge request of add event hook
type EventChallenge struct {
	Token     string `json:"token,omitempty"`
	Challenge string `json:"challenge,omitempty"`
	Type      string `json:"type,omitempty"`
}

// EventChallengeReq is deprecated. Keep for legacy versions.
type EventChallengeReq = EventChallenge

// EncryptedReq request of encrypted challenge
type EncryptedReq struct {
	Encrypt string `json:"encrypt,omitempty"`
}

// EventCardCallback request of card
type EventCardCallback struct {
	AppID     string          `json:"app_id,omitempty"`
	TenantKey string          `json:"tenant_key,omitempty"`
	Token     string          `json:"token,omitempty"`
	OpenID    string          `json:"open_id,omitempty"`
	UserID    string          `json:"user_id,omitempty"`
	MessageID string          `json:"open_message_id,omitempty"`
	ChatID    string          `json:"open_chat_id,omitempty"`
	Action    EventCardAction `json:"action,omitempty"`
}

// EventCardAction .
type EventCardAction struct {
	Tag      string          `json:"tag,omitempty"`      // button, overflow, select_static, select_person, &datepicker
	Option   string          `json:"option,omitempty"`   // only for Overflow and SelectMenu
	Timezone string          `json:"timezone,omitempty"` // only for DatePicker
	Value    json.RawMessage `json:"value,omitempty"`    // for any elements with value
}
