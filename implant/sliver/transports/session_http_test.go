package transports

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

type controlledHTTPEnvelopeWriter struct {
	started      chan int64
	completed    chan int64
	firstRelease <-chan struct{}
	fail         error
}

func (writer *controlledHTTPEnvelopeWriter) WriteEnvelope(envelope *sliverpb.Envelope) error {
	writer.started <- envelope.ID
	if envelope.ID == 0 && writer.firstRelease != nil {
		<-writer.firstRelease
	}
	writer.completed <- envelope.ID
	return writer.fail
}

// The scenario synchronizes a full request window, oldest-request retirement,
// and scheduler shutdown in one ordered-transport lifecycle.
//
//nolint:gocyclo
func TestSendHTTPEnvelopesRetiresOldestBeforeAdvancingWindow(t *testing.T) {
	connection := &Connection{}
	t.Cleanup(connection.Cleanup)
	send := make(chan *sliverpb.Envelope, httpEnvelopeWriteWindow+1)
	for ordinal := 0; ordinal <= httpEnvelopeWriteWindow; ordinal++ {
		send <- &sliverpb.Envelope{ID: int64(ordinal)}
	}
	close(send)

	firstRelease := make(chan struct{})
	var releaseOnce sync.Once
	releaseFirst := func() { releaseOnce.Do(func() { close(firstRelease) }) }
	t.Cleanup(releaseFirst)
	writer := &controlledHTTPEnvelopeWriter{
		started:      make(chan int64, httpEnvelopeWriteWindow+1),
		completed:    make(chan int64, httpEnvelopeWriteWindow+1),
		firstRelease: firstRelease,
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		sendHTTPEnvelopes(connection, send, writer, httpEnvelopeWriteWindow)
	}()

	started := map[int64]struct{}{}
	for len(started) < httpEnvelopeWriteWindow {
		select {
		case ordinal := <-writer.started:
			if ordinal >= int64(httpEnvelopeWriteWindow) {
				t.Fatalf("HTTP envelope ordinal %d launched before oldest retirement", ordinal)
			}
			started[ordinal] = struct{}{}
		case <-time.After(time.Second):
			t.Fatalf("only %d HTTP envelope writes started", len(started))
		}
	}
	select {
	case ordinal := <-writer.started:
		t.Fatalf("HTTP envelope ordinal %d launched while ordinal 0 was still active", ordinal)
	case <-time.After(50 * time.Millisecond):
	}

	releaseFirst()
	select {
	case ordinal := <-writer.started:
		if ordinal != int64(httpEnvelopeWriteWindow) {
			t.Fatalf("first write after retirement = %d, want %d", ordinal, httpEnvelopeWriteWindow)
		}
	case <-time.After(time.Second):
		t.Fatalf("HTTP envelope ordinal %d did not launch after oldest retirement", httpEnvelopeWriteWindow)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("HTTP envelope scheduler did not stop after its source closed")
	}
	for completed := 0; completed <= httpEnvelopeWriteWindow; completed++ {
		select {
		case <-writer.completed:
		case <-time.After(time.Second):
			t.Fatalf("only %d HTTP envelope writes completed", completed)
		}
	}
}

func TestSendHTTPEnvelopesWriteFailureClosesConnection(t *testing.T) {
	connection := &Connection{}
	send := make(chan *sliverpb.Envelope, 1)
	send <- &sliverpb.Envelope{ID: 1}
	writer := &controlledHTTPEnvelopeWriter{
		started:   make(chan int64, 1),
		completed: make(chan int64, 1),
		fail:      errors.New("write failed"),
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		sendHTTPEnvelopes(connection, send, writer, httpEnvelopeWriteWindow)
	}()

	select {
	case <-connection.Done():
	case <-time.After(time.Second):
		t.Fatal("HTTP envelope write failure did not close the connection")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("HTTP envelope scheduler survived connection cleanup")
	}
}
