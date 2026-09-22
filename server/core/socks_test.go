package core

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func newUnitSocksTunnel(id uint64, sessionID string) *TcpTunnel {
	return newTcpTunnel(id, sessionID, nil)
}

func TestTcpTunnelDeliverFromImplantCloseRace(t *testing.T) {
	const iterations = 1000
	for iteration := 0; iteration < iterations; iteration++ {
		tunnel := newUnitSocksTunnel(uint64(iteration+1), "session")
		start := make(chan struct{})
		delivered := make(chan bool, 1)
		closed := make(chan bool, 1)
		go func() {
			<-start
			delivered <- tunnel.DeliverFromImplant(&sliverpb.SocksData{TunnelID: tunnel.ID})
		}()
		go func() {
			<-start
			closed <- tunnel.close()
		}()
		close(start)

		select {
		case <-delivered:
		case <-time.After(time.Second):
			t.Fatalf("iteration %d: implant delivery blocked during close", iteration)
		}
		select {
		case firstClose := <-closed:
			if !firstClose {
				t.Fatalf("iteration %d: first tunnel close returned false", iteration)
			}
		case <-time.After(time.Second):
			t.Fatalf("iteration %d: tunnel close blocked during delivery", iteration)
		}
		if tunnel.DeliverFromImplant(&sliverpb.SocksData{TunnelID: tunnel.ID}) {
			t.Fatalf("iteration %d: closed tunnel accepted implant data", iteration)
		}
	}
}

func TestSocksTunnelRegistryCloseIfIsGenerationSafeAndIdempotent(t *testing.T) {
	registry := tcpTunnel{tunnels: map[uint64]*TcpTunnel{}, mutex: &sync.RWMutex{}}
	old := newUnitSocksTunnel(7, "session")
	current := newUnitSocksTunnel(7, "session")
	registry.tunnels[7] = current

	if registry.CloseIf(old) {
		t.Fatal("stale SOCKS tunnel generation closed the current tunnel")
	}
	if got := registry.Get(7); got != current {
		t.Fatalf("stale close changed registry: got=%p want=%p", got, current)
	}
	if !registry.CloseIf(current) {
		t.Fatal("current SOCKS tunnel generation did not close")
	}
	if registry.CloseIf(current) {
		t.Fatal("second SOCKS tunnel close returned true")
	}
	if got := registry.Get(7); got != nil {
		t.Fatalf("closed SOCKS tunnel remained registered: %p", got)
	}
	select {
	case <-current.Done():
	default:
		t.Fatal("closed SOCKS tunnel did not signal done")
	}
}

func TestTcpTunnelClientBindingIsExclusive(t *testing.T) {
	tunnel := newUnitSocksTunnel(11, "session")
	first := NewSocksClient(&trackingSocksSender{})
	second := NewSocksClient(&trackingSocksSender{})

	if owned, newlyBound := tunnel.BindClient(first); !owned || !newlyBound {
		t.Fatalf("first bind = owned:%v new:%v, want true,true", owned, newlyBound)
	}
	if owned, newlyBound := tunnel.BindClient(first); !owned || newlyBound {
		t.Fatalf("repeat owner bind = owned:%v new:%v, want true,false", owned, newlyBound)
	}
	if owned, newlyBound := tunnel.BindClient(second); owned || newlyBound {
		t.Fatalf("second client bind = owned:%v new:%v, want false,false", owned, newlyBound)
	}
	tunnel.close()
	if owned, newlyBound := tunnel.BindClient(first); owned || newlyBound {
		t.Fatalf("closed tunnel bind = owned:%v new:%v, want false,false", owned, newlyBound)
	}
}

//nolint:gocyclo // The test intentionally checks every lifecycle snapshot transition.
func TestTcpTunnelTracksLifecycleCapabilityAndFirstPayload(t *testing.T) {
	tunnel := newUnitSocksTunnel(111, "session")
	client := NewSocksClient(&trackingSocksSender{})
	if owned, newlyBound, err := tunnel.BindClientWithCapabilities(client, "user", "password", true); err != nil || !owned || !newlyBound {
		t.Fatalf("capability bind = owned:%v new:%v err:%v", owned, newlyBound, err)
	}
	bound := tunnel.ClientLifecycle()
	if bound.BoundAt.IsZero() || bound.LastActivity.IsZero() || bound.ReceivedPayload || !bound.SendsTerminal {
		t.Fatalf("initial lifecycle = %+v", bound)
	}

	if err := tunnel.AdmitToImplant(&sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 0, Data: []byte("greeting")}); err != nil {
		t.Fatalf("admit first payload: %v", err)
	}
	afterPayload := tunnel.ClientLifecycle()
	if !afterPayload.ReceivedPayload || !afterPayload.SendsTerminal || afterPayload.LastActivity.Before(bound.LastActivity) {
		t.Fatalf("lifecycle after first payload = %+v, initial %+v", afterPayload, bound)
	}

	legacy := newUnitSocksTunnel(112, "session")
	if owned, newlyBound, err := legacy.BindClientWithCredentials(client, "", ""); err != nil || !owned || !newlyBound {
		t.Fatalf("legacy bind = owned:%v new:%v err:%v", owned, newlyBound, err)
	}
	if lifecycle := legacy.ClientLifecycle(); lifecycle.SendsTerminal {
		t.Fatalf("legacy lifecycle unexpectedly advertises terminals: %+v", lifecycle)
	}
}

//nolint:gocyclo // Capability ownership and both acknowledgement directions form one scenario.
func TestTcpTunnelNegotiatesCapabilitiesAndCoalescesOwnedAcknowledgements(t *testing.T) {
	connection := NewImplantConnection("flow-control", "test")
	t.Cleanup(connection.Close)
	tunnel := newTCPTunnelWithCapabilities(113, "session", connection, sliverpb.CapabilitySocksFlowControlV1)
	t.Cleanup(func() { tunnel.close() })
	client := NewSocksClient(&trackingSocksSender{})
	otherClient := NewSocksClient(&trackingSocksSender{})

	if owned, newlyBound, err := tunnel.BindClientWithCapabilities(client, "", "", true); !errors.Is(err, ErrSocksCapabilityMismatch) || owned || newlyBound {
		t.Fatalf("bind without negotiated capability echo = owned:%v new:%v err:%v", owned, newlyBound, err)
	}
	if owned, newlyBound, err := tunnel.BindClientWithNegotiatedCapabilities(client, "user", "password", true, sliverpb.CapabilitySocksFlowControlV1); err != nil || !owned || !newlyBound {
		t.Fatalf("negotiated capability bind = owned:%v new:%v err:%v", owned, newlyBound, err)
	}
	if got := tunnel.Capabilities(); got != sliverpb.CapabilitySocksFlowControlV1 || !tunnel.FlowControlEnabled() {
		t.Fatalf("negotiated capabilities = %d enabled:%v", got, tunnel.FlowControlEnabled())
	}

	atomic.StoreUint64(&tunnel.FromImplantSequence, 3)
	for _, ack := range []uint64{1, 3, 2} {
		if err := tunnel.RelayClientAcknowledgement(client, ack); err != nil {
			t.Fatalf("relay client acknowledgement %d: %v", ack, err)
		}
	}
	select {
	case ack := <-tunnel.AcknowledgementsToImplant():
		if ack != 3 {
			t.Fatalf("coalesced client acknowledgement = %d, want 3", ack)
		}
	default:
		t.Fatal("client acknowledgement mailbox was empty")
	}
	select {
	case ack := <-tunnel.AcknowledgementsToImplant():
		t.Fatalf("client acknowledgement mailbox retained duplicate %d", ack)
	default:
	}
	if err := tunnel.RelayClientAcknowledgement(otherClient, 1); !errors.Is(err, ErrSocksOwner) {
		t.Fatalf("wrong-client acknowledgement = %v, want %v", err, ErrSocksOwner)
	}
	if err := tunnel.RelayClientAcknowledgement(client, 0); !errors.Is(err, ErrSocksAcknowledgement) {
		t.Fatalf("zero client acknowledgement = %v, want %v", err, ErrSocksAcknowledgement)
	}
	if err := tunnel.RelayClientAcknowledgement(client, 4); !errors.Is(err, ErrSocksAcknowledgement) {
		t.Fatalf("future client acknowledgement = %v, want %v", err, ErrSocksAcknowledgement)
	}

	atomic.StoreUint64(&tunnel.ToImplantSequence, 4)
	for _, ack := range []uint64{2, 4, 3} {
		if err := tunnel.RelayImplantAcknowledgement(connection, ack); err != nil {
			t.Fatalf("relay implant acknowledgement %d: %v", ack, err)
		}
	}
	select {
	case ack := <-tunnel.AcknowledgementsToClient():
		if ack != 4 {
			t.Fatalf("coalesced implant acknowledgement = %d, want 4", ack)
		}
	default:
		t.Fatal("implant acknowledgement mailbox was empty")
	}
	wrongConnection := NewImplantConnection("wrong-flow-owner", "test")
	t.Cleanup(wrongConnection.Close)
	if err := tunnel.RelayImplantAcknowledgement(wrongConnection, 1); !errors.Is(err, ErrSocksOwner) {
		t.Fatalf("wrong-implant acknowledgement = %v, want %v", err, ErrSocksOwner)
	}
	if err := tunnel.RelayImplantAcknowledgement(connection, 5); !errors.Is(err, ErrSocksAcknowledgement) {
		t.Fatalf("future implant acknowledgement = %v, want %v", err, ErrSocksAcknowledgement)
	}

	select {
	case frame := <-tunnel.ToImplant():
		t.Fatalf("acknowledgement entered to-implant data queue: %+v", frame)
	default:
	}
	select {
	case frame := <-tunnel.FromImplant():
		t.Fatalf("acknowledgement entered from-implant data queue: %+v", frame)
	default:
	}
}

func TestTcpTunnelRejectsAcknowledgementsWithoutFlowNegotiation(t *testing.T) {
	connection := NewImplantConnection("legacy-flow", "test")
	t.Cleanup(connection.Close)
	tunnel := newTcpTunnel(114, "session", connection)
	t.Cleanup(func() { tunnel.close() })
	client := NewSocksClient(&trackingSocksSender{})
	if owned, newlyBound := tunnel.BindClient(client); !owned || !newlyBound {
		t.Fatalf("legacy bind = owned:%v new:%v", owned, newlyBound)
	}
	atomic.StoreUint64(&tunnel.FromImplantSequence, 1)
	atomic.StoreUint64(&tunnel.ToImplantSequence, 1)
	if err := tunnel.RelayClientAcknowledgement(client, 1); !errors.Is(err, ErrSocksFlowControl) {
		t.Fatalf("legacy client acknowledgement = %v, want %v", err, ErrSocksFlowControl)
	}
	if err := tunnel.RelayImplantAcknowledgement(connection, 1); !errors.Is(err, ErrSocksFlowControl) {
		t.Fatalf("legacy implant acknowledgement = %v, want %v", err, ErrSocksFlowControl)
	}
}

func TestSocksTunnelRegistryBoundsPerSessionPreGreetingState(t *testing.T) {
	connection := NewImplantConnection("test", "socks-capacity")
	session := NewSession(connection)
	Sessions.Add(session)
	registry := tcpTunnel{tunnels: map[uint64]*TcpTunnel{}, mutex: &sync.RWMutex{}}
	for index := 0; index < maxSocksTunnelsPerSession; index++ {
		tunnel := newTcpTunnel(uint64(index+1), session.ID, connection)
		registry.tunnels[tunnel.ID] = tunnel
	}
	t.Cleanup(func() {
		for _, tunnel := range registry.List() {
			registry.CloseIf(tunnel)
		}
		Sessions.Remove(session.ID)
		connection.Close()
	})

	if tunnel, err := registry.Create(session.ID); !errors.Is(err, ErrSocksTunnelLimit) || tunnel != nil {
		t.Fatalf("create beyond per-session limit = tunnel:%p err:%v, want nil/%v", tunnel, err, ErrSocksTunnelLimit)
	}
	registry.CloseIf(registry.tunnels[1])
	if tunnel, err := registry.Create(session.ID); err != nil || tunnel == nil {
		t.Fatalf("create after releasing capacity = tunnel:%p err:%v", tunnel, err)
	}
}

func TestSocksTunnelCreateForSessionRejectsReplacedGeneration(t *testing.T) {
	originalConnection := NewImplantConnection("original-generation", "test")
	original := NewSession(originalConnection)
	Sessions.Add(original)
	replacementConnection := NewImplantConnection("replacement-generation", "test")
	replacement := NewSession(replacementConnection)
	replacement.ID = original.ID
	Sessions.Add(replacement)
	t.Cleanup(func() {
		Sessions.RemoveIf(replacement)
		originalConnection.Close()
		replacementConnection.Close()
	})

	registry := tcpTunnel{tunnels: map[uint64]*TcpTunnel{}, mutex: &sync.RWMutex{}}
	tunnel, err := registry.CreateForSession(original, sliverpb.CapabilitySocksFlowControlV1)
	if !errors.Is(err, ErrInvalidSessionID) || tunnel != nil {
		t.Fatalf("create with replaced session = tunnel:%p err:%v, want nil/%v", tunnel, err, ErrInvalidSessionID)
	}
	if got := len(registry.List()); got != 0 {
		t.Fatalf("replaced generation published %d SOCKS tunnels", got)
	}

	tunnel, err = registry.CreateForSession(replacement, sliverpb.CapabilitySocksFlowControlV1)
	if err != nil || tunnel == nil {
		t.Fatalf("create with exact replacement = tunnel:%p err:%v", tunnel, err)
	}
	if tunnel.ImplantConnection() != replacementConnection || tunnel.Capabilities() != sliverpb.CapabilitySocksFlowControlV1 {
		t.Fatalf("replacement tunnel owner/capabilities = %p/%d", tunnel.ImplantConnection(), tunnel.Capabilities())
	}
	registry.CloseIf(tunnel)
}

//nolint:gocyclo // The test keeps bind, credential, and budget invariants in one scenario.
func TestTcpTunnelCredentialsAreBoundOnceAndExcludedFromFrameBudget(t *testing.T) {
	tunnel := newUnitSocksTunnel(12, "session")
	client := NewSocksClient(&trackingSocksSender{})
	if owned, newlyBound, err := tunnel.BindClientWithCredentials(client, "user", "password"); err != nil || !owned || !newlyBound {
		t.Fatalf("credential bind = owned:%v new:%v err:%v", owned, newlyBound, err)
	}
	if owned, newlyBound, err := tunnel.BindClientWithCredentials(client, string(make([]byte, maxSocksCredentialBytes+1)), "replacement"); err != nil || !owned || newlyBound {
		t.Fatalf("repeat bind metadata = owned:%v new:%v err:%v", owned, newlyBound, err)
	}
	if username, password := tunnel.Credentials(); username != "user" || password != "password" {
		t.Fatalf("later metadata replaced credentials: %q/%q", username, password)
	}

	requestMetadata := string(make([]byte, MaxSocksFrameBytes))
	if err := tunnel.AdmitToImplant(&sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: 1,
		Data:     []byte("counted"),
		Username: requestMetadata,
		Password: requestMetadata,
		Request:  &commonpb.Request{SessionID: requestMetadata},
	}); err != nil {
		t.Fatalf("admit metadata-bearing frame: %v", err)
	}
	tunnel.toImplant.mu.Lock()
	pending := tunnel.toImplant.pending[1]
	tunnel.toImplant.mu.Unlock()
	if pending == nil || pending.Username != "" || pending.Password != "" || pending.Request != nil {
		t.Fatal("pending queue retained arbitrary metadata")
	}
	_, frames, size := tunnel.toImplant.snapshot()
	if frames != 1 || size != len("counted") {
		t.Fatalf("metadata inflated queue budget: frames=%d bytes=%d", frames, size)
	}
	if !tunnel.close() {
		t.Fatal("close credential-bearing tunnel")
	}
	if username, password := tunnel.Credentials(); username != "" || password != "" {
		t.Fatalf("closed tunnel retained credentials: %q/%q", username, password)
	}

	unbound := newUnitSocksTunnel(13, "session")
	tooLong := string(make([]byte, maxSocksCredentialBytes+1))
	if owned, newlyBound, err := unbound.BindClientWithCredentials(client, tooLong, ""); !errors.Is(err, ErrSocksCredentialSize) || owned || newlyBound {
		t.Fatalf("oversized credential bind = owned:%v new:%v err:%v", owned, newlyBound, err)
	}
	if unbound.Client() != nil {
		t.Fatal("oversized credentials claimed tunnel")
	}
}

type trackingSocksSender struct {
	inFlight atomic.Int32
	overlap  atomic.Bool
	calls    atomic.Int32
}

func (sender *trackingSocksSender) Send(*sliverpb.SocksData) error {
	if sender.inFlight.Add(1) != 1 {
		sender.overlap.Store(true)
	}
	time.Sleep(time.Millisecond)
	sender.calls.Add(1)
	sender.inFlight.Add(-1)
	return nil
}

func TestSocksClientSerializesConcurrentSends(t *testing.T) {
	const sendCount = 32
	sender := &trackingSocksSender{}
	client := NewSocksClient(sender)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for index := 0; index < sendCount; index++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			<-start
			if err := client.Send(&sliverpb.SocksData{TunnelID: uint64(index + 1)}); err != nil {
				t.Errorf("send %d: %v", index, err)
			}
		}(index)
	}
	close(start)
	workers.Wait()
	if sender.overlap.Load() {
		t.Fatal("SOCKS client stream observed concurrent Send calls")
	}
	if got := sender.calls.Load(); got != sendCount {
		t.Fatalf("serialized send calls = %d, want %d", got, sendCount)
	}
}

func TestSocksFrameQueueRejectsFarFutureAndConflictingDuplicate(t *testing.T) {
	tunnel := newUnitSocksTunnel(21, "session")
	ready := &sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 0, Data: []byte("ready")}
	if err := tunnel.AdmitToImplant(ready); err != nil {
		t.Fatalf("admit ready frame: %v", err)
	}
	if err := tunnel.AdmitToImplant(ready); err != nil {
		t.Fatalf("byte-identical ready duplicate = %v, want nil", err)
	}
	if err := tunnel.AdmitToImplant(&sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 0, Data: []byte("ready conflict")}); !errors.Is(err, ErrSocksSequenceConflict) {
		t.Fatalf("conflicting ready duplicate = %v, want ErrSocksSequenceConflict", err)
	}
	if err := tunnel.ProcessDataFromImplant(&sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: 1,
		Data:     []byte("future"),
	}); err != nil {
		t.Fatalf("admit future frame: %v", err)
	}
	if err := tunnel.ProcessDataFromImplant(&sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: 1,
		Data:     []byte("future"),
	}); err != nil {
		t.Fatalf("byte-identical duplicate = %v, want nil", err)
	}
	if err := tunnel.ProcessDataFromImplant(&sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: 1,
		Data:     []byte("conflict"),
	}); !errors.Is(err, ErrSocksSequenceConflict) {
		t.Fatalf("conflicting duplicate = %v, want ErrSocksSequenceConflict", err)
	}
	if err := tunnel.ProcessDataFromImplant(&sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: maxSocksPendingFrames,
	}); !errors.Is(err, ErrTunnelSequenceWindow) {
		t.Fatalf("far-future frame = %v, want ErrTunnelSequenceWindow", err)
	}
	_, frames, size := tunnel.fromImplant.snapshot()
	if frames != 1 || size != len("future") {
		t.Fatalf("duplicate/invalid frames changed reservation: frames=%d bytes=%d", frames, size)
	}
}

func TestSocksFrameQueueIgnoresStaleReplayAndDropsRequestMetadata(t *testing.T) {
	tunnel := newUnitSocksTunnel(22, "session")
	frame := &sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Data:     []byte("payload"),
		Request:  &commonpb.Request{SessionID: "credential-bearing-metadata"},
	}
	if err := tunnel.AdmitToImplant(frame); err != nil {
		t.Fatalf("admit operator frame: %v", err)
	}
	ordered := <-tunnel.ToImplant()
	if ordered.Request != nil {
		t.Fatalf("ordered frame retained request metadata: %+v", ordered.Request)
	}
	if ordered.TunnelID != tunnel.ID || ordered.Sequence != 0 || string(ordered.Data) != "payload" {
		t.Fatalf("ordered frame = %+v", ordered)
	}
	tunnel.CompleteToImplant(ordered)

	if err := tunnel.AdmitToImplant(&sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Data:     []byte("different stale payload"),
	}); err != nil {
		t.Fatalf("stale replay = %v, want nil", err)
	}
	select {
	case replay := <-tunnel.ToImplant():
		t.Fatalf("stale replay was queued: %+v", replay)
	default:
	}
	_, frames, size := tunnel.toImplant.snapshot()
	if frames != 0 || size != 0 {
		t.Fatalf("stale replay retained reservation: frames=%d bytes=%d", frames, size)
	}
}

func TestSocksFrameQueueCompletionUsesAdmissionSize(t *testing.T) {
	queue := newSocksFrameQueue(1, 4)
	if err := queue.admit(22, &sliverpb.SocksData{Data: []byte("four")}); err != nil {
		t.Fatalf("admit frame: %v", err)
	}
	frame := <-queue.ready
	frame.Data = frame.Data[:1]
	queue.complete(frame)
	if _, frames, size := queue.snapshot(); frames != 0 || size != 0 {
		t.Fatalf("mutated completed frame retained reservation: frames=%d bytes=%d", frames, size)
	}
}

func TestSocksFrameQueueBoundsFrameCountAndBytes(t *testing.T) {
	if MaxSocksFrameBytes != 64*1024 || maxSocksPendingFrames != 128 || maxSocksPendingBytes != 8*1024*1024 {
		t.Fatalf("SOCKS framing limits = payload:%d frames:%d bytes:%d", MaxSocksFrameBytes, maxSocksPendingFrames, maxSocksPendingBytes)
	}
	tunnel := newUnitSocksTunnel(23, "session")
	// Retain half the window in the ready channel and half behind a sequence
	// gap, proving the admission limit is combined rather than per container.
	for sequence := uint64(0); sequence < uint64(maxSocksPendingFrames/2); sequence++ {
		if err := tunnel.AdmitToImplant(&sliverpb.SocksData{
			TunnelID: tunnel.ID,
			Sequence: sequence,
		}); err != nil {
			t.Fatalf("admit frame %d: %v", sequence, err)
		}
	}
	for sequence := uint64(maxSocksPendingFrames/2 + 1); sequence <= uint64(maxSocksPendingFrames); sequence++ {
		if err := tunnel.AdmitToImplant(&sliverpb.SocksData{
			TunnelID: tunnel.ID,
			Sequence: sequence,
		}); err != nil {
			t.Fatalf("admit future frame %d: %v", sequence, err)
		}
	}
	if err := tunnel.AdmitToImplant(&sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: uint64(maxSocksPendingFrames / 2),
	}); !errors.Is(err, ErrTunnelIngressLimit) {
		t.Fatalf("129th admitted frame = %v, want ErrTunnelIngressLimit", err)
	}

	byteLimited := newSocksFrameQueue(maxSocksPendingFrames, 3)
	if err := byteLimited.admit(tunnel.ID, &sliverpb.SocksData{Sequence: 1, Data: []byte("four")}); !errors.Is(err, ErrTunnelPendingBytes) {
		t.Fatalf("frame beyond byte budget = %v, want ErrTunnelPendingBytes", err)
	}
	_, frames, size := byteLimited.snapshot()
	if frames != 0 || size != 0 {
		t.Fatalf("rejected byte-budget frame retained reservation: frames=%d bytes=%d", frames, size)
	}
}

//nolint:gocyclo // Admission, wakeup, and accounting assertions are one lifecycle scenario.
func TestTcpTunnelAdmitToImplantContextWaitsForCapacityAndPreservesAccounting(t *testing.T) {
	tunnel := newUnitSocksTunnel(2301, "session")
	tunnel.toImplant = newSocksFrameQueue(2, 5)
	t.Cleanup(func() { tunnel.close() })

	if err := tunnel.AdmitToImplant(&sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 0, Data: []byte("aa")}); err != nil {
		t.Fatalf("admit first frame: %v", err)
	}
	if err := tunnel.AdmitToImplant(&sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 1, Data: []byte("bbb")}); err != nil {
		t.Fatalf("admit second frame: %v", err)
	}
	first := <-tunnel.ToImplant()
	second := <-tunnel.ToImplant()

	result := make(chan error, 1)
	go func() {
		result <- tunnel.AdmitToImplantContext(context.Background(), &sliverpb.SocksData{
			TunnelID: tunnel.ID,
			Sequence: 2,
			Data:     []byte("c"),
		})
	}()

	select {
	case err := <-result:
		t.Fatalf("capacity waiter returned before a reservation was released: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	if next, frames, size := tunnel.toImplant.snapshot(); next != 2 || frames != 2 || size != 5 {
		t.Fatalf("full queue snapshot = next:%d frames:%d bytes:%d, want 2/2/5", next, frames, size)
	}

	tunnel.CompleteToImplant(first)
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("capacity waiter after release: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("released reservation did not wake capacity waiter")
	}
	third := <-tunnel.ToImplant()
	if third.Sequence != 2 || string(third.Data) != "c" {
		t.Fatalf("admitted frame after release = %+v", third)
	}
	if next, frames, size := tunnel.toImplant.snapshot(); next != 3 || frames != 2 || size != 4 {
		t.Fatalf("refilled queue snapshot = next:%d frames:%d bytes:%d, want 3/2/4", next, frames, size)
	}

	tunnel.CompleteToImplant(second)
	tunnel.CompleteToImplant(third)
	if next, frames, size := tunnel.toImplant.snapshot(); next != 3 || frames != 0 || size != 0 {
		t.Fatalf("drained queue snapshot = next:%d frames:%d bytes:%d, want 3/0/0", next, frames, size)
	}
}

//nolint:gocyclo // Context and tunnel cancellation share the same bounded-wait contract.
func TestTcpTunnelAdmitToImplantContextCancellationKeepsBounds(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		tunnel := newUnitSocksTunnel(2302, "session")
		tunnel.toImplant = newSocksFrameQueue(2, 2)
		t.Cleanup(func() { tunnel.close() })
		for sequence := uint64(0); sequence < 2; sequence++ {
			if err := tunnel.AdmitToImplant(&sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: sequence, Data: []byte("x")}); err != nil {
				t.Fatalf("admit frame %d: %v", sequence, err)
			}
		}

		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() {
			result <- tunnel.AdmitToImplantContext(ctx, &sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 2, Data: []byte("x")})
		}()
		select {
		case err := <-result:
			t.Fatalf("capacity waiter returned before cancellation: %v", err)
		case <-time.After(25 * time.Millisecond):
		}
		cancel()
		select {
		case err := <-result:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled capacity waiter = %v, want %v", err, context.Canceled)
			}
		case <-time.After(time.Second):
			t.Fatal("context cancellation did not wake capacity waiter")
		}
		if next, frames, size := tunnel.toImplant.snapshot(); next != 2 || frames != 2 || size != 2 {
			t.Fatalf("canceled queue snapshot = next:%d frames:%d bytes:%d, want 2/2/2", next, frames, size)
		}
	})

	t.Run("tunnel cancellation", func(t *testing.T) {
		tunnel := newUnitSocksTunnel(2303, "session")
		tunnel.toImplant = newSocksFrameQueue(2, 2)
		for sequence := uint64(0); sequence < 2; sequence++ {
			if err := tunnel.AdmitToImplant(&sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: sequence, Data: []byte("x")}); err != nil {
				t.Fatalf("admit frame %d: %v", sequence, err)
			}
		}

		result := make(chan error, 1)
		go func() {
			result <- tunnel.AdmitToImplantContext(context.Background(), &sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 2, Data: []byte("x")})
		}()
		select {
		case err := <-result:
			t.Fatalf("capacity waiter returned before tunnel cancellation: %v", err)
		case <-time.After(25 * time.Millisecond):
		}
		if !tunnel.close() {
			t.Fatal("close saturated tunnel")
		}
		select {
		case err := <-result:
			if !errors.Is(err, ErrTunnelClosed) {
				t.Fatalf("closed-tunnel capacity waiter = %v, want %v", err, ErrTunnelClosed)
			}
		case <-time.After(time.Second):
			t.Fatal("tunnel cancellation did not wake capacity waiter")
		}
		if _, frames, size := tunnel.toImplant.snapshot(); frames != 0 || size != 0 {
			t.Fatalf("closed queue retained frames:%d bytes:%d", frames, size)
		}
	})
}

func TestTcpTunnelAdmitToImplantContextKeepsProtocolErrorsFailFast(t *testing.T) {
	tunnel := newUnitSocksTunnel(2304, "session")
	tunnel.toImplant = newSocksFrameQueue(2, 2)
	t.Cleanup(func() { tunnel.close() })
	for sequence := uint64(0); sequence < 2; sequence++ {
		if err := tunnel.AdmitToImplant(&sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: sequence, Data: []byte("x")}); err != nil {
			t.Fatalf("admit frame %d: %v", sequence, err)
		}
	}

	tests := []struct {
		name  string
		frame *sliverpb.SocksData
		want  error
	}{
		{name: "conflicting duplicate", frame: &sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 0, Data: []byte("y")}, want: ErrSocksSequenceConflict},
		{name: "outside sequence window", frame: &sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 4, Data: []byte("x")}, want: ErrTunnelSequenceWindow},
		{name: "oversized frame", frame: &sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 2, Data: make([]byte, MaxSocksFrameBytes+1)}, want: ErrTunnelFrameTooLarge},
		{name: "terminal payload", frame: &sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 2, Data: []byte("x"), CloseConn: true}, want: ErrSocksTerminalPayload},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := tunnel.AdmitToImplantContext(ctx, test.frame); !errors.Is(err, test.want) {
				t.Fatalf("admission error = %v, want %v", err, test.want)
			}
		})
	}

	impossible := newUnitSocksTunnel(2305, "session")
	impossible.toImplant = newSocksFrameQueue(2, 1)
	t.Cleanup(func() { impossible.close() })
	if err := impossible.AdmitToImplantContext(context.Background(), &sliverpb.SocksData{
		TunnelID: impossible.ID,
		Sequence: 0,
		Data:     []byte("xx"),
	}); !errors.Is(err, ErrTunnelPendingBytes) {
		t.Fatalf("permanently over-budget frame = %v, want %v", err, ErrTunnelPendingBytes)
	}
}

func TestSocksFrameQueueRejectsOversizedPayloadAndCloseReleasesState(t *testing.T) {
	tunnel := newUnitSocksTunnel(24, "session")
	if err := tunnel.ProcessDataFromImplant(&sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Data:     make([]byte, MaxSocksFrameBytes+1),
	}); !errors.Is(err, ErrTunnelFrameTooLarge) {
		t.Fatalf("oversized payload = %v, want ErrTunnelFrameTooLarge", err)
	}
	terminalWithData := &sliverpb.SocksData{
		TunnelID:  tunnel.ID,
		Sequence:  0,
		Data:      []byte("not allowed on terminal"),
		CloseConn: true,
	}
	if err := tunnel.AdmitToImplant(terminalWithData); !errors.Is(err, ErrSocksTerminalPayload) {
		t.Fatalf("operator terminal payload = %v, want ErrSocksTerminalPayload", err)
	}
	if err := tunnel.ProcessDataFromImplant(terminalWithData); !errors.Is(err, ErrSocksTerminalPayload) {
		t.Fatalf("implant terminal payload = %v, want ErrSocksTerminalPayload", err)
	}
	if err := tunnel.ProcessDataFromImplant(&sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 1, Data: []byte("pending")}); err != nil {
		t.Fatalf("admit pending frame: %v", err)
	}
	if err := tunnel.AdmitToImplant(&sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 0, Data: []byte("ready")}); err != nil {
		t.Fatalf("admit ready frame: %v", err)
	}
	if !tunnel.close() {
		t.Fatal("close tunnel")
	}
	for name, queue := range map[string]*socksFrameQueue{"to implant": tunnel.toImplant, "from implant": tunnel.fromImplant} {
		_, frames, size := queue.snapshot()
		if frames != 0 || size != 0 {
			t.Fatalf("%s queue retained frames=%d bytes=%d", name, frames, size)
		}
	}
}

func TestSocksOrderingStateIsScopedToExactTunnelGeneration(t *testing.T) {
	old := newUnitSocksTunnel(25, "session")
	if err := old.ProcessDataFromImplant(&sliverpb.SocksData{
		TunnelID: old.ID,
		Sequence: 1,
		Data:     []byte("old pending"),
	}); err != nil {
		t.Fatalf("admit old generation frame: %v", err)
	}
	old.close()

	current := newUnitSocksTunnel(old.ID, old.SessionID)
	if err := current.ProcessDataFromImplant(&sliverpb.SocksData{
		TunnelID: current.ID,
		Sequence: 0,
		Data:     []byte("current"),
	}); err != nil {
		t.Fatalf("admit current generation frame: %v", err)
	}
	select {
	case frame := <-current.FromImplant():
		current.CompleteFromImplant(frame)
		if string(frame.Data) != "current" || frame.Sequence != 0 {
			t.Fatalf("current generation frame = %+v", frame)
		}
	case <-time.After(time.Second):
		t.Fatal("old generation ordering state blocked current generation")
	}
	if err := old.ProcessDataFromImplant(&sliverpb.SocksData{TunnelID: old.ID, Sequence: 0}); !errors.Is(err, ErrTunnelClosed) {
		t.Fatalf("old generation accepted delayed frame: %v", err)
	}
}

// FuzzSocksTunnelAdmissionState exercises only the server framing actor: it
// performs no network operations or untrusted dials and is safe to run with
// `go test -fuzz=FuzzSocksTunnelAdmissionState ./server/core`.
//
//nolint:gocyclo // The fuzz target validates all admission-state invariants together.
func FuzzSocksTunnelAdmissionState(f *testing.F) {
	f.Add([]byte{0x00, 0x00, 'a', 0x01, 0x01, 'b', 0x10, 0x00, 'c'})
	f.Add([]byte{0x00, 0x01, 'a', 0x00, 0x01, 'b'}) // conflicting pending duplicate
	f.Add([]byte{0x00, 0x80, 'a'})                  // edge of the future window
	// Admission rejects this immutable probe before retaining or modifying it.
	// Reuse it so every fuzz execution does not allocate another 64 KiB buffer.
	oversized := make([]byte, MaxSocksFrameBytes+1)
	f.Fuzz(func(t *testing.T, operations []byte) {
		tunnel := newUnitSocksTunnel(0xfeed, "fuzz-session")
		if len(operations) > 3*maxSocksPendingFrames {
			operations = operations[:3*maxSocksPendingFrames]
		}
		for offset := 0; offset+2 < len(operations); offset += 3 {
			flags := operations[offset]
			sequence := uint64(operations[offset+1])
			payload := []byte{operations[offset+2]}
			if flags&0x40 != 0 {
				payload = oversized
			}
			frame := &sliverpb.SocksData{
				TunnelID:  tunnel.ID,
				Sequence:  sequence,
				Data:      payload,
				CloseConn: flags&0x20 != 0,
			}
			queue := tunnel.toImplant
			var err error
			if flags&0x80 == 0 {
				err = tunnel.AdmitToImplant(frame)
			} else {
				queue = tunnel.fromImplant
				err = tunnel.ProcessDataFromImplant(frame)
			}
			if len(payload) > MaxSocksFrameBytes && !errors.Is(err, ErrTunnelFrameTooLarge) {
				t.Fatalf("oversized fuzz frame error = %v, want ErrTunnelFrameTooLarge", err)
			}
			_, frames, size := queue.snapshot()
			if frames < 0 || frames > maxSocksPendingFrames || size < 0 || size > maxSocksPendingBytes {
				t.Fatalf("unbounded queue state: frames=%d bytes=%d err=%v", frames, size, err)
			}
			if flags&0x10 != 0 {
				select {
				case ready := <-queue.ready:
					queue.complete(ready)
				default:
				}
			}
		}
		if !tunnel.close() {
			t.Fatal("fuzz tunnel did not close")
		}
		for _, queue := range []*socksFrameQueue{tunnel.toImplant, tunnel.fromImplant} {
			_, frames, size := queue.snapshot()
			if frames != 0 || size != 0 {
				t.Fatalf("closed fuzz queue retained frames=%d bytes=%d", frames, size)
			}
		}
		if err := tunnel.AdmitToImplant(&sliverpb.SocksData{}); !errors.Is(err, ErrTunnelClosed) {
			t.Fatalf("closed fuzz tunnel admission = %v, want ErrTunnelClosed", err)
		}
	})
}
