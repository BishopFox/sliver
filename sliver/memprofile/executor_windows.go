package memprofile

import (
	"github.com/bishopfox/sliver/sliver/syscalls"
	"golang.org/x/sys/windows"
)

// GenericExecutor stores the basic info required for
// a memory executor
type GenericExecutor struct {
	Process   windows.Handle
	StartAddr uintptr
	ArgsAddr  uintptr
}

// ThreadExecutor implements the MemExecutor interface
// by using CreateRemoteThread
type ThreadExecutor struct {
	GenericExecutor
	ThreadID *uint32
	Thread   windows.Handle
}

func (e *ThreadExecutor) Execute() (err error) {
	var lpThreadID uint32
	e.Thread, err = syscalls.CreateRemoteThread(e.Process, nil, uint32(0), e.StartAddr, e.ArgsAddr, 0, &lpThreadID)
	if err != nil {
		return
	}
	e.ThreadID = lpThreadID
	return
}

type APCExecutor struct {
	GenericExecutor
	Thread windows.Handle
}

func (e *APCExecutor) Execute() (err error) {
	err = QueueUserAPC(e.StartAddr, e.Thread, 0)
	if err != nil {
		return
	}
	err = windows.ResumeThread(e.Thread)
	if err != nil {
		return
	}
	err = windows.CloseHandle(e.Thread)
	return
}
