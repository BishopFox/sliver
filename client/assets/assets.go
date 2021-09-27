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
	"embed"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	ver "github.com/bishopfox/sliver/client/version"
)

var (
	//go:embed fs/extensions/*
	assetsFs embed.FS
)

const (
	// SliverClientDirName - Directory storing all of the client configs/logs
	SliverClientDirName = ".sliver-client"
	// SliverExtensionsDirName - Directory storing the client side extensions
	SliverExtensionsDirName = "extensions"
	versionFileName         = "version"
)

// GetRootAppDir - Get the Sliver app dir ~/.sliver-client/
func GetRootAppDir() string {
	user, _ := user.Current()
	dir := path.Join(user.HomeDir, SliverClientDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

// GetExtensionsDir - Get the Sliver extension directory: ~/.sliver-client/extensions
func GetExtensionsDir() string {
	user, _ := user.Current()
	dir := path.Join(user.HomeDir, SliverClientDirName, SliverExtensionsDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

func assetVersion() string {
	appDir := GetRootAppDir()
	data, err := ioutil.ReadFile(path.Join(appDir, versionFileName))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func saveAssetVersion(appDir string) {
	versionFilePath := path.Join(appDir, versionFileName)
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
			fmt.Printf("Unpacking assets ...\n")
		}
		err := setupCoffLoaderExt(appDir)
		if err != nil {
			fmt.Println(err)
			log.Fatal(err)
		}
		saveAssetVersion(appDir)
	}
	if _, err := os.Stat(filepath.Join(appDir, settingsFileName)); os.IsNotExist(err) {
		SaveSettings(nil)
	}
}

func setupCoffLoaderExt(appDir string) error {
	extDir := GetExtensionsDir()
	win32ExtDir := path.Join("windows", "386")
	win64ExtDir := path.Join("windows", "amd64")
	coffLoader32 := path.Join("fs", SliverExtensionsDirName, win32ExtDir, "COFFLoader.x86.dll")
	coffLoader64 := path.Join("fs", SliverExtensionsDirName, win64ExtDir, "COFFLoader.x64.dll")
	manifestPath := path.Join("fs", SliverExtensionsDirName, "manifest.json")
	loader64, err := assetsFs.ReadFile(coffLoader64)
	if err != nil {
		return err
	}
	loader32, err := assetsFs.ReadFile(coffLoader32)
	if err != nil {
		return err
	}
	manifest, err := assetsFs.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	localWin32ExtDir := path.Join(extDir, win32ExtDir)
	err = os.MkdirAll(localWin32ExtDir, 0700)
	if err != nil {
		return err
	}
	localWin64ExtDir := path.Join(extDir, win64ExtDir)
	err = os.MkdirAll(localWin64ExtDir, 0700)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(localWin32ExtDir, "COFFLoader.x86.dll"), loader32, 0744)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(extDir, "manifest.json"), manifest, 0700)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(localWin64ExtDir, "COFFLoader.x64.dll"), loader64, 0744)
}
