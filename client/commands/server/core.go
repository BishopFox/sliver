package server

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
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/evilsocket/islazy/fs"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
)

// Exit - Kill the current client console
type Exit struct{}

// Execute - Run
func (e *Exit) Execute(args []string) (err error) {

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Confirm exit (Y/y): ")
	text, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(text)

	if (answer == "Y") || (answer == "y") {
		transport.CloseClientConnGRPC()
		os.Exit(0)
	}

	fmt.Println()
	return
}

// ChangeClientDirectory - Change the working directory of the client console
type ChangeClientDirectory struct {
	Positional struct {
		Path string `description:"local path" required:"1-1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Handler for ChangeDirectory
func (cd *ChangeClientDirectory) Execute(args []string) (err error) {

	dir, err := fs.Expand(cd.Positional.Path)

	err = os.Chdir(dir)
	if err != nil {
		fmt.Printf(util.CommandError+"%s \n", err)
	} else {
		fmt.Printf(util.Info+"Changed directory to %s \n", dir)
	}

	return
}

// ListClientDirectories - List directory contents
type ListClientDirectories struct {
	Positional struct {
		Path []string `description:"local directory/file"`
	} `positional-args:"yes"`
}

// Execute - Command
func (ls *ListClientDirectories) Execute(args []string) error {

	base := []string{"ls", "--color", "-l"}

	if len(ls.Positional.Path) == 0 {
		ls.Positional.Path = []string{"."}
	}

	fullPaths := []string{}
	for _, path := range ls.Positional.Path {
		full, _ := fs.Expand(path)
		fullPaths = append(fullPaths, full)
	}
	base = append(base, fullPaths...)

	err := util.Shell(base)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}
