package models

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Credential - Represents a piece of loot
type Credential struct {
	ID                 uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt          time.Time `gorm:"->;<-:create;"`
	HashedCredentialID uuid.UUID `gorm:"type:uuid;"`

	Username  string
	Plaintext string

	OriginHost uuid.UUID `gorm:"type:uuid;"`
}

func (c *Credential) ToProtobuf() *clientpb.Credential {
	return &clientpb.Credential{
		ID:           c.ID.String(),
		Username:     c.Username,
		Plaintext:    c.Plaintext,
		OriginHostID: c.OriginHost.String(),
	}
}

type HashedCredential struct {
	ID         uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt  time.Time `gorm:"->;<-:create;"`
	Credential Credential

	Hash      string // https://hashcat.net/wiki/doku.php?id=example_hashes
	HashType  int32
	IsCracked bool

	OriginHost uuid.UUID `gorm:"type:uuid;"`
}

func (c *HashedCredential) ToProtobuf() *clientpb.HashedCredential {
	return &clientpb.HashedCredential{
		ID:           c.ID.String(),
		Credential:   c.Credential.ToProtobuf(),
		Hash:         c.Hash,
		HashType:     clientpb.HashType(c.HashType),
		IsCracked:    c.IsCracked,
		OriginHostID: c.OriginHost.String(),
	}
}

// BeforeCreate - GORM hook
func (c *Credential) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	c.CreatedAt = time.Now()
	return nil
}
