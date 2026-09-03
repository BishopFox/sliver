package models

import (
	"time"

	"gorm.io/gorm"
)

// WGKeys - WGKeys database model
type WGKeys struct {
	ID        UUID      `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`
	PrivKey   string
	PubKey    string
}

// MultiplayerWGKeys - Multiplayer WireGuard server keys database model.
type MultiplayerWGKeys struct {
	ID        UUID      `gorm:"primaryKey;->;<-:create;type:uuid;"`
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
	ID        UUID      `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`
	PrivKey   string
	PubKey    string
	TunIP     string
}

// BeforeCreate - GORM hook to automatically set values
func (c *WGPeer) BeforeCreate(tx *gorm.DB) (err error) {
	return initWGKeysModel(&c.ID, &c.CreatedAt)
}

func initWGKeysModel(id *UUID, createdAt *time.Time) error {
	*id = NewUUID()
	*createdAt = time.Now()
	return nil
}
