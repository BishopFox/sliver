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
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/extensions"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

type extensionMessage struct {
	Name string        `json:"Name"`
	Args []interface{} `json:"Args"`
}

type bofArg struct {
	ArgType string      `json:"type"`
	Value   interface{} `json:"value"`
}

func runExtension(message string, activeImplant ActiveImplant, rpc rpcpb.SliverRPCClient, onFinish func(string, int, int)) (string, int, int) {
	var (
		msg     extensionMessage
		extName string
		export  string
		extArgs []byte
	)
	err := json.Unmarshal([]byte(message), &msg)
	if err != nil {
		println(message)
		return err.Error(), ErrorExitStatus, ErrorExitStatus
	}
	ext, err := extensions.GetLoadedExtension(msg.Name)
	if err != nil {
		return err.Error(), ErrorExitStatus, ErrorExitStatus
	}
	// Load extension into implant
	loadExtRequest := MakeRequest(activeImplant)
	if loadExtRequest == nil {
		return "could not create RPC request", ErrorExitStatus, ErrorExitStatus
	}
	err = extensions.LoadExtension(activeImplant.GetOS(), activeImplant.GetArch(), true, ext, loadExtRequest, rpc)
	if err != nil {
		return err.Error(), ErrorExitStatus, ErrorExitStatus
	}
	// Determine whether the extensions has dependencies (BOF),
	// if so, get dependency name and extension file

	if ext.DependsOn != "" {
		depExt, err := extensions.GetLoadedExtension(ext.DependsOn)
		if err != nil {
			return err.Error(), ErrorExitStatus, ErrorExitStatus
		}
		extName = depExt.CommandName
		export = depExt.Entrypoint
	} else {
		extName = ext.CommandName
		export = ext.Entrypoint
	}
	// Build the arguments param (depending if BOF or not)
	extFilePath, err := ext.GetFileForTarget(ext.CommandName, activeImplant.GetOS(), activeImplant.GetArch())
	if err != nil {
		return err.Error(), ErrorExitStatus, ErrorExitStatus
	}
	if strings.HasSuffix(".o", extFilePath) {
		// We have a BOF
		extData, err := os.ReadFile(extFilePath)
		if err != nil {
			return err.Error(), ErrorExitStatus, ErrorExitStatus
		}
		extArgs, err = parseBOFArgs(extData, ext, msg.Args)
		if err != nil {
			return err.Error(), ErrorExitStatus, ErrorExitStatus
		}
	} else {
		// We have a regular extension
		var extArgStr []string
		for _, arg := range msg.Args {
			extArgStr = append(extArgStr, arg.(string))
		}
		extArgsLst := []byte(strings.Join(extArgStr, " "))
		extArgs = []byte(extArgsLst)
	}

	// Call extension
	callResp, err := rpc.CallExtension(context.Background(), &sliverpb.CallExtensionReq{
		Name:        extName,
		ServerStore: true,
		Export:      export,
		Request:     MakeRequest(activeImplant),
		Args:        extArgs,
	})

	if err != nil {
		return err.Error(), ErrorExitStatus, ErrorExitStatus
	}
	if callResp.Response != nil && callResp.Response.Async {
		onFinish(string(callResp.Output), SuccessExitStatus, int(activeImplant.GetPID()))
		return "", SuccessExitStatus, int(activeImplant.GetPID())
	}
	return string(callResp.Output), SuccessExitStatus, int(activeImplant.GetPID())
}

func parseBOFArgs(extData []byte, extManifest *extensions.ExtensionManifest, args []interface{}) ([]byte, error) {
	var (
		err   error
		bArgs []bofArg
	)
	for _, arg := range args {
		kv, ok := arg.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid argument: %v", arg)
		}
		bArgs = append(bArgs, bofArg{
			ArgType: kv["type"].(string),
			Value:   kv["value"],
		})
	}
	bofArgs := core.BOFArgsBuffer{
		Buffer: new(bytes.Buffer),
	}

	for _, a := range bArgs {
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
			return nil, nil
		}
	}

	extArgs := core.BOFArgsBuffer{
		Buffer: new(bytes.Buffer),
	}

	parsedArgs, err := bofArgs.GetBuffer()
	if err != nil {
		return nil, err
	}
	err = extArgs.AddString(extManifest.Entrypoint)
	if err != nil {
		return nil, err
	}
	err = extArgs.AddData(extData)
	if err != nil {
		return nil, err
	}
	err = extArgs.AddData(parsedArgs)
	if err != nil {
		return nil, err
	}
	return extArgs.GetBuffer()
}
