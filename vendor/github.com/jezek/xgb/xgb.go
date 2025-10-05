package xgb

import (
	"errors"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

var (
	// Where to log error-messages. Defaults to stderr.
	// To disable logging, just set this to log.New(ioutil.Discard, "", 0)
	Logger = log.New(os.Stderr, "XGB: ", log.Lshortfile)
)

const (
	// cookieBuffer represents the queue size of cookies existing at any
	// point in time. The size of the buffer is really only important when
	// there are many requests without replies made in sequence. Once the
	// buffer fills, a round trip request is made to clear the buffer.
	cookieBuffer = 1000

	// xidBuffer represents the queue size of the xid channel.
	// I don't think this value matters much, since xid generation is not
	// that expensive.
	xidBuffer = 5

	// seqBuffer represents the queue size of the sequence number channel.
	// I don't think this value matters much, since sequence number generation
	// is not that expensive.
	seqBuffer = 5

	// reqBuffer represents the queue size of the number of requests that
	// can be made until new ones block. This value seems OK.
	reqBuffer = 100

	// eventBuffer represents the queue size of the number of events or errors
	// that can be loaded off the wire and not grabbed with WaitForEvent
	// until reading an event blocks. This value should be big enough to handle
	// bursts of events.
	eventBuffer = 5000
)

// A Conn represents a connection to an X server.
type Conn struct {
	host          string
	conn          net.Conn
	display       string
	DisplayNumber int
	DefaultScreen int
	SetupBytes    []byte

	setupResourceIdBase uint32
	setupResourceIdMask uint32

	eventChan  chan eventOrError
	cookieChan chan *Cookie
	xidChan    chan xid
	seqChan    chan uint16
	reqChan    chan *request
	doneSend   chan struct{}
	doneRead   chan struct{}

	// ExtLock is a lock used whenever new extensions are initialized.
	// It should not be used. It is exported for use in the extension
	// sub-packages.
	ExtLock sync.RWMutex

	// Extensions is a map from extension name to major opcode. It should
	// not be used. It is exported for use in the extension sub-packages.
	Extensions map[string]byte
}

// NewConn creates a new connection instance. It initializes locks, data
// structures, and performs the initial handshake. (The code for the handshake
// has been relegated to conn.go.)
// It is up to user to close connection with Close() method to finish all unfinished requests and clean up spawned goroutines.
// If the connection unexpectedly closes itself and WaitForEvent() returns "nil, nil", everything is cleaned by that moment, but nothing bad happens if you call Close() after.
func NewConn() (*Conn, error) {
	return NewConnDisplay("")
}

// NewConnDisplay is just like NewConn (see closing instructions), but allows a specific DISPLAY
// string to be used.
// If 'display' is empty it will be taken from os.Getenv("DISPLAY").
//
// Examples:
//	NewConn(":1") -> net.Dial("unix", "", "/tmp/.X11-unix/X1")
//	NewConn("/tmp/launch-12/:0") -> net.Dial("unix", "", "/tmp/launch-12/:0")
//	NewConn("hostname:2.1") -> net.Dial("tcp", "", "hostname:6002")
//	NewConn("tcp/hostname:1.0") -> net.Dial("tcp", "", "hostname:6001")
func NewConnDisplay(display string) (*Conn, error) {
	c := &Conn{}

	// First connect. This reads authority, checks DISPLAY environment
	// variable, and loads the initial Setup info.
	err := c.connect(display)
	if err != nil {
		return nil, err
	}

	return postNewConn(c)
}

// NewConnNet is just like NewConn (see closing instructions), but allows a specific net.Conn
// to be used.
func NewConnNet(netConn net.Conn) (*Conn, error) {
	c := &Conn{}

	// First connect. This reads authority, checks DISPLAY environment
	// variable, and loads the initial Setup info.
	err := c.connectNet(netConn)

	if err != nil {
		return nil, err
	}

	return postNewConn(c)
}

func postNewConn(c *Conn) (*Conn, error) {
	c.Extensions = make(map[string]byte)

	c.cookieChan = make(chan *Cookie, cookieBuffer)
	c.xidChan = make(chan xid, xidBuffer)
	c.seqChan = make(chan uint16, seqBuffer)
	c.reqChan = make(chan *request, reqBuffer)
	c.eventChan = make(chan eventOrError, eventBuffer)
	c.doneSend = make(chan struct{})
	c.doneRead = make(chan struct{})

	go c.generateXIds()
	go c.generateSeqIds()
	go c.sendRequests()
	go c.readResponses()

	return c, nil
}

// Close gracefully closes the connection to the X server.
// When everything is cleaned up, the WaitForEvent method will return (nil, nil)
func (c *Conn) Close() {
	select {
	case c.reqChan <- nil:
	case <-c.doneSend:
	}
}

// Event is an interface that can contain any of the events returned by the
// server. Use a type assertion switch to extract the Event structs.
type Event interface {
	Bytes() []byte
	String() string
}

// NewEventFun is the type of function use to construct events from raw bytes.
// It should not be used. It is exported for use in the extension sub-packages.
type NewEventFun func(buf []byte) Event

// NewEventFuncs is a map from event numbers to functions that create
// the corresponding event. It should not be used. It is exported for use
// in the extension sub-packages.
var NewEventFuncs = make(map[int]NewEventFun)

// NewExtEventFuncs is a temporary map that stores event constructor functions
// for each extension. When an extension is initialized, each event for that
// extension is added to the 'NewEventFuncs' map. It should not be used. It is
// exported for use in the extension sub-packages.
var NewExtEventFuncs = make(map[string]map[int]NewEventFun)

// Error is an interface that can contain any of the errors returned by
// the server. Use a type assertion switch to extract the Error structs.
type Error interface {
	SequenceId() uint16
	BadId() uint32
	Error() string
}

// NewErrorFun is the type of function use to construct errors from raw bytes.
// It should not be used. It is exported for use in the extension sub-packages.
type NewErrorFun func(buf []byte) Error

// NewErrorFuncs is a map from error numbers to functions that create
// the corresponding error. It should not be used. It is exported for use in
// the extension sub-packages.
var NewErrorFuncs = make(map[int]NewErrorFun)

// NewExtErrorFuncs is a temporary map that stores error constructor functions
// for each extension. When an extension is initialized, each error for that
// extension is added to the 'NewErrorFuncs' map. It should not be used. It is
// exported for use in the extension sub-packages.
var NewExtErrorFuncs = make(map[string]map[int]NewErrorFun)

// eventOrError corresponds to values that can be either an event or an
// error.
type eventOrError interface{}

// NewId generates a new unused ID for use with requests like CreateWindow.
// If no new ids can be generated, the id returned is 0 and error is non-nil.
// This shouldn't be used directly, and is exported for use in the extension
// sub-packages.
// If you need identifiers, use the appropriate constructor.
// e.g., For a window id, use xproto.NewWindowId. For
// a new pixmap id, use xproto.NewPixmapId. And so on.
// Returns (0, io.EOF) when the connection is closed.
func (c *Conn) NewId() (uint32, error) {
	xid, ok := <-c.xidChan
	if !ok {
		return 0, io.EOF
	}
	if xid.err != nil {
		return 0, xid.err
	}
	return xid.id, nil
}

// xid encapsulates a resource identifier being sent over the Conn.xidChan
// channel. If no new resource id can be generated, id is set to 0 and a
// non-nil error is set in xid.err.
type xid struct {
	id  uint32
	err error
}

// generateXids sends new Ids down the channel for NewId to use.
// generateXids should be run in its own goroutine.
// This needs to be updated to use the XC Misc extension once we run out of
// new ids.
// Thanks to libxcb/src/xcb_xid.c. This code is greatly inspired by it.
func (c *Conn) generateXIds() {
	defer close(c.xidChan)

	// This requires some explanation. From the horse's mouth:
	// "The resource-id-mask contains a single contiguous set of bits (at least
	// 18).  The client allocates resource IDs for types WINDOW, PIXMAP,
	// CURSOR, FONT, GCONTEXT, and COLORMAP by choosing a value with only some
	// subset of these bits set and ORing it with resource-id-base. Only values
	// constructed in this way can be used to name newly created resources over
	// this connection."
	// So for example (using 8 bit integers), the mask might look like:
	// 00111000
	// So that valid values would be 00101000, 00110000, 00001000, and so on.
	// Thus, the idea is to increment it by the place of the last least
	// significant '1'. In this case, that value would be 00001000. To get
	// that value, we can AND the original mask with its two's complement:
	// 00111000 & 11001000 = 00001000.
	// And we use that value to increment the last resource id to get a new one.
	// (And then, of course, we OR it with resource-id-base.)
	inc := c.setupResourceIdMask & -c.setupResourceIdMask
	max := c.setupResourceIdMask
	last := uint32(0)
	for {
		id := xid{}
		if last > 0 && last >= max-inc+1 {
			// TODO: Use the XC Misc extension to look for released ids.
			id = xid{
				id:  0,
				err: errors.New("There are no more available resource identifiers."),
			}
		} else {
			last += inc
			id = xid{
				id:  last | c.setupResourceIdBase,
				err: nil,
			}
		}

		select {
		case c.xidChan <- id:
		case <-c.doneSend:
			// c.sendRequests is down and since this id is used by requests, we don't need this goroutine running anymore.
			return
		}
	}
}

// newSeqId fetches the next sequence id from the Conn.seqChan channel.
func (c *Conn) newSequenceId() uint16 {
	return <-c.seqChan
}

// generateSeqIds returns new sequence ids. It is meant to be run in its
// own goroutine.
// A sequence id is generated for *every* request. It's the identifier used
// to match up replies with requests.
// Since sequence ids can only be 16 bit integers we start over at zero when it
// comes time to wrap.
// N.B. As long as the cookie buffer is less than 2^16, there are no limitations
// on the number (or kind) of requests made in sequence.
func (c *Conn) generateSeqIds() {
	defer close(c.seqChan)

	seqid := uint16(1)
	for {
		select {
		case c.seqChan <- seqid:
			if seqid == uint16((1<<16)-1) {
				seqid = 0
			} else {
				seqid++
			}
		case <-c.doneSend:
			// c.sendRequests is down and since only that function uses sequence ids (via newSequenceId method), we don't need this goroutine running anymore.
			return
		}
	}
}

// request encapsulates a buffer of raw bytes (containing the request data)
// and a cookie, which when combined represents a single request.
// The cookie is used to match up the reply/error.
type request struct {
	buf    []byte
	cookie *Cookie

	// seq is closed when the request (cookie) has been sequenced by the Conn.
	seq chan struct{}
}

// NewRequest takes the bytes and a cookie of a particular request, constructs
// a request type, and sends it over the Conn.reqChan channel.
// Note that the sequence number is added to the cookie after it is sent
// over the request channel, but before it is sent to X.
//
// Note that you may safely use NewRequest to send arbitrary byte requests
// to X. The resulting cookie can be used just like any normal cookie and
// abides by the same rules, except that for replies, you'll get back the
// raw byte data. This may be useful for performance critical sections where
// every allocation counts, since all X requests in XGB allocate a new byte
// slice. In contrast, NewRequest allocates one small request struct and
// nothing else. (Except when the cookie buffer is full and has to be flushed.)
//
// If you're using NewRequest manually, you'll need to use NewCookie to create
// a new cookie.
//
// In all likelihood, you should be able to copy and paste with some minor
// edits the generated code for the request you want to issue.
func (c *Conn) NewRequest(buf []byte, cookie *Cookie) {
	seq := make(chan struct{})
	select {
	case c.reqChan <- &request{buf: buf, cookie: cookie, seq: seq}:
		// request is in buffer
		// wait until request is processed or connection is closed
		select {
		case <-seq:
			// request was successfully sent to X server
		case <-c.doneSend:
			// c.sendRequests is down, your request was not handled
		}
	case <-c.doneSend:
		// c.sendRequests is down, nobody is listening to your requests
	}
}

// sendRequests is run as a single goroutine that takes requests and writes
// the bytes to the wire and adds the cookie to the cookie queue.
// It is meant to be run as its own goroutine.
func (c *Conn) sendRequests() {
	defer close(c.cookieChan)
	defer c.conn.Close()
	defer close(c.doneSend)

	for {
		select {
		case req := <-c.reqChan:
			if req == nil {
				// a request by c.Close() to gracefully exit
				// Flush the response reading goroutine.
				if err := c.noop(); err != nil {
					c.conn.Close()
					<-c.doneRead
				}
				return
			}
			// ho there! if the cookie channel is nearly full, force a round
			// trip to clear out the cookie buffer.
			// Note that we circumvent the request channel, because we're *in*
			// the request channel.
			if len(c.cookieChan) == cookieBuffer-1 {
				if err := c.noop(); err != nil {
					// Shut everything down.
					c.conn.Close()
					<-c.doneRead
					return
				}
			}
			req.cookie.Sequence = c.newSequenceId()
			c.cookieChan <- req.cookie
			if err := c.writeBuffer(req.buf); err != nil {
				c.conn.Close()
				<-c.doneRead
				return
			}
			close(req.seq)
		case <-c.doneRead:
			return
		}
	}
}

// noop circumvents the usual request sending goroutines and forces a round
// trip request manually.
func (c *Conn) noop() error {
	cookie := c.NewCookie(true, true)
	cookie.Sequence = c.newSequenceId()
	c.cookieChan <- cookie
	if err := c.writeBuffer(c.getInputFocusRequest()); err != nil {
		return err
	}
	cookie.Reply() // wait for the buffer to clear
	return nil
}

// writeBuffer is a convenience function for writing a byte slice to the wire.
func (c *Conn) writeBuffer(buf []byte) error {
	if _, err := c.conn.Write(buf); err != nil {
		Logger.Printf("A write error is unrecoverable: %s", err)
		return err
	}
	return nil
}

// readResponses is a goroutine that reads events, errors and
// replies off the wire.
// When an event is read, it is always added to the event channel.
// When an error is read, if it corresponds to an existing checked cookie,
// it is sent to that cookie's error channel. Otherwise it is added to the
// event channel.
// When a reply is read, it is added to the corresponding cookie's reply
// channel. (It is an error if no such cookie exists in this case.)
// Finally, cookies that came "before" this reply are always cleaned up.
func (c *Conn) readResponses() {
	defer close(c.eventChan)
	defer c.conn.Close()
	defer close(c.doneRead)

	var (
		err        Error
		seq        uint16
		replyBytes []byte
	)

	for {
		buf := make([]byte, 32)
		err, seq = nil, 0
		if _, err := io.ReadFull(c.conn, buf); err != nil {
			select {
			case <-c.doneSend:
				// gracefully closing
				return
			default:
			}
			Logger.Printf("A read error is unrecoverable: %s", err)
			c.eventChan <- err
			return
		}
		switch buf[0] {
		case 0: // This is an error
			// Use the constructor function for this error (that is auto
			// generated) by looking it up by the error number.
			newErrFun, ok := NewErrorFuncs[int(buf[1])]
			if !ok {
				Logger.Printf("BUG: Could not find error constructor function "+
					"for error with number %d.", buf[1])
				continue
			}
			err = newErrFun(buf)
			seq = err.SequenceId()

			// This error is either sent to the event channel or a specific
			// cookie's error channel below.
		case 1: // This is a reply
			seq = Get16(buf[2:])

			// check to see if this reply has more bytes to be read
			size := Get32(buf[4:])
			if size > 0 {
				byteCount := 32 + size*4
				biggerBuf := make([]byte, byteCount)
				copy(biggerBuf[:32], buf)
				if _, err := io.ReadFull(c.conn, biggerBuf[32:]); err != nil {
					Logger.Printf("A read error is unrecoverable: %s", err)
					c.eventChan <- err
					return
				}
				replyBytes = biggerBuf
			} else {
				replyBytes = buf
			}

			// This reply is sent to its corresponding cookie below.
		default: // This is an event
			// Use the constructor function for this event (like for errors,
			// and is also auto generated) by looking it up by the event number.
			// Note that we AND the event number with 127 so that we ignore
			// the most significant bit (which is set when it was sent from
			// a SendEvent request).
			evNum := int(buf[0] & 127)
			newEventFun, ok := NewEventFuncs[evNum]
			if !ok {
				Logger.Printf("BUG: Could not find event construct function "+
					"for event with number %d.", evNum)
				continue
			}
			c.eventChan <- newEventFun(buf)
			continue
		}

		// At this point, we have a sequence number and we're either
		// processing an error or a reply, which are both responses to
		// requests. So all we have to do is find the cookie corresponding
		// to this error/reply, and send the appropriate data to it.
		// In doing so, we make sure that any cookies that came before it
		// are marked as successful if they are void and checked.
		// If there's a cookie that requires a reply that is before this
		// reply, then something is wrong.
		for cookie := range c.cookieChan {
			// This is the cookie we're looking for. Process and break.
			if cookie.Sequence == seq {
				if err != nil { // this is an error to a request
					// synchronous processing
					if cookie.errorChan != nil {
						cookie.errorChan <- err
					} else { // asynchronous processing
						c.eventChan <- err
						// if this is an unchecked reply, ping the cookie too
						if cookie.pingChan != nil {
							cookie.pingChan <- true
						}
					}
				} else { // this is a reply
					if cookie.replyChan == nil {
						Logger.Printf("Reply with sequence id %d does not "+
							"have a cookie with a valid reply channel.", seq)
						continue
					} else {
						cookie.replyChan <- replyBytes
					}
				}
				break
			}

			switch {
			// Checked requests with replies
			case cookie.replyChan != nil && cookie.errorChan != nil:
				Logger.Printf("Found cookie with sequence id %d that is "+
					"expecting a reply but will never get it. Currently "+
					"on sequence number %d", cookie.Sequence, seq)
			// Unchecked requests with replies
			case cookie.replyChan != nil && cookie.pingChan != nil:
				Logger.Printf("Found cookie with sequence id %d that is "+
					"expecting a reply (and not an error) but will never "+
					"get it. Currently on sequence number %d",
					cookie.Sequence, seq)
			// Checked requests without replies
			case cookie.pingChan != nil && cookie.errorChan != nil:
				cookie.pingChan <- true
				// Unchecked requests without replies don't have any channels,
				// so we can't do anything with them except let them pass by.
			}
		}
	}
}

// processEventOrError takes an eventOrError, type switches on it,
// and returns it in Go idiomatic style.
func processEventOrError(everr eventOrError) (Event, Error) {
	switch ee := everr.(type) {
	case Event:
		return ee, nil
	case Error:
		return nil, ee
	case error:
		// c.conn read error
	case nil:
		// c.eventChan is closed
	default:
		Logger.Printf("Invalid event/error type: %T", everr)
	}
	return nil, nil
}

// WaitForEvent returns the next event from the server.
// It will block until an event is available.
// WaitForEvent returns either an Event or an Error. (Returning both
// is a bug.) Note than an Error here is an X error and not an XGB error. That
// is, X errors are sometimes completely expected (and you may want to ignore
// them in some cases).
//
// If both the event and error are nil, then the connection has been closed.
func (c *Conn) WaitForEvent() (Event, Error) {
	return processEventOrError(<-c.eventChan)
}

// PollForEvent returns the next event from the server if one is available in
// the internal queue without blocking. Note that unlike WaitForEvent, both
// Event and Error could be nil. Indeed, they are both nil when the event queue
// is empty.
func (c *Conn) PollForEvent() (Event, Error) {
	select {
	case everr := <-c.eventChan:
		return processEventOrError(everr)
	default:
		return nil, nil
	}
}
