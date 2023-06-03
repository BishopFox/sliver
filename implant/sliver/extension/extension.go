package extension

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
	"errors"

	// {{if .Config.Debug}}
	"log"
	// {{end}}
)

var extensions map[string]Extension

type Extension interface {
	Load() error
	Call(exportName string, arguments []byte, callback func([]byte)) error
	GetID() string
	GetArch() string
}

func Add(e Extension) {
	extensions[e.GetID()] = e
}

func List() []string {
	var extList []string
	for id := range extensions {
		extList = append(extList, id)
	}
	return extList
}

func Run(extID string, funcName string, arguments []byte, callback func([]byte)) error {
	if ext, found := extensions[extID]; found {
		return ext.Call(funcName, arguments, callback)
	}
	// {{if .Config.Debug}}
	for id, ext := range extensions {
		log.Printf("Extension '%s' (%s)", id, ext.GetArch())
	}
	//{{end}}
	return errors.New("{{if .Config.Debug}} extension not found{{end}}")
}

func init() {
	extensions = make(map[string]Extension)
}
