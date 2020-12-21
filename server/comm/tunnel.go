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
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"gopkg.in/djherbis/buffer.v1"
	"gopkg.in/djherbis/nio.v2"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
)

// tunnel - An ordered stream used by a Comm to run an SSH multiplexing session.
// This is used for all implants whose' active C2 protocol stack is not able to yield a net.Conn,
// such as DNS or procedurally-generated HTTP(S). This object implements the net.Conn interface.
type tunnel struct {
	ID   uint64
	Sess *core.Session

	// Implant conn
	FromImplant         chan *sliverpb.CommTunnelData
	FromImplantSequence uint64
	ToImplant           chan []byte
	ToImplantSequence   uint64
	cache               map[uint64]*sliverpb.CommTunnelData

	// Read/Write & buffer
	ConnBuf buffer.Buffer
	Reader  *nio.PipeReader
	Writer  *nio.PipeWriter

	mutex *sync.RWMutex
}

// newTunnel - Setup and start a Comm tunnel
func newTunnel(conn *core.Session) (t *tunnel, err error) {
	t = &tunnel{
		ID:          newTunnelID(),
		Sess:        conn,
		FromImplant: make(chan *sliverpb.CommTunnelData),
		ToImplant:   make(chan []byte, 100),
		cache:       map[uint64]*sliverpb.CommTunnelData{},
		mutex:       &sync.RWMutex{},
	}
	t.ConnBuf = buffer.New(32 * 1024)
	t.Reader, t.Writer = nio.Pipe(t.ConnBuf)

	// Add tunnels to map, so we can receive data
	Tunnels.AddTunnel(t)

	// Ask implant to start tunnel.
	err = t.startRemote()
	if err != nil {
		Tunnels.RemoveTunnel(t.ID)
		return
	}

	// Monitor incoming data
	go t.handleFromImplant()

	return
}

// Read - Implements net.Conn Read(), by reading from the tunnel buffer,
// which is being continuously filled in the background. Blocks when buffer is empty.
func (t *tunnel) Read(data []byte) (n int, err error) {
	rLog.Debugf("SSH called Read() on tunnel")
	n, err = t.Reader.Read(data)
	rLog.Debugf("SSH read %d bytes from tunnel", len(data))
	return
}

// Write - Implements net.Conn Write(), by sending data through the Session's RPC tunnels.
func (t *tunnel) Write(data []byte) (n int, err error) {
	sdata, _ := proto.Marshal(&sliverpb.CommTunnelData{
		Sequence:  t.ToImplantSequence,
		TunnelID:  t.ID,
		SessionID: t.Sess.ID,
		Data:      data,
		Closed:    false,
	})
	t.ToImplantSequence++
	rLog.Debugf("[tunnel] Sequence: %d ", t.ToImplantSequence)
	t.Sess.Send <- &sliverpb.Envelope{
		Type: sliverpb.MsgCommTunnelData,
		Data: sdata,
	}
	rLog.Debugf("[tunnel] To implant %d byte(s)", len(data))
	return len(data), err
}

// Close - Implements net.Conn Close(), by sending a request to the implant to end the tunnel.
func (t *tunnel) Close() error {
	rLog.Debugf("Closing tunnel %d (To Client)", t.ID)
	data, _ := proto.Marshal(&sliverpb.CommTunnelData{
		TunnelID:  t.ID,
		SessionID: t.Sess.ID,
		Data:      make([]byte, 0),
		Closed:    true,
	})
	t.Sess.Send <- &sliverpb.Envelope{
		Type: sliverpb.MsgCommTunnelData,
		Data: data,
	}
	return nil
}

// RemoteAddr - Implements net.Conn RemoteAddr(), reducing the addr to either a TCP/UDP addr (don't need more)
func (t *tunnel) RemoteAddr() (addr net.Addr) {
	c2, _ := url.Parse(t.Sess.ActiveC2)
	host, _ := url.Parse(t.Sess.RemoteAddress)
	switch c2.Scheme {
	case "dns", "named_pipe", "namedpipe":
		port, _ := strconv.Atoi(host.Port())
		addr = &net.UDPAddr{
			IP:   net.ParseIP(host.Host),
			Port: port,
		}
	case "mtls", "tcp", "http", "https":
		port, _ := strconv.Atoi(host.Port())
		addr = &net.TCPAddr{
			IP:   net.ParseIP(host.Host),
			Port: port,
		}
	}
	return
}

// LocalAddr - Implements net.Conn LocalAddr().
func (t *tunnel) LocalAddr() (addr net.Addr) {
	c2, _ := url.Parse(t.Sess.ActiveC2)
	switch c2.Scheme {
	case "dns", "named_pipe", "namedpipe":
		port, _ := strconv.Atoi(c2.Port())
		addr = &net.UDPAddr{
			IP:   net.ParseIP(c2.Host),
			Port: port,
		}
	case "mtls", "tcp", "http", "https":
		port, _ := strconv.Atoi(c2.Port())
		addr = &net.TCPAddr{
			IP:   net.ParseIP(c2.Host),
			Port: port,
		}
	}
	return
}

func (t *tunnel) SetDeadline(d time.Time) error {
	return nil
}

func (t tunnel) SetReadDeadline(rd time.Time) error {
	return nil
}

func (t tunnel) SetWriteDeadline(rwd time.Time) error {
	return nil
}

// startRemote - Request to create and start tunnel
func (t *tunnel) startRemote() (err error) {
	muxOpenReq := &sliverpb.CommTunnelOpenReq{
		TunnelID: t.ID,
		Request:  &commonpb.Request{SessionID: t.Sess.ID},
	}
	reqData, _ := proto.Marshal(muxOpenReq)

	data, err := t.Sess.Request(sliverpb.MsgNumber(muxOpenReq), 60*time.Second, reqData)
	if err != nil {
		return
	}
	resp := &sliverpb.CommTunnelOpen{}
	err = proto.Unmarshal(data, resp)
	if err != nil {
		return
	}
	if !resp.Success {
		return fmt.Errorf("Error starting remote tunnel end: %s", resp.Response.Err)
	}
	return
}

// handleFromImplant - Receives all tunnel data and write it to the buffer in the good order.
func (t *tunnel) handleFromImplant() {
	for data := range t.FromImplant {
		rLog.Debugf("[tunnel] From implant %d byte(s)", len(data.Data))

		t.cache[data.Sequence] = data

		for recv, ok := t.cache[t.FromImplantSequence]; ok; recv, ok = t.cache[t.FromImplantSequence] {

			written, err := t.Writer.Write(recv.Data)
			if err != nil && written == 0 {
				rLog.Debugf("[tunnel] Error writing %d bytes to tunnel %d", len(recv.Data))
			}

			delete(t.cache, t.FromImplantSequence)
			t.FromImplantSequence++
			rLog.Debugf("[tunnel] wrote %d bytes to comm tunnel buffer", len(recv.Data))
			rLog.Debugf("[tunnel] Sequence: %d ", t.FromImplantSequence)
		}
	}
}
