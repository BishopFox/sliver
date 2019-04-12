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
		taskrunner.RefreshPE(`c:\windows\system32\ntdll.dll`)
		taskrunner.RefreshPE(`c:\windows\system32\dbgcore.dll`)
		return minidump(int(pid), int(hProc))
	}
	return res, fmt.Errorf("Could not dump process memory")
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
