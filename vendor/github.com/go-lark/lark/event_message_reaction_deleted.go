package lark

// EventV2MessageReactionDeleted .
type EventV2MessageReactionDeleted struct {
	MessageID    string        `json:"message_id,omitempty"`
	OperatorType string        `json:"operator_type,omitempty"`
	UserID       EventV2UserID `json:"user_id,omitempty"`
	AppID        string        `json:"app_id,omitempty"`
	ActionTime   string        `json:"action_time,omitempty"`
	ReactionType struct {
		EmojiType string `json:"emoji_type,omitempty"`
	} `json:"reaction_type,omitempty"`
}

// GetMessageReactionDeleted .
func (e EventV2) GetMessageReactionDeleted() (*EventV2MessageReactionDeleted, error) {
	var body EventV2MessageReactionDeleted
	err := e.GetEvent(EventTypeMessageReactionDeleted, &body)
	return &body, err
}
