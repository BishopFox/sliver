package lark

// EventV2MessageRecalled .
type EventV2MessageRecalled struct {
	MessageID  string `json:"message_id,omitempty"`
	ChatID     string `json:"chat_id,omitempty"`
	RecallTime string `json:"recall_time,omitempty"`
	RecallType string `json:"recall_type,omitempty"`
}

// GetMessageRecalled .
func (e EventV2) GetMessageRecalled() (*EventV2MessageRecalled, error) {
	var body EventV2MessageRecalled
	err := e.GetEvent(EventTypeMessageRecalled, &body)
	return &body, err
}
