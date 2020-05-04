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
	// implantBucketName - Name of the bucket that stores data related to slivers
	implantBucketName = "implants"

	// implantConfigNamespace - Namespace that contains sliver configs
	implantConfigNamespace   = "config"
	implantFileNamespace     = "file"
	implantDatetimeNamespace = "datetime"
)

var (
	storageLog = log.NamedLogger("generate", "storage")

	// ErrImplantNotFound - More descriptive 'key not found' error
	ErrImplantNotFound = errors.New("Implant not found")
)

// ImplantConfigByName - Get a implant's config by it's codename
func ImplantConfigByName(name string) (*ImplantConfig, error) {
	bucket, err := db.GetBucket(implantBucketName)
	if err != nil {
		return nil, err
	}
	rawConfig, err := bucket.Get(fmt.Sprintf("%s.%s", implantConfigNamespace, name))
	if err != nil {
		return nil, err
	}
	config := &ImplantConfig{}
	err = json.Unmarshal(rawConfig, config)
	return config, err
}

// ImplantConfigMap - Get a sliver's config by it's codename
func ImplantConfigMap() (map[string]*ImplantConfig, error) {
	bucket, err := db.GetBucket(implantBucketName)
	if err != nil {
		return nil, err
	}
	ls, err := bucket.List(implantConfigNamespace)
	configs := map[string]*ImplantConfig{}
	for _, config := range ls {
		sliverName := config[len(implantConfigNamespace)+1:]
		config, err := ImplantConfigByName(sliverName)
		if err != nil {
			continue
		}
		configs[sliverName] = config
	}
	return configs, nil
}

// ImplantConfigSave - Save a configuration to the database
func ImplantConfigSave(config *ImplantConfig) error {
	bucket, err := db.GetBucket(implantBucketName)
	if err != nil {
		return err
	}
	rawConfig, err := json.Marshal(config)
	if err != nil {
		return err
	}
	storageLog.Infof("Saved config for '%s'", config.Name)
	return bucket.Set(fmt.Sprintf("%s.%s", implantConfigNamespace, config.Name), rawConfig)
}

// ImplantFileSave - Saves a binary file into the database
func ImplantFileSave(name, fPath string) error {
	bucket, err := db.GetBucket(implantBucketName)
	if err != nil {
		return err
	}

	rootAppDir, _ := filepath.Abs(assets.GetRootAppDir())
	fPath, _ = filepath.Abs(fPath)
	if !strings.HasPrefix(fPath, rootAppDir) {
		return fmt.Errorf("Invalid path '%s' is not a subdirectory of '%s'", fPath, rootAppDir)
	}

	data, err := ioutil.ReadFile(fPath)
	if err != nil {
		return err
	}
	storageLog.Infof("Saved '%s' file to database %d byte(s)", name, len(data))
	bucket.Set(fmt.Sprintf("%s.%s", implantDatetimeNamespace, name), []byte(time.Now().Format(time.RFC1123)))
	return bucket.Set(fmt.Sprintf("%s.%s", implantFileNamespace, name), data)
}

// ImplantFileByName - Saves a binary file into the database
func ImplantFileByName(name string) ([]byte, error) {
	bucket, err := db.GetBucket(implantBucketName)
	if err != nil {
		return nil, err
	}
	sliver, err := bucket.Get(fmt.Sprintf("%s.%s", implantFileNamespace, name))
	if err != nil {
		return nil, ErrImplantNotFound
	}
	return sliver, nil
}

// ImplantFiles - List all sliver files
func ImplantFiles() ([]string, error) {
	bucket, err := db.GetBucket(implantBucketName)
	if err != nil {
		return nil, err
	}
	keys, err := bucket.List(implantFileNamespace)
	if err != nil {
		return nil, err
	}

	// Remove namespace prefix
	names := []string{}
	for _, key := range keys {
		names = append(names, key[len(implantFileNamespace):])
	}
	return names, nil
}
