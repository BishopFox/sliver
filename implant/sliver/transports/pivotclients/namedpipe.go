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

	// {{if .Config.Debug}}
	"log"
	// {{end}}
)

var (
	defaultNamedPipeDeadline = time.Second * 10
)

// NamedPipePivotOptions - Options for the NamedPipe pivot
type NamedPipePivotOptions struct {
	Timeout       time.Duration
	ReadDeadline  time.Duration
	WriteDeadline time.Duration
}

// ParseNamedPipePivotOptions - Parse the options for the TCP pivot from a C2 URL
func ParseNamedPipePivotOptions(uri *url.URL) *NamedPipePivotOptions {
	opts := &NamedPipePivotOptions{}
	if readDeadline, err := time.ParseDuration(uri.Query().Get("read-deadline")); err == nil {
		opts.ReadDeadline = readDeadline
	} else {
		// {{if .Config.Debug}}
		if uri.Query().Get("read-deadline") != "" {
			log.Printf("failed to parse read-deadline: %s", err)
		}
		// {{end}}
		opts.ReadDeadline = defaultNamedPipeDeadline
	}
	if writeDeadline, err := time.ParseDuration(uri.Query().Get("write-deadline")); err == nil {
		opts.WriteDeadline = writeDeadline
	} else {
		// {{if .Config.Debug}}
		if uri.Query().Get("write-deadline") != "" {
			log.Printf("failed to parse write-deadline: %s", err)
		}
		// {{end}}
		opts.WriteDeadline = defaultNamedPipeDeadline
	}

	if timeout, err := time.ParseDuration(uri.Query().Get("timeout")); err == nil {
		opts.Timeout = timeout
	} else {
		// {{if .Config.Debug}}
		if uri.Query().Get("timeout") != "" {
			log.Printf("failed to parse timeout: %s", err)
		}
		// {{end}}
		opts.Timeout = defaultNamedPipeDeadline
	}

	return opts
}
