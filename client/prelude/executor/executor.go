package executor

import (
	"encoding/json"

	"github.com/bishopfox/sliver/client/prelude/bridge"
	"github.com/bishopfox/sliver/client/prelude/config"
	"google.golang.org/protobuf/proto"
)

type Formatter func(msg proto.Message, pid int) (string, int, int)
type handler func(interface{}, []byte, *bridge.OperatorImplantBridge, func(string, int, int), string) (string, int, int)

// TODO: populate
var handlers = map[string]handler{
	"ls":                LsHandler,
	"download":          nil,
	"upload":            nil,
	"execute-assembly":  executeAssemblyHandler,
	"execute-shellcode": nil,
	"sideload":          nil,
	"spawndll":          nil,
	"run-extension":     runExtension,
}

func GetHandlers() map[string]handler {
	return handlers
}

func GetHandler(name string) handler {
	h, ok := handlers[name]
	if ok {
		return h
	}
	return nil
}

func JSONFormatter(msg proto.Message, pid int) (string, int, int) {
	out, err := json.Marshal(msg)
	if err != nil {
		return "", config.ErrorExitStatus, config.ErrorExitStatus
	}
	return string(out), config.SuccessExitStatus, pid
}

func sendError(err error) (string, int, int) {
	return err.Error(), config.ErrorExitStatus, config.ErrorExitStatus
}
