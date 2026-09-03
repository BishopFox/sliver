package tunnel_handlers

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

type recordingWriteCloser struct {
	bytes.Buffer
	closed bool
}

func (writer *recordingWriteCloser) Close() error {
	writer.closed = true
	return nil
}

func TestReverseTunnelIgnoresLegacyResendControlFrame(t *testing.T) {
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 1)}
	writer := &recordingWriteCloser{}
	tunnel := transports.NewReverseTunnel(0x5101, writer)
	if !connection.AddTunnel(tunnel) {
		t.Fatal("failed to add reverse tunnel")
	}
	t.Cleanup(func() { connection.CloseTunnelRemote(tunnel) })

	payload, err := proto.Marshal(&sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Resend:   true,
		Data:     []byte("must-not-be-relayed"),
	})
	if err != nil {
		t.Fatalf("marshal resend frame: %v", err)
	}
	TunnelDataHandler(&sliverpb.Envelope{Type: sliverpb.MsgTunnelData, Data: payload}, connection)

	if got := writer.String(); got != "" {
		t.Fatalf("resend control payload reached reverse destination: %q", got)
	}
	if got := tunnel.ReadSequence(); got != 0 {
		t.Fatalf("resend control advanced reverse read sequence to %d", got)
	}
	if got := connection.Tunnel(tunnel.ID); got != tunnel {
		t.Fatalf("resend control detached reverse tunnel: got %p, want %p", got, tunnel)
	}
	select {
	case envelope := <-connection.Send:
		t.Fatalf("resend control emitted unexpected envelope type %d", envelope.Type)
	default:
	}
}

type closeUnblocksWriteCloser struct {
	started   chan struct{}
	released  chan struct{}
	startOnce sync.Once
	closeOnce sync.Once
}

func newCloseUnblocksWriteCloser() *closeUnblocksWriteCloser {
	return &closeUnblocksWriteCloser{
		started:  make(chan struct{}),
		released: make(chan struct{}),
	}
}

func (writer *closeUnblocksWriteCloser) Write([]byte) (int, error) {
	writer.startOnce.Do(func() { close(writer.started) })
	<-writer.released
	return 0, io.ErrClosedPipe
}

func (writer *closeUnblocksWriteCloser) Close() error {
	writer.closeOnce.Do(func() { close(writer.released) })
	return nil
}

func TestTunnelCloseUnblocksInboundWriter(t *testing.T) {
	tests := []struct {
		name          string
		sequence      uint64
		closeTimeout  time.Duration
		wantC2Cleanup bool
	}{
		{name: "legacy immediate close", sequence: 0, closeTimeout: time.Second},
		{name: "sequenced close deadline", sequence: 1, closeTimeout: 20 * time.Millisecond, wantC2Cleanup: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 1)}
			writer := newCloseUnblocksWriteCloser()
			tunnel := transports.NewTunnel(0x5201, writer)
			if !connection.AddTunnel(tunnel) {
				t.Fatal("failed to add tunnel")
			}

			inboundDone := make(chan error, 1)
			go func() {
				_, err := tunnel.ProcessInbound(0, []byte("blocked"), func(data []byte) error {
					_, writeErr := writer.Write(data)
					return writeErr
				})
				inboundDone <- err
			}()
			select {
			case <-writer.started:
			case <-time.After(time.Second):
				t.Fatal("inbound writer did not block")
			}

			closeDone := make(chan struct{})
			go func() {
				handleTunnelClose(&sliverpb.TunnelData{
					Closed:   true,
					TunnelID: tunnel.ID,
					Sequence: test.sequence,
				}, connection, test.closeTimeout)
				close(closeDone)
			}()
			select {
			case <-closeDone:
			case <-time.After(time.Second):
				t.Fatal("tunnel close remained blocked behind inbound writer")
			}
			select {
			case err := <-inboundDone:
				if !errors.Is(err, io.ErrClosedPipe) {
					t.Fatalf("inbound error = %v, want %v", err, io.ErrClosedPipe)
				}
			case <-time.After(time.Second):
				t.Fatal("closing tunnel did not unblock inbound writer")
			}
			if active := connection.Tunnel(tunnel.ID); active != nil {
				t.Fatalf("closed tunnel remained published: %p", active)
			}
			select {
			case <-connection.Done():
				if !test.wantC2Cleanup {
					t.Fatal("legacy close unexpectedly tore down the C2 connection")
				}
			default:
				if test.wantC2Cleanup {
					t.Fatal("incomplete sequenced close did not fail the C2 connection closed")
				}
			}
		})
	}
}
