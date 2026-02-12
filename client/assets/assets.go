package assets

/*
	Sliver Implant Framework
	Sliver implant 框架
	Copyright (C) 2019  Bishop Fox
	版权所有 (C) 2019 Bishop Fox

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
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	ver "github.com/bishopfox/sliver/client/version"
)

const (
	// SliverClientDirName - Directory storing all of the client configs/logs
	// SliverClientDirName - 存储所有 client 配置/日志的目录
	SliverClientDirName = ".sliver-client"

	versionFileName = "version"
	envVarName      = "SLIVER_CLIENT_ROOT_DIR"
)

// GetRootAppDir - Get the Sliver app dir ~/.sliver-client/
// GetRootAppDir - 获取 Sliver 应用目录 ~/.sliver-client/
func GetRootAppDir() string {
	value := os.Getenv(envVarName)
	var dir string
	if len(value) == 0 {
		user, _ := user.Current()
		dir = filepath.Join(user.HomeDir, SliverClientDirName)
	} else {
		dir = value
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

// GetClientLogsDir - Get the Sliver client logs dir ~/.sliver-client/logs/
// GetClientLogsDir - 获取 Sliver client 日志目录 ~/.sliver-client/logs/
func GetClientLogsDir() string {
	logsDir := filepath.Join(GetRootAppDir(), "logs")
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		err = os.MkdirAll(logsDir, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return logsDir
}

// GetConsoleLogsDir - Get the Sliver client console logs dir ~/.sliver-client/logs/console/
// GetConsoleLogsDir - 获取 Sliver client console 日志目录 ~/.sliver-client/logs/console/
func GetConsoleLogsDir() string {
	consoleLogsDir := filepath.Join(GetClientLogsDir(), "console")
	if _, err := os.Stat(consoleLogsDir); os.IsNotExist(err) {
		err = os.MkdirAll(consoleLogsDir, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return consoleLogsDir
}

// GetMCPLogsDir - Get the Sliver client MCP logs dir ~/.sliver-client/logs/mcp/
// GetMCPLogsDir - 获取 Sliver client MCP 日志目录 ~/.sliver-client/logs/mcp/
func GetMCPLogsDir() string {
	mcpLogsDir := filepath.Join(GetClientLogsDir(), "mcp")
	if _, err := os.Stat(mcpLogsDir); os.IsNotExist(err) {
		err = os.MkdirAll(mcpLogsDir, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return mcpLogsDir
}

func assetVersion() string {
	appDir := GetRootAppDir()
	data, err := os.ReadFile(filepath.Join(appDir, versionFileName))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func saveAssetVersion(appDir string) {
	versionFilePath := filepath.Join(appDir, versionFileName)
	fVer, _ := os.Create(versionFilePath)
	defer fVer.Close()
	fVer.Write([]byte(ver.GitCommit))
}

// Setup - Extract or create local assets
// Setup - 解压或创建本地 assets
func Setup(force bool, echo bool) {
	appDir := GetRootAppDir()
	localVer := assetVersion()
	if force || localVer == "" || localVer != ver.GitCommit {
		if echo {
			fmt.Printf(`
Sliver  Copyright (C) 2026  Bishop Fox
This program comes with ABSOLUTELY NO WARRANTY; for details type 'licenses'.
This is free software, and you are welcome to redistribute it
under certain conditions; type 'licenses' for details.`)
			fmt.Printf("\n\nUnpacking assets ...\n")
		}
		saveAssetVersion(appDir)
	}
	_, _ = LoadSettings()
}
