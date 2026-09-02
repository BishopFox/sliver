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
	if tunnel.ImplantConnection() != session.Connection {
		t.Fatalf("expected creating implant connection %p, got %p", session.Connection, tunnel.ImplantConnection())
	}
}

func TestTunnelsCreateCapturesExactTerminalCapability(t *testing.T) {
	session := newTestSession()
	session.Capabilities = sliverpb.CapabilityTunnelTerminalV1
	Sessions.Add(session)
	t.Cleanup(func() {
		Sessions.RemoveIf(session)
		session.Connection.Close()
	})

	tunnel, err := Tunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create capable tunnel: %v", err)
	}
	t.Cleanup(func() { Tunnels.CloseIf(tunnel) })
	if !tunnel.TunnelTerminalEnabled() {
		t.Fatal("created tunnel did not retain the exact implant terminal capability")
	}

	// Capability belongs to the creating transport generation, not a mutable
	// session snapshot consulted during later teardown.
	session.Capabilities = 0
	if !tunnel.TunnelTerminalEnabled() {
		t.Fatal("tunnel terminal capability changed with the session snapshot")
	}
}

func TestTunnelImplantConnectionSurvivesSessionReplacement(t *testing.T) {
	creatingSession := newTestSession()
	Sessions.Add(creatingSession)
	t.Cleanup(func() {
		if Sessions.Get(creatingSession.ID) == creatingSession {
			Sessions.RemoveIf(creatingSession)
		}
		creatingSession.Connection.Close()
	})

	tunnel, err := Tunnels.Create(creatingSession.ID)
	if err != nil {
		t.Fatalf("create tunnel: %v", err)
	}
	t.Cleanup(func() { Tunnels.CloseIf(tunnel) })

	replacement := newTestSession()
	replacement.ID = creatingSession.ID
	Sessions.Add(replacement)
	t.Cleanup(func() {
		Sessions.Remove(replacement.ID)
		replacement.Connection.Close()
	})

	if tunnel.ImplantConnection() != creatingSession.Connection {
		t.Fatalf("tunnel implant connection after session replacement = %p, want creating connection %p", tunnel.ImplantConnection(), creatingSession.Connection)
	}
	if tunnel.ImplantConnection() == replacement.Connection {
		t.Fatal("tunnel implant connection changed to the replacement generation")
	}
	if got := Sessions.Get(creatingSession.ID); got != replacement {
		t.Fatalf("registered session after replacement = %p, want %p", got, replacement)
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

func TestTunnelsCloseForSessionIsScopedAndIdempotent(t *testing.T) {
	first := NewTunnel(101, "disconnecting-session")
	second := NewTunnel(102, "disconnecting-session")
	other := NewTunnel(103, "other-session")
	registry := &tunnels{
		tunnels: map[uint64]*Tunnel{first.ID: first, second.ID: second, other.ID: other},
		mutex:   &sync.Mutex{},
	}

	if closed := registry.CloseForSession("disconnecting-session"); closed != 2 {
		t.Fatalf("closed tunnel count = %d, want 2", closed)
	}
	for _, tunnel := range []*Tunnel{first, second} {
		select {
		case <-tunnel.Done():
		default:
			t.Fatalf("tunnel %d did not close with its session", tunnel.ID)
		}
		if got := registry.Get(tunnel.ID); got != nil {
			t.Fatalf("closed tunnel %d remains registered", tunnel.ID)
		}
	}
	if got := registry.Get(other.ID); got != other {
		t.Fatalf("unrelated tunnel = %p, want %p", got, other)
	}
	select {
	case <-other.Done():
		t.Fatal("unrelated session tunnel was closed")
	default:
	}
	if closed := registry.CloseForSession("disconnecting-session"); closed != 0 {
		t.Fatalf("idempotent close count = %d, want 0", closed)
	}
	if closed := registry.CloseForSession(""); closed != 0 {
		t.Fatalf("empty-session close count = %d, want 0", closed)
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

func TestTunnelClientBindLeaseStateTransitions(t *testing.T) {
	t.Run("expiry-before-reservation", func(t *testing.T) {
		tunnel := NewTunnel(201, "test-session")
		registry := &tunnels{
			tunnels: map[uint64]*Tunnel{tunnel.ID: tunnel},
			mutex:   &sync.Mutex{},
		}
		expiry := make(chan time.Time, 1)
		expiry <- time.Time{}
		registry.waitForClientBindLease(tunnel, expiry)

		if got := registry.Get(tunnel.ID); got != nil {
			t.Fatalf("expired tunnel remains registered: %p", got)
		}
		select {
		case <-tunnel.Done():
		default:
			t.Fatal("expired tunnel did not close")
		}
		if tunnel.BindClient(&testTunnelDataServer{}) {
			t.Fatal("client reserved a tunnel after its bind lease expired")
		}
		if !tunnel.ClientBindLeaseExpired() {
			t.Fatal("closed tunnel did not retain its bind-expired state")
		}
	})

	t.Run("expiry-after-reservation", func(t *testing.T) {
		tunnel := NewTunnel(202, "test-session")
		client := &testTunnelDataServer{}
		if !tunnel.BindClient(client) {
			t.Fatal("client failed to reserve tunnel")
		}
		registry := &tunnels{
			tunnels: map[uint64]*Tunnel{tunnel.ID: tunnel},
			mutex:   &sync.Mutex{},
		}
		expiry := make(chan time.Time, 1)
		expiry <- time.Time{}
		registry.waitForClientBindLease(tunnel, expiry)

		if tunnel.MarkClientBound(client) {
			t.Fatal("reserved client completed a bind after its lease expired")
		}
		if !tunnel.ClientBindLeaseExpired() {
			t.Fatal("reserved tunnel did not report bind expiry")
		}
		if got := registry.Get(tunnel.ID); got != nil {
			t.Fatalf("expired reserved tunnel remains registered: %p", got)
		}
		select {
		case <-tunnel.Done():
		default:
			t.Fatal("expired reserved tunnel did not close")
		}
	})

	t.Run("completed-bind-wins", func(t *testing.T) {
		tunnel := NewTunnel(203, "test-session")
		client := &testTunnelDataServer{}
		if !tunnel.BindClient(client) || !tunnel.MarkClientBound(client) {
			t.Fatal("client failed to complete tunnel bind")
		}
		registry := &tunnels{
			tunnels: map[uint64]*Tunnel{tunnel.ID: tunnel},
			mutex:   &sync.Mutex{},
		}
		expiry := make(chan time.Time, 1)
		expiry <- time.Time{}
		registry.waitForClientBindLease(tunnel, expiry)

		if got := registry.Get(tunnel.ID); got != tunnel {
			t.Fatalf("completed tunnel after lease signal = %p, want %p", got, tunnel)
		}
		select {
		case <-tunnel.Done():
			t.Fatal("lease signal closed a completed client bind")
		default:
		}
		if tunnel.ClientBindLeaseExpired() {
			t.Fatal("completed client bind was marked expired")
		}
		tunnel.Close()
	})
}

func TestTunnelsClientBindLeaseCloseIsGenerationScoped(t *testing.T) {
	const tunnelID = uint64(204)
	old := NewTunnel(tunnelID, "old-session")
	current := NewTunnel(tunnelID, "current-session")
	registry := &tunnels{
		tunnels: map[uint64]*Tunnel{tunnelID: current},
		mutex:   &sync.Mutex{},
	}
	expiry := make(chan time.Time, 1)
	expiry <- time.Time{}
	registry.waitForClientBindLease(old, expiry)

	if got := registry.Get(tunnelID); got != current {
		t.Fatalf("stale lease changed current generation: got=%p want=%p", got, current)
	}
	select {
	case <-current.Done():
		t.Fatal("stale lease closed the current tunnel generation")
	default:
	}
	old.Close()
	current.Close()
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

func TestTunnelClientDataRefreshesOnlyToImplantCloseActivity(t *testing.T) {
	tunnel := NewTunnel(13, "session")
	beforeToImplant := tunnel.LastToImplantTime()
	beforeFromImplant := tunnel.LastFromImplantTime()
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
	if after := tunnel.LastToImplantTime(); !after.After(beforeToImplant) {
		t.Fatalf("client data did not refresh to-implant close activity: before=%v after=%v", beforeToImplant, after)
	}
	if after := tunnel.LastFromImplantTime(); !after.Equal(beforeFromImplant) {
		t.Fatalf("client data changed from-implant close activity: before=%v after=%v", beforeFromImplant, after)
	}
}

func TestTunnelImplantDataRefreshesOnlyFromImplantCloseActivity(t *testing.T) {
	tunnel := NewTunnel(14, "session")
	beforeToImplant := tunnel.LastToImplantTime()
	beforeFromImplant := tunnel.LastFromImplantTime()
	time.Sleep(time.Millisecond)
	received := make(chan *sliverpb.TunnelData, 1)
	go func() { received <- <-tunnel.FromImplant }()
	frame := &sliverpb.TunnelData{TunnelID: tunnel.ID, Sequence: 0, Data: []byte("keepalive")}
	if err := tunnel.ProcessDataFromImplant(frame); err != nil {
		t.Fatalf("process implant data: %v", err)
	}
	if got := <-received; string(got.Data) != string(frame.Data) {
		t.Fatalf("delivered data = %q, want %q", got.Data, frame.Data)
	}
	if after := tunnel.LastFromImplantTime(); !after.After(beforeFromImplant) {
		t.Fatalf("implant data did not refresh from-implant close activity: before=%v after=%v", beforeFromImplant, after)
	}
	if after := tunnel.LastToImplantTime(); !after.Equal(beforeToImplant) {
		t.Fatalf("implant data changed to-implant close activity: before=%v after=%v", beforeToImplant, after)
	}
}

func TestTunnelDirectionalCloseTouchesAreIndependent(t *testing.T) {
	tunnel := NewTunnel(141, "session")
	stale := time.Now().Add(-2 * delayBeforeClose)
	tunnel.mutex.Lock()
	tunnel.lastToImplantTime = stale
	tunnel.lastFromImplantTime = stale
	tunnel.mutex.Unlock()

	tunnel.TouchToImplant()
	toImplant := tunnel.LastToImplantTime()
	if age := time.Since(toImplant); age < 0 || age >= delayBeforeClose {
		t.Fatalf("TouchToImplant did not establish a fresh close grace period: age=%v", age)
	}
	if fromImplant := tunnel.LastFromImplantTime(); !fromImplant.Equal(stale) {
		t.Fatalf("TouchToImplant changed from-implant activity: got=%v want=%v", fromImplant, stale)
	}

	tunnel.TouchFromImplant()
	if age := time.Since(tunnel.LastFromImplantTime()); age < 0 || age >= delayBeforeClose {
		t.Fatalf("TouchFromImplant did not establish a fresh close grace period: age=%v", age)
	}
	if got := tunnel.LastToImplantTime(); !got.Equal(toImplant) {
		t.Fatalf("TouchFromImplant changed to-implant activity: got=%v want=%v", got, toImplant)
	}
}

func TestTunnelClaimFromImplantCloseIsIdempotent(t *testing.T) {
	tunnel := NewTunnel(144, "session")
	before := tunnel.LastFromImplantTime()
	time.Sleep(time.Millisecond)
	if !tunnel.ClaimFromImplantClose() {
		t.Fatal("first implant terminal did not claim close ownership")
	}
	claimedAt := tunnel.LastFromImplantTime()
	if !claimedAt.After(before) {
		t.Fatalf("first terminal did not establish close grace: before=%v after=%v", before, claimedAt)
	}

	time.Sleep(time.Millisecond)
	results := make(chan bool, 100)
	for range 100 {
		go func() { results <- tunnel.ClaimFromImplantClose() }()
	}
	for range 100 {
		if <-results {
			t.Fatal("duplicate implant terminal claimed another close scheduler")
		}
	}
	if got := tunnel.LastFromImplantTime(); !got.Equal(claimedAt) {
		t.Fatalf("duplicate terminal refreshed close grace: got=%v want=%v", got, claimedAt)
	}
}

//nolint:gocyclo // The assertions cover every stage of one ordered terminal lifecycle.
func TestTunnelSequencedTerminalWaitsForPrecedingFrames(t *testing.T) {
	tunnel := NewTunnel(145, "session")
	ready, err := tunnel.MarkFromImplantTerminal(&sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 2,
		Closed:   true,
	})
	if err != nil {
		t.Fatalf("mark sequenced terminal: %v", err)
	}
	if ready {
		t.Fatal("terminal was ready before preceding frames arrived")
	}
	if err := tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 1,
		Data:     []byte("second"),
	}); err != nil {
		t.Fatalf("queue second frame: %v", err)
	}

	received := make(chan []*sliverpb.TunnelData, 1)
	go func() {
		frames := make([]*sliverpb.TunnelData, 0, 2)
		for range 2 {
			frames = append(frames, <-tunnel.FromImplant)
		}
		received <- frames
	}()
	if err := tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Data:     []byte("first"),
	}); err != nil {
		t.Fatalf("process first frame: %v", err)
	}
	var frames []*sliverpb.TunnelData
	select {
	case frames = <-received:
		if len(frames) != 2 || frames[0].Sequence != 0 || string(frames[0].Data) != "first" ||
			frames[1].Sequence != 1 || string(frames[1].Data) != "second" {
			t.Fatalf("sequenced terminal frames = %+v", frames)
		}
	case <-time.After(time.Second):
		t.Fatal("preceding frames did not drain in order")
	}
	select {
	case <-tunnel.FromImplantTerminalReady():
		t.Fatal("terminal became ready before preceding frames were forwarded")
	default:
	}
	for _, frame := range frames {
		if err := tunnel.CompleteDataFromImplantForward(frame.Sequence); err != nil {
			t.Fatalf("complete forwarded frame %d: %v", frame.Sequence, err)
		}
	}
	select {
	case <-tunnel.FromImplantTerminalReady():
	case <-time.After(time.Second):
		t.Fatal("terminal did not become ready after preceding frames were forwarded")
	}
	if err := tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 2,
		Data:     []byte("after terminal"),
	}); !errors.Is(err, ErrTunnelTerminal) {
		t.Fatalf("data at terminal error = %v, want ErrTunnelTerminal", err)
	}
}

func TestTunnelConnectionWatcherClosesOnlyOldGeneration(t *testing.T) {
	originalConnection := NewImplantConnection("test", "original")
	replacementConnection := NewImplantConnection("test", "replacement")
	t.Cleanup(originalConnection.Close)
	t.Cleanup(replacementConnection.Close)
	original := newTunnel(146, "same-session", originalConnection)
	replacement := newTunnel(original.ID, original.SessionID, replacementConnection)
	registry := &tunnels{
		tunnels: map[uint64]*Tunnel{original.ID: original},
		mutex:   &sync.Mutex{},
	}
	go registry.waitForImplantConnection(original)

	registry.mutex.Lock()
	registry.tunnels[original.ID] = replacement
	registry.mutex.Unlock()
	originalConnection.Close()
	select {
	case <-original.Done():
	case <-time.After(time.Second):
		t.Fatal("old implant connection teardown did not close its detached tunnel")
	}
	if got := registry.Get(original.ID); got != replacement {
		t.Fatalf("old connection watcher changed replacement registry entry: got=%p want=%p", got, replacement)
	}
	select {
	case <-replacement.Done():
		t.Fatal("old connection watcher closed replacement tunnel")
	default:
	}
	select {
	case <-replacementConnection.Done():
		t.Fatal("old connection watcher closed replacement transport")
	default:
	}
	replacement.Close()
}

func TestTunnelsClientCloseIgnoresFromImplantActivity(t *testing.T) {
	tunnel := NewTunnel(142, "session")
	registry := &tunnels{
		tunnels: map[uint64]*Tunnel{tunnel.ID: tunnel},
		mutex:   &sync.Mutex{},
	}
	stale := time.Now().Add(-2 * delayBeforeClose)
	tunnel.mutex.Lock()
	tunnel.lastToImplantTime = stale
	tunnel.lastFromImplantTime = stale
	tunnel.mutex.Unlock()
	received := make(chan *sliverpb.TunnelData, 1)
	go func() { received <- <-tunnel.FromImplant }()
	frame := &sliverpb.TunnelData{TunnelID: tunnel.ID, Sequence: 0, Data: []byte("target-keepalive")}
	if err := tunnel.ProcessDataFromImplant(frame); err != nil {
		t.Fatalf("process target keepalive: %v", err)
	}
	if got := <-received; string(got.Data) != string(frame.Data) {
		t.Fatalf("delivered target keepalive = %q, want %q", got.Data, frame.Data)
	}
	if got := tunnel.LastToImplantTime(); !got.Equal(stale) {
		t.Fatalf("target keepalive changed client-close activity: got=%v want=%v", got, stale)
	}
	if got := tunnel.LastFromImplantTime(); !got.After(stale) {
		t.Fatalf("target keepalive did not refresh implant-close activity: got=%v stale=%v", got, stale)
	}

	registry.ScheduleCloseTunnelToImplant(tunnel)
	if got := registry.Get(tunnel.ID); got != nil {
		t.Fatalf("client-requested close was extended by from-implant activity: %+v", got)
	}
	select {
	case <-tunnel.Done():
	default:
		t.Fatal("client-requested close did not close tunnel")
	}
}

func TestTunnelQuiesceDataToImplantJoinsAcceptedForward(t *testing.T) {
	tunnel := NewTunnel(149, "session")
	received := make(chan struct{})
	release := make(chan struct{})
	go func() {
		<-tunnel.ToImplant
		close(received)
		<-release
		tunnel.CompleteDataToImplant()
	}()

	accepted := make(chan bool, 1)
	go func() { accepted <- tunnel.SendDataToImplant([]byte("final")) }()
	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("forwarding worker did not receive accepted final payload")
	}
	if !<-accepted {
		t.Fatal("final payload was not accepted")
	}

	quiesced := make(chan struct{})
	go func() {
		tunnel.QuiesceDataToImplant()
		close(quiesced)
	}()
	select {
	case <-quiesced:
		t.Fatal("graceful close did not wait for the accepted final payload")
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	select {
	case <-quiesced:
	case <-time.After(time.Second):
		t.Fatal("graceful close did not resume after final payload completion")
	}

	rejected := make(chan bool, 1)
	go func() { rejected <- tunnel.SendDataToImplant([]byte("late")) }()
	select {
	case acceptedLate := <-rejected:
		if acceptedLate {
			t.Fatal("quiesced tunnel accepted a later payload")
		}
	case <-time.After(time.Second):
		t.Fatal("late payload blocked after tunnel quiescence")
	}
}

func TestTunnelTerminalSequenceTracksOnlySuccessfullyForwardedPrefix(t *testing.T) {
	tunnel := newTunnelWithCapabilities(150, "session", nil, sliverpb.CapabilityTunnelTerminalV1)
	first, err := tunnel.NextDataToImplant([]byte("first"))
	if err != nil {
		t.Fatalf("assign first frame: %v", err)
	}
	if got := tunnel.ToImplantTerminalSequence(); got != 0 {
		t.Fatalf("terminal included assigned but unforwarded frame: got %d, want 0", got)
	}
	if err := tunnel.CompleteDataToImplantForward(first.Sequence); err != nil {
		t.Fatalf("complete first frame: %v", err)
	}
	if got := tunnel.ToImplantTerminalSequence(); got != 1 {
		t.Fatalf("terminal after first forwarded frame = %d, want 1", got)
	}
	second, err := tunnel.NextDataToImplant([]byte("second"))
	if err != nil {
		t.Fatalf("assign second frame: %v", err)
	}
	if got := tunnel.ToImplantTerminalSequence(); got != 1 {
		t.Fatalf("terminal included second unforwarded frame: got %d, want 1", got)
	}
	if err := tunnel.CompleteDataToImplantForward(second.Sequence); err != nil {
		t.Fatalf("complete second frame: %v", err)
	}
	if got := tunnel.ToImplantTerminalSequence(); got != 2 {
		t.Fatalf("terminal after forwarded prefix = %d, want 2", got)
	}
}

//nolint:gocyclo // The test drives the terminal deadline through a blocked ordered delivery.
func TestTunnelTerminalDeadlineBreaksBlockedAdmissionBeforeMarkReturns(t *testing.T) {
	connection := NewImplantConnection("mtls", "blocked-terminal-mark")
	t.Cleanup(connection.Close)
	tunnel := newTunnelWithCapabilities(151, "session", connection, sliverpb.CapabilityTunnelTerminalV1)
	registry := &tunnels{
		tunnels: map[uint64]*Tunnel{tunnel.ID: tunnel},
		mutex:   &sync.Mutex{},
	}

	dataDone := make(chan error, 1)
	go func() {
		dataDone <- tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{
			TunnelID: tunnel.ID,
			Sequence: 0,
			Data:     []byte("blocked operator delivery"),
		})
	}()
	deadline := time.Now().Add(time.Second)
	for tunnel.fromImplantMutex.TryLock() {
		tunnel.fromImplantMutex.Unlock()
		if time.Now().After(deadline) {
			t.Fatal("data admission did not block while holding the ordering mutex")
		}
		time.Sleep(time.Millisecond)
	}

	expiry := make(chan time.Time, 1)
	registry.armFromImplantTerminalClose(tunnel, expiry)
	markDone := make(chan error, 1)
	go func() {
		_, err := tunnel.MarkFromImplantTerminal(&sliverpb.TunnelData{
			TunnelID: tunnel.ID,
			Sequence: 1,
			Closed:   true,
		})
		markDone <- err
	}()
	expiry <- time.Now()

	select {
	case <-tunnel.Done():
	case <-time.After(time.Second):
		t.Fatal("terminal deadline did not close the blocked tunnel")
	}
	select {
	case err := <-dataDone:
		if !errors.Is(err, ErrTunnelClosed) {
			t.Fatalf("blocked data result = %v, want ErrTunnelClosed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked data admission did not wake on terminal deadline")
	}
	select {
	case err := <-markDone:
		if !errors.Is(err, ErrTunnelClosed) {
			t.Fatalf("blocked terminal mark result = %v, want ErrTunnelClosed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("terminal mark did not return after deadline closure")
	}
	select {
	case <-connection.Done():
	case <-time.After(time.Second):
		t.Fatal("incomplete terminal deadline did not fail the exact connection closed")
	}
}

func TestTunnelsImplantCloseIgnoresToImplantActivity(t *testing.T) {
	tunnel := NewTunnel(143, "session")
	registry := &tunnels{
		tunnels: map[uint64]*Tunnel{tunnel.ID: tunnel},
		mutex:   &sync.Mutex{},
	}
	tunnel.mutex.Lock()
	tunnel.lastToImplantTime = time.Now()
	tunnel.lastFromImplantTime = time.Now().Add(-2 * delayBeforeClose)
	tunnel.mutex.Unlock()

	registry.ScheduleCloseTunnelFromImplant(tunnel)
	if got := registry.Get(tunnel.ID); got != nil {
		t.Fatalf("implant-requested close was extended by to-implant activity: %+v", got)
	}
	select {
	case <-tunnel.Done():
	default:
		t.Fatal("implant-requested close did not close tunnel")
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
	if tunnel.ImplantConnection() != session.Connection {
		t.Fatal("SOCKS tunnel did not retain its creating session connection")
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
