package completion

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
	"strconv"

	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// CompleteWireGuardPortfwds - Returns the list of active WireGuard port forwarders for the current session.
func CompleteWireGuardPortfwds() (comps []*readline.CompletionGroup) {

	comp := &readline.CompletionGroup{
		Name:         "wireguard forwarders",
		MaxLength:    20,
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
	}

	fwdList, err := transport.RPC.WGListForwarders(context.Background(), &sliverpb.WGTCPForwardersReq{
		Request: core.ActiveSessionRequest(),
	})

	if err != nil {
		return
	}
	if fwdList.Response != nil && fwdList.Response.Err != "" {
		return
	}
	if fwdList.Forwarders == nil || len(fwdList.Forwarders) == 0 {
		return
	}

	for _, fwd := range fwdList.Forwarders {
		id := strconv.Itoa(int(fwd.ID))
		comp.Suggestions = append(comp.Suggestions, id)
		desc := fmt.Sprintf(" %s  -->  %s", fwd.LocalAddr, fwd.RemoteAddr)
		comp.Descriptions[id] = readline.DIM + desc + readline.RESET
	}

	return []*readline.CompletionGroup{comp}
}

// CompleteWireGuardSocksServers - Returns the list of active WireGuard Socks listeners for the current session.
func CompleteWireGuardSocksServers() (comps []*readline.CompletionGroup) {

	comp := &readline.CompletionGroup{
		Name:         "wireguard socks listeners",
		MaxLength:    20,
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
	}

	socksList, err := transport.RPC.WGListSocksServers(context.Background(), &sliverpb.WGSocksServersReq{
		Request: core.ActiveSessionRequest(),
	})

	if err != nil {
		return
	}
	if socksList.Response != nil && socksList.Response.Err != "" {
		return
	}
	if socksList.Servers == nil || len(socksList.Servers) == 0 {
		return
	}

	for _, s := range socksList.Servers {
		id := strconv.Itoa(int(s.ID))
		comp.Suggestions = append(comp.Suggestions, id)
		desc := fmt.Sprintf(" %s", s.LocalAddr)
		comp.Descriptions[id] = readline.DIM + desc + readline.RESET
	}

	return []*readline.CompletionGroup{comp}
}
