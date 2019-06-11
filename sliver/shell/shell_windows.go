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

var (
	// Shell constants
	commandPrompt = []string{"C:\\Windows\\System32\\cmd.exe"}
	powerShell    = []string{
		"C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe",
		"-NoExit",
		"-WindowStyle", "Hidden",
		"-Command", "[Console]::OutputEncoding=[Text.UTF8Encoding]::UTF8",
	}
)

// GetSystemShellPath - Find powershell or cmd
func GetSystemShellPath() []string {
	if exists(powerShell[0]) {
		return powerShell
	}
	return commandPrompt
}
