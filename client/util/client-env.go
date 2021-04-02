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
	"os"
	"strings"
)

// ClientEnv - Contains all OS environment variables, client-side.
// This is used for things like downloading/uploading files from localhost, etc.,
// therefore we need completion and parsing stuff, sometimes.
var ClientEnv = map[string]string{}

// ParseEnvironmentVariables - Parses a line of input and replace detected environment variables with their values.
func ParseEnvironmentVariables(args []string) (processed []string, err error) {

	for _, arg := range args {

		// Anywhere a $ is assigned means there is an env variable
		if strings.Contains(arg, "$") || strings.Contains(arg, "~") {

			//Split in case env is embedded in path
			envArgs := strings.Split(arg, "/")

			// If its not a path
			if len(envArgs) == 1 {
				processed = append(processed, handleCuratedVar(arg))
			}

			// If len of the env var split is > 1, its a path
			if len(envArgs) > 1 {
				processed = append(processed, handleEmbeddedVar(arg))
			}
		} else if arg != "" && arg != " " {
			// Else, if arg is not an environment variable, return it as is
			processed = append(processed, arg)
		}

	}
	return
}

// handleCuratedVar - Replace an environment variable alone and without any undesired characters attached
func handleCuratedVar(arg string) (value string) {
	if strings.HasPrefix(arg, "$") && arg != "" && arg != "$" {
		envVar := strings.TrimPrefix(arg, "$")
		val, ok := ClientEnv[envVar]
		if !ok {
			return envVar
		}
		return val
	}
	if arg != "" && arg == "~" {
		return ClientEnv["HOME"]
	}

	return arg
}

// handleEmbeddedVar - Replace an environment variable that is in the middle of a path, or other one-string combination
func handleEmbeddedVar(arg string) (value string) {

	envArgs := strings.Split(arg, "/")
	var path []string

	for _, arg := range envArgs {
		if strings.HasPrefix(arg, "$") && arg != "" && arg != "$" {
			envVar := strings.TrimPrefix(arg, "$")
			val, ok := ClientEnv[envVar]
			if !ok {
				// Err will be caught when command is ran anyway, or completion will stop...
				path = append(path, arg)
			}
			path = append(path, val)
		} else if arg != "" && arg == "~" {
			path = append(path, ClientEnv["HOME"])
		} else if arg != " " && arg != "" {
			path = append(path, arg)
		}
	}

	return strings.Join(path, "/")
}

// LoadClientEnv - Loads all user environment variables
func LoadClientEnv() error {
	env := os.Environ()

	for _, kv := range env {
		key := strings.Split(kv, "=")[0]
		value := strings.Split(kv, "=")[1]
		ClientEnv[key] = value
	}
	return nil
}
