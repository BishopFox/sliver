package models

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

const (
	WGIPOwnerTypeOperator = "operator"
	WGIPOwnerTypePeer     = "peer"
)

// WGIPReservation tracks allocated WireGuard tunnel IPs across all consumers.
type WGIPReservation struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	TunIP     string `gorm:"uniqueIndex"`
	OwnerType string
	OwnerID   string
}

// BeforeCreate - GORM hook to automatically set values.
func (r *WGIPReservation) BeforeCreate(tx *gorm.DB) (err error) {
	r.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	r.CreatedAt = time.Now()
	return nil
}
