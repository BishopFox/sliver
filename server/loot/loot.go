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
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
)

type LootBackend interface {
	Add(*clientpb.Loot) error
	Rm(string) error
	All() *clientpb.AllLoot
	AllOf(clientpb.LootType) *clientpb.AllLoot
	GetContent(string) (*clientpb.Loot, error)
}

type LootStore struct {
	backend LootBackend
	mirrors []LootBackend
}

func (l *LootStore) Add(loot *clientpb.Loot) error {
	err := l.backend.Add(loot)
	if err != nil {
		return err
	}
	for _, mirror := range l.mirrors {
		mirror.Add(loot)
	}
	return nil
}

func (l *LootStore) Rm(lootID string) error {
	err := l.backend.Rm(lootID)
	if err != nil {
		return err
	}
	for _, mirror := range l.mirrors {
		mirror.Rm(lootID)
	}
	return nil
}

func (l *LootStore) GetContent(lootID string) (*clientpb.Loot, error) {
	return l.backend.GetContent(lootID)
}

func (l *LootStore) All() *clientpb.AllLoot {
	return l.backend.All()
}

func (l *LootStore) AllOf(lootType clientpb.LootType) *clientpb.AllLoot {
	return l.backend.AllOf(lootType)
}

func GetLootStore() *LootStore {
	return &LootStore{
		backend: &LocalBackend{
			LocalFileDir: GetLootFileDir(),
			LocalCredDir: GetLootCredentialDir(),
		},
	}
}

func GetLootDir() string {
	lootDir := filepath.Join(assets.GetRootAppDir(), "loot")
	if _, err := os.Stat(lootDir); os.IsNotExist(err) {
		err = os.MkdirAll(lootDir, 0700)
		if err != nil {
			panic(err.Error())
		}
	}
	return lootDir
}

func GetLootFileDir() string {
	lootFileDir := filepath.Join(GetLootDir(), "files")
	if _, err := os.Stat(lootFileDir); os.IsNotExist(err) {
		err = os.MkdirAll(lootFileDir, 0700)
		if err != nil {
			panic(err.Error())
		}
	}
	return lootFileDir
}

func GetLootCredentialDir() string {
	lootCredDir := filepath.Join(GetLootDir(), "credentials")
	if _, err := os.Stat(lootCredDir); os.IsNotExist(err) {
		err = os.MkdirAll(lootCredDir, 0700)
		if err != nil {
			panic(err.Error())
		}
	}
	return lootCredDir
}
