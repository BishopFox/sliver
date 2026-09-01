package rpc

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/protobuf/proto"
)

type tunnelStreamRecv struct {
	data *sliverpb.TunnelData
	err  error
}

type testTunnelStream struct {
	rpcpb.SliverRPC_TunnelDataServer
	recv      chan tunnelStreamRecv
	sent      chan *sliverpb.TunnelData
	sendDelay time.Duration
	failAfter int32
	sendCalls atomic.Int32
	inSend    atomic.Int32
	overlap   atomic.Bool
}

func (s *testTunnelStream) Recv() (*sliverpb.TunnelData, error) {
	result := <-s.recv
	return result.data, result.err
}

func (s *testTunnelStream) Send(data *sliverpb.TunnelData) error {
	if s.inSend.Add(1) > 1 {
		s.overlap.Store(true)
	}
	defer s.inSend.Add(-1)
	if s.sendDelay > 0 {
		time.Sleep(s.sendDelay)
	}
	call := s.sendCalls.Add(1)
	if s.failAfter > 0 && call > s.failAfter {
		return errors.New("test stream send failure")
	}
	s.sent <- proto.Clone(data).(*sliverpb.TunnelData)
	return nil
}

func TestTunnelDataDropsUnknownFrameWithoutClosingStream(t *testing.T) {
	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 2),
		sent: make(chan *sliverpb.TunnelData, 1),
	}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: 0xdeadbeef, Data: []byte("late")}}
	stream.recv <- tunnelStreamRecv{err: io.EOF}

	if err := (&Server{}).TunnelData(stream); err != nil {
		t.Fatalf("TunnelData returned for stale frame: %v", err)
	}
}

func TestTunnelDataClosesOwnedTunnelsAndSerializesSends(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 2)
	drainImplantCloseEnvelopes(session, len(tunnels))
	stream := &testTunnelStream{
		recv:      make(chan tunnelStreamRecv, 3),
		sent:      make(chan *sliverpb.TunnelData, 16),
		sendDelay: 20 * time.Millisecond,
	}
	for _, tunnel := range tunnels {
		stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: tunnel.ID, SessionID: session.ID}}
	}

	handlerDone := make(chan error, 1)
	go func() { handlerDone <- (&Server{}).TunnelData(stream) }()
	for range tunnels {
		waitForTunnelStreamSend(t, stream.sent)
	}

	for index, tunnel := range tunnels {
		index := index
		tunnel := tunnel
		go tunnel.SendDataFromImplant(&sliverpb.TunnelData{
			TunnelID: tunnel.ID,
			Sequence: 0,
			Data:     []byte{byte(index + 1)},
		})
	}
	for range tunnels {
		waitForTunnelStreamSend(t, stream.sent)
	}
	if stream.overlap.Load() {
		t.Fatal("server called Send concurrently on one TunnelData stream")
	}

	stream.recv <- tunnelStreamRecv{err: io.EOF}
	select {
	case err := <-handlerDone:
		if err != nil {
			t.Fatalf("TunnelData EOF result = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("TunnelData did not return after EOF")
	}
	for _, tunnel := range tunnels {
		select {
		case <-tunnel.Done():
		case <-time.After(time.Second):
			t.Fatalf("tunnel %d did not close after its client stream ended", tunnel.ID)
		}
	}
}

func TestTunnelDataSendFailureClosesTunnel(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	drainImplantCloseEnvelopes(session, 1)
	tunnel := tunnels[0]
	stream := &testTunnelStream{
		recv:      make(chan tunnelStreamRecv, 2),
		sent:      make(chan *sliverpb.TunnelData, 4),
		failAfter: 1, // Bind acknowledgement succeeds; first data frame fails.
	}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: tunnel.ID, SessionID: session.ID}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- (&Server{}).TunnelData(stream) }()
	waitForTunnelStreamSend(t, stream.sent)

	go tunnel.SendDataFromImplant(&sliverpb.TunnelData{TunnelID: tunnel.ID, Sequence: 0, Data: []byte("output")})
	select {
	case <-tunnel.Done():
	case <-time.After(time.Second):
		t.Fatal("tunnel remained open after stream Send failed")
	}
	stream.recv <- tunnelStreamRecv{err: io.EOF}
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("TunnelData did not return after EOF")
	}
}

func TestForEachTunnelPayloadFrame(t *testing.T) {
	testCases := []int{
		0,
		core.MaxTunnelFrameBytes,
		core.MaxTunnelFrameBytes + 1,
		96 * 1024,
		2*core.MaxTunnelFrameBytes + 1,
	}
	for _, size := range testCases {
		t.Run(fmt.Sprintf("bytes-%d", size), func(t *testing.T) {
			payload := make([]byte, size)
			for index := range payload {
				payload[index] = byte(index)
			}

			frames := [][]byte{}
			if err := forEachTunnelPayloadFrame(payload, func(frame []byte) error {
				frames = append(frames, append([]byte(nil), frame...))
				return nil
			}); err != nil {
				t.Fatalf("segment payload: %v", err)
			}
			wantFrames := 1
			if size > 0 {
				wantFrames = 1 + (size-1)/core.MaxTunnelFrameBytes
			}
			if len(frames) != wantFrames {
				t.Fatalf("frame count = %d, want %d", len(frames), wantFrames)
			}
			for index, frame := range frames {
				if len(frame) > core.MaxTunnelFrameBytes {
					t.Fatalf("frame %d length = %d, limit %d", index, len(frame), core.MaxTunnelFrameBytes)
				}
			}
			if got := bytes.Join(frames, nil); !bytes.Equal(got, payload) {
				t.Fatal("segmented payload did not reassemble byte-for-byte")
			}
		})
	}

	stopErr := errors.New("stop iteration")
	frames := 0
	err := forEachTunnelPayloadFrame(make([]byte, 2*core.MaxTunnelFrameBytes+1), func([]byte) error {
		frames++
		return stopErr
	})
	if !errors.Is(err, stopErr) || frames != 1 {
		t.Fatalf("callback stop = (%v, %d frames), want (%v, 1 frame)", err, frames, stopErr)
	}
}

func TestTunnelDataSegmentsOversizedClientPayload(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	session.Connection.Send = make(chan *sliverpb.Envelope, 4)
	tunnel := tunnels[0]
	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 3),
		sent: make(chan *sliverpb.TunnelData, 2),
	}

	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: session.ID,
	}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- (&Server{}).TunnelData(stream) }()
	waitForTunnelStreamSend(t, stream.sent)

	payload := make([]byte, 96*1024)
	for index := range payload {
		payload[index] = byte(index)
	}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: session.ID,
		Data:      payload,
	}}

	reassembled := make([]byte, 0, len(payload))
	for wantSequence := uint64(0); wantSequence < 2; wantSequence++ {
		var envelope *sliverpb.Envelope
		select {
		case envelope = <-session.Connection.Send:
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for implant frame %d", wantSequence)
		}
		if envelope.Type != sliverpb.MsgTunnelData {
			t.Fatalf("envelope type = %d, want MsgTunnelData", envelope.Type)
		}
		frame := &sliverpb.TunnelData{}
		if err := proto.Unmarshal(envelope.Data, frame); err != nil {
			t.Fatalf("unmarshal frame %d: %v", wantSequence, err)
		}
		if frame.Sequence != wantSequence {
			t.Fatalf("frame sequence = %d, want %d", frame.Sequence, wantSequence)
		}
		if len(frame.Data) > core.MaxTunnelFrameBytes {
			t.Fatalf("frame %d length = %d, limit %d", wantSequence, len(frame.Data), core.MaxTunnelFrameBytes)
		}
		reassembled = append(reassembled, frame.Data...)
	}
	if !bytes.Equal(reassembled, payload) {
		t.Fatal("implant frames did not reassemble to the client payload")
	}

	stream.recv <- tunnelStreamRecv{err: io.EOF}
	select {
	case err := <-handlerDone:
		if err != nil {
			t.Fatalf("TunnelData EOF result = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("TunnelData did not return after EOF")
	}
}

func newTunnelStreamTestSession(t *testing.T, count int) (*core.Session, []*core.Tunnel) {
	t.Helper()
	session := core.NewSession(core.NewImplantConnection("mtls", "test"))
	core.Sessions.Add(session)
	t.Cleanup(func() { core.Sessions.Remove(session.ID) })
	tunnels := make([]*core.Tunnel, 0, count)
	for range count {
		tunnel, err := core.Tunnels.Create(session.ID)
		if err != nil {
			t.Fatalf("create tunnel: %v", err)
		}
		tunnels = append(tunnels, tunnel)
		t.Cleanup(func() { core.Tunnels.CloseIf(tunnel) })
	}
	return session, tunnels
}

func drainImplantCloseEnvelopes(session *core.Session, count int) {
	go func() {
		for range count {
			<-session.Connection.Send
		}
	}()
}

func waitForTunnelStreamSend(t *testing.T, sent <-chan *sliverpb.TunnelData) *sliverpb.TunnelData {
	t.Helper()
	select {
	case data := <-sent:
		return data
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for TunnelData Send")
		return nil
	}
}
