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
)

var (
	//go:embed fs/english.txt fs/sliver.asc
	assetsFs embed.FS

	TrafficEncoderFS embed.FS // Always empty for client
)

func setupGo(appDir string) error {
	return nil
}

func setupCodenames(appDir string) error {
	return nil
}
