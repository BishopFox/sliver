package command

import (
	"context"
	"fmt"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"io"
	"log"
	"net"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

	"github.com/desertbit/grumble"
)

func socks(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	port := ctx.Flags.Int("port")
	go startSocksServer(ctx, port, rpc)
}

func startSocksServer(ctx *grumble.Context, port int, rpc rpcpb.SliverRPCClient) {
	fmt.Printf(Info + fmt.Sprintf("Starting socks server on port %d\n\n", port))
	session := ActiveSession.Get()
	if session == nil {
		fmt.Println("Current session is none")
		return
	}

	netAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	listener, err := net.ListenTCP("tcp4", netAddr)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	//defer listener.Close()

	fmt.Println("Waiting for new connections")

	for {
		fmt.Println("Waiting for new connections in a loop")
		conn, err := listener.AcceptTCP()
		fmt.Println("Got new connection")
		if err != nil {
			fmt.Printf(Warn+"%s\n", err)
			continue
		}
		go handleConnection(ctx, conn, rpc)
	}


}

func handleConnection(ctx *grumble.Context, conn *net.TCPConn, rpc rpcpb.SliverRPCClient) {
	fmt.Println("Handling new connection")

	socksConn := new(SocksConn)
	socksConn.ClientConn = conn

	// Negotiate authentication
	err := socksConn.HandleAuthRequest()
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}

	// Handle socks connection request
	err = socksConn.HandleConnectRequest()
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}

	fmt.Printf("Requested tcp tunnel to %s:%d\n", socksConn.RemoteHost, socksConn.RemotePort)

	session := ActiveSession.Get()
	if session == nil {
		return
	}

	// Create an RPC tunnel, then start it before binding the shell to the newly created tunnel
	rpcTunnel, err := rpc.CreateTunnel(context.Background(), &sliverpb.Tunnel{
		SessionID: session.ID,
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}
	fmt.Printf("Created new tunnel with id: %d, binding to tcptunnel ...\n", rpcTunnel.TunnelID)

	// Start() takes an RPC tunnel and creates a local Reader/Writer tunnel object
	tunnel := core.Tunnels.Start(rpcTunnel.TunnelID, rpcTunnel.SessionID)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}
	log.Printf("Bound remote tcp tunnel to tunnel %d", tunnel.ID)

	tcpTunnel, err := rpc.TCPTunnel(context.Background(), &sliverpb.TCPTunnelReq{
		RemoteHost : socksConn.RemoteHost,
		RemotePort : socksConn.RemotePort,
		Request:   ActiveSession.Request(ctx),
		TunnelID:  tunnel.ID,
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	if tcpTunnel.IsSuccess {
		socksConn.ReturnSuccessConnectMessage()
		fmt.Println("Successfully opened tunnel to implant")
		go io.Copy(tunnel, socksConn.ClientConn)
		go io.Copy(socksConn.ClientConn, tunnel)
	} else {
		socksConn.ReturnFailureConnectMessage()
		fmt.Println("Could not open tunnel to implant")
	}
}