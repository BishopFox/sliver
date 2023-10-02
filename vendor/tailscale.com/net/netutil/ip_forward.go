// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

// Package netutil contains misc shared networking code & types.
package netutil

import (
	"bytes"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"tailscale.com/net/interfaces"
)

// protocolsRequiredForForwarding reports whether IPv4 and/or IPv6 protocols are
// required to forward the specified routes.
// The state param must be specified.
func protocolsRequiredForForwarding(routes []netip.Prefix, state *interfaces.State) (v4, v6 bool) {
	if len(routes) == 0 {
		// Nothing to route, so no need to warn.
		return false, false
	}

	localIPs := make(map[netip.Addr]bool)
	for _, addrs := range state.InterfaceIPs {
		for _, pfx := range addrs {
			localIPs[pfx.Addr()] = true
		}
	}

	for _, r := range routes {
		// It's possible to advertise a route to one of the local
		// machine's local IPs. IP forwarding isn't required for this
		// to work, so we shouldn't warn for such exports.
		if r.IsSingleIP() && localIPs[r.Addr()] {
			continue
		}
		if r.Addr().Is4() {
			v4 = true
		} else {
			v6 = true
		}
	}
	return v4, v6
}

// CheckIPForwarding reports whether IP forwarding is enabled correctly
// for subnet routing and exit node functionality on any interface.
// The state param must not be nil.
// The routes should only be advertised routes, and should not contain the
// nodes Tailscale IPs.
// It returns an error if it is unable to determine if IP forwarding is enabled.
// It returns a warning describing configuration issues if IP forwarding is
// non-functional or partly functional.
func CheckIPForwarding(routes []netip.Prefix, state *interfaces.State) (warn, err error) {
	if runtime.GOOS != "linux" {
		switch runtime.GOOS {
		case "dragonfly", "freebsd", "netbsd", "openbsd":
			return fmt.Errorf("Subnet routing and exit nodes only work with additional manual configuration on %v, and is not currently officially supported.", runtime.GOOS), nil
		}
		return nil, nil
	}
	if state == nil {
		return nil, fmt.Errorf("Couldn't check system's IP forwarding configuration; no link state")
	}
	const kbLink = "\nSee https://tailscale.com/s/ip-forwarding"
	wantV4, wantV6 := protocolsRequiredForForwarding(routes, state)
	if !wantV4 && !wantV6 {
		return nil, nil
	}

	v4e, err := ipForwardingEnabledLinux(ipv4, "")
	if err != nil {
		return nil, fmt.Errorf("Couldn't check system's IP forwarding configuration, subnet routing/exit nodes may not work: %w%s", err, kbLink)
	}
	v6e, err := ipForwardingEnabledLinux(ipv6, "")
	if err != nil {
		return nil, fmt.Errorf("Couldn't check system's IP forwarding configuration, subnet routing/exit nodes may not work: %w%s", err, kbLink)
	}

	if v4e && v6e {
		// IP forwarding is enabled systemwide, all is well.
		return nil, nil
	}

	if !wantV4 {
		if !v6e {
			return nil, fmt.Errorf("IPv6 forwarding is disabled, subnet routing/exit nodes may not work.%s", kbLink)
		}
		return nil, nil
	}
	// IP forwarding isn't enabled globally, but it might be enabled
	// on a per-interface basis. Check if it's on for all interfaces,
	// and warn appropriately if it's not.
	// Note: you might be wondering why we check only the state of
	// ipv6.conf.all.forwarding, rather than per-interface forwarding
	// configuration. According to kernel documentation, it seems
	// that to actually forward packets, you need to enable
	// forwarding globally, and the per-interface forwarding
	// setting only alters other things such as how router
	// advertisements are handled. The kernel itself warns that
	// enabling forwarding per-interface and not globally will
	// probably not work, so I feel okay calling those configs
	// broken until we have proof otherwise.
	var (
		anyEnabled bool
		warnings   []string
	)
	if wantV6 && !v6e {
		warnings = append(warnings, "IPv6 forwarding is disabled.")
	}
	for _, iface := range state.Interface {
		if iface.Name == "lo" {
			continue
		}
		v4e, err := ipForwardingEnabledLinux(ipv4, iface.Name)
		if err != nil {
			return nil, fmt.Errorf("Couldn't check system's IP forwarding configuration, subnet routing/exit nodes may not work: %w%s", err, kbLink)
		} else if !v4e {
			warnings = append(warnings, fmt.Sprintf("Traffic received on %s won't be forwarded (%s disabled)", iface.Name, ipForwardSysctlKey(dotFormat, ipv4, iface.Name)))
		} else {
			anyEnabled = true
		}
	}
	if !anyEnabled {
		// IP forwarding is completely disabled, just say that rather
		// than enumerate all the interfaces on the system.
		return fmt.Errorf("IP forwarding is disabled, subnet routing/exit nodes will not work.%s", kbLink), nil
	}
	if len(warnings) > 0 {
		// If partially enabled, enumerate the bits that won't work.
		return fmt.Errorf("%s\nSubnet routes and exit nodes may not work correctly.%s", strings.Join(warnings, "\n"), kbLink), nil
	}

	return nil, nil
}

// ipForwardSysctlKey returns the sysctl key for the given protocol and iface.
// When the dotFormat parameter is true the output is formatted as `net.ipv4.ip_forward`,
// else it is `net/ipv4/ip_forward`
func ipForwardSysctlKey(format sysctlFormat, p protocol, iface string) string {
	if iface == "" {
		if format == dotFormat {
			if p == ipv4 {
				return "net.ipv4.ip_forward"
			}
			return "net.ipv6.conf.all.forwarding"
		}
		if p == ipv4 {
			return "net/ipv4/ip_forward"
		}
		return "net/ipv6/conf/all/forwarding"
	}

	var k string
	if p == ipv4 {
		k = "net/ipv4/conf/%s/forwarding"
	} else {
		k = "net/ipv6/conf/%s/forwarding"
	}
	if format == dotFormat {
		// Swap the delimiters.
		iface = strings.ReplaceAll(iface, ".", "/")
		k = strings.ReplaceAll(k, "/", ".")
	}
	return fmt.Sprintf(k, iface)
}

type sysctlFormat int

const (
	dotFormat sysctlFormat = iota
	slashFormat
)

type protocol int

const (
	ipv4 protocol = iota
	ipv6
)

// ipForwardingEnabledLinux reports whether the IP Forwarding is enabled for the
// given interface.
// The iface param determines which interface to check against, "" means to check
// global config.
// This is Linux-specific: it only reads from /proc/sys and doesn't shell out to
// sysctl (which on Linux just reads from /proc/sys anyway).
func ipForwardingEnabledLinux(p protocol, iface string) (bool, error) {
	k := ipForwardSysctlKey(slashFormat, p, iface)
	bs, err := os.ReadFile(filepath.Join("/proc/sys", k))
	if err != nil {
		if os.IsNotExist(err) {
			// If IPv6 is disabled, sysctl keys like "net.ipv6.conf.all.forwarding" just don't
			// exist on disk. But first diagnose whether procfs is even mounted before assuming
			// absence means false.
			if fi, err := os.Stat("/proc/sys"); err != nil {
				return false, fmt.Errorf("failed to check sysctl %v; no procfs? %w", k, err)
			} else if !fi.IsDir() {
				return false, fmt.Errorf("failed to check sysctl %v; /proc/sys isn't a directory, is %v", k, fi.Mode())
			}
			return false, nil
		}
		return false, err
	}

	val, err := strconv.ParseInt(string(bytes.TrimSpace(bs)), 10, 32)
	if err != nil {
		return false, fmt.Errorf("couldn't parse %s: %w", k, err)
	}
	// 0 = disabled, 1 = enabled, 2 = enabled (but uncommon)
	// https://github.com/tailscale/tailscale/issues/8375
	if val < 0 || val > 2 {
		return false, fmt.Errorf("unexpected value %d for %s", val, k)
	}
	on := val == 1 || val == 2
	return on, nil
}
