package extensions

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"os"
	"path"

	"github.com/bishopfox/sliver/client/assets"
)

const (
	// ManifestFileName - Extension manifest file name
	ManifestFileName = "extension.json"
)

var loadedExtensions = map[string]*ExtensionManifest{}

type ExtensionManifest struct {
	Name            string               `json:"name"`
	CommandName     string               `json:"command_name"`
	Version         string               `json:"version"`
	ExtensionAuthor string               `json:"extension_author"`
	OriginalAuthor  string               `json:"original_author"`
	RepoURL         string               `json:"repo_url"`
	Help            string               `json:"help"`
	LongHelp        string               `json:"long_help"`
	Files           []*extensionFile     `json:"files"`
	Arguments       []*extensionArgument `json:"arguments"`
	Entrypoint      string               `json:"entrypoint"`
	DependsOn       string               `json:"depends_on"`
	Init            string               `json:"init"`

	RootPath string `json:"-"`
}

type extensionFile struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
	Path string `json:"path"`
}

type extensionArgument struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Desc     string `json:"desc"`
	Optional bool   `json:"optional"`
}

func AddExtension(ext *ExtensionManifest) {
	loadedExtensions[ext.CommandName] = ext
}

func GetLoadedExtension(name string) (*ExtensionManifest, error) {
	if ext, ok := loadedExtensions[name]; ok {
		return ext, nil
	}
	return nil, fmt.Errorf("extension not found: %s", name)
}

func GetExtensions() []*ExtensionManifest {
	extensions := []*ExtensionManifest{}
	for _, ext := range loadedExtensions {
		extensions = append(extensions, ext)
	}
	return extensions
}

func (e *ExtensionManifest) GetFileForTarget(cmdName string, targetOS string, targetArch string) (string, error) {
	filePath := ""
	for _, extFile := range e.Files {
		if targetOS == extFile.OS && targetArch == extFile.Arch {
			filePath = path.Join(assets.GetExtensionsDir(), e.CommandName, extFile.Path)
			break
		}
	}
	if filePath == "" {
		err := fmt.Errorf("no extension file found for %s/%s", targetOS, targetArch)
		return "", err
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		err = fmt.Errorf("extension file not found: %s", filePath)
		return "", err
	}
	return filePath, nil
}
