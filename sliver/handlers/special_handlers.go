package handlers

import (
	"os"
	pb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/sliver/transports"

	// {{if .IsSharedLib}}
	"runtime"
	"syscall"

	// {{end}}

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
	// {{if .IsSharedLib}}
	if runtime.GOOS == "windows" {
		// Windows only: ExitThread() instead of os.Exit() for DLL/shellcode slivers
		// so that the parent process is not killed
		exitFunc := syscall.MustLoadDLL("kernel32.dll").MustFindProc("ExitThread")
		exitFunc.Call(uintptr(0))
		return nil
	}
	// {{else}}
	// Exit now if we've received a force request
	if killReq.Force {
		os.Exit(0)
	}
	//{{end}}
	// Cleanup connection
	connection.Cleanup()
	// {{if .Debug}}
	println("Let's exit!")
	// {{end}}
	os.Exit(0)
	return nil
}
