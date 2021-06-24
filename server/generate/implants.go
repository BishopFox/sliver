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
	"crypto/md5"
	"encoding/hex"
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
	"github.com/bishopfox/sliver/server/watchtower"
)

var (
	storageLog = log.NamedLogger("generate", "storage")

	// ErrImplantBuildFileNotFound - More descriptive 'key not found' error
	ErrImplantBuildFileNotFound = errors.New("Implant build file not found")
)

func getBuildsDir() (string, error) {
	buildsDir := filepath.Join(assets.GetRootAppDir(), "builds")
	storageLog.Debugf("Builds dir: %s", buildsDir)
	if _, err := os.Stat(buildsDir); os.IsNotExist(err) {
		err = os.MkdirAll(buildsDir, 0700)
		if err != nil {
			return "", err
		}
	}
	return buildsDir, nil
}

// ImplantBuildSave - Saves a binary file into the database
func ImplantBuildSave(name string, config *models.ImplantConfig, fPath string) error {
	rootAppDir, _ := filepath.Abs(assets.GetRootAppDir())
	fPath, _ = filepath.Abs(fPath)
	if !strings.HasPrefix(fPath, rootAppDir) {
		return fmt.Errorf("Invalid path '%s' is not a subdirectory of '%s'", fPath, rootAppDir)
	}

	data, err := ioutil.ReadFile(fPath)
	if err != nil {
		return err
	}
	sum := md5.Sum(data)
	hash := hex.EncodeToString(sum[:])
	buildsDir, err := getBuildsDir()
	if err != nil {
		return err
	}
	dbSession := db.Session()
	implantBuild := &models.ImplantBuild{
		Name:          name,
		ImplantConfig: (*config),
		Checksum:      hash,
	}
	watchtower.AddImplantToWatchlist(implantBuild)
	result := dbSession.Create(&implantBuild)
	if result.Error != nil {
		return result.Error
	}
	storageLog.Infof("%s -> %s", implantBuild.ID, implantBuild.Name)
	return ioutil.WriteFile(path.Join(buildsDir, implantBuild.ID.String()), data, 0600)
}

// ImplantFileFromBuild - Saves a binary file into the database
func ImplantFileFromBuild(build *models.ImplantBuild) ([]byte, error) {
	buildsDir, err := getBuildsDir()
	if err != nil {
		return nil, err
	}
	buildFilePath := path.Join(buildsDir, build.ID.String())
	if _, err := os.Stat(buildFilePath); os.IsNotExist(err) {
		return nil, ErrImplantBuildFileNotFound
	}
	return ioutil.ReadFile(buildFilePath)
}

// ImplantFileDelete - Delete the implant from the file system
func ImplantFileDelete(build *models.ImplantBuild) error {
	buildsDir, err := getBuildsDir()
	if err != nil {
		return err
	}
	buildFilePath := path.Join(buildsDir, build.ID.String())
	if _, err := os.Stat(buildFilePath); os.IsNotExist(err) {
		return ErrImplantBuildFileNotFound
	}
	return os.Remove(buildFilePath)
}
