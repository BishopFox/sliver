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
	"github.com/bishopfox/sliver/server/assets"
)

const (
	// MaxLootSize - The maximum size of a loot file in bytes
	MaxLootSize = 2 * 1024 * 1024 * 1024 // 2Gb, shouldn't matter the gRPC message size limit is 2Gb
)

// LootBackend - The interface any loot backend must implement
type LootBackend interface {
	Add(*clientpb.Loot) (*clientpb.Loot, error)
	Rm(string) error
	Update(*clientpb.Loot) (*clientpb.Loot, error)
	GetContent(string, bool) (*clientpb.Loot, error)
	All() *clientpb.AllLoot
}

// LootStore - The struct that represents the loot store
type LootStore struct {
	backend LootBackend
}

// Add - Add a piece of loot to the loot store
func (l *LootStore) Add(lootReq *clientpb.Loot) (*clientpb.Loot, error) {
	if lootReq.File != nil && MaxLootSize < len(lootReq.File.Data) {
		return nil, errors.New("max loot size exceeded")
	}
	loot, err := l.backend.Add(lootReq)
	if err != nil {
		return nil, err
	}
	return loot, nil
}

// Update - Update a piece of loot in the loot store
func (l *LootStore) Update(lootReq *clientpb.Loot) (*clientpb.Loot, error) {
	loot, err := l.backend.Update(lootReq)
	if err != nil {
		return nil, err
	}
	return loot, nil
}

// Remove - Remove a piece of loot from the loot store
func (l *LootStore) Rm(lootID string) error {
	err := l.backend.Rm(lootID)
	if err != nil {
		return err
	}
	return nil
}

// GetContent - Get the content of a piece of loot from the loot store
func (l *LootStore) GetContent(lootID string, eager bool) (*clientpb.Loot, error) {
	return l.backend.GetContent(lootID, eager)
}

// All - Get all loot from the loot store
func (l *LootStore) All() *clientpb.AllLoot {
	return l.backend.All()
}

// GetLootStore - Get an instances of the core LootStore
func GetLootStore() *LootStore {
	return &LootStore{
		backend: &LocalBackend{
			LocalFileDir: GetLootDir(),
		},
	}
}

// GetLootDir - Get the directory that contains all loot
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
