package slack

// SlackMetadata https://api.slack.com/reference/metadata
type SlackMetadata struct {
	EventType    string                 `json:"event_type"`
	EventPayload map[string]interface{} `json:"event_payload"`
}
