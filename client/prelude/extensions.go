package prelude

import (
	"encoding/json"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

//{"Name":"coff-loader", "ServerStore":false, "Args":"PAYLOAD.BYTES", "Export":"LoadAndRun", "arguments": [{"type": "wstring", "value": "C:\Users\lab\Downloads\RELIEVED_TV.exe"}]}
type extensionMessage struct {
	Name        string    `json:"Name"`
	ServerStore bool      `json:"ServerStore"`
	Args        string    `json:"Args"`
	Export      string    `json:"Export"`
	Arguments   []bofArgs `json:"arguments"`
}

func runExtension(message string, payload []byte, activeImplant ActiveImplant, rpc rpcpb.SliverRPCClient, onFinish func(string, int, int)) (string, int, int) {
	var msg extensionMessage
	err := json.Unmarshal([]byte(message), &msg)
	if err != nil {
		println(message)
		return err.Error(), ErrorExitStatus, ErrorExitStatus
	}
	if msg.Args == "PAYLOAD.BYTES" {
		if err != nil {
			return err.Error(), ErrorExitStatus, ErrorExitStatus
		}
		out, err := runBOF(activeImplant, rpc, payload, msg.Arguments, onFinish)
		if err != nil {
			return err.Error(), ErrorExitStatus, ErrorExitStatus
		}
		return out, SuccessExitStatus, SuccessExitStatus
	}
	return "Unsupported extension", ErrorExitStatus, ErrorExitStatus
}
