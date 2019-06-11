package generate

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"encoding/json"

	"github.com/bishopfox/sliver/server/db"
)

const (
	profilesBucketName = "profiles"
)

// ProfileSave - Save a sliver profile to disk
func ProfileSave(name string, config *SliverConfig) error {
	bucket, err := db.GetBucket(profilesBucketName)
	if err != nil {
		return err
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return bucket.Set(name, configJSON)
}

// ProfileByName - Fetch a single profile from the database
func ProfileByName(name string) (*SliverConfig, error) {
	bucket, err := db.GetBucket(profilesBucketName)
	if err != nil {
		return nil, err
	}
	rawProfile, err := bucket.Get(name)
	config := &SliverConfig{}
	err = json.Unmarshal(rawProfile, config)
	return config, err
}

// Profiles - Fetch a map of name<->profiles current in the database
func Profiles() map[string]*SliverConfig {
	bucket, err := db.GetBucket(profilesBucketName)
	if err != nil {
		return nil
	}
	rawProfiles, err := bucket.Map("")
	if err != nil {
		return nil
	}

	profiles := map[string]*SliverConfig{}
	for name, rawProfile := range rawProfiles {
		config := &SliverConfig{}
		err := json.Unmarshal(rawProfile, config)
		if err != nil {
			continue // We should probably log these failures ...
		}
		profiles[name] = config
	}
	return profiles
}
