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
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/commands"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
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
					suggestions = append(suggestions, ip.String()+" ")

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
					suggestions = append(suggestions, a.String()+" ")
				} else {
					suggestions = append(suggestions, a.String())
				}
			}
		}
	}

	comp.Suggestions = suggestions
	return
}

// sessionIfacePublicNetworks - Get all non-loopback addresses for a session host.
func sessionIfacePublicNetworks(last string, sess *clientpb.Session, alone bool) (comp *readline.CompletionGroup) {
	comp = &readline.CompletionGroup{
		Name:        fmt.Sprintf("networks (session %d)", sess.ID),
		MaxLength:   5,
		DisplayType: readline.TabDisplayGrid,
	}
	var suggestions []string

	ifconfig, err := transport.RPC.Ifconfig(context.Background(), &sliverpb.IfconfigReq{
		Request: commands.ContextRequest(sess),
	})
	if err != nil {
		return
	}

	for _, iface := range ifconfig.NetInterfaces {
		for _, ip := range iface.IPAddresses {

			if !strings.HasPrefix(ip, last) {
				continue
			}

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
				if alone {
					suggestions = append(suggestions, ip+" ")
				} else {
					suggestions = append(suggestions, ip)
				}
			} else if 32 < subnet && !isLoopback(ip) {
				if alone {
					suggestions = append(suggestions, ip+" ")
				} else {
					suggestions = append(suggestions, ip)
				}
			}
		}
	}

	comp.Suggestions = suggestions
	return
}

// sessionIfaceAddrs - Get all available addresses (including loopback) for an implant host
func sessionIfaceAddrs(last string, sess *clientpb.Session, alone bool) (comp *readline.CompletionGroup) {
	comp = &readline.CompletionGroup{
		Name:        fmt.Sprintf("addresses (session %d)", sess.ID),
		MaxLength:   5,
		DisplayType: readline.TabDisplayGrid,
	}
	var suggestions []string

	ifconfig, err := transport.RPC.Ifconfig(context.Background(), &sliverpb.IfconfigReq{
		Request: commands.ContextRequest(sess),
	})
	if err != nil {
		return
	}

	for _, iface := range ifconfig.NetInterfaces {
		for _, ip := range iface.IPAddresses {

			if !strings.HasPrefix(ip, last) {
				continue
			}

			ip, subnet, err := net.ParseCIDR(ip)
			if err != nil && ip == nil && subnet == nil {
				continue
			}
			if ip != nil {
				if alone {
					suggestions = append(suggestions, ip.String()+" ")

				} else {
					suggestions = append(suggestions, ip.String())

				}
			}
		}
	}

	comp.Suggestions = suggestions
	return
}

// Returns all interfaces on implants that are reachable via a route.
func routedSessionIfaceAddrs(last string, except uint32, alone bool) (comps []*readline.CompletionGroup) {
	// If except is 0, do not include the matching session

	return
}

// Returns all available IPs, for each registered/active implant (each has a group)
func allSessionIfaceAddrs(last string, except uint32, alone bool) (comps []*readline.CompletionGroup) {
	// If except is 0, do not include the matching session
	sessions := GetAllSessions()
	if len(sessions) == 0 {
		return
	}
	if except != 0 {
		delete(sessions, except)
	}

	for _, sess := range sessions {
		comps = append(comps, sessionIfaceAddrs(last, sess, alone))
	}

	return
}

// Returns all networks to which implants belong
func allSessionsIfaceNetworks(last string, except uint32, alone bool) (comps []*readline.CompletionGroup) {
	// If except is 0, do not include the matching session
	sessions := GetAllSessions()
	if len(sessions) == 0 {
		return
	}
	if except != 0 {
		delete(sessions, except)
	}

	for _, sess := range sessions {
		comps = append(comps, sessionIfacePublicNetworks(last, sess, alone))
	}

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
