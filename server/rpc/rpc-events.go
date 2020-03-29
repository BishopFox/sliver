package rpc

import (
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
)

var (
	rpcEventsLog = log.NamedLogger("rpc", "events")
)

// Events - Stream events to client
func (s *Server) Events(_ *commonpb.Empty, stream rpcpb.SliverRPC_EventsServer) error {
	events := core.EventBroker.Subscribe()
	for event := range events {
		pbEvent := &clientpb.Event{
			EventType: event.EventType,
			Data:      event.Data,
		}

		if event.Job != nil {
			pbEvent.Job = event.Job.ToProtobuf()
		}
		if event.Client != nil {
			pbEvent.Client = event.Client.ToProtobuf()
		}
		if event.Session != nil {
			pbEvent.Session = event.Session.ToProtobuf()
		}

		err := stream.Send(pbEvent)
		if err != nil {
			rpcEventsLog.Warnf(err.Error())
		}
	}
	return nil
}
