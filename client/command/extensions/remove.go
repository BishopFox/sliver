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
	"errors"
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

// ExtensionsRemoveCmd - Remove an extension.
// ExtensionsRemoveCmd - Remove 和 extension.
func ExtensionsRemoveCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	name := args[0]
	if name == "" {
		con.PrintErrorf("Extension name is required\n")
		return
	}
	confirm := false
	_ = forms.Confirm(fmt.Sprintf("Remove '%s' extension?", name), &confirm)
	if !confirm {
		return
	}
	found, err := RemoveExtensionByManifestName(name, con)
	if err != nil {
		con.PrintErrorf("Error removing extensions: %s\n", err)
		return
	}
	if !found {
		err = RemoveExtensionByCommandName(name, con)
		if err != nil {
			con.PrintErrorf("Error removing extension: %s\n", err)
			return
		} else {
			con.PrintInfof("Extension '%s' removed\n", name)
		}
	} else {
		//found, and no error, manifest must have removed good
		//找到了，也没有错误，manifest一定已经去掉好了
		con.PrintInfof("Extensions from %s removed\n", name)
	}
}

// RemoveExtensionByCommandName - Remove an extension by command name.
// RemoveExtensionByCommandName - Remove 命令的扩展 name.
func RemoveExtensionByCommandName(commandName string, con *console.SliverClient) error {
	if commandName == "" {
		return errors.New("command name is required")
	}
	if _, ok := loadedExtensions[commandName]; !ok {
		return errors.New("extension not loaded")
	}
	delete(loadedExtensions, commandName)
	extPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(commandName))
	if _, err := os.Stat(extPath); os.IsNotExist(err) {
		return nil
	}
	forceRemoveAll(extPath)
	return nil
}

// RemoveExtensionByManifestName - remove by the named manifest, returns true if manifest was removed, false if no manifest with that name was found
// RemoveExtensionByManifestName - 按指定清单删除，如果清单已删除则返回 true，如果未找到具有该名称的清单则返回 false
func RemoveExtensionByManifestName(manifestName string, con *console.SliverClient) (bool, error) {
	if manifestName == "" {
		return false, errors.New("command name is required")
	}
	if man, ok := loadedManifests[manifestName]; ok {
		// Found the manifest
		// Found 清单
		var extPath string
		if man.RootPath != "" {
			// Use RootPath for temporarily loaded extensions
			// Use RootPath 用于临时加载的扩展
			extPath = man.RootPath
		} else {
			// Fall back to extensions dir for installed extensions
			// Fall 返回已安装扩展的扩展目录
			extPath = filepath.Join(assets.GetExtensionsDir(), filepath.Base(manifestName))
		}

		if _, err := os.Stat(extPath); os.IsNotExist(err) {
			return true, nil
		}

		// If path is outside extensions directory, prompt for confirmation
		// If 路径在扩展目录之外，提示确认
		if !strings.HasPrefix(extPath, assets.GetExtensionsDir()) {
			confirm := false
			_ = forms.Confirm(fmt.Sprintf("Remove '%s' extension directory from filesystem?", manifestName), &confirm)
			if !confirm {
				// Skip the forceRemoveAll but continue with the rest
				// Skip forceRemoveAll 但继续其余部分
				delete(loadedManifests, manifestName)
				for _, cmd := range man.ExtCommand {
					delete(loadedExtensions, cmd.CommandName)
				}
				return true, nil
			}
		}

		forceRemoveAll(extPath)
		delete(loadedManifests, manifestName)
		for _, cmd := range man.ExtCommand {
			delete(loadedExtensions, cmd.CommandName)
		}
		return true, nil
	}
	return false, nil
}

func forceRemoveAll(rootPath string) {
	util.ChmodR(rootPath, 0o600, 0o700)
	os.RemoveAll(rootPath)
}
