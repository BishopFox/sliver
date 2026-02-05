package yamux

import (
	"io"
	"net"
	"sync"
)

const (
	defaultAcceptBacklog = 256
	defaultWriteBacklog  = 256
	defaultMaxFrameData  = 32 * 1024
)

// Session multiplexes multiple independent streams over a single net.Conn.
type Session struct {
	conn net.Conn

	isClient     bool
	nextStreamID uint32

	mu      sync.Mutex
	streams map[uint32]*Stream

	acceptCh chan *Stream
	writeCh  chan frame

	closed    chan struct{}
	closeOnce sync.Once
}

// Server starts a server-side session.
func Server(conn net.Conn, config *Config) (*Session, error) {
	return newSession(conn, false), nil
}

// Client starts a client-side session.
func Client(conn net.Conn, config *Config) (*Session, error) {
	return newSession(conn, true), nil
}

func newSession(conn net.Conn, isClient bool) *Session {
	s := &Session{
		conn:      conn,
		isClient:  isClient,
		streams:   make(map[uint32]*Stream),
		acceptCh:  make(chan *Stream, defaultAcceptBacklog),
		writeCh:   make(chan frame, defaultWriteBacklog),
		closed:    make(chan struct{}),
		closeOnce: sync.Once{},
	}
	if isClient {
		s.nextStreamID = 1
	} else {
		s.nextStreamID = 2
	}
	go s.readLoop()
	go s.writeLoop()
	return s
}

// Open creates a new outgoing stream.
func (s *Session) Open() (net.Conn, error) {
	s.mu.Lock()
	if s.isClosedLocked() {
		s.mu.Unlock()
		return nil, ErrSessionShutdown
	}
	sid := s.nextStreamID
	s.nextStreamID += 2
	st := newStream(s, sid)
	s.streams[sid] = st
	s.mu.Unlock()

	if err := s.send(frame{typ: frameTypeOpen, sid: sid}); err != nil {
		_ = st.Close()
		return nil, err
	}
	return st, nil
}

// Accept waits for an incoming stream opened by the remote peer.
func (s *Session) Accept() (net.Conn, error) {
	select {
	case st, ok := <-s.acceptCh:
		if !ok {
			return nil, ErrSessionShutdown
		}
		return st, nil
	case <-s.closed:
		return nil, ErrSessionShutdown
	}
}

// Close shuts down the session and the underlying connection.
func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		close(s.closed)
		_ = s.conn.Close()
		s.mu.Lock()
		for _, st := range s.streams {
			st.closeRemote()
		}
		s.streams = map[uint32]*Stream{}
		s.mu.Unlock()
		close(s.acceptCh)
	})
	return nil
}

func (s *Session) isClosedLocked() bool {
	select {
	case <-s.closed:
		return true
	default:
		return false
	}
}

func (s *Session) send(f frame) error {
	select {
	case s.writeCh <- f:
		return nil
	case <-s.closed:
		return ErrSessionShutdown
	}
}

func (s *Session) readLoop() {
	defer s.Close()
	for {
		f, err := readFrame(s.conn, defaultMaxFrameData)
		if err != nil {
			if err == io.EOF {
				return
			}
			return
		}

		switch f.typ {
		case frameTypeOpen:
			if len(f.buf) != 0 {
				return
			}
			st, err := s.getOrCreateInboundStream(f.sid)
			if err != nil {
				return
			}
			s.enqueueAccept(st)

		case frameTypeData:
			st, err := s.getOrCreateInboundStream(f.sid)
			if err != nil {
				return
			}
			st.enqueueData(f.buf)

		case frameTypeClose:
			s.mu.Lock()
			st := s.streams[f.sid]
			s.mu.Unlock()
			if st != nil {
				st.closeRemote()
			}

		default:
			return
		}
	}
}

func (s *Session) getOrCreateInboundStream(sid uint32) (*Stream, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isClosedLocked() {
		return nil, ErrSessionShutdown
	}
	if st, ok := s.streams[sid]; ok {
		return st, nil
	}

	// Enforce odd/even stream ownership.
	if s.isClient {
		// Client should only accept even stream IDs from the server.
		if sid%2 != 0 {
			return nil, errProtocolViolation
		}
	} else {
		// Server should only accept odd stream IDs from the client.
		if sid%2 == 0 {
			return nil, errProtocolViolation
		}
	}

	st := newStream(s, sid)
	s.streams[sid] = st
	return st, nil
}

func (s *Session) enqueueAccept(st *Stream) {
	select {
	case s.acceptCh <- st:
	case <-s.closed:
	}
}

func (s *Session) writeLoop() {
	defer s.Close()
	for {
		select {
		case f := <-s.writeCh:
			if err := writeFrame(s.conn, f); err != nil {
				return
			}
		case <-s.closed:
			return
		}
	}
}

func (s *Session) removeStream(sid uint32) {
	s.mu.Lock()
	delete(s.streams, sid)
	s.mu.Unlock()
}
