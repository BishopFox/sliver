package xgb

import (
	"bytes"
	"errors"
	"io"
	"net"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Leaks monitor

type goroutine struct {
	id    int
	name  string
	stack []byte
}

type leaks struct {
	name       string
	goroutines map[int]goroutine
	report     []*leaks
}

func leaksMonitor(name string, monitors ...*leaks) *leaks {
	return &leaks{
		name,
		leaks{}.collectGoroutines(),
		monitors,
	}
}

// ispired by https://golang.org/src/runtime/debug/stack.go?s=587:606#L21
// stack returns a formatted stack trace of all goroutines.
// It calls runtime.Stack with a large enough buffer to capture the entire trace.
func (_ leaks) stack() []byte {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return buf[:n]
		}
		buf = make([]byte, 2*len(buf))
	}
}

func (l leaks) collectGoroutines() map[int]goroutine {
	res := make(map[int]goroutine)
	stacks := bytes.Split(l.stack(), []byte{'\n', '\n'})

	regexpId := regexp.MustCompile(`^\s*goroutine\s*(\d+)`)
	for _, st := range stacks {
		lines := bytes.Split(st, []byte{'\n'})
		if len(lines) < 2 {
			panic("routine stach has less tnan two lines: " + string(st))
		}

		idMatches := regexpId.FindSubmatch(lines[0])
		if len(idMatches) < 2 {
			panic("no id found in goroutine stack's first line: " + string(lines[0]))
		}
		id, err := strconv.Atoi(string(idMatches[1]))
		if err != nil {
			panic("converting goroutine id to number error: " + err.Error())
		}
		if _, ok := res[id]; ok {
			panic("2 goroutines with same id: " + strconv.Itoa(id))
		}
		name := strings.TrimSpace(string(lines[1]))

		//filter out our stack routine
		if strings.Contains(name, "xgb.leaks.stack") {
			continue
		}

		res[id] = goroutine{id, name, st}
	}
	return res
}

func (l leaks) leakingGoroutines() []goroutine {
	goroutines := l.collectGoroutines()
	res := []goroutine{}
	for id, gr := range goroutines {
		if _, ok := l.goroutines[id]; ok {
			continue
		}
		res = append(res, gr)
	}
	return res
}
func (l leaks) checkTesting(t *testing.T) {
	if len(l.leakingGoroutines()) == 0 {
		return
	}
	leakTimeout := 10 * time.Millisecond
	time.Sleep(leakTimeout)
	//t.Logf("possible goroutine leakage, waiting %v", leakTimeout)
	grs := l.leakingGoroutines()
	for _, gr := range grs {
		t.Errorf("%s: %s is leaking", l.name, gr.name)
		//t.Errorf("%s: %s is leaking\n%v", l.name, gr.name, string(gr.stack))
	}
	for _, rl := range l.report {
		rl.ignoreLeak(grs...)
	}
}
func (l *leaks) ignoreLeak(grs ...goroutine) {
	for _, gr := range grs {
		l.goroutines[gr.id] = gr
	}
}

// dummy net.Conn

type dAddr struct {
	s string
}

func (_ dAddr) Network() string { return "dummy" }
func (a dAddr) String() string  { return a.s }

var (
	dNCErrNotImplemented = errors.New("command not implemented")
	dNCErrClosed         = errors.New("server closed")
	dNCErrWrite          = errors.New("server write failed")
	dNCErrRead           = errors.New("server read failed")
	dNCErrResponse       = errors.New("server response error")
)

type dNCIoResult struct {
	n   int
	err error
}
type dNCIo struct {
	b      []byte
	result chan dNCIoResult
}

type dNCCWriteLock struct{}
type dNCCWriteUnlock struct{}
type dNCCWriteError struct{}
type dNCCWriteSuccess struct{}
type dNCCReadLock struct{}
type dNCCReadUnlock struct{}
type dNCCReadError struct{}
type dNCCReadSuccess struct{}

// dummy net.Conn interface. Needs to be constructed via newDummyNetConn([...]) function.
type dNC struct {
	reply   func([]byte) []byte
	addr    dAddr
	in, out chan dNCIo
	control chan interface{}
	done    chan struct{}
}

// Results running dummy server, satisfying net.Conn interface for test purposes.
// 'name' parameter will be returned via (*dNC).Local/RemoteAddr().String()
// 'reply' parameter function will be runned only on successful (*dNC).Write(b) with 'b' as parameter to 'reply'. The result will be stored in internal buffer and can be retrieved later via (*dNC).Read([...]) method.
// It is users responsibility to stop and clean up resources with (*dNC).Close, if not needed anymore.
// By default, the (*dNC).Write([...]) and (*dNC).Read([...]) methods are unlocked and will not result in error.
//TODO make (*dNC).SetDeadline, (*dNC).SetReadDeadline, (*dNC).SetWriteDeadline work proprely.
func newDummyNetConn(name string, reply func([]byte) []byte) *dNC {

	s := &dNC{
		reply,
		dAddr{name},
		make(chan dNCIo), make(chan dNCIo),
		make(chan interface{}),
		make(chan struct{}),
	}

	in, out := s.in, chan dNCIo(nil)
	buf := &bytes.Buffer{}
	errorRead, errorWrite := false, false
	lockRead := false

	go func() {
		defer close(s.done)
		for {
			select {
			case dxsio := <-in:
				if errorWrite {
					dxsio.result <- dNCIoResult{0, dNCErrWrite}
					break
				}

				response := s.reply(dxsio.b)

				buf.Write(response)
				dxsio.result <- dNCIoResult{len(dxsio.b), nil}

				if !lockRead && buf.Len() > 0 && out == nil {
					out = s.out
				}
			case dxsio := <-out:
				if errorRead {
					dxsio.result <- dNCIoResult{0, dNCErrRead}
					break
				}

				n, err := buf.Read(dxsio.b)
				dxsio.result <- dNCIoResult{n, err}

				if buf.Len() == 0 {
					out = nil
				}
			case ci := <-s.control:
				if ci == nil {
					return
				}
				switch ci.(type) {
				case dNCCWriteLock:
					in = nil
				case dNCCWriteUnlock:
					in = s.in
				case dNCCWriteError:
					errorWrite = true
				case dNCCWriteSuccess:
					errorWrite = false
				case dNCCReadLock:
					out = nil
					lockRead = true
				case dNCCReadUnlock:
					lockRead = false
					if buf.Len() > 0 && out == nil {
						out = s.out
					}
				case dNCCReadError:
					errorRead = true
				case dNCCReadSuccess:
					errorRead = false
				default:
				}
			}
		}
	}()
	return s
}

// Shuts down dummy net.Conn server. Every blocking or future method calls will do nothing and result in error.
// Result will be dNCErrClosed if server was allready closed.
// Server can not be unclosed.
func (s *dNC) Close() error {
	select {
	case s.control <- nil:
		<-s.done
		return nil
	case <-s.done:
	}
	return dNCErrClosed
}

// Performs a write action to server.
// If not locked by (*dNC).WriteLock, it results in error or success. If locked, this method will block until unlocked, or closed.
//
// This method can be set to result in error or success, via (*dNC).WriteError() or (*dNC).WriteSuccess() methods.
//
// If setted to result in error, the 'reply' function will NOT be called and internal buffer will NOT increasethe.
// Result will be (0, dNCErrWrite).
//
// If setted to result in success, the 'reply' function will be called and its result will be writen to internal buffer.
// If there is something in the internal buffer, the (*dNC).Read([...]) will be unblocked (if not previously locked with (*dNC).ReadLock).
// Result will be (len(b), nil)
//
// If server was closed previously, result will be (0, dNCErrClosed).
func (s *dNC) Write(b []byte) (int, error) {
	resChan := make(chan dNCIoResult)
	select {
	case s.in <- dNCIo{b, resChan}:
		res := <-resChan
		return res.n, res.err
	case <-s.done:
	}
	return 0, dNCErrClosed
}

// Performs a read action from server.
// If locked by (*dNC).ReadLock(), this method will block until unlocked with (*dNC).ReadUnlock(), or server closes.
//
// If not locked, this method can be setted to result imidiatly in error, will block if internal buffer is empty or will perform an read operation from internal buffer.
//
// If setted to result in error via (*dNC).ReadError(), the result will be (0, dNCErrWrite).
//
// If not locked and not setted to result in error via (*dNC).ReadSuccess(), this method will block until internall buffer is not empty, than it returns the result of the buffer read operation via (*bytes.Buffer).Read([...]).
// If the internal buffer is empty after this method, all follwing (*dNC).Read([...]), requests will block until internall buffer is filled after successful write requests.
//
// If server was closed previously, result will be (0, io.EOF).
func (s *dNC) Read(b []byte) (int, error) {
	resChan := make(chan dNCIoResult)
	select {
	case s.out <- dNCIo{b, resChan}:
		res := <-resChan
		return res.n, res.err
	case <-s.done:
	}
	return 0, io.EOF
}
func (s *dNC) LocalAddr() net.Addr                { return s.addr }
func (s *dNC) RemoteAddr() net.Addr               { return s.addr }
func (s *dNC) SetDeadline(t time.Time) error      { return dNCErrNotImplemented }
func (s *dNC) SetReadDeadline(t time.Time) error  { return dNCErrNotImplemented }
func (s *dNC) SetWriteDeadline(t time.Time) error { return dNCErrNotImplemented }

func (s *dNC) Control(i interface{}) error {
	select {
	case s.control <- i:
		return nil
	case <-s.done:
	}
	return dNCErrClosed
}

// Locks writing. All write requests will be blocked until write is unlocked with (*dNC).WriteUnlock, or server closes.
func (s *dNC) WriteLock() error {
	return s.Control(dNCCWriteLock{})
}

// Unlocks writing. All blocked write requests until now will be accepted.
func (s *dNC) WriteUnlock() error {
	return s.Control(dNCCWriteUnlock{})
}

// Unlocks writing and makes (*dNC).Write to result (0, dNCErrWrite).
func (s *dNC) WriteError() error {
	if err := s.WriteUnlock(); err != nil {
		return err
	}
	return s.Control(dNCCWriteError{})
}

// Unlocks writing and makes (*dNC).Write([...]) not result in error. See (*dNC).Write for details.
func (s *dNC) WriteSuccess() error {
	if err := s.WriteUnlock(); err != nil {
		return err
	}
	return s.Control(dNCCWriteSuccess{})
}

// Locks reading. All read requests will be blocked until read is unlocked with (*dNC).ReadUnlock, or server closes.
// (*dNC).Read([...]) wil block even after successful write.
func (s *dNC) ReadLock() error {
	return s.Control(dNCCReadLock{})
}

// Unlocks reading. If the internall buffer is not empty, next read will not block.
func (s *dNC) ReadUnlock() error {
	return s.Control(dNCCReadUnlock{})
}

// Unlocks read and makes every blocked and following (*dNC).Read([...]) imidiatly result in error. See (*dNC).Read for details.
func (s *dNC) ReadError() error {
	if err := s.ReadUnlock(); err != nil {
		return err
	}
	return s.Control(dNCCReadError{})
}

// Unlocks read and makes every blocked and following (*dNC).Read([...]) requests be handled, if according to internal buffer. See (*dNC).Read for details.
func (s *dNC) ReadSuccess() error {
	if err := s.ReadUnlock(); err != nil {
		return err
	}
	return s.Control(dNCCReadSuccess{})
}

// dummy X server replier for dummy net.Conn

type dXSEvent struct{}

func (_ dXSEvent) Bytes() []byte  { return nil }
func (_ dXSEvent) String() string { return "dummy X server event" }

type dXSError struct {
	seqId uint16
}

func (e dXSError) SequenceId() uint16 { return e.seqId }
func (_ dXSError) BadId() uint32      { return 0 }
func (_ dXSError) Error() string      { return "dummy X server error reply" }

func newDummyXServerReplier() func([]byte) []byte {
	// register xgb error & event replies
	NewErrorFuncs[255] = func(buf []byte) Error {
		return dXSError{Get16(buf[2:])}
	}
	NewEventFuncs[128&127] = func(buf []byte) Event {
		return dXSEvent{}
	}

	// sequence number generator
	seqId := uint16(1)
	incrementSequenceId := func() {
		// this has to be the same algorithm as in (*Conn).generateSeqIds
		if seqId == uint16((1<<16)-1) {
			seqId = 0
		} else {
			seqId++
		}
	}
	return func(request []byte) []byte {
		res := make([]byte, 32)
		switch string(request) {
		case "event":
			res[0] = 128
			return res
		case "error":
			res[0] = 0   // error
			res[1] = 255 // error function
		default:
			res[0] = 1 // reply
		}
		Put16(res[2:], seqId) // sequence number
		incrementSequenceId()
		if string(request) == "noreply" {
			return nil
		}
		return res
	}
}
