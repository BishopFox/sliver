package config

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
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/log"
)

const (
	configBucketName = "config"
)

var (
	configRootLog = log.NamedLogger("config", "root")
)

// SetConfig - Set config key/value pair
func SetConfig(key, value string) error {
	bucket, err := db.GetBucket(configBucketName)
	if err != nil {
		return err
	}
	return bucket.Set(key, []byte(value))
}

// GetConfig - Get config value
func GetConfig(key string) (string, error) {
	bucket, err := db.GetBucket(configBucketName)
	if err != nil {
		return "", err
	}
	value, err := bucket.Get(key)
	return string(value), err
}

// ListConfig - List config contents
func ListConfig() (map[string]string, error) {
	bucket, err := db.GetBucket(configBucketName)
	if err != nil {
		return nil, err
	}
	config, err := bucket.Map("")
	if err != nil {
		return nil, err
	}
	// Convert []byte's to strings
	configStr := map[string]string{}
	for key, value := range config {
		configStr[key] = string(value)
	}
	return configStr, nil
}
