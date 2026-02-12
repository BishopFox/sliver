package extensions

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
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

// ExtensionsInstallCmd - Install an extension.
// ExtensionsInstallCmd - Install 和 extension.
func ExtensionsInstallCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	extLocalPath := args[0]

	_, err := os.Stat(extLocalPath)
	if os.IsNotExist(err) {
		con.PrintErrorf("Extension path '%s' does not exist", extLocalPath)
		return
	}
	InstallFromDir(extLocalPath, true, con, strings.HasSuffix(extLocalPath, ".tar.gz"))
}

// InstallFromDir installs a Sliver extension from either a local directory or gzipped archive.
// InstallFromDir 从本地目录或 gzipped archive. 安装 Sliver 扩展
// It reads the extension manifest, validates it, and copies all required files to the extensions
// It 读取扩展清单，验证它，并将所有必需的文件复制到扩展中
// directory. If an extension with the same name already exists, it can optionally prompt for
// directory. If 已存在同名扩展，可以选择提示
// overwrite confirmation.
// 覆盖 confirmation.
//
// Parameters:
//   - extLocalPath: Path to the source directory or gzipped archive containing the extension
//   - extLocalPath: Path 到源目录或包含扩展名的 gzip 压缩档案
//   - promptToOverwrite: If true, prompts for confirmation before overwriting existing extension
//   - promptToOverwrite: If true，覆盖现有扩展之前提示确认
//   - con: Sliver console client for displaying status and error messages
//   - con: Sliver 控制台客户端，用于显示状态和错误消息
//   - isGz: Whether the source is a gzipped archive (true) or directory (false)
//   - isGz: Whether 源是 gzip 压缩档案 (true) 或目录 (false)
//
// The function will return early with error messages printed to console if:
// 如果出现以下情况，The 函数将提前返回，并将错误消息打印到控制台：
//   - The manifest cannot be read or parsed
//   - The 清单无法读取或解析
//   - Required directories cannot be created
//   - 无法创建 Required 目录
//   - File copy operations fail
//   - File 复制操作失败
//   - User declines overwrite when prompted
//   - User 在出现提示时拒绝覆盖
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

	// Use package name if available, otherwise use extension name
	// Use 包名称（如果可用），否则使用扩展名
	// (Note, for v1 manifests this will actually be command_name)
	// （Note，对于 v1 清单，这实际上是 command_name）
	packageID := manifestF.Name
	if manifestF.PackageName != "" {
		packageID = manifestF.PackageName
	}

	//create repo path
	//创建仓库路径
	minstallPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(packageID))
	if _, err := os.Stat(minstallPath); !os.IsNotExist(err) {
		if promptToOverwrite {
			con.PrintInfof("Extension '%s' already exists", manifestF.Name)
			confirm := false
			_ = forms.Confirm("Overwrite current install?", &confirm)
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
					err = os.MkdirAll(filepath.Dir(dst), 0o700) //.o 文件和清单之间具有多个目录的扩展名是必需的
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
