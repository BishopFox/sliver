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
	"io"
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
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
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
		id, _ := models.ParseUUID(dbConfig.ImplantProfileID)
		if id == models.NilUUID() {
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

// ImplantBuildSave stores the final file and its authentication rows together.
func ImplantBuildSave(build *clientpb.ImplantBuild, config *clientpb.ImplantConfig, fPath string) error {
	return implantBuildSave(build, config, fPath, publishNewImplantFile)
}

// The private publish argument lets tests fail the final file operation after
// all database writes have been prepared in the transaction.
func implantBuildSave(build *clientpb.ImplantBuild, config *clientpb.ImplantConfig, fPath string, publish func(string, string) error) error {
	if build == nil || config == nil {
		return errors.New("missing implant build or config")
	}
	rootAppDir, err := filepath.Abs(assets.GetRootAppDir())
	if err != nil {
		return err
	}
	fPath, err = filepath.Abs(fPath)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(rootAppDir, fPath)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("invalid path '%s' is not a subdirectory of '%s'", fPath, rootAppDir)
	}
	data, err := os.ReadFile(fPath)
	if err != nil {
		return err
	}
	return saveImplantBuildData(build, config, data, "", publish)
}

//nolint:gocyclo // Row updates, publication, and rollback share one save transaction.
func saveImplantBuildData(build *clientpb.ImplantBuild, config *clientpb.ImplantConfig, data []byte, resourceType string, publish func(string, string) error) (returnErr error) {
	if build == nil || config == nil {
		return errors.New("missing implant build or config")
	}
	rootAppDir, err := filepath.Abs(assets.GetRootAppDir())
	if err != nil {
		return err
	}
	// An external Sliver build may already have a credential-bearing row. If
	// final publication fails, revoke that exact row after the transaction rolls
	// back. A completed build is never revoked by a failed later save attempt.
	if build.ID != "" {
		originalID, err := models.ParseUUID(build.ID)
		if err != nil {
			return fmt.Errorf("invalid implant build ID: %w", err)
		}
		var original models.ImplantBuild
		if err := db.Session().Where("id = ? AND name = ?", originalID, build.Name).First(&original).Error; err != nil {
			return err
		}
		pending := original.SHA256 == ""
		if !pending {
			return errors.New("implant build is already complete; use ImplantBuildUpdateFile")
		}
		if _, err := os.Stat(filepath.Join(rootAppDir, "builds", build.ID)); err == nil {
			return errors.New("implant build file already exists; use ImplantBuildUpdateFile")
		} else if !os.IsNotExist(err) {
			return err
		}
		defer func() {
			if returnErr == nil || !pending {
				return
			}
			if _, err := os.Stat(filepath.Join(rootAppDir, "builds", build.ID)); err == nil || !os.IsNotExist(err) {
				return
			}
			if err := db.Session().Where("id = ? AND name = ? AND sha256 = ?", originalID, build.Name, "").Delete(&models.ImplantBuild{}).Error; err != nil {
				returnErr = errors.Join(returnErr, fmt.Errorf("revoke unfinished implant build: %w", err))
			}
		}()
	}
	md5Hash, sha1Hash, sha256Hash := computeHashes(data)
	buildsDir, err := getBuildsDir()
	if err != nil {
		return err
	}
	stagedPath, err := stageImplantFile(buildsDir, data)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(stagedPath) }()

	workingBuild := proto.Clone(build).(*clientpb.ImplantBuild)
	workingBuild.ImplantID = encoders.GetRandomID()
	workingBuild.MD5, workingBuild.SHA1, workingBuild.SHA256 = md5Hash, sha1Hash, sha256Hash
	var implantBuild *clientpb.ImplantBuild
	var finalPath string
	published := false
	err = db.Session().Transaction(func(tx *gorm.DB) error {
		resource := &models.ResourceID{Name: workingBuild.Name, Type: resourceType, Value: workingBuild.ImplantID}
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(resource).Error; err != nil {
			return err
		}
		modelConfig := models.ImplantConfigFromProtobuf(config)
		if config.ID != "" {
			if err := tx.First(&models.ImplantConfig{}, "id = ?", modelConfig.ID).Error; err != nil {
				return err
			}
			if err := tx.Save(modelConfig).Error; err != nil {
				return err
			}
		} else if err := tx.Create(modelConfig).Error; err != nil {
			return err
		}
		workingBuild.ImplantConfigID = modelConfig.ID.String()
		modelBuild := models.ImplantBuildFromProtobuf(workingBuild)
		if workingBuild.ID != "" {
			pending := tx.Model(&models.ImplantBuild{}).
				Where("id = ? AND name = ? AND sha256 = ?", modelBuild.ID, workingBuild.Name, "").
				Select("ImplantID", "MD5", "SHA1", "SHA256", "ImplantConfigID", "Stage").Updates(modelBuild)
			if pending.Error != nil {
				return pending.Error
			}
			if pending.RowsAffected != 1 {
				return errors.New("implant build is no longer pending")
			}
		} else if err := tx.Create(modelBuild).Error; err != nil {
			return err
		}
		implantBuild = modelBuild.ToProtobuf()
		finalPath = filepath.Join(buildsDir, implantBuild.ID)
		if err := publish(stagedPath, finalPath); err != nil {
			return err
		}
		published = true
		return nil
	})
	if err != nil {
		if published {
			if removeErr := os.Remove(finalPath); removeErr != nil && !os.IsNotExist(removeErr) {
				err = errors.Join(err, fmt.Errorf("remove uncommitted implant file: %w", removeErr))
			}
		}
		return err
	}
	overwriteImplantBuild(build, implantBuild)
	watchtower.AddImplantToWatchlist(implantBuild)
	storageLog.Infof("%s -> %s", implantBuild.ID, implantBuild.Name)
	return nil
}

// ImplantBuildUpdateFile updates an existing build's stored binary and checksums.
func ImplantBuildUpdateFile(build *clientpb.ImplantBuild, data []byte) error {
	return implantBuildUpdateFile(build, data, os.Rename)
}

//nolint:gocyclo // Publication, backup restoration, and metadata updates share a rollback boundary.
func implantBuildUpdateFile(build *clientpb.ImplantBuild, data []byte, publish func(string, string) error) error {
	if build == nil {
		return errors.New("nil implant build")
	}
	if build.ID == "" {
		return errors.New("missing implant build id")
	}
	buildID, err := models.ParseUUID(build.ID)
	if err != nil {
		return fmt.Errorf("invalid implant build ID: %w", err)
	}
	buildsDir, err := getBuildsDir()
	if err != nil {
		return err
	}
	finalPath := filepath.Join(buildsDir, build.ID)
	fileInfo, err := os.Stat(finalPath)
	if err != nil {
		return err
	}
	if !fileInfo.Mode().IsRegular() {
		return errors.New("stored implant build is not a regular file")
	}
	stagedPath, err := stageImplantFile(buildsDir, data)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(stagedPath) }()
	backupDir, err := os.MkdirTemp(buildsDir, ".implant-backup-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(backupDir) }()
	backupPath := filepath.Join(backupDir, "previous")
	md5Hash, sha1Hash, sha256Hash := computeHashes(data)
	var updatedBuild *clientpb.ImplantBuild
	var oldMD5, oldSHA1, oldSHA256 string
	movedOriginal := false
	err = db.Session().Transaction(func(tx *gorm.DB) error {
		var current models.ImplantBuild
		if err := tx.Where("id = ? AND name = ?", buildID, build.Name).First(&current).Error; err != nil {
			return err
		}
		if current.SHA256 != build.SHA256 {
			return errors.New("implant build changed since it was read")
		}
		oldMD5, oldSHA1, oldSHA256 = current.MD5, current.SHA1, current.SHA256
		result := tx.Model(&models.ImplantBuild{}).Where("id = ? AND name = ? AND sha256 = ?", buildID, build.Name, oldSHA256).
			Select("MD5", "SHA1", "SHA256").Updates(&models.ImplantBuild{MD5: md5Hash, SHA1: sha1Hash, SHA256: sha256Hash})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("implant build changed during file update")
		}
		current.MD5, current.SHA1, current.SHA256 = md5Hash, sha1Hash, sha256Hash
		updatedBuild = current.ToProtobuf()
		if err := os.Rename(finalPath, backupPath); err != nil {
			return err
		}
		movedOriginal = true
		if err := publish(stagedPath, finalPath); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if movedOriginal {
			if removeErr := os.Remove(finalPath); removeErr != nil && !os.IsNotExist(removeErr) {
				err = errors.Join(err, fmt.Errorf("remove failed implant update: %w", removeErr))
			}
			if restoreErr := os.Rename(backupPath, finalPath); restoreErr != nil {
				err = errors.Join(err, fmt.Errorf("restore previous implant file: %w", restoreErr))
			} else {
				// A failed commit may still have reached the database. Restore
				// only this build's attempted hashes after its old file is back.
				rollback := db.Session().Model(&models.ImplantBuild{}).
					Where("id = ? AND name = ? AND sha256 = ?", buildID, build.Name, sha256Hash).
					Select("MD5", "SHA1", "SHA256").Updates(&models.ImplantBuild{MD5: oldMD5, SHA1: oldSHA1, SHA256: oldSHA256})
				if rollback.Error != nil {
					err = errors.Join(err, fmt.Errorf("restore previous implant checksums: %w", rollback.Error))
				}
			}
		}
		return err
	}
	overwriteImplantBuild(build, updatedBuild)
	watchtower.AddImplantToWatchlist(updatedBuild)
	storageLog.Infof("updated build %s (%s)", build.ID, build.Name)
	return nil
}

// overwriteImplantBuild replaces dst message content without copying internal
// proto runtime state that includes lock values.
func overwriteImplantBuild(dst *clientpb.ImplantBuild, src *clientpb.ImplantBuild) {
	if dst == nil || src == nil {
		return
	}
	proto.Reset(dst)
	proto.Merge(dst, src)
}

func SaveStage(build *clientpb.ImplantBuild, config *clientpb.ImplantConfig, stage2 []byte, stageType string) error {
	return saveStage(build, config, stage2, stageType, publishNewImplantFile)
}

func saveStage(build *clientpb.ImplantBuild, config *clientpb.ImplantConfig, stage2 []byte, stageType string, publish func(string, string) error) error {
	return saveImplantBuildData(build, config, stage2, stageType, publish)
}

func stageImplantFile(buildsDir string, data []byte) (string, error) {
	staged, err := os.CreateTemp(buildsDir, ".implant-build-*")
	if err != nil {
		return "", err
	}
	path := staged.Name()
	if written, err := staged.Write(data); err != nil {
		_ = staged.Close()
		_ = os.Remove(path)
		return "", err
	} else if written != len(data) {
		_ = staged.Close()
		_ = os.Remove(path)
		return "", io.ErrShortWrite
	}
	if err := staged.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

// Hard-linking a staged file in the same directory publishes it atomically
// without replacing a build that another finalization already published.
// The caller removes the staged name after the transaction resolves.
func publishNewImplantFile(stagedPath, finalPath string) error {
	return os.Link(stagedPath, finalPath)
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
