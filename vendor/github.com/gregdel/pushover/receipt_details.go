package pushover

import (
	"bytes"
	"encoding/json"
	"time"
)

// ReceiptDetails represents the receipt informations in case of emergency
// priority.
type ReceiptDetails struct {
	Status          int
	Acknowledged    bool
	AcknowledgedBy  string
	Expired         bool
	CalledBack      bool
	ID              string
	AcknowledgedAt  *time.Time
	LastDeliveredAt *time.Time
	ExpiresAt       *time.Time
	CalledBackAt    *time.Time
}

// UnmarshalJSON is a custom unmarshal function to handle timestamps and
// boolean as int and convert them to the right type.
func (r *ReceiptDetails) UnmarshalJSON(data []byte) error {
	dataBytes := bytes.NewReader(data)
	var aux struct {
		ID              string     `json:"request"`
		Status          int        `json:"status"`
		Acknowledged    intBool    `json:"acknowledged"`
		AcknowledgedBy  string     `json:"acknowledged_by"`
		Expired         intBool    `json:"expired"`
		CalledBack      intBool    `json:"called_back"`
		AcknowledgedAt  *timestamp `json:"acknowledged_at"`
		LastDeliveredAt *timestamp `json:"last_delivered_at"`
		ExpiresAt       *timestamp `json:"expires_at"`
		CalledBackAt    *timestamp `json:"called_back_at"`
	}

	// Decode json into the aux struct
	if err := json.NewDecoder(dataBytes).Decode(&aux); err != nil {
		return err
	}

	// Set the RecipientDetails with the right types
	r.Status = aux.Status
	r.Acknowledged = bool(aux.Acknowledged)
	r.AcknowledgedBy = aux.AcknowledgedBy
	r.Expired = bool(aux.Expired)
	r.CalledBack = bool(aux.CalledBack)
	r.ID = aux.ID
	r.AcknowledgedAt = aux.AcknowledgedAt.Time
	r.LastDeliveredAt = aux.LastDeliveredAt.Time
	r.ExpiresAt = aux.ExpiresAt.Time
	r.CalledBackAt = aux.CalledBackAt.Time

	return nil
}
