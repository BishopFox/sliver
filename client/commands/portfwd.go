package commands

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
	"fmt"

	"github.com/bishopfox/sliver/client/comm"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commpb"
	"github.com/evilsocket/islazy/tui"
)

// Portfwd - Manage port forwarders on the client
type Portfwd struct {
	Options struct {
		Protocol  string `long:"protocol" description:"Show only for given protocol"`
		Direct    bool   `long:"direct" description:"Show only direct forwarders"`
		Reverse   bool   `long:"reverse" description:"Show only reverse forwarders"`
		SessionID uint32 `long:"session-id" description:"Show only forwarders on a given session"`
	} `group:"print filters"`
}

// Execute - Print the port forwarders by default
func (p *Portfwd) Execute(args []string) (err error) {
	// If no port forwards return anyway
	forwarders := comm.Forwarders.All()
	if len(forwarders) == 0 {
		fmt.Println(util.Info + "No active port forwarders running on this console client")
		return
	}

	// Filter direct/reverse forwarders
	var direct = map[string]comm.Forwarder{}
	var reverse = map[string]comm.Forwarder{}
	for id, f := range forwarders {
		switch f.Info().Type {
		case commpb.HandlerType_Bind:
			direct[id] = f
		case commpb.HandlerType_Reverse:
			reverse[id] = f
		}
	}

	// If we are in an active session, we print the portForwarders
	// for this session first, then the next sessions, if any.
	session := cctx.Context.Sliver
	var info *clientpb.Session
	if session != nil {
		info = session.Session
	}

	// If no direction filters, print all forwarders, maybe filtered by session, or protocol
	if !p.Options.Direct && !p.Options.Reverse {
		p.printForwarders(true, direct, info)
		p.printForwarders(false, reverse, info)
		return
	}

	// Else, check for each direction
	if p.Options.Direct {
		p.printForwarders(true, direct, info)
	}
	if p.Options.Reverse {
		p.printForwarders(false, reverse, info)
	}

	return
}

// printReverseForwarders - We don't have an optional dial source address.
func (p *Portfwd) printForwarders(direct bool, forwarders map[string]comm.Forwarder, session *clientpb.Session) {
	var title string
	if direct {
		title = " Direct"
	} else {
		title = "\n Reverse"
	}
	if len(forwarders) == 0 {
		fmt.Printf("\n"+util.Info+"No active %s forwarders (all sessions) \n", title)
		return
	}

	table := util.NewTable(tui.Bold(tui.Yellow(title)))
	headers := []string{"ID", "Protocol", "Local Address", "   ", "Remote Address", "Conn Stats", "Session"}
	headLen := []int{0, 0, 0, 0, 0, 0, 0}
	table.SetColumns(headers, headLen)

	// Add each forwarder to table
	for _, f := range forwarders {

		// Check remaining filters
		if p.Options.Protocol == "udp" && f.Info().Transport == commpb.Transport_TCP {
			continue
		}
		if p.Options.Protocol == "tcp" && f.Info().Transport == commpb.Transport_UDP {
			continue
		}

		rhost := fmt.Sprintf("%s:%d", f.Info().RHost, f.Info().RPort)
		sessID := fmt.Sprintf("%d", f.SessionID())
		var dir string
		if direct {
			dir = "-->"
		} else {
			dir = "<--"
		}
		row := []string{"", f.Info().Transport.String(), f.LocalAddr(), dir, rhost, f.ConnStats(), sessID}

		if session != nil && f.SessionID() == session.ID {
			row = table.ApplyCurrentRowColor(row, fmt.Sprintf("%s%s", tui.DIM, tui.YELLOW))
		}

		table.AppendRow(row)
	}
	table.Output()
}

// PortfwdOpen - Start a client-implant port forwarder
type PortfwdOpen struct {
	Options struct {
		Protocol  string `long:"protocol" description:"Transport protocol" default:"tcp"`
		Reverse   bool   `long:"reverse" description:"Reverse forwards from Rhost (implant) to LHost (client)"`
		LHost     string `long:"lhost" description:"Console address to dial/listen on" default:"127.0.0.1"`
		LPort     int32  `long:"lport" description:"Console listen port" default:"2020"`
		RHost     string `long:"rhost" description:"Remote host address to dial/listen on" default:"0.0.0.0"`
		RPort     int32  `long:"rport" description:"Remote port number" required:"true"`
		SessionID uint32 `long:"session-id" description:"Start the forwarder on a specific session"`
	} `group:"forwarder options"`
}

// Execute -  Start a client-implant port forwarder
func (p *PortfwdOpen) Execute(args []string) (err error) {

	// Check a session is targeted (active or with option)
	session := cctx.Context.Sliver
	if session == nil && p.Options.SessionID == 0 {
		fmt.Println(util.Error + "No active session or session specified with --session-id")
		return
	}
	var sessID uint32
	if session != nil {
		sessID = session.ID
	} else {
		sessID = p.Options.SessionID
	}

	// User must always specify at least the remote port.
	if p.Options.RPort == 0 {
		fmt.Println(util.Error + "Missing remote port number (--rport) \n")
		return
	}

	// Handler information, completed at startup by the port forwarder.
	info := &commpb.Handler{
		LHost: p.Options.LHost,
		LPort: p.Options.LPort,
		RHost: p.Options.RHost,
		RPort: p.Options.RPort,
	}

	// Host:port to listen/dial locally (console client)
	lhost := p.Options.LHost
	lport := uint(p.Options.LPort)

	// Elements to print after starting the port forwarder
	var dir string
	var portfwdErr error
	lAddr := fmt.Sprintf("%s:%d", lhost, lport)
	rAddr := fmt.Sprintf("%s:%d", info.RHost, info.RPort)

	// Start the port forwarder with the appropriate call, depending on direction and protocol.
	switch p.Options.Protocol {
	case "tcp":
		if p.Options.Reverse {
			dir = "<--"
			portfwdErr = comm.PortfwdReverseTCP(sessID, info)
		} else {
			dir = "-->"
			// Direct TCP: We specify the optional remote source TCP address in place
			// of the client listener which is being specified here, after info
			info.LHost = lhost
			info.LPort = int32(lport)

			// Create and start the forwarder
			portfwdErr = comm.PortfwdDirectTCP(sessID, info, lhost, int(lport))
		}
	case "udp":
		if p.Options.Reverse {
			dir = "<--"
			portfwdErr = comm.PortfwdReverseUDP(sessID, info)

		} else {
			dir = "-->"
			// Direct UDP : We specify the optional remote source UDP address in place
			// of the client listener which is being specified here, after info
			info.LHost = lhost
			info.LPort = int32(lport)

			portfwdErr = comm.PortfwdDirectUDP(sessID, info, lhost, int(lport))
		}
	default:
		fmt.Printf(util.Error+"Invalid transport protocol (must be tcp or udp): %s\n", p.Options.Protocol)
		return
	}

	// Catch the error that might arise from starting the forwarder, no matter type/protocol
	if portfwdErr != nil {
		fmt.Printf(util.Error+"Failed to start %s %s port forwarder: %v \n",
			info.Type.String(), info.Transport.String(), portfwdErr)
		return
	}

	// Else print success
	fmt.Printf(util.Info+"Started %s %s port forwarder (%s %s %s) [Session ID: %d] \n",
		info.Type.String(), info.Transport.String(), lAddr, dir, rAddr, sessID)
	return
}

// PortfwdClose - Stop a port forwarder
type PortfwdClose struct {
	Options struct {
		ForwarderID []string `long:"id" description:"Forwarder IDs, comma-separated" env-delim:" "`
		Protocol    string   `long:"protocol" description:"Close only for given transport protocol"`
		Direct      bool     `long:"direct" description:"Close only if direct"`
		Reverse     bool     `long:"reverse" description:"Close only if reverse"`
		SessionID   uint32   `long:"session-id" description:"Close if forwarder belongs to session"`
		CloseConns  bool     `long:"close-conns" description:"Close active connections initiated by forwarder (TCP-only)"`
	} `group:"forwarder options"`
}

// Execute -  Start a client-implant port forwarder
func (p *PortfwdClose) Execute(args []string) (err error) {

	// Check a session is targeted (active or with option)
	session := cctx.Context.Sliver.Session
	if session == nil && p.Options.SessionID == 0 {
		fmt.Println(util.Error + "No active session or session specified with --session-id")
		return
	}

	// If none of the flags are set, return
	if len(p.Options.ForwarderID) == 0 && p.Options.SessionID == 0 && session == nil &&
		!p.Options.Reverse && !p.Options.Direct {
		fmt.Printf(util.Error + "You must specifiy at least one option/filter to close one or more forwarders.\n")
	}

	// Check if we have to close active connections for these forwarders to be closed.
	var closeActive = false
	if p.Options.CloseConns {
		closeActive = true
	}

	// If an ID is given, close this forwarder
	for _, id := range p.Options.ForwarderID {

		if id != "" {
			f := comm.Forwarders.Get(id)
			if f != nil {
				err := f.Close(closeActive)
				if err != nil {
					fmt.Printf(util.Error+"Failed to close %s %s port forwarder: %v \n",
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
				fmt.Printf(util.Info+"Closed %s %s port forwarder (%s %s %s) [Session ID: %d] \n",
					f.Info().Type.String(), f.Info().Transport.String(), f.LocalAddr(),
					dir, rAddr, f.SessionID())
			}
		}
	}

	// If there is a Session ID, close all portForwarders for this session
	if p.Options.SessionID != 0 {
		forwarders := comm.Forwarders.All()
		for _, f := range forwarders {
			if f.SessionID() == p.Options.SessionID {

				err := f.Close(closeActive)
				if err != nil {
					fmt.Printf(util.Error+"Failed to close %s %s port forwarder: %v \n",
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
				fmt.Printf(util.Info+"Closed %s %s port forwarder (%s %s %s) [Session ID: %d] \n",
					f.Info().Type.String(), f.Info().Transport.String(), f.LocalAddr(),
					dir, rAddr, f.SessionID())

			}
		}
	}

	// If all direct are selected and we have an active session, close all direct for this session.
	if p.Options.Direct {
		forwarders := comm.Forwarders.All()
		for _, f := range forwarders {
			if f.Info().Type == commpb.HandlerType_Bind {

				// If we are in a session, we only close the ones belonging to it, otherwise close all.
				if (session != nil && f.SessionID() == session.ID) || session == nil {

					err := f.Close(closeActive)
					if err != nil {
						fmt.Printf(util.Error+"Failed to close %s %s port forwarder: %v \n",
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
					fmt.Printf(util.Info+"Closed %s %s port forwarder (%s %s %s) [Session ID: %d] \n",
						f.Info().Type.String(), f.Info().Transport.String(), f.LocalAddr(),
						dir, rAddr, f.SessionID())
				}
			}
		}
	}

	// If all reverse are selected and we have an active session, close all reverse for this session.
	if p.Options.Reverse {
		forwarders := comm.Forwarders.All()
		for _, f := range forwarders {
			if f.Info().Type == commpb.HandlerType_Reverse {

				// If we are in a session, we only close the ones belonging to it, otherwise close all.
				if (session != nil && f.SessionID() == session.ID) || session == nil {

					err := f.Close(closeActive)
					if err != nil {
						fmt.Printf(util.Error+"Failed to close %s %s port forwarder: %v \n",
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
					fmt.Printf(util.Info+"Closed %s %s port forwarder (%s %s %s) [Session ID: %d] \n",
						f.Info().Type.String(), f.Info().Transport.String(), lAddr,
						dir, rAddr, f.SessionID())
				}
			}
		}
	}
	return
}
