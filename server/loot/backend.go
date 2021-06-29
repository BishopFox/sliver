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
	"io/ioutil"
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
	lootLog = log.NamedLogger("loot", "backend")
)

type LocalBackend struct {
	LocalFileDir string
	LocalCredDir string
}

func (l *LocalBackend) Add(loot *clientpb.Loot) (*clientpb.Loot, error) {
	dbLoot := &models.Loot{
		Name: loot.GetName(),
		Type: int(loot.GetType()),
	}
	dbSession := db.Session()
	err := dbSession.Create(dbLoot).Error
	if err != nil {
		return nil, err
	}

	if loot.File != nil {
		lootLocalFile := filepath.Join(l.LocalFileDir, dbLoot.ID.String())
		data, err := proto.Marshal(loot.File)
		if err != nil {
			return nil, err
		}
		err = ioutil.WriteFile(lootLocalFile, data, 0600)
		if err != nil {
			return nil, err
		}
	}

	if loot.Credential != nil {
		lootLocalCred := filepath.Join(l.LocalCredDir, dbLoot.ID.String())
		data, err := proto.Marshal(loot.Credential)
		if err != nil {
			return nil, err
		}
		err = ioutil.WriteFile(lootLocalCred, data, 0600)
		if err != nil {
			return nil, err
		}
	}

	return l.GetContent(dbLoot.ID.String())
}

func (l *LocalBackend) Rm(lootID string) error {
	dbSession := db.Session()
	lootUUID, err := uuid.FromString(lootID)
	if err != nil {
		return errors.New("invalid loot id")
	}
	dbLoot := &models.Loot{}
	result := dbSession.Where(&models.Loot{ID: lootUUID}).First(dbLoot)
	if errors.Is(result.Error, db.ErrRecordNotFound) {
		return errors.New("loot not found")
	}

	// File Loot
	lootLocalFile := filepath.Join(l.LocalFileDir, dbLoot.ID.String())
	if _, err := os.Stat(lootLocalFile); !os.IsNotExist(err) {
		err = os.Remove(lootLocalFile)
		if err != nil {
			lootLog.Error(err)
		}
	}

	// Credential Loot
	lootCredFile := filepath.Join(l.LocalCredDir, dbLoot.ID.String())
	if _, err := os.Stat(lootCredFile); !os.IsNotExist(err) {
		err = os.Remove(lootCredFile)
		if err != nil {
			lootLog.Error(err)
		}
	}

	result = dbSession.Delete(&dbLoot)
	return result.Error
}

func (l *LocalBackend) GetContent(lootID string) (*clientpb.Loot, error) {
	dbSession := db.Session()
	lootUUID, err := uuid.FromString(lootID)
	if err != nil {
		return nil, errors.New("invalid loot id")
	}
	dbLoot := &models.Loot{}
	result := dbSession.Where(&models.Loot{ID: lootUUID}).First(dbLoot)
	if errors.Is(result.Error, db.ErrRecordNotFound) {
		return nil, errors.New("loot not found")
	}

	// Re-construct protobuf object
	loot := &clientpb.Loot{
		LootID: dbLoot.ID.String(),
		Name:   dbLoot.Name,
	}

	// File Loot
	lootLocalFile := filepath.Join(l.LocalFileDir, dbLoot.ID.String())
	if _, err := os.Stat(lootLocalFile); !os.IsNotExist(err) {
		data, err := ioutil.ReadFile(lootLocalFile)
		if err != nil {
			return nil, err
		}
		loot.File = &commonpb.File{}
		err = proto.Unmarshal(data, loot.File)
		if err != nil {
			return nil, err
		}
	}

	// Credential Loot
	lootCredFile := filepath.Join(l.LocalCredDir, dbLoot.ID.String())
	if _, err := os.Stat(lootCredFile); !os.IsNotExist(err) {
		data, err := ioutil.ReadFile(lootCredFile)
		if err != nil {
			return nil, err
		}
		loot.Credential = &clientpb.Credential{}
		err = proto.Unmarshal(data, loot.Credential)
		if err != nil {
			return nil, err
		}
	}

	return loot, nil
}

func (l *LocalBackend) All() *clientpb.AllLoot {
	dbSession := db.Session()
	allDBLoot := []*models.Loot{}
	result := dbSession.Where(&models.Loot{}).Find(allDBLoot)
	if result.Error != nil {
		lootLog.Error(result.Error)
		return nil
	}
	all := &clientpb.AllLoot{Loot: []*clientpb.Loot{}}
	for _, dbLoot := range allDBLoot {
		all.Loot = append(all.Loot, &clientpb.Loot{
			LootID: dbLoot.ID.String(),
			Name:   dbLoot.Name,
			Type:   clientpb.LootType(dbLoot.Type),
		})
	}
	return all
}

func (l *LocalBackend) AllOf(lootType clientpb.LootType) *clientpb.AllLoot {
	dbSession := db.Session()
	allDBLoot := []*models.Loot{}
	result := dbSession.Where(&models.Loot{Type: int(lootType)}).Find(allDBLoot)
	if result.Error != nil {
		lootLog.Error(result.Error)
		return nil
	}
	all := &clientpb.AllLoot{Loot: []*clientpb.Loot{}}
	for _, dbLoot := range allDBLoot {
		all.Loot = append(all.Loot, &clientpb.Loot{
			LootID: dbLoot.ID.String(),
			Name:   dbLoot.Name,
			Type:   clientpb.LootType(dbLoot.Type),
		})
	}
	return all
}
