package core

import (
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

type tunnelGenerationTestStream struct {
	rpcpb.SliverRPC_TunnelDataClient

	sent        chan *sliverpb.TunnelData
	dataEntered chan struct{}
	releaseData chan struct{}
	dataErr     error
	enterOnce   sync.Once
}

func (stream *tunnelGenerationTestStream) Send(frame *sliverpb.TunnelData) error {
	if len(frame.GetData()) > 0 && stream.dataEntered != nil {
		stream.enterOnce.Do(func() { close(stream.dataEntered) })
		<-stream.releaseData
		if stream.dataErr != nil {
			return stream.dataErr
		}
	}
	stream.sent <- frame
	return nil
}

func TestTunnelsCloseIfDoesNotCloseReplacementGeneration(t *testing.T) {
	tunnels := newTunnelGenerationTestStore()
	stream := &tunnelGenerationTestStream{sent: make(chan *sliverpb.TunnelData, 2)}
	tunnels.SetStream(stream)

	old := tunnels.Start(41, "old-session")
	waitForTunnelGenerationFrame(t, stream.sent)
	replacement := tunnels.Start(41, "new-session")
	waitForTunnelGenerationFrame(t, stream.sent)

	select {
	case <-old.Done():
	default:
		t.Fatal("replaced tunnel generation remained open")
	}
	if current := tunnels.Get(41); current != replacement {
		t.Fatal("replacement tunnel generation was not published")
	}
	if tunnels.CloseIf(old) {
		t.Fatal("CloseIf reported closing a superseded tunnel generation")
	}
	select {
	case <-replacement.Done():
		t.Fatal("stale CloseIf closed the replacement tunnel generation")
	default:
	}
	if !tunnels.CloseIf(replacement) {
		t.Fatal("CloseIf did not close the current tunnel generation")
	}
}

func TestTunnelSendFailureCannotClosePublishedReplacement(t *testing.T) {
	tunnels := newTunnelGenerationTestStore()
	sendErr := errors.New("test old-generation send failure")
	stream := &tunnelGenerationTestStream{
		sent:        make(chan *sliverpb.TunnelData, 2),
		dataEntered: make(chan struct{}),
		releaseData: make(chan struct{}),
		dataErr:     sendErr,
	}
	tunnels.SetStream(stream)

	old := tunnels.Start(42, "old-session")
	waitForTunnelGenerationFrame(t, stream.sent)
	type writeResult struct {
		n   int
		err error
	}
	writeDone := make(chan writeResult, 1)
	go func() {
		n, err := old.Write([]byte("old generation"))
		writeDone <- writeResult{n: n, err: err}
	}()
	waitForTunnelGenerationSignal(t, stream.dataEntered, "old-generation stream send")
	select {
	case result := <-writeDone:
		t.Fatalf("Write completed before stream Send: (%d, %v)", result.n, result.err)
	case <-time.After(50 * time.Millisecond):
	}

	replacementDone := make(chan *TunnelIO, 1)
	go func() {
		replacementDone <- tunnels.Start(42, "new-session")
	}()
	select {
	case replacement := <-replacementDone:
		t.Fatalf("replacement %p was published before the old send completed", replacement)
	case <-time.After(50 * time.Millisecond):
	}
	if current := tunnels.Get(42); current != old {
		t.Fatal("old generation stopped being current while its stream send was in flight")
	}

	close(stream.releaseData)
	select {
	case result := <-writeDone:
		if result.n != 0 || result.err == nil {
			t.Fatalf("failed stream Write = (%d, %v), want (0, error)", result.n, result.err)
		}
	case <-time.After(time.Second):
		t.Fatal("Write did not return after stream Send failed")
	}
	var replacement *TunnelIO
	select {
	case replacement = <-replacementDone:
	case <-time.After(time.Second):
		t.Fatal("replacement Start did not return after the old send completed")
	}
	waitForTunnelGenerationFrame(t, stream.sent)

	if current := tunnels.Get(42); current != replacement {
		t.Fatal("old send failure removed the published replacement generation")
	}
	select {
	case <-replacement.Done():
		t.Fatal("old send failure closed the published replacement generation")
	default:
	}
	tunnels.CloseIf(replacement)
}

func TestTunnelWriteWaitsForSuccessfulExactGenerationStreamSend(t *testing.T) {
	tunnels := newTunnelGenerationTestStore()
	stream := &tunnelGenerationTestStream{
		sent:        make(chan *sliverpb.TunnelData, 2),
		dataEntered: make(chan struct{}),
		releaseData: make(chan struct{}),
	}
	tunnels.SetStream(stream)

	tunnel := tunnels.Start(43, "final-write-session")
	waitForTunnelGenerationFrame(t, stream.sent)
	payload := []byte("final payload")
	type writeResult struct {
		n   int
		err error
	}
	writeDone := make(chan writeResult, 1)
	go func() {
		n, err := tunnel.Write(payload)
		writeDone <- writeResult{n: n, err: err}
	}()
	waitForTunnelGenerationSignal(t, stream.dataEntered, "final stream send")
	select {
	case result := <-writeDone:
		t.Fatalf("Write completed before stream Send: (%d, %v)", result.n, result.err)
	case <-time.After(50 * time.Millisecond):
	}

	close(stream.releaseData)
	select {
	case result := <-writeDone:
		if result.n != len(payload) || result.err != nil {
			t.Fatalf("successful stream Write = (%d, %v), want (%d, nil)", result.n, result.err, len(payload))
		}
	case <-time.After(time.Second):
		t.Fatal("Write did not complete after successful stream Send")
	}
	select {
	case frame := <-stream.sent:
		if string(frame.GetData()) != string(payload) {
			t.Fatalf("stream payload = %q, want %q", frame.GetData(), payload)
		}
	case <-time.After(time.Second):
		t.Fatal("successful Write returned before the payload was recorded")
	}
	if !tunnels.CloseIf(tunnel) {
		t.Fatal("failed to close final-write tunnel")
	}
}

func TestTunnelCloseWakesWriteWaitingForStreamSend(t *testing.T) {
	tunnels := newTunnelGenerationTestStore()
	stream := &tunnelGenerationTestStream{
		sent:        make(chan *sliverpb.TunnelData, 2),
		dataEntered: make(chan struct{}),
		releaseData: make(chan struct{}),
	}
	tunnels.SetStream(stream)

	tunnel := tunnels.Start(44, "closing-write-session")
	waitForTunnelGenerationFrame(t, stream.sent)
	type writeResult struct {
		n   int
		err error
	}
	writeDone := make(chan writeResult, 1)
	go func() {
		n, err := tunnel.Write([]byte("blocked payload"))
		writeDone <- writeResult{n: n, err: err}
	}()
	waitForTunnelGenerationSignal(t, stream.dataEntered, "blocked stream send")
	if !tunnels.CloseIf(tunnel) {
		t.Fatal("failed to close tunnel with in-flight Write")
	}
	select {
	case result := <-writeDone:
		if result.n != 0 || !errors.Is(result.err, io.EOF) {
			t.Fatalf("Write after concurrent close = (%d, %v), want (0, EOF)", result.n, result.err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close did not wake Write waiting for stream Send")
	}
	close(stream.releaseData)
}

func newTunnelGenerationTestStore() *tunnels {
	tunnelMap := map[uint64]*TunnelIO{}
	return &tunnels{
		tunnels:     &tunnelMap,
		mutex:       &sync.RWMutex{},
		streamMutex: &sync.Mutex{},
	}
}

func waitForTunnelGenerationFrame(t *testing.T, frames <-chan *sliverpb.TunnelData) {
	t.Helper()
	select {
	case <-frames:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for tunnel stream frame")
	}
}

func waitForTunnelGenerationSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}
