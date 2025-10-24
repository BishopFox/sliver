/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2023 WireGuard LLC. All Rights Reserved.
 */

// Package conn implements WireGuard's network connections.
package conn

import (
	"errors"
	"fmt"
	"net/netip"
	"reflect"
	"runtime"
	"strings"
)

const (
	IdealBatchSize = 128 // maximum number of packets handled per read and write
)

// A ReceiveFunc receives at least one packet from the network and writes them
// into packets. On a successful read it returns the number of elements of
// sizes, packets, and endpoints that should be evaluated. Some elements of
// sizes may be zero, and callers should ignore them. Callers must pass a sizes
// and eps slice with a length greater than or equal to the length of packets.
// These lengths must not exceed the length of the associated Bind.BatchSize().
type ReceiveFunc func(packets [][]byte, sizes []int, eps []Endpoint) (n int, err error)

// A Bind listens on a port for both IPv6 and IPv4 UDP traffic.
//
// A Bind interface may also be a PeekLookAtSocketFd or BindSocketToInterface,
// depending on the platform-specific implementation.
type Bind interface {
	// Open puts the Bind into a listening state on a given port and reports the actual
	// port that it bound to. Passing zero results in a random selection.
	// fns is the set of functions that will be called to receive packets.
	Open(port uint16) (fns []ReceiveFunc, actualPort uint16, err error)

	// Close closes the Bind listener.
	// All fns returned by Open must return net.ErrClosed after a call to Close.
	Close() error

	// SetMark sets the mark for each packet sent through this Bind.
	// This mark is passed to the kernel as the socket option SO_MARK.
	SetMark(mark uint32) error

	// Send writes one or more packets in bufs to address ep. A nonzero offset
	// can be used to instruct the Bind on where packet data begins in each
	// element of the bufs slice. Space preceding offset is free to use for
	// additional encapsulation. The length of bufs must not exceed BatchSize().
	Send(bufs [][]byte, ep Endpoint, offset int) error

	// ParseEndpoint creates a new endpoint from a string.
	ParseEndpoint(s string) (Endpoint, error)

	// BatchSize is the number of buffers expected to be passed to
	// the ReceiveFuncs, and the maximum expected to be passed to SendBatch.
	BatchSize() int
}

// BindSocketToInterface is implemented by Bind objects that support being
// tied to a single network interface. Used by wireguard-windows.
type BindSocketToInterface interface {
	BindSocketToInterface4(interfaceIndex uint32, blackhole bool) error
	BindSocketToInterface6(interfaceIndex uint32, blackhole bool) error
}

// PeekLookAtSocketFd is implemented by Bind objects that support having their
// file descriptor peeked at. Used by wireguard-android.
type PeekLookAtSocketFd interface {
	PeekLookAtSocketFd4() (fd int, err error)
	PeekLookAtSocketFd6() (fd int, err error)
}

// An Endpoint maintains the source/destination caching for a peer.
//
//	dst: the remote address of a peer ("endpoint" in uapi terminology)
//	src: the local address from which datagrams originate going to the peer
type Endpoint interface {
	ClearSrc()           // clears the source address
	SrcToString() string // returns the local source address (ip:port)
	DstToString() string // returns the destination address (ip:port)
	DstToBytes() []byte  // used for mac2 cookie calculations
	DstIP() netip.Addr
	SrcIP() netip.Addr
}

// InitiationAwareEndpoint is an optional [Endpoint] specialization for
// integrations that want to know when a WireGuard handshake initiation
// message has been received, enabling just-in-time peer configuration before
// attempted decryption.
//
// It's most useful when used in combination with [PeerAwareEndpoint], enabling
// JIT peer configuration and post-decryption peer verification from a single
// implementer.
type InitiationAwareEndpoint interface {
	// InitiationMessagePublicKey is called when a handshake initiation message
	// has been received, and the sender's public key has been identified, but
	// BEFORE an attempt has been made to verify it.
	InitiationMessagePublicKey(peerPublicKey [32]byte)
}

// PeerAwareEndpoint is an optional Endpoint specialization for
// integrations that want to know about the outcome of Cryptokey Routing
// identification.
//
// If they receive a packet from a source they had not pre-identified,
// to learn the identification WireGuard can derive from the session
// or handshake.
//
// A [PeerAwareEndpoint] may be installed as the [conn.Endpoint] following
// successful decryption unless endpoint roaming has been disabled for
// the peer.
type PeerAwareEndpoint interface {
	// FromPeer is called at least once per successfully Cryptokey Routing ID'd
	// [ReceiveFunc] packets batch for a given node key. wireguard-go will
	// always call it for the latest/tail packet in the batch, only ever
	// suppressing calls for older packets.
	FromPeer(peerPublicKey [32]byte)
}

var (
	ErrBindAlreadyOpen   = errors.New("bind is already open")
	ErrWrongEndpointType = errors.New("endpoint type does not correspond with bind type")
)

func (fn ReceiveFunc) PrettyName() string {
	name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	// 0. cheese/taco.beansIPv6.func12.func21218-fm
	name = strings.TrimSuffix(name, "-fm")
	// 1. cheese/taco.beansIPv6.func12.func21218
	if idx := strings.LastIndexByte(name, '/'); idx != -1 {
		name = name[idx+1:]
		// 2. taco.beansIPv6.func12.func21218
	}
	for {
		var idx int
		for idx = len(name) - 1; idx >= 0; idx-- {
			if name[idx] < '0' || name[idx] > '9' {
				break
			}
		}
		if idx == len(name)-1 {
			break
		}
		const dotFunc = ".func"
		if !strings.HasSuffix(name[:idx+1], dotFunc) {
			break
		}
		name = name[:idx+1-len(dotFunc)]
		// 3. taco.beansIPv6.func12
		// 4. taco.beansIPv6
	}
	if idx := strings.LastIndexByte(name, '.'); idx != -1 {
		name = name[idx+1:]
		// 5. beansIPv6
	}
	if name == "" {
		return fmt.Sprintf("%p", fn)
	}
	if strings.HasSuffix(name, "IPv4") {
		return "v4"
	}
	if strings.HasSuffix(name, "IPv6") {
		return "v6"
	}
	return name
}
