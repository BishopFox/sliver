package comm

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
	"net"

	"github.com/golang/protobuf/proto"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
)

func checkSessionNetIfaces(ip net.IP, sess *core.Session) (err error) {

	// Request interfaces to implant.
	ifacesReq := &sliverpb.IfconfigReq{
		Request: &commonpb.Request{SessionID: sess.ID},
	}
	data, _ := proto.Marshal(ifacesReq)
	resp, err := sess.Request(sliverpb.MsgNumber(ifacesReq), defaultNetTimeout, data)
	if err != nil {
		return fmt.Errorf("Error getting session interfaces: %s", err.Error())
	}
	ifaces := &sliverpb.Ifconfig{}
	proto.Unmarshal(resp, ifaces)

	// For all net interfaces, check there is one that the new route subnet contains.
	var found = false
	for _, iface := range ifaces.NetInterfaces {

		// Normally the first field is the host's interface IP in CIDR notation.
		ipv4CIDR := iface.IPAddresses[0]

		// Always check if we can have both Network CIDR and IP address
		ipAddr, subnet, err := net.ParseCIDR(ipv4CIDR)
		if err != nil {
			ip = net.ParseIP(iface.IPAddresses[0])
		}
		// Loopback are not allowed when routing, only with port forward.
		if ipAddr.IsLoopback() {
			continue
		}
		if subnet.Contains(ip) {
			found = true
		}
	}
	// If yes, we can go on, else we return.
	if !found {
		return fmt.Errorf("Error adding route: implant host has no network for IP %s",
			ip.String())
	}
	return
}

// checkAllSessionIfaces - Verifies all network (non-loopback) interfaces for implants.
func checkAllSessionIfaces(subnet *net.IPNet) (session *core.Session, err error) {

	var found, doublon = false, false
	var sessionIDs []uint32
	var sessDesc []string

	for _, sess := range core.Sessions.All() {

		err = checkSessionNetIfaces(subnet.IP, sess)
		if err != nil {
			continue
		}

		if found {
			doublon = true
		} else if doublon {
			return nil, fmt.Errorf("Sessions %s and %s have colliding interfaces for subnet %s",
				sessDesc[0], sessDesc[1], subnet.IP.String())
		} else {
			sessionIDs = append(sessionIDs, sess.ID)
			sessDesc = append(sessDesc, fmt.Sprintf("%s (ID:%d)", sess.Name, sess.ID))
			found = true
		}

		// If we found one, we have the last node's session.
		if found && !doublon {
			session = sess
		}
	}

	if !found {
		return nil, fmt.Errorf("Error adding route: no implant hosts have access to subnet %s", subnet.IP.String())
	}

	return
}
