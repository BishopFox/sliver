package core

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/leaky"
	"golang.org/x/time/rate"
)

var (
	// SocksProxies - Struct instance that holds all the portfwds
	SocksProxies = socksProxy{
		tcpProxies: map[uint64]*SocksProxy{},
		mutex:      &sync.RWMutex{},
	}
	SocksConnPool = sync.Map{}
	SocksProxyID  = (uint64)(0)
)

// PortfwdMeta - Metadata about a portfwd listener
type SocksProxyMeta struct {
	ID        uint64
	SessionID string
	BindAddr  string
	Username  string
	Password  string
}
type TcpProxy struct {
	Rpc     rpcpb.SliverRPCClient
	Session *clientpb.Session

	Username        string
	Password        string
	BindAddr        string
	Listener        net.Listener
	KeepAlivePeriod time.Duration
	DialTimeout     time.Duration
}

func (tcp *TcpProxy) Stop() error {
	err := tcp.Listener.Close()

	// Closing all connections in pool - always iterate all entries
	SocksConnPool.Range(func(key, value interface{}) bool {
		if con, ok := value.(net.Conn); ok {
			con.Close()
		}
		SocksConnPool.Delete(key)
		return true // always continue iteration
	})

	return err
}

// SocksProxy - Tracks portfwd<->tcpproxy
type SocksProxy struct {
	ID           uint64
	ChannelProxy *TcpProxy
}

// GetMetadata - Get metadata about the portfwd
func (p *SocksProxy) GetMetadata() *SocksProxyMeta {
	return &SocksProxyMeta{
		ID:        p.ID,
		SessionID: p.ChannelProxy.Session.ID,
		BindAddr:  p.ChannelProxy.BindAddr,
		Username:  p.ChannelProxy.Username,
		Password:  p.ChannelProxy.Password,
	}
}

type socksProxy struct {
	tcpProxies map[uint64]*SocksProxy
	mutex      *sync.RWMutex
}

// Add - Add a TCP proxy instance
func (f *socksProxy) Add(tcpProxy *TcpProxy) *SocksProxy {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	Sockser := &SocksProxy{
		ID:           nextSocksProxyID(),
		ChannelProxy: tcpProxy,
	}
	f.tcpProxies[Sockser.ID] = Sockser

	return Sockser
}

func (f *socksProxy) Start(tcpProxy *TcpProxy) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy, err := tcpProxy.Rpc.SocksProxy(ctx)
	if err != nil {
		return err
	}

	// Receiver goroutine: reads data from implant and writes to local connections
	go func() {
		for {
			socksData, err := proxy.Recv()
			if err != nil {
				log.Printf("[socks] recv stream closed: %s", err)
				return
			}

			if v, ok := SocksConnPool.Load(socksData.TunnelID); ok {
				conn := v.(net.Conn)

				if socksData.CloseConn {
					conn.Close()
					SocksConnPool.Delete(socksData.TunnelID)
					continue
				}
				_, err := conn.Write(socksData.Data)
				if err != nil {
					log.Printf("[socks] write error tunnel %d: %s", socksData.TunnelID, err)
					// Close the dead connection and notify
					conn.Close()
					SocksConnPool.Delete(socksData.TunnelID)
					continue
				}
			}
		}
	}()

	// Accept loop: accepts new SOCKS connections
	for {
		connection, err := tcpProxy.Listener.Accept()
		if err != nil {
			log.Printf("[socks] listener closed: %s", err)
			break
		}

		// Set TCP options on accepted connections for stability
		if tcpConn, ok := connection.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(60 * time.Second)
			tcpConn.SetNoDelay(true)
		}

		rpcSocks, err := tcpProxy.Rpc.CreateSocks(context.Background(), &sliverpb.Socks{
			SessionID: tcpProxy.Session.ID,
		})
		if err != nil {
			log.Printf("[socks] failed to create tunnel: %s", err)
			connection.Close()
			continue // Don't break - allow retry on next connection
		}

		go connect(connection, proxy, &sliverpb.SocksData{
			Username: tcpProxy.Username,
			Password: tcpProxy.Password,
			TunnelID: rpcSocks.TunnelID,
			Request:  &commonpb.Request{SessionID: rpcSocks.SessionID},
		})
	}
	log.Printf("[socks] proxy stopped: %s", tcpProxy.BindAddr)
	tcpProxy.Stop()
	proxy.CloseSend()
	return nil
}

// Remove - Remove a TCP proxy instance
func (f *socksProxy) Remove(socksId uint64) bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if _, ok := f.tcpProxies[socksId]; ok {
		f.tcpProxies[socksId].ChannelProxy.Stop()
		delete(f.tcpProxies, socksId)
		return true
	}
	return false
}

// List - List all TCP proxy instances
func (f *socksProxy) List() []*SocksProxyMeta {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	socksProxy := []*SocksProxyMeta{}
	for _, socks := range f.tcpProxies {
		socksProxy = append(socksProxy, socks.GetMetadata())
	}
	return socksProxy
}

func nextSocksProxyID() uint64 {
	return atomic.AddUint64(&SocksProxyID, 1)
}

// Increased buffer size from 4KB to 64KB for high-bandwidth protocols (RDP, etc.)
const leakyBufSize = 65546 // data.len(2) + hmacsha1(10) + data(65534)

var leakyBuf = leaky.NewLeakyBuf(2048, leakyBufSize)

func connect(conn net.Conn, stream rpcpb.SliverRPC_SocksProxyClient, frame *sliverpb.SocksData) {
	// Removed aggressive rate limiter (was 10 ops/s) - use generous limit for RDP/high-bandwidth
	limiter := rate.NewLimiter(rate.Limit(50000), 100)

	SocksConnPool.Store(frame.TunnelID, conn)

	// Set TCP keepalive on the connection to detect dead peers
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
		tcpConn.SetNoDelay(true) // Disable Nagle for lower latency
	}

	defer func() {
		c, ok := SocksConnPool.LoadAndDelete(frame.TunnelID)
		if !ok {
			return
		}
		conn := c.(net.Conn)
		conn.Close()
		// Send close notification to implant
		closeFrame := &sliverpb.SocksData{
			TunnelID:  frame.TunnelID,
			CloseConn: true,
			Request:   frame.Request,
			Username:  frame.Username,
			Password:  frame.Password,
		}
		stream.Send(closeFrame)
		log.Printf("[socks] connection closed for tunnel %d", frame.TunnelID)
	}()

	log.Printf("[socks] new connection %q <-> %q (tunnel %d)", conn.LocalAddr(), conn.RemoteAddr(), frame.TunnelID)

	buff := leakyBuf.Get()
	defer leakyBuf.Put(buff)
	var ToImplantSequence uint64 = 0
	for {
		n, err := conn.Read(buff)

		if err != nil {
			log.Printf("[socks] read error on tunnel %d: %s", frame.TunnelID, err)
			return
		}
		if n > 0 {
			if err := limiter.Wait(context.Background()); err != nil {
				log.Printf("[socks] rate limiter error: %s", err)
				return
			}

			data := make([]byte, n)
			copy(data, buff[:n])
			frame.Data = data
			frame.Sequence = ToImplantSequence
			err := stream.Send(frame)
			if err != nil {
				log.Printf("[socks] send error on tunnel %d: %s", frame.TunnelID, err)
				return
			}
			ToImplantSequence++
		}
	}
}
