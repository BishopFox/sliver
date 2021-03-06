// +build windows

package hostuuid

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
	"golang.org/x/sys/windows/registry"
)

// Stored Format: {U-U-I-D}
var uuid_keypath = "HKEY_LOCAL_MACHINE\\SYSTEM\\HardwareConfig"
var uuid_key = "LastConfig"

func GetUUID() string {
	key, err := registry.OpenKey(registry.CURRENT_USER, uuid_keypath, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}

	str, _, err := key.GetStringValue(uuid_key)
	if err != nil {
		return ""
	}

	return str[1:37]
}
