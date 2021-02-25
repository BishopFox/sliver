package completers

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
	"context"
	"net"
	"strings"

	"github.com/bishopfox/sliver/client/readline"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// clientInterfaceAddrs - All addresses of the client host
func clientInterfaceAddrs(last string, alone bool) (comp *readline.CompletionGroup) {

	// Completions
	comp = &readline.CompletionGroup{
		Name:        "client addresses",
		MaxLength:   5,
		DisplayType: readline.TabDisplayGrid,
	}
	var suggestions []string

	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}

		for _, a := range addrs {
			ip, _, err := net.ParseCIDR(a.String())
			if err != nil {
				continue
			}
			if strings.HasPrefix(ip.String(), last) {
				if alone {
					suggestions = append(suggestions, ip.String())
					// suggestions = append(suggestions, ip.String()+" ")
				} else {
					suggestions = append(suggestions, ip.String())
				}
			}
		}
	}

	comp.Suggestions = suggestions
	return
}

// clientInterfaceNetworks - All network to which client belongs.
func clientInterfaceNetworks(last string, alone bool) (comp *readline.CompletionGroup) {

	// Completions
	comp = &readline.CompletionGroup{
		Name:        "client networks",
		MaxLength:   5,
		DisplayType: readline.TabDisplayGrid,
	}
	var suggestions []string

	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {

			if strings.HasPrefix(a.String(), last) {
				if alone {
					suggestions = append(suggestions, a.String())
					// suggestions = append(suggestions, a.String()+" ")
				} else {
					suggestions = append(suggestions, a.String())
				}
			}
		}
	}

	comp.Suggestions = suggestions
	return
}

// GetAllSessions - Get a map of all sessions
func GetAllSessions() (sessionsMap map[uint32]*clientpb.Session) {
	sessions, err := transport.RPC.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		return
	}
	sessionsMap = map[uint32]*clientpb.Session{}
	for _, session := range sessions.GetSessions() {
		sessionsMap[session.ID] = session
	}

	return
}

func isLoopback(ip string) bool {
	if strings.HasPrefix(ip, "127") || strings.HasPrefix(ip, "::1") {
		return true
	}
	return false
}
