package rpc

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
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
	"github.com/golang/protobuf/proto"
	"net"
	"sync"
	"time"
)

var (
	fromImplantCacheSocks = map[uint64]map[uint64]*sliverpb.SocksData{}
	socksLog              = log.NamedLogger("c2", "socks")
)
var (
	// SocksProxys - Struct instance that holds all the socks
	SocksProxys = socksProxy{
		tcpProxys: map[uint32]*SocksProxy{},
		mutex:     &sync.RWMutex{},
	}
	SocksProxyID uint32 = 0
)

type socksProxy struct {
	tcpProxys map[uint32]*SocksProxy
	mutex     *sync.RWMutex
}

// SocksProxy
type SocksProxy struct {
	ID           uint32
	ChannelProxy *TcpProxy
}
type TcpProxy struct {
	SessionID       string
	Username        string
	Password        string
	BindAddr        string
	Listener        net.Listener
	stopChan        bool
	KeepAlivePeriod time.Duration
	DialTimeout     time.Duration
}

func nextSocksProxyID() uint32 {
	SocksProxyID++
	return SocksProxyID
}

// Add - Add a TCP proxy instance
func (f *socksProxy) Add(tcpProxy *TcpProxy) *SocksProxy {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	Sockser := &SocksProxy{
		ID:           nextSocksProxyID(),
		ChannelProxy: tcpProxy,
	}
	f.tcpProxys[Sockser.ID] = Sockser

	return Sockser
}

// Remove - Remove a TCP proxy instance
func (f *socksProxy) Remove(socksId uint32) bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if _, ok := f.tcpProxys[socksId]; ok {
		f.tcpProxys[socksId].ChannelProxy.stopChan = true
		f.tcpProxys[socksId].ChannelProxy.Listener.Close()
		delete(f.tcpProxys, socksId)
		return true
	}
	return false
}

// List - List all TCP proxy instances
func (f *socksProxy) List() *sliverpb.ListSocks {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	socksProxy := &sliverpb.ListSocks{}
	for _, socks := range f.tcpProxys {
		socksProxy.List = append(socksProxy.List, &sliverpb.SocksInfo{
			BindAddr: socks.ChannelProxy.BindAddr,
			Username: socks.ChannelProxy.Username,
			Password: socks.ChannelProxy.Password,
			Request:  &commonpb.Request{SessionID: socks.ChannelProxy.SessionID},
			ID:       socks.ID,
		})
	}
	return socksProxy
}

func (s *Server) ListSocks(ctx context.Context, in *commonpb.Empty) (*sliverpb.ListSocks, error) {
	return SocksProxys.List(), nil
}

func (rpc *Server) CreateSocks(ctx context.Context, req *sliverpb.SocksInfo) (*sliverpb.SocksInfo, error) {
	bindAddr := fmt.Sprintf("%s:%s", req.Host, req.Port)
	ln, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return nil, err
	}
	// Start Listener
	tp := &TcpProxy{
		SessionID: req.Request.SessionID,
		Username:  req.Username,
		Password:  req.Password,
		BindAddr:  bindAddr,
		Listener:  ln,
	}
	socks := SocksProxys.Add(tp)
	go SocksTransform(req, tp)

	return &sliverpb.SocksInfo{
		ID: socks.ID,
	}, nil
}

func SocksTransform(req *sliverpb.SocksInfo, tp *TcpProxy) {
	session := core.Sessions.Get(req.Request.SessionID)
	mux := sync.RWMutex{}
	for !tp.stopChan {
		conn, err := tp.Listener.Accept()
		if err != nil {
			tp.Listener.Close()
			return
		}
		mux.Lock()

		tunnel := core.SocksTunnels.Create(session.ID)
		if tunnel == nil {
			return
		}
		fromImplantCacheSocks[tunnel.ID] = map[uint64]*sliverpb.SocksData{}
		//fromTunnelPool[tunnel.ID] = &l
		req.TunnelID = tunnel.ID

		// data verification
		s := &Setting{
			Username: tp.Username,
			Password: tp.Password,
			tcpConn:  conn,
		}
		if s.checkAuth() {
			socksLog.Debugf("[socks] New TunnelID (%d) %x tcp conn %q<-->%q \n", req.TunnelID, conn, conn.LocalAddr(), conn.RemoteAddr())
			go handlerSocks(tunnel.ID, conn, session)
		} else {
			core.SocksTunnels.Close(tunnel.ID)
		}

		mux.Unlock()
	}
	socksLog.Printf("\n Socks Stop -> %s\n", tp.BindAddr)
	tp.Listener.Close()
}

func (s *Server) CloseSocks(ctx context.Context, req *sliverpb.SocksInfo) (*commonpb.Empty, error) {
	status := SocksProxys.Remove(req.ID)
	if status {
		return &commonpb.Empty{}, nil
	}
	return nil, fmt.Errorf("Not Found")
}

func (s *Setting) checkAuth() bool {
	buffer := make([]byte, 20480)
	length, err := s.tcpConn.Read(buffer)
	if err != nil {
		s.tcpConn.Close()
		return false
	}
	if length > 0 {
		s.checkMethod(s.tcpConn, buffer[:length])
	}

	if !s.isAuthed && s.method == "PASSWORD" {
		buffer := make([]byte, 20480)
		length, err := s.tcpConn.Read(buffer)
		if err != nil {
			s.tcpConn.Close()
			return false
		}
		s.auth(buffer[:length])
	}
	if s.isAuthed && !s.tcpConnected {
		return true
	}
	return false
}

type Setting struct {
	method       string
	Username     string
	Password     string
	isAuthed     bool
	tcpConnected bool
	success      bool
	tcpConn      net.Conn
}

func (s *Setting) checkMethod(conn net.Conn, data []byte) {
	failMess := []byte{0x05, 0xff}

	noneMess := []byte{0x05, 0x00}

	passMess := []byte{0x05, 0x02}

	// avoid the scenario that we can get full socks protocol header (rarely happen,just in case)
	defer func() {
		if r := recover(); r != nil {
			s.isAuthed = false
		}
	}()

	if data[0] == 0x05 {
		nMethods := int(data[1])

		var supportMethodFinded, userPassFinded, noAuthFinded bool

		for _, method := range data[2 : 2+nMethods] {
			if method == 0x00 {
				noAuthFinded = true
				supportMethodFinded = true
			} else if method == 0x02 {
				userPassFinded = true
				supportMethodFinded = true
			}
		}

		if !supportMethodFinded {
			conn.Write(failMess)
			return
		}

		if noAuthFinded && (s.Username == "" && s.Password == "") {
			conn.Write(noneMess)
			s.method = "NONE"
			s.isAuthed = true
			return
		} else if userPassFinded && (s.Username != "" && s.Password != "") {
			conn.Write(passMess)
			s.method = "PASSWORD"
			return
		} else {
			conn.Write(failMess)
			s.method = "ILLEGAL"
			return
		}
	}
	// send nothing
	s.method = "ILLEGAL"
}

func (s *Setting) auth(data []byte) {

	failMess := []byte{0x01, 0x01}

	succMess := []byte{0x01, 0x00}

	defer func() {
		if r := recover(); r != nil {
			s.isAuthed = false
		}
	}()

	ulen := int(data[1])
	slen := int(data[2+ulen])
	clientName := string(data[2 : 2+ulen])
	clientPass := string(data[3+ulen : 3+ulen+slen])

	if clientName != s.Username || clientPass != s.Password {
		s.tcpConn.Write(failMess)
		s.isAuthed = false
		// auth fail
		socksLog.Warnf("[socks] socks auth fail IP: %s ,Username: %s,Password: %s", s.tcpConn.RemoteAddr(), clientName, clientPass)
		return
	}
	// username && password all fits!
	s.tcpConn.Write(succMess)
	s.isAuthed = true
}

func handlerSocks(tunnelID uint64, conn net.Conn, session *core.Session) {
	go func() {
		socks := core.SocksTunnels.Get(tunnelID)
		if socks == nil {
			return
		}
		for {
			if data, ok := <-socks.FromImplant; ok {
				// TCP
				socksLog.Debugf("[socks] agent to Server TunnelID (%d) first  %d Seq %d DataLen %d", data.TunnelID, socks.FromImplantSequence, data.Sequence, data.DataLen)

				if data.Sequence == 0 {
					conn.Write(data.Data)
					continue
				}
				socksLog.Debugf("[socks] agent to Server TunnelID (%d) fromImplantCacheSocks %d Seq %d DataLen %d", data.TunnelID, socks.FromImplantSequence, data.Sequence, data.DataLen)

				recvCache, ok := fromImplantCacheSocks[data.TunnelID]
				if ok {
					recvCache[data.Sequence] = data
				}
				socksLog.Debugf("[socks] agent to Server TunnelID (%d) recvCache %#v", data.TunnelID, recvCache)
				for {
					recv, ok := recvCache[socks.FromImplantSequence]
					if !ok {
						break
					}
					socksLog.Debugf("[socks] agent to Server TunnelID (%d) FromImplantSequence %d Seq %d DataLen %d", data.TunnelID, socks.FromImplantSequence, data.Sequence, data.DataLen)

					socks.FromImplantSequence++
					conn.Write(recv.Data)
					if recv.CloseConn {
						socksLog.Debugf("[socks] agent to Server Close TunnelID (%d) ", recv.TunnelID)
						conn.Close()
						core.SocksTunnels.Close(recv.TunnelID)
						delete(fromImplantCacheSocks, recv.TunnelID)
						return
					}
					delete(recvCache, socks.FromImplantSequence-1)
				}
			}
		}
	}()

	// handling data that comes from browser
	buffer := make([]byte, 20480)

	// try to receive first packet
	// avoid browser to close the conn but sends nothing
	length, err := conn.Read(buffer)
	if err != nil {
		conn.Close() // close conn immediately
		return
	}
	if length > 0 {
		frame := &sliverpb.SocksData{
			TunnelID: tunnelID,
			Data:     buffer[:length],
			//Sequence:toImplantSequence,
		}
		marshal, err := proto.Marshal(frame)
		if err != nil {

		}
		session.Connection.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgSocksData,
			Data: marshal,
		}

	}

	for {
		length, err := conn.Read(buffer)
		if err != nil {
			conn.Close() // close conn immediately
			return
		}
		if length > 0 {
			frame := &sliverpb.SocksData{
				TunnelID: tunnelID,
				Data:     buffer[:length],
			}
			marshal, err := proto.Marshal(frame)
			if err != nil {
				conn.Close()
				return
			}
			session.Connection.Send <- &sliverpb.Envelope{
				Type: sliverpb.MsgSocksData,
				Data: marshal,
			}
		}
	}
}
