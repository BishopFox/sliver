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

	// Closing all connections in pool
	SocksConnPool.Range(func(key, value interface{}) bool {
		con := value.(net.Conn)

		con.Close()

		return err == nil
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
	proxy, err := tcpProxy.Rpc.SocksProxy(ctx)
	defer cancel()

	if err != nil {
		return err
	}
	go func() {
		for {
			FromImplantSequence := 0
			socksData, err := proxy.Recv()
			if err != nil {
				log.Printf("Failed to Recv from proxy, %s\n", err)
				return
			}

			if v, ok := SocksConnPool.Load(socksData.TunnelID); ok {
				conn := v.(net.Conn)

				if socksData.CloseConn {
					conn.Close()
					SocksConnPool.Delete(socksData.TunnelID)
					continue
				}
				log.Printf("[socks] agent to Server To (Client to User) Data Sequence %d , Data Size %d \n", FromImplantSequence, len(socksData.Data))
				//fmt.Printf("recv data len %d \n", len(p.Data))
				_, err := conn.Write(socksData.Data)
				if err != nil {
					log.Printf("Failed to write data to proxy connection, %s\n", err)
					continue
				}
				FromImplantSequence++
			}
		}
	}()
	for {
		connection, err := tcpProxy.Listener.Accept()
		if err != nil {
			log.Printf("Failed to accept new listener, probably already closed: %s\n", err)
			break
		}
		rpcSocks, err := tcpProxy.Rpc.CreateSocks(context.Background(), &sliverpb.Socks{
			SessionID: tcpProxy.Session.ID,
		})
		if err != nil {
			log.Printf("Failed rcp call to create socks %s\n", err)
			break
		}

		go connect(connection, proxy, &sliverpb.SocksData{
			Username: tcpProxy.Username,
			Password: tcpProxy.Password,
			TunnelID: rpcSocks.TunnelID,
			Request:  &commonpb.Request{SessionID: rpcSocks.SessionID},
		})
	}
	log.Printf("Socks Stop -> %s\n", tcpProxy.BindAddr)
	tcpProxy.Stop() // well, at this moment we already in stop state, but anyway
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

const leakyBufSize = 4108 // data.len(2) + hmacsha1(10) + data(4096)

var leakyBuf = leaky.NewLeakyBuf(2048, leakyBufSize)

func connect(conn net.Conn, stream rpcpb.SliverRPC_SocksProxyClient, frame *sliverpb.SocksData) {

	SocksConnPool.Store(frame.TunnelID, conn)

	defer func() {
		// It's neccessary to close and remove connection once we done with it
		c, ok := SocksConnPool.LoadAndDelete(frame.TunnelID)
		if !ok {
			return
		}
		conn := c.(net.Conn)

		conn.Close()

		log.Printf("[socks] connection closed")
	}()

	log.Printf("tcp conn %q<--><-->%q \n", conn.LocalAddr(), conn.RemoteAddr())

	buff := leakyBuf.Get()
	defer leakyBuf.Put(buff)
	var ToImplantSequence uint64 = 0
	for {
		n, err := conn.Read(buff)

		if err != nil {
			log.Printf("[socks] (User to Client) failed to read data, %s ", err)
			// Error basically means that the connection is closed(EOF) OR deadline exceeded
			// In any of that cases, it's better to just giveup
			return
		}
		if n > 0 {
			frame.Data = buff[:n]
			frame.Sequence = ToImplantSequence
			log.Printf("[socks] (User to Client) to Server to agent  Data Sequence %d , Data Size %d \n", ToImplantSequence, len(frame.Data))
			err := stream.Send(frame)
			if err != nil {
				log.Printf("[socks] (User to Client) failed to send data, %s ", err)
				return
			}
			ToImplantSequence++

		}

	}
}
