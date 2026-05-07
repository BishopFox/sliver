package lark

// EventV2BotDeleted .
type EventV2BotDeleted = EventV2BotAdded

// GetBotDeleted .
func (e EventV2) GetBotDeleted() (*EventV2BotDeleted, error) {
	var body EventV2BotDeleted
	err := e.GetEvent(EventTypeBotDeleted, &body)
	return &body, err
}
