package prelude

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"path"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

const (
	coffLoaderName   = "coff-loader"
	loaderEntryPoint = "LoadAndRun"
	bofEntryPoint    = "go"
)

type bofArgs struct {
	ArgType string      `json:"type"`
	Value   interface{} `json:"value"`
}

func runBOF(implant ActiveImplant, rpc rpcpb.SliverRPCClient, bof []byte, args []bofArgs, onFinishCallback func(string, int, int)) (output string, err error) {
	if !isLoaderLoaded(implant, rpc) {
		err = registerLoader(implant, rpc)
		if err != nil {
			return
		}
	}
	bofArgs := core.BOFArgsBuffer{
		Buffer: new(bytes.Buffer),
	}
	for _, a := range args {
		switch a.ArgType {
		case "integer":
			fallthrough
		case "int":
			if v, ok := a.Value.(float64); ok {
				err = bofArgs.AddInt(uint32(v))
			}
		case "string":
			if v, ok := a.Value.(string); ok {
				err = bofArgs.AddString(v)
			}
		case "wstring":
			if v, ok := a.Value.(string); ok {
				err = bofArgs.AddWString(v)
			}
		case "short":
			if v, ok := a.Value.(float64); ok {
				err = bofArgs.AddShort(uint16(v))
			}
		}
		if err != nil {
			return
		}
	}

	extArgs := core.BOFArgsBuffer{
		Buffer: new(bytes.Buffer),
	}

	parsedArgs, err := bofArgs.GetBuffer()
	if err != nil {
		return
	}
	err = extArgs.AddString(bofEntryPoint)
	if err != nil {
		return
	}
	err = extArgs.AddData(bof)
	if err != nil {
		return
	}
	err = extArgs.AddData(parsedArgs)
	if err != nil {
		return
	}
	extArgsBuffer, err := extArgs.GetBuffer()
	if err != nil {
		return
	}

	extResp, err := rpc.CallExtension(context.Background(), &sliverpb.CallExtensionReq{
		Name:        coffLoaderName,
		ServerStore: false,
		Args:        extArgsBuffer,
		Export:      loaderEntryPoint,
		Request:     MakeRequest(implant),
	})

	if err != nil {
		return
	}

	// If Async req, onFinishCallback won't be nil
	if onFinishCallback != nil {
		onFinishCallback(string(extResp.Output), SuccessExitStatus, int(implant.GetPID()))
		return
	}

	if extResp.Response != nil && extResp.Response.Err != "" {
		err = errors.New(extResp.Response.Err)
	}
	output = string(extResp.Output)

	return
}

func registerLoader(implant ActiveImplant, rpc rpcpb.SliverRPCClient) error {
	var coffLoaderPath string

	switch implant.GetArch() {
	case "amd64":
		coffLoaderPath = "COFFLoader.x64.dll"
	case "386":
		coffLoaderPath = "COFFLoader.x86.dll"
	}
	loaderPath := path.Join(assets.GetExtensionsDir(), coffLoaderName, coffLoaderPath)
	loaderData, err := ioutil.ReadFile(loaderPath)
	if err != nil {
		return err
	}
	resp, err := rpc.RegisterExtension(context.Background(), &sliverpb.RegisterExtensionReq{
		Name:    coffLoaderName,
		Data:    loaderData,
		OS:      implant.GetOS(),
		Init:    "",
		Request: MakeRequest(implant),
	})
	if err != nil {
		return err
	}
	if resp.Response != nil && resp.Response.Err != "" {
		return errors.New(resp.Response.Err)
	}
	return nil
}

func isLoaderLoaded(implant ActiveImplant, rpc rpcpb.SliverRPCClient) bool {
	extList, err := rpc.ListExtensions(context.Background(), &sliverpb.ListExtensionsReq{
		Request: MakeRequest(implant),
	})
	if err != nil {
		return false
	}
	for _, ext := range extList.Names {
		if ext == coffLoaderName {
			return true
		}
	}
	return false
}
