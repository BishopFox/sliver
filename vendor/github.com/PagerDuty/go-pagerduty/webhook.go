package pagerduty

import (
	"encoding/json"
	"io"
	"time"
)

// IncidentDetails contains a representation of the incident associated with the action that caused this webhook message
type IncidentDetails struct {
	APIObject
	IncidentNumber       int               `json:"incident_number"`
	Title                string            `json:"title"`
	CreatedAt            time.Time         `json:"created_at"`
	Status               string            `json:"status"`
	IncidentKey          *string           `json:"incident_key"`
	PendingActions       []PendingAction   `json:"pending_actions"`
	Service              Service           `json:"service"`
	Assignments          []Assignment      `json:"assignments"`
	Acknowledgements     []Acknowledgement `json:"acknowledgements"`
	LastStatusChangeAt   time.Time         `json:"last_status_change_at"`
	LastStatusChangeBy   APIObject         `json:"last_status_change_by"`
	FirstTriggerLogEntry APIObject         `json:"first_trigger_log_entry"`
	EscalationPolicy     APIObject         `json:"escalation_policy"`
	Teams                []APIObject       `json:"teams"`
	Priority             Priority          `json:"priority"`
	Urgency              string            `json:"urgency"`
	ResolveReason        *string           `json:"resolve_reason"`
	AlertCounts          AlertCounts       `json:"alert_counts"`
	Metadata             interface{}       `json:"metadata"`

	// Alerts is the list of alerts within this incident. Each item in the slice
	// is not fully hydrated, so only the AlertKey field will be set.
	Alerts []IncidentAlert `json:"alerts,omitempty"`

	// Description is deprecated, use Title instead.
	Description string `json:"description"`
}

// WebhookPayloadMessages is the wrapper around the Webhook payloads. The Array may contain multiple message elements if webhook firing actions occurred in quick succession
type WebhookPayloadMessages struct {
	Messages []WebhookPayload `json:"messages"`
}

// WebhookPayload represents the V2 webhook payload
type WebhookPayload struct {
	ID         string          `json:"id"`
	Event      string          `json:"event"`
	CreatedOn  time.Time       `json:"created_on"`
	Incident   IncidentDetails `json:"incident"`
	LogEntries []LogEntry      `json:"log_entries"`
}

// DecodeWebhook decodes a webhook from a response object.
func DecodeWebhook(r io.Reader) (*WebhookPayloadMessages, error) {
	var payload WebhookPayloadMessages
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, err
	}
	return &payload, nil
}
