package extensions

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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/util"
)

// Install an extension from a directory
func InstallFromDir(extLocalPath string) error {
	manifestData, err := ioutil.ReadFile(filepath.Join(extLocalPath, ManifestFileName))
	if err != nil {
		return fmt.Errorf("error reading %s: %s", ManifestFileName, err)
	}
	manifest, err := ParseExtensionManifest(manifestData)
	if err != nil {
		return fmt.Errorf("error parsing %s: %s", ManifestFileName, err)
	}
	installPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(manifest.CommandName))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		confirm := false
		msg := fmt.Sprintf("Extension '%s' already exists. Overwrite current install?", manifest.CommandName)
		prompt := &survey.Confirm{Message: msg}
		survey.AskOne(prompt, &confirm)
		if !confirm {
			return err
		}
		forceRemoveAll(installPath)
	}

	err = os.MkdirAll(installPath, 0o700)
	if err != nil {
		return fmt.Errorf("error creating extension directory: %s", err)
	}
	err = ioutil.WriteFile(filepath.Join(installPath, ManifestFileName), manifestData, 0o600)
	if err != nil {
		forceRemoveAll(installPath)
		return fmt.Errorf("error writing %s: %s", ManifestFileName, err)
	}

	for _, manifestFile := range manifest.Files {
		if manifestFile.Path != "" {
			src := filepath.Join(extLocalPath, util.ResolvePath(manifestFile.Path))
			dst := filepath.Join(installPath, util.ResolvePath(manifestFile.Path))
			err := util.CopyFile(src, dst)
			if err != nil {
				forceRemoveAll(installPath)
				return fmt.Errorf("error copying file '%s' -> '%s': %s", src, dst, err)
			}
		}
	}
	return nil
}

// InstallFromFilePath - Install an extension from a .tar.gz file
func InstallFromFilePath(extLocalPath string, autoOverwrite bool) (*string, error) {
	manifestData, err := util.ReadFileFromTarGz(extLocalPath, fmt.Sprintf("./%s", ManifestFileName))
	if err != nil {
		return nil, fmt.Errorf("failed to read %s from '%s': %s", ManifestFileName, extLocalPath, err)
	}
	manifest, err := ParseExtensionManifest(manifestData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %s", ManifestFileName, err)
	}
	installPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(manifest.CommandName))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		if !autoOverwrite {
			confirm := false
			msg := fmt.Sprintf("Extension '%s' already exists. Overwrite current install?", manifest.CommandName)
			prompt := &survey.Confirm{Message: msg}
			survey.AskOne(prompt, &confirm)
			if !confirm {
				return nil, err
			}
		}
		forceRemoveAll(installPath)
	}

	// con.PrintInfof("Installing extension '%s' (%s) ... ", manifest.CommandName, manifest.Version)
	err = os.MkdirAll(installPath, 0o700)
	if err != nil {
		return nil, fmt.Errorf("failed to create extension directory: %s", err)
	}
	err = ioutil.WriteFile(filepath.Join(installPath, ManifestFileName), manifestData, 0o600)
	if err != nil {
		forceRemoveAll(installPath)
		return nil, fmt.Errorf("failed to write %s: %s", ManifestFileName, err)
	}
	for _, manifestFile := range manifest.Files {
		if manifestFile.Path != "" {
			err = installArtifact(extLocalPath, installPath, manifestFile.Path)
			if err != nil {
				forceRemoveAll(installPath)
				return nil, fmt.Errorf("failed to install file: %s", err)
			}
		}
	}
	return &installPath, err
}

func installArtifact(extGzFilePath string, installPath string, artifactPath string) error {
	data, err := util.ReadFileFromTarGz(extGzFilePath, "."+artifactPath)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("archive path '%s' is empty", "."+artifactPath)
	}
	localArtifactPath := filepath.Join(installPath, util.ResolvePath(artifactPath))
	artifactDir := filepath.Dir(localArtifactPath)
	if _, err := os.Stat(artifactDir); os.IsNotExist(err) {
		err := os.MkdirAll(artifactDir, 0o700)
		if err != nil {
			return err
		}
	}
	err = ioutil.WriteFile(localArtifactPath, data, 0o600)
	if err != nil {
		return err
	}
	return nil
}
