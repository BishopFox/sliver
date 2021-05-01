package rpc

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

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// GetServerInterfaces - Get the server interfaces for C2 listeners
func (rpc *Server) GetServerInterfaces(ctx context.Context, req *sliverpb.IfconfigReq) (*sliverpb.Ifconfig, error) {

	resp := &sliverpb.Ifconfig{}

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
			iface := &sliverpb.NetInterface{}
			iface.IPAddresses = append(iface.IPAddresses, ip.String())
			resp.NetInterfaces = append(resp.NetInterfaces, iface)
		}
	}

	return resp, nil
}
