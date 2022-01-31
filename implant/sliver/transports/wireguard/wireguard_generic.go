//go:build !windows && !darwin && !linux

package wireguard

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

// WireguardConnect - Placeholder for unsupported platforms
func WireguardConnect(_ string, _ uint16) (net.Conn, *device.Device, error) {
	return nil, nil, errors.New("{{if .Config.Debug}}WireguardConnect not implemented on this platform{{end}}")
}
