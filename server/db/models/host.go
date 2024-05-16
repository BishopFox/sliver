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
type Host struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	HostUUID  uuid.UUID `gorm:"type:uuid;unique"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	Hostname  string
	OSVersion string // Verbose OS version
	Locale    string // Detected language code

	IOCs          []IOC           `gorm:"foreignKey:HostID;references:HostUUID"`
	ExtensionData []ExtensionData `gorm:"foreignKey:HostID;references:HostUUID"`
}

// BeforeCreate - GORM hook
func (h *Host) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	h.CreatedAt = time.Now()
	return nil
}

func (h *Host) ToProtobuf() *clientpb.Host {
	return &clientpb.Host{
		HostUUID:      h.HostUUID.String(),
		Hostname:      h.Hostname,
		OSVersion:     h.OSVersion,
		Locale:        h.Locale,
		IOCs:          h.iocsToProtobuf(),
		ExtensionData: h.extensionDataToProtobuf(),
		FirstContact:  h.CreatedAt.Unix(),
	}
}

func (h *Host) iocsToProtobuf() []*clientpb.IOC {
	iocs := []*clientpb.IOC{}
	for _, dbIOC := range h.IOCs {
		iocs = append(iocs, dbIOC.ToProtobuf())
	}
	return iocs
}

func (h *Host) extensionDataToProtobuf() map[string]*clientpb.ExtensionData {
	extData := map[string]*clientpb.ExtensionData{}
	for _, dbExtData := range h.ExtensionData {
		extData[dbExtData.Name] = &clientpb.ExtensionData{
			Output: dbExtData.Output,
		}
	}
	return extData
}

// IOC - Represents an indicator of compromise, generally a file we've
// uploaded to a remote system.
type IOC struct {
	gorm.Model

	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	HostID    uuid.UUID `gorm:"type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	Path     string
	FileHash string
}

// BeforeCreate - GORM hook
func (i *IOC) BeforeCreate(tx *gorm.DB) (err error) {
	i.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	i.CreatedAt = time.Now()
	return nil
}

func (i *IOC) ToProtobuf() *clientpb.IOC {
	return &clientpb.IOC{
		Path:     i.Path,
		FileHash: i.FileHash,
		ID:       i.ID.String(),
	}
}

// ExtensionData - Represents an indicator of compromise, generally a file we've
// uploaded to a remote system.
type ExtensionData struct {
	gorm.Model

	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	HostID    uuid.UUID `gorm:"type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	Name   string
	Output string
}

// BeforeCreate - GORM hook
func (e *ExtensionData) BeforeCreate(tx *gorm.DB) (err error) {
	e.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	e.CreatedAt = time.Now()
	return nil
}
