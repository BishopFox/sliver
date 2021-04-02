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

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// Ls - List a directory
func (rpc *Server) Ls(ctx context.Context, req *sliverpb.LsReq) (*sliverpb.Ls, error) {
	resp := &sliverpb.Ls{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Rm - Remove file or directory
func (rpc *Server) Rm(ctx context.Context, req *sliverpb.RmReq) (*sliverpb.Rm, error) {
	resp := &sliverpb.Rm{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Mkdir - Make a directory
func (rpc *Server) Mkdir(ctx context.Context, req *sliverpb.MkdirReq) (*sliverpb.Mkdir, error) {
	resp := &sliverpb.Mkdir{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Cd - Change directory
func (rpc *Server) Cd(ctx context.Context, req *sliverpb.CdReq) (*sliverpb.Pwd, error) {
	resp := &sliverpb.Pwd{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Pwd - Change directory
func (rpc *Server) Pwd(ctx context.Context, req *sliverpb.PwdReq) (*sliverpb.Pwd, error) {
	resp := &sliverpb.Pwd{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Download - Download a file from the remote file system
func (rpc *Server) Download(ctx context.Context, req *sliverpb.DownloadReq) (*sliverpb.Download, error) {
	resp := &sliverpb.Download{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Upload - Upload a file from the remote file system
func (rpc *Server) Upload(ctx context.Context, req *sliverpb.UploadReq) (*sliverpb.Upload, error) {
	resp := &sliverpb.Upload{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
