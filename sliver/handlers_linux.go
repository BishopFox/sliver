package main

import pb "sliver/protobuf"

var (
	linuxHandlers = map[uint32]RPCHandler{
		pb.MsgPsListReq: psHandler,
	}
)

func getSystemHandlers() map[uint32]RPCHandler {
	return linuxHandlers
}
