// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build unix || js || wasip1

package net

import (
	"context"
	"errors"
	"internal/bytealg"
	"strings"
	"sync"
	"syscall"
)

// readProtocolsOnce loads contents of /etc/protocols into protocols map
// for quick access.
var readProtocolsOnce = sync.OnceFunc(func() {
	file, err := open("/etc/protocols")
	if err != nil {
		return
	}
	defer file.close()

	for line, ok := file.readLine(); ok; line, ok = file.readLine() {
		// tcp    6   TCP    # transmission control protocol
		if i := bytealg.IndexByteString(line, '#'); i >= 0 {
			line = line[0:i]
		}
		f := getFields(line)
		if len(f) < 2 {
			continue
		}
		if proto, _, ok := dtoi(f[1]); ok {
			if _, ok := protocols[f[0]]; !ok {
				protocols[f[0]] = proto
			}
			for _, alias := range f[2:] {
				if _, ok := protocols[alias]; !ok {
					protocols[alias] = proto
				}
			}
		}
	}
})

// lookupProtocol looks up IP protocol name in /etc/protocols and
// returns correspondent protocol number.
func lookupProtocol(_ context.Context, name string) (int, error) {
	readProtocolsOnce()
	return lookupProtocolMap(name)
}

func (r *Resolver) lookupHost(ctx context.Context, host string) ([]string, error) {
	return wasiNetLookup(ctx, "ip", host)
}

func (r *Resolver) lookupIP(ctx context.Context, network, host string) ([]IPAddr, error) {
	lookupNetwork := wasiNetLookupNetwork(network)
	if lookupNetwork == "" {
		return nil, UnknownNetworkError(network)
	}

	addresses, err := wasiNetLookup(ctx, lookupNetwork, host)
	if err != nil {
		return nil, err
	}
	results := make([]IPAddr, 0, len(addresses))
	for _, address := range addresses {
		hostPart, zone := splitHostZone(address)
		ip := ParseIP(hostPart)
		if ip == nil {
			continue
		}
		switch lookupNetwork {
		case "ip4":
			ip = ip.To4()
			if ip == nil {
				continue
			}
		case "ip6":
			if ip.To4() != nil {
				continue
			}
			ip = ip.To16()
			if ip == nil {
				continue
			}
		}
		results = append(results, IPAddr{IP: ip, Zone: zone})
	}
	if len(results) == 0 {
		return nil, &DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	return results, nil
}

func wasiNetLookupNetwork(network string) string {
	switch network {
	case "ip", "tcp", "udp":
		return "ip"
	case "ip4", "tcp4", "udp4":
		return "ip4"
	case "ip6", "tcp6", "udp6":
		return "ip6"
	default:
		return ""
	}
}

func wasiNetLookup(ctx context.Context, network, name string) ([]string, error) {
	var operation uint32
	errno := wasiNetLookupStart(network, name, wasiNetContextDeadline(ctx), &operation)
	if errno != 0 {
		return nil, wasiNetLookupError(wasiNetError("lookup", errno), name)
	}
	result, err := wasiNetWaitOperation(ctx, operation, nil, 1024)
	if err != nil {
		return nil, wasiNetLookupError(err, name)
	}

	parts := strings.Split(string(result.address), "\x00")
	addresses := parts[:0]
	for _, address := range parts {
		if address != "" {
			addresses = append(addresses, address)
		}
	}
	if len(addresses) == 0 {
		return nil, &DNSError{Err: "no such host", Name: name, IsNotFound: true}
	}
	return addresses, nil
}

func wasiNetLookupError(err error, name string) error {
	dnsError := &DNSError{Err: err.Error(), Name: name}
	switch {
	case errors.Is(err, syscall.ENOENT):
		dnsError.Err = "no such host"
		dnsError.IsNotFound = true
	case errors.Is(err, context.Canceled):
		dnsError.UnwrapErr = err
	case errors.Is(err, context.DeadlineExceeded):
		dnsError.UnwrapErr = err
		dnsError.IsTimeout = true
	case errors.Is(err, syscall.ETIMEDOUT):
		dnsError.IsTimeout = true
	}
	return dnsError
}

func (r *Resolver) lookupPort(ctx context.Context, network, service string) (int, error) {
	// Port lookup is not a DNS operation.
	// Prefer the cgo resolver if possible.
	if !systemConf().mustUseGoResolver(r) {
		port, err := cgoLookupPort(ctx, network, service)
		if err != nil {
			// Issue 18213: if cgo fails, first check to see whether we
			// have the answer baked-in to the net package.
			if port, err := goLookupPort(network, service); err == nil {
				return port, nil
			}
		}
		return port, err
	}
	return goLookupPort(network, service)
}

func (r *Resolver) lookupCNAME(ctx context.Context, name string) (string, error) {
	order, conf := systemConf().hostLookupOrder(r, name)
	if order == hostLookupCgo {
		if cname, err := cgoLookupCNAME(ctx, name); err == nil {
			return cname, nil
		}
	}
	return r.goLookupCNAME(ctx, name, order, conf)
}

func (r *Resolver) lookupSRV(ctx context.Context, service, proto, name string) (string, []*SRV, error) {
	return r.goLookupSRV(ctx, service, proto, name)
}

func (r *Resolver) lookupMX(ctx context.Context, name string) ([]*MX, error) {
	return r.goLookupMX(ctx, name)
}

func (r *Resolver) lookupNS(ctx context.Context, name string) ([]*NS, error) {
	return r.goLookupNS(ctx, name)
}

func (r *Resolver) lookupTXT(ctx context.Context, name string) ([]string, error) {
	return r.goLookupTXT(ctx, name)
}

func (r *Resolver) lookupAddr(ctx context.Context, addr string) ([]string, error) {
	order, conf := systemConf().addrLookupOrder(r, addr)
	if order == hostLookupCgo {
		return cgoLookupPTR(ctx, addr)
	}
	return r.goLookupPTR(ctx, addr, order, conf)
}
