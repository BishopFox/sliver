package handlers

import (
	pb "sliver/protobuf/sliver"
)

var (
	tunnelHandlers = &map[uint64]TunnelHandler{
		pb.MsgTunnelData: tunnelDataHandler,
	}
)

func GetTunnelHandlers() *map[uint64]TunnelHandler {
	return tunnelHandlers
}
