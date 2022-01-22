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
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/util"
	"github.com/desertbit/grumble"
)

// ExtensionsInstallCmd - Install an extension
func ExtensionsInstallCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	extLocalPath := ctx.Args.String("path")
	fi, err := os.Stat(extLocalPath)
	if os.IsNotExist(err) {
		con.PrintErrorf("Extension path '%s' does not exist", extLocalPath)
		return
	}
	if !fi.IsDir() {
		InstallFromFilePath(extLocalPath, false, con)
	} else {
		installFromDir(extLocalPath, con)
	}
}

// Install an extension from a directory
func installFromDir(extLocalPath string, con *console.SliverConsoleClient) {
	manifestData, err := ioutil.ReadFile(filepath.Join(extLocalPath, ManifestFileName))
	if err != nil {
		con.PrintErrorf("Error reading %s: %s", ManifestFileName, err)
		return
	}
	manifest, err := ParseExtensionManifest(manifestData)
	if err != nil {
		con.PrintErrorf("Error parsing %s: %s", ManifestFileName, err)
		return
	}
	installPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(manifest.CommandName))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		con.PrintInfof("Extension '%s' already exists", manifest.CommandName)
		confirm := false
		prompt := &survey.Confirm{Message: "Overwrite current install?"}
		survey.AskOne(prompt, &confirm)
		if !confirm {
			return
		}
		forceRemoveAll(installPath)
	}

	con.PrintInfof("Installing extension '%s' (%s) ... ", manifest.CommandName, manifest.Version)
	err = os.MkdirAll(installPath, 0o700)
	if err != nil {
		con.PrintErrorf("\nError creating extension directory: %s\n", err)
		return
	}
	err = ioutil.WriteFile(filepath.Join(installPath, ManifestFileName), manifestData, 0o600)
	if err != nil {
		con.PrintErrorf("\nFailed to write %s: %s\n", ManifestFileName, err)
		forceRemoveAll(installPath)
		return
	}

	for _, manifestFile := range manifest.Files {
		if manifestFile.Path != "" {
			src := filepath.Join(extLocalPath, util.ResolvePath(manifestFile.Path))
			dst := filepath.Join(installPath, util.ResolvePath(manifestFile.Path))
			err := util.CopyFile(src, dst)
			if err != nil {
				con.PrintErrorf("\nError copying file '%s' -> '%s': %s\n", src, dst, err)
				forceRemoveAll(installPath)
				return
			}
		}
	}

}

// InstallFromFilePath - Install an extension from a .tar.gz file
func InstallFromFilePath(extLocalPath string, autoOverwrite bool, con *console.SliverConsoleClient) *string {
	manifestData, err := util.ReadFileFromTarGz(extLocalPath, fmt.Sprintf("./%s", ManifestFileName))
	if err != nil {
		con.PrintErrorf("Failed to read %s from '%s': %s\n", ManifestFileName, extLocalPath, err)
		return nil
	}
	manifest, err := ParseExtensionManifest(manifestData)
	if err != nil {
		con.PrintErrorf("Failed to parse %s: %s\n", ManifestFileName, err)
		return nil
	}
	installPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(manifest.CommandName))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		if !autoOverwrite {
			con.PrintInfof("Extension '%s' already exists\n", manifest.CommandName)
			confirm := false
			prompt := &survey.Confirm{Message: "Overwrite current install?"}
			survey.AskOne(prompt, &confirm)
			if !confirm {
				return nil
			}
		}
		forceRemoveAll(installPath)
	}

	con.PrintInfof("Installing extension '%s' (%s) ... ", manifest.CommandName, manifest.Version)
	err = os.MkdirAll(installPath, 0o700)
	if err != nil {
		con.PrintErrorf("\nFailed to create extension directory: %s\n", err)
		return nil
	}
	err = ioutil.WriteFile(filepath.Join(installPath, ManifestFileName), manifestData, 0o600)
	if err != nil {
		con.PrintErrorf("\nFailed to write %s: %s\n", ManifestFileName, err)
		forceRemoveAll(installPath)
		return nil
	}
	for _, manifestFile := range manifest.Files {
		if manifestFile.Path != "" {
			err = installArtifact(extLocalPath, installPath, manifestFile.Path, con)
			if err != nil {
				con.PrintErrorf("\nFailed to install file: %s\n", err)
				forceRemoveAll(installPath)
				return nil
			}
		}
	}
	con.Printf("done!\n")
	return &installPath
}

func installArtifact(extGzFilePath string, installPath string, artifactPath string, con *console.SliverConsoleClient) error {
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
