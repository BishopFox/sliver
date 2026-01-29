package c2_test

/*
	Sliver Implant Framework
	Copyright (C) 2025  Bishop Fox

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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	implantCrypto "github.com/bishopfox/sliver/implant/sliver/cryptography"
	implantHandlers "github.com/bishopfox/sliver/implant/sliver/handlers"
	implantTransports "github.com/bishopfox/sliver/implant/sliver/transports"
	implantHTTP "github.com/bishopfox/sliver/implant/sliver/transports/httpclient"
	implantMTLS "github.com/bishopfox/sliver/implant/sliver/transports/mtls"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	c2 "github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/core"
	serverTransport "github.com/bishopfox/sliver/server/transport"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

const (
	sessionWaitTimeout = 10 * time.Second
	requestTimeout     = 10 * time.Second

	testMTLSServerHost = "localhost"
	testHTTPOrigin     = "https://sliver.test"
)

func TestE2EMTLSInfoAndLs(t *testing.T) {
	clearSessions()
	implantMTLS.SetTestCertificates(mtlsCACertPEM, mtlsCertPEM, mtlsKeyPEM)

	serverTLS, clientTLS, cleanup := newMTLSConnPair(t)
	defer cleanup()

	go c2.HandleSliverConnectionForTest(serverTLS)

	register := newRegister(t, fmt.Sprintf("e2e-mtls-%d", time.Now().UnixNano()), "mtls://"+testMTLSServerHost)
	conn := newMTLSConnection(t, clientTLS)
	stopImplant := startImplantSession(t, conn, register)
	defer stopImplant()

	session := waitForSession(t, register.Name)
	defer session.Connection.Cleanup()

	rpcClient, rpcCleanup := newGRPCClient(t)
	defer rpcCleanup()

	assertSessionInfo(t, session, register, consts.MtlsStr)
	assertLsRoundTrip(t, rpcClient, session.ID)
}

func TestE2EHTTPSInfoAndLs(t *testing.T) {
	clearSessions()

	server := newHTTPTestServer(t)
	conn := newHTTPConnection(t, server.HTTPServer.Handler, testHTTPOrigin)

	register := newRegister(t, fmt.Sprintf("e2e-https-%d", time.Now().UnixNano()), testHTTPOrigin)
	stopImplant := startImplantSession(t, conn, register)
	defer stopImplant()

	session := waitForSession(t, register.Name)
	defer session.Connection.Cleanup()

	rpcClient, rpcCleanup := newGRPCClient(t)
	defer rpcCleanup()

	assertSessionInfo(t, session, register, "http(s)")
	assertLsRoundTrip(t, rpcClient, session.ID)
}

func newMTLSConnPair(t *testing.T) (*tls.Conn, *tls.Conn, func()) {
	t.Helper()
	serverConfig := c2.TestServerTLSConfig(testMTLSServerHost)
	if serverConfig == nil {
		t.Fatalf("mtls server tls config missing")
	}
	clientConfig := testMTLSClientConfig(t)

	serverRaw, clientRaw := net.Pipe()
	serverTLS := tls.Server(serverRaw, serverConfig)
	clientTLS := tls.Client(clientRaw, clientConfig)

	errCh := make(chan error, 2)
	go func() {
		errCh <- serverTLS.Handshake()
	}()
	go func() {
		errCh <- clientTLS.Handshake()
	}()
	for range 2 {
		if err := <-errCh; err != nil {
			serverTLS.Close()
			clientTLS.Close()
			t.Fatalf("mtls handshake failed: %v", err)
		}
	}

	cleanup := func() {
		_ = serverTLS.Close()
		_ = clientTLS.Close()
	}
	return serverTLS, clientTLS, cleanup
}

func testMTLSClientConfig(t *testing.T) *tls.Config {
	t.Helper()
	cert, err := tls.X509KeyPair([]byte(mtlsCertPEM), []byte(mtlsKeyPEM))
	if err != nil {
		t.Fatalf("mtls client cert parse: %v", err)
	}
	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			return implantCrypto.RootOnlyVerifyCertificate(mtlsCACertPEM, rawCerts, verifiedChains)
		},
	}
}

func newMTLSConnection(t *testing.T, conn *tls.Conn) *implantTransports.Connection {
	t.Helper()
	send := make(chan *sliverpb.Envelope)
	recv := make(chan *sliverpb.Envelope)
	done := make(chan struct{})

	var once sync.Once
	var wg sync.WaitGroup

	start := func() error {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				case envelope, ok := <-send:
					if !ok {
						return
					}
					if err := implantMTLS.WriteEnvelope(conn, envelope); err != nil {
						return
					}
				case <-time.After(implantMTLS.PingInterval):
					_ = implantMTLS.WritePing(conn)
				}
			}
		}()

		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					envelope, err := implantMTLS.ReadEnvelope(conn)
					if err != nil {
						return
					}
					if envelope == nil {
						continue
					}
					select {
					case recv <- envelope:
					case <-done:
						return
					}
				}
			}
		}()
		return nil
	}

	stop := func() error {
		once.Do(func() {
			close(done)
			close(send)
			_ = conn.Close()
		})
		wg.Wait()
		close(recv)
		return nil
	}

	return &implantTransports.Connection{
		Send:  send,
		Recv:  recv,
		Start: start,
		Stop:  stop,
	}
}

func newHTTPTestServer(t *testing.T) *c2.SliverHTTPC2 {
	t.Helper()
	req := &clientpb.HTTPListenerReq{
		Host:   "sliver.test",
		Port:   443,
		Secure: true,
		Domain: "sliver.test",
	}
	server, err := c2.StartHTTPListener(req)
	if err != nil {
		t.Fatalf("start https listener: %v", err)
	}
	return server
}

type inMemoryHTTPDriver struct {
	handler    http.Handler
	jar        http.CookieJar
	baseURL    *url.URL
	remoteAddr string
}

func newInMemoryHTTPDriver(t *testing.T, handler http.Handler, baseURL *url.URL) *inMemoryHTTPDriver {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	return &inMemoryHTTPDriver{
		handler:    handler,
		jar:        jar,
		baseURL:    baseURL,
		remoteAddr: "127.0.0.1:12345",
	}
}

func (d *inMemoryHTTPDriver) Do(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme == "" {
		req.URL.Scheme = d.baseURL.Scheme
	}
	if req.URL.Host == "" {
		req.URL.Host = d.baseURL.Host
	}
	if req.Host == "" {
		req.Host = req.URL.Host
	}
	if req.RemoteAddr == "" {
		req.RemoteAddr = d.remoteAddr
	}
	if req.TLS == nil && req.URL.Scheme == "https" {
		req.TLS = &tls.ConnectionState{}
	}
	if req.RequestURI == "" {
		req.RequestURI = req.URL.RequestURI()
	}

	for _, cookie := range d.jar.Cookies(req.URL) {
		req.AddCookie(cookie)
	}

	rr := httptest.NewRecorder()
	d.handler.ServeHTTP(rr, req)
	resp := rr.Result()
	d.jar.SetCookies(req.URL, resp.Cookies())
	return resp, nil
}

func newHTTPConnection(t *testing.T, handler http.Handler, origin string) *implantTransports.Connection {
	t.Helper()
	baseURL, err := url.Parse(origin)
	if err != nil {
		t.Fatalf("parse origin: %v", err)
	}

	driver := newInMemoryHTTPDriver(t, handler, baseURL)
	opts := implantHTTP.ParseHTTPOptions(baseURL)
	opts.NetTimeout = 5 * time.Second
	opts.PollTimeout = 2 * time.Second
	opts.MaxErrors = 3

	client := implantHTTP.NewSliverHTTPClient(origin, driver, opts)
	client.PathPrefix = baseURL.Path
	if err := client.SessionInit(); err != nil {
		t.Fatalf("http session init: %v", err)
	}

	send := make(chan *sliverpb.Envelope)
	recv := make(chan *sliverpb.Envelope)
	done := make(chan struct{})

	var once sync.Once
	var wg sync.WaitGroup

	start := func() error {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				case envelope, ok := <-send:
					if !ok {
						return
					}
					if err := client.WriteEnvelope(envelope); err != nil {
						return
					}
				}
			}
		}()

		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					envelope, err := client.ReadEnvelope()
					if err == io.EOF {
						return
					}
					if err != nil {
						return
					}
					if envelope == nil {
						continue
					}
					select {
					case recv <- envelope:
					case <-done:
						return
					}
				}
			}
		}()
		return nil
	}

	stop := func() error {
		once.Do(func() {
			close(done)
			close(send)
			_ = client.CloseSession()
		})
		wg.Wait()
		close(recv)
		return nil
	}

	return &implantTransports.Connection{
		Send:  send,
		Recv:  recv,
		Start: start,
		Stop:  stop,
	}
}

func newGRPCClient(t *testing.T) (rpcpb.SliverRPCClient, func()) {
	t.Helper()
	grpcServer, ln, err := serverTransport.LocalListener()
	if err != nil {
		t.Fatalf("grpc listener: %v", err)
	}

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return ln.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		grpcServer.Stop()
		_ = ln.Close()
		t.Fatalf("grpc dial: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		grpcServer.Stop()
		_ = ln.Close()
	}

	return rpcpb.NewSliverRPCClient(conn), cleanup
}

func startImplantSession(t *testing.T, conn *implantTransports.Connection, register *sliverpb.Register) func() {
	t.Helper()
	if err := conn.Start(); err != nil {
		t.Fatalf("start implant connection: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runImplantSession(ctx, conn, register)
	}()

	return func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("implant session error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("implant session did not stop")
		}
		_ = conn.Stop()
	}
}

func runImplantSession(ctx context.Context, conn *implantTransports.Connection, register *sliverpb.Register) error {
	if conn == nil {
		return fmt.Errorf("nil implant connection")
	}

	regData, err := proto.Marshal(register)
	if err != nil {
		return err
	}
	conn.Send <- &sliverpb.Envelope{Type: sliverpb.MsgRegister, Data: regData}

	sysHandlers := implantHandlers.GetSystemHandlers()
	killHandlers := implantHandlers.GetKillHandlers()

	for {
		select {
		case <-ctx.Done():
			return nil
		case envelope, ok := <-conn.Recv:
			if !ok {
				return nil
			}
			if _, ok := killHandlers[envelope.Type]; ok {
				return nil
			}
			if handler, ok := sysHandlers[envelope.Type]; ok {
				envelopeID := envelope.ID
				dispatchHandler(handler, envelope.Data, func(data []byte, err error) {
					conn.Send <- &sliverpb.Envelope{ID: envelopeID, Data: data}
				})
				continue
			}
			conn.Send <- &sliverpb.Envelope{ID: envelope.ID, UnknownMessageType: true}
		}
	}
}

func newRegister(t *testing.T, name string, activeC2 string) *sliverpb.Register {
	t.Helper()
	host, _ := os.Hostname()
	id, _ := uuid.NewV4()
	return &sliverpb.Register{
		Name:              name,
		Hostname:          host,
		Uuid:              id.String(),
		Username:          "sliver-test",
		Uid:               "1000",
		Gid:               "1000",
		Os:                runtime.GOOS,
		Version:           "test",
		Arch:              runtime.GOARCH,
		Pid:               int32(os.Getpid()),
		Filename:          "sliver-test",
		ActiveC2:          activeC2,
		ReconnectInterval: int64(time.Second),
		Locale:            "en-US",
	}
}

func waitForSession(t *testing.T, name string) *core.Session {
	t.Helper()
	deadline := time.Now().Add(sessionWaitTimeout)
	for time.Now().Before(deadline) {
		for _, session := range core.Sessions.All() {
			if session.Name == name {
				return session
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("session %q not registered", name)
	return nil
}

func clearSessions() {
	for _, session := range core.Sessions.All() {
		core.Sessions.Remove(session.ID)
	}
}

func assertSessionInfo(t *testing.T, session *core.Session, register *sliverpb.Register, transport string) {
	t.Helper()
	if session.Name != register.Name {
		t.Fatalf("session name mismatch: got %q want %q", session.Name, register.Name)
	}
	if session.Hostname != register.Hostname {
		t.Fatalf("session hostname mismatch: got %q want %q", session.Hostname, register.Hostname)
	}
	if session.Username != register.Username {
		t.Fatalf("session username mismatch: got %q want %q", session.Username, register.Username)
	}
	if session.OS != register.Os {
		t.Fatalf("session os mismatch: got %q want %q", session.OS, register.Os)
	}
	if session.Arch != register.Arch {
		t.Fatalf("session arch mismatch: got %q want %q", session.Arch, register.Arch)
	}
	if session.ActiveC2 != register.ActiveC2 {
		t.Fatalf("session active c2 mismatch: got %q want %q", session.ActiveC2, register.ActiveC2)
	}
	if session.Connection == nil {
		t.Fatalf("session connection missing")
	}
	if session.Connection.Transport != transport {
		t.Fatalf("session transport mismatch: got %q want %q", session.Connection.Transport, transport)
	}
	if session.Connection.RemoteAddress == "" {
		t.Fatalf("session remote address not set")
	}
}

func assertLsRoundTrip(t *testing.T, rpcClient rpcpb.SliverRPCClient, sessionID string) {
	t.Helper()
	tmpDir := t.TempDir()
	fileName := "sliver-test.txt"
	filePath := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(filePath, []byte("hello"), 0600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	resp, err := rpcClient.Ls(ctx, &sliverpb.LsReq{
		Path: tmpDir,
		Request: &commonpb.Request{
			SessionID: sessionID,
			Timeout:   int64(requestTimeout),
		},
	})
	if err != nil {
		t.Fatalf("ls request failed: %v", err)
	}
	if !resp.Exists {
		t.Fatalf("ls response marked as non-existent: %v", resp.GetResponse())
	}
	for _, entry := range resp.Files {
		if entry.Name == fileName {
			return
		}
	}
	t.Fatalf("ls response missing %q", fileName)
}
