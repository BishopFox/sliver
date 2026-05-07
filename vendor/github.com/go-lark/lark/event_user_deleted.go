package lark

// EventV2UserDeleted .
type EventV2UserDeleted = EventV2UserAdded

// GetUserDeleted .
func (e EventV2) GetUserDeleted() (*EventV2UserDeleted, error) {
	var body EventV2UserDeleted
	err := e.GetEvent(EventTypeUserDeleted, &body)
	return &body, err
}
