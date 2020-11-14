package models

import (
	"time"

	"github.com/gofrs/uuid"
)

// CertificateModel - Certificate database model
type CertificateModel struct {
	ID             uuid.UUID `gorm:"->;type:uuid;"`
	CreatedAt      time.Time `gorm:"->;"`
	CommonName     string
	CAType         string
	KeyType        string
	CertificatePEM string
	PrivateKeyPEM  string
}
