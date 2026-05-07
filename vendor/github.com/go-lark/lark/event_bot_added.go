package lark

// EventV2BotAdded .
type EventV2BotAdded struct {
	ChatID            string        `json:"chat_id,omitempty"`
	OperatorID        EventV2UserID `json:"operator_id,omitempty"`
	External          bool          `json:"external,omitempty"`
	OperatorTenantKey string        `json:"operator_tenant_key,omitempty"`
}

// GetBotAdded .
func (e EventV2) GetBotAdded() (*EventV2BotAdded, error) {
	var body EventV2BotAdded
	err := e.GetEvent(EventTypeBotAdded, &body)
	return &body, err
}
