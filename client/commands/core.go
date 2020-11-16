package commands

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

	"github.com/bishopfox/sliver/client/util"
	"github.com/evilsocket/islazy/fs"
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
		os.Exit(0)
	}

	fmt.Println()
	return
}

// ChangeDirectory - Change the working directory of the client console
type ChangeDirectory struct {
	Positional struct {
		Path string `description:"Local path" required:"1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Handler for ChangeDirectory
func (cd *ChangeDirectory) Execute(args []string) (err error) {

	dir, err := fs.Expand(cd.Positional.Path)

	err = os.Chdir(dir)
	if err != nil {
		fmt.Printf(util.CommandError+"%s \n", err)
	} else {
		fmt.Printf(util.Info+"Changed directory to %s \n", dir)
	}

	return
}

// ListDirectories - List directory contents
type ListDirectories struct {
	Positional struct {
		Path      string   `description:"Local directory/file"`
		OtherPath []string `description:"Local directory/file" `
	} `positional-args:"yes"`
}

// Execute - Command
func (ls *ListDirectories) Execute(args []string) error {

	base := []string{"ls", "--color", "-l"}

	var fullPath string
	if ls.Positional.Path == "" {
		wd, _ := os.Getwd()
		fullPath, _ = fs.Expand(wd)
	} else {
		fullPath, _ = fs.Expand(ls.Positional.Path)
	}

	base = append(base, fullPath)

	fullOtherPaths := []string{}
	for _, path := range ls.Positional.OtherPath {
		full, _ := fs.Expand(path)
		fullOtherPaths = append(fullOtherPaths, full)
	}
	base = append(base, fullOtherPaths...)

	err := util.Shell(base)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}
