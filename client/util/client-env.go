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

// ClientEnv - Contains all OS environment variables, client-side.
// This is used for things like downloading/uploading files from localhost, etc.,
// therefore we need completion and parsing stuff, sometimes.
var ClientEnv = map[string]string{}

// ParseEnvironmentVariables - Parses a line of input and replace detected environment variables with their values.
func ParseEnvironmentVariables(args []string) (processed []string, err error) {
	return args, nil
}

// handleCuratedVar - Replace an environment variable alone and without any undesired characters attached
func handleCuratedVar(arg []string) (value string) {
	return
}

// handleEmbeddedVar - Replace an environment variable that is in the middle of a path, or other one-string combination
func handleEmbeddedVar(arg []string) (value string) {
	return
}

// LoadClientEnv - Loads all user environment variables
func LoadClientEnv() error {
	return nil
}

//
// // LoadSystemEnv - Loads all system environment variables
// func LoadSystemEnv() error {
//         env := os.Environ()
//
//         for _, kv := range env {
//                 key := strings.Split(kv, "=")[0]
//                 value := strings.Split(kv, "=")[1]
//                 SystemEnv[key] = value
//         }
//         return nil
// }
