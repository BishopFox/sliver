package handlers

import (
	"log"
	consts "sliver/client/constants"
	pb "sliver/protobuf/client"
	"sliver/server/core"

	"github.com/golang/protobuf/proto"
)

// RPCResponse - Called with response data, mapped back to reqID
type RPCResponse func([]byte)

// RPCHandler - RPC handlers accept bytes and return bytes
type RPCHandler func([]byte, RPCResponse)

var (
	rpcHandlers = map[string]RPCHandler{
		consts.SessionsStr: rpcSessions,
	}
)

// GetRPCHandlers - Returns a map of server-side msg handlers
func GetRPCHandlers() map[string]RPCHandler {
	return rpcHandlers
}

func rpcSessions(_ []byte, resp RPCResponse) {
	sessions := &pb.Sessions{}
	if 0 < len(*core.Hive.Slivers) {
		for _, sliver := range *core.Hive.Slivers {
			sessions.Slivers = append(sessions.Slivers, &pb.Sliver{
				ID:            int32(sliver.ID),
				Name:          sliver.Name,
				Hostname:      sliver.Hostname,
				Username:      sliver.Username,
				UID:           sliver.UID,
				GID:           sliver.GID,
				OS:            sliver.Os,
				Arch:          sliver.Arch,
				Transport:     sliver.Transport,
				RemoteAddress: sliver.RemoteAddress,
				PID:           sliver.PID,
				Filename:      sliver.Filename,
			})
		}
	}
	data, err := proto.Marshal(sessions)
	if err != nil {
		log.Printf("Error encoding rpc response %v", err)
	}
	resp(data)
}
