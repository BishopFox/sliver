package handlers

import (
	"os"
	pb "sliver/protobuf/sliver"
	"sliver/sliver/transports"

	"github.com/golang/protobuf/proto"
)

var specialHandlers = map[uint32]SpecialHandler{
	pb.MsgKill: killHandler,
}

// GetSpecialHandlers returns the specialHandlers map
func GetSpecialHandlers() map[uint32]SpecialHandler {
	return specialHandlers
}

func killHandler(data []byte, connection *transports.Connection) error {
	killReq := &pb.KillReq{}
	err := proto.Unmarshal(data, killReq)
	// {{if .Debug}}
	println("KILL called")
	// {{end}}
	if err != nil {
		return err
	}
	// Exit now if we've received a force request
	if killReq.Force {
		os.Exit(0)
	}
	// Cleanup connection
	connection.Cleanup()
	// {{if .Debug}}
	println("Let's exit!")
	// {{end}}
	os.Exit(0)
	return nil
}
