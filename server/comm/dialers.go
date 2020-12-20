package comm

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"
)

// Dialer - A common net.Dialer, with custom functions for spawning connections from the Comm system.
// This dialer may be assigned a specific Resolver, but generally it uses the Comm system to get the
// appropriate route.
type Dialer struct {
	*net.Dialer        // Gives us most fields used by the Comms.
	route       *Route // The dialer uses a route for getting to its host target.
}

// Dial - Get a network connection to a host in the Comm system. Available networks are tcp/udp/unix/ip
func (d *Dialer) Dial(network string, host string) (conn net.Conn, err error) {
	return d.DialContext(context.Background(), network, host)
}

// DialContext - Get a network connection to a host, with a Context. Available networks are tcp/udp/unix/ip
func (d *Dialer) DialContext(ctx context.Context, network string, host string) (conn net.Conn, err error) {
	d.route, err = ResolveAddress(host)
	if err != nil {
		return nil, fmt.Errorf("Address lookup failed: %s", err.Error())
	}

	// Get RHost/RPort
	uri, _ := url.Parse(fmt.Sprintf("%s://%s", network, host))
	if uri == nil {
		return nil, fmt.Errorf("Address parsing failed: %s", host)
	}

	return d.dialContext(ctx, uri)
}

// dialContext - Actually dial and get the conn.
func (d *Dialer) dialContext(ctx context.Context, uri *url.URL) (conn net.Conn, err error) {

	info := newConnInfo(uri, d.route)                       // Prepare connection info
	conn, err = d.route.comm.dial(info)                     // Instantiate connection over Comms
	d.route.Connections = append(d.route.Connections, conn) // Add connection to active

	return
}

// DialerDefault - A dialer with default connection options. Most use cases.
func DialerDefault() (dialer *Dialer) {
	dialer = &Dialer{
		&net.Dialer{
			KeepAlive: 30 * time.Second, // Should be 15 on OS, do a bit less.
			Timeout:   10 * time.Second,
		},
		nil, // No routes found yet
	}

	return
}

// DialerStealthy - A dialer for either stealthy routing, quick handlers,
// with no useless things sent over the network.
func DialerStealthy() (dialer *Dialer) {
	dialer = &Dialer{
		&net.Dialer{
			// Could use randomly generated values ?
			KeepAlive: -1,              // Disabled
			Timeout:   3 * time.Second, // plenty enough on modern network.
		},
		nil, // No routes found yet
	}
	return
}

// DialerTight - A dialer with sticter monitoring and expectations.
func DialerTight() (dialer *Dialer) {
	dialer = &Dialer{
		&net.Dialer{
			KeepAlive: 10 * time.Second, // A bit slighter than OS default.
			Timeout:   3 * time.Second,
		},
		nil, // No routes found yet
	}
	return
}

// DialerScan - Sane default for callers that are scans. Lightweight.
func DialerScan() (dialer *Dialer) {
	dialer = &Dialer{
		&net.Dialer{
			KeepAlive: -1,               // Disabled by default because the scan is working.
			Timeout:   10 * time.Second, // More time if scan is running in bad conditions.
		},
		nil, // No routes found yet
	}
	return
}
