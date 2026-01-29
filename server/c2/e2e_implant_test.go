package c2

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
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	implantHandlers "github.com/bishopfox/sliver/implant/sliver/handlers"
	implantTransports "github.com/bishopfox/sliver/implant/sliver/transports"
	implantMTLS "github.com/bishopfox/sliver/implant/sliver/transports/mtls"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/proto"
)

const (
	sessionWaitTimeout = 10 * time.Second
	requestTimeout     = 10 * time.Second
)

func TestE2EMTLSInfoAndLs(t *testing.T) {
	clearSessions()
	implantMTLS.SetTestCertificates(mtlsCACertPEM, mtlsCertPEM, mtlsKeyPEM)

	addr, cleanupServer := startMTLSListener(t)
	defer cleanupServer()

	register := newRegister(t, fmt.Sprintf("e2e-mtls-%d", time.Now().UnixNano()), "mtls://"+addr)
	stopImplant := startImplantSession(t, "mtls://"+addr, register)
	defer stopImplant()

	session := waitForSession(t, register.Name)
	defer session.Connection.Cleanup()

	assertSessionInfo(t, session, register, consts.MtlsStr)
	assertLsRoundTrip(t, session)
}

func TestE2EHTTPSInfoAndLs(t *testing.T) {
	clearSessions()

	addr, cleanupServer := startHTTPSServer(t)
	defer cleanupServer()

	register := newRegister(t, fmt.Sprintf("e2e-https-%d", time.Now().UnixNano()), "https://"+addr)
	stopImplant := startImplantSession(t, "https://"+addr, register)
	defer stopImplant()

	session := waitForSession(t, register.Name)
	defer session.Connection.Cleanup()

	assertSessionInfo(t, session, register, "http(s)")
	assertLsRoundTrip(t, session)
}

func startMTLSListener(t *testing.T) (string, func()) {
	t.Helper()
	ln, err := StartMutualTLSListener("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start mtls listener: %v", err)
	}
	addr := ln.Addr().String()
	return addr, func() {
		ln.Close()
	}
}

func startHTTPSServer(t *testing.T) (string, func()) {
	t.Helper()
	req := &clientpb.HTTPListenerReq{
		Host:   "127.0.0.1",
		Port:   0,
		Secure: true,
	}
	server, err := StartHTTPListener(req)
	if err != nil {
		t.Fatalf("start https listener: %v", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen https: %v", err)
	}

	tlsConfig := server.HTTPServer.TLSConfig
	if tlsConfig == nil {
		t.Fatalf("https tls config missing")
	}
	cert, err := tls.X509KeyPair(req.Cert, req.Key)
	if err != nil {
		t.Fatalf("https cert parse: %v", err)
	}
	tlsConfig.Certificates = []tls.Certificate{cert}
	tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, tlsConfig)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.HTTPServer.Serve(tlsListener)
	}()

	addr := ln.Addr().String()
	return addr, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.HTTPServer.Shutdown(ctx)
		ln.Close()
		select {
		case err := <-errCh:
			if err != nil && err != http.ErrServerClosed {
				t.Fatalf("https server error: %v", err)
			}
		default:
		}
	}
}

func startImplantSession(t *testing.T, c2URL string, register *sliverpb.Register) func() {
	t.Helper()
	conn := nextConnection(t, c2URL)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runImplantSession(ctx, conn, register)
	}()

	return func() {
		cancel()
		_ = conn.Stop()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("implant session did not stop")
		}
	}
}

func nextConnection(t *testing.T, c2URL string) *implantTransports.Connection {
	t.Helper()
	abort := make(chan struct{})
	connections := implantTransports.StartConnectionLoop(abort, c2URL)
	var conn *implantTransports.Connection
	select {
	case conn = <-connections:
	case <-time.After(5 * time.Second):
		close(abort)
		t.Fatalf("timed out waiting for implant connection")
	}
	close(abort)
	if conn == nil {
		t.Fatalf("nil implant connection")
	}
	return conn
}

func runImplantSession(ctx context.Context, conn *implantTransports.Connection, register *sliverpb.Register) error {
	if conn == nil {
		return fmt.Errorf("nil implant connection")
	}
	if err := conn.Start(); err != nil {
		return err
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
				if runtime.GOOS == "windows" {
					go implantHandlers.WrapperHandler(handler, envelope.Data, func(data []byte, err error) {
						conn.Send <- &sliverpb.Envelope{ID: envelopeID, Data: data}
					})
				} else {
					go handler(envelope.Data, func(data []byte, err error) {
						conn.Send <- &sliverpb.Envelope{ID: envelopeID, Data: data}
					})
				}
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

func assertLsRoundTrip(t *testing.T, session *core.Session) {
	t.Helper()
	tmpDir := t.TempDir()
	fileName := "sliver-test.txt"
	filePath := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(filePath, []byte("hello"), 0600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	req := &sliverpb.LsReq{Path: tmpDir}
	reqData, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("encode ls request: %v", err)
	}
	respData, err := session.Request(sliverpb.MsgLsReq, requestTimeout, reqData)
	if err != nil {
		t.Fatalf("ls request failed: %v", err)
	}
	resp := &sliverpb.Ls{}
	if err := proto.Unmarshal(respData, resp); err != nil {
		t.Fatalf("decode ls response: %v", err)
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
