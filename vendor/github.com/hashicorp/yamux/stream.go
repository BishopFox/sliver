package yamux

import (
	"bytes"
	"io"
	"net"
	"sync"
	"time"
)

// Stream is a bidirectional byte stream multiplexed over a Session.
type Stream struct {
	sess *Session
	sid  uint32

	mu sync.Mutex
	cv *sync.Cond

	buf bytes.Buffer

	remoteClosed bool
	localClosed  bool

	closeOnce sync.Once
}

func newStream(sess *Session, sid uint32) *Stream {
	st := &Stream{
		sess: sess,
		sid:  sid,
	}
	st.cv = sync.NewCond(&st.mu)
	return st
}

func (s *Stream) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for {
		if 0 < s.buf.Len() {
			return s.buf.Read(p)
		}
		if s.remoteClosed {
			return 0, io.EOF
		}
		s.cv.Wait()
	}
}

func (s *Stream) Write(p []byte) (int, error) {
	s.mu.Lock()
	if s.localClosed {
		s.mu.Unlock()
		return 0, io.ErrClosedPipe
	}
	s.mu.Unlock()

	written := 0
	for 0 < len(p) {
		chunkLen := len(p)
		if chunkLen > defaultMaxFrameData {
			chunkLen = defaultMaxFrameData
		}
		if err := s.sess.send(frame{typ: frameTypeData, sid: s.sid, buf: append([]byte(nil), p[:chunkLen]...)}); err != nil {
			return written, err
		}
		written += chunkLen
		p = p[chunkLen:]
	}
	return written, nil
}

func (s *Stream) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.localClosed = true
		s.mu.Unlock()

		_ = s.sess.send(frame{typ: frameTypeClose, sid: s.sid})
		s.sess.removeStream(s.sid)

		s.mu.Lock()
		s.cv.Broadcast()
		s.mu.Unlock()
	})
	return nil
}

func (s *Stream) enqueueData(p []byte) {
	if len(p) == 0 {
		return
	}
	s.mu.Lock()
	s.buf.Write(p)
	s.cv.Signal()
	s.mu.Unlock()
}

func (s *Stream) closeRemote() {
	s.mu.Lock()
	s.remoteClosed = true
	s.cv.Broadcast()
	s.mu.Unlock()
}

func (s *Stream) LocalAddr() net.Addr  { return s.sess.conn.LocalAddr() }
func (s *Stream) RemoteAddr() net.Addr { return s.sess.conn.RemoteAddr() }

func (s *Stream) SetDeadline(t time.Time) error {
	_ = t
	return nil
}

func (s *Stream) SetReadDeadline(t time.Time) error {
	_ = t
	return nil
}

func (s *Stream) SetWriteDeadline(t time.Time) error {
	_ = t
	return nil
}
