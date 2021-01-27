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
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/evilsocket/islazy/tui"
	"github.com/jessevdk/go-flags"

	"github.com/bishopfox/sliver/client/help"
	"github.com/bishopfox/sliver/client/util"
)

// LoadExtension - Load an extension onto the current session host.
type LoadExtension struct {
	Positional struct {
		Path string `description:"directory path to extension file" required:"1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Load an extension onto the current session host.
// The commands' end result is to return a function containing a complete command,
// and this function will be called along each refresh of the client parser commands.
func (l *LoadExtension) Execute(args []string) (err error) {

	// retrieve extension manifest
	manifestPath := fmt.Sprintf("%s/%s", l.Positional.Path, "manifest.json")
	jsonBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		fmt.Printf(util.Error+"%v", err)
	}
	// parse it
	ext := &extension{}
	err = json.Unmarshal(jsonBytes, ext)
	if err != nil {
		fmt.Printf(util.Error+"error loading extension: %v\n", err)
		return
	}
	ext.Path = l.Positional.Path

	// If the root extension name is already taken
	// by another command, return and notify
	for _, c := range Sliver.Commands() {
		if ext.Name == c.Name {
			fmt.Printf(util.Error+"Error loading extension: another command has name %s\n",
				tui.Yellow(ext.Name))
			return nil
		}
	}

	// We set up a function that will bind the commands to
	// the Sliver parser any number of times it needs them.
	var bindExtensionCommands = func() {

		// There is always at least one root command
		root, _ := Sliver.AddCommand(ext.Name,
			fmt.Sprintf("%s extension commands", ext.Name), "",
			&ExtensionCommand{})
		root.Aliases = []string{"extensions"}

		// For each extension command, add a new subcommand.
		for _, extCmd := range ext.Commands {

			// do not add if the command already exists
			// Try to load all other commands still.
			if cmdExists(root, extCmd.Name) {
				continue
			}

			// Add the subcommand
			sub, _ := Sliver.AddCommand(extCmd.Name,
				extCmd.Help,
				help.FormatHelpTmpl(extCmd.LongHelp),
				&ExtensionCommand{root: ext, sub: &extCmd},
			)
			sub.Aliases = []string{fmt.Sprintf("%s commands", root.Name)}

			// Add base & || assembly options
			sub.AddGroup("base options", "", &ExtensionOptions{})
			if extCmd.IsAssembly {
				sub.AddGroup("assembly options", "", &ExtensionLibraryOptions{})
			}
		}
	}

	// Assign the function, can now be called on each command input loop.
	LoadedExtensions[ext.Path] = bindExtensionCommands

	return
}

// -------------------------------------------------------------------------------------------
//                                  Extension Loading Code                                  //
// -------------------------------------------------------------------------------------------

const (
	windowsDefaultHostProc = `c:\windows\system32\notepad.exe`
	linuxDefaultHostProc   = "/bin/bash"
	macosDefaultHostProc   = "/Applications/Safari.app/Contents/MacOS/SafariForWebKitDevelopment"
)

var commandMap map[string]extension
var defaultHostProc = map[string]string{
	"windows": windowsDefaultHostProc,
	"linux":   windowsDefaultHostProc,
	"darwin":  macosDefaultHostProc,
}

type binFiles struct {
	Ext64Path string `json:"x64"`
	Ext32Path string `json:"x86"`
}

type extFile struct {
	OS    string   `json:"os"`
	Files binFiles `json:"files"`
}

type extensionCommand struct {
	Name           string    `json:"name"`
	Entrypoint     string    `json:"entrypoint"`
	Help           string    `json:"help"`
	LongHelp       string    `json:"longHelp"`
	AllowArgs      bool      `json:"allowArgs"`
	DefaultArgs    string    `json:"defaultArgs"`
	ExtensionFiles []extFile `json:"extFiles"`
	IsReflective   bool      `json:"isReflective"`
	IsAssembly     bool      `json:"IsAssembly"`
}

func (ec *extensionCommand) getDefaultProcess(targetOS string) (proc string, err error) {
	proc, ok := defaultHostProc[targetOS]
	if !ok {
		err = fmt.Errorf("no default process for %s target, please specify one", targetOS)
	}
	return
}

type extension struct {
	Name     string             `json:"extensionName"`
	Commands []extensionCommand `json:"extensionCommands"`
	Path     string
}

func (e *extension) getFileForTarget(cmdName string, targetOS string, targetArch string) (filePath string, err error) {
	for _, c := range e.Commands {
		if cmdName == c.Name {
			for _, ef := range c.ExtensionFiles {
				if targetOS == ef.OS {
					switch targetArch {
					case "x86":
						filePath = fmt.Sprintf("%s/%s", e.Path, ef.Files.Ext32Path)
					case "x64":
						filePath = fmt.Sprintf("%s/%s", e.Path, ef.Files.Ext64Path)
					default:
						filePath = fmt.Sprintf("%s/%s", e.Path, ef.Files.Ext64Path)
					}
				}
			}

		}
	}
	if filePath == "" {
		err = fmt.Errorf("no extension file found for %s/%s", targetOS, targetArch)
	}
	return
}

func (e *extension) getCommandFromName(name string) (extCmd *extensionCommand, err error) {
	for _, x := range e.Commands {
		if x.Name == name {
			extCmd = &x
			return
		}
	}
	err = fmt.Errorf("no extension command found for name %s", name)
	return
}

func cmdExists(cmd *flags.Command, name string) bool {
	for _, c := range cmd.Commands() {
		if name == c.Name {
			return true
		}
	}
	return false
}

func init() {
	commandMap = make(map[string]extension)
}
