//go:build server

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
	"embed"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/util/encoders"
)

var (
	//go:embed traffic-encoders/*.wasm
	defaultTrafficEncoders embed.FS
)

type PassthroughEncoderFS struct {
	rootDir string
}

func (p PassthroughEncoderFS) Open(name string) (fs.File, error) {
	localPath := filepath.Join(p.rootDir, filepath.Base(name))
	if !strings.HasSuffix(localPath, ".wasm") {
		return nil, os.ErrNotExist
	}
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return nil, os.ErrNotExist
	}
	return os.Open(localPath)
}

func (p PassthroughEncoderFS) ReadDir(name string) ([]fs.DirEntry, error) {
	localPath := filepath.Join(p.rootDir, filepath.Base(name))
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return nil, os.ErrNotExist
	}
	ls, err := os.ReadDir(localPath)
	if err != nil {
		return nil, err
	}
	var entries []fs.DirEntry
	for _, entry := range ls {
		if !strings.HasSuffix(entry.Name(), ".wasm") {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (p PassthroughEncoderFS) ReadFile(name string) ([]byte, error) {
	localPath := filepath.Join(p.rootDir, filepath.Base(name))
	if !strings.HasSuffix(localPath, ".wasm") {
		return nil, os.ErrNotExist
	}
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return nil, os.ErrNotExist
	}
	return os.ReadFile(localPath)
}

func loadTrafficEncoders(appDir string) encoders.EncoderFS {
	return PassthroughEncoderFS{rootDir: filepath.Join(appDir, "traffic-encoders")}
}

func setupTrafficEncoders(appDir string) error {
	localTrafficEncodersDir := filepath.Join(appDir, "traffic-encoders")
	if _, err := os.Stat(localTrafficEncodersDir); os.IsNotExist(err) {
		err = os.MkdirAll(localTrafficEncodersDir, 0700)
		if err != nil {
			return err
		}
	}

	encoders, err := defaultTrafficEncoders.ReadDir("traffic-encoders")
	if err != nil {
		return err
	}
	for _, encoder := range encoders {
		if encoder.IsDir() {
			continue
		}
		encoderName := encoder.Name()
		encoderBytes, err := defaultTrafficEncoders.ReadFile(path.Join("traffic-encoders", encoderName))
		if err != nil {
			return err
		}
		err = os.WriteFile(filepath.Join(localTrafficEncodersDir, filepath.Base(encoderName)), encoderBytes, 0600)
		if err != nil {
			return err
		}
	}
	return nil
}
