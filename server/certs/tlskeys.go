package certs

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
)

var (
	// TLSKeyLogger - File descriptor for logging TLS keys
	TLSKeyLogger = newKeyLogger()
)

func newKeyLogger() *os.File {
	keyFilePath, present := os.LookupEnv("SSLKEYLOGFILE")
	if present {
		keyFile, err := os.OpenFile(keyFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			certsLog.Errorf(fmt.Sprintf("Failed to open TLS key file %v", err))
			return nil
		}
		fmt.Printf("NOTICE: TLS Keys logged to '%s'\n", keyFilePath)
		return keyFile
	}
	return nil
}
