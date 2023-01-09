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
	insecureRand "math/rand"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	ver "github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/util/encoders"
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

	trafficEncoderLog = log.NamedLogger("assets", "traffic-encoders")
)

// GetRootAppDir - Get the Sliver app dir, default is: ~/.sliver/
func GetRootAppDir() string {
	value := os.Getenv(envVarName)
	var dir string
	if len(value) == 0 {
		user, _ := user.Current()
		dir = filepath.Join(user.HomeDir, ".sliver")
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

func GetChunkDataDir() string {
	chunkDir := filepath.Join(GetRootAppDir(), "crack", "chunks")
	if _, err := os.Stat(chunkDir); os.IsNotExist(err) {
		err = os.MkdirAll(chunkDir, 0700)
		if err != nil {
			setupLog.Errorf("Failed to create chunk data directory: %s", err)
			return ""
		}
	}
	return chunkDir
}

func assetVersion() string {
	appDir := GetRootAppDir()
	data, err := os.ReadFile(path.Join(appDir, versionFileName))
	if err != nil {
		setupLog.Infof("No version detected %s", err)
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
		setupTrafficEncoders(appDir)
		saveAssetVersion(appDir)
	}
	setupLog.Infof("Initializing encoders ...")
	err := encoders.InitEncoderMap(loadTrafficEncoders(appDir), func(msg string) {
		trafficEncoderLog.Infof("[traffic encoder] %s", msg)
	})
	if err != nil {
		trafficEncoderLog.Errorf("Failed to initialize traffic encoders: %s", err)
	}
	encoders.InitEnglishDictionary(English())
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

		fPath := filepath.Clean(filepath.Join(dest, file.Name))
		if !strings.HasPrefix(fPath, filepath.Clean(dest)) {
			panic("illegal zip file path")
		}
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

func unzipBuf(src []byte, dest string) ([]string, error) {
	var filenames []string
	reader, err := zip.NewReader(bytes.NewReader(src), int64(len(src)))
	if err != nil {
		return filenames, err
	}

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
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[insecureRand.Intn(len(letterRunes))]
	}
	return string(b)
}
