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

	"github.com/evilsocket/islazy/str"
)

// Shell - Use the system shell transparently through the console
func Shell(args []string) error {
	out, err := Exec(args[0], args[1:])
	if err != nil {
		fmt.Printf(CommandError+"%s \n", err.Error())
		return nil
	}

	// Print output
	fmt.Println(out)

	return nil
}

// Exec - Execute a program
func Exec(executable string, args []string) (string, error) {
	path, err := exec.LookPath(executable)
	if err != nil {
		return "", err
	}

	cmd := exec.Command(path, args...)

	// Load OS environment
	cmd.Env = os.Environ()

	out, err := cmd.CombinedOutput()

	if err != nil {
		return "", err
	}
	return str.Trim(string(out)), nil
}

// inputIsBinary - Check if first input is a system program
func inputIsBinary(args []string) bool {
	_, err := exec.LookPath(args[0])
	if err != nil {
		return false
	}
	return true
}
