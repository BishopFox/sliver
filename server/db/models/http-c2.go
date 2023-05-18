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

// HttpC2Config -
type HttpC2Config struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	Name string `gorm:"unique;"`

	ServerConfig  HttpC2ServerConfig
	ImplantConfig HttpC2ImplantConfig
}

func (h *HttpC2Config) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	h.CreatedAt = time.Now()
	return nil
}

func (h *HttpC2Config) ToProtobuf() *clientpb.HTTPC2Config {
	return &clientpb.HTTPC2Config{
		ID:      h.ID.String(),
		Created: h.CreatedAt.Unix(),
		Name:    h.Name,

		ServerConfig:  h.ServerConfig.ToProtobuf(),
		ImplantConfig: h.ImplantConfig.ToProtobuf(),
	}
}

// HttpC2ServerConfig - HTTP C2 Server Configuration
type HttpC2ServerConfig struct {
	ID             uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	HttpC2ConfigID uuid.UUID `gorm:"type:uuid;"`

	RandomVersionHeaders bool
	Headers              []HttpC2Header
	Cookies              []HttpC2Cookie
}

func (h *HttpC2ServerConfig) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID, err = uuid.NewV4()
	return err
}

func (h *HttpC2ServerConfig) ToProtobuf() *clientpb.HTTPC2ServerConfig {
	headers := make([]*clientpb.HTTPC2Header, len(h.Headers))
	for i, header := range h.Headers {
		headers[i] = header.ToProtobuf()
	}
	cookies := make([]*clientpb.HTTPC2Cookie, len(h.Cookies))
	for i, cookie := range h.Cookies {
		cookies[i] = cookie.ToProtobuf()
	}
	return &clientpb.HTTPC2ServerConfig{
		ID:                   h.ID.String(),
		RandomVersionHeaders: h.RandomVersionHeaders,
		Headers:              headers,
		Cookies:              cookies,
	}
}

// HttpC2ImplantConfig - HTTP C2 Implant Configuration
type HttpC2ImplantConfig struct {
	ID             uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	HttpC2ConfigID uuid.UUID `gorm:"type:uuid;"`

	UserAgent          string
	ChromeBaseVersion  int32
	MacOSVersion       string
	NonceQueryArgChars string
	ExtraURLParameters []HttpC2URLParameter
	Headers            []HttpC2Header

	MaxFiles int32
	MinFiles int32
	MaxPaths int32
	MinPaths int32

	StagerFileExtension       string
	PollFileExtension         string
	StartSessionFileExtension string
	SessionFileExtension      string
	CloseFileExtension        string

	PathSegments []HttpC2PathSegment
}

func (h *HttpC2ImplantConfig) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID, err = uuid.NewV4()
	return err
}

func (h *HttpC2ImplantConfig) ToProtobuf() *clientpb.HTTPC2ImplantConfig {
	params := make([]*clientpb.HTTPC2URLParameter, len(h.ExtraURLParameters))
	for i, param := range h.ExtraURLParameters {
		params[i] = param.ToProtobuf()
	}
	headers := make([]*clientpb.HTTPC2Header, len(h.Headers))
	for i, header := range h.Headers {
		headers[i] = header.ToProtobuf()
	}
	pathSegments := make([]*clientpb.HTTPC2PathSegment, len(h.PathSegments))
	for i, segment := range h.PathSegments {
		pathSegments[i] = segment.ToProtobuf()
	}
	return &clientpb.HTTPC2ImplantConfig{
		ID:                        h.ID.String(),
		UserAgent:                 h.UserAgent,
		ChromeBaseVersion:         h.ChromeBaseVersion,
		MacOSVersion:              h.MacOSVersion,
		NonceQueryArgChars:        h.NonceQueryArgChars,
		ExtraURLParameters:        params,
		Headers:                   headers,
		MaxFiles:                  h.MaxFiles,
		MinFiles:                  h.MinFiles,
		MaxPaths:                  h.MaxPaths,
		MinPaths:                  h.MinPaths,
		StagerFileExtension:       h.StagerFileExtension,
		PollFileExtension:         h.PollFileExtension,
		StartSessionFileExtension: h.StartSessionFileExtension,
		SessionFileExtension:      h.SessionFileExtension,
		CloseFileExtension:        h.CloseFileExtension,
		PathSegments:              pathSegments,
	}
}

//
// >>> Sub-Models <<<
//

// HttpC2Cookie - HTTP C2 Cookie (server only)
type HttpC2Cookie struct {
	ID                   uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	HttpC2ServerConfigID uuid.UUID `gorm:"type:uuid;"`

	Name string
}

func (h *HttpC2Cookie) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID, err = uuid.NewV4()
	return err
}

func (h *HttpC2Cookie) ToProtobuf() *clientpb.HTTPC2Cookie {
	return &clientpb.HTTPC2Cookie{
		ID:   h.ID.String(),
		Name: h.Name,
	}
}

// HttpC2Header - HTTP C2 Header (server and implant)
type HttpC2Header struct {
	ID                    uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	HttpC2ServerConfigID  uuid.UUID `gorm:"type:uuid;"`
	HttpC2ImplantConfigID uuid.UUID `gorm:"type:uuid;"`

	Method      string
	Name        string
	Value       string
	Probability int32
}

func (h *HttpC2Header) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID, err = uuid.NewV4()
	return err
}

func (h *HttpC2Header) ToProtobuf() *clientpb.HTTPC2Header {
	return &clientpb.HTTPC2Header{
		ID:          h.ID.String(),
		Method:      h.Method,
		Name:        h.Name,
		Value:       h.Value,
		Probability: h.Probability,
	}
}

// HttpC2URLParameter - Extra URL parameters (implant only)
type HttpC2URLParameter struct {
	ID                    uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	HttpC2ImplantConfigID uuid.UUID `gorm:"type:uuid;"`

	Method      string // HTTP Method
	Name        string // Name of URL parameter, must be 3+ characters
	Value       string // Value of the URL parameter
	Probability int32  // 0 - 100
}

func (h *HttpC2URLParameter) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID, err = uuid.NewV4()
	return err
}

func (h *HttpC2URLParameter) ToProtobuf() *clientpb.HTTPC2URLParameter {
	return &clientpb.HTTPC2URLParameter{
		ID:          h.ID.String(),
		Method:      h.Method,
		Name:        h.Name,
		Value:       h.Value,
		Probability: h.Probability,
	}
}

// HttpC2PathSegment - Represents a list of file/path URL segments (implant only)
type HttpC2PathSegment struct {
	ID                    uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	HttpC2ImplantConfigID uuid.UUID `gorm:"type:uuid;"`

	IsFile      bool
	SegmentType int32 // Poll, Session, Close
	Value       string
}

func (h *HttpC2PathSegment) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID, err = uuid.NewV4()
	return err
}

func (h *HttpC2PathSegment) ToProtobuf() *clientpb.HTTPC2PathSegment {
	return &clientpb.HTTPC2PathSegment{
		ID:          h.ID.String(),
		IsFile:      h.IsFile,
		SegmentType: clientpb.HTTPC2SegmentType(h.SegmentType),
		Value:       h.Value,
	}
}
