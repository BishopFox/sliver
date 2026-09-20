package core

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/bishopfox/sliver/client/tcpproxy"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var (
	// Portfwds - Struct instance that holds all the portfwds
	Portfwds = portfwds{
		forwards: map[int]*Portfwd{},
		mutex:    &sync.RWMutex{},
	}

	portfwdID = 0
)

const (
	portfwdBindTimeout  = 5 * time.Second
	portfwdCloseTimeout = 5 * time.Second
)

// PortfwdMeta - Metadata about a portfwd listener
type PortfwdMeta struct {
	ID         int
	SessionID  string
	BindAddr   string
	RemoteAddr string
}

// Portfwd - Tracks portfwd<->tcpproxy
type Portfwd struct {
	ID           int
	TCPProxy     *tcpproxy.Proxy
	ChannelProxy *ChannelProxy
}

// GetMetadata - Get metadata about the portfwd
func (p *Portfwd) GetMetadata() *PortfwdMeta {
	return &PortfwdMeta{
		ID:         p.ID,
		SessionID:  p.ChannelProxy.Session.ID,
		BindAddr:   p.ChannelProxy.BindAddr,
		RemoteAddr: p.ChannelProxy.RemoteAddr,
	}
}

type portfwds struct {
	forwards map[int]*Portfwd
	mutex    *sync.RWMutex
}

// Add - Add a TCP proxy instance
func (f *portfwds) Add(tcpProxy *tcpproxy.Proxy, channelProxy *ChannelProxy) *Portfwd {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	portfwd := &Portfwd{
		ID:           nextPortfwdID(),
		TCPProxy:     tcpProxy,
		ChannelProxy: channelProxy,
	}
	f.forwards[portfwd.ID] = portfwd
	return portfwd
}

// Remove - Remove a TCP proxy instance
func (f *portfwds) Remove(portfwdID int) bool {
	f.mutex.Lock()
	portfwd, ok := f.forwards[portfwdID]
	if ok {
		delete(f.forwards, portfwdID)
	}
	f.mutex.Unlock()
	if !ok {
		return false
	}
	// Remove registry ownership before joining connection workers. A slow or
	// broken peer must not hold the global inventory mutex and block unrelated
	// proxy lifecycle operations.
	portfwd.ChannelProxy.Stop()
	_ = portfwd.TCPProxy.Close()
	return true
}

// List - List all TCP proxy instances
func (f *portfwds) List() []*PortfwdMeta {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	portForwards := []*PortfwdMeta{}
	for _, portfwd := range f.forwards {
		portForwards = append(portForwards, portfwd.GetMetadata())
	}
	return portForwards
}

// ChannelProxy binds the Sliver Tunnel to a net.Conn object
// one ChannelProxy per port bind.
//
// Implements the Target interface from tcpproxy pkg
type ChannelProxy struct {
	Rpc     rpcpb.SliverRPCClient
	Session *clientpb.Session

	BindAddr        string
	RemoteAddr      string
	KeepAlivePeriod time.Duration
	DialTimeout     time.Duration

	connectionsMutex sync.Mutex
	connections      map[net.Conn]struct{}
	lifecycleCtx     context.Context
	lifecycleCancel  context.CancelFunc
	stopped          bool
	handlerWG        sync.WaitGroup
}

// HandleConn - Handle a TCP connection
func (p *ChannelProxy) HandleConn(conn net.Conn) {
	log.Printf("[tcpproxy] Handling new connection")
	if conn == nil {
		return
	}
	ctx, tracked := p.trackConnection(conn)
	if !tracked {
		_ = conn.Close()
		return
	}
	defer p.handlerWG.Done()
	defer p.untrackConnection(conn)
	defer func() { _ = conn.Close() }()

	var cancel context.CancelFunc
	if p.DialTimeout >= 0 {
		ctx, cancel = context.WithTimeout(ctx, p.dialTimeout())
		defer cancel()
	}
	tunnel, err := p.dialImplant(ctx)
	if err != nil {
		return
	}

	errs := make(chan error, 2)
	copyWG := sync.WaitGroup{}
	copyWG.Add(2)
	go func() {
		defer copyWG.Done()
		errs <- toImplantLoop(conn, tunnel)
	}()
	go func() {
		defer copyWG.Done()
		errs <- fromImplantLoop(conn, tunnel)
	}()

	// The first completed direction or a receive-admission failure owns shutdown.
	// Closing both endpoints wakes the peer direction; join it before allowing
	// this handler to finish.
	select {
	case err = <-errs:
	case <-tunnel.receiveFailed():
		// A receive-overflow close normally allows queued final frames to
		// drain. That is unsafe for a port forward whose local peer stopped
		// reading: close the accepted socket so both copy loops wake and the
		// bounded remote cleanup below can run.
		_ = conn.Close()
		err = <-errs
	}
	if err != nil {
		log.Printf("[tcpproxy] Closing tunnel %d with error %s", tunnel.ID, err)
	}
	_ = conn.Close()
	GetTunnels().CloseIf(tunnel)
	copyWG.Wait()
	if err := p.closeTunnel(tunnel); err != nil {
		log.Printf("[tcpproxy] Failed to close tunnel %d: %v", tunnel.ID, err)
	}
}

// Stop closes every connection accepted by this target and rejects handlers
// that race the listener shutdown performed by Portfwds.Remove.
func (p *ChannelProxy) Stop() {
	p.connectionsMutex.Lock()
	p.stopped = true
	lifecycleCancel := p.lifecycleCancel
	connections := make([]net.Conn, 0, len(p.connections))
	for connection := range p.connections {
		connections = append(connections, connection)
	}
	p.connectionsMutex.Unlock()

	if lifecycleCancel != nil {
		lifecycleCancel()
	}
	for _, connection := range connections {
		_ = connection.Close()
	}
	p.handlerWG.Wait()
}

func (p *ChannelProxy) trackConnection(connection net.Conn) (context.Context, bool) {
	p.connectionsMutex.Lock()
	defer p.connectionsMutex.Unlock()
	if p.stopped {
		return nil, false
	}
	if p.lifecycleCtx == nil {
		p.lifecycleCtx, p.lifecycleCancel = context.WithCancel(context.Background())
	}
	if p.connections == nil {
		p.connections = map[net.Conn]struct{}{}
	}
	// Add is serialized with Stop's stopped transition. Once Stop releases this
	// mutex, no later Add can race its Wait.
	p.handlerWG.Add(1)
	p.connections[connection] = struct{}{}
	return p.lifecycleCtx, true
}

func (p *ChannelProxy) untrackConnection(connection net.Conn) {
	p.connectionsMutex.Lock()
	delete(p.connections, connection)
	p.connectionsMutex.Unlock()
}

// HostPort returns the remote host and port of the TCP proxy. Invalid values
// retain the historical empty-host/8080 fallback for source and behavioral
// compatibility. New call sites that must reject malformed destinations should
// use ValidatedHostPort.
func (p *ChannelProxy) HostPort() (string, uint32) {
	defaultPort := uint32(8080)
	host, rawPort, err := net.SplitHostPort(p.RemoteAddr)
	if err != nil {
		log.Printf("Failed to parse addr %s", p.RemoteAddr)
		return "", defaultPort
	}
	portNumber, err := strconv.Atoi(rawPort)
	if err != nil {
		log.Printf("Failed to parse number from %s", rawPort)
		return "", defaultPort
	}
	port := uint32(portNumber)
	if port < 1 || 65535 < port {
		log.Printf("Invalid port number %d", port)
		return "", defaultPort
	}
	return host, port
}

// ValidatedHostPort validates and returns the configured remote destination.
func (p *ChannelProxy) ValidatedHostPort() (string, uint32, error) {
	host, rawPort, err := net.SplitHostPort(p.RemoteAddr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid remote target %q: %w", p.RemoteAddr, err)
	}
	if host == "" {
		return "", 0, fmt.Errorf("invalid remote target %q: host is required", p.RemoteAddr)
	}
	if strings.IndexFunc(host, func(value rune) bool {
		return unicode.IsSpace(value) || unicode.IsControl(value)
	}) >= 0 {
		return "", 0, fmt.Errorf("invalid remote target %q: host contains whitespace or control characters", p.RemoteAddr)
	}
	portNumber, err := strconv.ParseUint(rawPort, 10, 16)
	if err != nil {
		return "", 0, fmt.Errorf("invalid remote target %q: port must be a number from 1 to 65535", p.RemoteAddr)
	}
	if portNumber == 0 {
		return "", 0, fmt.Errorf("invalid remote target %q: port must be a number from 1 to 65535", p.RemoteAddr)
	}
	return host, uint32(portNumber), nil
}

// Port - Returns the TCP port of the proxy
func (p *ChannelProxy) Port() uint32 {
	_, port := p.HostPort()
	return port
}

// Host - Returns the host (i.e., interface) of the TCP proxy
func (p *ChannelProxy) Host() string {
	host, _ := p.HostPort()
	return host
}

func (p *ChannelProxy) dialImplant(ctx context.Context) (_ *TunnelIO, resultErr error) {
	host, port, err := p.ValidatedHostPort()
	if err != nil {
		return nil, err
	}

	log.Printf("[tcpproxy] Dialing implant to create tunnel ...")

	// Create an RPC tunnel, then start it before binding the shell to the newly created tunnel
	rpcTunnel, err := p.Rpc.CreateTunnel(ctx, &sliverpb.Tunnel{
		SessionID: p.Session.ID,
	})
	if err != nil {
		log.Printf("[tcpproxy] Failed to dial implant %s", err)
		return nil, err
	}

	log.Printf("[tcpproxy] Created new tunnel with id %d (session %s)", rpcTunnel.TunnelID, p.Session.ID)
	tunnel := GetTunnels().Start(rpcTunnel.TunnelID, rpcTunnel.SessionID)
	setupComplete := false
	defer func() {
		if !setupComplete {
			resultErr = errors.Join(resultErr, p.closeTunnel(tunnel))
		}
	}()

	bindTimer := time.NewTimer(portfwdBindTimeout)
	defer bindTimer.Stop()
	select {
	case <-tunnel.Bound():
	case <-tunnel.Done():
		return nil, errors.New("port forward tunnel closed before it was bound")
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-bindTimer.C:
		return nil, errors.New("timed out binding port forward tunnel")
	}

	log.Printf("[tcpproxy] Binding tunnel to portfwd %d", port)
	portfwdResp, err := p.Rpc.Portfwd(ctx, &sliverpb.PortfwdReq{
		Request: &commonpb.Request{
			SessionID: p.Session.ID,
		},
		Host:      host,
		Port:      port,
		Protocol:  sliverpb.PortFwdProtoTCP,
		TunnelID:  tunnel.ID,
		KeepAlive: int32(p.KeepAlivePeriod.Seconds()),
	})
	if err != nil {
		return nil, err
	}
	// Close tunnel in case of error on the implant side
	if portfwdResp.Response != nil && portfwdResp.Response.Err != "" {
		return nil, errors.New(portfwdResp.Response.Err)
	}
	log.Printf("Portfwd response: %v", portfwdResp)
	select {
	case <-tunnel.Done():
		return nil, errors.New("port forward tunnel closed during setup")
	default:
	}

	setupComplete = true
	return tunnel, nil
}

func (p *ChannelProxy) closeTunnel(tunnel *TunnelIO) error {
	if tunnel == nil {
		return nil
	}
	GetTunnels().CloseIf(tunnel)
	if p.Rpc == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), portfwdCloseTimeout)
	defer cancel()
	_, err := p.Rpc.CloseTunnel(ctx, &sliverpb.Tunnel{
		TunnelID:  tunnel.ID,
		SessionID: tunnel.SessionID,
	})
	return err
}

// func (p *ChannelProxy) keepAlivePeriod() time.Duration {
// 	if p.KeepAlivePeriod != 0 {
// 		return p.KeepAlivePeriod
// 	}
// 	return time.Minute
// }

func (p *ChannelProxy) dialTimeout() time.Duration {
	if p.DialTimeout > 0 {
		return p.DialTimeout
	}
	return 30 * time.Second
}

func toImplantLoop(conn net.Conn, tunnel *TunnelIO) error {
	if wc, ok := conn.(*tcpproxy.Conn); ok && len(wc.Peeked) > 0 {
		if _, err := tunnel.Write(wc.Peeked); err != nil {
			return err
		}
		wc.Peeked = nil
	}
	n, err := io.Copy(tunnel, conn)
	log.Printf("[tcpproxy] Closing to-implant after %d byte(s)", n)
	return err
}

func fromImplantLoop(conn net.Conn, tunnel *TunnelIO) error {
	n, err := io.Copy(conn, tunnel)
	log.Printf("[tcpproxy] Closing from-implant after %d byte(s)", n)
	return err
}

func nextPortfwdID() int {
	portfwdID++
	return portfwdID
}
