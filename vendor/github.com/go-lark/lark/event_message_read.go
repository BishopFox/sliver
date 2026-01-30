package lark

// EventV2MessageRead .
type EventV2MessageRead struct {
	Reader struct {
		ReaderID  EventV2UserID `json:"reader_id,omitempty"`
		ReadTime  string        `json:"read_time,omitempty"`
		TenantKey string        `json:"tenant_key,omitempty"`
	} `json:"reader,omitempty"`
	MessageIDList []string `json:"message_id_list,omitempty"`
}

// GetMessageRead .
func (e EventV2) GetMessageRead() (*EventV2MessageRead, error) {
	var body EventV2MessageRead
	err := e.GetEvent(EventTypeMessageRead, &body)
	return &body, err
}
