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
	"context"
	"io"
	"sync"
	"time"
)

// Comms - (SSH-multiplexing) ------------------------------------------------------------

var (
	// ClientComm - The only Comm instance client-side, handles all port forwarders, proxies.
	ClientComm *Comm
)

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
func closeConnections(src io.Closer, dst io.Closer) {

	// We always leave some time before closing the connections,
	// because some of the traffic might still be processed by
	// the SSH RPC tunnel, which can be a bit slow to process data.
	time.Sleep(200 * time.Millisecond)
	// time.Sleep(1 * time.Second)

	// Close connections
	if dst != nil {
		dst.Close()
	}
	if src != nil {
		src.Close()
	}
}

func isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
