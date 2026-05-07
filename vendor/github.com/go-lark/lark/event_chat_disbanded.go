package lark

// EventV2ChatDisbanded .
type EventV2ChatDisbanded struct {
	ChatID            string        `json:"chat_id,omitempty"`
	OperatorID        EventV2UserID `json:"operator_id,omitempty"`
	External          bool          `json:"external,omitempty"`
	OperatorTenantKey string        `json:"operator_tenant_key,omitempty"`
}

// GetChatDisbanded .
func (e EventV2) GetChatDisbanded() (*EventV2ChatDisbanded, error) {
	var body EventV2ChatDisbanded
	err := e.GetEvent(EventTypeChatDisbanded, &body)
	return &body, err
}
