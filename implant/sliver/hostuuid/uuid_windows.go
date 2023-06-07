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
	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"golang.org/x/sys/windows/registry"
)

// Stored Format: {U-U-I-D}
var uuidKeyPath = "SYSTEM\\HardwareConfig"
var uuidKey = "LastConfig"

func GetUUID() string {
	var uuidStr string
	var err error

	key, err := registry.OpenKey(registry.LOCAL_MACHINE, uuidKeyPath, registry.QUERY_VALUE)
	if err == nil {
		uuidStr, _, err = key.GetStringValue(uuidKey)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Failed to read reg key string value: %s", err)
			// {{end}}
			return UUIDFromMAC()
		}
		if 37 <= len(uuidStr) {
			// {{if .Config.Debug}}
			log.Printf("Registry host uuid value too short")
			// {{end}}
			return uuidStr[1:37]
		}
	} else {
		// {{if .Config.Debug}}
		log.Printf("Failed to read reg key: %s", err)
		// {{end}}
	}
	return UUIDFromMAC()
}
