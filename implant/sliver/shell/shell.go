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
	"os"
	"os/exec"
)

// Shell - Struct to hold shell related data
type Shell struct {
	ID      uint64
	Command *exec.Cmd
	Stdout  io.ReadCloser
	Stdin   io.WriteCloser
}

// StartAndWait starts a system shell then waits for it to complete
func (s *Shell) StartAndWait() {
	s.Command.Start()
	s.Command.Wait()
}

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
