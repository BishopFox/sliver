package models

/*
	Sliver Implant Framework
	Copyright (C) 2020  Bishop Fox

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

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// KeyValue - Represents an implant
type KeyValue struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	Key   string `gorm:"unique;"`
	Value string
}

// BeforeCreate - GORM hook
func (k *KeyValue) BeforeCreate(tx *gorm.DB) (err error) {
	k.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	k.CreatedAt = time.Now()
	return nil
}
