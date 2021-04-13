package command

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/tcpproxy"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/desertbit/grumble"
)

func portfwd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	portfwds := core.Portfwds.List()
	if len(portfwds) == 0 {
		fmt.Printf(Info + "No port forwards\n")
		return
	}
	sort.Slice(portfwds[:], func(i, j int) bool {
		return portfwds[i].ID < portfwds[j].ID
	})
	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "ID\tSession ID\tBind Address\tRemote Address\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("ID")),
		strings.Repeat("=", len("Session ID")),
		strings.Repeat("=", len("Bind Address")),
		strings.Repeat("=", len("Remote Address")),
	)
	for _, p := range portfwds {
		fmt.Fprintf(table, "%d\t%d\t%s\t%s\t\n", p.ID, p.SessionID, p.BindAddr, p.RemoteAddr)
	}
	table.Flush()
	fmt.Printf("%s", outputBuf.String())
}

func portfwdAdd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	if session.GetActiveC2() == "dns" {
		fmt.Printf(Warn + "Current C2 is DNS, this is going to be a very slow tunnel!\n")
	}
	if session.Transport == "wg" {
		fmt.Printf(Warn + "Current C2 is WireGuard, we recommend using the `wg-portfwd` command!\n")
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
	if remotePort == "3389" {
		fmt.Print(Warn + "RDP is unstable over tunneled portfwds, we recommend using WireGuard portfwds\n")
	}
	bindAddr := ctx.Flags.String("bind")
	if remoteAddr == "" {
		fmt.Println(Warn + "Must specify a bind target host:port")
		return
	}

	tcpProxy := &tcpproxy.Proxy{}
	channelProxy := &core.ChannelProxy{
		Rpc:             rpc,
		Session:         session,
		RemoteAddr:      remoteAddr,
		BindAddr:        bindAddr,
		KeepAlivePeriod: 60 * time.Second,
		DialTimeout:     30 * time.Second,
	}
	tcpProxy.AddRoute(bindAddr, channelProxy)
	core.Portfwds.Add(tcpProxy, channelProxy)

	go func() {
		err := tcpProxy.Run()
		if err != nil {
			// fmt.Printf("\r\n"+Warn+"Proxy error %s\n", err)
			log.Printf("Proxy error %s", err)
		}
	}()

	fmt.Printf(Info+"Port forwarding %s -> %s:%s\n", bindAddr, remoteHost, remotePort)
}

func portfwdRm(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	portfwdID := ctx.Flags.Int("id")
	if portfwdID < 1 {
		fmt.Println(Warn + "Must specify a valid portfwd id")
		return
	}
	found := core.Portfwds.Remove(portfwdID)
	if !found {
		fmt.Printf(Warn+"No portfwd with id %d\n", portfwdID)
	} else {
		fmt.Println(Info + "Removed portfwd")
	}
}
