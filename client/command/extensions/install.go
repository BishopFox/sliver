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
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/util"
	"github.com/spf13/cobra"
)

// ExtensionsInstallCmd - Install an extension.
func ExtensionsInstallCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	extLocalPath := args[0]

	_, err := os.Stat(extLocalPath)
	if os.IsNotExist(err) {
		con.PrintErrorf("Extension path '%s' does not exist", extLocalPath)
		return
	}
	InstallFromDir(extLocalPath, true, con, strings.HasSuffix(extLocalPath, ".tar.gz"))
}

// Install an extension from a directory
func InstallFromDir(extLocalPath string, promptToOverwrite bool, con *console.SliverClient, isGz bool) {
	var manifestData []byte
	var err error

	if isGz {
		manifestData, err = util.ReadFileFromTarGz(extLocalPath, fmt.Sprintf("./%s", ManifestFileName))
	} else {
		manifestData, err = os.ReadFile(filepath.Join(extLocalPath, ManifestFileName))
	}
	if err != nil {
		con.PrintErrorf("Error reading %s: %s", ManifestFileName, err)
		return
	}

	manifestF, err := ParseExtensionManifest(manifestData)
	if err != nil {
		con.PrintErrorf("Error parsing %s: %s", ManifestFileName, err)
		return
	}

	//create repo path
	minstallPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(manifestF.Name))
	if _, err := os.Stat(minstallPath); !os.IsNotExist(err) {
		if promptToOverwrite {
			con.PrintInfof("Extension '%s' already exists", manifestF.Name)
			confirm := false
			prompt := &survey.Confirm{Message: "Overwrite current install?"}
			survey.AskOne(prompt, &confirm)
			if !confirm {
				return
			}
		}
		forceRemoveAll(minstallPath)
	}
	con.PrintInfof("Installing extension '%s' (%s) ... \n", manifestF.Name, manifestF.Version)
	err = os.MkdirAll(minstallPath, 0o700)
	if err != nil {
		con.PrintErrorf("Error creating extension directory: %s\n", err)
		return
	}
	err = os.WriteFile(filepath.Join(minstallPath, ManifestFileName), manifestData, 0o600)
	if err != nil {
		con.PrintErrorf("Failed to write %s: %s\n", ManifestFileName, err)
		forceRemoveAll(minstallPath)
		return
	}

	for _, manifest := range manifestF.ExtCommand {
		installPath := filepath.Join(minstallPath)
		for _, manifestFile := range manifest.Files {
			if manifestFile.Path != "" {
				if isGz {
					err = installArtifact(extLocalPath, installPath, manifestFile.Path)
				} else {
					src := filepath.Join(extLocalPath, util.ResolvePath(manifestFile.Path))
					dst := filepath.Join(installPath, util.ResolvePath(manifestFile.Path))
					err = os.MkdirAll(filepath.Dir(dst), 0o700) //required for extensions with multiple dirs between the .o file and the manifest
					if err != nil {
						con.PrintErrorf("\nError creating extension directory: %s\n", err)
						forceRemoveAll(installPath)
						return
					}
					err = util.CopyFile(src, dst)
					if err != nil {
						err = fmt.Errorf("error copying file '%s' -> '%s': %s", src, dst, err)
					}
				}
				if err != nil {
					con.PrintErrorf("Error installing command: %s\n", err)
					forceRemoveAll(installPath)
					return
				}
			}
		}
	}
}

func installArtifact(extGzFilePath string, installPath string, artifactPath string) error {
	data, err := util.ReadFileFromTarGz(extGzFilePath, "."+filepath.ToSlash(artifactPath))
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
	err = os.WriteFile(localArtifactPath, data, 0o600)
	if err != nil {
		return err
	}
	return nil
}
