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
	"fmt"
	"io"
	"log"
	"net"
	"sync"
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
		tcpProxies: map[int]*SocksProxy{},
		mutex:      &sync.RWMutex{},
	}
	SocksConnPool = sync.Map{}
	SocksProxyID  = 0
)

// PortfwdMeta - Metadata about a portfwd listener
type SocksProxyMeta struct {
	ID        int
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
	stopChan        bool
	KeepAlivePeriod time.Duration
	DialTimeout     time.Duration
}

// SocksProxy - Tracks portfwd<->tcpproxy
type SocksProxy struct {
	ID           int
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
	tcpProxies map[int]*SocksProxy
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
	proxy, err := tcpProxy.Rpc.SocksProxy(context.Background())
	if err != nil {
		return err
	}
	go func() {
		for !tcpProxy.stopChan {
			FromImplantSequence := 0
			p, err := proxy.Recv()
			if err != nil {
				return
			}

			if v, ok := SocksConnPool.Load(p.TunnelID); ok {
				n := v.(net.Conn)
				if p.CloseConn {
					n.Close()
					SocksConnPool.Delete(p.TunnelID)
					continue
				}
				log.Printf("[socks] agent to Server To (Client to User) Data Sequence %d , Data Size %d \n", FromImplantSequence, len(p.Data))
				//fmt.Printf("recv data len %d \n", len(p.Data))
				_, err := n.Write(p.Data)
				if err != nil {
					continue
				}
				FromImplantSequence++
			}
		}
	}()
	for !tcpProxy.stopChan {
		l, err := tcpProxy.Listener.Accept()
		if err != nil {
			return err
		}
		rpcSocks, err := tcpProxy.Rpc.CreateSocks(context.Background(), &sliverpb.Socks{
			SessionID: tcpProxy.Session.ID,
		})
		if err != nil {
			fmt.Println(err)
			return err
		}

		go connect(l, proxy, &sliverpb.SocksData{
			Username: tcpProxy.Username,
			Password: tcpProxy.Password,
			TunnelID: rpcSocks.TunnelID,
			Request:  &commonpb.Request{SessionID: rpcSocks.SessionID},
		})
	}
	log.Printf("Socks Stop -> %s\n", tcpProxy.BindAddr)
	tcpProxy.Listener.Close()
	proxy.CloseSend()
	return nil
}

// Remove - Remove a TCP proxy instance
func (f *socksProxy) Remove(socksId int) bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if _, ok := f.tcpProxies[socksId]; ok {
		f.tcpProxies[socksId].ChannelProxy.stopChan = true
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

func nextSocksProxyID() int {
	SocksProxyID++
	return SocksProxyID
}

const leakyBufSize = 4108 // data.len(2) + hmacsha1(10) + data(4096)

var leakyBuf = leaky.NewLeakyBuf(2048, leakyBufSize)

func connect(conn net.Conn, stream rpcpb.SliverRPC_SocksProxyClient, frame *sliverpb.SocksData) {

	SocksConnPool.Store(frame.TunnelID, conn)

	log.Printf("tcp conn %q<--><-->%q \n", conn.LocalAddr(), conn.RemoteAddr())

	buff := leakyBuf.Get()
	defer leakyBuf.Put(buff)
	var ToImplantSequence uint64 = 0
	for {
		n, err := conn.Read(buff)
		if err != nil {
			if err == io.EOF {
				return
			}
			continue
		}
		if n > 0 {
			frame.Data = buff[:n]
			frame.Sequence = ToImplantSequence
			log.Printf("[socks] (User to Client) to Server to agent  Data Sequence %d , Data Size %d \n", ToImplantSequence, len(frame.Data))
			err := stream.Send(frame)
			if err != nil {
				return
			}
			ToImplantSequence++

		}

	}
}
