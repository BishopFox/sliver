//go:build !windows

package transports

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

	-----------------------------------------------------------------------

	Place holder so we don't get undefined functions when compiling to
	non-Windows platforms.

*/

import (
	"errors"
	"net/url"
)

func namedPipeConnect(_ *url.URL) (*Connection, error) {
	return nil, errors.New("{{if .Config.Debug}}Named pipe not supported{{end}}")
}
