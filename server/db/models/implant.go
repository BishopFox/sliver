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

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// ImplantBuild - Represents an implant
type ImplantBuild struct {
	gorm.Model

	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;index:unique;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	Name string

	ImplantConfig *ImplantConfig
}

// BeforeCreate - GORM hook
func (ib *ImplantBuild) BeforeCreate(tx *gorm.DB) (err error) {
	ib.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	ib.CreatedAt = time.Now()
	return nil
}

// ImplantConfig - An implant build configuration
type ImplantConfig struct {
	gorm.Model

	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;index:unique;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	// Go
	GOOS   string
	GOARCH string

	// Standard
	// Name                string
	CACert              string
	Cert                string
	Key                 string
	Debug               bool
	Evasion             bool
	ObfuscateSymbols    bool
	ReconnectInterval   uint32
	MaxConnectionErrors uint32

	C2 []*ImplantC2

	MTLSc2Enabled bool
	HTTPc2Enabled bool
	DNSc2Enabled  bool

	CanaryDomains     []string
	NamePipec2Enabled bool
	TCPPivotc2Enabled bool

	// Limits
	LimitDomainJoined bool
	LimitHostname     string
	LimitUsername     string
	LimitDatetime     string
	LimitFileExists   string

	// Output Format
	Format clientpb.ImplantConfig_OutputFormat

	// For 	IsSharedLib bool
	IsSharedLib bool
	IsService   bool
	IsShellcode bool

	FileName string
}

// BeforeCreate - GORM hook
func (ic *ImplantConfig) BeforeCreate(tx *gorm.DB) (err error) {
	ic.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	ic.CreatedAt = time.Now()
	return nil
}

// ToProtobuf - Convert ImplantConfig to protobuf equiv
func (ic *ImplantConfig) ToProtobuf() *clientpb.ImplantConfig {
	config := &clientpb.ImplantConfig{
		GOOS:   ic.GOOS,
		GOARCH: ic.GOARCH,
		// Name:             ic.Name,
		CACert:           ic.CACert,
		Cert:             ic.Cert,
		Key:              ic.Key,
		Debug:            ic.Debug,
		Evasion:          ic.Evasion,
		ObfuscateSymbols: ic.ObfuscateSymbols,
		CanaryDomains:    ic.CanaryDomains,

		ReconnectInterval:   ic.ReconnectInterval,
		MaxConnectionErrors: ic.MaxConnectionErrors,

		LimitDatetime:     ic.LimitDatetime,
		LimitDomainJoined: ic.LimitDomainJoined,
		LimitHostname:     ic.LimitHostname,
		LimitUsername:     ic.LimitUsername,
		LimitFileExists:   ic.LimitFileExists,

		IsSharedLib: ic.IsSharedLib,
		IsService:   ic.IsService,
		IsShellcode: ic.IsShellcode,
		Format:      ic.Format,

		FileName: ic.FileName,
	}
	config.C2 = []*clientpb.ImplantC2{}
	for _, c2 := range ic.C2 {
		config.C2 = append(config.C2, c2.ToProtobuf())
	}
	return config
}

// ImplantC2 - C2 struct
type ImplantC2 struct {
	gorm.Model

	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;index:unique;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	Priority uint32 `json:"priority"`
	URL      string `json:"url"`
	Options  string `json:"options"`
}

// ToProtobuf - Convert to protobuf version
func (c2 *ImplantC2) ToProtobuf() *clientpb.ImplantC2 {
	return &clientpb.ImplantC2{
		Priority: c2.Priority,
		URL:      c2.URL,
		Options:  c2.Options,
	}
}

func (c2 *ImplantC2) String() string {
	return c2.URL
}

// ImplantProfile - An implant build configuration
type ImplantProfile struct {
	gorm.Model

	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;index:unique;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	Name          string
	ImplantConfig *ImplantConfig
}

// BeforeCreate - GORM hook
func (ip *ImplantProfile) BeforeCreate(tx *gorm.DB) (err error) {
	ip.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	ip.CreatedAt = time.Now()
	return nil
}
