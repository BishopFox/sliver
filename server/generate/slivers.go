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
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/log"
)

const (
	// sliverBucketName - Name of the bucket that stores data related to slivers
	sliverBucketName = "slivers"

	// sliverConfigNamespace - Namespace that contains sliver configs
	sliverConfigNamespace   = "config"
	sliverFileNamespace     = "file"
	sliverDatetimeNamespace = "datetime"
)

var (
	storageLog = log.NamedLogger("generate", "storage")

	// ErrSliverNotFound - More descriptive 'key not found' error
	ErrSliverNotFound = errors.New("Sliver not found")
)

// SliverConfigByName - Get a sliver's config by it's codename
func SliverConfigByName(name string) (*SliverConfig, error) {
	bucket, err := db.GetBucket(sliverBucketName)
	if err != nil {
		return nil, err
	}
	rawConfig, err := bucket.Get(fmt.Sprintf("%s.%s", sliverConfigNamespace, name))
	if err != nil {
		return nil, err
	}
	config := &SliverConfig{}
	err = json.Unmarshal(rawConfig, config)
	return config, err
}

// SliverConfigMap - Get a sliver's config by it's codename
func SliverConfigMap() (map[string]*SliverConfig, error) {
	bucket, err := db.GetBucket(sliverBucketName)
	if err != nil {
		return nil, err
	}
	ls, err := bucket.List(sliverConfigNamespace)
	configs := map[string]*SliverConfig{}
	for _, config := range ls {
		sliverName := config[len(sliverConfigNamespace)+1:]
		config, err := SliverConfigByName(sliverName)
		if err != nil {
			continue
		}
		configs[sliverName] = config
	}
	return configs, nil
}

// SliverConfigSave - Save a configuration to the database
func SliverConfigSave(config *SliverConfig) error {
	bucket, err := db.GetBucket(sliverBucketName)
	if err != nil {
		return err
	}
	rawConfig, err := json.Marshal(config)
	if err != nil {
		return err
	}
	storageLog.Infof("Saved config for '%s'", config.Name)
	return bucket.Set(fmt.Sprintf("%s.%s", sliverConfigNamespace, config.Name), rawConfig)
}

// SliverFileSave - Saves a binary file into the database
func SliverFileSave(name, fpath string) error {
	bucket, err := db.GetBucket(sliverBucketName)
	if err != nil {
		return err
	}

	rootAppDir, _ := filepath.Abs(assets.GetRootAppDir())
	fpath, _ = filepath.Abs(fpath)
	if !strings.HasPrefix(fpath, rootAppDir) {
		return fmt.Errorf("Invalid path '%s' is not a subdirectory of '%s'", fpath, rootAppDir)
	}

	data, err := ioutil.ReadFile(fpath)
	if err != nil {
		return err
	}
	storageLog.Infof("Saved '%s' file to database %d byte(s)", name, len(data))
	bucket.Set(fmt.Sprintf("%s.%s", sliverDatetimeNamespace, name), []byte(time.Now().Format(time.RFC1123)))
	return bucket.Set(fmt.Sprintf("%s.%s", sliverFileNamespace, name), data)
}

// SliverFileByName - Saves a binary file into the database
func SliverFileByName(name string) ([]byte, error) {
	bucket, err := db.GetBucket(sliverBucketName)
	if err != nil {
		return nil, err
	}
	sliver, err := bucket.Get(fmt.Sprintf("%s.%s", sliverFileNamespace, name))
	if err != nil {
		return nil, ErrSliverNotFound
	}
	return sliver, nil
}

// SliverFiles - List all sliver files
func SliverFiles() ([]string, error) {
	bucket, err := db.GetBucket(sliverBucketName)
	if err != nil {
		return nil, err
	}
	keys, err := bucket.List(sliverFileNamespace)
	if err != nil {
		return nil, err
	}

	// Remove namespace prefix
	names := []string{}
	for _, key := range keys {
		names = append(names, key[len(sliverFileNamespace):])
	}
	return names, nil
}
