package handlers

import (
	"log"
	consts "sliver/client/constants"
	sliverpb "sliver/protobuf/sliver"
	"sliver/server/core"

	"github.com/golang/protobuf/proto"
)

var (
	serverHandlers = map[uint32]interface{}{
		sliverpb.MsgRegister:   registerSliverHandler,
		sliverpb.MsgTunnelData: tunnelDataHandler,
	}
)

// GetSliverHandlers - Returns a map of server-side msg handlers
func GetSliverHandlers() map[uint32]interface{} {
	return serverHandlers
}

func registerSliverHandler(sliver *core.Sliver, data []byte) {
	register := &sliverpb.Register{}
	err := proto.Unmarshal(data, register)
	if err != nil {
		log.Printf("error decoding message: %v", err)
		return
	}

	// If this is the first time we're getting reg info alert user(s)
	if sliver.Name == "" {
		defer func() {
			core.EventBroker.Publish(core.Event{
				EventType: consts.ConnectedEvent,
				Sliver:    sliver,
			})
		}()
	}

	sliver.Name = register.Name
	sliver.Hostname = register.Hostname
	sliver.Username = register.Username
	sliver.UID = register.Uid
	sliver.GID = register.Gid
	sliver.Os = register.Os
	sliver.Arch = register.Arch
	sliver.PID = register.Pid
	sliver.Filename = register.Filename
}

func tunnelDataHandler(sliver *core.Sliver, data []byte) {
	tunnelData := &sliverpb.TunnelData{}
	proto.Unmarshal(data, tunnelData)
	tunnel := core.Tunnels.Tunnel(tunnelData.TunnelID)
	if tunnel != nil && sliver.ID == tunnel.Sliver.ID {
		tunnel.Client.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgTunnelData,
			Data: data,
		}
	} else {
		log.Printf("Data sent on nil tunnel %d", tunnelData.TunnelID)
	}
}
