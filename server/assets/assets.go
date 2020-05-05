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
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	ver "github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/server/log"

	"github.com/gobuffalo/packr"
)

const (
	// GoDirName - The directory to store the go compiler/toolchain files in
	GoDirName       = "go"
	goPathDirName   = "gopath"
	versionFileName = "version"
	dataDirName     = "data"
	envVarName      = "SLIVER_ROOT_DIR"
)

var (
	setupLog = log.NamedLogger("assets", "setup")

	assetsBox   = packr.NewBox("../../assets")
	protobufBox = packr.NewBox("../../protobuf")
)

// GetRootAppDir - Get the Sliver app dir, default is: ~/.sliver/
func GetRootAppDir() string {

	value := os.Getenv(envVarName)

	var dir string
	if len(value) == 0 {
		user, _ := user.Current()
		dir = path.Join(user.HomeDir, ".sliver")
	} else {
		dir = value
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			setupLog.Fatalf("Cannot write to sliver root dir %s", err)
		}
	}
	return dir
}

// GetDataDir - Returns the full path to the data directory
func GetDataDir() string {
	dir := path.Join(GetRootAppDir(), dataDirName)
	return dir
}

func assetVersion() string {
	appDir := GetRootAppDir()
	data, err := ioutil.ReadFile(path.Join(appDir, versionFileName))
	if err != nil {
		setupLog.Infof("No version detected %s", err)
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
func Setup(force bool) {
	appDir := GetRootAppDir()
	localVer := assetVersion()
	if force || localVer == "" || localVer != ver.GitCommit {
		setupLog.Infof("Version mismatch %v != %v", localVer, ver.GitCommit)
		fmt.Printf("Unpacking assets ...\n")
		setupGo(appDir)
		setupCodenames(appDir)
		setupDataPath(appDir)
		saveAssetVersion(appDir)
	}
}

// English - Extracts the english dictionary for the english encoder
func English() []string {
	rawEnglish, err := assetsBox.Find("english.txt")
	if err != nil {
		return []string{}
	}
	englishWords := strings.Split(string(rawEnglish), "\n")
	return englishWords
}

// SetupGo - Unzip Go compiler assets
func setupGo(appDir string) error {

	setupLog.Infof("Unpacking to '%s'", appDir)
	goRootPath := path.Join(appDir, GoDirName)
	setupLog.Infof("GOPATH = %s", goRootPath)
	if _, err := os.Stat(goRootPath); !os.IsNotExist(err) {
		setupLog.Info("Removing old go root directory")
		os.RemoveAll(goRootPath)
	}
	os.MkdirAll(goRootPath, 0700)

	// Go compiler and stdlib
	goZip, err := assetsBox.Find(path.Join(runtime.GOOS, "go.zip"))
	if err != nil {
		setupLog.Info("static asset not found: go.zip")
		return err
	}

	goZipPath := path.Join(appDir, "go.zip")
	defer os.Remove(goZipPath)
	ioutil.WriteFile(goZipPath, goZip, 0644)
	_, err = unzip(goZipPath, appDir)
	if err != nil {
		setupLog.Infof("Failed to unzip file %s -> %s", goZipPath, appDir)
		return err
	}

	goSrcZip, err := assetsBox.Find("src.zip")
	if err != nil {
		setupLog.Info("static asset not found: src.zip")
		return err
	}
	goSrcZipPath := path.Join(appDir, "src.zip")
	defer os.Remove(goSrcZipPath)
	ioutil.WriteFile(goSrcZipPath, goSrcZip, 0644)
	_, err = unzip(goSrcZipPath, goRootPath)
	if err != nil {
		setupLog.Infof("Failed to unzip file %s -> %s/go", goSrcZipPath, appDir)
		return err
	}

	return nil
}

// SetupGoPath - Extracts dependencies to goPathSrc
func SetupGoPath(goPathSrc string) error {

	// GOPATH setup
	if _, err := os.Stat(goPathSrc); os.IsNotExist(err) {
		setupLog.Infof("Creating GOPATH directory: %s", goPathSrc)
		os.MkdirAll(goPathSrc, 0700)
	}

	// Sliver PB
	sliverpbGoSrc, err := protobufBox.Find("sliverpb/sliver.pb.go")
	if err != nil {
		setupLog.Info("static asset not found: sliver.pb.go")
		return err
	}
	sliverpbConstSrc, err := protobufBox.Find("sliverpb/constants.go")
	if err != nil {
		setupLog.Info("static asset not found: constants.go")
		return err
	}
	sliverpbDir := path.Join(goPathSrc, "github.com", "bishopfox", "sliver", "protobuf", "sliverpb")
	os.MkdirAll(sliverpbDir, 0700)
	ioutil.WriteFile(path.Join(sliverpbDir, "constants.go"), sliverpbGoSrc, 0644)
	ioutil.WriteFile(path.Join(sliverpbDir, "sliver.pb.go"), sliverpbConstSrc, 0644)

	// Common PB
	commonpbSrc, err := protobufBox.Find("commonpb/common.pb.go")
	if err != nil {
		setupLog.Info("static asset not found: common.pb.go")
		return err
	}
	commonpbDir := path.Join(goPathSrc, "github.com", "bishopfox", "sliver", "protobuf", "commonpb")
	os.MkdirAll(commonpbDir, 0700)
	ioutil.WriteFile(path.Join(commonpbDir, "common.pb.go"), commonpbSrc, 0644)

	// GOPATH 3rd party dependencies
	protobufPath := path.Join(goPathSrc, "github.com", "golang")
	err = unzipGoDependency("protobuf.zip", protobufPath, assetsBox)
	if err != nil {
		setupLog.Fatalf("Failed to unzip go dependency: %v", err)
	}
	golangXPath := path.Join(goPathSrc, "golang.org", "x")
	err = unzipGoDependency("golang_x_sys.zip", golangXPath, assetsBox)
	if err != nil {
		setupLog.Fatalf("Failed to unzip go dependency: %v", err)
	}

	return nil
}

// setupDataPath - Sets the data directory up
func setupDataPath(appDir string) error {
	dataDir := path.Join(appDir, dataDirName)
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		setupLog.Infof("Creating data directory: %s", dataDir)
		os.MkdirAll(dataDir, 0700)
	}
	hostingDll, err := assetsBox.Find("dll/HostingCLRx64.dll")
	if err != nil {
		setupLog.Info("failed to find the dll")
		return err
	}
	err = ioutil.WriteFile(dataDir+"/HostingCLRx64.dll", hostingDll, 0644)
	return err
}

func unzipGoDependency(fileName string, targetPath string, assetsBox packr.Box) error {
	setupLog.Infof("Unpacking go dependency %s -> %s", fileName, targetPath)

	appDir := GetRootAppDir()
	goDep, err := assetsBox.Find(fileName)
	if err != nil {
		setupLog.Infof("static asset not found: %s", fileName)
		return err
	}

	goDepZipPath := path.Join(appDir, fileName)
	defer os.Remove(goDepZipPath)
	ioutil.WriteFile(goDepZipPath, goDep, 0644)
	_, err = unzip(goDepZipPath, targetPath)
	if err != nil {
		setupLog.Infof("Failed to unzip file %s -> %s", goDepZipPath, appDir)
		return err
	}

	return nil
}

func setupCodenames(appDir string) error {
	nouns, err := assetsBox.Find("nouns.txt")
	adjectives, err := assetsBox.Find("adjectives.txt")

	err = ioutil.WriteFile(path.Join(appDir, "nouns.txt"), nouns, 0600)
	if err != nil {
		setupLog.Infof("Failed to write noun data to: %s", appDir)
		return err
	}

	err = ioutil.WriteFile(path.Join(appDir, "adjectives.txt"), adjectives, 0600)
	if err != nil {
		setupLog.Infof("Failed to write adjective data to: %s", appDir)
		return err
	}
	return nil
}

func unzip(src string, dest string) ([]string, error) {

	var filenames []string

	reader, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer reader.Close()

	for _, file := range reader.File {

		rc, err := file.Open()
		if err != nil {
			return filenames, err
		}
		defer rc.Close()

		fPath := filepath.Join(dest, file.Name)
		filenames = append(filenames, fPath)

		if file.FileInfo().IsDir() {
			os.MkdirAll(fPath, 0700)
		} else {
			if err = os.MkdirAll(filepath.Dir(fPath), 0700); err != nil {
				return filenames, err
			}
			outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
			if err != nil {
				return filenames, err
			}
			_, err = io.Copy(outFile, rc)
			outFile.Close()
			if err != nil {
				return filenames, err
			}
		}
	}
	return filenames, nil
}
