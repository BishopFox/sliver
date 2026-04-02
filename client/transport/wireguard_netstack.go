package transport

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	insecurerand "math/rand"
	"net"
	"net/netip"
	"os"
	"syscall"

	"golang.zx2c4.com/wireguard/tun"
	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

type transportTun struct {
	ep             *channel.Endpoint
	stack          *stack.Stack
	events         chan tun.Event
	notifyHandle   *channel.NotificationHandle
	incomingPacket chan *buffer.View
	mtu            int
	primaryAddr    netip.Addr
	hasV4          bool
	hasV6          bool
}

type transportNet transportTun

const (
	transportTCPReceiveBufferMax = 8 << 20
	transportTCPSendBufferMax    = 6 << 20
	transportTCPBindPortMin      = 49152
	transportTCPBindPortSpan     = 65535 - transportTCPBindPortMin + 1
)

func createTransportNetTUN(localAddresses []netip.Addr, mtu int) (tun.Device, *transportNet, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(0xFFFFFFFF))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create random number: %w", err)
	}
	clock := tcpip.NewStdClock()
	rng := insecurerand.New(insecurerand.NewSource(n.Int64()))

	opts := stack.Options{
		NetworkProtocols:   []stack.NetworkProtocolFactory{ipv4.NewProtocol, ipv6.NewProtocol},
		TransportProtocols: []stack.TransportProtocolFactory{tcp.NewProtocol, udp.NewProtocol, icmp.NewProtocol6, icmp.NewProtocol4},
		HandleLocal:        true,
		IPTables:           stack.DefaultTables(clock, rng),
	}
	dev := &transportTun{
		ep:             channel.New(1024, uint32(mtu), ""),
		stack:          stack.New(opts),
		events:         make(chan tun.Event, 10),
		incomingPacket: make(chan *buffer.View),
		mtu:            mtu,
	}

	if err := configureTransportTCPStack(dev.stack); err != nil {
		return nil, nil, err
	}

	dev.notifyHandle = dev.ep.AddNotify(dev)
	if tcpipErr := dev.stack.CreateNIC(1, dev.ep); tcpipErr != nil {
		return nil, nil, fmt.Errorf("CreateNIC: %v", tcpipErr)
	}

	for _, ip := range localAddresses {
		var protoNumber tcpip.NetworkProtocolNumber
		if ip.Is4() {
			protoNumber = ipv4.ProtocolNumber
		} else if ip.Is6() {
			protoNumber = ipv6.ProtocolNumber
		}
		protoAddr := tcpip.ProtocolAddress{
			Protocol:          protoNumber,
			AddressWithPrefix: tcpip.AddrFromSlice(ip.AsSlice()).WithPrefix(),
		}
		if tcpipErr := dev.stack.AddProtocolAddress(1, protoAddr, stack.AddressProperties{}); tcpipErr != nil {
			return nil, nil, fmt.Errorf("AddProtocolAddress(%v): %v", ip, tcpipErr)
		}
		if !dev.primaryAddr.IsValid() {
			dev.primaryAddr = ip
		}
		if ip.Is4() {
			dev.hasV4 = true
		} else if ip.Is6() {
			dev.hasV6 = true
		}
	}

	if dev.hasV4 {
		dev.stack.AddRoute(tcpip.Route{Destination: header.IPv4EmptySubnet, NIC: 1})
	}
	if dev.hasV6 {
		dev.stack.AddRoute(tcpip.Route{Destination: header.IPv6EmptySubnet, NIC: 1})
	}

	dev.events <- tun.EventUp
	return dev, (*transportNet)(dev), nil
}

func configureTransportTCPStack(ipstack *stack.Stack) error {
	sackEnabledOpt := tcpip.TCPSACKEnabled(true)
	if tcpipErr := ipstack.SetTransportProtocolOption(tcp.ProtocolNumber, &sackEnabledOpt); tcpipErr != nil {
		return fmt.Errorf("could not enable TCP SACK: %v", tcpipErr)
	}

	tcpRecoveryOpt := tcpip.TCPRecovery(0)
	if tcpipErr := ipstack.SetTransportProtocolOption(tcp.ProtocolNumber, &tcpRecoveryOpt); tcpipErr != nil {
		return fmt.Errorf("could not disable TCP RACK: %v", tcpipErr)
	}

	renoOpt := tcpip.CongestionControlOption("reno")
	if tcpipErr := ipstack.SetTransportProtocolOption(tcp.ProtocolNumber, &renoOpt); tcpipErr != nil {
		return fmt.Errorf("could not set TCP congestion control to reno: %v", tcpipErr)
	}

	tcpRXBufOpt := tcpip.TCPReceiveBufferSizeRangeOption{
		Min:     tcp.MinBufferSize,
		Default: tcp.DefaultSendBufferSize,
		Max:     transportTCPReceiveBufferMax,
	}
	if tcpipErr := ipstack.SetTransportProtocolOption(tcp.ProtocolNumber, &tcpRXBufOpt); tcpipErr != nil {
		return fmt.Errorf("could not set TCP RX buffer size: %v", tcpipErr)
	}

	tcpTXBufOpt := tcpip.TCPSendBufferSizeRangeOption{
		Min:     tcp.MinBufferSize,
		Default: tcp.DefaultReceiveBufferSize,
		Max:     transportTCPSendBufferMax,
	}
	if tcpipErr := ipstack.SetTransportProtocolOption(tcp.ProtocolNumber, &tcpTXBufOpt); tcpipErr != nil {
		return fmt.Errorf("could not set TCP TX buffer size: %v", tcpipErr)
	}

	return nil
}

func (tun *transportTun) Name() (string, error) {
	return "go", nil
}

func (tun *transportTun) File() *os.File {
	return nil
}

func (tun *transportTun) Events() <-chan tun.Event {
	return tun.events
}

func (tun *transportTun) Read(buf [][]byte, sizes []int, offset int) (int, error) {
	view, ok := <-tun.incomingPacket
	if !ok {
		return 0, os.ErrClosed
	}

	n, err := view.Read(buf[0][offset:])
	if err != nil {
		return 0, err
	}
	sizes[0] = n
	return 1, nil
}

func (tun *transportTun) Write(buf [][]byte, offset int) (int, error) {
	for _, buf := range buf {
		packet := buf[offset:]
		if len(packet) == 0 {
			continue
		}

		pkb := stack.NewPacketBuffer(stack.PacketBufferOptions{Payload: buffer.MakeWithData(packet)})
		switch packet[0] >> 4 {
		case 4:
			tun.ep.InjectInbound(header.IPv4ProtocolNumber, pkb)
		case 6:
			tun.ep.InjectInbound(header.IPv6ProtocolNumber, pkb)
		default:
			return 0, syscall.EAFNOSUPPORT
		}
	}
	return len(buf), nil
}

func (tun *transportTun) WriteNotify() {
	pkt := tun.ep.Read()
	if pkt == nil {
		return
	}

	view := pkt.ToView()
	pkt.DecRef()

	tun.incomingPacket <- view
}

func (tun *transportTun) Close() error {
	tun.stack.RemoveNIC(1)
	tun.stack.Close()
	tun.ep.RemoveNotify(tun.notifyHandle)
	tun.ep.Close()

	if tun.events != nil {
		close(tun.events)
	}
	if tun.incomingPacket != nil {
		close(tun.incomingPacket)
	}

	return nil
}

func (tun *transportTun) MTU() (int, error) {
	return tun.mtu, nil
}

func (tun *transportTun) BatchSize() int {
	return 1
}

func convertTransportFullAddr(endpoint netip.AddrPort) (tcpip.FullAddress, tcpip.NetworkProtocolNumber) {
	var protoNumber tcpip.NetworkProtocolNumber
	if endpoint.Addr().Is4() {
		protoNumber = ipv4.ProtocolNumber
	} else {
		protoNumber = ipv6.ProtocolNumber
	}
	return tcpip.FullAddress{
		NIC:  1,
		Addr: tcpip.AddrFromSlice(endpoint.Addr().AsSlice()),
		Port: endpoint.Port(),
	}, protoNumber
}

func (netstack *transportNet) DialContextTCPAddrPort(ctx context.Context, addr netip.AddrPort) (*gonet.TCPConn, error) {
	remoteAddr, pn := convertTransportFullAddr(addr)
	localAddr, err := netstack.randomTCPBindAddr()
	if err != nil {
		return nil, err
	}
	return gonet.DialTCPWithBind(ctx, netstack.stack, localAddr, remoteAddr, pn)
}

func (netstack *transportNet) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	switch network {
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, fmt.Errorf("unsupported network %q", network)
	}

	host, portString, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	port, err := net.LookupPort("tcp", portString)
	if err != nil {
		return nil, err
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return nil, err
	}
	return netstack.DialContextTCPAddrPort(ctx, netip.AddrPortFrom(addr, uint16(port)))
}

func (netstack *transportNet) randomTCPBindAddr() (tcpip.FullAddress, error) {
	if netstack == nil || !netstack.primaryAddr.IsValid() {
		return tcpip.FullAddress{}, errors.New("wireguard transport has no local address")
	}

	port, err := randomTransportTCPBindPort()
	if err != nil {
		return tcpip.FullAddress{}, err
	}

	return tcpip.FullAddress{
		NIC:  1,
		Addr: tcpip.AddrFromSlice(netstack.primaryAddr.AsSlice()),
		Port: port,
	}, nil
}

func randomTransportTCPBindPort() (uint16, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(transportTCPBindPortSpan))
	if err != nil {
		return 0, fmt.Errorf("failed to choose random wireguard TCP bind port: %w", err)
	}
	return uint16(transportTCPBindPortMin + n.Int64()), nil
}
