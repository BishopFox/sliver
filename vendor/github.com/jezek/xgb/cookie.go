package xgb

import (
	"errors"
	"io"
)

// Cookie is the internal representation of a cookie, where one is generated
// for *every* request sent by XGB.
// 'cookie' is most frequently used by embedding it into a more specific
// kind of cookie, i.e., 'GetInputFocusCookie'.
type Cookie struct {
	conn      *Conn
	Sequence  uint16
	replyChan chan []byte
	errorChan chan error
	pingChan  chan bool
}

// NewCookie creates a new cookie with the correct channels initialized
// depending upon the values of 'checked' and 'reply'. Together, there are
// four different kinds of cookies. (See more detailed comments in the
// function for more info on those.)
// Note that a sequence number is not set until just before the request
// corresponding to this cookie is sent over the wire.
//
// Unless you're building requests from bytes by hand, this method should
// not be used.
func (c *Conn) NewCookie(checked, reply bool) *Cookie {
	cookie := &Cookie{
		conn:      c,
		Sequence:  0, // we add the sequence id just before sending a request
		replyChan: nil,
		errorChan: nil,
		pingChan:  nil,
	}

	// There are four different kinds of cookies:
	// Checked requests with replies get a reply channel and an error channel.
	// Unchecked requests with replies get a reply channel and a ping channel.
	// Checked requests w/o replies get a ping channel and an error channel.
	// Unchecked requests w/o replies get no channels.
	// The reply channel is used to send reply data.
	// The error channel is used to send error data.
	// The ping channel is used when one of the 'reply' or 'error' channels
	// is missing but the other is present. The ping channel is way to force
	// the blocking to stop and basically say "the error has been received
	// in the main event loop" (when the ping channel is coupled with a reply
	// channel) or "the request you made that has no reply was successful"
	// (when the ping channel is coupled with an error channel).
	if checked {
		cookie.errorChan = make(chan error, 1)
		if !reply {
			cookie.pingChan = make(chan bool, 1)
		}
	}
	if reply {
		cookie.replyChan = make(chan []byte, 1)
		if !checked {
			cookie.pingChan = make(chan bool, 1)
		}
	}

	return cookie
}

// Reply detects whether this is a checked or unchecked cookie, and calls
// 'replyChecked' or 'replyUnchecked' appropriately.
//
// Unless you're building requests from bytes by hand, this method should
// not be used.
func (c Cookie) Reply() ([]byte, error) {
	// checked
	if c.errorChan != nil {
		return c.replyChecked()
	}
	return c.replyUnchecked()
}

// replyChecked waits for a response on either the replyChan or errorChan
// channels. If the former arrives, the bytes are returned with a nil error.
// If the latter arrives, no bytes are returned (nil) and the error received
// is returned.
// Returns (nil, io.EOF) when the connection is closed.
//
// Unless you're building requests from bytes by hand, this method should
// not be used.
func (c Cookie) replyChecked() ([]byte, error) {
	if c.replyChan == nil {
		return nil, errors.New("Cannot call 'replyChecked' on a cookie that " +
			"is not expecting a *reply* or an error.")
	}
	if c.errorChan == nil {
		return nil, errors.New("Cannot call 'replyChecked' on a cookie that " +
			"is not expecting a reply or an *error*.")
	}

	select {
	case reply := <-c.replyChan:
		return reply, nil
	case err := <-c.errorChan:
		return nil, err
	case <-c.conn.doneRead:
		// c.conn.readResponses is no more, there will be no replys or errors
		return nil, io.EOF
	}
}

// replyUnchecked waits for a response on either the replyChan or pingChan
// channels. If the former arrives, the bytes are returned with a nil error.
// If the latter arrives, no bytes are returned (nil) and a nil error
// is returned. (In the latter case, the corresponding error can be retrieved
// from (Wait|Poll)ForEvent asynchronously.)
// Returns (nil, io.EOF) when the connection is closed.
// In all honesty, you *probably* don't want to use this method.
//
// Unless you're building requests from bytes by hand, this method should
// not be used.
func (c Cookie) replyUnchecked() ([]byte, error) {
	if c.replyChan == nil {
		return nil, errors.New("Cannot call 'replyUnchecked' on a cookie " +
			"that is not expecting a *reply*.")
	}

	select {
	case reply := <-c.replyChan:
		return reply, nil
	case <-c.pingChan:
		return nil, nil
	case <-c.conn.doneRead:
		// c.conn.readResponses is no more, there will be no replys or pings
		return nil, io.EOF
	}
}

// Check is used for checked requests that have no replies. It is a mechanism
// by which to report "success" or "error" in a synchronous fashion. (Therefore,
// unchecked requests without replies cannot use this method.)
// If the request causes an error, it is sent to this cookie's errorChan.
// If the request was successful, there is no response from the server.
// Thus, pingChan is sent a value when the *next* reply is read.
// If no more replies are being processed, we force a round trip request with
// GetInputFocus.
// Returns io.EOF error when the connection is closed.
//
// Unless you're building requests from bytes by hand, this method should
// not be used.
func (c Cookie) Check() error {
	if c.replyChan != nil {
		return errors.New("Cannot call 'Check' on a cookie that is " +
			"expecting a *reply*. Use 'Reply' instead.")
	}
	if c.errorChan == nil {
		return errors.New("Cannot call 'Check' on a cookie that is " +
			"not expecting a possible *error*.")
	}

	// First do a quick non-blocking check to see if we've been pinged.
	select {
	case err := <-c.errorChan:
		return err
	case <-c.pingChan:
		return nil
	default:
	}

	// Now force a round trip and try again, but block this time.
	c.conn.Sync()
	select {
	case err := <-c.errorChan:
		return err
	case <-c.pingChan:
		return nil
	case <-c.conn.doneRead:
		// c.conn.readResponses is no more, there will be no errors or pings
		return io.EOF
	}
}
