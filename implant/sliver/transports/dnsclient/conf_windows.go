//go:build windows

package dnsclient

/*
	MIT License

	Copyright (c) 2021 awgh

	Permission is hereby granted, free of charge, to any person obtaining a copy
	of this software and associated documentation files (the "Software"), to deal
	in the Software without restriction, including without limitation the rights
	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
	copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all
	copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
	SOFTWARE.
	-------------------------------------------------------------------------------

	Based on https://github.com/awgh/netutils
	Modifications have been made for better interoperability with Sliver.

*/

import (
	"strings"
	"unsafe"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/miekg/dns"
	"golang.org/x/sys/windows"
)

// dnsClientConfig - returns all DNS server addresses associated with the given address
// using various windows fuckery.
func dnsClientConfig() (*dns.ClientConfig, error) {
	l := uint32(20000)
	b := make([]byte, l)

	// Windows is an utter fucking trash fire of an operating system.
	if err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, windows.GAA_FLAG_INCLUDE_PREFIX, 0, (*windows.IpAdapterAddresses)(unsafe.Pointer(&b[0])), &l); err != nil {
		return nil, err
	}
	var adapters []*windows.IpAdapterAddresses
	for addr := (*windows.IpAdapterAddresses)(unsafe.Pointer(&b[0])); addr != nil; addr = addr.Next {
		adapters = append(adapters, addr)
	}

	resolvers := map[string]bool{}
	for _, adapter := range adapters {
		if adapter.OperStatus != windows.IfOperStatusUp {
			continue // Skip down interfaces
		}
		for next := adapter.FirstUnicastAddress; next != nil; next = next.Next {
			if next.Address.IP() != nil {
				for dnsServer := adapter.FirstDnsServerAddress; dnsServer != nil; dnsServer = dnsServer.Next {
					ip := dnsServer.Address.IP()
					if ip.IsMulticast() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
						continue
					}
					if ip.To16() != nil && strings.HasPrefix(ip.To16().String(), "fec0:") {
						continue
					}
					// {{if .Config.Debug}}
					log.Printf("Possible resolver: %v", ip)
					// {{end}}
					resolvers[ip.String()] = true
				}
				break
			}
		}
	}

	// Take unique values only
	servers := []string{}
	for server := range resolvers {
		servers = append(servers, server)
	}

	// TODO: Make configurable, based on defaults in https://github.com/miekg/dns/blob/master/clientconfig.go
	return &dns.ClientConfig{
		Servers:  servers,
		Search:   []string{},
		Port:     "53",
		Ndots:    1,
		Timeout:  5, // seconds
		Attempts: 1,
	}, nil
}
