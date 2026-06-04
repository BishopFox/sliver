// SPDX-License-Identifier: GPL-3.0-or-later

package auth

import (
	"fmt"
	"net"
	"strings"
)

// SourceAllowlist matches a remote source IP against a configured set
// of exact IPs and CIDR ranges. An empty allowlist matches everything
// (allow-all).
//
// Mixed exact-IP and CIDR-range entries are accepted. IPv4 and IPv6
// both work. Exact addresses without a /N suffix are treated as /32
// (v4) or /128 (v6).
type SourceAllowlist struct {
	exact map[string]struct{}
	cidrs []*net.IPNet
}

// NewSourceAllowlist parses entries into an allowlist. Entries may be:
//
//   - "1.2.3.4"        (exact IPv4)
//   - "2001:db8::1"    (exact IPv6)
//   - "10.0.0.0/8"     (IPv4 CIDR)
//   - "2001:db8::/32"  (IPv6 CIDR)
//
// Returns an error on any malformed entry, naming the offending input.
// An empty slice produces an allow-all instance.
func NewSourceAllowlist(entries []string) (*SourceAllowlist, error) {
	a := &SourceAllowlist{exact: make(map[string]struct{})}
	for _, raw := range entries {
		e := strings.TrimSpace(raw)
		if e == "" {
			continue
		}
		if strings.Contains(e, "/") {
			_, n, err := net.ParseCIDR(e)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR %q: %w", e, err)
			}
			a.cidrs = append(a.cidrs, n)
			continue
		}
		if ip := net.ParseIP(e); ip != nil {
			a.exact[ip.String()] = struct{}{} // normalize via ParseIP+String
			continue
		}
		return nil, fmt.Errorf("invalid IP %q", e)
	}
	return a, nil
}

// IsEmpty reports whether the allowlist contains no entries — in which
// case Contains returns true for every input (allow-all).
func (a *SourceAllowlist) IsEmpty() bool {
	return len(a.exact) == 0 && len(a.cidrs) == 0
}

// Contains returns true if remote (a string in standard text form, as
// emitted by net.IP.String()) is in the allowlist. Empty allowlist
// returns true.
func (a *SourceAllowlist) Contains(remote string) bool {
	if a.IsEmpty() {
		return true
	}
	if _, ok := a.exact[remote]; ok {
		return true
	}
	ip := net.ParseIP(remote)
	if ip == nil {
		return false
	}
	for _, n := range a.cidrs {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// Size returns the number of entries (exact + CIDR) for diagnostics.
func (a *SourceAllowlist) Size() int { return len(a.exact) + len(a.cidrs) }
