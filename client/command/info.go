package command

import (
	"fmt"
	consts "github.com/bishopfox/sliver/client/constants"
	clientpb "github.com/bishopfox/sliver/protobuf/client"

	"github.com/desertbit/grumble"
)

func info(ctx *grumble.Context, rpc RPCServer) {

	var sliver *clientpb.Sliver
	if ActiveSliver.Sliver != nil {
		sliver = ActiveSliver.Sliver
	} else if 0 < len(ctx.Args) {
		sliver = getSliver(ctx.Args[0], rpc)
	}

	if sliver != nil {
		fmt.Printf(bold+"            ID: %s%d\n", normal, sliver.ID)
		fmt.Printf(bold+"          Name: %s%s\n", normal, sliver.Name)
		fmt.Printf(bold+"      Hostname: %s%s\n", normal, sliver.Hostname)
		fmt.Printf(bold+"      Username: %s%s\n", normal, sliver.Username)
		fmt.Printf(bold+"           UID: %s%s\n", normal, sliver.UID)
		fmt.Printf(bold+"           GID: %s%s\n", normal, sliver.GID)
		fmt.Printf(bold+"           PID: %s%d\n", normal, sliver.PID)
		fmt.Printf(bold+"            OS: %s%s\n", normal, sliver.OS)
		fmt.Printf(bold+"          Arch: %s%s\n", normal, sliver.Arch)
		fmt.Printf(bold+"Remote Address: %s%s\n", normal, sliver.RemoteAddress)
	} else {
		fmt.Printf(Warn+"No target sliver, see `help %s`\n", consts.InfoStr)
	}
}

func ping(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
}

func getPID(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	fmt.Printf("%d\n", ActiveSliver.Sliver.PID)
}

func getUID(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	fmt.Printf("%s\n", ActiveSliver.Sliver.UID)
}

func getGID(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	fmt.Printf("%s\n", ActiveSliver.Sliver.GID)
}

func whoami(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	fmt.Printf("%s\n", ActiveSliver.Sliver.Username)
}
