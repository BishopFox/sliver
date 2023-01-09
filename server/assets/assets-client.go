//go:build client

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

	"github.com/bishopfox/sliver/util/encoders"
)

var (
	//go:embed fs/english.txt fs/sliver.asc
	assetsFs embed.FS

	defaultTrafficEncoders embed.FS // Always empty for client
)

// Blank for client, no need for traffic encoders
func setupTrafficEncoders(appDir string) error {
	return nil
}

type EmptyEncoderFS struct{}

func (e EmptyEncoderFS) Open(name string) (fs.File, error) {
	return nil, os.ErrNotExist
}

func (e EmptyEncoderFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return []fs.DirEntry{}, nil
}

func (e EmptyEncoderFS) ReadFile(name string) ([]byte, error) {
	return nil, os.ErrNotExist
}

func loadTrafficEncoders(appDir string) encoders.EncoderFS {
	return EmptyEncoderFS{}
}

func setupGo(appDir string) error {
	return nil
}

func setupCodenames(appDir string) error {
	return nil
}
