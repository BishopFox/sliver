package pivotclients

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
	"net/url"
	"time"
)

var (
	defaultNamedPipeDeadline = time.Second * 10
)

// ParseNamedPipePivotOptions - Parse the options for the TCP pivot from a C2 URL
func ParseNamedPipePivotOptions(uri *url.URL) *NamedPipePivotOptions {
	readDeadline, err := time.ParseDuration(uri.Query().Get("read-deadline"))
	if err != nil {
		readDeadline = defaultNamedPipeDeadline
	}
	writeDeadline, err := time.ParseDuration(uri.Query().Get("write-deadline"))
	if err != nil {
		writeDeadline = defaultNamedPipeDeadline
	}
	return &NamedPipePivotOptions{
		ReadDeadline:  readDeadline,
		WriteDeadline: writeDeadline,
	}
}

// NamedPipePivotOptions - Options for the NamedPipe pivot
type NamedPipePivotOptions struct {
	ReadDeadline  time.Duration
	WriteDeadline time.Duration
}
