package alias

/*
	Sliver Implant Framework
	Sliver implant 框架
	Copyright (C) 2021  Bishop Fox
	版权所有 (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	本程序是自由软件：你可以再发布和/或修改它
	it under the terms of the GNU General Public License as published by
	在自由软件基金会发布的 GNU General Public License 条款下，
	the Free Software Foundation, either version 3 of the License, or
	可以使用许可证第 3 版，或
	(at your option) any later version.
	（由你选择）任何更高版本。

	This program is distributed in the hope that it will be useful,
	发布本程序是希望它能发挥作用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但不提供任何担保；甚至不包括
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	对适销性或特定用途适用性的默示担保。请参阅
	GNU General Public License for more details.
	GNU General Public License 以获取更多细节。

	You should have received a copy of the GNU General Public License
	你应当已随本程序收到一份 GNU General Public License 副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	如果没有，请参见 <https://www.gnu.org/licenses/>。
*/

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/util"
	"github.com/spf13/cobra"
)

// AliasesInstallCmd - Install an alias.
// AliasesInstallCmd - 安装 alias。
func AliasesInstallCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	aliasLocalPath := args[0]
	fi, err := os.Stat(aliasLocalPath)
	if os.IsNotExist(err) {
		con.PrintErrorf("Alias path '%s' does not exist", aliasLocalPath)
		return
	}
	if !fi.IsDir() {
		InstallFromFile(aliasLocalPath, "", false, con)
	} else {
		installFromDir(aliasLocalPath, con)
	}
}

// Install an extension from a directory.
// 从目录安装 extension。
func installFromDir(aliasLocalPath string, con *console.SliverClient) {
	manifestData, err := os.ReadFile(filepath.Join(aliasLocalPath, ManifestFileName))
	if err != nil {
		con.PrintErrorf("Error reading %s: %s", ManifestFileName, err)
		return
	}
	manifest, err := ParseAliasManifest(manifestData)
	if err != nil {
		con.PrintErrorf("Error parsing %s: %s", ManifestFileName, err)
		return
	}
	installPath := filepath.Join(assets.GetAliasesDir(), filepath.Base(manifest.CommandName))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		con.PrintInfof("Alias '%s' already exists", manifest.CommandName)
		confirm := false
		_ = forms.Confirm("Overwrite current install?", &confirm)
		if !confirm {
			return
		}
		forceRemoveAll(installPath)
	}

	con.PrintInfof("Installing alias '%s' (%s) ... ", manifest.Name, manifest.Version)
	err = os.MkdirAll(installPath, 0o700)
	if err != nil {
		con.PrintErrorf("Error creating alias directory: %s\n", err)
		return
	}
	err = os.WriteFile(filepath.Join(installPath, ManifestFileName), manifestData, 0o600)
	if err != nil {
		con.PrintErrorf("Failed to write %s: %s\n", ManifestFileName, err)
		forceRemoveAll(installPath)
		return
	}

	for _, cmdFile := range manifest.Files {
		if cmdFile.Path != "" {
			src := filepath.Join(aliasLocalPath, util.ResolvePath(cmdFile.Path))
			dst := filepath.Join(installPath, util.ResolvePath(cmdFile.Path))
			err := util.CopyFile(src, dst)
			if err != nil {
				con.PrintErrorf("Error copying file '%s' -> '%s': %s\n", src, dst, err)
				forceRemoveAll(installPath)
				return
			}
		}
	}

	con.Printf("done!\n")
}

// Install an extension from a .tar.gz file.
// 从 .tar.gz 文件安装 extension。
func InstallFromFile(aliasGzFilePath string, aliasName string, promptToOverwrite bool, con *console.SliverClient) *string {
	manifestData, err := util.ReadFileFromTarGz(aliasGzFilePath, fmt.Sprintf("./%s", ManifestFileName))
	if err != nil {
		con.PrintErrorf("Failed to read %s from '%s': %s\n", ManifestFileName, aliasGzFilePath, err)
		return nil
	}
	manifest, err := ParseAliasManifest(manifestData)
	if err != nil {
		errorMsg := ""
		if aliasName != "" {
			errorMsg = fmt.Sprintf("Error processing manifest for alias %s - failed to parse %s: %s\n", aliasName, ManifestFileName, err)
		} else {
			errorMsg = fmt.Sprintf("Failed to parse %s: %s\n", ManifestFileName, err)
		}
		con.PrintErrorf("%s", errorMsg)
		return nil
	}
	installPath := filepath.Join(assets.GetAliasesDir(), filepath.Base(manifest.CommandName))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		if promptToOverwrite {
			con.PrintInfof("Alias '%s' already exists\n", manifest.CommandName)
			confirm := false
			_ = forms.Confirm("Overwrite current install?", &confirm)
			if !confirm {
				return nil
			}
		}
		forceRemoveAll(installPath)
	}

	con.PrintInfof("Installing alias '%s' (%s) ... ", manifest.Name, manifest.Version)
	err = os.MkdirAll(installPath, 0o700)
	if err != nil {
		con.PrintErrorf("Failed to create alias directory: %s\n", err)
		return nil
	}
	err = os.WriteFile(filepath.Join(installPath, ManifestFileName), manifestData, 0o600)
	if err != nil {
		con.PrintErrorf("Failed to write %s: %s\n", ManifestFileName, err)
		forceRemoveAll(installPath)
		return nil
	}
	for _, aliasFile := range manifest.Files {
		if aliasFile.Path != "" {
			err := installArtifact(aliasGzFilePath, installPath, aliasFile.Path)
			if err != nil {
				con.PrintErrorf("Failed to install file: %s\n", err)
				forceRemoveAll(installPath)
				return nil
			}
		}
	}
	con.Printf("done!\n")
	return &installPath
}

func installArtifact(aliasGzFilePath string, installPath, artifactPath string) error {
	data, err := util.ReadFileFromTarGz(aliasGzFilePath, fmt.Sprintf("./%s", strings.TrimPrefix(artifactPath, string(os.PathSeparator))))
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("empty file %s", artifactPath)
	}
	localArtifactPath := filepath.Join(installPath, util.ResolvePath(artifactPath))
	artifactDir := filepath.Dir(localArtifactPath)
	if _, err := os.Stat(artifactDir); os.IsNotExist(err) {
		os.MkdirAll(artifactDir, 0o700)
	}
	err = os.WriteFile(localArtifactPath, data, 0o600)
	if err != nil {
		return err
	}
	return nil
}

func forceRemoveAll(rootPath string) {
	util.ChmodR(rootPath, 0o600, 0o700)
	os.RemoveAll(rootPath)
}
