package comm

import (
	"context"
	"fmt"
	"net"
	"time"
)

// Dialer - A common net.Dialer, with custom functions for spawning connections from the Comm system.
// This dialer may be assigned a specific Resolver, but generally it uses the Comm system to get the
// appropriate route.
type Dialer struct {
	*net.Dialer                 // Gives us most fields used by the Comms.
	*Route                      // The dialer uses a route for getting to its host target.
	ctx         context.Context // The context used to dial destinations with timeoutes, deadlines, etc
	cancel      context.CancelFunc
}

// Dial - Get a network connection to a host in the Comm system.
// Available stream networks are "tcp", "tcp4", "tcp6".
// Available packet networks are "udp", "udp4", "udp6".
func (d *Dialer) Dial(network string, host string) (conn net.Conn, err error) {
	if d.ctx == nil {
		d.ctx, d.cancel = context.WithTimeout(context.Background(), d.Timeout)
		defer d.cancel()
	}
	return d.DialContext(d.ctx, network, host)
}

// DialContext - Get a network connection to a host in the Comm system, with a Context.
// Available stream networks are "tcp", "tcp4", "tcp6".
// Available packet networks are "udp", "udp4", "udp6".
func (d *Dialer) DialContext(ctx context.Context, network string, host string) (conn net.Conn, err error) {
	defer d.cancel()

	// Parse the context and setup the dialer with it.
	// This overwrites any context set up in any of the dialers below.
	if deadline, exists := ctx.Deadline(); exists {
		timeout := time.Until(deadline)
		d.ctx, d.cancel = context.WithTimeout(ctx, timeout)
	}

	// Get a route to the address.
	d.Route, err = ResolveAddress(host)
	if err != nil {
		return nil, fmt.Errorf("Address lookup failed: %s", err.Error())
	}

	// If no route and no error, dial on the server's interfaces.
	if d.Route == nil {
		return net.Dial(network, host)
	}

	// Dial the appropriate network
	switch network {
	case "tcp", "tcp4", "tcp6":
		return d.Route.comm.dialContextTCP(ctx, network, host)
	case "udp", "udp4", "udp6":
		uc, err := d.Route.comm.dialContextUDP(ctx, network, host)
		if err != nil {
			return nil, err
		}
		// Cast the packet conn into our custom udpConn, which satisfies net.Conn.
		conn, _ = uc.(*udpConn)
		return conn, nil
	default:
		return d.Route.comm.dialContextTCP(ctx, network, host)
	}
}

// DialerDefault - A dialer with default connection options. Most use cases.
func DialerDefault() (dialer *Dialer) {
	dialer = &Dialer{
		&net.Dialer{
			KeepAlive: 30 * time.Second, // Should be 15 on OS, do a bit less.
			Timeout:   10 * time.Second,
		},
		nil,
		context.Background(),
		nil,
	}

	dialer.ctx, dialer.cancel = context.WithTimeout(dialer.ctx, dialer.Timeout)
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
		nil,
		context.Background(),
		nil,
	}
	dialer.ctx, dialer.cancel = context.WithTimeout(dialer.ctx, dialer.Timeout)
	return
}

// DialerTight - A dialer with sticter monitoring and expectations.
func DialerTight() (dialer *Dialer) {
	dialer = &Dialer{
		&net.Dialer{
			KeepAlive: 10 * time.Second, // A bit slighter than OS default.
			Timeout:   3 * time.Second,
		},
		nil,
		context.Background(),
		nil,
	}
	dialer.ctx, dialer.cancel = context.WithTimeout(dialer.ctx, dialer.Timeout)
	return
}

// DialerScan - Sane default for callers that are scans. Lightweight.
func DialerScan() (dialer *Dialer) {
	dialer = &Dialer{
		&net.Dialer{
			KeepAlive: -1,               // Disabled by default because the scan is working.
			Timeout:   10 * time.Second, // More time if scan is running in bad conditions.
		},
		nil,
		context.Background(),
		nil,
	}

	dialer.ctx, dialer.cancel = context.WithTimeout(dialer.ctx, dialer.Timeout)

	return
}
