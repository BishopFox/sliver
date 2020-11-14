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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
	"gorm.io/gorm"
)

var (
	storageLog = log.NamedLogger("generate", "storage")

	// ErrImplantNotFound - More descriptive 'key not found' error
	ErrImplantNotFound = errors.New("Implant not found")
)

// ImplantConfigByName - Get a implant's config by it's codename
func ImplantConfigByName(name string) (*models.ImplantConfig, error) {
	config := &models.ImplantConfig{}
	dbSession := db.Session()
	result := dbSession.Where(&models.ImplantConfig{Name: name}).First(&config)
	return config, result.Error
}

// ImplantConfigSave - Save a configuration to the database
func ImplantConfigSave(config *models.ImplantConfig) error {
	dbSession := db.Session()
	var result *gorm.DB
	if config.ID != "" {
		result = dbSession.Save(&config)
	} else {
		result = dbSession.Create(&config)
	}
	return result.Error
}

// ImplantBuildSave - Saves a binary file into the database
func ImplantBuildSave(config *models.ImplantConfig, fPath string) error {

	rootAppDir, _ := filepath.Abs(assets.GetRootAppDir())
	fPath, _ = filepath.Abs(fPath)
	if !strings.HasPrefix(fPath, rootAppDir) {
		return fmt.Errorf("Invalid path '%s' is not a subdirectory of '%s'", fPath, rootAppDir)
	}

	data, err := ioutil.ReadFile(fPath)
	if err != nil {
		return err
	}

	buildsDir := filepath.Join(GetRootAppDir(), "builds")
	storageLog.Debugf("Builds dir: %s", buildsDir)
	if _, err := os.Stat(buildsDir); os.IsNotExist(err) {
		err = os.MkdirAll(buildsDir, 0700)
		if err != nil {
			return err
		}
	}

	dbSession := db.Session()
	implantBuild := &models.ImplantBuild{
		Name:          name,
		ImplantConfig: config,
	}
	result := dbSession.Create(&implantBuild)
	if result.Error != nil {
		return result.Error
	}
	storageLog.Infof("%s -> %s", implantBuild.name, implantBuild.Name)
	return ioutil.WriteFile(path.Join(buildsDir, implantBuild.ID), data, 0600)
}

// ImplantFileByName - Saves a binary file into the database
func ImplantFileByName(name string) ([]byte, error) {

	return nil, ErrImplantNotFound
}

// ImplantFiles - List all sliver files
func ImplantFiles() ([]string, error) {
	names := []string{}
	return names, nil
}
