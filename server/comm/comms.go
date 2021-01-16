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
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
)

// Comms - (SSH-multiplexing) ------------------------------------------------------------

var (
	// Comms - All IMPLANT multiplexers currently running in Sliver, providing connection routing.
	// Client comms are NOT stored in this map, because they don't need to be referenced this way.
	Comms = &comms{
		Active: map[uint32]*Comm{},
		mutex:  &sync.RWMutex{},
	}
)

type comms struct {
	Active map[uint32]*Comm
	mutex  *sync.RWMutex
}

// Get - Get a Session Comm by ID
func (c *comms) Get(commID uint32) *Comm {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Active[commID]
}

// GetBySession - Get a Comm by Session ID
func (c *comms) GetBySession(sessID uint32) *Comm {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	for _, comm := range c.Active {
		if comm.session != nil && comm.session.ID == sessID {
			return comm
		}
	}
	return nil
}

// Add - Add a Comm to the map of IMPLANTS' comms
func (c *comms) Add(mux *Comm) *Comm {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Active[mux.ID] = mux
	return mux
}

// Remove - Remove a Comm from the map
func (c *comms) Remove(commID uint32) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	mux := c.Active[commID]
	if mux != nil {
		delete(c.Active, commID)
	}
}

// Tunnels - ReadWriteClosers over Sliver Session ----------------------------------------

var (
	// Tunnels - Stores all Tunnels used by the Comm system to route traffic.
	Tunnels = &tunnels{
		tunnels: map[uint64]*tunnel{},
		mutex:   &sync.RWMutex{},
	}
)

type tunnels struct {
	tunnels map[uint64]*tunnel
	mutex   *sync.RWMutex
}

// Tunnel - Add tunnel to mapping
func (c *tunnels) Tunnel(ID uint64) *tunnel {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.tunnels[ID]
}

// AddTunnel - Add tunnel to mapping
func (c *tunnels) AddTunnel(tun *tunnel) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tunnels[tun.ID] = tun
}

// RemoveTunnel - Add tunnel to mapping
func (c *tunnels) RemoveTunnel(ID uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.tunnels, ID)
}

// newTunnelID - New 64-bit identifier
func newTunnelID() uint64 {
	randBuf := make([]byte, 8)
	rand.Read(randBuf)
	return binary.LittleEndian.Uint64(randBuf)
}

// newID- Returns an incremental nonce as an id
func newID() uint32 {
	newID := transportID + 1
	transportID++
	return newID
}

var transportID = uint32(0)

// Stream piping ----------------------------------------------------------------------------

// transportConn - Original function taked from the gost project, with comments added
// and without an error group to wait for both sides to declare an error/EOF.
// This is used to transport stream-oriented traffic like TCP, because EOF matters here.
func transportConn(rw1, rw2 io.ReadWriter) error {
	errc := make(chan error, 1)

	// Source reads from
	go func() {
		errc <- copyBuffer(rw1, rw2)
	}()

	// Source writes to
	go func() {
		errc <- copyBuffer(rw2, rw1)
	}()

	// Any error arising from either the source
	// or the destination connections and we return.
	// Connections are not automatically closed
	// so a function is called after.
	err := <-errc
	if err != nil && err == io.EOF {
		err = nil
	}
	return err
}

// transportPacketConn - UDP streams cause streams to end with io.EOF error,
// while we don't care about them in this case. This works the same
// as transport(rw1, rw2), but ignores end of file.
// The channel is used to control when to stop piping.
func transportPacketConn(rw1, rw2 io.ReadWriter) error {
	errc := make(chan error, 1)

	// Source reads from
	go func() {
		for {
			readErr := copyBuffer(rw1, rw2)
			// A nil error is an EOF when we transport UDP,
			// and strangely the copyBuffer() returns nil when EOF.
			// For any nil error we then know the connection has been
			// closed, so we return, otherwise we keep copying.
			if readErr != nil {
				continue
			}
			readErr = io.EOF // Error is nil, so notify the conn is closed
			errc <- readErr
			return
		}
	}()

	// Source writes to
	go func() {
		for {
			errWrite := copyBuffer(rw2, rw1)
			// A nil error is an EOF when we transport UDP,
			// and strangely the copyBuffer() returns nil when EOF.
			// For any nil error we then know the connection has been
			// closed, so we return, otherwise we keep copying.
			if errWrite != nil {
				continue
			}
			errWrite = io.EOF // Error is nil, so notify the conn is closed
			errc <- errWrite
			return
		}
	}()

	// Any error arising from either the source
	// or the destination connections and we return.
	// Connections are not automatically closed
	// so a function is called after.
	err := <-errc
	if err != nil && err == io.EOF {
		err = nil
	}

	return err
}

// transport - Blocking function that pipes 2 connections (stream oriented io.ReadWriteClosers)
// and blocks until both ends have returned either io.EOF (conn close) or any other error.
// Not used currently, might be useless.
func transport(rw1, rw2 io.ReadWriter) error {
	wg := sync.WaitGroup{}

	// Source reads from
	wg.Add(1)
	go func() {
		// Wait for source to close or error out.
		readErr := copyBuffer(rw1, rw2)
		if readErr != nil && readErr == io.EOF {
			readErr = nil
		}
		wg.Done()
	}()

	// Source writes to
	wg.Add(1)
	go func() {
		// Wait for destination to close or error out.
		writeErr := copyBuffer(rw2, rw1)
		if writeErr != nil && writeErr == io.EOF {
			writeErr = nil
		}
		wg.Done()
	}()

	// Wait until both ends of the connection have called close
	wg.Wait()
	return nil
}

func copyBuffer(dst io.Writer, src io.Reader) error {
	buf := lPool.Get().([]byte)
	defer lPool.Put(buf)

	_, err := io.CopyBuffer(dst, src, buf)
	return err
}

var (
	sPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, smallBufferSize)
		},
	}
	mPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, mediumBufferSize)
		},
	}
	lPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, largeBufferSize)
		},
	}
)

var (
	tinyBufferSize   = 512
	smallBufferSize  = 2 * 1024  // 2KB small buffer
	mediumBufferSize = 8 * 1024  // 8KB medium buffer
	largeBufferSize  = 32 * 1024 // 32KB large buffer
)

// closeConnections - When a src (SSH channel) is done piping to/from a net.Conn, we close both.
// This function is used both by UDP and TCP channels being piped.
func closeConnections(src io.Closer, dst io.Closer) {

	// We always leave some time before closing the connections,
	// because some of the traffic might still be processed by
	// the SSH RPC tunnel, which can be a bit slow to process data.
	// time.Sleep(1 * time.Second)
	time.Sleep(200 * time.Millisecond)

	// Close connections
	if dst != nil {
		dst.Close()
	}
	if src != nil {
		src.Close()
	}
}

// Comm address printing (package-wide) ----------------------------------------------------------------

// SetCommString - Get a string for a host given a route. If route is nil simply returns host.
func SetCommString(session *core.Session) string {

	// The session object already has an address in its RemoteAddr field:
	// HTTP/DNS/mTLS handlers have populated it. Make a copy of this field.
	addr := session.RemoteAddress

	// Given the remote address, check all existing routes,
	// and if any contains the given address, use this route.
	host := "scheme://" + addr
	route, err := ResolveAddress(host)
	if err != nil || route == nil {
		return addr
	}

	return route.String() + " " + addr
}

// SetHandlerCommString - Set the address string of a handler, based on active routes.
func SetHandlerCommString(host string) string {

	// Given the remote address, check all existing routes,
	// and if any contains the given address, use this route.
	ip := net.ParseIP(strings.Split(host, ":")[0])
	if ip == nil {
		return "[server] " + host
	}

	route, err := ResolveIP(ip)
	if err != nil || route == nil {
		return "[server] " + host
	}

	return route.String() + " " + host
}

// Handler requests/responses ----------------------------------------------------------------

// remoteHandlerRequest - Send a protobuf request to gateway session and get response.
func remoteHandlerRequest(sess *core.Session, req, resp proto.Message) (err error) {
	reqData, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	data, err := sess.Request(sliverpb.MsgNumber(req), defaultNetTimeout, reqData)
	if err != nil {
		return err
	}
	err = proto.Unmarshal(data, resp)
	if err != nil {
		return err
	}
	return nil
}
