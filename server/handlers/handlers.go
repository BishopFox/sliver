package main

import (
	"log"
	pb "sliver/protobuf"

	"github.com/golang/protobuf/proto"
)

var (
	serverHandlers = map[string]interface{}{
		pb.MsgRegister: registerSliverHandler,
	}
)

func getServerHandlers() map[string]interface{} {
	return serverHandlers
}

func registerSliverHandler(sliver *Sliver, data []byte) {
	register := &pb.Register{}
	err := proto.Unmarshal(data, register)
	if err != nil {
		log.Printf("error decoding message: %v", err)
		return
	}

	// If this is the first time we're getting reg info alert user(s)
	if sliver.Name == "" {
		defer func() { events <- Event{Sliver: sliver, EventType: "connected"} }()
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
