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

// BeforeCreate - GORM hook to automatically set values
func (c *WGKeys) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	c.CreatedAt = time.Now()
	return nil
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
	c.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	c.CreatedAt = time.Now()
	return nil
}
