package assets

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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
	"embed"
	"io"
	insecureRand "math/rand"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	protobufs "github.com/bishopfox/sliver/protobuf"
	"github.com/bishopfox/sliver/util"
)

var (
	//go:embed traffic-encoders/*.wasm
	trafficEncoderFS embed.FS
)

func unpackDefaultTrafficEncoders(force bool) error {
	encoders, err := trafficEncoderFS.ReadDir("traffic-encoders")
	if err != nil {
		return err
	}
	for _, encoder := range encoders {
		if encoder.IsDir() {
			continue
		}
		encoderName := path.Base(encoder.Name())
		encoderPath := path.Join("traffic-encoders", encoderName)
		encoderBytes, err := trafficEncoderFS.ReadFile(encoderPath)
		if err != nil {
			return err
		}

		localPath := filepath.Join(GetTrafficEncoderDir(), encoderName)
		if _, err := os.Stat(localPath); os.IsNotExist(err) || force {
			err = os.WriteFile(localPath, encoderBytes, 0600)
			if err != nil {
				return err
			}
		} else {
			setupLog.Infof("Skipping unpacking %s, already exists", encoderName)
		}
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

// SetupGo - Unzip Go compiler assets
func setupGo(appDir string) error {
	setupLog.Infof("Unpacking to '%s'", appDir)
	goRootPath := filepath.Join(appDir, GoDirName)
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
	goZipFSPath := filepath.Join("fs", runtime.GOOS, runtime.GOARCH, "go.zip")
	goZip, err := assetsFs.ReadFile(goZipFSPath)
	if err != nil {
		setupLog.Errorf("static asset not found: %s", goZipFSPath)
		return err
	}

	goZipPath := filepath.Join(appDir, "go.zip")
	defer os.Remove(goZipPath)
	os.WriteFile(goZipPath, goZip, 0600)
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
	goSrcZipPath := filepath.Join(appDir, "src.zip")
	defer os.Remove(goSrcZipPath)
	os.WriteFile(goSrcZipPath, goSrcZip, 0600)
	_, err = unzip(goSrcZipPath, goRootPath)
	if err != nil {
		setupLog.Infof("Failed to unzip file %s -> %s/go", goSrcZipPath, appDir)
		return err
	}

	garbleFileName := "garble"
	if runtime.GOOS == "windows" {
		garbleFileName = "garble.exe"
	}
	garbleAssetPath := filepath.Join("fs", runtime.GOOS, runtime.GOARCH, garbleFileName)
	garbleFile, err := assetsFs.ReadFile(garbleAssetPath)
	if err != nil {
		setupLog.Errorf("Static asset not found: %s", garbleFile)
		return err
	}
	garbleLocalPath := filepath.Join(appDir, "go", "bin", garbleFileName)
	err = os.WriteFile(garbleLocalPath, garbleFile, 0700)
	if err != nil {
		setupLog.Errorf("Failed to write garble %s", err)
		return err
	}

	return setupSGN(appDir)
}

func setupSGN(appDir string) error {
	goBinPath := filepath.Join(appDir, "go", "bin")
	sgnZipFSPath := filepath.Join("fs", runtime.GOOS, runtime.GOARCH, "sgn.zip")
	sgnZip, err := assetsFs.ReadFile(sgnZipFSPath)
	if err != nil {
		setupLog.Errorf("static asset not found: %s", sgnZipFSPath)
		return err
	}
	_, err = unzipBuf(sgnZip, goBinPath)
	return err
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
	sliverpbDir := filepath.Join(goPathSrc, "github.com", "bishopfox", "sliver", "protobuf", "sliverpb")
	os.MkdirAll(sliverpbDir, 0700)
	os.WriteFile(filepath.Join(sliverpbDir, "sliver.pb.go"), sliverpbGoSrc, 0600)
	os.WriteFile(filepath.Join(sliverpbDir, "constants.go"), sliverpbConstSrc, 0600)

	// Common PB
	commonpbSrc, err := protobufs.FS.ReadFile("commonpb/common.pb.go")
	if err != nil {
		setupLog.Info("Static asset not found: common.pb.go")
		return err
	}
	commonpbDir := filepath.Join(goPathSrc, "github.com", "bishopfox", "sliver", "protobuf", "commonpb")
	os.MkdirAll(commonpbDir, 0700)
	os.WriteFile(filepath.Join(commonpbDir, "common.pb.go"), commonpbSrc, 0600)

	// DNS PB
	dnspbSrc, err := protobufs.FS.ReadFile("dnspb/dns.pb.go")
	if err != nil {
		setupLog.Info("Static asset not found: dns.pb.go")
		return err
	}
	dnspbDir := filepath.Join(goPathSrc, "github.com", "bishopfox", "sliver", "protobuf", "dnspb")
	os.MkdirAll(dnspbDir, 0700)
	os.WriteFile(filepath.Join(dnspbDir, "dns.pb.go"), dnspbSrc, 0600)
	return nil
}

func stripSliverpb(src []byte) []byte {
	out := src
	re := regexp.MustCompile(`protobuf:"[a-z]+,\d+,[a-z]+,name=(?P<FieldName1>[a-zA-Z0-9]+),proto3(,enum=(?P<EnumName>[a-zA-Z\.]+))?" json:"(?P<FiledName2>[a-zA-Z0-9]+),[a-z]+"`)
	found := re.FindAllSubmatch(src, -1)
	for _, x := range found {
		line := x[0]     // line that matched
		typeName := x[1] // first named capturing group (FieldName1)
		enumName := x[3]
		if string(enumName) != "" {
			newEnumName := pseudoRandStringRunes(len(enumName))
			newEnumLine := bytes.ReplaceAll(line, enumName, []byte(newEnumName))
			out = bytes.ReplaceAll(out, line, []byte(newEnumLine))
			line = newEnumLine
		}
		// we don't care about FieldName2 because its value is the same as FieldName1
		newItem := pseudoRandStringRunes(len(typeName))
		newLine := bytes.ReplaceAll(line, typeName, []byte(newItem))
		out = bytes.ReplaceAll(out, line, []byte(newLine))
	}
	return out
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

	err = os.WriteFile(filepath.Join(appDir, "nouns.txt"), nouns, 0600)
	if err != nil {
		setupLog.Infof("Failed to write noun data to: %s", appDir)
		return err
	}

	err = os.WriteFile(filepath.Join(appDir, "adjectives.txt"), adjectives, 0600)
	if err != nil {
		setupLog.Infof("Failed to write adjective data to: %s", appDir)
		return err
	}
	return nil
}
