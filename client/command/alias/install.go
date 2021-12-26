package alias

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
	"path"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/util"
	"github.com/desertbit/grumble"
)

// AliasesInstallCmd - Install an alias
func AliasesInstallCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	aliasLocalPath := ctx.Args.String("path")
	fi, err := os.Stat(aliasLocalPath)
	if os.IsNotExist(err) {
		con.PrintErrorf("Extension path '%s' does not exist", aliasLocalPath)
		return
	}
	if !fi.IsDir() {
		InstallFromFile(aliasLocalPath, con)
	} else {
		installFromDir(aliasLocalPath, con)
	}
}

// Install an extension from a directory
func installFromDir(aliasLocalPath string, con *console.SliverConsoleClient) {
	manifestData, err := ioutil.ReadFile(filepath.Join(aliasLocalPath, ManifestFileName))
	if err != nil {
		con.PrintErrorf("Error reading %s: %s", ManifestFileName, err)
		return
	}
	manifest, err := parseAliasManifest(manifestData)
	if err != nil {
		con.PrintErrorf("Error parsing %s: %s", ManifestFileName, err)
		return
	}
	installPath := filepath.Join(assets.GetAliasesDir(), filepath.Base(manifest.Name))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		con.PrintInfof("Extension '%s' already exists", manifest.Name)
		confirm := false
		prompt := &survey.Confirm{Message: "Overwrite current install?"}
		survey.AskOne(prompt, &confirm)
		if !confirm {
			return
		}
		os.RemoveAll(installPath)
	}

	con.Printf("Installing extension '%s' (v%s) ... ", manifest.Name, manifest.Version)
	err = os.MkdirAll(installPath, 0o700)
	if err != nil {
		con.PrintErrorf("\nError creating extension directory: %s\n", err)
		return
	}
	err = ioutil.WriteFile(filepath.Join(installPath, ManifestFileName), manifestData, 0o600)
	if err != nil {
		con.PrintErrorf("\nFailed to write %s: %s\n", ManifestFileName, err)
		os.RemoveAll(installPath)
		return
	}

	for _, cmdFile := range manifest.Files {
		if cmdFile.Path != "" {
			src := filepath.Join(aliasLocalPath, path.Clean("/"+cmdFile.Path))
			dst := filepath.Join(installPath, path.Clean("/"+cmdFile.Path))
			err := util.CopyFile(src, dst)
			if err != nil {
				con.PrintErrorf("\nError copying file '%s' -> '%s': %s\n", src, dst, err)
				os.RemoveAll(installPath)
				return
			}
		}
	}

	con.Printf("done!\n")
}

// Install an extension from a .tar.gz file
func InstallFromFile(aliasGzFilePath string, con *console.SliverConsoleClient) {
	manifestData, err := util.ReadFileFromTarGz(aliasGzFilePath, fmt.Sprintf("./%s", ManifestFileName))
	if err != nil {
		con.PrintErrorf("Failed to read %s from '%s': %s\n", ManifestFileName, aliasGzFilePath, err)
		return
	}
	manifest, err := parseAliasManifest(manifestData)
	if err != nil {
		con.PrintErrorf("Failed to parse %s: %s\n", ManifestFileName, err)
		return
	}
	installPath := filepath.Join(assets.GetAliasesDir(), filepath.Base(manifest.Name))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		con.PrintInfof("Alias '%s' already exists\n", manifest.CommandName)
		confirm := false
		prompt := &survey.Confirm{Message: "Overwrite current install?"}
		survey.AskOne(prompt, &confirm)
		if !confirm {
			return
		}
		os.RemoveAll(installPath)
	}

	con.Printf("Installing alias '%s' (v%s) ... ", manifest.Name, manifest.Version)
	err = os.MkdirAll(installPath, 0o700)
	if err != nil {
		con.PrintErrorf("\nFailed to create alias directory: %s\n", err)
		return
	}
	err = ioutil.WriteFile(filepath.Join(installPath, ManifestFileName), manifestData, 0o600)
	if err != nil {
		con.PrintErrorf("\nFailed to write %s: %s\n", ManifestFileName, err)
		os.RemoveAll(installPath)
		return
	}
	for _, aliasFile := range manifest.Files {
		if aliasFile.Path != "" {
			err := installArtifact(aliasGzFilePath, installPath, aliasFile.Path, con)
			if err != nil {
				con.PrintErrorf("\nFailed to install file: %s\n", err)
				os.RemoveAll(installPath)
				return
			}
		}
	}
	con.Printf("done!\n")
}

func installArtifact(aliasGzFilePath string, installPath, artifactPath string, con *console.SliverConsoleClient) error {
	data, err := util.ReadFileFromTarGz(aliasGzFilePath, artifactPath)
	if err != nil {
		return err
	}
	localArtifactPath := filepath.Join(installPath, path.Clean("/"+artifactPath))
	artifactDir := filepath.Dir(localArtifactPath)
	if _, err := os.Stat(artifactDir); os.IsNotExist(err) {
		os.MkdirAll(artifactDir, 0o700)
	}
	err = ioutil.WriteFile(localArtifactPath, data, 0o600)
	if err != nil {
		return err
	}
	return nil
}
