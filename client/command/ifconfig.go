package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"

	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
)

func ifconfig(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	data, _ := proto.Marshal(&sliverpb.IfconfigReq{SliverID: ActiveSliver.Sliver.ID})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgIfconfigReq,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}

	ifaceConfigs := &sliverpb.Ifconfig{}
	err := proto.Unmarshal(resp.Data, ifaceConfigs)
	if err != nil {
		fmt.Printf(Warn + "Failed to decode response\n")
		return
	}

	for ifaceIndex, iface := range ifaceConfigs.NetInterfaces {
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
