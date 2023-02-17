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
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/watchtower"
	"gorm.io/gorm/clause"
)

var (
	storageLog = log.NamedLogger("generate", "storage")

	// ErrImplantBuildFileNotFound - More descriptive 'key not found' error
	ErrImplantBuildFileNotFound = errors.New("implant build file not found")
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

// ImplantConfigSave - Save only the config to the database
func ImplantConfigSave(config *models.ImplantConfig) error {
	dbSession := db.Session()
	result := dbSession.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&config)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// ImplantBuildSave - Saves a binary file into the database
func ImplantBuildSave(name string, config *models.ImplantConfig, fPath string) error {
	rootAppDir, _ := filepath.Abs(assets.GetRootAppDir())
	fPath, _ = filepath.Abs(fPath)
	if !strings.HasPrefix(fPath, rootAppDir) {
		return fmt.Errorf("invalid path '%s' is not a subdirectory of '%s'", fPath, rootAppDir)
	}

	data, err := os.ReadFile(fPath)
	if err != nil {
		return err
	}
	md5Hash, sha1Hash, sha256Hash := computeHashes(data)
	buildsDir, err := getBuildsDir()
	if err != nil {
		return err
	}
	dbSession := db.Session()
	implantBuild := &models.ImplantBuild{
		Name:          name,
		ImplantConfig: (*config),
		MD5:           md5Hash,
		SHA1:          sha1Hash,
		SHA256:        sha256Hash,
	}
	watchtower.AddImplantToWatchlist(implantBuild)
	result := dbSession.Create(&implantBuild)
	if result.Error != nil {
		return result.Error
	}
	storageLog.Infof("%s -> %s", implantBuild.ID, implantBuild.Name)
	return os.WriteFile(filepath.Join(buildsDir, implantBuild.ID.String()), data, 0600)
}

func computeHashes(data []byte) (string, string, string) {
	md5Sum := md5.Sum(data)
	md5Hash := hex.EncodeToString(md5Sum[:])
	sha1Sum := sha1.Sum(data)
	sha1Hash := hex.EncodeToString(sha1Sum[:])
	sha256Sum := sha256.Sum256(data)
	sha256Hash := hex.EncodeToString(sha256Sum[:])
	return md5Hash, sha1Hash, sha256Hash
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
	return os.ReadFile(buildFilePath)
}

// ImplantFileDelete - Delete the implant from the file system
func ImplantFileDelete(build *models.ImplantBuild) error {
	buildsDir, err := getBuildsDir()
	if err != nil {
		return err
	}
	buildFilePath := filepath.Join(buildsDir, build.ID.String())
	if _, err := os.Stat(buildFilePath); os.IsNotExist(err) {
		return ErrImplantBuildFileNotFound
	}
	return os.Remove(buildFilePath)
}
