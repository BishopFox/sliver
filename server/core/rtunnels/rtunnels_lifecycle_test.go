package rtunnels

import (
	"bytes"
	"errors"
	"io"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRTunnelCloseIsIdempotentAndNilSafe(t *testing.T) {
	reader := &countReadCloser{}
	writer := &countWriteCloser{}
	tunnel := NewAuthorizedRTunnel(91001, "session", AuthorizationID("authorization"), writer, reader, nil)

	tunnel.Close()
	tunnel.Close()
	assert.Equal(t, int32(1), reader.closes.Load())
	assert.Equal(t, int32(1), writer.closes.Load())
	assert.Equal(t, AuthorizationID("authorization"), tunnel.AuthorizationID())
	select {
	case <-tunnel.Done():
	default:
		t.Fatal("Close() did not signal tunnel completion")
	}
}

func TestRTunnelProcessInboundIsOrderedBoundedAndGenerationOwned(t *testing.T) {
	first := NewRTunnel(92001, "session", &countWriteCloser{})
	second := NewRTunnel(92001, "session", &countWriteCloser{})

	var output bytes.Buffer
	write := func(data []byte) error {
		_, err := output.Write(data)
		return err
	}
	pending, err := first.ProcessInbound(1, []byte("second"), write)
	must.NoError(t, err)
	assert.Equal(t, 1, pending)
	pending, err = first.ProcessInbound(0, []byte("first-"), write)
	must.NoError(t, err)
	assert.Zero(t, pending)
	assert.Equal(t, "first-second", output.String())

	first.Close()
	var secondOutput bytes.Buffer
	pending, err = second.ProcessInbound(0, []byte("new-generation"), func(data []byte) error {
		_, writeErr := secondOutput.Write(data)
		return writeErr
	})
	must.NoError(t, err)
	assert.Zero(t, pending)
	assert.Equal(t, "new-generation", secondOutput.String())
}

func TestRTunnelProcessInboundRejectsResourceExhaustion(t *testing.T) {
	tests := []struct {
		name     string
		sequence uint64
		data     []byte
		want     error
	}{
		{name: "oversized frame", data: make([]byte, maxReverseTunnelFrameBytes+1), want: ErrReverseTunnelFrameTooLarge},
		{name: "sequence window", sequence: maxReverseTunnelPendingFrames, data: []byte("x"), want: ErrReverseTunnelWindow},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tunnel := NewRTunnel(92002, "session", &countWriteCloser{})
			_, err := tunnel.ProcessInbound(test.sequence, test.data, func([]byte) error { return nil })
			if !errors.Is(err, test.want) {
				t.Fatalf("ProcessInbound() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestTryAddRTunnelAndConditionalRemovalRejectStaleCleanup(t *testing.T) {
	const tunnelID = uint64(91002)

	first := NewRTunnel(tunnelID, "first", &countWriteCloser{})
	duplicate := NewRTunnel(tunnelID, "second", &countWriteCloser{})
	t.Cleanup(func() {
		RemoveRTunnelIf(tunnelID, first)
	})
	must.True(t, TryAddRTunnel(first))
	assert.False(t, TryAddRTunnel(duplicate))
	assert.Same(t, first, GetRTunnel(tunnelID))
	assert.False(t, RemoveRTunnelIf(tunnelID, duplicate))
	assert.Same(t, first, GetRTunnel(tunnelID))
	assert.True(t, RemoveRTunnelIf(tunnelID, first))
	assert.Nil(t, GetRTunnel(tunnelID))
	assert.False(t, RemoveRTunnelIf(tunnelID, first))
	assert.False(t, TryAddRTunnel(nil))
}

func TestCloseAuthorizationAndCloseSessionAreScopedAndIdempotent(t *testing.T) {
	const (
		matchingID  = uint64(91003)
		otherAuthID = uint64(91004)
		otherSID    = uint64(91005)
	)
	t.Cleanup(func() {
		CloseSession("session")
		CloseSession("other-session")
	})

	matchingWriter := &countWriteCloser{}
	otherAuthWriter := &countWriteCloser{}
	otherSessionWriter := &countWriteCloser{}
	matching := NewAuthorizedRTunnel(matchingID, "session", AuthorizationID("auth-a"), matchingWriter)
	otherAuth := NewAuthorizedRTunnel(otherAuthID, "session", AuthorizationID("auth-b"), otherAuthWriter)
	otherSession := NewAuthorizedRTunnel(otherSID, "other-session", AuthorizationID("auth-a"), otherSessionWriter)
	must.True(t, TryAddRTunnel(matching))
	must.True(t, TryAddRTunnel(otherAuth))
	must.True(t, TryAddRTunnel(otherSession))

	assert.Equal(t, 1, CloseAuthorization("session", AuthorizationID("auth-a")))
	assert.Equal(t, int32(1), matchingWriter.closes.Load())
	assert.Zero(t, otherAuthWriter.closes.Load())
	assert.Zero(t, otherSessionWriter.closes.Load())
	assert.Equal(t, 0, CloseAuthorization("session", AuthorizationID("auth-a")))
	assert.Equal(t, 0, CloseAuthorization("session", ""))

	assert.Equal(t, 1, CloseSession("session"))
	assert.Equal(t, int32(1), otherAuthWriter.closes.Load())
	assert.Zero(t, otherSessionWriter.closes.Load())
	assert.Equal(t, 0, CloseSession("session"))
	assert.Equal(t, 1, CloseSession("other-session"))
	assert.Equal(t, int32(1), otherSessionWriter.closes.Load())
}

type countReadCloser struct {
	closes atomic.Int32
}

func (*countReadCloser) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (closer *countReadCloser) Close() error {
	closer.closes.Add(1)
	return nil
}

type countWriteCloser struct {
	closes atomic.Int32
}

func (*countWriteCloser) Write(buffer []byte) (int, error) {
	return len(buffer), nil
}

func (closer *countWriteCloser) Close() error {
	closer.closes.Add(1)
	return nil
}
