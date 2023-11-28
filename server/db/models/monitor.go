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
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type MonitoringProvider struct {
	ID          uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	Type        string    // currently vt or xforce
	APIKey      string
	APIPassword string
}

// creation hooks

func (m *MonitoringProvider) BeforeCreate(tx *gorm.DB) (err error) {
	m.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	return nil
}

// convert to protobuf
func (m *MonitoringProvider) ToProtobuf() *clientpb.MonitoringProvider {
	return &clientpb.MonitoringProvider{
		ID:          m.ID.String(),
		Type:        m.Type,
		APIKey:      m.APIKey,
		APIPassword: m.APIPassword,
	}
}

// convert from protobuf
func MonitorFromProtobuf(m *clientpb.MonitoringProvider) MonitoringProvider {
	uuid, _ := uuid.FromString(m.ID)
	return MonitoringProvider{
		ID:          uuid,
		Type:        m.Type,
		APIKey:      m.APIKey,
		APIPassword: m.APIPassword,
	}
}
