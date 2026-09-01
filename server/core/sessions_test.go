package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func TestSessionToProtobufNilConnection(t *testing.T) {
	s := &Session{Name: "test-canary", Capabilities: 42}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ToProtobuf panicked with nil Connection: %v", r)
		}
	}()

	pb := s.ToProtobuf()
	if pb.Name != "test-canary" {
		t.Fatalf("expected Name=%q, got %q", "test-canary", pb.Name)
	}
	if pb.Transport != "" {
		t.Fatalf("expected empty Transport, got %q", pb.Transport)
	}
	if pb.Capabilities != s.Capabilities {
		t.Fatalf("expected Capabilities=%d, got %d", s.Capabilities, pb.Capabilities)
	}
}

func TestSessionRequestDisconnectCleansResponseWaiter(t *testing.T) {
	connection := NewImplantConnection("test", "test")
	session := &Session{Connection: connection}
	result := make(chan error, 1)
	go func() {
		_, err := session.Request(sliverpb.MsgPing, time.Hour, nil)
		result <- err
	}()

	var request *sliverpb.Envelope
	select {
	case request = <-connection.Send:
	case <-time.After(time.Second):
		t.Fatal("request was not queued")
	}
	connection.RespMutex.RLock()
	_, waiting := connection.Resp[request.ID]
	connection.RespMutex.RUnlock()
	if !waiting {
		t.Fatal("request response waiter was not registered")
	}

	connection.Close()
	select {
	case err := <-result:
		if !errors.Is(err, ErrImplantConnectionClosed) {
			t.Fatalf("Request error = %v, want %v", err, ErrImplantConnectionClosed)
		}
	case <-time.After(time.Second):
		t.Fatal("Request remained blocked after connection close")
	}

	connection.RespMutex.RLock()
	waiters := len(connection.Resp)
	connection.RespMutex.RUnlock()
	if waiters != 0 {
		t.Fatalf("response waiter count = %d, want 0", waiters)
	}
}

func TestSessionRequestLateResponseCannotStrandTransportReader(t *testing.T) {
	connection := NewImplantConnection("test", "test")
	session := &Session{Connection: connection}
	result := make(chan error, 1)
	go func() {
		_, err := session.Request(sliverpb.MsgPing, 25*time.Millisecond, nil)
		result <- err
	}()

	var request *sliverpb.Envelope
	select {
	case request = <-connection.Send:
	case <-time.After(time.Second):
		t.Fatal("request was not queued")
	}
	connection.RespMutex.RLock()
	response := connection.Resp[request.ID]
	connection.RespMutex.RUnlock()
	if response == nil {
		t.Fatal("request response waiter was not registered")
	}

	select {
	case err := <-result:
		if !errors.Is(err, ErrImplantTimeout) {
			t.Fatalf("Request error = %v, want %v", err, ErrImplantTimeout)
		}
	case <-time.After(time.Second):
		t.Fatal("Request did not time out")
	}

	// A transport may have looked up the waiter immediately before the request
	// deadline removed it from the map. The single-message buffer guarantees
	// that this stale delivery still cannot block that transport reader.
	select {
	case response <- &sliverpb.Envelope{ID: request.ID}:
	default:
		t.Fatal("late response blocked on a stale waiter")
	}
	if connection.DeliverResponse(&sliverpb.Envelope{ID: request.ID}) {
		t.Fatal("late response was delivered after waiter cleanup")
	}
}

func TestSessionRequestContextPreCanceledDoesNotInstallWaiter(t *testing.T) {
	connection := NewImplantConnection("test", "test")
	session := &Session{Connection: connection}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := session.RequestContext(ctx, sliverpb.MsgPing, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RequestContext error = %v, want %v", err, context.Canceled)
	}
	connection.RespMutex.RLock()
	waiters := len(connection.Resp)
	connection.RespMutex.RUnlock()
	if waiters != 0 {
		t.Fatalf("response waiter count = %d, want 0", waiters)
	}
	select {
	case envelope := <-connection.Send:
		t.Fatalf("pre-canceled request queued envelope: %#v", envelope)
	default:
	}
}

func TestSessionRequestContextSharesSendAndResponseDeadline(t *testing.T) {
	connection := NewImplantConnection("test", "test")
	session := &Session{Connection: connection}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	started := time.Now()
	result := make(chan error, 1)
	go func() {
		_, err := session.RequestContext(ctx, sliverpb.MsgPing, nil)
		result <- err
	}()

	// Consume most of the request budget before allowing the outbound envelope
	// to queue. The response wait must use only the remainder of that same
	// budget, rather than starting a second timeout window.
	time.Sleep(600 * time.Millisecond)
	select {
	case <-connection.Send:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("request was not queued within its remaining context budget")
	}
	select {
	case err := <-result:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("RequestContext error = %v, want %v", err, context.DeadlineExceeded)
		}
	case <-time.After(750 * time.Millisecond):
		t.Fatal("response wait started a second timeout window")
	}
	if elapsed := time.Since(started); elapsed > 1350*time.Millisecond {
		t.Fatalf("shared deadline elapsed = %v, want at most 1.35s", elapsed)
	}
	connection.RespMutex.RLock()
	waiters := len(connection.Resp)
	connection.RespMutex.RUnlock()
	if waiters != 0 {
		t.Fatalf("response waiter count = %d, want 0", waiters)
	}
}
