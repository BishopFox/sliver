package main

import pb "sliver/protobuf"

var (
	linuxHandlers = map[string]interface{}{
		pb.MsgPsListReq: psHandler,
	}
)

func getSystemHandlers() map[string]interface{} {
	return linuxHandlers
}
