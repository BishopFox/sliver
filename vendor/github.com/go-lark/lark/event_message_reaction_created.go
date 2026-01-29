package lark

// EventV2MessageReactionCreated .
type EventV2MessageReactionCreated struct {
	MessageID    string        `json:"message_id,omitempty"`
	OperatorType string        `json:"operator_type,omitempty"`
	UserID       EventV2UserID `json:"user_id,omitempty"`
	AppID        string        `json:"app_id,omitempty"`
	ActionTime   string        `json:"action_time,omitempty"`
	ReactionType struct {
		EmojiType string `json:"emoji_type,omitempty"`
	} `json:"reaction_type,omitempty"`
}

// GetMessageReactionCreated .
func (e EventV2) GetMessageReactionCreated() (*EventV2MessageReactionCreated, error) {
	var body EventV2MessageReactionCreated
	err := e.GetEvent(EventTypeMessageReactionCreated, &body)
	return &body, err
}
