package prelude

import (
	"bytes"
	"context"
	"encoding/json"
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
		extArgs, err = parseBOFArgs(extData, msg.Args)
		if err != nil {
			return err.Error(), ErrorExitStatus, ErrorExitStatus
		}
	} else {
		// We have a regular extension
		extArgStr := make([]string, len(msg.Args))
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

func parseBOFArgs(extData []byte, args []interface{}) ([]byte, error) {
	var (
		err error
	)
	bArgs := make([]bofArg, len(args))
	for _, arg := range args {
		var bArg bofArg
		err := json.Unmarshal([]byte(arg.(string)), &bArg)
		if err != nil {
			return nil, err
		}
		bArgs = append(bArgs, bArg)
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
	err = extArgs.AddString(bofEntryPoint)
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
