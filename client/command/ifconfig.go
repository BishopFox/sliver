package command

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/desertbit/grumble"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func ifconfig(ctx *grumble.Context, rpc *rpcpb.SliverRPCClient) {
	if ActiveSesssion.Session == nil {
		fmt.Printf(Warn + "Please select an active session via `use`\n")
		return
	}

	ifconfig, err := rpc.Ifconfig(context.Background(), &sliverpb.IfconfigReq{
		Request: &commonpb.Request{
			SessionID: ActiveSesssion.Session.ID,
		},
	})

	for ifaceIndex, iface := range ifconfig.NetInterfaces {
		fmt.Printf("%s%s%s (%d)\n", bold, iface.Name, normal, ifaceIndex)
		if 0 < len(iface.MAC) {
			fmt.Printf("   MAC Address: %s\n", iface.MAC)
		}
		for _, ip := range iface.IPAddresses {

			// Try to find local IPs and colorize them
			subnet := -1
			if strings.Contains(ip, "/") {
				parts := strings.Split(ip, "/")
				subnetStr := parts[len(parts)-1]
				subnet, err = strconv.Atoi(subnetStr)
				if err != nil {
					subnet = -1
				}
			}

			if 0 < subnet && subnet <= 32 && !isLoopback(ip) {
				fmt.Printf(bold+green+"    IP Address: %s%s\n", ip, normal)
			} else if 32 < subnet && !isLoopback(ip) {
				fmt.Printf(bold+cyan+"    IP Address: %s%s\n", ip, normal)
			} else {
				fmt.Printf("    IP Address: %s\n", ip)
			}
		}
	}
}

func isLoopback(ip string) bool {
	if strings.HasPrefix(ip, "127") || strings.HasPrefix(ip, "::1") {
		return true
	}
	return false
}
