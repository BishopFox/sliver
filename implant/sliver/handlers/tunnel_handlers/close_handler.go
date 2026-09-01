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
	"errors"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/shell"
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

const tunnelTerminalCloseTimeout = 10 * time.Second

func closeTunnelLocal(connection *transports.Connection, tunnel *transports.Tunnel) bool {
	if connection == nil || tunnel == nil {
		return false
	}
	shell.StopSession(tunnel.ID)
	return connection.CloseTunnelLocal(tunnel)
}

func closeTunnelRemote(connection *transports.Connection, tunnel *transports.Tunnel) bool {
	if connection == nil || tunnel == nil {
		return false
	}
	shell.StopSession(tunnel.ID)
	return connection.CloseTunnelRemote(tunnel)
}

func TunnelCloseHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	tunnelClose := &sliverpb.TunnelData{
		Closed: true,
	}
	if err := proto.Unmarshal(envelope.Data, tunnelClose); err != nil || !tunnelClose.Closed {
		return
	}
	handleTunnelClose(tunnelClose, connection, tunnelTerminalCloseTimeout)
}

func handleTunnelClose(tunnelClose *sliverpb.TunnelData, connection *transports.Connection, timeout time.Duration) {
	if tunnelClose == nil || connection == nil {
		return
	}
	tunnel := connection.Tunnel(tunnelClose.TunnelID)
	if tunnel != nil {
		// {{if .Config.Debug}}
		log.Printf("[tunnel] Closing tunnel with id %d", tunnel.ID)
		// {{end}}
		// Sequence zero is the legacy protocol's immediate terminal. It must not
		// wait behind an inbound destination write: closing the writer is what
		// unblocks a stalled shell, port-forward, or WASM handler.
		if tunnelClose.Sequence == 0 {
			closeTunnelRemote(connection, tunnel)
			return
		}

		// A sequenced terminal normally waits for every lower data frame. Arm the
		// fail-closed deadline first because MarkPeerClose serializes with the
		// inbound writer and that writer may be non-cooperative until Close.
		tunnel.StartPeerCloseDeadline(timeout, func() {
			if closeTunnelRemote(connection, tunnel) {
				connection.Cleanup()
			}
		})
		ready, err := tunnel.MarkPeerClose(tunnelClose.Sequence)
		if err != nil {
			if errors.Is(err, transports.ErrTunnelClosed) {
				return
			}
			if closeTunnelRemote(connection, tunnel) {
				connection.Cleanup()
			}
			return
		}
		if ready {
			closeTunnelRemote(connection, tunnel)
			return
		}
	} else {
		// Preserve the startup tombstone used when a close overtakes shell
		// publication; StopSession is a no-op for non-shell tunnel IDs.
		shell.StopSession(tunnelClose.TunnelID)
		// {{if .Config.Debug}}
		log.Printf("[tunnel][tunnelCloseHandler] Received close message for unknown tunnel id %d", tunnelClose.TunnelID)
		// {{end}}
	}
}
