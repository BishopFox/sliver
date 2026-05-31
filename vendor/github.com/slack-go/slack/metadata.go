package slack

// SlackMetadata https://api.slack.com/reference/metadata
type SlackMetadata struct {
	EventType    string         `json:"event_type"`
	EventPayload map[string]any `json:"event_payload"`
}

// Work Object entity type constants.
// See https://docs.slack.dev/messaging/work-objects/
const (
	EntityTypeTask        = "slack#/entities/task"
	EntityTypeFile        = "slack#/entities/file"
	EntityTypeItem        = "slack#/entities/item"
	EntityTypeIncident    = "slack#/entities/incident"
	EntityTypeContentItem = "slack#/entities/content_item"
)

// WorkObjectExternalRef represents an external reference for a Work Object
type WorkObjectExternalRef struct {
	ID   string `json:"id"`
	Type string `json:"type,omitempty"`
}

// WorkObjectEntity represents a single Work Object entity
type WorkObjectEntity struct {
	AppUnfurlURL  string                 `json:"app_unfurl_url,omitempty"`
	URL           string                 `json:"url"`
	ExternalRef   WorkObjectExternalRef  `json:"external_ref"`
	EntityType    string                 `json:"entity_type"`
	EntityPayload map[string]interface{} `json:"entity_payload"`
}

// WorkObjectMetadata represents the metadata for Work Objects
// Used in chat.unfurl and chat.postMessage for Work Objects support
type WorkObjectMetadata struct {
	Entities []WorkObjectEntity `json:"entities"`
}
