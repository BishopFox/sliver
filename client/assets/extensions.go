package assets

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
	"log"
	"os"
	"path/filepath"
)

const (
	// ExtensionsDirName - Directory storing the client side extensions
	// ExtensionsDirName - 存储 client 侧 extension 的目录
	ExtensionsDirName = "extensions"
)

// GetExtensionsDir - Get the Sliver extension directory: ~/.sliver-client/extensions
// GetExtensionsDir - 获取 Sliver extension 目录：~/.sliver-client/extensions
func GetExtensionsDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := filepath.Join(rootDir, ExtensionsDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

// GetInstalledExtensionManifests - Returns a list of installed extension manifests
// GetInstalledExtensionManifests - 返回已安装 extension manifest 列表
func GetInstalledExtensionManifests() []string {
	extDir := GetExtensionsDir()
	extDirContent, err := os.ReadDir(extDir)
	if err != nil {
		log.Printf("error loading aliases: %s", err)
		return []string{}
	}
	manifests := []string{}
	for _, fi := range extDirContent {
		if fi.IsDir() {
			manifestPath := filepath.Join(extDir, fi.Name(), "extension.json")
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				log.Printf("no manifest in %s, skipping ...", manifestPath)
				continue
			}
			manifests = append(manifests, manifestPath)
		}
	}
	return manifests
}
