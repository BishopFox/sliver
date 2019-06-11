package transport

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"fmt"
	"time"

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/rpc"

	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
)

// LocalClientConnect - Handles local connections to the server console
// keep in mind the arguments to this function are in the context of the client
// so send = "send to server" and recv = "recv from server"
func LocalClientConnect(send, recv chan *sliverpb.Envelope) {
	client := core.GetClient(nil)
	client.Send = recv // Client's recv channel
	core.Clients.AddClient(client)

	go func() {
		rpcHandlers := rpc.GetRPCHandlers()
		tunHandlers := rpc.GetTunnelHandlers()
		for envelope := range send {
			if rpcHandler, ok := (*rpcHandlers)[envelope.Type]; ok {
				timeout := time.Duration(envelope.Timeout)
				go rpcHandler(envelope.Data, timeout, func(data []byte, err error) {
					errStr := ""
					if err != nil {
						errStr = fmt.Sprintf("%v", err)
					}
					client.Send <- &sliverpb.Envelope{
						ID:   envelope.ID,
						Data: data,
						Err:  errStr,
					}
				})
				log.AuditLogger.WithFields(logrus.Fields{
					"operator":      client.Operator,
					"envelope_type": envelope.Type,
				}).Info("rpc command")
			} else if tunHandler, ok := (*tunHandlers)[envelope.Type]; ok {
				go tunHandler(client, envelope.Data, func(data []byte, err error) {
					errStr := ""
					if err != nil {
						errStr = fmt.Sprintf("%v", err)
					}
					client.Send <- &sliverpb.Envelope{
						ID:   envelope.ID,
						Data: data,
						Err:  errStr,
					}
				})
			} else {
				client.Send <- &sliverpb.Envelope{
					ID:   envelope.ID,
					Data: []byte{},
					Err:  "Unknown rpc command",
				}
			}
		}
	}()

	go localEventLoop(client)
}

// Passes along events to the local server console
func localEventLoop(client *core.Client) {
	events := core.EventBroker.Subscribe()
	defer core.EventBroker.Unsubscribe(events)
	for event := range events {
		pbEvent := &clientpb.Event{EventType: event.EventType}

		if event.Job != nil {
			pbEvent.Job = event.Job.ToProtobuf()
		}
		if event.Client != nil {
			pbEvent.Client = event.Client.ToProtobuf()
		}
		if event.Sliver != nil {
			pbEvent.Sliver = event.Sliver.ToProtobuf()
		}
		if event.Err != nil {
			pbEvent.Err = fmt.Sprintf("%v", event.Err)
		}

		data, _ := proto.Marshal(pbEvent)
		client.Send <- &sliverpb.Envelope{
			Type: clientpb.MsgEvent,
			Data: data,
		}
	}
}
