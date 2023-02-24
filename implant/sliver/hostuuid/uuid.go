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
	"crypto/sha256"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"net"
	"sort"

	"github.com/gofrs/uuid"
)

var zeroGUID = uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000000"))

// UUIDFromMAC - Generate a UUID based on the machine's MAC addresses, this is
// generally used as a last resort to fingerprint the host machine. It creates
// a uuid by hashing the MAC addresses of all network interfaces and using the
// first 16 bytes of the hash as the UUID. This should work so long as network
// interfaces are not added or removed, since its physical addresses this should
// be uncommon; an except would be machines with USB WiFi/Ethernet or something.
func UUIDFromMAC() string {
	// {{if .Config.Debug}}
	log.Printf("Generating host UUID from hardware addresses ...")
	// {{end}}
	interfaces, err := net.Interfaces()
	if err != nil {
		return zeroGUID.String()
	}
	hardwareAddrs := []string{}
	for _, iface := range interfaces {
		if iface.HardwareAddr != nil {
			hardwareAddrs = append(hardwareAddrs, iface.HardwareAddr.String())
		}
	}
	if len(hardwareAddrs) == 0 {
		return zeroGUID.String()
	}
	sort.Strings(hardwareAddrs) // Ensure deterministic order
	digest := sha256.New()
	for _, addr := range hardwareAddrs {
		digest.Write([]byte(addr))
	}
	value, err := uuid.FromBytes(digest.Sum(nil)[:16]) // Must be 128-bits
	if err != nil {
		return zeroGUID.String()
	}
	return value.String()
}
