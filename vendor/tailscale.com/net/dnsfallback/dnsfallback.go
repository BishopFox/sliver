// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

//go:generate go run update-dns-fallbacks.go

// Package dnsfallback contains a DNS fallback mechanism
// for starting up Tailscale when the system DNS is broken or otherwise unavailable.
package dnsfallback

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"reflect"
	"slices"
	"sync/atomic"
	"time"

	"go4.org/netipx"
	"tailscale.com/atomicfile"
	"tailscale.com/envknob"
	"tailscale.com/net/dns/recursive"
	"tailscale.com/net/netmon"
	"tailscale.com/net/netns"
	"tailscale.com/net/tlsdial"
	"tailscale.com/net/tshttpproxy"
	"tailscale.com/tailcfg"
	"tailscale.com/types/logger"
	"tailscale.com/util/clientmetric"
	"tailscale.com/util/slicesx"
)

var disableRecursiveResolver = envknob.RegisterBool("TS_DNSFALLBACK_DISABLE_RECURSIVE_RESOLVER")

// MakeLookupFunc creates a function that can be used to resolve hostnames
// (e.g. as a LookupIPFallback from dnscache.Resolver).
// The netMon parameter is optional; if non-nil it's used to do faster interface lookups.
func MakeLookupFunc(logf logger.Logf, netMon *netmon.Monitor) func(ctx context.Context, host string) ([]netip.Addr, error) {
	return func(ctx context.Context, host string) ([]netip.Addr, error) {
		if disableRecursiveResolver() {
			return lookup(ctx, host, logf, netMon)
		}

		addrsCh := make(chan []netip.Addr, 1)

		// Run the recursive resolver in the background so we can
		// compare the results.
		go func() {
			logf := logger.WithPrefix(logf, "recursive: ")

			// Ensure that we catch panics while we're testing this
			// code path; this should never panic, but we don't
			// want to take down the process by having the panic
			// propagate to the top of the goroutine's stack and
			// then terminate.
			defer func() {
				if r := recover(); r != nil {
					logf("bootstrap DNS: recovered panic: %v", r)
					metricRecursiveErrors.Add(1)
				}
			}()

			resolver := recursive.Resolver{
				Dialer: netns.NewDialer(logf, netMon),
				Logf:   logf,
			}
			addrs, minTTL, err := resolver.Resolve(ctx, host)
			if err != nil {
				logf("error using recursive resolver: %v", err)
				metricRecursiveErrors.Add(1)
				return
			}
			slices.SortFunc(addrs, netipx.CompareAddr)

			// Wait for a response from the main function
			oldAddrs := <-addrsCh
			slices.SortFunc(oldAddrs, netipx.CompareAddr)

			matches := slices.Equal(addrs, oldAddrs)

			logf("bootstrap DNS comparison: matches=%v oldAddrs=%v addrs=%v minTTL=%v", matches, oldAddrs, addrs, minTTL)

			if matches {
				metricRecursiveMatches.Add(1)
			} else {
				metricRecursiveMismatches.Add(1)
			}
		}()

		addrs, err := lookup(ctx, host, logf, netMon)
		if err != nil {
			addrsCh <- nil
			return nil, err
		}

		addrsCh <- slices.Clone(addrs)
		return addrs, nil
	}
}

func lookup(ctx context.Context, host string, logf logger.Logf, netMon *netmon.Monitor) ([]netip.Addr, error) {
	if ip, err := netip.ParseAddr(host); err == nil && ip.IsValid() {
		return []netip.Addr{ip}, nil
	}

	type nameIP struct {
		dnsName string
		ip      netip.Addr
	}

	dm := getDERPMap()

	var cands4, cands6 []nameIP
	for _, dr := range dm.Regions {
		for _, n := range dr.Nodes {
			if ip, err := netip.ParseAddr(n.IPv4); err == nil {
				cands4 = append(cands4, nameIP{n.HostName, ip})
			}
			if ip, err := netip.ParseAddr(n.IPv6); err == nil {
				cands6 = append(cands6, nameIP{n.HostName, ip})
			}
		}
	}
	slicesx.Shuffle(cands4)
	slicesx.Shuffle(cands6)

	const maxCands = 6
	var cands []nameIP // up to maxCands alternating v4/v6 as long as we have both
	for (len(cands4) > 0 || len(cands6) > 0) && len(cands) < maxCands {
		if len(cands4) > 0 {
			cands = append(cands, cands4[0])
			cands4 = cands4[1:]
		}
		if len(cands6) > 0 {
			cands = append(cands, cands6[0])
			cands6 = cands6[1:]
		}
	}
	if len(cands) == 0 {
		return nil, fmt.Errorf("no DNS fallback options for %q", host)
	}
	for _, cand := range cands {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		logf("trying bootstrapDNS(%q, %q) for %q ...", cand.dnsName, cand.ip, host)
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		dm, err := bootstrapDNSMap(ctx, cand.dnsName, cand.ip, host, logf, netMon)
		if err != nil {
			logf("bootstrapDNS(%q, %q) for %q error: %v", cand.dnsName, cand.ip, host, err)
			continue
		}
		if ips := dm[host]; len(ips) > 0 {
			slicesx.Shuffle(ips)
			logf("bootstrapDNS(%q, %q) for %q = %v", cand.dnsName, cand.ip, host, ips)
			return ips, nil
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("no DNS fallback candidates remain for %q", host)
}

// serverName and serverIP of are, say, "derpN.tailscale.com".
// queryName is the name being sought (e.g. "controlplane.tailscale.com"), passed as hint.
func bootstrapDNSMap(ctx context.Context, serverName string, serverIP netip.Addr, queryName string, logf logger.Logf, netMon *netmon.Monitor) (dnsMap, error) {
	dialer := netns.NewDialer(logf, netMon)
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = tshttpproxy.ProxyFromEnvironment
	tr.DialContext = func(ctx context.Context, netw, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, "tcp", net.JoinHostPort(serverIP.String(), "443"))
	}
	tr.TLSClientConfig = tlsdial.Config(serverName, tr.TLSClientConfig)
	c := &http.Client{Transport: tr}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://"+serverName+"/bootstrap-dns?q="+url.QueryEscape(queryName), nil)
	if err != nil {
		return nil, err
	}
	dm := make(dnsMap)
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, errors.New(res.Status)
	}
	if err := json.NewDecoder(res.Body).Decode(&dm); err != nil {
		return nil, err
	}
	return dm, nil
}

// dnsMap is the JSON type returned by the DERP /bootstrap-dns handler:
// https://derp10.tailscale.com/bootstrap-dns
type dnsMap map[string][]netip.Addr

// getDERPMap returns some DERP map. The DERP servers also run a fallback
// DNS server.
func getDERPMap() *tailcfg.DERPMap {
	dm := getStaticDERPMap()

	// Merge in any DERP servers from the cached map that aren't in the
	// static map; this ensures that we're getting new region(s) while not
	// overriding the built-in fallbacks if things go horribly wrong and we
	// get a bad DERP map.
	//
	// TODO(andrew): should we expect OmitDefaultRegions here? We're not
	// forwarding traffic, just resolving DNS, so maybe we can ignore that
	// value anyway?
	cached := cachedDERPMap.Load()
	if cached == nil {
		return dm
	}

	for id, region := range cached.Regions {
		dr, ok := dm.Regions[id]
		if !ok {
			dm.Regions[id] = region
			continue
		}

		// Add any nodes that we don't already have.
		seen := make(map[string]bool)
		for _, n := range dr.Nodes {
			seen[n.HostName] = true
		}
		for _, n := range region.Nodes {
			if !seen[n.HostName] {
				dr.Nodes = append(dr.Nodes, n)
			}
		}
	}

	return dm
}

// getStaticDERPMap returns the DERP map that was compiled into this binary.
func getStaticDERPMap() *tailcfg.DERPMap {
	dm := new(tailcfg.DERPMap)
	if err := json.Unmarshal(staticDERPMapJSON, dm); err != nil {
		panic(err)
	}
	return dm
}

//go:embed dns-fallback-servers.json
var staticDERPMapJSON []byte

// cachedDERPMap is the path to a cached DERP map that we loaded from our on-disk cache.
var cachedDERPMap atomic.Pointer[tailcfg.DERPMap]

// cachePath is the path to the DERP map cache file, set by SetCachePath via
// ipnserver.New() if we have a state directory.
var cachePath string

// UpdateCache stores the DERP map cache back to disk.
//
// The caller must not mutate 'c' after calling this function.
func UpdateCache(c *tailcfg.DERPMap, logf logger.Logf) {
	// Don't do anything if nothing changed.
	curr := cachedDERPMap.Load()
	if reflect.DeepEqual(curr, c) {
		return
	}

	d, err := json.Marshal(c)
	if err != nil {
		logf("[v1] dnsfallback: UpdateCache error marshaling: %v", err)
		return
	}

	// Only store after we're confident this is at least valid JSON
	cachedDERPMap.Store(c)

	// Don't try writing if we don't have a cache path set; this can happen
	// when we don't have a state path (e.g. /var/lib/tailscale) configured.
	if cachePath != "" {
		err = atomicfile.WriteFile(cachePath, d, 0600)
		if err != nil {
			logf("[v1] dnsfallback: UpdateCache error writing: %v", err)
			return
		}
	}
}

// SetCachePath sets the path to the on-disk DERP map cache that we store and
// update. Additionally, if a file at this path exists, we load it and merge it
// with the DERP map baked into the binary.
//
// This function should be called before any calls to UpdateCache, as it is not
// concurrency-safe.
func SetCachePath(path string, logf logger.Logf) {
	cachePath = path

	f, err := os.Open(path)
	if err != nil {
		logf("[v1] dnsfallback: SetCachePath error reading %q: %v", path, err)
		return
	}
	defer f.Close()

	dm := new(tailcfg.DERPMap)
	if err := json.NewDecoder(f).Decode(dm); err != nil {
		logf("[v1] dnsfallback: SetCachePath error decoding %q: %v", path, err)
		return
	}

	cachedDERPMap.Store(dm)
	logf("[v2] dnsfallback: SetCachePath loaded cached DERP map")
}

var (
	metricRecursiveMatches    = clientmetric.NewCounter("dnsfallback_recursive_matches")
	metricRecursiveMismatches = clientmetric.NewCounter("dnsfallback_recursive_mismatches")
	metricRecursiveErrors     = clientmetric.NewCounter("dnsfallback_recursive_errors")
)
