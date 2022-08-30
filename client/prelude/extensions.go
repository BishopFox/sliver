package prelude

import (
	"encoding/json"

	"github.com/bishopfox/sliver/client/extensions"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

type extensionMessage struct {
	Name        string    `json:"Name"`
	ServerStore bool      `json:"ServerStore"`
	Args        string    `json:"Args"`
	Export      string    `json:"Export"`
	Arguments   []bofArgs `json:"arguments"`
}

func runExtension(message string, activeImplant ActiveImplant, rpc rpcpb.SliverRPCClient, onFinish func(string, int, int)) (string, int, int) {
	var msg extensionMessage
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
	// Call extension

	return "Unsupported extension", ErrorExitStatus, ErrorExitStatus
}
