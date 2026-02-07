//go:build !darwin && !linux && !freebsd && !openbsd && !dragonfly

package tunnel_handlers

import (
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

func ShellResizeReqHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	resp, _ := proto.Marshal(&commonpb.Empty{})
	connection.Send <- &sliverpb.Envelope{
		ID:   envelope.ID,
		Data: resp,
	}
}
