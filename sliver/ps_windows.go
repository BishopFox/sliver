// +build windows
package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Windows API functions
var (
	modKernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procCloseHandle              = modKernel32.NewProc("CloseHandle")
	procCreateToolhelp32Snapshot = modKernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = modKernel32.NewProc("Process32FirstW")
	procProcess32Next            = modKernel32.NewProc("Process32NextW")
)

// Some constants from the Windows API
const (
	ERROR_NO_MORE_FILES       = 0x12
	MAX_PATH                  = 260
	PROCESS_QUERY_INFORMATION = 0x0400
	PROCESS_VM_READ           = 0x0010
	TOKEN_QUERY               = 0x0008
)

// PROCESSENTRY32 is the Windows API structure that contains a process's
// information.
type PROCESSENTRY32 struct {
	Size              uint32
	CntUsage          uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	CntThreads        uint32
	ParentProcessID   uint32
	PriorityClassBase int32
	Flags             uint32
	ExeFile           [MAX_PATH]uint16
}

// WindowsProcess is an implementation of Process for Windows.
type WindowsProcess struct {
	pid   int
	ppid  int
	exe   string
	owner string
}

func (p *WindowsProcess) Pid() int {
	return p.pid
}

func (p *WindowsProcess) PPid() int {
	return p.ppid
}

func (p *WindowsProcess) Executable() string {
	return p.exe
}

func (p *WindowsProcess) Owner() string {
	return p.owner
}

func newWindowsProcess(e *PROCESSENTRY32) *WindowsProcess {
	// Find when the string ends for decoding
	end := 0
	for {
		if e.ExeFile[end] == 0 {
			break
		}
		end++
	}
	account, _ := getProcessOwner(e.ProcessID)

	return &WindowsProcess{
		pid:   int(e.ProcessID),
		ppid:  int(e.ParentProcessID),
		exe:   syscall.UTF16ToString(e.ExeFile[:end]),
		owner: account,
	}
}

func findProcess(pid int) (Process, error) {
	ps, err := processes()
	if err != nil {
		return nil, err
	}

	for _, p := range ps {
		if p.Pid() == pid {
			return p, nil
		}
	}

	return nil, nil
}

// getInfo retrieves a specified type of information about an access token.
func getInfo(t syscall.Token, class uint32, initSize int) (unsafe.Pointer, error) {
	n := uint32(initSize)
	for {
		b := make([]byte, n)
		e := syscall.GetTokenInformation(t, class, &b[0], uint32(len(b)), &n)
		if e == nil {
			return unsafe.Pointer(&b[0]), nil
		}
		if e != syscall.ERROR_INSUFFICIENT_BUFFER {
			return nil, e
		}
		if n <= uint32(len(b)) {
			return nil, e
		}
	}
}

// getTokenOwner retrieves access token t owner account information.
func getTokenOwner(t syscall.Token) (*syscall.Tokenuser, error) {
	i, e := getInfo(t, syscall.TokenOwner, 50)
	if e != nil {
		return nil, e
	}
	return (*syscall.Tokenuser)(i), nil
}

func getProcessOwner(pid uint32) (owner string, err error) {
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, pid)
	if err != nil {
		return
	}
	var token syscall.Token
	if err = syscall.OpenProcessToken(handle, syscall.TOKEN_QUERY, &token); err != nil {
		return
	}
	tokenUser, err := getTokenOwner(token)
	if err != nil {
		return
	}
	owner, domain, _, err := tokenUser.User.Sid.LookupAccount("")
	owner = fmt.Sprintf("%s\\%s", domain, owner)
	return
}

func processes() ([]Process, error) {
	handle, _, _ := procCreateToolhelp32Snapshot.Call(
		0x00000002,
		0)
	if handle < 0 {
		return nil, syscall.GetLastError()
	}
	defer procCloseHandle.Call(handle)

	var entry PROCESSENTRY32
	entry.Size = uint32(unsafe.Sizeof(entry))
	ret, _, _ := procProcess32First.Call(handle, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return nil, fmt.Errorf("Error retrieving process info.")
	}

	results := make([]Process, 0, 50)
	for {
		results = append(results, newWindowsProcess(&entry))

		ret, _, _ := procProcess32Next.Call(handle, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}

	return results, nil
}
