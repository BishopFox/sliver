package loot

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
	"errors"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/proto"
)

var (
	// ErrInvalidLootID - Invalid Loot ID
	ErrInvalidLootID = errors.New("invalid loot id")
	// ErrLootNotFound - Loot not found
	ErrLootNotFound = errors.New("loot not found")

	lootLog = log.NamedLogger("loot", "backend")
)

// LocalBackend - A loot backend that saves files locally to disk
type LocalBackend struct {
	LocalFileDir string
}

// Add - Add a piece of loot
func (l *LocalBackend) Add(loot *clientpb.Loot) (*clientpb.Loot, error) {
	host, err := db.HostByHostUUID(loot.OriginHostUUID)
	var hostID uuid.UUID
	if err != nil {
		lootLog.Warnf("Failed to find host %s for loot %s", loot.OriginHostUUID, loot.ID)
		hostID = uuid.Nil
	} else {
		hostID = host.HostUUID
	}
	dbLoot := &models.Loot{
		Name:         loot.GetName(),
		FileType:     int(loot.GetFileType()),
		Size:         int64(len(loot.File.Data)),
		OriginHostID: hostID,
	}
	dbSession := db.Session()
	err = dbSession.Create(dbLoot).Error
	if err != nil {
		return nil, err
	}

	if loot.File != nil {
		lootLocalFile := filepath.Join(l.LocalFileDir, dbLoot.ID.String())
		data, err := proto.Marshal(loot.File)
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(lootLocalFile, data, 0600)
		if err != nil {
			return nil, err
		}
	}

	// Fetch a fresh version of the object
	loot, err = l.GetContent(dbLoot.ID.String(), false)
	if loot.File != nil {
		loot.File.Data = nil
	}
	return loot, err
}

// Update - Update metadata about loot, currently only 'name' can be changed
func (l *LocalBackend) Update(lootReq *clientpb.Loot) (*clientpb.Loot, error) {
	dbSession := db.Session()
	lootUUID, err := uuid.FromString(lootReq.ID)
	if err != nil {
		return nil, ErrInvalidLootID
	}
	dbLoot := &models.Loot{}
	result := dbSession.Where(&models.Loot{ID: lootUUID}).First(dbLoot)
	if errors.Is(result.Error, db.ErrRecordNotFound) {
		return nil, ErrLootNotFound
	}

	if dbLoot.Name != lootReq.Name {
		err = dbSession.Model(&dbLoot).Update("Name", lootReq.Name).Error
		if err != nil {
			return nil, err
		}
	}

	return l.GetContent(lootReq.ID, false)
}

// Rm - Remove a piece of loot
func (l *LocalBackend) Rm(lootID string) error {
	dbSession := db.Session()
	lootUUID, err := uuid.FromString(lootID)
	if err != nil {
		return ErrInvalidLootID
	}
	dbLoot := &models.Loot{}
	result := dbSession.Where(&models.Loot{ID: lootUUID}).First(dbLoot)
	if errors.Is(result.Error, db.ErrRecordNotFound) {
		return ErrLootNotFound
	}

	// File Loot
	lootLocalFile := filepath.Join(l.LocalFileDir, dbLoot.ID.String())
	if _, err := os.Stat(lootLocalFile); !os.IsNotExist(err) {
		err = os.Remove(lootLocalFile)
		if err != nil {
			lootLog.Error(err)
		}
	}

	result = dbSession.Delete(&dbLoot)
	return result.Error
}

// GetContent - Get the content of a piece of loot
func (l *LocalBackend) GetContent(lootID string, eager bool) (*clientpb.Loot, error) {
	dbSession := db.Session()
	lootUUID, err := uuid.FromString(lootID)
	if err != nil {
		return nil, ErrInvalidLootID
	}
	dbLoot := &models.Loot{}
	result := dbSession.Where(&models.Loot{ID: lootUUID}).First(dbLoot)
	if errors.Is(result.Error, db.ErrRecordNotFound) {
		return nil, ErrLootNotFound
	}

	// Re-construct protobuf object
	loot := dbLoot.ToProtobuf()

	// File Loot
	lootLocalFile := filepath.Join(l.LocalFileDir, dbLoot.ID.String())
	if _, err := os.Stat(lootLocalFile); !os.IsNotExist(err) {
		data, err := os.ReadFile(lootLocalFile)
		if err != nil {
			return nil, err
		}
		loot.File = &commonpb.File{}
		err = proto.Unmarshal(data, loot.File)
		if err != nil {
			return nil, err
		}
	}

	return loot, nil
}

// All - Get all loot
func (l *LocalBackend) All() *clientpb.AllLoot {
	dbSession := db.Session()
	allDBLoot := []*models.Loot{}
	result := dbSession.Where(&models.Loot{}).Find(&allDBLoot)
	if result.Error != nil {
		lootLog.Error(result.Error)
		return nil
	}
	all := &clientpb.AllLoot{Loot: []*clientpb.Loot{}}
	for _, dbLoot := range allDBLoot {
		loot, err := l.GetContent(dbLoot.ID.String(), false)
		if err != nil {
			lootLog.Error(err)
			continue
		}
		if loot.File != nil {
			loot.File.Data = nil
		}
		all.Loot = append(all.Loot, loot)
	}
	return all
}
