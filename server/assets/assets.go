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
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	insecureRand "math/rand"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	ver "github.com/bishopfox/sliver/client/version"
	protobufs "github.com/bishopfox/sliver/protobuf"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/util"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

const (
	// GoDirName - The directory to store the go compiler/toolchain files in
	GoDirName = "go"

	goPathDirName   = "gopath"
	versionFileName = "version"
	envVarName      = "SLIVER_ROOT_DIR"
)

var (
	setupLog = log.NamedLogger("assets", "setup")
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
func Setup(force bool, echo bool) {
	appDir := GetRootAppDir()
	localVer := assetVersion()
	if force || localVer == "" || localVer != ver.GitCommit {
		setupLog.Infof("Version mismatch %v != %v", localVer, ver.GitCommit)
		if echo {
			fmt.Printf(`
Sliver  Copyright (C) 2022  Bishop Fox
This program comes with ABSOLUTELY NO WARRANTY; for details type 'licenses'.
This is free software, and you are welcome to redistribute it
under certain conditions; type 'licenses' for details.`)
			fmt.Printf("\n\nUnpacking assets ...\n")
		}
		setupGo(appDir)
		setupCodenames(appDir)
		saveAssetVersion(appDir)
	}
}

// English - Extracts the english dictionary for the english encoder
func English() []string {
	rawEnglish, err := assetsFs.ReadFile("fs/english.txt")
	if err != nil {
		return []string{}
	}
	englishWords := strings.Split(string(rawEnglish), "\n")
	return englishWords
}

// GetGPGPublicKey - Return the GPG public key from assets
func GetGPGPublicKey() (*packet.PublicKey, error) {
	rawPublicKey, err := assetsFs.ReadFile("fs/sliver.asc")
	if err != nil {
		return nil, err
	}
	// Decode armored public key
	block, err := armor.Decode(bytes.NewReader(rawPublicKey))
	if err != nil {
		return nil, fmt.Errorf("error decoding public key: %s", err)
	}
	if block.Type != "PGP PUBLIC KEY BLOCK" {
		return nil, errors.New("not an armored public key")
	}

	// Read the key
	pack, err := packet.Read(block.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading public key: %s", err)
	}

	// Was it really a public key file ? If yes, get the PublicKey
	publicKey, ok := pack.(*packet.PublicKey)
	if !ok {
		return nil, errors.New("invalid public key")
	}
	return publicKey, nil
}

// SetupGo - Unzip Go compiler assets
func setupGo(appDir string) error {

	setupLog.Infof("Unpacking to '%s'", appDir)
	goRootPath := path.Join(appDir, GoDirName)
	setupLog.Infof("GOPATH = %s", goRootPath)
	if _, err := os.Stat(goRootPath); !os.IsNotExist(err) {
		setupLog.Info("Removing old go root directory")
		os.Chmod(goRootPath, 0700)
		err = util.ChmodR(goRootPath, 0600, 0700) // Make sure everything is writable before we try to rm
		if err != nil {
			setupLog.Warnf("Failed to modify file system permissions of old go root directory %s", err)
		}
		err = os.RemoveAll(goRootPath)
		if err != nil {
			setupLog.Warnf("Failed to cleanup old go root directory %s", err)
		}
	}
	os.MkdirAll(goRootPath, 0700)

	// Go compiler and stdlib
	goZipFSPath := path.Join("fs", runtime.GOOS, runtime.GOARCH, "go.zip")
	goZip, err := assetsFs.ReadFile(goZipFSPath)
	if err != nil {
		setupLog.Errorf("static asset not found: %s", goZipFSPath)
		return err
	}

	goZipPath := path.Join(appDir, "go.zip")
	defer os.Remove(goZipPath)
	ioutil.WriteFile(goZipPath, goZip, 0600)
	_, err = unzip(goZipPath, appDir)
	if err != nil {
		setupLog.Infof("Failed to unzip file %s -> %s", goZipPath, appDir)
		return err
	}

	goSrcZip, err := assetsFs.ReadFile("fs/src.zip")
	if err != nil {
		setupLog.Info("static asset not found: src.zip")
		return err
	}
	goSrcZipPath := path.Join(appDir, "src.zip")
	defer os.Remove(goSrcZipPath)
	ioutil.WriteFile(goSrcZipPath, goSrcZip, 0600)
	_, err = unzip(goSrcZipPath, goRootPath)
	if err != nil {
		setupLog.Infof("Failed to unzip file %s -> %s/go", goSrcZipPath, appDir)
		return err
	}

	garbleFileName := "garble"
	if runtime.GOOS == "windows" {
		garbleFileName = "garble.exe"
	}
	garbleAssetPath := path.Join("fs", runtime.GOOS, runtime.GOARCH, garbleFileName)
	garbleFile, err := assetsFs.ReadFile(garbleAssetPath)
	if err != nil {
		setupLog.Errorf("Static asset not found: %s", garbleFile)
		return err
	}
	garbleLocalPath := path.Join(appDir, "go", "bin", garbleFileName)
	err = ioutil.WriteFile(garbleLocalPath, garbleFile, 0700)
	if err != nil {
		setupLog.Errorf("Failed to write garble %s", err)
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
	sliverpbGoSrc, err := protobufs.FS.ReadFile("sliverpb/sliver.pb.go")
	if err != nil {
		setupLog.Info("Static asset not found: sliver.pb.go")
		return err
	}
	sliverpbConstSrc, err := protobufs.FS.ReadFile("sliverpb/constants.go")
	if err != nil {
		setupLog.Info("Static asset not found: constants.go")
		return err
	}
	sliverpbGoSrc = stripSliverpb(sliverpbGoSrc)
	sliverpbDir := path.Join(goPathSrc, "github.com", "bishopfox", "sliver", "protobuf", "sliverpb")
	os.MkdirAll(sliverpbDir, 0700)
	ioutil.WriteFile(path.Join(sliverpbDir, "sliver.pb.go"), sliverpbGoSrc, 0600)
	ioutil.WriteFile(path.Join(sliverpbDir, "constants.go"), sliverpbConstSrc, 0600)

	// Common PB
	commonpbSrc, err := protobufs.FS.ReadFile("commonpb/common.pb.go")
	if err != nil {
		setupLog.Info("Static asset not found: common.pb.go")
		return err
	}
	commonpbDir := path.Join(goPathSrc, "github.com", "bishopfox", "sliver", "protobuf", "commonpb")
	os.MkdirAll(commonpbDir, 0700)
	ioutil.WriteFile(path.Join(commonpbDir, "common.pb.go"), commonpbSrc, 0600)

	// DNS PB
	dnspbSrc, err := protobufs.FS.ReadFile("dnspb/dns.pb.go")
	if err != nil {
		setupLog.Info("Static asset not found: dns.pb.go")
		return err
	}
	dnspbDir := path.Join(goPathSrc, "github.com", "bishopfox", "sliver", "protobuf", "dnspb")
	os.MkdirAll(dnspbDir, 0700)
	ioutil.WriteFile(path.Join(dnspbDir, "dns.pb.go"), dnspbSrc, 0600)
	return nil
}

func stripSliverpb(src []byte) []byte {
	out := src
	re := regexp.MustCompile(`protobuf:"[a-z]+,\d+,[a-z]+,name=(?P<FieldName1>[a-zA-Z0-9]+),proto3" json:"(?P<FiledName2>[a-zA-Z0-9]+),[a-z]+"`)
	found := re.FindAllSubmatch(src, -1)
	for _, x := range found {
		line := x[0] // line that matched
		item := x[1] // first named capturing group (FieldName1)
		// we don't care about FieldName2 because its value is the same as FieldName1
		newItem := pseudoRandStringRunes(len(item))
		newLine := bytes.ReplaceAll(line, item, []byte(newItem))
		out = bytes.ReplaceAll(out, line, []byte(newLine))
	}
	return out
}

func unzipGoDependency(fsPath string, targetPath string) error {
	setupLog.Infof("Unpacking go dependency %s -> %s", fsPath, targetPath)

	appDir := GetRootAppDir()
	goDep, err := assetsFs.ReadFile(fsPath)
	if err != nil {
		setupLog.Infof("static asset not found: %s", fsPath)
		return err
	}

	goDepZipPath := path.Join(appDir, path.Base(fsPath))
	defer os.Remove(goDepZipPath)
	ioutil.WriteFile(goDepZipPath, goDep, 0600)
	_, err = unzip(goDepZipPath, targetPath)
	if err != nil {
		setupLog.Infof("Failed to unzip file %s -> %s", goDepZipPath, appDir)
		return err
	}

	return nil
}

func setupCodenames(appDir string) error {
	nouns, err := assetsFs.ReadFile("fs/nouns.txt")
	if err != nil {
		setupLog.Infof("nouns.txt asset not found")
		return err
	}

	adjectives, err := assetsFs.ReadFile("fs/adjectives.txt")
	if err != nil {
		setupLog.Infof("adjectives.txt asset not found")
		return err
	}

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

func pseudoRandStringRunes(n int) string {
	insecureRand.Seed(time.Now().UnixNano())
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[insecureRand.Intn(len(letterRunes))]
	}
	return string(b)
}
