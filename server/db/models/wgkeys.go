package models

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// WGKeys - WGKeys database model
type WGKeys struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`
	PrivKey   string
	PubKey    string
}

// MultiplayerWGKeys - Multiplayer WireGuard server keys database model.
type MultiplayerWGKeys struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`
	PrivKey   string
	PubKey    string
}

// BeforeCreate - GORM hook to automatically set values
func (c *WGKeys) BeforeCreate(tx *gorm.DB) (err error) {
	return initWGKeysModel(&c.ID, &c.CreatedAt)
}

// BeforeCreate - GORM hook to automatically set values
func (c *MultiplayerWGKeys) BeforeCreate(tx *gorm.DB) (err error) {
	return initWGKeysModel(&c.ID, &c.CreatedAt)
}

// WGPeer- WGPeer database model
type WGPeer struct {
	// gorm.Model
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`
	PrivKey   string
	PubKey    string
	TunIP     string
}

// BeforeCreate - GORM hook to automatically set values
func (c *WGPeer) BeforeCreate(tx *gorm.DB) (err error) {
	return initWGKeysModel(&c.ID, &c.CreatedAt)
}

func initWGKeysModel(id *uuid.UUID, createdAt *time.Time) error {
	newID, err := uuid.NewV4()
	if err != nil {
		return err
	}
	*id = newID
	*createdAt = time.Now()
	return nil
}
