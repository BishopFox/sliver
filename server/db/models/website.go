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
	"os"
	"path/filepath"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Website - Colletions of content to serve from HTTP(S)
type Website struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	Name         string `gorm:"unique;"` // Website Name
	AllowsUpload bool

	WebContents []WebContent
}

// BeforeCreate - GORM hook
func (w *Website) BeforeCreate(tx *gorm.DB) (err error) {
	w.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	w.CreatedAt = time.Now()
	return nil
}

// ToProtobuf - Converts to protobuf object
func (w *Website) ToProtobuf(webContentDir string) *clientpb.Website {
	WebContents := map[string]*clientpb.WebContent{}
	for _, webcontent := range w.WebContents {
		contents, err := os.ReadFile(filepath.Join(webContentDir, webcontent.ID.String()))
		if err != nil {
			continue
		}
		WebContents[webcontent.Path] = webcontent.ToProtobuf(&contents)
	}
	return &clientpb.Website{
		ID:           w.ID.String(),
		Name:         w.Name,
		AllowsUpload: w.AllowsUpload,
		Contents:     WebContents,
	}
}

// WebContent - One piece of content mapped to a path
type WebContent struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	WebsiteID uuid.UUID `gorm:"type:uuid;"`

	Path         string `gorm:"primaryKey"`
	Size         uint64
	ContentType  string
	OriginalFile string
	Sha256       string
}

// BeforeCreate - GORM hook to automatically set values
func (wc *WebContent) BeforeCreate(tx *gorm.DB) (err error) {
	wc.ID, err = uuid.NewV4()
	return err
}

// ToProtobuf - Converts to protobuf object
func (wc *WebContent) ToProtobuf(content *[]byte) *clientpb.WebContent {
	return &clientpb.WebContent{
		ID:           wc.ID.String(),
		WebsiteID:    wc.WebsiteID.String(),
		Path:         wc.Path,
		Size:         uint64(wc.Size),
		ContentType:  wc.ContentType,
		OriginalFile: wc.OriginalFile,
		Sha256:       wc.Sha256,
		Content:      *content,
	}
}

func WebContentFromProtobuf(pbWebContent *clientpb.WebContent) WebContent {
	siteUUID, _ := uuid.FromString(pbWebContent.ID)
	websiteUUID, _ := uuid.FromString(pbWebContent.WebsiteID)

	return WebContent{
		ID:           siteUUID,
		WebsiteID:    websiteUUID,
		Path:         pbWebContent.Path,
		Size:         pbWebContent.Size,
		ContentType:  pbWebContent.ContentType,
		OriginalFile: pbWebContent.OriginalFile,
		Sha256:       pbWebContent.Sha256,
	}
}

// WebUploadedContent - One piece of content received by PUT/POST requests
type WebUploadedContent struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	WebsiteID uuid.UUID `gorm:"type:uuid;"`

	Method        string
	Path          string
	UserAgent     string
	Headers       string
	URLParameters string
	RemoteAddress string
	Size          uint64
	ContentType   string
	ReceivedAt    time.Time
	Sha256        string
}

// BeforeCreate - GORM hook to automatically set values
func (wc *WebUploadedContent) BeforeCreate(tx *gorm.DB) (err error) {
	wc.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	wc.ReceivedAt = time.Now()
	return nil
}

// ToProtobuf - Converts to protobuf object
func (wc *WebUploadedContent) ToProtobuf(content *[]byte) *clientpb.WebUploadedContent {
	return &clientpb.WebUploadedContent{
		ID:            wc.ID.String(),
		WebsiteID:     wc.WebsiteID.String(),
		Method:        wc.Method,
		Path:          wc.Path,
		UserAgent:     wc.UserAgent,
		Headers:       wc.Headers,
		URLParameters: wc.URLParameters,
		RemoteAddress: wc.RemoteAddress,
		Size:          uint64(wc.Size),
		ContentType:   wc.ContentType,
		ReceivedAt:    wc.ReceivedAt.Unix(),
		Sha256:        wc.Sha256,
		Content:       *content,
	}
}

func UploadedWebContentFromProtobuf(pbWebContent *clientpb.WebUploadedContent) WebUploadedContent {
	siteUUID, _ := uuid.FromString(pbWebContent.ID)
	websiteUUID, _ := uuid.FromString(pbWebContent.WebsiteID)

	return WebUploadedContent{
		ID:            siteUUID,
		WebsiteID:     websiteUUID,
		Method:        pbWebContent.Method,
		Path:          pbWebContent.Path,
		UserAgent:     pbWebContent.UserAgent,
		Headers:       pbWebContent.Headers,
		URLParameters: pbWebContent.URLParameters,
		RemoteAddress: pbWebContent.RemoteAddress,
		Size:          pbWebContent.Size,
		ContentType:   pbWebContent.ContentType,
		ReceivedAt:    time.Unix(pbWebContent.ReceivedAt, 0),
		Sha256:        pbWebContent.Sha256,
	}
}
