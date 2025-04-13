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
	"fmt"
	"net"

	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/db"
)

// StartTCPStagerListener starts a TCP stager listener
func (rpc *Server) StartTCPStagerListener(ctx context.Context, req *clientpb.StagerListenerReq) (*clientpb.StagerListener, error) {
	host := req.GetHost()

	// If host is not an IP address, try to resolve it as an interface name
	if net.ParseIP(host) == nil {
		ifaceIP, err := getInterfaceIP(host)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve interface %s: %v", host, err)
		}
		host = ifaceIP
	}

	job, err := c2.StartTCPStagerListenerJob(host, uint16(req.GetPort()), req.ProfileName, req.GetData())
	if err != nil {
		return nil, err
	}

	listenerJob := &clientpb.ListenerJob{
		JobID:   uint32(job.ID),
		Type:    constants.StageListenerStr,
		TCPConf: req,
	}
	err = db.SaveC2Listener(listenerJob)
	if err != nil {
		return nil, err
	}

	return &clientpb.StagerListener{JobID: uint32(job.ID)}, nil
}

// checkInterface verifies if an IP address or interface name is attached to an existing network interface and returns the IP
func checkInterface(host string) bool {
	// First check if it's an IP address
	if net.ParseIP(host) != nil {
		return true
	}

	// If not an IP, try to resolve as interface name
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}

	for _, iface := range ifaces {
		if iface.Name == host {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				switch v := addr.(type) {
				case *net.IPNet:
					if v.IP.To4() != nil {
						return true
					}
				}
			}
		}
	}

	return false
}

// getInterfaceIP returns the first IPv4 address of the specified interface
func getInterfaceIP(ifaceName string) (string, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return "", err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		switch v := addr.(type) {
		case *net.IPNet:
			if v.IP.To4() != nil {
				return v.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no IPv4 address found for interface %s", ifaceName)
}