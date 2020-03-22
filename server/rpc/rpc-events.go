package rpc

import (
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

// SubscribeEvents - Stream events to client
func (s *Server) SubscribeEvents(_ *commonpb.Empty, _ rpcpb.SliverRPC_SubscribeEventsServer) error {
	return nil
}
