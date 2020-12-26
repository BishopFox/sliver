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
	"strings"
	"sync"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/golang/protobuf/proto"
)

// Utils - (package-wide) ----------------------------------------------------------------

// SetCommString - Get a string for a host given a route. If route is nil simply returns host.
func SetCommString(session *core.Session) string {

	// The session object already has an address in its RemoteAddr field:
	// HTTP/DNS/mTLS handlers have populated it. Make a copy of this field.
	addr := session.RemoteAddress

	// Given the remote address, check all existing routes,
	// and if any contains the given address, use this route.
	host := strings.Split(addr, ":")[0]
	route, err := ResolveAddress(host)
	if err != nil || route == nil {
		return addr
	}

	return route.String() + " " + addr
}

// Comms - (SSH-multiplexing) ------------------------------------------------------------

var (
	// Comms - All multiplexers currently running in Sliver, providing connection routing.
	Comms = &comms{
		Active: map[uint32]*Comm{},
		mutex:  &sync.RWMutex{},
	}
)

type comms struct {
	Active map[uint32]*Comm
	mutex  *sync.RWMutex
}

// Get - Get a session by ID
func (c *comms) Get(commID uint32) *Comm {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Active[commID]
}

// Add - Add a sliver to the hive (atomically)
func (c *comms) Add(mux *Comm) *Comm {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Active[mux.ID] = mux
	return mux
}

// Remove - Remove a sliver from the hive (atomically)
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

func transport(rw1, rw2 io.ReadWriter) error {
	errc := make(chan error, 1)
	go func() {
		errc <- copyBuffer(rw1, rw2)
	}()

	go func() {
		errc <- copyBuffer(rw2, rw1)
	}()

	err := <-errc
	if err != nil && err == io.EOF {
		err = nil
	}
	return err
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
