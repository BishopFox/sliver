//go:build !windows

package dnsclient

import (
	"github.com/miekg/dns"
)

// dnsClientConfig - returns all DNS server addresses associated with the given address
// on non-windows, we ignore the ip parameter because routing is not insane
func dnsClientConfig() (*dns.ClientConfig, error) {
	return dns.ClientConfigFromFile("/etc/resolv.conf")
}
