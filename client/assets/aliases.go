package assets

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"log"
	"os"
	"path/filepath"
)

const (
	AliasesDirName = "aliases"
)

// GetAliasesDir - Returns the path to the config dir
func GetAliasesDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := filepath.Join(rootDir, AliasesDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

// GetInstalledAliasManifests - Returns a list of installed alias manifests
func GetInstalledAliasManifests() []string {
	aliasDir := GetAliasesDir()
	aliasDirContent, err := os.ReadDir(aliasDir)
	if err != nil {
		log.Printf("error loading aliases: %s", err)
		return []string{}
	}
	manifests := []string{}
	for _, fi := range aliasDirContent {
		if fi.IsDir() {
			manifestPath := filepath.Join(aliasDir, fi.Name(), "alias.json")
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				log.Printf("no manifest in %s, skipping ...", manifestPath)
				continue
			}
			manifests = append(manifests, manifestPath)
		}
	}
	return manifests
}
