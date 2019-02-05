package main

import pb "sliver/protobuf"

var (
	linuxHandlers = map[int32]RPCHandler{
		pb.MsgPsListReq: psHandler,
	}
)

func getSystemHandlers() map[int32]RPCHandler {
	return linuxHandlers
}
