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
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

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
	defer f.mutex.Unlock()
	if portfwd, ok := f.forwards[portfwdID]; ok {
		portfwd.TCPProxy.Close()
		delete(f.forwards, portfwdID)
		return true
	}
	return false
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
}

// HandleConn - Handle a TCP connection
func (p *ChannelProxy) HandleConn(conn net.Conn) {
	log.Printf("[tcpproxy] Handling new connection")
	ctx := context.Background()
	var cancel context.CancelFunc
	if p.DialTimeout >= 0 {
		ctx, cancel = context.WithTimeout(ctx, p.dialTimeout())
	}
	tunnel, err := p.dialImplant(ctx)
	if cancel != nil {
		defer cancel()
	}
	if err != nil {
		return
	}

	// Cleanup
	defer func() {
		conn.Close()
		GetTunnels().Close(tunnel.ID)
	}()

	errs := make(chan error, 1)
	go toImplantLoop(conn, tunnel, errs)
	go fromImplantLoop(conn, tunnel, errs)

	// Block until error, then cleanup
	err = <-errs
	if err != nil {
		log.Printf("[tcpproxy] Closing tunnel %d with error %s", tunnel.ID, err)
	}
}

// HostPort - Returns the host and port of the TCP proxy
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

func (p *ChannelProxy) dialImplant(ctx context.Context) (*TunnelIO, error) {

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

	log.Printf("[tcpproxy] Binding tunnel to portfwd %d", p.Port())
	portfwdResp, err := p.Rpc.Portfwd(ctx, &sliverpb.PortfwdReq{
		Request: &commonpb.Request{
			SessionID: p.Session.ID,
		},
		Host:     p.Host(),
		Port:     p.Port(),
		Protocol: sliverpb.PortFwdProtoTCP,
		TunnelID: tunnel.ID,
	})
	if err != nil {
		return nil, err
	}
	// Close tunnel in case of error on the implant side
	if portfwdResp.Response != nil && portfwdResp.Response.Err != "" {
		p.Rpc.CloseTunnel(ctx, &sliverpb.Tunnel{
			TunnelID:  tunnel.ID,
			SessionID: p.Session.ID,
		})
		return nil, errors.New(portfwdResp.Response.Err)
	}
	log.Printf("Portfwd response: %v", portfwdResp)

	return tunnel, nil
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

func toImplantLoop(conn net.Conn, tunnel *TunnelIO, errs chan<- error) {
	if wc, ok := conn.(*tcpproxy.Conn); ok && len(wc.Peeked) > 0 {
		if _, err := tunnel.Write(wc.Peeked); err != nil {
			errs <- err
			return
		}
		wc.Peeked = nil
	}
	n, err := io.Copy(tunnel, conn)
	log.Printf("[tcpproxy] Closing to-implant after %d byte(s)", n)
	errs <- err
}

func fromImplantLoop(conn net.Conn, tunnel *TunnelIO, errs chan<- error) {
	n, err := io.Copy(conn, tunnel)
	log.Printf("[tcpproxy] Closing from-implant after %d byte(s)", n)
	errs <- err
}

func nextPortfwdID() int {
	portfwdID++
	return portfwdID
}
