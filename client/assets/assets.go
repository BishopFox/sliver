package assets

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
	SliverClientDirName = ".sliver-client"

	versionFileName = "version"
)

// GetRootAppDir - Get the Sliver app dir ~/.sliver-client/
func GetRootAppDir() string {
	user, _ := user.Current()
	dir := filepath.Join(user.HomeDir, SliverClientDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

// GetClientLogsDir - Get the Sliver client logs dir ~/.sliver-client/logs/
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
func Setup(force bool, echo bool) {
	appDir := GetRootAppDir()
	localVer := assetVersion()
	if force || localVer == "" || localVer != ver.GitCommit {
		if echo {
			fmt.Printf(`
Sliver  Copyright (C) 2025  Bishop Fox
This program comes with ABSOLUTELY NO WARRANTY; for details type 'licenses'.
This is free software, and you are welcome to redistribute it
under certain conditions; type 'licenses' for details.`)
			fmt.Printf("\n\nUnpacking assets ...\n")
		}
		saveAssetVersion(appDir)
	}
	if _, err := os.Stat(filepath.Join(appDir, settingsFileName)); os.IsNotExist(err) {
		SaveSettings(nil)
	}
}
