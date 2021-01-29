package grpc_net_conn

import (
	"net"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
)

// Conn implements net.Conn across a gRPC stream. You must populate many
// of the exported structs on this field so please read the documentation.
//
// There are a number of limitations to this implementation, typically due
// limitations of visibility given by the gRPC stream. Methods such as
// LocalAddr, RemoteAddr, deadlines, etc. do not work.
//
// As documented on net.Conn, it is safe for concurrent read/write.
type Conn struct {
	// Stream is the stream to wrap into a Conn. This can be either a client
	// or server stream and we will perform correctly.
	Stream grpc.Stream

	// Request is the type to use for sending request data to the streaming
	// endpoint. This must be a non-nil allocated value and must NOT point to
	// the same value as Response since they may be used concurrently.
	//
	// The Reset method is never called on Request so you may set some
	// fields on the request type and they will be sent for every request
	// unless the Encode field changes it.
	Request proto.Message

	// Response is the type to use for reading response data. This must be
	// a non-nil allocated value and must NOT point to the same value as Request
	// since they may be used concurrently.
	//
	// The Reset method will be called on Response during Reads so data you
	// set initially will be lost.
	Response proto.Message

	// ResponseLock, if non-nil, will be locked while calling SendMsg
	// on the Stream. This can be used to prevent concurrent access to
	// SendMsg which is unsafe.
	ResponseLock *sync.Mutex

	// Encode encodes messages into the Request. See Encoder for more information.
	Encode Encoder

	// Decode decodes messages from the Response into a byte slice. See
	// Decoder for more information.
	Decode Decoder

	// readOffset tracks where we've read up to if we're reading a result
	// that didn't fully fit into the target slice. See Read.
	readOffset int

	// locks to ensure that only one reader/writer are operating at a time.
	// Go documents the `net.Conn` interface as being safe for simultaneous
	// readers/writers so we need to implement locking.
	readLock, writeLock sync.Mutex
}

// Read implements io.Reader.
func (c *Conn) Read(p []byte) (int, error) {
	c.readLock.Lock()
	defer c.readLock.Unlock()

	// Attempt to read a value only if we're not still decoding a
	// partial read value from the last result.
	if c.readOffset == 0 {
		if err := c.Stream.RecvMsg(c.Response); err != nil {
			return 0, err
		}
	}

	// Decode into our slice
	data, err := c.Decode(c.Response, c.readOffset, p)

	// If we have an error or we've decoded the full amount then we're done.
	// The error case is obvious. The case where we've read the full amount
	// is also a terminating condition and err == nil (we know this since its
	// checking that case) so we return the full amount and no error.
	if err != nil || len(data) <= (len(p)+c.readOffset) {
		// Reset the read offset since we're done.
		n := len(data) - c.readOffset
		c.readOffset = 0

		// Reset our response value for the next read and so that we
		// don't potentially store a large response structure in memory.
		c.Response.Reset()

		return n, err
	}

	// We didn't read the full amount so we need to store this for future reads
	c.readOffset += len(p)

	return len(p), nil
}

// Write implements io.Writer.
func (c *Conn) Write(p []byte) (int, error) {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	total := len(p)
	for {
		// Encode our data into the request. Any error means we abort.
		n, err := c.Encode(c.Request, p)
		if err != nil {
			return 0, err
		}

		// We lock for SendMsg if we have a lock set.
		if c.ResponseLock != nil {
			c.ResponseLock.Lock()
		}

		// Send our message. Any error we also just abort out.
		err = c.Stream.SendMsg(c.Request)
		if c.ResponseLock != nil {
			c.ResponseLock.Unlock()
		}
		if err != nil {
			return 0, err
		}

		// If we sent the full amount of data, we're done. We respond with
		// "total" in case we sent across multiple frames.
		if n == len(p) {
			return total, nil
		}

		// We sent partial data so we continue writing the remainder
		p = p[n:]
	}
}

// Close will close the client if this is a client. If this is a server
// stream this does nothing since gRPC expects you to close the stream by
// returning from the RPC call.
//
// This calls CloseSend underneath for clients, so read the documentation
// for that to understand the semantics of this call.
func (c *Conn) Close() error {
	if cs, ok := c.Stream.(grpc.ClientStream); ok {
		// We have to acquire the write lock since the gRPC docs state:
		// "It is also not safe to call CloseSend concurrently with SendMsg."
		c.writeLock.Lock()
		defer c.writeLock.Unlock()
		return cs.CloseSend()
	}

	return nil
}

// LocalAddr returns nil.
func (c *Conn) LocalAddr() net.Addr { return nil }

// RemoteAddr returns nil.
func (c *Conn) RemoteAddr() net.Addr { return nil }

// SetDeadline is non-functional due to limitations on how gRPC works.
// You can mimic deadlines often using call options.
func (c *Conn) SetDeadline(time.Time) error { return nil }

// SetReadDeadline is non-functional, see SetDeadline.
func (c *Conn) SetReadDeadline(time.Time) error { return nil }

// SetWriteDeadline is non-functional, see SetDeadline.
func (c *Conn) SetWriteDeadline(time.Time) error { return nil }

var _ net.Conn = (*Conn)(nil)
