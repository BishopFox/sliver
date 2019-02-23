package handlers

import "sliver/sliver/transports"

type RPCResponse func([]byte, error)
type RPCHandler func([]byte, RPCResponse)
type TunnelHandler func([]byte, *transports.Connection)
