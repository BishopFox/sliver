package core

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

type testTunnelDataServer struct {
	rpcpb.SliverRPC_TunnelDataServer
}

func newTestSession() *Session {
	conn := NewImplantConnection("mtls", "test-conn")
	return NewSession(conn)
}

func TestTunnelsCreateUnknownSession(t *testing.T) {
	tunnel, err := Tunnels.Create("does-not-exist")
	if err == nil {
		t.Fatalf("expected error for unknown session, got tunnel %+v", tunnel)
	}
	if tunnel != nil {
		t.Fatalf("expected nil tunnel on error, got %+v", tunnel)
	}
}

func TestTunnelsCreateKnownSession(t *testing.T) {
	session := newTestSession()
	Sessions.Add(session)
	defer Sessions.Remove(session.ID)

	tunnel, err := Tunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tunnel == nil {
		t.Fatal("expected non-nil tunnel")
	}
	if tunnel.SessionID != session.ID {
		t.Fatalf("expected SessionID %q, got %q", session.ID, tunnel.SessionID)
	}
}

func TestTunnelsCloseUnboundTunnelDoesNotBlock(t *testing.T) {
	tunnel := NewTunnel(1, "test-session")
	tunnels := &tunnels{
		tunnels: map[uint64]*Tunnel{tunnel.ID: tunnel},
		mutex:   &sync.Mutex{},
	}

	result := make(chan error, 1)
	go func() {
		result <- tunnels.Close(tunnel.ID)
	}()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("unexpected close error: %v", err)
		}
	case <-time.After(time.Second):
		// Unblock the legacy in-band close send so a failing test does not
		// leave a goroutine holding the local tunnel registry mutex.
		go func() {
			<-tunnel.ToImplant
		}()
		select {
		case <-result:
		case <-time.After(time.Second):
		}
		t.Fatal("closing an unbound tunnel blocked")
	}

	if got := tunnels.Get(tunnel.ID); got != nil {
		t.Fatalf("closed tunnel remains registered: %+v", got)
	}
	select {
	case <-tunnel.Done():
	default:
		t.Fatal("closed tunnel did not signal completion")
	}
}

func TestTunnelSignalsClientBoundAfterAcknowledgement(t *testing.T) {
	tunnel := NewTunnel(1, "test-session")
	client := &testTunnelDataServer{}
	otherClient := &testTunnelDataServer{}

	if !tunnel.BindClient(client) {
		t.Fatal("first client failed to reserve tunnel")
	}
	if tunnel.BindClient(otherClient) {
		t.Fatal("second client replaced tunnel reservation")
	}
	select {
	case <-tunnel.ClientBound():
		t.Fatal("tunnel reported bound before acknowledgement")
	default:
	}
	if tunnel.MarkClientBound(otherClient) {
		t.Fatal("non-owning client acknowledged tunnel")
	}
	if !tunnel.MarkClientBound(client) {
		t.Fatal("owning client failed to acknowledge tunnel")
	}
	select {
	case <-tunnel.ClientBound():
	default:
		t.Fatal("tunnel did not report acknowledged client binding")
	}
}

func TestTunnelCloseUnblocksSenders(t *testing.T) {
	tunnel := NewTunnel(1, "test-session")

	fromResult := make(chan bool, 1)
	go func() {
		fromResult <- tunnel.SendDataFromImplant(&sliverpb.TunnelData{Sequence: 0, Data: []byte("blocked")})
	}()

	toResult := make(chan bool, 1)
	go func() {
		toResult <- tunnel.SendDataToImplant(nil)
	}()

	deadline := time.Now().Add(time.Second)
	blocked := false
	for time.Now().Before(deadline) {
		fromBlocked := len(tunnel.fromImplantAdmission) == 1
		toBlocked := !tunnel.toImplantQueue.TryLock()
		if !toBlocked {
			tunnel.toImplantQueue.Unlock()
		}
		if fromBlocked && toBlocked {
			blocked = true
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !blocked {
		tunnel.Close()
		t.Fatal("tunnel senders did not reach their blocked channel operations")
	}
	tunnel.Close()

	for name, result := range map[string]<-chan bool{
		"from implant": fromResult,
		"to implant":   toResult,
	} {
		select {
		case sent := <-result:
			if sent {
				t.Errorf("%s sender reported success after close", name)
			}
		case <-time.After(time.Second):
			t.Errorf("%s sender remained blocked after close", name)
		}
	}
}

func TestTunnelsCloseForClientOnlyClosesOwnedTunnels(t *testing.T) {
	client := &testTunnelDataServer{}
	otherClient := &testTunnelDataServer{}
	owned := NewTunnel(11, "session")
	other := NewTunnel(12, "session")
	if !owned.BindClient(client) || !other.BindClient(otherClient) {
		t.Fatal("bind test clients")
	}
	tunnels := &tunnels{
		tunnels: map[uint64]*Tunnel{owned.ID: owned, other.ID: other},
		mutex:   &sync.Mutex{},
	}

	tunnels.CloseForClient(client)
	if got := tunnels.Get(owned.ID); got != nil {
		t.Fatal("stream-owned tunnel remained registered")
	}
	select {
	case <-owned.Done():
	default:
		t.Fatal("stream-owned tunnel was not closed")
	}
	if got := tunnels.Get(other.ID); got != other {
		t.Fatal("unrelated client's tunnel was removed")
	}
	select {
	case <-other.Done():
		t.Fatal("unrelated client's tunnel was closed")
	default:
	}
}

func TestTunnelClientDataRefreshesCloseActivity(t *testing.T) {
	tunnel := NewTunnel(13, "session")
	before := tunnel.GetLastMessageTime()
	time.Sleep(time.Millisecond)
	received := make(chan []byte, 1)
	go func() { received <- <-tunnel.ToImplant }()
	payload := []byte("exit\n")
	if !tunnel.SendDataToImplant(payload) {
		t.Fatal("client data was not queued")
	}
	if got := <-received; string(got) != string(payload) {
		t.Fatalf("queued data = %q, want %q", got, payload)
	}
	if after := tunnel.GetLastMessageTime(); !after.After(before) {
		t.Fatalf("client data did not refresh close activity: before=%v after=%v", before, after)
	}
}

func TestTunnelTouchRefreshesCloseActivity(t *testing.T) {
	tunnel := NewTunnel(14, "session")
	tunnel.mutex.Lock()
	tunnel.lastDataMessageTime = time.Now().Add(-2 * delayBeforeClose)
	tunnel.mutex.Unlock()

	tunnel.Touch()
	if age := time.Since(tunnel.GetLastMessageTime()); age < 0 || age >= delayBeforeClose {
		t.Fatalf("Touch did not establish a fresh close grace period: age=%v", age)
	}
}

func TestTunnelProcessDataFromImplantRejectsOversizedAndOutOfWindowFrames(t *testing.T) {
	tunnel := NewTunnel(15, "session")

	err := tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{
		Sequence: 0,
		Data:     make([]byte, MaxTunnelFrameBytes+1),
	})
	if !errors.Is(err, ErrTunnelFrameTooLarge) {
		t.Fatalf("oversized frame error = %v, want ErrTunnelFrameTooLarge", err)
	}
	err = tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{
		Sequence: maxTunnelPendingFrames,
		Data:     []byte("outside-window"),
	})
	if !errors.Is(err, ErrTunnelSequenceWindow) {
		t.Fatalf("out-of-window frame error = %v, want ErrTunnelSequenceWindow", err)
	}
	if len(tunnel.pendingFromImplant) != 0 || tunnel.pendingFromBytes != 0 {
		t.Fatalf("rejected frames changed pending state: frames=%d bytes=%d", len(tunnel.pendingFromImplant), tunnel.pendingFromBytes)
	}
}

//nolint:gocyclo // The test validates every state transition across a full bounded receive window.
func TestTunnelProcessDataFromImplantReordersFullReceiveWindow(t *testing.T) {
	tunnel := NewTunnel(16, "session")
	for sequence := maxTunnelPendingFrames - 1; sequence > 0; sequence-- {
		if err := tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{
			Sequence: uint64(sequence),
			Data:     []byte{byte(sequence)},
		}); err != nil {
			t.Fatalf("queue sequence %d: %v", sequence, err)
		}
	}

	received := make(chan []*sliverpb.TunnelData, 1)
	go func() {
		frames := make([]*sliverpb.TunnelData, 0, maxTunnelPendingFrames)
		for range maxTunnelPendingFrames {
			frames = append(frames, <-tunnel.FromImplant)
		}
		received <- frames
	}()
	processed := make(chan error, 1)
	go func() {
		processed <- tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{Sequence: 0, Data: []byte{0}})
	}()

	select {
	case err := <-processed:
		if err != nil {
			t.Fatalf("process sequence zero: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("full receive window did not drain")
	}
	select {
	case frames := <-received:
		for sequence, frame := range frames {
			if frame.Sequence != uint64(sequence) || len(frame.Data) != 1 || frame.Data[0] != byte(sequence) {
				t.Fatalf("frame %d = sequence %d data %v", sequence, frame.Sequence, frame.Data)
			}
		}
	case <-time.After(time.Second):
		t.Fatal("timed out collecting reordered frames")
	}
	if tunnel.FromImplantSequence != maxTunnelPendingFrames {
		t.Fatalf("next receive sequence = %d, want %d", tunnel.FromImplantSequence, maxTunnelPendingFrames)
	}
	if len(tunnel.pendingFromImplant) != 0 || tunnel.pendingFromBytes != 0 {
		t.Fatalf("drained receive state retained frames=%d bytes=%d", len(tunnel.pendingFromImplant), tunnel.pendingFromBytes)
	}
	if tunnel.fromImplantFrames != 0 || tunnel.fromImplantBytes != 0 {
		t.Fatalf("drained receive budget retained frames=%d bytes=%d", tunnel.fromImplantFrames, tunnel.fromImplantBytes)
	}
}

func TestTunnelProcessDataFromImplantBoundsBlockedHandlers(t *testing.T) {
	tunnel := NewTunnel(17, "session")
	results := make(chan error, maxTunnelPendingFrames)
	for range maxTunnelPendingFrames {
		go func() {
			results <- tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{Sequence: 0, Data: []byte("blocked")})
		}()
	}

	deadline := time.Now().Add(time.Second)
	for len(tunnel.fromImplantAdmission) != maxTunnelPendingFrames && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := len(tunnel.fromImplantAdmission); got != maxTunnelPendingFrames {
		tunnel.Close()
		t.Fatalf("admitted blocked handlers = %d, want %d", got, maxTunnelPendingFrames)
	}
	if err := tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{Sequence: 0}); !errors.Is(err, ErrTunnelIngressLimit) {
		tunnel.Close()
		t.Fatalf("handler beyond admission window error = %v, want ErrTunnelIngressLimit", err)
	}

	tunnel.Close()
	for range maxTunnelPendingFrames {
		select {
		case err := <-results:
			if !errors.Is(err, ErrTunnelClosed) {
				t.Errorf("blocked handler result = %v, want ErrTunnelClosed", err)
			}
		case <-time.After(time.Second):
			t.Fatal("blocked handler did not exit after tunnel close")
		}
	}
	if tunnel.fromImplantFrames != 0 || tunnel.fromImplantBytes != 0 {
		t.Fatalf("closed receive budget retained frames=%d bytes=%d", tunnel.fromImplantFrames, tunnel.fromImplantBytes)
	}
}

func TestTunnelProcessDataFromImplantBoundsCombinedReceiveBytes(t *testing.T) {
	tunnel := NewTunnel(171, "session")
	payload := make([]byte, MaxTunnelFrameBytes)
	for sequence := 1; sequence < maxTunnelPendingFrames; sequence++ {
		if err := tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{
			Sequence: uint64(sequence),
			Data:     payload,
		}); err != nil {
			t.Fatalf("queue full-size sequence %d: %v", sequence, err)
		}
	}

	blocked := make(chan error, 1)
	go func() {
		blocked <- tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{Sequence: 0, Data: payload})
	}()
	deadline := time.Now().Add(time.Second)
	reservedFrames := 0
	reservedBytes := 0
	for time.Now().Before(deadline) {
		tunnel.fromImplantBudget.Lock()
		reservedFrames = tunnel.fromImplantFrames
		reservedBytes = tunnel.fromImplantBytes
		tunnel.fromImplantBudget.Unlock()
		if reservedFrames == maxTunnelPendingFrames && reservedBytes == maxTunnelPendingBytes {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if reservedFrames != maxTunnelPendingFrames || reservedBytes != maxTunnelPendingBytes {
		tunnel.Close()
		t.Fatalf("combined receive reservation = %d frames/%d bytes, want %d/%d", reservedFrames, reservedBytes, maxTunnelPendingFrames, maxTunnelPendingBytes)
	}
	if err := tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{Sequence: 0}); !errors.Is(err, ErrTunnelIngressLimit) {
		tunnel.Close()
		t.Fatalf("frame beyond combined receive window error = %v, want ErrTunnelIngressLimit", err)
	}

	tunnel.Close()
	select {
	case err := <-blocked:
		if !errors.Is(err, ErrTunnelClosed) {
			t.Fatalf("blocked full-window frame result = %v, want ErrTunnelClosed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("full-byte-window handler did not exit after close")
	}
	tunnel.fromImplantBudget.Lock()
	defer tunnel.fromImplantBudget.Unlock()
	if tunnel.fromImplantFrames != 0 || tunnel.fromImplantBytes != 0 {
		t.Fatalf("closed receive budget retained frames=%d bytes=%d", tunnel.fromImplantFrames, tunnel.fromImplantBytes)
	}
}

//nolint:gocyclo // The test exercises cache eviction, replay, mutation isolation, and acknowledgement bounds together.
func TestTunnelOutboundResendCacheBoundsAndAcknowledgements(t *testing.T) {
	tunnel := NewTunnel(18, "session")
	for sequence := uint64(0); sequence <= maxTunnelResendFrames; sequence++ {
		input := []byte{byte(sequence)}
		frame, err := tunnel.NextDataToImplant(input)
		if err != nil {
			t.Fatalf("reserve sequence %d: %v", sequence, err)
		}
		if frame.Sequence != sequence {
			t.Fatalf("assigned sequence = %d, want %d", frame.Sequence, sequence)
		}
		input[0]++
	}
	if got := len(tunnel.toImplantCache); got != maxTunnelResendFrames {
		t.Fatalf("resend cache frames = %d, want %d", got, maxTunnelResendFrames)
	}
	if _, ok, err := tunnel.ResendDataToImplant(0); err != nil || ok {
		t.Fatalf("evicted resend = (_, %v, %v), want cache miss", ok, err)
	}
	resend, ok, err := tunnel.ResendDataToImplant(1)
	if err != nil || !ok {
		t.Fatalf("retained resend = (_, %v, %v), want hit", ok, err)
	}
	if len(resend.Data) != 1 || resend.Data[0] != 1 {
		t.Fatalf("cached frame was mutated through producer input: %v", resend.Data)
	}
	resend.Data[0] = 0xff
	resendAgain, ok, err := tunnel.ResendDataToImplant(1)
	if err != nil || !ok || resendAgain.Data[0] != 1 {
		t.Fatalf("cached frame was mutated through resend result: frame=%v ok=%v err=%v", resendAgain, ok, err)
	}

	if err := tunnel.AcknowledgeDataToImplant(64); err != nil {
		t.Fatalf("acknowledge sequence 64: %v", err)
	}
	if got := len(tunnel.toImplantCache); got != maxTunnelResendFrames+1-64 {
		t.Fatalf("cache after ACK = %d frames, want %d", got, maxTunnelResendFrames+1-64)
	}
	if err := tunnel.AcknowledgeDataToImplant(32); err != nil {
		t.Fatalf("stale acknowledgement: %v", err)
	}
	if err := tunnel.AcknowledgeDataToImplant(maxTunnelResendFrames + 2); !errors.Is(err, ErrTunnelAcknowledgement) {
		t.Fatalf("future acknowledgement error = %v, want ErrTunnelAcknowledgement", err)
	}
	if _, _, err := tunnel.ResendDataToImplant(maxTunnelResendFrames + 1); !errors.Is(err, ErrTunnelAcknowledgement) {
		t.Fatalf("future resend error = %v, want ErrTunnelAcknowledgement", err)
	}
	if err := tunnel.AcknowledgeDataToImplant(maxTunnelResendFrames + 1); err != nil {
		t.Fatalf("acknowledge all frames: %v", err)
	}
	if got := len(tunnel.toImplantCache); got != 0 {
		t.Fatalf("fully acknowledged cache retained %d frames", got)
	}
}

//nolint:gocyclo // The test keeps old and replacement generations in one lifecycle assertion.
func TestTunnelProtocolStateIsGenerationOwnedAndCleared(t *testing.T) {
	const tunnelID = uint64(19)
	old := NewTunnel(tunnelID, "old-session")
	registry := &tunnels{
		tunnels: map[uint64]*Tunnel{tunnelID: old},
		mutex:   &sync.Mutex{},
	}
	if err := old.ProcessDataFromImplant(&sliverpb.TunnelData{Sequence: 1, Data: []byte("old-pending")}); err != nil {
		t.Fatalf("queue old inbound frame: %v", err)
	}
	if _, err := old.NextDataToImplant([]byte("old-resend")); err != nil {
		t.Fatalf("queue old outbound frame: %v", err)
	}
	if !registry.CloseIf(old) {
		t.Fatal("failed to close old generation")
	}
	if len(old.pendingFromImplant) != 0 || old.pendingFromBytes != 0 || old.fromImplantFrames != 0 || old.fromImplantBytes != 0 || len(old.toImplantCache) != 0 {
		t.Fatalf("closed generation retained inbound=%d bytes=%d reserved-frames=%d reserved-bytes=%d outbound=%d", len(old.pendingFromImplant), old.pendingFromBytes, old.fromImplantFrames, old.fromImplantBytes, len(old.toImplantCache))
	}

	current := NewTunnel(tunnelID, "current-session")
	registry.mutex.Lock()
	registry.tunnels[tunnelID] = current
	registry.mutex.Unlock()
	if registry.CloseIf(old) {
		t.Fatal("stale close detached current generation")
	}
	if got := registry.Get(tunnelID); got != current {
		t.Fatalf("current generation changed after stale close: got=%p want=%p", got, current)
	}
	if err := old.ProcessDataFromImplant(&sliverpb.TunnelData{Sequence: 0}); !errors.Is(err, ErrTunnelClosed) {
		t.Fatalf("stale inbound result = %v, want ErrTunnelClosed", err)
	}
	frame, err := current.NextDataToImplant([]byte("current"))
	if err != nil || frame.Sequence != 0 {
		t.Fatalf("current generation first frame = %+v, %v", frame, err)
	}
	if !registry.CloseIf(current) {
		t.Fatal("failed to close current generation")
	}
	if err := current.AcknowledgeDataToImplant(1); !errors.Is(err, ErrTunnelClosed) {
		t.Fatalf("closed generation acknowledgement = %v, want ErrTunnelClosed", err)
	}
}

func TestSocksTunnelsCreateUnknownSession(t *testing.T) {
	tunnel, err := SocksTunnels.Create("does-not-exist")
	if err == nil {
		t.Fatalf("expected error for unknown session, got tunnel %+v", tunnel)
	}
	if tunnel != nil {
		t.Fatalf("expected nil tunnel on error, got %+v", tunnel)
	}
}

func TestSocksTunnelsCreateKnownSession(t *testing.T) {
	session := newTestSession()
	Sessions.Add(session)
	defer Sessions.Remove(session.ID)

	tunnel, err := SocksTunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tunnel == nil {
		t.Fatal("expected non-nil tunnel")
	}
	if tunnel.SessionID != session.ID {
		t.Fatalf("expected SessionID %q, got %q", session.ID, tunnel.SessionID)
	}
}

func TestPivotGraphEntryToProtobufUnknownSession(t *testing.T) {
	entry := &PivotGraphEntry{
		PeerID:    1,
		SessionID: "does-not-exist",
		Name:      "ghost",
		Children:  map[int64]*PivotGraphEntry{},
	}
	pb := entry.ToProtobuf()
	if pb == nil {
		t.Fatal("expected non-nil protobuf entry")
	}
	if pb.Session != nil {
		t.Fatalf("expected nil session for unknown session, got %+v", pb.Session)
	}
	if pb.PeerID != 1 || pb.Name != "ghost" {
		t.Fatalf("unexpected entry fields: %+v", pb)
	}
}
