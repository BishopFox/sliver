package xgb

// Sync sends a round trip request and waits for the response.
// This forces all pending cookies to be dealt with.
// You actually shouldn't need to use this like you might with Xlib. Namely,
// buffers are automatically flushed using Go's channels and round trip requests
// are forced where appropriate automatically.
func (c *Conn) Sync() {
	cookie := c.NewCookie(true, true)
	c.NewRequest(c.getInputFocusRequest(), cookie)
	cookie.Reply() // wait for the buffer to clear
}

// getInputFocusRequest writes the raw bytes to a buffer.
// It is duplicated from xproto/xproto.go.
func (c *Conn) getInputFocusRequest() []byte {
	size := 4
	b := 0
	buf := make([]byte, size)

	buf[b] = 43 // request opcode
	b += 1

	b += 1                         // padding
	Put16(buf[b:], uint16(size/4)) // write request size in 4-byte units
	b += 2

	return buf
}
