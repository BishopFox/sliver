//go:build !(386 || amd64 || arm64)

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

import "errors"

type LinuxExtension struct {
	id   string
	arch string
}

func NewLinuxExtension(_ []byte, id string, arch string, _ string) *LinuxExtension {
	return &LinuxExtension{id: id, arch: arch}
}

func (l *LinuxExtension) GetID() string {
	return l.id
}

func (l *LinuxExtension) GetArch() string {
	return l.arch
}

func (l *LinuxExtension) Load() error {
	return errors.New("{{if .Config.Debug}} native extensions are not supported on this Linux architecture {{end}}")
}

func (l *LinuxExtension) Call(_ string, _ []byte, _ func([]byte)) error {
	return errors.New("{{if .Config.Debug}} native extensions are not supported on this Linux architecture {{end}}")
}
