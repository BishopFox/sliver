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

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/encoders"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/watchtower"
	"github.com/gofrs/uuid"
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
func ImplantConfigSave(config *clientpb.ImplantConfig) (*clientpb.ImplantConfig, error) {

	dbConfig, err := db.ImplantConfigByID(config.ID)
	if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
		return nil, err
	}

	modelConfig := models.ImplantConfigFromProtobuf(config)
	dbSession := db.Session()
	if errors.Is(err, db.ErrRecordNotFound) {
		err = dbSession.Clauses(clause.OnConflict{
			UpdateAll: true,
		}).Create(modelConfig).Error

	} else {
		id, _ := uuid.FromString(dbConfig.ImplantProfileID)
		if id == uuid.Nil {
			modelConfig.ImplantProfileID = nil
		} else {
			modelConfig.ImplantProfileID = &id
		}

		// this avoids gorm saving duplicate c2 objects ...
		tempC2 := modelConfig.C2
		modelConfig.C2 = nil
		err = dbSession.Save(modelConfig).Error
		modelConfig.C2 = tempC2
	}

	return modelConfig.ToProtobuf(), err
}

// ImplantBuildSave - Saves a binary file into the database
func ImplantBuildSave(build *clientpb.ImplantBuild, config *clientpb.ImplantConfig, fPath string) error {
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

	implantID := uint64(encoders.GetRandomID())
	err = db.SaveResourceID(&clientpb.ResourceID{
		Type:  "",
		Value: implantID,
		Name:  build.Name,
	})
	if err != nil {
		return err
	}

	build.ImplantID = implantID
	build.MD5 = md5Hash
	build.SHA1 = sha1Hash
	build.SHA256 = sha256Hash

	config, err = db.SaveImplantConfig(config)
	if err != nil {
		return err
	}

	build.ImplantConfigID = config.ID
	implantBuild, err := db.SaveImplantBuild(build)
	if err != nil {
		return err
	}

	watchtower.AddImplantToWatchlist(implantBuild)
	storageLog.Infof("%s -> %s", implantBuild.ID, implantBuild.Name)
	return os.WriteFile(filepath.Join(buildsDir, implantBuild.ID), data, 0600)
}

func SaveStage(build *clientpb.ImplantBuild, config *clientpb.ImplantConfig, stage2 []byte, stageType string) error {
	md5Hash, sha1Hash, sha256Hash := computeHashes(stage2)
	buildsDir, err := getBuildsDir()
	if err != nil {
		return err
	}

	implantID := uint64(encoders.GetRandomID())
	err = db.SaveResourceID(&clientpb.ResourceID{
		Type:  stageType,
		Value: implantID,
		Name:  build.Name,
	})
	if err != nil {
		return err
	}

	build.ImplantID = implantID
	build.MD5 = md5Hash
	build.SHA1 = sha1Hash
	build.SHA256 = sha256Hash

	config, err = db.SaveImplantConfig(config)
	if err != nil {
		return err
	}

	build.ImplantConfigID = config.ID
	implantBuild, err := db.SaveImplantBuild(build)
	if err != nil {
		return err
	}

	watchtower.AddImplantToWatchlist(implantBuild)
	storageLog.Infof("%s -> %s", implantBuild.ID, implantBuild.Name)
	return os.WriteFile(filepath.Join(buildsDir, implantBuild.ID), stage2, 0600)
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
func ImplantFileFromBuild(build *clientpb.ImplantBuild) ([]byte, error) {
	buildsDir, err := getBuildsDir()
	if err != nil {
		return nil, err
	}
	buildFilePath := path.Join(buildsDir, build.ID)
	if _, err := os.Stat(buildFilePath); os.IsNotExist(err) {
		return nil, ErrImplantBuildFileNotFound
	}
	return os.ReadFile(buildFilePath)
}

// ImplantFileDelete - Delete the implant from the file system
func ImplantFileDelete(build *clientpb.ImplantBuild) error {
	buildsDir, err := getBuildsDir()
	if err != nil {
		return err
	}
	buildFilePath := filepath.Join(buildsDir, build.ID)
	if _, err := os.Stat(buildFilePath); os.IsNotExist(err) {
		return ErrImplantBuildFileNotFound
	}
	return os.Remove(buildFilePath)
}
