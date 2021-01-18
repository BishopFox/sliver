package assets

import (
	"log"
	"os"
	"path"
)

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

const (
	userDirPath = "users"
)

// GetUserDirectory - Each user has its own directory.
func GetUserDirectory(name string) (dir string) {

	dir = path.Join(GetRootAppDir(), userDirPath, name)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.Fatalf("Cannot write to Wiregost Data Service directory %s", err)
		}
	}
	return
}

// GetUserHistoryDir - Directory where all history files for a user are stored.
func GetUserHistoryDir(name string) (dir string) {

	dir = path.Join(GetUserDirectory(name), ".history")

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.Fatalf("Cannot write to Wiregost Data Service directory %s", err)
		}
	}
	return
}
