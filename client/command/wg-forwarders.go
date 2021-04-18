package command

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

func wgPortFwdAddCmd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	if session.Transport != "wg" {
		fmt.Println(Warn + "This command is only supported for WireGuard implants")
		return
	}

	localPort := ctx.Flags.Int("bind")
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

	pfwdAdd, err := rpc.WGStartPortForward(context.Background(), &sliverpb.WGPortForwardStartReq{
		LocalPort:     int32(localPort),
		RemoteAddress: remoteAddr,
		Request:       ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if pfwdAdd.Response != nil && pfwdAdd.Response.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", pfwdAdd.Response.Err)
		return
	}
	fmt.Printf(Info+"Port forwarding %s -> %s:%s\n", pfwdAdd.Forwarder.LocalAddr, remoteHost, remotePort)
}

func wgPortFwdListCmd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	if session.Transport != "wg" {
		fmt.Println(Warn + "This command is only supported for WireGuard implants")
		return
	}

	fwdList, err := rpc.WGListForwarders(context.Background(), &sliverpb.WGTCPForwardersReq{
		Request: ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if fwdList.Response != nil && fwdList.Response.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", fwdList.Response.Err)
		return
	}

	if fwdList.Forwarders != nil {
		if len(fwdList.Forwarders) == 0 {
			fmt.Printf(Info + "No port forwards\n")
		} else {
			outBuf := bytes.NewBufferString("")
			table := tabwriter.NewWriter(outBuf, 0, 3, 3, ' ', 0)
			fmt.Fprintf(table, "ID\tLocal Address\tRemote Address\t\n")
			fmt.Fprintf(table, "%s\t%s\t%s\t\n",
				strings.Repeat("=", len("ID")),
				strings.Repeat("=", len("Local Address")),
				strings.Repeat("=", len("Remote Address")))
			for _, fwd := range fwdList.Forwarders {
				fmt.Fprintf(table, "%d\t%s\t%s\t\n", fwd.ID, fwd.LocalAddr, fwd.RemoteAddr)
			}
			table.Flush()
			fmt.Println(outBuf.String())
		}
	}
}

func wgPortFwdRmCmd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	if session.Transport != "wg" {
		fmt.Println(Warn + "This command is only supported for WireGuard implants")
		return
	}

	if len(ctx.Args) <= 0 {
		fmt.Println(Warn + "you must provide a rule identifier")
		return
	}
	idStr := ctx.Args[0]
	fwdID, err := strconv.Atoi(idStr)
	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if fwdID == -1 {
		return
	}

	stopReq, err := rpc.WGStopPortForward(context.Background(), &sliverpb.WGPortForwardStopReq{
		ID:      int32(fwdID),
		Request: ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if stopReq.Response != nil && stopReq.Response.Err != "" {
		fmt.Printf(Warn+"Error: %v\n", stopReq.Response.Err)
		return
	}

	if stopReq.Forwarder != nil {
		fmt.Printf(Info+"Removed port forwarding rule %s -> %s\n", stopReq.Forwarder.LocalAddr, stopReq.Forwarder.RemoteAddr)
	}
}

func wgSocksStartCmd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	if session.Transport != "wg" {
		fmt.Println(Warn + "This command is only supported for Wireguard implants")
		return
	}

	bindPort := ctx.Flags.Int("bind")

	socks, err := rpc.WGStartSocks(context.Background(), &sliverpb.WGSocksStartReq{
		Port:    int32(bindPort),
		Request: ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if socks.Response != nil && socks.Response.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", err)
		return
	}

	if socks.Server != nil {
		fmt.Printf(Info+"Started SOCKS server on %s\n", socks.Server.LocalAddr)
	}
}

func wgSocksListCmd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	if session.Transport != "wg" {
		fmt.Println(Warn + "This command is only supported for WireGuard implants")
		return
	}

	socksList, err := rpc.WGListSocksServers(context.Background(), &sliverpb.WGSocksServersReq{
		Request: ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if socksList.Response != nil && socksList.Response.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", socksList.Response.Err)
		return
	}

	if socksList.Servers != nil {
		if len(socksList.Servers) > 0 {
			outBuf := bytes.NewBufferString("")
			table := tabwriter.NewWriter(outBuf, 0, 3, 3, ' ', 0)
			fmt.Fprintf(table, "ID\tLocal Address\n")
			fmt.Fprintf(table, "%s\t%s\t\n",
				strings.Repeat("=", len("ID")),
				strings.Repeat("=", len("Local Address")))
			for _, server := range socksList.Servers {
				fmt.Fprintf(table, "%d\t%s\t\n", server.ID, server.LocalAddr)
			}
			table.Flush()
			fmt.Println(outBuf.String())
		}
	}

}

func wgSocksRmCmd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}
	if session.Transport != "wg" {
		fmt.Println(Warn + "This command is only supported for WireGuard implants")
		return
	}

	if len(ctx.Args) <= 0 {
		fmt.Println(Warn + "you must provide a listener identifier")
		return
	}
	idStr := ctx.Args[0]
	socksID, err := strconv.Atoi(idStr)
	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if socksID == -1 {
		return
	}

	stopReq, err := rpc.WGStopSocks(context.Background(), &sliverpb.WGSocksStopReq{
		ID:      int32(socksID),
		Request: ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if stopReq.Response != nil && stopReq.Response.Err != "" {
		fmt.Printf(Warn+"Error: %v\n", stopReq.Response.Err)
		return
	}

	if stopReq.Server != nil {
		fmt.Printf(Info+"Removed socks listener rule %s \n", stopReq.Server.LocalAddr)
	}
}
