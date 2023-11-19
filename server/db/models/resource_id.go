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
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Host - Represents a host machine
type ResourceID struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	Type  string // encoder or stager
	Name  string
	Value uint64 // prime number used to reference resource in requests
}

// BeforeCreate - GORM hook
func (h *ResourceID) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	h.CreatedAt = time.Now()
	return nil
}

// ToProtobuf - Converts to protobuf object
func (rid *ResourceID) ToProtobuf() *clientpb.ResourceID {
	return &clientpb.ResourceID{
		ID:    rid.ID.String(),
		Type:  rid.Type,
		Name:  rid.Name,
		Value: rid.Value,
	}
}
