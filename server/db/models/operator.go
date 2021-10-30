package models

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Operator - Colletions of content to serve from HTTP(S)
type Operator struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`
	Name      string
	Token     string `gorm:"uniqueIndex"`
}

// BeforeCreate - GORM hook
func (o *Operator) BeforeCreate(tx *gorm.DB) (err error) {
	o.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	o.CreatedAt = time.Now()
	return nil
}

// GenerateOperatorToken - Generate a new operator auth token
func GenerateOperatorToken() string {
	buf := make([]byte, 32)
	n, err := rand.Read(buf)
	if err != nil || n != len(buf) {
		panic(errors.New("failed to read from secure rand"))
	}
	return hex.EncodeToString(buf)
}
