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
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"

	"github.com/golang/protobuf/proto"
)

// Ls - List a directory
func (rpc *Server) Ls(ctx context.Context, msg *sliverpb.LsReq) (*sliverpb.Ls, error) {
	resp := &sliverpb.Ls{}
	err := rpc.GenericHandler(msg, msg.Request, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Rm - Remove file or directory
func (rpc *Server) Rm(ctx context.Context, msg *sliverpb.RmReq) (*sliverpb.Rm, error) {
	resp := &sliverpb.Rm{}
	err := rpc.GenericHandler(msg, msg.Request, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Mkdir - Make a directory
func (rpc *Server) Mkdir(ctx context.Context, msg *sliverpb.MkdirReq) (*sliverpb.Mkdir, error) {
	resp := &sliverpb.Mkdir{}
	err := rpc.GenericHandler(msg, msg.Request, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Cd - Change directory
func (rpc *Server) Cd(ctx context.Context, msg *sliverpb.CdReq) (*sliverpb.Pwd, error) {
	resp := &sliverpb.Pwd{}
	err := rpc.GenericHandler(msg, msg.Request, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func rpcPwd(req []byte, timeout time.Duration, resp RPCResponse) {
	pwdReq := &sliverpb.PwdReq{}
	err := proto.Unmarshal(req, pwdReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := (*core.Hive.Slivers)[pwdReq.SliverID]

	data, _ := proto.Marshal(&sliverpb.PwdReq{})
	data, err = sliver.Request(sliverpb.MsgPwdReq, timeout, data)
	resp(data, err)
}

func rpcDownload(req []byte, timeout time.Duration, resp RPCResponse) {
	downloadReq := &sliverpb.DownloadReq{}
	err := proto.Unmarshal(req, downloadReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(downloadReq.SliverID)

	data, _ := proto.Marshal(&sliverpb.DownloadReq{
		Path: downloadReq.Path,
	})
	data, err = sliver.Request(sliverpb.MsgDownloadReq, timeout, data)
	resp(data, err)
}

func rpcUpload(req []byte, timeout time.Duration, resp RPCResponse) {
	uploadReq := &sliverpb.UploadReq{}
	err := proto.Unmarshal(req, uploadReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(uploadReq.SliverID)

	data, _ := proto.Marshal(&sliverpb.UploadReq{
		Encoder: uploadReq.Encoder,
		Path:    uploadReq.Path,
		Data:    uploadReq.Data,
	})
	data, err = sliver.Request(sliverpb.MsgUploadReq, timeout, data)
	resp(data, err)
}
