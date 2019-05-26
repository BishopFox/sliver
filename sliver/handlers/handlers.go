package handlers

import (
	pb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/sliver/transports"
)

type RPCResponse func([]byte, error)
type RPCHandler func([]byte, RPCResponse)
type SpecialHandler func([]byte, *transports.Connection) error
type TunnelHandler func(*pb.Envelope, *transports.Connection)
