/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019-2021 WireGuard LLC. All Rights Reserved.
 * https://git.zx2c4.com/wireguard-go/tree/tun/netstack/tun.go
 */

package netstack

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.zx2c4.com/wireguard/tun"

	"golang.org/x/net/dns/dnsmessage"
	"inet.af/netstack/tcpip"
	"inet.af/netstack/tcpip/adapters/gonet"
	"inet.af/netstack/tcpip/buffer"
	"inet.af/netstack/tcpip/header"
	"inet.af/netstack/tcpip/network/ipv4"
	"inet.af/netstack/tcpip/network/ipv6"
	"inet.af/netstack/tcpip/stack"
	"inet.af/netstack/tcpip/transport/tcp"
	"inet.af/netstack/tcpip/transport/udp"
)

type netTun struct {
	stack          *stack.Stack
	dispatcher     stack.NetworkDispatcher
	events         chan tun.Event
	incomingPacket chan buffer.VectorisedView
	mtu            int
	dnsServers     []net.IP
	hasV4, hasV6   bool
}
type endpoint netTun
type Net netTun

func (e *endpoint) Attach(dispatcher stack.NetworkDispatcher) {
	e.dispatcher = dispatcher
}

func (e *endpoint) IsAttached() bool {
	return e.dispatcher != nil
}

func (e *endpoint) MTU() uint32 {
	mtu, err := (*netTun)(e).MTU()
	if err != nil {
		panic(err)
	}
	return uint32(mtu)
}

func (*endpoint) Capabilities() stack.LinkEndpointCapabilities {
	return stack.CapabilityNone
}

func (*endpoint) MaxHeaderLength() uint16 {
	return 0
}

func (*endpoint) LinkAddress() tcpip.LinkAddress {
	return ""
}

func (*endpoint) Wait() {}

func (e *endpoint) WritePacket(_ stack.RouteInfo, _ *stack.GSO, _ tcpip.NetworkProtocolNumber, pkt *stack.PacketBuffer) tcpip.Error {
	e.incomingPacket <- buffer.NewVectorisedView(pkt.Size(), pkt.Views())
	return nil
}

func (e *endpoint) WritePackets(stack.RouteInfo, *stack.GSO, stack.PacketBufferList, tcpip.NetworkProtocolNumber) (int, tcpip.Error) {
	panic("not implemented")
}

func (*endpoint) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

func (e *endpoint) AddHeader(tcpip.LinkAddress, tcpip.LinkAddress, tcpip.NetworkProtocolNumber, *stack.PacketBuffer) {
}

func CreateNetTUN(localAddresses, dnsServers []net.IP, mtu int) (tun.Device, *Net, error) {
	opts := stack.Options{
		NetworkProtocols:   []stack.NetworkProtocolFactory{ipv4.NewProtocol, ipv6.NewProtocol},
		TransportProtocols: []stack.TransportProtocolFactory{tcp.NewProtocol, udp.NewProtocol},
		HandleLocal:        true,
	}
	dev := &netTun{
		stack:          stack.New(opts),
		events:         make(chan tun.Event, 10),
		incomingPacket: make(chan buffer.VectorisedView),
		dnsServers:     dnsServers,
		mtu:            mtu,
	}
	tcpipErr := dev.stack.CreateNIC(1, (*endpoint)(dev))
	if tcpipErr != nil {
		return nil, nil, fmt.Errorf("CreateNIC: %v", tcpipErr)
	}
	for _, ip := range localAddresses {
		if ip4 := ip.To4(); ip4 != nil {
			tcpipErr = dev.stack.AddAddress(1, ipv4.ProtocolNumber, tcpip.Address(ip4))
			if tcpipErr != nil {
				return nil, nil, fmt.Errorf("AddAddress(%v): %v", ip4, tcpipErr)
			}
			dev.hasV4 = true
		} else {
			tcpipErr = dev.stack.AddAddress(1, ipv6.ProtocolNumber, tcpip.Address(ip))
			if tcpipErr != nil {
				return nil, nil, fmt.Errorf("AddAddress(%v): %v", ip4, tcpipErr)
			}
			dev.hasV6 = true
		}
	}
	if dev.hasV4 {
		dev.stack.AddRoute(tcpip.Route{Destination: header.IPv4EmptySubnet, NIC: 1})
		tcpipErr = dev.stack.SetForwarding(header.IPv4ProtocolNumber, true)
		if tcpipErr != nil {
			return nil, nil, fmt.Errorf("SetForwarding(): %v", tcpipErr)
		}
	}
	if dev.hasV6 {
		dev.stack.AddRoute(tcpip.Route{Destination: header.IPv6EmptySubnet, NIC: 1})
		tcpipErr = dev.stack.SetForwarding(header.IPv6ProtocolNumber, true)
		if tcpipErr != nil {
			return nil, nil, fmt.Errorf("SetForwarding(): %v", tcpipErr)
		}
	}

	dev.events <- tun.EventUp
	return dev, (*Net)(dev), nil
}

func (tun *netTun) Name() (string, error) {
	return "go", nil
}

func (tun *netTun) File() *os.File {
	return nil
}

func (tun *netTun) Events() chan tun.Event {
	return tun.events
}

func (tun *netTun) Read(buf []byte, offset int) (int, error) {
	view, ok := <-tun.incomingPacket
	if !ok {
		return 0, os.ErrClosed
	}
	return view.Read(buf[offset:])
}

func (tun *netTun) Write(buf []byte, offset int) (int, error) {
	packet := buf[offset:]
	if len(packet) == 0 {
		return 0, nil
	}

	pkb := stack.NewPacketBuffer(stack.PacketBufferOptions{Data: buffer.NewVectorisedView(len(packet), []buffer.View{buffer.NewViewFromBytes(packet)})})
	switch packet[0] >> 4 {
	case 4:
		tun.dispatcher.DeliverNetworkPacket("", "", ipv4.ProtocolNumber, pkb)
	case 6:
		tun.dispatcher.DeliverNetworkPacket("", "", ipv6.ProtocolNumber, pkb)
	}

	return len(buf), nil
}

func (tun *netTun) Flush() error {
	return nil
}

func (tun *netTun) Close() error {
	tun.stack.RemoveNIC(1)

	if tun.events != nil {
		close(tun.events)
	}
	if tun.incomingPacket != nil {
		close(tun.incomingPacket)
	}
	return nil
}

func (tun *netTun) MTU() (int, error) {
	return tun.mtu, nil
}

func convertToFullAddr(ip net.IP, port int) (tcpip.FullAddress, tcpip.NetworkProtocolNumber) {
	if ip4 := ip.To4(); ip4 != nil {
		return tcpip.FullAddress{
			NIC:  1,
			Addr: tcpip.Address(ip4),
			Port: uint16(port),
		}, ipv4.ProtocolNumber
	} else {
		return tcpip.FullAddress{
			NIC:  1,
			Addr: tcpip.Address(ip),
			Port: uint16(port),
		}, ipv6.ProtocolNumber
	}
}

func (net *Net) DialContextTCP(ctx context.Context, addr *net.TCPAddr) (*gonet.TCPConn, error) {
	if addr == nil {
		panic("todo: deal with auto addr semantics for nil addr")
	}
	fa, pn := convertToFullAddr(addr.IP, addr.Port)
	return gonet.DialContextTCP(ctx, net.stack, fa, pn)
}

func (net *Net) DialTCP(addr *net.TCPAddr) (*gonet.TCPConn, error) {
	if addr == nil {
		panic("todo: deal with auto addr semantics for nil addr")
	}
	fa, pn := convertToFullAddr(addr.IP, addr.Port)
	return gonet.DialTCP(net.stack, fa, pn)
}

func (net *Net) ListenTCP(addr *net.TCPAddr) (*gonet.TCPListener, error) {
	if addr == nil {
		panic("todo: deal with auto addr semantics for nil addr")
	}
	fa, pn := convertToFullAddr(addr.IP, addr.Port)
	return gonet.ListenTCP(net.stack, fa, pn)
}

func (net *Net) DialUDP(laddr, raddr *net.UDPAddr) (*gonet.UDPConn, error) {
	var lfa, rfa *tcpip.FullAddress
	var pn tcpip.NetworkProtocolNumber
	if laddr != nil {
		var addr tcpip.FullAddress
		addr, pn = convertToFullAddr(laddr.IP, laddr.Port)
		lfa = &addr
	}
	if raddr != nil {
		var addr tcpip.FullAddress
		addr, pn = convertToFullAddr(raddr.IP, raddr.Port)
		rfa = &addr
	}
	return gonet.DialUDP(net.stack, lfa, rfa, pn)
}

var (
	errNoSuchHost                   = errors.New("no such host")
	errLameReferral                 = errors.New("lame referral")
	errCannotUnmarshalDNSMessage    = errors.New("cannot unmarshal DNS message")
	errCannotMarshalDNSMessage      = errors.New("cannot marshal DNS message")
	errServerMisbehaving            = errors.New("server misbehaving")
	errInvalidDNSResponse           = errors.New("invalid DNS response")
	errNoAnswerFromDNSServer        = errors.New("no answer from DNS server")
	errServerTemporarilyMisbehaving = errors.New("server misbehaving")
	errCanceled                     = errors.New("operation was canceled")
	errTimeout                      = errors.New("i/o timeout")
	errNumericPort                  = errors.New("port must be numeric")
	errNoSuitableAddress            = errors.New("no suitable address found")
	errMissingAddress               = errors.New("missing address")
)

func (net *Net) LookupHost(host string) (addrs []string, err error) {
	return net.LookupContextHost(context.Background(), host)
}

func isDomainName(s string) bool {
	l := len(s)
	if l == 0 || l > 254 || l == 254 && s[l-1] != '.' {
		return false
	}
	last := byte('.')
	nonNumeric := false
	partlen := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		default:
			return false
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c == '_':
			nonNumeric = true
			partlen++
		case '0' <= c && c <= '9':
			partlen++
		case c == '-':
			if last == '.' {
				return false
			}
			partlen++
			nonNumeric = true
		case c == '.':
			if last == '.' || last == '-' {
				return false
			}
			if partlen > 63 || partlen == 0 {
				return false
			}
			partlen = 0
		}
		last = c
	}
	if last == '-' || partlen > 63 {
		return false
	}
	return nonNumeric
}

func randU16() uint16 {
	var b [2]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint16(b[:])
}

func newRequest(q dnsmessage.Question) (id uint16, udpReq, tcpReq []byte, err error) {
	id = randU16()
	b := dnsmessage.NewBuilder(make([]byte, 2, 514), dnsmessage.Header{ID: id, RecursionDesired: true})
	b.EnableCompression()
	if err := b.StartQuestions(); err != nil {
		return 0, nil, nil, err
	}
	if err := b.Question(q); err != nil {
		return 0, nil, nil, err
	}
	tcpReq, err = b.Finish()
	udpReq = tcpReq[2:]
	l := len(tcpReq) - 2
	tcpReq[0] = byte(l >> 8)
	tcpReq[1] = byte(l)
	return id, udpReq, tcpReq, err
}

func equalASCIIName(x, y dnsmessage.Name) bool {
	if x.Length != y.Length {
		return false
	}
	for i := 0; i < int(x.Length); i++ {
		a := x.Data[i]
		b := y.Data[i]
		if 'A' <= a && a <= 'Z' {
			a += 0x20
		}
		if 'A' <= b && b <= 'Z' {
			b += 0x20
		}
		if a != b {
			return false
		}
	}
	return true
}

func checkResponse(reqID uint16, reqQues dnsmessage.Question, respHdr dnsmessage.Header, respQues dnsmessage.Question) bool {
	if !respHdr.Response {
		return false
	}
	if reqID != respHdr.ID {
		return false
	}
	if reqQues.Type != respQues.Type || reqQues.Class != respQues.Class || !equalASCIIName(reqQues.Name, respQues.Name) {
		return false
	}
	return true
}

func dnsPacketRoundTrip(c net.Conn, id uint16, query dnsmessage.Question, b []byte) (dnsmessage.Parser, dnsmessage.Header, error) {
	if _, err := c.Write(b); err != nil {
		return dnsmessage.Parser{}, dnsmessage.Header{}, err
	}
	b = make([]byte, 512)
	for {
		n, err := c.Read(b)
		if err != nil {
			return dnsmessage.Parser{}, dnsmessage.Header{}, err
		}
		var p dnsmessage.Parser
		h, err := p.Start(b[:n])
		if err != nil {
			continue
		}
		q, err := p.Question()
		if err != nil || !checkResponse(id, query, h, q) {
			continue
		}
		return p, h, nil
	}
}

func dnsStreamRoundTrip(c net.Conn, id uint16, query dnsmessage.Question, b []byte) (dnsmessage.Parser, dnsmessage.Header, error) {
	if _, err := c.Write(b); err != nil {
		return dnsmessage.Parser{}, dnsmessage.Header{}, err
	}
	b = make([]byte, 1280)
	if _, err := io.ReadFull(c, b[:2]); err != nil {
		return dnsmessage.Parser{}, dnsmessage.Header{}, err
	}
	l := int(b[0])<<8 | int(b[1])
	if l > len(b) {
		b = make([]byte, l)
	}
	n, err := io.ReadFull(c, b[:l])
	if err != nil {
		return dnsmessage.Parser{}, dnsmessage.Header{}, err
	}
	var p dnsmessage.Parser
	h, err := p.Start(b[:n])
	if err != nil {
		return dnsmessage.Parser{}, dnsmessage.Header{}, errCannotUnmarshalDNSMessage
	}
	q, err := p.Question()
	if err != nil {
		return dnsmessage.Parser{}, dnsmessage.Header{}, errCannotUnmarshalDNSMessage
	}
	if !checkResponse(id, query, h, q) {
		return dnsmessage.Parser{}, dnsmessage.Header{}, errInvalidDNSResponse
	}
	return p, h, nil
}

func (tnet *Net) exchange(ctx context.Context, server net.IP, q dnsmessage.Question, timeout time.Duration) (dnsmessage.Parser, dnsmessage.Header, error) {
	q.Class = dnsmessage.ClassINET
	id, udpReq, tcpReq, err := newRequest(q)
	if err != nil {
		return dnsmessage.Parser{}, dnsmessage.Header{}, errCannotMarshalDNSMessage
	}

	for _, useUDP := range []bool{true, false} {
		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(timeout))
		defer cancel()

		var c net.Conn
		var err error
		if useUDP {
			c, err = tnet.DialUDP(nil, &net.UDPAddr{IP: server, Port: 53})
		} else {
			c, err = tnet.DialContextTCP(ctx, &net.TCPAddr{IP: server, Port: 53})
		}

		if err != nil {
			return dnsmessage.Parser{}, dnsmessage.Header{}, err
		}
		if d, ok := ctx.Deadline(); ok && !d.IsZero() {
			c.SetDeadline(d)
		}
		var p dnsmessage.Parser
		var h dnsmessage.Header
		if useUDP {
			p, h, err = dnsPacketRoundTrip(c, id, q, udpReq)
		} else {
			p, h, err = dnsStreamRoundTrip(c, id, q, tcpReq)
		}
		c.Close()
		if err != nil {
			if err == context.Canceled {
				err = errCanceled
			} else if err == context.DeadlineExceeded {
				err = errTimeout
			}
			return dnsmessage.Parser{}, dnsmessage.Header{}, err
		}
		if err := p.SkipQuestion(); err != dnsmessage.ErrSectionDone {
			return dnsmessage.Parser{}, dnsmessage.Header{}, errInvalidDNSResponse
		}
		if h.Truncated {
			continue
		}
		return p, h, nil
	}
	return dnsmessage.Parser{}, dnsmessage.Header{}, errNoAnswerFromDNSServer
}

func checkHeader(p *dnsmessage.Parser, h dnsmessage.Header) error {
	if h.RCode == dnsmessage.RCodeNameError {
		return errNoSuchHost
	}
	_, err := p.AnswerHeader()
	if err != nil && err != dnsmessage.ErrSectionDone {
		return errCannotUnmarshalDNSMessage
	}
	if h.RCode == dnsmessage.RCodeSuccess && !h.Authoritative && !h.RecursionAvailable && err == dnsmessage.ErrSectionDone {
		return errLameReferral
	}
	if h.RCode != dnsmessage.RCodeSuccess && h.RCode != dnsmessage.RCodeNameError {
		if h.RCode == dnsmessage.RCodeServerFailure {
			return errServerTemporarilyMisbehaving
		}
		return errServerMisbehaving
	}
	return nil
}

func skipToAnswer(p *dnsmessage.Parser, qtype dnsmessage.Type) error {
	for {
		h, err := p.AnswerHeader()
		if err == dnsmessage.ErrSectionDone {
			return errNoSuchHost
		}
		if err != nil {
			return errCannotUnmarshalDNSMessage
		}
		if h.Type == qtype {
			return nil
		}
		if err := p.SkipAnswer(); err != nil {
			return errCannotUnmarshalDNSMessage
		}
	}
}

func (tnet *Net) tryOneName(ctx context.Context, name string, qtype dnsmessage.Type) (dnsmessage.Parser, string, error) {
	var lastErr error

	n, err := dnsmessage.NewName(name)
	if err != nil {
		return dnsmessage.Parser{}, "", errCannotMarshalDNSMessage
	}
	q := dnsmessage.Question{
		Name:  n,
		Type:  qtype,
		Class: dnsmessage.ClassINET,
	}

	for i := 0; i < 2; i++ {
		for _, server := range tnet.dnsServers {
			p, h, err := tnet.exchange(ctx, server, q, time.Second*5)
			if err != nil {
				dnsErr := &net.DNSError{
					Err:    err.Error(),
					Name:   name,
					Server: server.String(),
				}
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					dnsErr.IsTimeout = true
				}
				if _, ok := err.(*net.OpError); ok {
					dnsErr.IsTemporary = true
				}
				lastErr = dnsErr
				continue
			}

			if err := checkHeader(&p, h); err != nil {
				dnsErr := &net.DNSError{
					Err:    err.Error(),
					Name:   name,
					Server: server.String(),
				}
				if err == errServerTemporarilyMisbehaving {
					dnsErr.IsTemporary = true
				}
				if err == errNoSuchHost {
					dnsErr.IsNotFound = true
					return p, server.String(), dnsErr
				}
				lastErr = dnsErr
				continue
			}

			err = skipToAnswer(&p, qtype)
			if err == nil {
				return p, server.String(), nil
			}
			lastErr = &net.DNSError{
				Err:    err.Error(),
				Name:   name,
				Server: server.String(),
			}
			if err == errNoSuchHost {
				lastErr.(*net.DNSError).IsNotFound = true
				return p, server.String(), lastErr
			}
		}
	}
	return dnsmessage.Parser{}, "", lastErr
}

func (tnet *Net) LookupContextHost(ctx context.Context, host string) ([]string, error) {
	if host == "" || (!tnet.hasV6 && !tnet.hasV4) {
		return nil, &net.DNSError{Err: errNoSuchHost.Error(), Name: host, IsNotFound: true}
	}
	zlen := len(host)
	if strings.IndexByte(host, ':') != -1 {
		if zidx := strings.LastIndexByte(host, '%'); zidx != -1 {
			zlen = zidx
		}
	}
	if ip := net.ParseIP(host[:zlen]); ip != nil {
		return []string{host[:zlen]}, nil
	}

	if !isDomainName(host) {
		return nil, &net.DNSError{Err: errNoSuchHost.Error(), Name: host, IsNotFound: true}
	}
	type result struct {
		p      dnsmessage.Parser
		server string
		error
	}
	var addrsV4, addrsV6 []net.IP
	lanes := 0
	if tnet.hasV4 {
		lanes++
	}
	if tnet.hasV6 {
		lanes++
	}
	lane := make(chan result, lanes)
	var lastErr error
	if tnet.hasV4 {
		go func() {
			p, server, err := tnet.tryOneName(ctx, host+".", dnsmessage.TypeA)
			lane <- result{p, server, err}
		}()
	}
	if tnet.hasV6 {
		go func() {
			p, server, err := tnet.tryOneName(ctx, host+".", dnsmessage.TypeAAAA)
			lane <- result{p, server, err}
		}()
	}
	for l := 0; l < lanes; l++ {
		result := <-lane
		if result.error != nil {
			if lastErr == nil {
				lastErr = result.error
			}
			continue
		}

	loop:
		for {
			h, err := result.p.AnswerHeader()
			if err != nil && err != dnsmessage.ErrSectionDone {
				lastErr = &net.DNSError{
					Err:    errCannotMarshalDNSMessage.Error(),
					Name:   host,
					Server: result.server,
				}
			}
			if err != nil {
				break
			}
			switch h.Type {
			case dnsmessage.TypeA:
				a, err := result.p.AResource()
				if err != nil {
					lastErr = &net.DNSError{
						Err:    errCannotMarshalDNSMessage.Error(),
						Name:   host,
						Server: result.server,
					}
					break loop
				}
				addrsV4 = append(addrsV4, net.IP(a.A[:]))

			case dnsmessage.TypeAAAA:
				aaaa, err := result.p.AAAAResource()
				if err != nil {
					lastErr = &net.DNSError{
						Err:    errCannotMarshalDNSMessage.Error(),
						Name:   host,
						Server: result.server,
					}
					break loop
				}
				addrsV6 = append(addrsV6, net.IP(aaaa.AAAA[:]))

			default:
				if err := result.p.SkipAnswer(); err != nil {
					lastErr = &net.DNSError{
						Err:    errCannotMarshalDNSMessage.Error(),
						Name:   host,
						Server: result.server,
					}
					break loop
				}
				continue
			}
		}
	}
	// We don't do RFC6724. Instead just put V6 addresess first if an IPv6 address is enabled
	var addrs []net.IP
	if tnet.hasV6 {
		addrs = append(addrsV6, addrsV4...)
	} else {
		addrs = append(addrsV4, addrsV6...)
	}

	if len(addrs) == 0 && lastErr != nil {
		return nil, lastErr
	}
	saddrs := make([]string, 0, len(addrs))
	for _, ip := range addrs {
		saddrs = append(saddrs, ip.String())
	}
	return saddrs, nil
}

func partialDeadline(now, deadline time.Time, addrsRemaining int) (time.Time, error) {
	if deadline.IsZero() {
		return deadline, nil
	}
	timeRemaining := deadline.Sub(now)
	if timeRemaining <= 0 {
		return time.Time{}, errTimeout
	}
	timeout := timeRemaining / time.Duration(addrsRemaining)
	const saneMinimum = 2 * time.Second
	if timeout < saneMinimum {
		if timeRemaining < saneMinimum {
			timeout = timeRemaining
		} else {
			timeout = saneMinimum
		}
	}
	return now.Add(timeout), nil
}

func (tnet *Net) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if ctx == nil {
		panic("nil context")
	}
	var acceptV4, acceptV6, useUDP bool
	if len(network) == 3 {
		acceptV4 = true
		acceptV6 = true
	} else if len(network) == 4 {
		acceptV4 = network[3] == '4'
		acceptV6 = network[3] == '6'
	}
	if !acceptV4 && !acceptV6 {
		return nil, &net.OpError{Op: "dial", Err: net.UnknownNetworkError(network)}
	}
	if network[:3] == "udp" {
		useUDP = true
	} else if network[:3] != "tcp" {
		return nil, &net.OpError{Op: "dial", Err: net.UnknownNetworkError(network)}
	}
	host, sport, err := net.SplitHostPort(address)
	if err != nil {
		return nil, &net.OpError{Op: "dial", Err: err}
	}
	port, err := strconv.Atoi(sport)
	if err != nil || port < 0 || port > 65535 {
		return nil, &net.OpError{Op: "dial", Err: errNumericPort}
	}
	allAddr, err := tnet.LookupContextHost(ctx, host)
	if err != nil {
		return nil, &net.OpError{Op: "dial", Err: err}
	}
	var addrs []net.IP
	for _, addr := range allAddr {
		if strings.IndexByte(addr, ':') != -1 && acceptV6 {
			addrs = append(addrs, net.ParseIP(addr))
		} else if strings.IndexByte(addr, '.') != -1 && acceptV4 {
			addrs = append(addrs, net.ParseIP(addr))
		}
	}
	if len(addrs) == 0 && len(allAddr) != 0 {
		return nil, &net.OpError{Op: "dial", Err: errNoSuitableAddress}
	}

	var firstErr error
	for i, addr := range addrs {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if err == context.Canceled {
				err = errCanceled
			} else if err == context.DeadlineExceeded {
				err = errTimeout
			}
			return nil, &net.OpError{Op: "dial", Err: err}
		default:
		}

		dialCtx := ctx
		if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
			partialDeadline, err := partialDeadline(time.Now(), deadline, len(addrs)-i)
			if err != nil {
				if firstErr == nil {
					firstErr = &net.OpError{Op: "dial", Err: err}
				}
				break
			}
			if partialDeadline.Before(deadline) {
				var cancel context.CancelFunc
				dialCtx, cancel = context.WithDeadline(ctx, partialDeadline)
				defer cancel()
			}
		}

		var c net.Conn
		if useUDP {
			c, err = tnet.DialUDP(nil, &net.UDPAddr{IP: addr, Port: port})
		} else {
			c, err = tnet.DialContextTCP(dialCtx, &net.TCPAddr{IP: addr, Port: port})
		}
		if err == nil {
			return c, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr == nil {
		firstErr = &net.OpError{Op: "dial", Err: errMissingAddress}
	}
	return nil, firstErr
}

func (tnet *Net) Dial(network, address string) (net.Conn, error) {
	return tnet.DialContext(context.Background(), network, address)
}
