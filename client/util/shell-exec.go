package util

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
	"os"
	"os/exec"

	"github.com/go-cmd/cmd"
)

// Shell - Use the system shell transparently through the console
func Shell(args []string) error {
	err := ShellExec(args[0], args[1:])
	if err != nil {
		fmt.Printf(CommandError+"%s \n", err.Error())
		return nil
	}
	return nil
}

// ShellExec - Execute a program via OS, with realtime output
func ShellExec(executable string, args []string) (err error) {

	// Check executable exists
	path, err := exec.LookPath(executable)
	if err != nil {
		return err
	}

	// No output buffering, enable streaming (print output in real time)
	cmdOptions := cmd.Options{
		Buffered:  false,
		Streaming: true,
	}

	// Prepare the command
	runCmd := cmd.NewCmdOptions(cmdOptions, path, args...)

	// Load OS environment
	runCmd.Env = os.Environ()

	// Print Stdout & Stderr lines in real time
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Done when both channels have been closed.
		// https://dave.cheney.net/2013/04/30/curious-channels
		for runCmd.Stdout != nil || runCmd.Stderr != nil {
			select {
			case line, open := <-runCmd.Stdout:
				if !open {
					runCmd.Stdout = nil
					continue
				}
				fmt.Println(line)
			case line, open := <-runCmd.Stderr:
				if !open {
					runCmd.Stderr = nil
					continue
				}
				fmt.Println(line)
				// fmt.Fprintln(os.Stderr, line)
			}
		}
	}()

	// Run and wait for command to return, discard exit status
	<-runCmd.Start()

	// Wait for goroutine to print everything
	<-done

	return
}

// inputIsBinary - Check if first input is a system program
func inputIsBinary(args []string) bool {
	_, err := exec.LookPath(args[0])
	if err != nil {
		return false
	}
	return true
}
