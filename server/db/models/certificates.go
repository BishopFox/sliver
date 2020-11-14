package models

import (
	"time"

	"github.com/gofrs/uuid"
)

// Certificate - Certificate database model
type Certificate struct {
	ID             uuid.UUID `gorm:"->;type:uuid;"`
	CreatedAt      time.Time `gorm:"->;"`
	CommonName     string
	CAType         string
	KeyType        string
	CertificatePEM string
	PrivateKeyPEM  string
}
