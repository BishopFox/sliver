package executor

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
	"fmt"
	"os"
	"strings"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/extensions"
	"github.com/bishopfox/sliver/client/prelude/bridge"
	"github.com/bishopfox/sliver/client/prelude/config"
	"github.com/bishopfox/sliver/client/prelude/implant"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

type extensionMessage struct {
	Name      string        `json:"Name"`
	Arguments []interface{} `json:"Arguments"`
}

type bofArg struct {
	ArgType string      `json:"type"`
	Value   interface{} `json:"value"`
}

func runExtension(message interface{}, _ []byte, impBridge *bridge.OperatorImplantBridge, cb func(string, int, int), outputFormat string) (string, int, int) {
	var (
		msg     extensionMessage
		extName string
		export  string
		extArgs []byte
	)
	extArgsInt, ok := message.(map[string](interface{}))["Arguments"].([]interface{})
	if !ok {
		return sendError(errors.New("missing extension arguments"))
	}
	extensionName, ok := message.(map[string](interface{}))["Name"].(string)
	if !ok {
		return sendError(errors.New("missing extension name"))
	}
	msg = extensionMessage{
		Name:      extensionName,
		Arguments: extArgsInt,
	}

	ext, err := extensions.GetLoadedExtension(msg.Name)
	if err != nil {
		return sendError(err)
	}
	// Load extension into implant
	loadExtRequest := implant.MakeRequest(impBridge.Implant)
	if loadExtRequest == nil {
		return sendError(errors.New("could not create RPC request"))
	}
	err = extensions.LoadExtension(impBridge.Implant.GetOS(), impBridge.Implant.GetArch(), true, ext, loadExtRequest, impBridge.RPC)
	if err != nil {
		return sendError(err)
	}
	// Determine whether the extensions has dependencies (BOF),
	// if so, get dependency name and extension file

	if ext.DependsOn != "" {
		depExt, err := extensions.GetLoadedExtension(ext.DependsOn)
		if err != nil {
			return sendError(err)
		}
		extName = depExt.CommandName
		export = depExt.Entrypoint
	} else {
		extName = ext.CommandName
		export = ext.Entrypoint
	}
	// Build the arguments param (depending if BOF or not)
	extFilePath, err := ext.GetFileForTarget(ext.CommandName, impBridge.Implant.GetOS(), impBridge.Implant.GetArch())
	if err != nil {
		return sendError(err)
	}
	if strings.HasSuffix(extFilePath, ".o") {
		// We have a BOF
		extData, err := os.ReadFile(extFilePath)
		if err != nil {
			return sendError(err)
		}
		extArgs, err = parseBOFArgs(extData, ext, msg.Arguments)
		if err != nil {
			return sendError(err)
		}
	} else {
		// We have a regular extension
		var extArgStr []string
		for _, arg := range msg.Arguments {
			converted, ok := arg.(string)
			if !ok {
				return sendError(errors.New("arguments must be strings"))
			}
			extArgStr = append(extArgStr, converted)
		}
		extArgsLst := []byte(strings.Join(extArgStr, " "))
		extArgs = []byte(extArgsLst)
	}

	// Call extension
	callResp, err := impBridge.RPC.CallExtension(context.Background(), &sliverpb.CallExtensionReq{
		Name:        extName,
		ServerStore: true,
		Export:      export,
		Request:     implant.MakeRequest(impBridge.Implant),
		Args:        extArgs,
	})

	if err != nil {
		return sendError(err)
	}
	if callResp.Response != nil && callResp.Response.Async {
		impBridge.BeaconCallback(callResp.Response.TaskID, func(task *clientpb.BeaconTask) {
			err := proto.Unmarshal(task.Response, callResp)
			if err != nil {
				cb(sendError(err))
				return
			}
			cb(handleExtensionOutput(callResp, int(impBridge.Implant.GetPID()), outputFormat))
		})
		return "", config.SuccessExitStatus, int(impBridge.Implant.GetPID())
	}
	if callResp.Response != nil && callResp.Response.Err != "" {
		return sendError(errors.New(callResp.Response.Err))
	}

	return handleExtensionOutput(callResp, int(impBridge.Implant.GetPID()), outputFormat)
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

func handleExtensionOutput(callExt *sliverpb.CallExtension, pid int, format string) (string, int, int) {
	if format == "json" {
		return JSONFormatter(callExt, pid)
	}
	return string(callExt.Output), config.SuccessExitStatus, pid
}
