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
	"net"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}
)

func NewSystemResolver() DNSResolver {
	return &SystemResolver{}
}

type SystemResolver struct{}

func (r *SystemResolver) Address() string {
	return "system"
}

func (r *SystemResolver) A(domain string) ([][]byte, time.Duration, error) {
	// {{if .Config.Debug}}
	log.Printf("[dns] %s->A record of %s?", r.Address(), domain)
	// {{end}}
	started := time.Now()
	ips, err := net.LookupIP(domain)
	rtt := time.Since(started)
	if err != nil {
		return nil, rtt, err
	}
	var addrs [][]byte
	for _, ip := range ips {
		if ip.To4() != nil {
			addrs = append(addrs, ip.To4())
		}
		if ip.To16() != nil {
			addrs = append(addrs, ip.To16())
		}
	}
	return addrs, rtt, nil
}
