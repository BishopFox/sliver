package yamux

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type logCapture struct {
	mu  sync.Mutex
	buf *bytes.Buffer
}

var _ io.Writer = (*logCapture)(nil)

func (l *logCapture) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.buf == nil {
		l.buf = &bytes.Buffer{}
	}
	return l.buf.Write(p)
}
func (l *logCapture) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

func (l *logCapture) logs() []string {
	return strings.Split(strings.TrimSpace(l.String()), "\n")
}

func (l *logCapture) match(expect []string) bool {
	return reflect.DeepEqual(l.logs(), expect)
}

type pipeConn struct {
	reader       *io.PipeReader
	writer       *io.PipeWriter
	writeBlocker sync.Mutex
}

func (p *pipeConn) Read(b []byte) (int, error) {
	return p.reader.Read(b)
}

func (p *pipeConn) Write(b []byte) (int, error) {
	p.writeBlocker.Lock()
	defer p.writeBlocker.Unlock()
	return p.writer.Write(b)
}

func (p *pipeConn) Close() error {
	p.reader.Close()
	return p.writer.Close()
}

func testConnPipe(testing.TB) (io.ReadWriteCloser, io.ReadWriteCloser) {
	read1, write1 := io.Pipe()
	read2, write2 := io.Pipe()
	conn1 := &pipeConn{reader: read1, writer: write2}
	conn2 := &pipeConn{reader: read2, writer: write1}
	return conn1, conn2
}

func testConnTCP(t testing.TB) (io.ReadWriteCloser, io.ReadWriteCloser) {
	l, err := net.ListenTCP("tcp", nil)
	if err != nil {
		t.Fatalf("error creating listener: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	network := l.Addr().Network()
	addr := l.Addr().String()

	var server net.Conn
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		var err error
		server, err = l.Accept()
		if err != nil {
			errCh <- err
			return
		}
	}()

	t.Logf("Connecting to %s: %s", network, addr)
	client, err := net.DialTimeout(network, addr, 10*time.Second)
	if err != nil {
		t.Fatalf("error dialing tls listener: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	if err := <-errCh; err != nil {
		t.Fatalf("error creating tls server: %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })

	return client, server
}

func testConnTLS(t testing.TB) (io.ReadWriteCloser, io.ReadWriteCloser) {
	cert, err := tls.LoadX509KeyPair("testdata/cert.pem", "testdata/key.pem")
	if err != nil {
		t.Fatalf("error loading certificate: %v", err)
	}

	l, err := net.ListenTCP("tcp", nil)
	if err != nil {
		t.Fatalf("error creating listener: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	var server net.Conn
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		conn, err := l.Accept()
		if err != nil {
			errCh <- err
			return
		}

		server = tls.Server(conn, &tls.Config{
			Certificates: []tls.Certificate{cert},
		})
	}()

	t.Logf("Connecting to %s: %s", l.Addr().Network(), l.Addr())
	client, err := net.DialTimeout(l.Addr().Network(), l.Addr().String(), 10*time.Second)
	if err != nil {
		t.Fatalf("error dialing tls listener: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	tlsClient := tls.Client(client, &tls.Config{
		// InsecureSkipVerify is safe to use here since this is only for tests.
		InsecureSkipVerify: true,
	})

	if err := <-errCh; err != nil {
		t.Fatalf("error creating tls server: %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })

	return tlsClient, server
}

// connTypeFunc is func that returns a client and server connection for testing
// like testConnTLS.
//
// See connTypeTest
type connTypeFunc func(t testing.TB) (io.ReadWriteCloser, io.ReadWriteCloser)

// connTypeTest is a test case for a specific conn type.
//
// See testConnType
type connTypeTest struct {
	Name  string
	Conns connTypeFunc
}

// testConnType runs subtests of the given testFunc against multiple connection
// types.
func testConnTypes(t *testing.T, testFunc func(t testing.TB, client, server io.ReadWriteCloser)) {
	reverse := func(f connTypeFunc) connTypeFunc {
		return func(t testing.TB) (io.ReadWriteCloser, io.ReadWriteCloser) {
			c, s := f(t)
			return s, c
		}
	}
	cases := []connTypeTest{
		{
			Name:  "Pipes",
			Conns: testConnPipe,
		},
		{
			Name:  "TCP",
			Conns: testConnTCP,
		},
		{
			Name:  "TCP_Reverse",
			Conns: reverse(testConnTCP),
		},
		{
			Name:  "TLS",
			Conns: testConnTLS,
		},
		{
			Name:  "TLS_Reverse",
			Conns: reverse(testConnTLS),
		},
	}
	for i := range cases {
		tc := cases[i]
		t.Run(tc.Name, func(t *testing.T) {
			client, server := tc.Conns(t)
			testFunc(t, client, server)
		})
	}
}

func testConf() *Config {
	conf := DefaultConfig()
	conf.AcceptBacklog = 64
	conf.KeepAliveInterval = 100 * time.Millisecond
	conf.ConnectionWriteTimeout = 250 * time.Millisecond
	return conf
}

func captureLogs(conf *Config) *logCapture {
	buf := new(logCapture)
	conf.Logger = log.New(buf, "", 0)
	conf.LogOutput = nil
	return buf
}

func testConfNoKeepAlive() *Config {
	conf := testConf()
	conf.EnableKeepAlive = false
	return conf
}

func testClientServer(t testing.TB) (*Session, *Session) {
	client, server := testConnTLS(t)
	return testClientServerConfig(t, client, server, testConf(), testConf())
}

func testClientServerConfig(
	t testing.TB,
	clientConn, serverConn io.ReadWriteCloser,
	clientConf, serverConf *Config,
) (clientSession *Session, serverSession *Session) {

	var err error

	clientSession, err = Client(clientConn, clientConf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })

	serverSession, err = Server(serverConn, serverConf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	return clientSession, serverSession
}

func TestPing(t *testing.T) {
	client, server := testClientServer(t)

	rtt, err := client.Ping()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rtt == 0 {
		t.Fatalf("bad: %v", rtt)
	}

	rtt, err = server.Ping()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rtt == 0 {
		t.Fatalf("bad: %v", rtt)
	}
}

func TestPing_Timeout(t *testing.T) {
	conf := testConfNoKeepAlive()
	clientPipe, serverPipe := testConnPipe(t)
	client, server := testClientServerConfig(t, clientPipe, serverPipe, conf.Clone(), conf.Clone())

	// Prevent the client from responding
	clientConn := client.conn.(*pipeConn)
	clientConn.writeBlocker.Lock()

	errCh := make(chan error, 1)
	go func() {
		_, err := server.Ping() // Ping via the server session
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err != ErrTimeout {
			t.Fatalf("err: %v", err)
		}
	case <-time.After(client.config.ConnectionWriteTimeout * 2):
		t.Fatalf("failed to timeout within expected %v", client.config.ConnectionWriteTimeout)
	}

	// Verify that we recover, even if we gave up
	clientConn.writeBlocker.Unlock()

	go func() {
		_, err := server.Ping() // Ping via the server session
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	case <-time.After(client.config.ConnectionWriteTimeout):
		t.Fatalf("timeout")
	}
}

func TestCloseBeforeAck(t *testing.T) {
	testConnTypes(t, func(t testing.TB, clientConn, serverConn io.ReadWriteCloser) {
		cfg := testConf()
		cfg.AcceptBacklog = 8
		client, server := testClientServerConfig(t, clientConn, serverConn, cfg, cfg.Clone())

		for i := 0; i < 8; i++ {
			s, err := client.OpenStream()
			if err != nil {
				t.Fatal(err)
			}
			s.Close()
		}

		for i := 0; i < 8; i++ {
			s, err := server.AcceptStream()
			if err != nil {
				t.Fatal(err)
			}
			s.Close()
		}

		errCh := make(chan error, 1)
		go func() {
			s, err := client.OpenStream()
			if err != nil {
				errCh <- err
				return
			}
			s.Close()
			errCh <- nil
		}()

		drainErrorsUntil(t, errCh, 1, time.Second*5, "timed out trying to open stream")
	})
}

func TestAccept(t *testing.T) {
	client, server := testClientServer(t)

	if client.NumStreams() != 0 {
		t.Fatalf("bad")
	}
	if server.NumStreams() != 0 {
		t.Fatalf("bad")
	}

	errCh := make(chan error, 4)
	acceptOne := func(streamFunc func() (*Stream, error), expectID uint32) {
		stream, err := streamFunc()
		if err != nil {
			errCh <- err
			return
		}
		if id := stream.StreamID(); id != expectID {
			errCh <- fmt.Errorf("bad: %v", id)
			return
		}
		if err := stream.Close(); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}

	go acceptOne(server.AcceptStream, 1)
	go acceptOne(client.AcceptStream, 2)
	go acceptOne(server.OpenStream, 2)
	go acceptOne(client.OpenStream, 1)

	drainErrorsUntil(t, errCh, 4, time.Second, "timeout")
}

func TestOpenStreamTimeout(t *testing.T) {
	const timeout = 25 * time.Millisecond

	testConnTypes(t, func(t testing.TB, clientConn, serverConn io.ReadWriteCloser) {
		serverConf := testConf()
		serverConf.StreamOpenTimeout = timeout

		clientConf := serverConf.Clone()
		clientLogs := captureLogs(clientConf)

		client, _ := testClientServerConfig(t, clientConn, serverConn, clientConf, serverConf)

		// Open a single stream without a server to acknowledge it.
		s, err := client.OpenStream()
		if err != nil {
			t.Fatal(err)
		}

		// Sleep for longer than the stream open timeout.
		// Since no ACKs are received, the stream and session should be closed.
		time.Sleep(timeout * 5)

		// Support multiple underlying connection types
		var dest string
		switch conn := clientConn.(type) {
		case net.Conn:
			dest = conn.RemoteAddr().String()
		case *pipeConn:
			dest = "yamux:remote"
		default:
			t.Fatalf("unsupported connection type %T - please update test", conn)
		}
		exp := fmt.Sprintf("[ERR] yamux: aborted stream open (destination=%s): i/o deadline reached", dest)

		if !clientLogs.match([]string{exp}) {
			t.Fatalf("server log incorect: %v\nexpected: %v", clientLogs.logs(), exp)
		}

		s.stateLock.Lock()
		state := s.state
		s.stateLock.Unlock()

		if state != streamClosed {
			t.Fatalf("stream should have been closed")
		}
		if !client.IsClosed() {
			t.Fatalf("session should have been closed")
		}
	})
}

func TestClose_closeTimeout(t *testing.T) {
	conf := testConf()
	conf.StreamCloseTimeout = 10 * time.Millisecond
	clientConn, serverConn := testConnTLS(t)
	client, server := testClientServerConfig(t, clientConn, serverConn, conf, conf.Clone())

	if client.NumStreams() != 0 {
		t.Fatalf("bad")
	}
	if server.NumStreams() != 0 {
		t.Fatalf("bad")
	}

	errCh := make(chan error, 2)

	// Open a stream on the client but only close it on the server.
	// We want to see if the stream ever gets cleaned up on the client.

	var clientStream *Stream
	go func() {
		var err error
		clientStream, err = client.OpenStream()
		errCh <- err
	}()

	go func() {
		stream, err := server.AcceptStream()
		if err != nil {
			errCh <- err
			return
		}
		if err := stream.Close(); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	drainErrorsUntil(t, errCh, 2, time.Second, "timeout")

	// We should have zero streams after our timeout period
	time.Sleep(100 * time.Millisecond)

	if v := server.NumStreams(); v > 0 {
		t.Fatalf("should have zero streams: %d", v)
	}
	if v := client.NumStreams(); v > 0 {
		t.Fatalf("should have zero streams: %d", v)
	}

	if _, err := clientStream.Write([]byte("hello")); err == nil {
		t.Fatal("should error on write")
	} else if err.Error() != "connection reset" {
		t.Fatalf("expected connection reset, got %q", err)
	}
}

func TestNonNilInterface(t *testing.T) {
	_, server := testClientServer(t)
	server.Close()

	conn, err := server.Accept()
	if err == nil || !errors.Is(err, ErrSessionShutdown) || conn != nil {
		t.Fatal("bad: accept should return a shutdown error and a connection of nil value")
	}
	if err != nil && conn != nil {
		t.Error("bad: accept should return a connection of nil value")
	}

	conn, err = server.Open()
	if err == nil || !errors.Is(err, ErrSessionShutdown) || conn != nil {
		t.Fatal("bad: open should return a shutdown error and a connection of nil value")
	}
}

func TestSendData_Small(t *testing.T) {
	client, server := testClientServer(t)

	errCh := make(chan error, 2)

	// Accept an incoming client and perform some reads before closing
	go func() {
		stream, err := server.AcceptStream()
		if err != nil {
			errCh <- err
			return
		}

		if server.NumStreams() != 1 {
			errCh <- fmt.Errorf("bad")
			return
		}

		buf := make([]byte, 4)
		for i := 0; i < 1000; i++ {
			n, err := stream.Read(buf)
			if err != nil {
				errCh <- err
				return
			}
			if n != 4 {
				errCh <- fmt.Errorf("short read: %d", n)
				return
			}
			if string(buf) != "test" {
				errCh <- fmt.Errorf("bad: %s", buf)
				return
			}
		}

		if err := stream.Close(); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	// Open a client and perform some writes before closing
	go func() {
		stream, err := client.Open()
		if err != nil {
			errCh <- err
			return
		}

		if client.NumStreams() != 1 {
			errCh <- fmt.Errorf("bad")
			return
		}

		for i := 0; i < 1000; i++ {
			n, err := stream.Write([]byte("test"))
			if err != nil {
				errCh <- err
				return
			}
			if n != 4 {
				errCh <- fmt.Errorf("short write %d", n)
				return
			}
		}

		if err := stream.Close(); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	drainErrorsUntil(t, errCh, 2, 5*time.Second, "timeout")

	// Give client and server a second to receive FINs and close streams
	time.Sleep(time.Second)

	if n := client.NumStreams(); n != 0 {
		t.Errorf("expected 0 client streams but found %d", n)
	}
	if n := server.NumStreams(); n != 0 {
		t.Errorf("expected 0 server streams but found %d", n)
	}
}

func TestSendData_Large(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test that may time out on the race detector")
	}
	client, server := testClientServer(t)

	const (
		sendSize = 250 * 1024 * 1024
		recvSize = 4 * 1024
	)

	data := make([]byte, sendSize)
	for idx := range data {
		data[idx] = byte(idx % 256)
	}

	errCh := make(chan error, 2)

	go func() {
		stream, err := server.AcceptStream()
		if err != nil {
			errCh <- err
			return
		}
		var sz int
		buf := make([]byte, recvSize)
		for i := 0; i < sendSize/recvSize; i++ {
			n, err := stream.Read(buf)
			if err != nil {
				errCh <- err
				return
			}
			if n != recvSize {
				errCh <- fmt.Errorf("short read: %d", n)
				return
			}
			sz += n
			for idx := range buf {
				if buf[idx] != byte(idx%256) {
					errCh <- fmt.Errorf("bad: %v %v %v", i, idx, buf[idx])
					return
				}
			}
		}

		if err := stream.Close(); err != nil {
			errCh <- err
			return
		}

		t.Logf("cap=%d, n=%d\n", stream.recvBuf.Cap(), sz)
		errCh <- nil
	}()

	go func() {
		stream, err := client.Open()
		if err != nil {
			errCh <- err
			return
		}

		n, err := stream.Write(data)
		if err != nil {
			errCh <- err
			return
		}
		if n != len(data) {
			errCh <- fmt.Errorf("short write %d", n)
			return
		}

		if err := stream.Close(); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	drainErrorsUntil(t, errCh, 2, 10*time.Second, "timeout")
}

func TestGoAway(t *testing.T) {
	client, server := testClientServer(t)

	if err := server.GoAway(); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Give the other side time to process the goaway after receiving it.
	time.Sleep(100 * time.Millisecond)

	_, err := client.Open()
	if err != ErrRemoteGoAway {
		t.Fatalf("err: %v", err)
	}
}

func TestManyStreams(t *testing.T) {
	client, server := testClientServer(t)

	const streams = 50

	errCh := make(chan error, 2*streams)

	acceptor := func() {
		stream, err := server.AcceptStream()
		if err != nil {
			errCh <- err
			return
		}
		defer stream.Close()

		buf := make([]byte, 512)
		for {
			n, err := stream.Read(buf)
			if err == io.EOF {
				errCh <- nil
				return
			}
			if err != nil {
				errCh <- err
				return
			}
			if n == 0 {
				errCh <- fmt.Errorf("no bytes read")
				return
			}
		}
	}
	sender := func(id int) {
		stream, err := client.Open()
		if err != nil {
			errCh <- err
			return
		}
		defer stream.Close()

		msg := fmt.Sprintf("%08d", id)
		for i := 0; i < 1000; i++ {
			n, err := stream.Write([]byte(msg))
			if err != nil {
				errCh <- err
				return
			}
			if n != len(msg) {
				errCh <- fmt.Errorf("short write %d", n)
				return
			}
		}
		errCh <- nil
	}

	for i := 0; i < streams; i++ {
		go acceptor()
		go sender(i)
	}

	drainErrorsUntil(t, errCh, 2*streams, 0, "")
}

func TestManyStreams_PingPong(t *testing.T) {
	client, server := testClientServer(t)

	const streams = 50

	errCh := make(chan error, 2*streams)

	ping := []byte("ping")
	pong := []byte("pong")

	acceptor := func() {
		stream, err := server.AcceptStream()
		if err != nil {
			errCh <- err
			return
		}
		defer stream.Close()

		buf := make([]byte, 4)
		for {
			// Read the 'ping'
			n, err := stream.Read(buf)
			if err == io.EOF {
				errCh <- nil
				return
			}
			if err != nil {
				errCh <- err
				return
			}
			if n != 4 {
				errCh <- fmt.Errorf("short read %d", n)
				return
			}
			if !bytes.Equal(buf, ping) {
				errCh <- fmt.Errorf("bad: %s", buf)
				return
			}

			// Shrink the internal buffer!
			stream.Shrink()

			// Write out the 'pong'
			n, err = stream.Write(pong)
			if err != nil {
				errCh <- err
				return
			}
			if n != 4 {
				errCh <- fmt.Errorf("short write %d", n)
				return
			}
		}
	}
	sender := func() {
		stream, err := client.OpenStream()
		if err != nil {
			errCh <- err
			return
		}
		defer stream.Close()

		buf := make([]byte, 4)
		for i := 0; i < 1000; i++ {
			// Send the 'ping'
			n, err := stream.Write(ping)
			if err != nil {
				errCh <- err
				return
			}
			if n != 4 {
				errCh <- fmt.Errorf("short write %d", n)
				return
			}

			// Read the 'pong'
			n, err = stream.Read(buf)
			if err != nil {
				errCh <- err
				return
			}
			if n != 4 {
				errCh <- fmt.Errorf("short read %d", n)
				return
			}
			if !bytes.Equal(buf, pong) {
				errCh <- fmt.Errorf("bad: %s", buf)
				return
			}

			// Shrink the buffer
			stream.Shrink()
		}
		errCh <- nil
	}

	for i := 0; i < streams; i++ {
		go acceptor()
		go sender()
	}

	drainErrorsUntil(t, errCh, 2*streams, 0, "")
}

// TestHalfClose asserts that half closed streams can still read.
func TestHalfClose(t *testing.T) {
	testConnTypes(t, func(t testing.TB, clientConn, serverConn io.ReadWriteCloser) {
		client, server := testClientServerConfig(t, clientConn, serverConn, testConf(), testConf())

		clientStream, err := client.Open()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if _, err = clientStream.Write([]byte("a")); err != nil {
			t.Fatalf("err: %v", err)
		}

		serverStream, err := server.Accept()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		serverStream.Close() // Half close

		// Server reads 1 byte written by Client
		buf := make([]byte, 4)
		n, err := serverStream.Read(buf)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if n != 1 {
			t.Fatalf("bad: %v", n)
		}

		// Send more
		if _, err = clientStream.Write([]byte("bcd")); err != nil {
			t.Fatalf("err: %v", err)
		}
		clientStream.Close()

		// Read after close always returns the bytes written but may or may not
		// receive the EOF.
		n, err = serverStream.Read(buf)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if n != 3 {
			t.Fatalf("bad: %v", n)
		}

		// EOF after close
		n, err = serverStream.Read(buf)
		if err != io.EOF {
			t.Fatalf("err: %v", err)
		}
		if n != 0 {
			t.Fatalf("bad: %v", n)
		}
	})
}

func TestHalfCloseSessionShutdown(t *testing.T) {
	client, server := testClientServer(t)

	// dataSize must be large enough to ensure the server will send a window
	// update
	dataSize := int64(server.config.MaxStreamWindowSize)

	data := make([]byte, dataSize)
	for idx := range data {
		data[idx] = byte(idx % 256)
	}

	stream, err := client.Open()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if _, err = stream.Write(data); err != nil {
		t.Fatalf("err: %v", err)
	}

	stream2, err := server.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if err := stream.Close(); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Shut down the session of the sending side. This should not cause reads
	// to fail on the receiving side.
	if err := client.Close(); err != nil {
		t.Fatalf("err: %v", err)
	}

	buf := make([]byte, dataSize)
	n, err := stream2.Read(buf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if int64(n) != dataSize {
		t.Fatalf("bad: %v", n)
	}

	// EOF after close
	n, err = stream2.Read(buf)
	if err != io.EOF {
		t.Fatalf("err: %v", err)
	}
	if n != 0 {
		t.Fatalf("bad: %v", n)
	}
}

func TestReadDeadline(t *testing.T) {
	client, server := testClientServer(t)

	stream, err := client.Open()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream.Close()

	stream2, err := server.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream2.Close()

	if err := stream.SetReadDeadline(time.Now().Add(5 * time.Millisecond)); err != nil {
		t.Fatalf("err: %v", err)
	}

	buf := make([]byte, 4)
	_, err = stream.Read(buf)
	if err != ErrTimeout {
		t.Fatalf("err: %v", err)
	}

	// See https://github.com/hashicorp/yamux/issues/90
	// The standard library's http server package will read from connections in
	// the background to detect if they are alive.
	//
	// It sets a read deadline on connections and detect if the returned error
	// is a network timeout error which implements net.Error.
	//
	// The HTTP server will cancel all server requests if it isn't timeout error
	// from the connection.
	//
	// We assert that we return an error meeting the interface to avoid
	// accidently breaking yamux session compatability with the standard
	// library's http server implementation.
	if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		t.Fatalf("reading timeout error is expected to implement net.Error and return true when calling Timeout()")
	}
}

func TestReadDeadline_BlockedRead(t *testing.T) {
	client, server := testClientServer(t)

	stream, err := client.Open()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream.Close()

	stream2, err := server.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream2.Close()

	// Start a read that will block
	errCh := make(chan error, 1)
	go func() {
		buf := make([]byte, 4)
		_, err := stream.Read(buf)
		errCh <- err
		close(errCh)
	}()

	// Wait to ensure the read has started.
	time.Sleep(5 * time.Millisecond)

	// Update the read deadline
	if err := stream.SetReadDeadline(time.Now().Add(5 * time.Millisecond)); err != nil {
		t.Fatalf("err: %v", err)
	}

	select {
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected read timeout")
	case err := <-errCh:
		if err != ErrTimeout {
			t.Fatalf("expected ErrTimeout; got %v", err)
		}
	}
}

func TestWriteDeadline(t *testing.T) {
	client, server := testClientServer(t)

	stream, err := client.Open()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream.Close()

	stream2, err := server.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream2.Close()

	if err := stream.SetWriteDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
		t.Fatalf("err: %v", err)
	}

	buf := make([]byte, 512)
	for i := 0; i < int(initialStreamWindow); i++ {
		_, err := stream.Write(buf)
		if err != nil && err == ErrTimeout {
			return
		} else if err != nil {
			t.Fatalf("err: %v", err)
		}
	}
	t.Fatalf("Expected timeout")
}

func TestWriteDeadline_BlockedWrite(t *testing.T) {
	client, server := testClientServer(t)

	stream, err := client.Open()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream.Close()

	stream2, err := server.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream2.Close()

	// Start a goroutine making writes that will block
	errCh := make(chan error, 1)
	go func() {
		buf := make([]byte, 512)
		for i := 0; i < int(initialStreamWindow); i++ {
			_, err := stream.Write(buf)
			if err == nil {
				continue
			}

			errCh <- err
			close(errCh)
			return
		}

		close(errCh)
	}()

	// Wait to ensure the write has started.
	time.Sleep(5 * time.Millisecond)

	// Update the write deadline
	if err := stream.SetWriteDeadline(time.Now().Add(5 * time.Millisecond)); err != nil {
		t.Fatalf("err: %v", err)
	}

	select {
	case <-time.After(1 * time.Second):
		t.Fatal("expected write timeout")
	case err := <-errCh:
		if err != ErrTimeout {
			t.Fatalf("expected ErrTimeout; got %v", err)
		}
	}
}

func TestBacklogExceeded(t *testing.T) {
	client, server := testClientServer(t)

	// Fill the backlog
	max := client.config.AcceptBacklog
	for i := 0; i < max; i++ {
		stream, err := client.Open()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		if _, err := stream.Write([]byte("foo")); err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	// Attempt to open a new stream
	errCh := make(chan error, 1)
	go func() {
		_, err := client.Open()
		errCh <- err
	}()

	// Shutdown the server
	go func() {
		time.Sleep(10 * time.Millisecond)
		server.Close()
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatalf("open should fail")
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}
}

func TestKeepAlive(t *testing.T) {
	testConnTypes(t, func(t testing.TB, clientConn, serverConn io.ReadWriteCloser) {
		client, server := testClientServerConfig(t, clientConn, serverConn, testConf(), testConf())

		// Give keepalives time to happen
		time.Sleep(200 * time.Millisecond)

		// Ping value should increase
		client.pingLock.Lock()
		defer client.pingLock.Unlock()
		if client.pingID == 0 {
			t.Fatalf("should ping")
		}

		server.pingLock.Lock()
		defer server.pingLock.Unlock()
		if server.pingID == 0 {
			t.Fatalf("should ping")
		}
	})
}

func TestKeepAlive_Timeout(t *testing.T) {
	conn1, conn2 := testConnPipe(t)

	clientConf := testConf()
	clientConf.ConnectionWriteTimeout = time.Hour // We're testing keep alives, not connection writes
	clientConf.EnableKeepAlive = false            // Just test one direction, so it's deterministic who hangs up on whom
	_ = captureLogs(clientConf)                   // Client logs aren't part of the test
	client, err := Client(conn1, clientConf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer client.Close()

	serverConf := testConf()
	serverLogs := captureLogs(serverConf)
	server, err := Server(conn2, serverConf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer server.Close()

	errCh := make(chan error, 1)
	go func() {
		_, err := server.Accept() // Wait until server closes
		errCh <- err
	}()

	// Prevent the client from responding
	clientConn := client.conn.(*pipeConn)
	clientConn.writeBlocker.Lock()

	select {
	case err := <-errCh:
		if err != ErrKeepAliveTimeout {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for timeout")
	}

	clientConn.writeBlocker.Unlock()

	if !server.IsClosed() {
		t.Fatalf("server should have closed")
	}

	if !serverLogs.match([]string{"[ERR] yamux: keepalive failed: i/o deadline reached"}) {
		t.Fatalf("server log incorect: %v", serverLogs.logs())
	}
}

func TestLargeWindow(t *testing.T) {
	conf := DefaultConfig()
	conf.MaxStreamWindowSize *= 2

	clientConn, serverConn := testConnTLS(t)
	client, server := testClientServerConfig(t, clientConn, serverConn, conf, conf.Clone())

	stream, err := client.Open()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream.Close()

	stream2, err := server.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream2.Close()

	err = stream.SetWriteDeadline(time.Now().Add(10 * time.Millisecond))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	buf := make([]byte, conf.MaxStreamWindowSize)
	n, err := stream.Write(buf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != len(buf) {
		t.Fatalf("short write: %d", n)
	}
}

type UnlimitedReader struct{}

func (u *UnlimitedReader) Read(p []byte) (int, error) {
	runtime.Gosched()
	return len(p), nil
}

func TestSendData_VeryLarge(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test that may time out on the race detector")
	}
	client, server := testClientServer(t)

	var n int64 = 1 * 1024 * 1024 * 1024
	var workers int = 16

	errCh := make(chan error, workers*2)

	for i := 0; i < workers; i++ {
		go func() {
			stream, err := server.AcceptStream()
			if err != nil {
				errCh <- err
				return
			}
			defer stream.Close()

			buf := make([]byte, 4)
			_, err = stream.Read(buf)
			if err != nil {
				errCh <- err
				return
			}
			if !bytes.Equal(buf, []byte{0, 1, 2, 3}) {
				errCh <- errors.New("bad header")
				return
			}

			recv, err := io.Copy(io.Discard, stream)
			if err != nil {
				errCh <- err
				return
			}
			if recv != n {
				errCh <- fmt.Errorf("bad: %v", recv)
				return
			}

			errCh <- nil
		}()
	}
	for i := 0; i < workers; i++ {
		go func() {
			stream, err := client.Open()
			if err != nil {
				errCh <- err
				return
			}
			defer stream.Close()

			_, err = stream.Write([]byte{0, 1, 2, 3})
			if err != nil {
				errCh <- err
				return
			}

			unlimited := &UnlimitedReader{}
			sent, err := io.Copy(stream, io.LimitReader(unlimited, n))
			if err != nil {
				errCh <- err
				return
			}
			if sent != n {
				errCh <- fmt.Errorf("bad: %v", sent)
				return
			}

			errCh <- nil
		}()
	}

	drainErrorsUntil(t, errCh, workers*2, 120*time.Second, "timeout")
}

func TestBacklogExceeded_Accept(t *testing.T) {
	client, server := testClientServer(t)

	max := 5 * client.config.AcceptBacklog

	errCh := make(chan error, max)
	go func() {
		for i := 0; i < max; i++ {
			stream, err := server.Accept()
			if err != nil {
				errCh <- err
				return
			}
			defer stream.Close()
			errCh <- nil
		}
	}()

	// Fill the backlog
	for i := 0; i < max; i++ {
		stream, err := client.Open()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		if _, err := stream.Write([]byte("foo")); err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	drainErrorsUntil(t, errCh, max, 0, "")
}

func TestSession_WindowUpdateWriteDuringRead(t *testing.T) {
	conf := testConfNoKeepAlive()

	clientConn, serverConn := testConnPipe(t)
	client, server := testClientServerConfig(t, clientConn, serverConn, conf, conf.Clone())

	// Choose a huge flood size that we know will result in a window update.
	flood := int64(client.config.MaxStreamWindowSize) - 1

	errCh := make(chan error, 2)

	// The server will accept a new stream and then flood data to it.
	go func() {
		stream, err := server.AcceptStream()
		if err != nil {
			errCh <- err
			return
		}
		defer stream.Close()

		n, err := stream.Write(make([]byte, flood))
		if err != nil {
			errCh <- err
			return
		}
		if int64(n) != flood {
			errCh <- fmt.Errorf("short write: %d", n)
		}

		errCh <- nil
	}()

	// The client will open a stream, block outbound writes, and then
	// listen to the flood from the server, which should time out since
	// it won't be able to send the window update.
	go func() {
		stream, err := client.OpenStream()
		if err != nil {
			errCh <- err
			return
		}
		defer stream.Close()

		conn := clientConn.(*pipeConn)
		conn.writeBlocker.Lock()
		defer conn.writeBlocker.Unlock()

		_, err = stream.Read(make([]byte, flood))
		if err != ErrConnectionWriteTimeout {
			errCh <- err
			return
		}

		errCh <- nil
	}()

	drainErrorsUntil(t, errCh, 2, 0, "")
}

// TestSession_PartialReadWindowUpdate asserts that when a client performs a
// partial read it updates the server's send window.
func TestSession_PartialReadWindowUpdate(t *testing.T) {
	testConnTypes(t, func(t testing.TB, clientConn, serverConn io.ReadWriteCloser) {
		conf := testConfNoKeepAlive()

		client, server := testClientServerConfig(t, clientConn, serverConn, conf, conf.Clone())

		errCh := make(chan error, 1)

		// Choose a huge flood size that we know will result in a window update.
		flood := int64(client.config.MaxStreamWindowSize)
		var wr *Stream

		// The server will accept a new stream and then flood data to it.
		go func() {
			var err error
			wr, err = server.AcceptStream()
			if err != nil {
				errCh <- err
				return
			}
			defer wr.Close()

			window := atomic.LoadUint32(&wr.sendWindow)
			if window != client.config.MaxStreamWindowSize {
				errCh <- fmt.Errorf("sendWindow: exp=%d, got=%d", client.config.MaxStreamWindowSize, window)
				return
			}

			n, err := wr.Write(make([]byte, flood))
			if err != nil {
				errCh <- err
				return
			}
			if int64(n) != flood {
				errCh <- fmt.Errorf("short write: %d", n)
				return
			}
			window = atomic.LoadUint32(&wr.sendWindow)
			if window != 0 {
				errCh <- fmt.Errorf("sendWindow: exp=%d, got=%d", 0, window)
				return
			}
			errCh <- err
		}()

		stream, err := client.OpenStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		drainErrorsUntil(t, errCh, 1, 0, "")

		// Only read part of the flood
		partialReadSize := flood/2 + 1
		_, err = stream.Read(make([]byte, partialReadSize))
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		// Wait for window update to be applied by server. Should be "instant" but CI
		// can be slow.
		time.Sleep(2 * time.Second)

		// Assert server received window update
		window := atomic.LoadUint32(&wr.sendWindow)
		if exp := uint32(partialReadSize); window != exp {
			t.Fatalf("sendWindow: exp=%d, got=%d", exp, window)
		}
	})
}

func TestSession_sendNoWait_Timeout(t *testing.T) {
	conf := testConfNoKeepAlive()

	clientConn, serverConn := testConnPipe(t)
	client, server := testClientServerConfig(t, clientConn, serverConn, conf, conf.Clone())

	errCh := make(chan error, 2)

	go func() {
		stream, err := server.AcceptStream()
		if err != nil {
			errCh <- err
			return
		}
		defer stream.Close()
		errCh <- nil
	}()

	// The client will open the stream and then block outbound writes, we'll
	// probe sendNoWait once it gets into that state.
	go func() {
		stream, err := client.OpenStream()
		if err != nil {
			errCh <- err
			return
		}
		defer stream.Close()

		conn := clientConn.(*pipeConn)
		conn.writeBlocker.Lock()
		defer conn.writeBlocker.Unlock()

		hdr := header(make([]byte, headerSize))
		hdr.encode(typePing, flagACK, 0, 0)
		for {
			err = client.sendNoWait(hdr)
			if err == nil {
				continue
			} else if err == ErrConnectionWriteTimeout {
				break
			} else {
				errCh <- err
				return
			}
		}
		errCh <- nil
	}()

	drainErrorsUntil(t, errCh, 2, 0, "")
}

func TestSession_PingOfDeath(t *testing.T) {
	conf := testConfNoKeepAlive()

	clientConn, serverConn := testConnPipe(t)
	client, server := testClientServerConfig(t, clientConn, serverConn, conf, conf.Clone())

	errCh := make(chan error, 2)

	var doPingOfDeath sync.Mutex
	doPingOfDeath.Lock()

	// This is used later to block outbound writes.
	conn := server.conn.(*pipeConn)

	// The server will accept a stream, block outbound writes, and then
	// flood its send channel so that no more headers can be queued.
	go func() {
		stream, err := server.AcceptStream()
		if err != nil {
			errCh <- err
			return
		}
		defer stream.Close()

		conn.writeBlocker.Lock()
		for {
			hdr := header(make([]byte, headerSize))
			hdr.encode(typePing, 0, 0, 0)
			err = server.sendNoWait(hdr)
			if err == nil {
				continue
			} else if err == ErrConnectionWriteTimeout {
				break
			} else {
				errCh <- err
				return
			}
		}

		doPingOfDeath.Unlock()
		errCh <- nil
	}()

	// The client will open a stream and then send the server a ping once it
	// can no longer write. This makes sure the server doesn't deadlock reads
	// while trying to reply to the ping with no ability to write.
	go func() {
		stream, err := client.OpenStream()
		if err != nil {
			errCh <- err
			return
		}
		defer stream.Close()

		// This ping will never unblock because the ping id will never
		// show up in a response.
		doPingOfDeath.Lock()
		go func() { _, _ = client.Ping() }()

		// Wait for a while to make sure the previous ping times out,
		// then turn writes back on and make sure a ping works again.
		time.Sleep(2 * server.config.ConnectionWriteTimeout)
		conn.writeBlocker.Unlock()
		if _, err = client.Ping(); err != nil {
			errCh <- err
			return
		}

		errCh <- nil
	}()

	drainErrorsUntil(t, errCh, 2, 0, "")
}

func TestSession_ConnectionWriteTimeout(t *testing.T) {
	conf := testConfNoKeepAlive()

	clientConn, serverConn := testConnPipe(t)
	client, server := testClientServerConfig(t, clientConn, serverConn, conf, conf.Clone())

	errCh := make(chan error, 2)

	go func() {
		stream, err := server.AcceptStream()
		if err != nil {
			errCh <- err
			return
		}
		defer stream.Close()
		errCh <- nil
	}()

	// The client will open the stream and then block outbound writes, we'll
	// tee up a write and make sure it eventually times out.
	go func() {
		stream, err := client.OpenStream()
		if err != nil {
			errCh <- err
			return
		}
		defer stream.Close()

		conn := clientConn.(*pipeConn)
		conn.writeBlocker.Lock()
		defer conn.writeBlocker.Unlock()

		// Since the write goroutine is blocked then this will return a
		// timeout since it can't get feedback about whether the write
		// worked.
		n, err := stream.Write([]byte("hello"))
		if err != ErrConnectionWriteTimeout {
			errCh <- err
			return
		}
		if n != 0 {
			errCh <- fmt.Errorf("lied about writes: %d", n)
		}
		errCh <- nil
	}()

	drainErrorsUntil(t, errCh, 2, 0, "")
}

func TestCancelAccept(t *testing.T) {
	_, server := testClientServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	errCh := make(chan error, 1)

	go func() {
		stream, err := server.AcceptStreamWithContext(ctx)
		if err != context.Canceled {
			errCh <- err
			return
		}

		if stream != nil {
			defer stream.Close()
		}
		errCh <- nil
	}()

	cancel()

	drainErrorsUntil(t, errCh, 1, 0, "")
}

// drainErrorsUntil receives `expect` errors from errCh within `timeout`. Fails
// on any non-nil errors.
func drainErrorsUntil(t testing.TB, errCh chan error, expect int, timeout time.Duration, msg string) {
	t.Helper()
	start := time.Now()
	var timerC <-chan time.Time
	if timeout > 0 {
		timerC = time.After(timeout)
	}

	for found := 0; found < expect; {
		select {
		case <-timerC:
			t.Fatalf(msg+" (timeout was %v)", timeout)
		case err := <-errCh:
			if err != nil {
				t.Fatalf("err: %v", err)
			} else {
				found++
			}
		}
	}
	t.Logf("drain took %v (timeout was %v)", time.Since(start), timeout)
}
