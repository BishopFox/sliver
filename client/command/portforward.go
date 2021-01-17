package command

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/desertbit/grumble"

	"github.com/bishopfox/sliver/client/comm"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

// addPortForward - A single function to start a port forwarder, of any type (bind/reverse)
// and of any protocol (TCP/UDP), within the current session or a designated one.
func addPortForward(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {

	// Check a session is targeted (active or with option)
	session := ActiveSession.Get()
	if session == nil && ctx.Flags.Uint("session-id") == 0 {
		fmt.Println(Warn + "No active session or session specified with --session-id")
		return
	}
	var sessID uint32
	if session != nil {
		sessID = session.ID
	} else {
		sessID = uint32(ctx.Flags.Uint("session-id"))
	}

	// User must always specify at least the remote port.
	if ctx.Flags.Uint("rport") == 0 {
		fmt.Println(Warn + "Missing remote port number (--rport)")
		return
	}

	// Handler information, completed at startup by the port forwarder.
	info := &commpb.Handler{
		LHost: ctx.Flags.String("lhost"),
		LPort: int32(ctx.Flags.Uint("lport")),
		RHost: ctx.Flags.String("rhost"),
		RPort: int32(ctx.Flags.Uint("rport")),
	}

	// Host:port to listen/dial locally (console client)
	lhost := ctx.Flags.String("lhost")
	lport := ctx.Flags.Uint("lport")

	// Elements to print after starting the port forwarder
	var dir string
	var portfwdErr error
	lAddr := fmt.Sprintf("%s:%d", lhost, lport)
	rAddr := fmt.Sprintf("%s:%d", info.RHost, info.RPort)

	// Start the port forwarder with the appropriate call, depending on direction and protocol.
	switch ctx.Flags.String("protocol") {
	case "tcp":
		if ctx.Flags.Bool("reverse") {
			dir = "<--"
			portfwdErr = comm.PortfwdReverseTCP(sessID, info)
		} else {
			dir = "-->"
			// Direct TCP: We specify the optional remote source TCP address in place
			// of the client listener which is being specified here, after info
			info.LHost = ctx.Flags.String("srchost")
			info.LPort = int32(ctx.Flags.Uint("srcport"))

			// Create and start the forwarder
			portfwdErr = comm.PortfwdDirectTCP(sessID, info, lhost, int(lport))
		}
	case "udp":
		if ctx.Flags.Bool("reverse") {
			dir = "<--"
			portfwdErr = comm.PortfwdReverseUDP(sessID, info)

		} else {
			dir = "-->"
			// Direct UDP : We specify the optional remote source UDP address in place
			// of the client listener which is being specified here, after info
			info.LHost = ctx.Flags.String("srchost")
			info.LPort = int32(ctx.Flags.Uint("srcport"))

			portfwdErr = comm.PortfwdDirectUDP(sessID, info, lhost, int(lport))
		}
	default:
		fmt.Printf(Warn+"Invalid transport protocol specified (must be tcp or udp): %s\n", ctx.Flags.String("protocol"))
		return
	}

	// Catch the error that might arise from starting the forwarder, no matter type/protocol
	if portfwdErr != nil {
		fmt.Printf(Warn+"Failed to start %s %s port forwarder: %v \n",
			info.Type.String(), info.Transport.String(), portfwdErr)
		return
	}

	// Else print success
	fmt.Printf(Info+"Started %s %s port forwarder (%s %s %s) [Session ID: %d] \n",
		info.Type.String(), info.Transport.String(), lAddr, dir, rAddr, sessID)
}

// printPortForwarders - Print either all active port forwarders for this client, or only the
// forwarders working on the active session if there is one. Further filters in per-forwarder-type printers.
func printPortForwarders(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {

	// If no port forwards return anyway
	forwarders := comm.Forwarders.All()
	if len(forwarders) == 0 {
		fmt.Println(Warn + "No active port forwarders running on this console client")
		return
	}

	// If we are in an active session, we print the portForwarders
	// for this session first, then the next sessions, if any.
	session := ActiveSession.Get()

	// If no direction filters, print all forwarders, maybe filtered by session, or protocol
	if !ctx.Flags.Bool("direct") && !ctx.Flags.Bool("reverse") {
		printDirectForwarders(forwarders, ctx, session)
		printReverseForwarders(forwarders, ctx, session)
		return
	}

	// Else, check for each direction
	if ctx.Flags.Bool("direct") {
		printDirectForwarders(forwarders, ctx, session)
	}
	if ctx.Flags.Bool("reverse") {
		printReverseForwarders(forwarders, ctx, session)
	}
}

// closePortForwarder - Close one or more port forwarders running between this client console and one or more sessions.
func closePortForwarder(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {

	// Check a session is targeted (active or with option)
	session := ActiveSession.Get()
	if session == nil && ctx.Flags.Uint("session-id") == 0 {
		fmt.Println(Warn + "No active session or session specified with --session-id")
		return
	}

	// If none of the flags are set, return
	if ctx.Flags.String("id") == "" && ctx.Flags.Uint("session-id") == 0 && session == nil &&
		!ctx.Flags.Bool("reverse") && !ctx.Flags.Bool("reverse") {
		fmt.Printf(Warn + "You must specifiy at least one option/filter to close one or more forwarders.\n")
	}

	// Check if we have to close active connections for these forwarders to be closed.
	var closeActive = false
	if ctx.Flags.Bool("close-conns") {
		closeActive = true
	}

	// If an ID is given, close this forwarder
	if ctx.Flags.String("id") != "" {
		f := comm.Forwarders.Get(ctx.Flags.String("id"))
		if f != nil {
			err := f.Close(closeActive)
			if err != nil {
				fmt.Printf(Warn+"Failed to close %s %s port forwarder: %v \n",
					f.Info().Type.String(), f.Info().Transport.String(), err)
			}

			// Else print success
			var dir string
			switch f.Info().Type {
			case commpb.HandlerType_Bind:
				dir = "-->"
			case commpb.HandlerType_Reverse:
				dir = "<--"
			}
			rAddr := fmt.Sprintf("%s:%d", f.Info().RHost, f.Info().RPort)
			fmt.Printf(Info+"Closed %s %s port forwarder (%s %s %s) [Session ID: %d] \n",
				f.Info().Type.String(), f.Info().Transport.String(), f.LocalAddr(),
				dir, rAddr, f.SessionID())
		}
	}

	// If there is a Session ID, close all portForwarders for this session
	if ctx.Flags.Uint("session-id") != 0 {
		forwarders := comm.Forwarders.All()
		for _, f := range forwarders {
			if f.SessionID() == uint32(ctx.Flags.Uint("session-id")) {

				err := f.Close(closeActive)
				if err != nil {
					fmt.Printf(Warn+"Failed to close %s %s port forwarder: %v \n",
						f.Info().Type.String(), f.Info().Transport.String(), err)
				}

				// Else print success
				var dir string
				switch f.Info().Type {
				case commpb.HandlerType_Bind:
					dir = "-->"
				case commpb.HandlerType_Reverse:
					dir = "<--"
				}
				rAddr := fmt.Sprintf("%s:%d", f.Info().RHost, f.Info().RPort)
				fmt.Printf(Info+"Closed %s %s port forwarder (%s %s %s) [Session ID: %d] \n",
					f.Info().Type.String(), f.Info().Transport.String(), f.LocalAddr(),
					dir, rAddr, f.SessionID())

			}
		}
	}

	// If all direct are selected and we have an active session, close all direct for this session.
	if ctx.Flags.Bool("direct") {
		forwarders := comm.Forwarders.All()
		for _, f := range forwarders {
			if f.Info().Type == commpb.HandlerType_Bind {

				// If we are in a session, we only close the ones belonging to it, otherwise close all.
				if (session != nil && f.SessionID() == session.ID) || session == nil {

					err := f.Close(closeActive)
					if err != nil {
						fmt.Printf(Warn+"Failed to close %s %s port forwarder: %v \n",
							f.Info().Type.String(), f.Info().Transport.String(), err)
					}

					// Else print success
					var dir string
					switch f.Info().Type {
					case commpb.HandlerType_Bind:
						dir = "-->"
					case commpb.HandlerType_Reverse:
						dir = "<--"
					}
					rAddr := fmt.Sprintf("%s:%d", f.Info().RHost, f.Info().RPort)
					fmt.Printf(Info+"Closed %s %s port forwarder (%s %s %s) [Session ID: %d] \n",
						f.Info().Type.String(), f.Info().Transport.String(), f.LocalAddr(),
						dir, rAddr, f.SessionID())
				}
			}
		}
	}

	// If all reverse are selected and we have an active session, close all reverse for this session.
	if ctx.Flags.Bool("reverse") {
		forwarders := comm.Forwarders.All()
		for _, f := range forwarders {
			if f.Info().Type == commpb.HandlerType_Reverse {

				// If we are in a session, we only close the ones belonging to it, otherwise close all.
				if (session != nil && f.SessionID() == session.ID) || session == nil {

					err := f.Close(closeActive)
					if err != nil {
						fmt.Printf(Warn+"Failed to close %s %s port forwarder: %v \n",
							f.Info().Type.String(), f.Info().Transport.String(), err)
					}

					// Else print success
					var dir string
					switch f.Info().Type {
					case commpb.HandlerType_Bind:
						dir = "-->"
					case commpb.HandlerType_Reverse:
						dir = "<--"
					}
					rAddr := fmt.Sprintf("%s:%d", f.Info().RHost, f.Info().RPort)
					lAddr := f.LocalAddr()
					fmt.Printf(Info+"Closed %s %s port forwarder (%s %s %s) [Session ID: %d] \n",
						f.Info().Type.String(), f.Info().Transport.String(), lAddr,
						dir, rAddr, f.SessionID())
				}
			}
		}
	}
}

// We pass the command flags context because some of them are filtered here, not in the caller, like protocols.
func printDirectForwarders(forwarders map[string]comm.Forwarder, ctx *grumble.Context, session *clientpb.Session) {

	// Print title
	fmt.Printf("-- Direct Forwarders -- \n\n")

	direct := map[string]comm.Forwarder{}
	for id, f := range forwarders {
		if session != nil && session.ID == f.SessionID() {
			if f.Info().Type == commpb.HandlerType_Bind {
				direct[id] = f
			}
		} else {
			if f.Info().Type == commpb.HandlerType_Bind {
				direct[id] = f
			}
		}
	}
	if len(direct) == 0 {
		fmt.Println(" No active Direct port forwarders (all sessions)")
		return
	}

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	fmt.Fprintln(table, "Protocol\tLHost:LPort\tRHost:RPort\tConn Stats\tSession ID\tForwarder ID\t")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Protocol")),
		strings.Repeat("=", len("LHost:LPort")),
		strings.Repeat("=", len("RHost:RPort")),
		strings.Repeat("=", len("Conn Stats")),
		strings.Repeat("=", len("Session ID")),
		strings.Repeat("=", len("Forwarder ID")))

	for _, f := range direct {
		// Check remaining filters
		if ctx.Flags.Bool("udp") && f.Info().Transport == commpb.Transport_TCP { // Filter by transport TCP
			continue
		}
		if ctx.Flags.Bool("tcp") && f.Info().Transport == commpb.Transport_UDP { // Filter by transport UDP
			continue
		}

		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t\n",
			f.Info().Transport,
			f.LocalAddr(),
			"-->  "+fmt.Sprintf("%s:%d", f.Info().RHost, f.Info().RPort),
			f.ConnStats(),
			fmt.Sprintf("%d", f.SessionID()),
			f.Info().ID,
		)
	}

	table.Flush()
	fmt.Printf(outputBuf.String())
}

// printReverseForwarders - We don't have an optional dial source address.
func printReverseForwarders(forwarders map[string]comm.Forwarder, ctx *grumble.Context, session *clientpb.Session) {

	// Print title
	fmt.Printf("\n-- Reverse Forwarders -- \n\n")

	reverse := map[string]comm.Forwarder{}
	for id, f := range forwarders {
		if session != nil && session.ID == f.SessionID() {
			if f.Info().Type == commpb.HandlerType_Reverse {
				reverse[id] = f
			}
		} else {
			if f.Info().Type == commpb.HandlerType_Reverse {
				reverse[id] = f
			}
		}
	}
	if len(reverse) == 0 {
		fmt.Println(" No active Reverse port forwarders (all sessions)")
		return
	}

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	fmt.Fprintln(table, "Protocol\tLHost:LPort\tRHost:RPort\tConn Stats\tSession ID\tForwarder ID\t")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Protocol")),
		strings.Repeat("=", len("LHost:LPort")),
		strings.Repeat("=", len("RHost:RPort")),
		strings.Repeat("=", len("Conn Stats")),
		strings.Repeat("=", len("Session ID")),
		strings.Repeat("=", len("Forwarder ID")))

	for _, f := range reverse {
		// Check remaining filters
		if ctx.Flags.Bool("udp") && f.Info().Transport == commpb.Transport_TCP { // Filter by transport TCP
			continue
		}
		if ctx.Flags.Bool("tcp") && f.Info().Transport == commpb.Transport_UDP { // Filter by transport UDP
			continue
		}

		// Else print table
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t\n",
			f.Info().Transport,
			fmt.Sprintf("%s:%d", f.Info().LHost, f.Info().LPort),
			"<--  "+fmt.Sprintf("%s:%d", f.Info().RHost, f.Info().RPort),
			f.ConnStats(),
			fmt.Sprintf("%d", f.SessionID()),
			f.Info().ID,
		)
	}

	table.Flush()
	fmt.Printf(outputBuf.String())
}
