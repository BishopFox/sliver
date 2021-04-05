package command

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/tcpproxy"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

func portfwd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	fmt.Printf("Port Forwards\n")
	// TODO
}

func portfwdAdd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	if session.GetActiveC2() == "dns" {
		fmt.Printf(Warn + "Current C2 is DNS, this is going to be a very slow tunnel!\n")
	}
	remoteAddr := ctx.Flags.String("remote")
	if remoteAddr == "" {
		fmt.Println(Warn + "Must specify a remote target host:port")
		return
	}
	remoteHost, remotePort, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		fmt.Print(Warn+"Failed to parse remote target %s\n", err)
		return
	}
	fmt.Printf(Info+"Port forwarding -> %s:%s\n", remoteHost, remotePort)

	proxy := tcpproxy.Proxy{}
	proxy.AddRoute("127.0.0.1:8080", &ChannelProxy{
		rpc:             rpc,
		session:         session,
		Addr:            remoteAddr,
		KeepAlivePeriod: 60 * time.Second,
		DialTimeout:     30 * time.Second,
	})

	go func() {
		err := proxy.Run()
		if err != nil {
			fmt.Printf("\r\n"+Warn+"Proxy error %s\n", err)
		}
	}()

	fmt.Println(Info + "Started proxy!")
}

func portfwdRm(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	// TODO
}

// ChannelProxy binds the Sliver Tunnel to a net.Conn object
// one ChannelProxy per port bind.
//
// Implements the Target interface from tcpproxy pkg
type ChannelProxy struct {
	rpc     rpcpb.SliverRPCClient
	session *clientpb.Session

	Addr            string
	KeepAlivePeriod time.Duration
	DialTimeout     time.Duration
}

func (p *ChannelProxy) HandleConn(conn net.Conn) {
	log.Printf("[tcpproxy] Handling new connection")
	ctx := context.Background()
	var cancel context.CancelFunc
	if p.DialTimeout >= 0 {
		ctx, cancel = context.WithTimeout(ctx, p.dialTimeout())
	}
	tunnel, err := p.dialImplant(ctx)
	if cancel != nil {
		cancel()
	}
	if err != nil {
		return
	}

	// Cleanup
	defer func() { go conn.Close() }()

	errs := make(chan error, 1)
	go toImplantLoop(conn, tunnel, errs)
	go fromImplantLoop(conn, tunnel, errs)

	// Block until error, then cleanup
	err = <-errs
	if err != nil {
		log.Printf("[tcpproxy] Closing tunnel %d with error %s", tunnel.ID, err)
	}
}

func (p *ChannelProxy) HostPort() (string, uint32) {
	defaultPort := uint32(8080)
	host, rawPort, err := net.SplitHostPort(p.Addr)
	if err != nil {
		log.Printf("Failed to parse addr %s", p.Addr)
		return "", defaultPort
	}
	portNumber, err := strconv.Atoi(rawPort)
	if err != nil {
		log.Printf("Failed to parse number from %s", rawPort)
		return "", defaultPort
	}
	port := uint32(portNumber)
	if port < 1 || 65535 < port {
		log.Printf("Invalid port number %d", port)
		return "", defaultPort
	}
	return host, port
}

func (p *ChannelProxy) Port() uint32 {
	_, port := p.HostPort()
	return port
}

func (p *ChannelProxy) Host() string {
	host, _ := p.HostPort()
	return host
}

func (p *ChannelProxy) dialImplant(ctx context.Context) (*core.Tunnel, error) {

	log.Printf("[tcpproxy] Dialing implant to create tunnel ...")

	// Create an RPC tunnel, then start it before binding the shell to the newly created tunnel
	rpcTunnel, err := p.rpc.CreateTunnel(ctx, &sliverpb.Tunnel{
		SessionID: p.session.ID,
	})
	if err != nil {
		log.Printf("[tcpproxy] Failed to dial implant %s", err)
		return nil, err
	}
	log.Printf("[tcpproxy] Created new tunnel with id %d (session %d)", rpcTunnel.TunnelID, p.session.ID)
	tunnel := core.Tunnels.Start(rpcTunnel.TunnelID, rpcTunnel.SessionID)

	log.Printf("[tcpproxy] Binding tunnel to portfwd %d", p.Port())
	portfwdResp, err := p.rpc.Portfwd(ctx, &sliverpb.PortfwdReq{
		Request: &commonpb.Request{
			SessionID: p.session.ID,
		},
		Host:     p.Host(),
		Port:     p.Port(),
		Protocol: sliverpb.PortfwdProtocol_TCP,
		TunnelID: tunnel.ID,
	})
	if err != nil {
		return nil, err
	}
	log.Printf("Portfwd response: %v", portfwdResp)

	return tunnel, nil
}

func (p *ChannelProxy) keepAlivePeriod() time.Duration {
	if p.KeepAlivePeriod != 0 {
		return p.KeepAlivePeriod
	}
	return time.Minute
}

func (p *ChannelProxy) dialTimeout() time.Duration {
	if p.DialTimeout > 0 {
		return p.DialTimeout
	}
	return 30 * time.Second
}

func toImplantLoop(conn net.Conn, tunnel *core.Tunnel, errs chan<- error) {
	n, err := io.Copy(tunnel, conn)
	log.Printf("[tcpproxy] Closing to-implant after %d byte(s)", n)
	errs <- err
}

func fromImplantLoop(conn net.Conn, tunnel *core.Tunnel, errs chan<- error) {
	n, err := io.Copy(conn, tunnel)
	log.Printf("[tcpproxy] Closing from-implant after %d byte(s)", n)
	errs <- err
}
