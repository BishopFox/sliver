package tunnel_handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/things-go/go-socks5"
	"github.com/things-go/go-socks5/statute"
	"google.golang.org/protobuf/proto"
)

func newSocksTunnelState(onClose ...func()) *socksTunnelState {
	var callback func()
	if len(onClose) > 0 {
		callback = onClose[0]
	}
	return newOwnedSocksTunnelState(nil, callback)
}

func newOwnedSocksTunnelState(ownerDone <-chan struct{}, onClose func()) *socksTunnelState {
	return newOwnedSocksTunnelStateWithWindow(ownerDone, onClose, socksTunnelCloseReorderWindow)
}

func newOwnedSocksTunnelStateWithWindow(ownerDone <-chan struct{}, onClose func(), closeWindow time.Duration) *socksTunnelState {
	tunnel := newUnstartedSocksTunnelState(ownerDone, onClose, closeWindow)
	tunnel.start()
	return tunnel
}

func newSocksServer(username string, password string, owner ...<-chan struct{}) *socks5.Server {
	var ownerDone <-chan struct{}
	if len(owner) > 0 {
		ownerDone = owner[0]
	}
	return newSocksServerWithLifecycle(username, password, ownerDone, nil)
}

func (s *socksTunnelState) isFlowControlEnabled() bool {
	if s == nil {
		return false
	}
	s.flowMutex.Lock()
	defer s.flowMutex.Unlock()
	return s.flowEnabled
}

func (s *socksTunnelPool) loadOrCreate(tunnelID uint64, owner ...<-chan struct{}) (*socksTunnelState, bool) {
	var ownerDone <-chan struct{}
	if len(owner) > 0 {
		ownerDone = owner[0]
	}
	return s.loadOrCreateOwned(tunnelID, ownerDone, nil)
}

func (s *socksTunnelPool) close(tunnelID uint64) bool {
	s.mutex.Lock()
	tunnel, ok := s.tunnels[tunnelID]
	s.mutex.Unlock()
	if !ok {
		return false
	}
	return tunnel.close()
}

func (s *socksTunnelState) submit(frame socksTunnelFrame) bool {
	accepted, _ := s.submitForServer(frame)
	return accepted
}

func (s *socksTunnelState) read() ([]byte, error) {
	frame, err := s.readFrame()
	if err != nil {
		return nil, err
	}
	s.release(frame)
	return frame.data, nil
}

func FuzzSocks5ParseRequest(f *testing.F) {
	f.Add([]byte{0x05, statute.CommandConnect, 0x00, statute.ATYPIPv4, 127, 0, 0, 1, 0, 80})
	f.Add([]byte{0x05, statute.CommandConnect, 0x00, statute.ATYPDomain, 9, 'l', 'o', 'c', 'a', 'l', 'h', 'o', 's', 't', 0, 80})
	f.Add([]byte{0x05})
	f.Fuzz(func(_ *testing.T, data []byte) {
		if len(data) > socksTunnelMaxRawFrameBytes {
			return
		}
		_, _ = socks5.ParseRequest(bytes.NewReader(data))
	})
}

func socksRequestEnvelope(t *testing.T, data *sliverpb.SocksData) *sliverpb.Envelope {
	t.Helper()
	encoded, err := proto.Marshal(data)
	if err != nil {
		t.Fatalf("marshal SOCKS request: %v", err)
	}
	return &sliverpb.Envelope{Type: sliverpb.MsgSocksData, Data: encoded}
}

func TestSocksReqHandlerStartsAuthenticatedServerOnlyFromSequenceZero(t *testing.T) {
	const tunnelID = uint64(0xfeed1001)
	const username = "operator"
	const password = "correct-horse"
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 8)}
	t.Cleanup(connection.Cleanup)

	authRequest := append([]byte{0x01, byte(len(username))}, []byte(username)...)
	authRequest = append(authRequest, byte(len(password)))
	authRequest = append(authRequest, []byte(password)...)
	sequenceOne := socksRequestEnvelope(t, &sliverpb.SocksData{
		TunnelID: tunnelID,
		Sequence: 1,
		Data:     authRequest,
		// Future-frame metadata must not control the server policy.
		Username: "attacker-controlled",
		Password: "attacker-controlled",
	})
	sequenceOneDone := make(chan struct{})
	go func() {
		defer close(sequenceOneDone)
		SocksReqHandler(sequenceOne, connection)
	}()
	sequenceOneReturnedBeforeZero := false
	select {
	case <-sequenceOneDone:
		sequenceOneReturnedBeforeZero = true
	case <-time.After(100 * time.Millisecond):
	}

	tunnel, ok := socksTunnels.get(tunnelID)
	if !ok {
		t.Fatal("future frame did not create sequencing state")
	}
	t.Cleanup(func() { socksTunnels.release(tunnelID, tunnel) })
	tunnel.startMutex.Lock()
	startedFromFuture := tunnel.serverReady || tunnel.serverClaimed
	tunnel.startMutex.Unlock()

	sequenceZero := socksRequestEnvelope(t, &sliverpb.SocksData{
		TunnelID: tunnelID,
		Sequence: 0,
		Data:     []byte{0x05, 0x01, 0x02},
		Username: username,
		Password: password,
	})
	sequenceZeroDone := make(chan struct{})
	go func() {
		defer close(sequenceZeroDone)
		SocksReqHandler(sequenceZero, connection)
	}()
	methodResponse := readSocksTestEnvelope(t, connection.Send)
	if !bytes.Equal(methodResponse, []byte{0x05, 0x02}) {
		t.Fatalf("authentication method response = %x, want 0502", methodResponse)
	}
	authResponse := readSocksTestEnvelope(t, connection.Send)
	if !bytes.Equal(authResponse, []byte{0x01, 0x00}) {
		t.Fatalf("authentication response = %x, want 0100", authResponse)
	}

	SocksReqHandler(socksRequestEnvelope(t, &sliverpb.SocksData{
		TunnelID:  tunnelID,
		Sequence:  2,
		CloseConn: true,
	}), connection)
	select {
	case <-sequenceZeroDone:
	case <-time.After(2 * time.Second):
		t.Fatal("authenticated SOCKS handler did not stop after terminal")
	}
	select {
	case <-sequenceOneDone:
	case <-time.After(2 * time.Second):
		t.Fatal("future-frame handler did not return")
	}
	if !sequenceOneReturnedBeforeZero {
		t.Fatal("sequence one attempted to start SOCKS before canonical sequence zero")
	}
	if startedFromFuture {
		t.Fatal("future frame claimed SOCKS server authentication policy")
	}
}

func TestSocksReqHandlerIgnoresMetadataOnlyBind(t *testing.T) {
	const tunnelID = uint64(0xfeed1002)
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 1)}
	t.Cleanup(connection.Cleanup)
	SocksReqHandler(socksRequestEnvelope(t, &sliverpb.SocksData{
		TunnelID:     tunnelID,
		Sequence:     ^uint64(0),
		Username:     "not-needed-implant-side",
		Password:     "not-needed-implant-side",
		Capabilities: sliverpb.CapabilitySocksFlowControlV1,
	}), connection)
	if tunnel, ok := socksTunnels.get(tunnelID); ok {
		socksTunnels.release(tunnelID, tunnel)
		t.Fatal("metadata-only ownership bind created implant SOCKS state")
	}
}

func TestSocksReqHandlerAckDoesNotCreateTunnel(t *testing.T) {
	pool := socksTunnelPool{tunnels: map[uint64]*socksTunnelState{}}
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 1)}
	t.Cleanup(connection.Cleanup)

	for _, frame := range []*sliverpb.SocksData{
		{TunnelID: 0xfeed1003, Ack: 1},
		{TunnelID: 0xfeed1004, Ack: 1, Data: []byte("mixed")},
	} {
		handleSocksReq(socksRequestEnvelope(t, frame), connection, &pool)
	}
	if len(pool.tunnels) != 0 {
		t.Fatalf("ACK-only dispatch created %d SOCKS tunnel states", len(pool.tunnels))
	}
}

func TestSocksFlowControlActivatesOnlyFromSequenceZero(t *testing.T) {
	t.Run("data", func(t *testing.T) {
		tunnel := newSocksTunnelState()
		defer tunnel.close()
		if !tunnel.submit(socksTunnelFrame{
			sequence:     0,
			data:         []byte{0x05, 0x01, 0x00},
			capabilities: sliverpb.CapabilitySocksFlowControlV1,
		}) {
			t.Fatal("submit flow-control sequence-zero data")
		}
		if !tunnel.isFlowControlEnabled() {
			t.Fatal("sequence-zero capability did not enable flow control")
		}
	})

	t.Run("later-frame", func(t *testing.T) {
		tunnel := newSocksTunnelState()
		defer tunnel.close()
		if !tunnel.submit(socksTunnelFrame{sequence: 0, data: []byte{0x05, 0x01, 0x00}}) {
			t.Fatal("submit legacy sequence-zero data")
		}
		if !tunnel.submit(socksTunnelFrame{
			sequence:     1,
			data:         []byte("later"),
			capabilities: sliverpb.CapabilitySocksFlowControlV1,
		}) {
			t.Fatal("submit capability on later frame")
		}
		if tunnel.isFlowControlEnabled() {
			t.Fatal("later-frame capability enabled flow control")
		}
	})

	t.Run("empty-terminal", func(t *testing.T) {
		tunnel := newSocksTunnelState()
		if !tunnel.submit(socksTunnelFrame{
			sequence:     0,
			close:        true,
			capabilities: sliverpb.CapabilitySocksFlowControlV1,
		}) {
			t.Fatal("submit flow-control sequence-zero terminal")
		}
		select {
		case <-tunnel.done:
		case <-time.After(2 * time.Second):
			t.Fatal("empty negotiated terminal did not close tunnel")
		}
		if !tunnel.isFlowControlEnabled() {
			t.Fatal("sequence-zero terminal capability did not enable flow control")
		}
	})
}

func TestSocksReqHandlerPoolSaturationSendsExactTerminal(t *testing.T) {
	pool := socksTunnelPool{
		tunnels:           map[uint64]*socksTunnelState{},
		tombstoneDuration: time.Hour,
		maxActive:         1,
		maxRetained:       1,
		maxPendingBytes:   socksTunnelMaxPendingTotal,
		scheduleRemoval:   func(time.Duration, func()) {},
	}
	occupied, created := pool.loadOrCreate(0xfeed2000)
	if occupied == nil || !created {
		t.Fatal("create state that saturates SOCKS pool")
	}
	defer occupied.close()

	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 1)}
	t.Cleanup(connection.Cleanup)
	const rejectedTunnelID = uint64(0xfeed2001)
	handleSocksReq(socksRequestEnvelope(t, &sliverpb.SocksData{
		TunnelID: rejectedTunnelID,
		Sequence: 0,
		Data:     []byte{0x05, 0x01, 0x00},
	}), connection, &pool)

	terminal := readSocksTestFrame(t, connection.Send)
	if terminal.TunnelID != rejectedTunnelID || terminal.Sequence != 0 || !terminal.CloseConn || len(terminal.Data) != 0 {
		t.Fatalf("saturation terminal = tunnel %d sequence %d close %t data %x", terminal.TunnelID, terminal.Sequence, terminal.CloseConn, terminal.Data)
	}
}

func TestSocksReqHandlerBoundsBlockedPoolRejections(t *testing.T) {
	pool := socksTunnelPool{
		tunnels:           map[uint64]*socksTunnelState{},
		tombstoneDuration: time.Hour,
		maxActive:         1,
		maxRetained:       1,
		maxPendingBytes:   socksTunnelMaxPendingTotal,
		rejectionSlots:    make(chan struct{}, 2),
		scheduleRemoval:   func(time.Duration, func()) {},
	}
	occupied, created := pool.loadOrCreate(0xfeed2100)
	if occupied == nil || !created {
		t.Fatal("create state that saturates SOCKS pool")
	}
	defer occupied.close()

	connection := &transports.Connection{IsOpen: true, Send: make(chan *sliverpb.Envelope)}
	returned := make(chan struct{}, 3)
	for offset := uint64(0); offset < 2; offset++ {
		envelope := socksRequestEnvelope(t, &sliverpb.SocksData{
			TunnelID: 0xfeed2101 + offset,
			Sequence: 0,
			Data:     []byte{0x05, 0x01, 0x00},
		})
		go func(request *sliverpb.Envelope) {
			handleSocksReq(request, connection, &pool)
			returned <- struct{}{}
		}(envelope)
	}

	deadline := time.Now().Add(time.Second)
	for len(pool.rejectionSlots) != cap(pool.rejectionSlots) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := len(pool.rejectionSlots); got != cap(pool.rejectionSlots) {
		t.Fatalf("blocked rejection writers = %d, want %d", got, cap(pool.rejectionSlots))
	}

	excess := socksRequestEnvelope(t, &sliverpb.SocksData{
		TunnelID: 0xfeed2103,
		Sequence: 0,
		Data:     []byte{0x05, 0x01, 0x00},
	})
	go func() {
		handleSocksReq(excess, connection, &pool)
		returned <- struct{}{}
	}()
	select {
	case <-connection.Done():
	case <-time.After(time.Second):
		t.Fatal("excess blocked rejection did not fail the owning C2")
	}
	for index := 0; index < 3; index++ {
		select {
		case <-returned:
		case <-time.After(time.Second):
			t.Fatal("bounded rejection handler remained blocked after C2 cleanup")
		}
	}
	if connection.IsOpen {
		t.Fatal("rejection saturation left owning C2 marked open")
	}
}

func TestSocksReqHandlerAdmissionSaturationSendsExactTerminal(t *testing.T) {
	pool := socksTunnelPool{
		tunnels:           map[uint64]*socksTunnelState{},
		tombstoneDuration: 0,
		maxActive:         1,
		maxRetained:       1,
		maxPendingBytes:   1,
	}
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 1)}
	t.Cleanup(connection.Cleanup)
	const tunnelID = uint64(0xfeed2002)

	handleSocksReq(socksRequestEnvelope(t, &sliverpb.SocksData{
		TunnelID: tunnelID,
		Sequence: 1,
		Data:     []byte("a"),
	}), connection, &pool)
	handleSocksReq(socksRequestEnvelope(t, &sliverpb.SocksData{
		TunnelID: tunnelID,
		Sequence: 2,
		Data:     []byte("b"),
	}), connection, &pool)

	terminal := readSocksTestFrame(t, connection.Send)
	if terminal.TunnelID != tunnelID || terminal.Sequence != 0 || !terminal.CloseConn || len(terminal.Data) != 0 {
		t.Fatalf("admission terminal = tunnel %d sequence %d close %t data %x", terminal.TunnelID, terminal.Sequence, terminal.CloseConn, terminal.Data)
	}
	select {
	case extra := <-connection.Send:
		t.Fatalf("admission rejection emitted duplicate terminal %+v", extra)
	default:
	}
}

func TestSocksReqHandlerTerminalFailureFailsOwnerConnection(t *testing.T) {
	pool := socksTunnelPool{
		tunnels:           map[uint64]*socksTunnelState{},
		tombstoneDuration: time.Hour,
		maxActive:         1,
		maxRetained:       1,
		maxPendingBytes:   socksTunnelMaxPendingTotal,
		scheduleRemoval:   func(time.Duration, func()) {},
	}
	occupied, created := pool.loadOrCreate(0xfeed2003)
	if occupied == nil || !created {
		t.Fatal("create state that saturates SOCKS pool")
	}
	defer occupied.close()

	connection := &transports.Connection{IsOpen: true}
	done := connection.Done()
	handleSocksReq(socksRequestEnvelope(t, &sliverpb.SocksData{
		TunnelID: 0xfeed2004,
		Sequence: 0,
		Data:     []byte{0x05, 0x01, 0x00},
	}), connection, &pool)
	select {
	case <-done:
	default:
		t.Fatal("failed saturation terminal left owner connection live")
	}
	if connection.IsOpen {
		t.Fatal("failed saturation terminal did not mark owner connection closed")
	}
}

func TestSocksTunnelStateReadClosedReturnsEOF(t *testing.T) {
	tunnel := newSocksTunnelState()
	if !tunnel.close() {
		t.Fatal("first explicit SOCKS tunnel close returned false")
	}

	data, err := tunnel.read()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("read closed SOCKS tunnel error = %v, want io.EOF", err)
	}
	if data != nil {
		t.Fatalf("read closed SOCKS tunnel data = %x, want nil", data)
	}
}

func TestSocksTunnelPoolCloseIsExplicitAndIdempotent(t *testing.T) {
	pool := socksTunnelPool{
		tunnels:           map[uint64]*socksTunnelState{},
		tombstoneDuration: time.Hour,
		scheduleRemoval:   func(time.Duration, func()) {},
	}
	tunnel, created := pool.loadOrCreate(9)
	if !created {
		t.Fatal("new SOCKS tunnel state was not created")
	}

	if !pool.close(9) {
		t.Fatal("first explicit SOCKS tunnel close returned false")
	}
	if pool.close(9) {
		t.Fatal("second explicit SOCKS tunnel close returned true")
	}
	if _, err := tunnel.read(); !errors.Is(err, io.EOF) {
		t.Fatalf("read explicitly closed SOCKS tunnel error = %v, want io.EOF", err)
	}
	pool.release(9, tunnel)
	loaded, ok := pool.get(9)
	if !ok || loaded != tunnel || !loaded.isClosed() {
		t.Fatal("released SOCKS tunnel lost its closed replay tombstone")
	}
	pool.mutex.Lock()
	active := pool.active
	pool.mutex.Unlock()
	if active != 0 {
		t.Fatalf("released SOCKS tunnel left %d active states, want zero", active)
	}
}

func TestSocksTunnelConcurrentDeliverAndClose(t *testing.T) {
	const iterations = 1000
	for iteration := 0; iteration < iterations; iteration++ {
		pool := socksTunnelPool{tunnels: map[uint64]*socksTunnelState{}}
		tunnel, created := pool.loadOrCreate(uint64(iteration + 1))
		if !created {
			t.Fatalf("iteration %d: new SOCKS tunnel state was not created", iteration)
		}

		start := make(chan struct{})
		delivered := make(chan bool, 1)
		closed := make(chan bool, 1)
		go func() {
			<-start
			delivered <- tunnel.submit(socksTunnelFrame{data: []byte("race-payload")})
		}()
		go func() {
			<-start
			closed <- pool.close(uint64(iteration + 1))
		}()
		close(start)

		select {
		case <-delivered:
		case <-time.After(2 * time.Second):
			t.Fatalf("iteration %d: delivery blocked while close raced", iteration)
		}
		select {
		case firstClose := <-closed:
			if !firstClose {
				t.Fatalf("iteration %d: concurrent explicit close returned false", iteration)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("iteration %d: explicit close blocked while delivery raced", iteration)
		}
		if _, err := tunnel.read(); !errors.Is(err, io.EOF) {
			t.Fatalf("iteration %d: post-close read error = %v, want io.EOF", iteration, err)
		}
		pool.release(uint64(iteration+1), tunnel)
	}
}

func TestSocksTunnelStateOrdersDataBeforeTerminalClose(t *testing.T) {
	tunnel := newSocksTunnelState()
	defer tunnel.close()

	// Model the runner dispatching adjacent envelopes in adversarial goroutine
	// order: the second data frame and terminal arrive before the first frame.
	if !tunnel.submit(socksTunnelFrame{sequence: 1, data: []byte("one")}) {
		t.Fatal("submit sequence 1 failed")
	}
	if !tunnel.submit(socksTunnelFrame{sequence: 2, close: true}) {
		t.Fatal("submit terminal sequence 2 failed")
	}
	if !tunnel.submit(socksTunnelFrame{sequence: 0, data: []byte("zero")}) {
		t.Fatal("submit sequence 0 failed")
	}
	waitForSocksTunnelLifecycle(t, tunnel, socksTunnelGraceful)

	for index, want := range []string{"zero", "one"} {
		result := make(chan struct {
			data []byte
			err  error
		}, 1)
		go func() {
			data, err := tunnel.read()
			result <- struct {
				data []byte
				err  error
			}{data: data, err: err}
		}()
		select {
		case got := <-result:
			if got.err != nil || string(got.data) != want {
				t.Fatalf("ordered read %d = %q, %v, want %q, nil", index, got.data, got.err, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("ordered read %d timed out", index)
		}
	}
	if _, err := tunnel.read(); !errors.Is(err, io.EOF) {
		t.Fatalf("read after ordered terminal error = %v, want io.EOF", err)
	}
	if tunnel.lifecycle.Load() != socksTunnelGraceful {
		t.Fatal("ordered terminal became abortive close before the reader drained")
	}
}

func TestSocksTunnelLegacySequenceZeroTerminalAfterData(t *testing.T) {
	tunnel := newOwnedSocksTunnelStateWithWindow(nil, nil, 25*time.Millisecond)
	defer tunnel.close()
	if !tunnel.submit(socksTunnelFrame{sequence: 0, data: []byte("legacy")}) {
		t.Fatal("submit legacy SOCKS payload")
	}
	if data, err := tunnel.read(); err != nil || string(data) != "legacy" {
		t.Fatalf("read legacy SOCKS payload = %q, %v", data, err)
	}
	if !tunnel.submit(socksTunnelFrame{sequence: 0, close: true}) {
		t.Fatal("submit legacy sequence-zero terminal")
	}
	waitForSocksTunnelLifecycle(t, tunnel, socksTunnelGraceful)
	if _, err := tunnel.read(); !errors.Is(err, io.EOF) {
		t.Fatalf("read after legacy terminal error = %v, want io.EOF", err)
	}
}

func TestSocksTunnelLegacyTerminalMayArriveBeforeEarlierDataHandler(t *testing.T) {
	tunnel := newOwnedSocksTunnelStateWithWindow(nil, nil, 250*time.Millisecond)
	defer tunnel.close()
	if !tunnel.submit(socksTunnelFrame{sequence: 0, close: true}) {
		t.Fatal("submit overtaking legacy terminal")
	}
	if !tunnel.submit(socksTunnelFrame{sequence: 0, data: []byte("earlier-data")}) {
		t.Fatal("legacy terminal closed tunnel before earlier data handler arrived")
	}
	if data, err := tunnel.read(); err != nil || string(data) != "earlier-data" {
		t.Fatalf("read earlier legacy data = %q, %v", data, err)
	}
	waitForSocksTunnelLifecycle(t, tunnel, socksTunnelGraceful)
	if _, err := tunnel.read(); !errors.Is(err, io.EOF) {
		t.Fatalf("read after overtaking legacy terminal error = %v, want io.EOF", err)
	}
}

func TestSocksTunnelLegacyTerminalQuietWindowResetsOnProgress(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const closeWindow = 400 * time.Millisecond
		tunnel := newOwnedSocksTunnelStateWithWindow(nil, nil, closeWindow)
		defer tunnel.close()
		if !tunnel.submit(socksTunnelFrame{sequence: 0, close: true}) {
			t.Fatal("submit legacy terminal before delayed data")
		}

		started := time.Now()
		for sequence := uint64(0); sequence < 3; sequence++ {
			// The complete arrival span exceeds closeWindow, but each contiguous
			// advance arrives well inside the renewed quiet period.
			synctest.Sleep(150 * time.Millisecond)
			payload := []byte{byte(sequence)}
			if !tunnel.submit(socksTunnelFrame{sequence: sequence, data: payload}) {
				t.Fatalf("legacy quiet window expired before sequence %d", sequence)
			}
			if data, err := tunnel.read(); err != nil || !bytes.Equal(data, payload) {
				t.Fatalf("legacy progress read %d = %x, %v", sequence, data, err)
			}
		}
		if elapsed := time.Since(started); elapsed <= closeWindow {
			t.Fatalf("legacy progress span = %s, want longer than %s", elapsed, closeWindow)
		}
		synctest.Sleep(closeWindow)
		if lifecycle := tunnel.lifecycle.Load(); lifecycle != socksTunnelGraceful {
			t.Fatalf("SOCKS tunnel lifecycle = %d, want %d", lifecycle, socksTunnelGraceful)
		}
		if _, err := tunnel.read(); !errors.Is(err, io.EOF) {
			t.Fatalf("read after reset legacy terminal error = %v, want io.EOF", err)
		}
	})
}

func TestSocksTunnelGracefulCloseDrainsWholeFrameBeforeEOF(t *testing.T) {
	tunnel := newSocksTunnelState()
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 1)}
	adapter := &socks{
		stream: &sliverpb.SocksData{TunnelID: 0xfeed3001},
		conn:   connection,
		tunnel: tunnel,
	}
	defer tunnel.close()

	payload := bytes.Repeat([]byte{0x5a}, sliverpb.MaxTunnelFrameBytes)
	if !tunnel.submit(socksTunnelFrame{sequence: 0, data: payload}) {
		t.Fatal("submit full-size SOCKS frame")
	}
	if !tunnel.submit(socksTunnelFrame{sequence: 1, close: true}) {
		t.Fatal("submit terminal after full-size SOCKS frame")
	}
	waitForSocksTunnelLifecycle(t, tunnel, socksTunnelGraceful)
	if tunnel.submit(socksTunnelFrame{sequence: 2, data: []byte("late")}) {
		t.Fatal("gracefully closing SOCKS tunnel admitted a late frame")
	}
	if !tunnel.isGraceful() {
		t.Fatal("late frame changed graceful drain into an abort")
	}

	got := make([]byte, len(payload))
	for offset := 0; offset < len(got); {
		end := offset + 32*1024
		if end > len(got) {
			end = len(got)
		}
		count, err := adapter.Read(got[offset:end])
		if err != nil || count != end-offset {
			t.Fatalf("read full-size frame at offset %d = %d, %v", offset, count, err)
		}
		offset += count
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("graceful SOCKS close truncated or changed final frame")
	}
	if count, err := adapter.Read(make([]byte, 1)); count != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("read after drained final frame = %d, %v, want 0, io.EOF", count, err)
	}
	waitForSocksTunnelBudget(t, tunnel, 0)
}

func TestSocksTunnelAbortDropsQueuedInput(t *testing.T) {
	tunnel := newSocksTunnelState()
	if !tunnel.submit(socksTunnelFrame{sequence: 0, data: []byte("attacker-controlled")}) {
		t.Fatal("submit queued input")
	}
	if !tunnel.close() {
		t.Fatal("abort SOCKS tunnel")
	}
	if data, err := tunnel.read(); data != nil || !errors.Is(err, io.EOF) {
		t.Fatalf("read after abort = %q, %v, want nil, io.EOF", data, err)
	}
}

func TestSocksTunnelGracefulDrainWatchdogIsBounded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tunnel := newSocksTunnelState()
		tunnel.drainWindow = 25 * time.Millisecond
		if !tunnel.submit(socksTunnelFrame{sequence: 0, data: []byte("unread")}) {
			t.Fatal("submit unread SOCKS input")
		}
		if !tunnel.submit(socksTunnelFrame{sequence: 1, close: true}) {
			t.Fatal("submit terminal after unread SOCKS input")
		}
		synctest.Wait()
		if lifecycle := tunnel.lifecycle.Load(); lifecycle != socksTunnelGraceful {
			t.Fatalf("SOCKS tunnel lifecycle = %d, want %d", lifecycle, socksTunnelGraceful)
		}
		synctest.Sleep(tunnel.drainWindow)
		select {
		case <-tunnel.done:
		default:
			t.Fatal("graceful drain watchdog did not force-close unread SOCKS input")
		}
	})
}

func TestSocksTunnelHandshakeLeaseLastsUntilTargetDial(t *testing.T) {
	t.Run("incomplete handshake expires", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			tunnel := newSocksTunnelState()
			tunnel.handshakeTimeout = 25 * time.Millisecond
			tunnel.startHandshakeLease()
			synctest.Sleep(tunnel.handshakeTimeout)
			select {
			case <-tunnel.done:
			default:
				t.Fatal("incomplete SOCKS handshake did not expire")
			}
		})
	})

	t.Run("successful target dial releases lease", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			tunnel := newSocksTunnelState()
			defer tunnel.close()
			tunnel.handshakeTimeout = 25 * time.Millisecond
			tunnel.startHandshakeLease()
			tunnel.markEstablished()
			synctest.Sleep(3 * tunnel.handshakeTimeout)
			if tunnel.isClosed() {
				t.Fatal("handshake lease closed an established SOCKS tunnel")
			}
		})
	})
}

func TestSocksTunnelStateTerminalTimerResetsOnSequenceProgress(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const reorderWindow = 250 * time.Millisecond
		tunnel := newOwnedSocksTunnelStateWithWindow(nil, nil, reorderWindow)
		defer tunnel.close()

		if !tunnel.submit(socksTunnelFrame{sequence: 4, close: true}) {
			t.Fatal("submit terminal sequence 4 failed")
		}

		started := time.Now()
		for sequence := byte(0); sequence < 4; sequence++ {
			// Total drain time exceeds one reorder window, but each contiguous
			// frame arrives within the renewed window.
			synctest.Sleep(100 * time.Millisecond)
			if !tunnel.submit(socksTunnelFrame{sequence: uint64(sequence), data: []byte{sequence}}) {
				t.Fatalf("submit slow sequence %d failed", sequence)
			}
			data, err := tunnel.read()
			if err != nil || len(data) != 1 || data[0] != sequence {
				t.Fatalf("slow ordered read %d = %x, %v", sequence, data, err)
			}
		}
		if elapsed := time.Since(started); elapsed <= reorderWindow {
			t.Fatalf("test drained in %s, want longer than reorder window %s", elapsed, reorderWindow)
		}
		if _, err := tunnel.read(); !errors.Is(err, io.EOF) {
			t.Fatalf("read after slow ordered terminal error = %v, want io.EOF", err)
		}
	})
}

func TestSocksTunnelStateAdmissionBoundsBlockedReader(t *testing.T) {
	tunnel := newSocksTunnelState()
	for sequence := uint64(0); sequence < socksTunnelMaxPendingFrames; sequence++ {
		if !tunnel.submit(socksTunnelFrame{sequence: sequence, data: []byte{byte(sequence)}}) {
			t.Fatalf("submit admitted frame %d failed", sequence)
		}
	}
	// No Reader drains sequence zero. The bounded delivery queue may therefore
	// fill, but the reservation covers both actor ingress and queued delivery and
	// rejects the 129th handler before it clones or blocks.
	if tunnel.submit(socksTunnelFrame{
		sequence: socksTunnelMaxPendingFrames,
		data:     []byte("over-capacity"),
	}) {
		t.Fatal("frame beyond total admission limit was accepted")
	}
	select {
	case <-tunnel.done:
	case <-time.After(time.Second):
		t.Fatal("admission violation did not close SOCKS tunnel")
	}
	frames, bytes := socksTunnelReservedBudget(tunnel)
	if frames != 0 || bytes != 0 {
		t.Fatalf("closed tunnel retained reservations: frames=%d bytes=%d", frames, bytes)
	}
}

func TestSocksTunnelPoolBoundsActiveStatesAndAggregatePendingBytes(t *testing.T) {
	pool := socksTunnelPool{
		tunnels:           map[uint64]*socksTunnelState{},
		tombstoneDuration: 0,
		maxActive:         2,
		maxPendingBytes:   3,
	}
	first, created := pool.loadOrCreate(1)
	if first == nil || !created {
		t.Fatal("create first bounded SOCKS state")
	}
	second, created := pool.loadOrCreate(2)
	if second == nil || !created {
		t.Fatal("create second bounded SOCKS state")
	}
	if third, created := pool.loadOrCreate(3); third != nil || created {
		t.Fatal("active-state cap admitted a third SOCKS actor")
	}

	if !first.submit(socksTunnelFrame{sequence: 1, data: []byte("abc")}) {
		t.Fatal("aggregate byte budget rejected its exact capacity")
	}
	if second.submit(socksTunnelFrame{sequence: 1, data: []byte("d")}) {
		t.Fatal("aggregate byte budget admitted an excess byte")
	}
	if !second.isClosed() {
		t.Fatal("aggregate admission rejection did not close its tunnel")
	}
	if !first.close() {
		t.Fatal("close first bounded SOCKS state")
	}
	pool.mutex.Lock()
	pendingBytes := pool.pendingBytes
	active := pool.active
	pool.mutex.Unlock()
	if pendingBytes != 0 || active != 0 {
		t.Fatalf("closed bounded pool state = active %d pending bytes %d, want zero", active, pendingBytes)
	}

	replacement, created := pool.loadOrCreate(3)
	if replacement == nil || !created {
		t.Fatal("released active/byte budget did not admit replacement state")
	}
	replacement.close()
}

func TestSocksTunnelPoolRetainsByteAccountingUntilTombstoneRemoval(t *testing.T) {
	removals := make(chan func(), 3)
	pool := socksTunnelPool{
		tunnels:           map[uint64]*socksTunnelState{},
		tombstoneDuration: time.Hour,
		maxActive:         2,
		maxRetained:       2,
		maxPendingBytes:   4,
		scheduleRemoval: func(_ time.Duration, remove func()) {
			removals <- remove
		},
	}

	first, created := pool.loadOrCreate(101)
	if first == nil || !created {
		t.Fatal("create first retained-byte SOCKS state")
	}
	if !first.submit(socksTunnelFrame{sequence: 1, data: []byte("four")}) {
		t.Fatal("submit retained payload")
	}
	if !first.close() {
		t.Fatal("close retained-payload state")
	}
	removeFirst := <-removals
	pool.mutex.Lock()
	pendingBytes := pool.pendingBytes
	pool.mutex.Unlock()
	if pendingBytes != 4 {
		t.Fatalf("closed tombstone accounts for %d bytes, want 4", pendingBytes)
	}

	second, created := pool.loadOrCreate(102)
	if second == nil || !created {
		t.Fatal("create second retained-byte SOCKS state")
	}
	if second.submit(socksTunnelFrame{sequence: 1, data: []byte("x")}) {
		t.Fatal("aggregate budget ignored bytes retained by closed tombstone")
	}
	removeSecond := <-removals

	removeFirst()
	pool.mutex.Lock()
	pendingBytes = pool.pendingBytes
	pool.mutex.Unlock()
	if pendingBytes != 0 {
		t.Fatalf("removed tombstone retained %d accounted bytes, want zero", pendingBytes)
	}

	replacement, created := pool.loadOrCreate(103)
	if replacement == nil || !created {
		t.Fatal("tombstone removal did not release aggregate byte capacity")
	}
	if !replacement.submit(socksTunnelFrame{sequence: 1, data: []byte("four")}) {
		t.Fatal("released aggregate capacity rejected replacement payload")
	}
	replacement.close()
	removeReplacement := <-removals
	removeSecond()
	removeReplacement()
}

// This scenario checks the coupled active, retained-state, and timer bounds.
//
//nolint:gocyclo
func TestSocksTunnelPoolBoundsRetainedTombstonesAndTimers(t *testing.T) {
	const (
		retainedLimit = 8
		floodCount    = 1024
	)
	var scheduled atomic.Int32
	pool := socksTunnelPool{
		tunnels:           map[uint64]*socksTunnelState{},
		tombstoneDuration: time.Hour,
		closeWindow:       25 * time.Millisecond,
		maxActive:         retainedLimit,
		maxRetained:       retainedLimit,
		maxPendingBytes:   socksTunnelMaxPendingTotal,
		scheduleRemoval: func(time.Duration, func()) {
			scheduled.Add(1)
		},
	}

	tombstones := make(map[uint64]*socksTunnelState, retainedLimit)
	for tunnelID := uint64(1); tunnelID <= retainedLimit; tunnelID++ {
		tunnel, created := pool.loadOrCreate(tunnelID)
		if tunnel == nil || !created {
			t.Fatalf("create retained SOCKS state %d", tunnelID)
		}
		tombstones[tunnelID] = tunnel
		if !tunnel.submit(socksTunnelFrame{sequence: 0, close: true}) {
			t.Fatalf("submit terminal for retained SOCKS state %d", tunnelID)
		}
	}
	for tunnelID, tunnel := range tombstones {
		select {
		case <-tunnel.done:
		case <-time.After(time.Second):
			t.Fatalf("retained SOCKS state %d did not close", tunnelID)
		}
	}

	deadline := time.Now().Add(time.Second)
	for scheduled.Load() != retainedLimit && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := scheduled.Load(); got != retainedLimit {
		t.Fatalf("scheduled tombstone removals = %d, want %d", got, retainedLimit)
	}

	for index := 0; index < floodCount; index++ {
		tunnelID := uint64(retainedLimit + 1 + index)
		if tunnel, created := pool.loadOrCreate(tunnelID); tunnel != nil || created {
			t.Fatalf("retained-state cap admitted flood tunnel %d", tunnelID)
		}
	}
	pool.mutex.Lock()
	retained := len(pool.tunnels)
	active := pool.active
	pool.mutex.Unlock()
	if retained != retainedLimit || active != 0 {
		t.Fatalf("bounded pool retained %d states with %d active, want %d retained and zero active", retained, active, retainedLimit)
	}
	if got := scheduled.Load(); got != retainedLimit {
		t.Fatalf("unique-ID flood scheduled %d tombstone timers, want %d", got, retainedLimit)
	}

	// Saturation must not evict an existing tombstone: a duplicated or delayed
	// terminal for an admitted ID is absorbed by the exact closed generation
	// instead of resurrecting another SOCKS actor.
	for tunnelID, want := range tombstones {
		got, created := pool.loadOrCreate(tunnelID)
		if created || got != want {
			t.Fatalf("late terminal lookup for %d replaced its tombstone", tunnelID)
		}
		if got.submit(socksTunnelFrame{sequence: 0, close: true}) {
			t.Fatalf("closed tombstone %d accepted a delayed terminal", tunnelID)
		}
	}
}

func TestSocksTunnelStateTimesOutAnyPersistentSequenceGap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const reorderWindow = 25 * time.Millisecond
		tunnel := newOwnedSocksTunnelStateWithWindow(nil, nil, reorderWindow)
		if !tunnel.submit(socksTunnelFrame{sequence: 1, data: []byte("future without terminal")}) {
			t.Fatal("submit future SOCKS frame")
		}
		synctest.Sleep(reorderWindow)
		select {
		case <-tunnel.done:
		default:
			t.Fatal("nonterminal sequence gap did not expire")
		}
	})
}

func TestSocksTunnelStateRejectsOversizeAndAmbiguousTerminalFrames(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		frame socksTunnelFrame
	}{
		{
			name:  "oversize payload",
			frame: socksTunnelFrame{data: make([]byte, sliverpb.MaxTunnelFrameBytes+1)},
		},
		{
			name:  "terminal with payload",
			frame: socksTunnelFrame{close: true, data: []byte("ambiguous")},
		},
		{
			name: "oversize canonical username",
			frame: socksTunnelFrame{
				sequence: 0,
				data:     []byte{0x05},
				username: string(make([]byte, socksTunnelMaxCredentialBytes+1)),
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			tunnel := newSocksTunnelState()
			if tunnel.submit(testCase.frame) {
				t.Fatal("invalid frame was accepted")
			}
			select {
			case <-tunnel.done:
			case <-time.After(time.Second):
				t.Fatal("invalid frame did not close SOCKS tunnel")
			}
		})
	}
}

func TestSocksTunnelStateClearsCanonicalCredentialsOnClose(t *testing.T) {
	tunnel := newSocksTunnelState()
	if !tunnel.submit(socksTunnelFrame{
		sequence: 0,
		data:     []byte{0x05},
		username: "operator",
		password: "secret",
	}) {
		t.Fatal("submit canonical credential frame")
	}
	tunnel.startMutex.Lock()
	readyBeforeClose := tunnel.serverReady
	usernameBeforeClose := tunnel.username
	passwordBeforeClose := tunnel.password
	tunnel.startMutex.Unlock()
	if !readyBeforeClose || usernameBeforeClose != "operator" || passwordBeforeClose != "secret" {
		t.Fatal("canonical credentials were not owned by tunnel state")
	}
	if !tunnel.close() {
		t.Fatal("close credential-bearing tunnel")
	}
	tunnel.startMutex.Lock()
	readyAfterClose := tunnel.serverReady
	usernameAfterClose := tunnel.username
	passwordAfterClose := tunnel.password
	tunnel.startMutex.Unlock()
	if readyAfterClose || usernameAfterClose != "" || passwordAfterClose != "" {
		t.Fatal("closed tunnel retained canonical credentials")
	}
}

func TestSocksTunnelStateRejectsFarFutureSequence(t *testing.T) {
	tunnel := newSocksTunnelState()
	if !tunnel.submit(socksTunnelFrame{sequence: ^uint64(0), data: []byte("future")}) {
		t.Fatal("far-future frame was not handed to the sequencing actor")
	}
	select {
	case <-tunnel.done:
	case <-time.After(time.Second):
		t.Fatal("far-future sequence did not close SOCKS tunnel")
	}
}

// The duplicate policy test intentionally covers both accepted and conflicting
// generations as one table-like lifecycle scenario.
//
//nolint:gocyclo
func TestSocksTunnelStateDuplicatePolicyIsBounded(t *testing.T) {
	t.Run("identical duplicate ignored", func(t *testing.T) {
		tunnel := newSocksTunnelState()
		defer tunnel.close()
		if !tunnel.submit(socksTunnelFrame{sequence: 0, data: []byte("zero")}) {
			t.Fatal("submit sequence zero failed")
		}
		if data, err := tunnel.read(); err != nil || string(data) != "zero" {
			t.Fatalf("read sequence zero = %q, %v", data, err)
		}
		if !tunnel.submit(socksTunnelFrame{sequence: 0, data: []byte("zero")}) {
			t.Fatal("submit stale replay failed")
		}
		waitForSocksTunnelBudget(t, tunnel, 0)
		if tunnel.isClosed() {
			t.Fatal("stale replay closed tunnel")
		}
		frame := socksTunnelFrame{sequence: 2, data: []byte("two")}
		if !tunnel.submit(frame) {
			t.Fatal("submit first pending sequence failed")
		}
		if !tunnel.submit(frame) {
			t.Fatal("submit duplicate pending sequence failed")
		}
		waitForSocksTunnelBudget(t, tunnel, 1)
		if tunnel.isClosed() {
			t.Fatal("byte-identical duplicate closed tunnel")
		}
		if !tunnel.submit(socksTunnelFrame{sequence: 1, data: []byte("one")}) {
			t.Fatal("submit missing sequence one failed")
		}
		for _, want := range []string{"one", "two"} {
			if data, err := tunnel.read(); err != nil || string(data) != want {
				t.Fatalf("ordered duplicate-policy read = %q, %v, want %q, nil", data, err, want)
			}
		}
	})

	t.Run("conflicting duplicate closes", func(t *testing.T) {
		tunnel := newSocksTunnelState()
		if !tunnel.submit(socksTunnelFrame{sequence: 0, data: []byte("zero")}) {
			t.Fatal("submit sequence zero failed")
		}
		if _, err := tunnel.read(); err != nil {
			t.Fatal(err)
		}
		if !tunnel.submit(socksTunnelFrame{sequence: 2, data: []byte("first")}) ||
			!tunnel.submit(socksTunnelFrame{sequence: 2, data: []byte("conflict")}) {
			t.Fatal("submit conflicting pending sequence failed")
		}
		select {
		case <-tunnel.done:
		case <-time.After(time.Second):
			t.Fatal("conflicting duplicate did not close tunnel")
		}
	})
}

func socksTunnelReservedBudget(tunnel *socksTunnelState) (int, int) {
	tunnel.budgetMutex.Lock()
	defer tunnel.budgetMutex.Unlock()
	return tunnel.budgetFrames, tunnel.budgetBytes
}

func waitForSocksTunnelBudget(t *testing.T, tunnel *socksTunnelState, wantFrames int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		frames, _ := socksTunnelReservedBudget(tunnel)
		if frames == wantFrames {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("reserved frames = %d, want %d", frames, wantFrames)
		}
		time.Sleep(time.Millisecond)
	}
}

func waitForSocksTunnelLifecycle(t *testing.T, tunnel *socksTunnelState, want uint32) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for tunnel.lifecycle.Load() != want {
		if time.Now().After(deadline) {
			t.Fatalf("SOCKS tunnel lifecycle = %d, want %d", tunnel.lifecycle.Load(), want)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestSocksTunnelPoolCloseBeforeCreateRetainsTombstone(t *testing.T) {
	pool := socksTunnelPool{
		tunnels:           map[uint64]*socksTunnelState{},
		tombstoneDuration: time.Hour,
		closeWindow:       25 * time.Millisecond,
	}
	tunnel, created := pool.loadOrCreate(41)
	if !created {
		t.Fatal("new terminal state was not created")
	}
	if !tunnel.submit(socksTunnelFrame{sequence: 0, close: true}) {
		t.Fatal("submit close-before-data terminal failed")
	}
	deadline := time.Now().Add(2 * time.Second)
	for !tunnel.isClosed() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !tunnel.isClosed() {
		t.Fatal("close-before-data terminal was not applied")
	}

	loaded, newlyCreated := pool.loadOrCreate(41)
	if newlyCreated || loaded != tunnel {
		t.Fatal("closed tunnel tombstone was replaced by a new generation")
	}
	if loaded.submit(socksTunnelFrame{sequence: 0, data: []byte("late")}) {
		t.Fatal("late data was accepted by a closed tunnel tombstone")
	}
	if _, _, claimed := loaded.takeServerCredentials(); claimed {
		t.Fatal("closed tunnel tombstone allowed a SOCKS server to start")
	}
}

func TestSocksTunnelStateClosesWithOwningConnection(t *testing.T) {
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope)}
	pool := socksTunnelPool{
		tunnels:           map[uint64]*socksTunnelState{},
		tombstoneDuration: time.Hour,
		scheduleRemoval:   func(time.Duration, func()) {},
	}
	tunnel, created := pool.loadOrCreate(52, connection.Done())
	if !created {
		t.Fatal("failed to create connection-owned SOCKS tunnel")
	}

	readDone := make(chan error, 1)
	go func() {
		_, err := tunnel.read()
		readDone <- err
	}()
	connection.Cleanup()
	select {
	case err := <-readDone:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("read after owner cleanup error = %v, want io.EOF", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("connection cleanup did not unblock idle SOCKS tunnel read")
	}
	if !tunnel.isClosed() {
		t.Fatal("SOCKS tunnel remained open after owner connection cleanup")
	}
	loaded, created := pool.loadOrCreate(52, connection.Done())
	if created || loaded != tunnel {
		t.Fatal("connection-owned closed tunnel lost its replay tombstone")
	}
}

func TestSocksTunnelPoolPublishesBeforeClosedOwnerCanRetire(t *testing.T) {
	connection := &transports.Connection{IsOpen: true, Send: make(chan *sliverpb.Envelope, 1)}
	connection.Cleanup()
	removal := make(chan func(), 1)
	pool := socksTunnelPool{
		tunnels:           map[uint64]*socksTunnelState{},
		tombstoneDuration: time.Hour,
		scheduleRemoval: func(_ time.Duration, remove func()) {
			removal <- remove
		},
	}

	tunnel, created := pool.loadOrCreateForConnection(54, connection)
	if tunnel == nil || !created {
		t.Fatal("create SOCKS state for already-closed owner")
	}
	select {
	case <-tunnel.done:
	case <-time.After(2 * time.Second):
		t.Fatal("already-closed owner did not stop SOCKS actor")
	}
	select {
	case remove := <-removal:
		pool.mutex.Lock()
		loaded := pool.tunnels[54]
		active := pool.active
		pool.mutex.Unlock()
		if loaded != tunnel || active != 0 {
			t.Fatalf("closed-owner state = %p active %d, want exact tombstone %p and zero active", loaded, active, tunnel)
		}
		if tunnel.reserveTotal == nil || tunnel.releaseTotal == nil {
			t.Fatal("actor started before pool accounting callbacks were installed")
		}
		remove()
	case <-time.After(2 * time.Second):
		t.Fatal("already-closed owner did not retire exact SOCKS state")
	}
	if _, ok := pool.get(54); ok {
		t.Fatal("closed-owner tombstone remained after scheduled removal")
	}
}

func TestSocksTunnelOwnerCleanupWhileOrderedDeliveryIsUnread(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		connection := &transports.Connection{Send: make(chan *sliverpb.Envelope)}
		tunnel := newOwnedSocksTunnelState(connection.Done(), nil)
		if !tunnel.submit(socksTunnelFrame{sequence: 0, data: []byte("unread")}) {
			t.Fatal("submit unread ordered frame failed")
		}
		// Wait until the actor has enqueued the frame with no Reader present.
		synctest.Wait()
		connection.Cleanup()
		synctest.Wait()
		select {
		case <-tunnel.done:
		default:
			t.Fatal("owner cleanup did not close actor blocked on unread data")
		}
		if tunnel.submit(socksTunnelFrame{sequence: 1, data: []byte("late")}) {
			t.Fatal("actor accepted data after blocked owner cleanup")
		}
	})
}

// This adapter contract test covers zero-length I/O, partial reads, terminal
// ordering, and EOF as one connected stream lifecycle.
//
//nolint:gocyclo
func TestSocksReadRetainsShortBufferRemainderAndZeroLengthIO(t *testing.T) {
	tunnel := newSocksTunnelState()
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 1)}
	adapter := &socks{
		stream: &sliverpb.SocksData{TunnelID: 61},
		conn:   connection,
		tunnel: tunnel,
	}
	defer tunnel.close()

	zero := make([]byte, 0)
	if count, err := adapter.Read(zero); count != 0 || err != nil {
		t.Fatalf("zero-length read = %d, %v, want 0, nil", count, err)
	}
	if count, err := adapter.Write(nil); count != 0 || err != nil {
		t.Fatalf("zero-length write = %d, %v, want 0, nil", count, err)
	}
	select {
	case envelope := <-connection.Send:
		t.Fatalf("zero-length write emitted envelope %+v", envelope)
	default:
	}

	if !tunnel.submit(socksTunnelFrame{sequence: 0, data: []byte("abcdef")}) {
		t.Fatal("submit adapter payload failed")
	}
	first := make([]byte, 2)
	if count, err := adapter.Read(first); count != 2 || err != nil || string(first) != "ab" {
		t.Fatalf("first short read = %d %q, %v, want 2 ab, nil", count, first, err)
	}
	if !tunnel.submit(socksTunnelFrame{sequence: 1, close: true}) {
		t.Fatal("submit terminal after adapter payload failed")
	}
	waitForSocksTunnelLifecycle(t, tunnel, socksTunnelGraceful)
	second := make([]byte, 4)
	if count, err := adapter.Read(second); count != 4 || err != nil || string(second) != "cdef" {
		t.Fatalf("remainder read = %d %q, %v, want 4 cdef, nil", count, second, err)
	}
	if count, err := adapter.Read(make([]byte, 1)); count != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("post-terminal read = %d, %v, want 0, io.EOF", count, err)
	}
}

func TestSocksWriteDataPrecedesTerminalAtContiguousSequence(t *testing.T) {
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 3)}
	adapter := &socks{
		stream: &sliverpb.SocksData{TunnelID: 62},
		conn:   connection,
		tunnel: newSocksTunnelState(),
	}

	for _, payload := range []string{"first", "second"} {
		if count, err := adapter.Write([]byte(payload)); count != len(payload) || err != nil {
			t.Fatalf("write %q = %d, %v, want %d, nil", payload, count, err, len(payload))
		}
	}
	if err := adapter.Close(); err != nil {
		t.Fatalf("close SOCKS adapter: %v", err)
	}

	for sequence, payload := range []string{"first", "second"} {
		frame := readSocksTestFrame(t, connection.Send)
		if frame.Sequence != uint64(sequence) || frame.CloseConn || string(frame.Data) != payload {
			t.Fatalf("data frame %d = sequence %d, close %t, data %q", sequence, frame.Sequence, frame.CloseConn, frame.Data)
		}
	}
	terminal := readSocksTestFrame(t, connection.Send)
	if terminal.Sequence != 2 || !terminal.CloseConn || len(terminal.Data) != 0 {
		t.Fatalf("terminal frame = sequence %d, close %t, data %x, want sequence 2 close", terminal.Sequence, terminal.CloseConn, terminal.Data)
	}
}

func TestSocksWriteAfterCloseIsRejectedWithoutEnvelope(t *testing.T) {
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 2)}
	adapter := &socks{
		stream: &sliverpb.SocksData{TunnelID: 63},
		conn:   connection,
		tunnel: newSocksTunnelState(),
	}
	if err := adapter.Close(); err != nil {
		t.Fatalf("close SOCKS adapter: %v", err)
	}
	terminal := readSocksTestFrame(t, connection.Send)
	if !terminal.CloseConn || terminal.Sequence != 0 {
		t.Fatalf("terminal frame = sequence %d, close %t, want sequence 0 close", terminal.Sequence, terminal.CloseConn)
	}
	if count, err := adapter.Write([]byte("late")); count != 0 || !errors.Is(err, transports.ErrTunnelClosed) {
		t.Fatalf("write after close = %d, %v, want 0, ErrTunnelClosed", count, err)
	}
	select {
	case envelope := <-connection.Send:
		t.Fatalf("write after close emitted envelope %+v", envelope)
	default:
	}
}

func TestSocksWriteConcurrentWithCloseCannotFollowTerminal(t *testing.T) {
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope)}
	tunnel := newSocksTunnelState()
	adapter := &socks{
		stream: &sliverpb.SocksData{TunnelID: 67},
		conn:   connection,
		tunnel: tunnel,
	}

	closeResult := make(chan error, 1)
	go func() {
		closeResult <- adapter.Close()
	}()
	select {
	case <-tunnel.done:
		// Close owns outboundMu and is blocked publishing the terminal to the
		// unbuffered transport. Start a concurrent writer in that exact window.
	case <-time.After(2 * time.Second):
		t.Fatal("close did not publish the closed state")
	}
	writeResult := make(chan error, 1)
	go func() {
		_, err := adapter.Write([]byte("late"))
		writeResult <- err
	}()

	terminal := readSocksTestFrame(t, connection.Send)
	if !terminal.CloseConn || terminal.Sequence != 0 {
		t.Fatalf("terminal frame = sequence %d, close %t, want sequence 0 close", terminal.Sequence, terminal.CloseConn)
	}
	select {
	case err := <-closeResult:
		if err != nil {
			t.Fatalf("close SOCKS adapter: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("close remained blocked after terminal delivery")
	}
	select {
	case err := <-writeResult:
		if !errors.Is(err, transports.ErrTunnelClosed) {
			t.Fatalf("write concurrent with close error = %v, want ErrTunnelClosed", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("write concurrent with close remained blocked")
	}
	select {
	case envelope := <-connection.Send:
		t.Fatalf("write concurrent with close emitted envelope after terminal %+v", envelope)
	default:
	}
}

func TestSocksCloseIsIdempotentAndReturnsStableResult(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 2)}
		adapter := &socks{
			stream: &sliverpb.SocksData{TunnelID: 64},
			conn:   connection,
			tunnel: newSocksTunnelState(),
		}
		if err := adapter.Close(); err != nil {
			t.Fatalf("first close: %v", err)
		}
		if err := adapter.Close(); err != nil {
			t.Fatalf("repeated close: %v", err)
		}
		_ = readSocksTestFrame(t, connection.Send)
		select {
		case envelope := <-connection.Send:
			t.Fatalf("repeated close emitted envelope %+v", envelope)
		default:
		}
	})

	t.Run("failure", func(t *testing.T) {
		adapter := &socks{
			stream: &sliverpb.SocksData{TunnelID: 65},
			conn:   &transports.Connection{},
			tunnel: newSocksTunnelState(),
		}
		firstErr := adapter.Close()
		secondErr := adapter.Close()
		if firstErr != transports.ErrTunnelClosed || secondErr != firstErr {
			t.Fatalf("close errors = %v, %v, want stable ErrTunnelClosed", firstErr, secondErr)
		}
	})
}

func TestSocksTerminalSendFailureFailsLiveConnectionClosed(t *testing.T) {
	connection := &transports.Connection{IsOpen: true}
	done := connection.Done()
	select {
	case <-done:
		t.Fatal("test C2 connection started closed")
	default:
	}
	adapter := &socks{
		stream: &sliverpb.SocksData{TunnelID: 68},
		conn:   connection,
		tunnel: newSocksTunnelState(),
	}

	if err := adapter.Close(); !errors.Is(err, transports.ErrTunnelClosed) {
		t.Fatalf("terminal send failure = %v, want ErrTunnelClosed", err)
	}
	select {
	case <-done:
	default:
		t.Fatal("terminal send failure left the owning C2 connection live")
	}
	if connection.IsOpen {
		t.Fatal("terminal send failure did not mark the owning C2 connection closed")
	}
}

func TestSocksWriteHasNoArtificialPerFramePacing(t *testing.T) {
	const frameCount = 64
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, frameCount)}
	adapter := &socks{
		stream: &sliverpb.SocksData{TunnelID: 66},
		conn:   connection,
		tunnel: newSocksTunnelState(),
	}
	defer adapter.tunnel.close()

	started := time.Now()
	for sequence := 0; sequence < frameCount; sequence++ {
		if count, err := adapter.Write([]byte{byte(sequence)}); count != 1 || err != nil {
			t.Fatalf("write frame %d = %d, %v, want 1, nil", sequence, count, err)
		}
	}
	if elapsed := time.Since(started); elapsed >= 500*time.Millisecond {
		t.Fatalf("%d immediately writable frames took %s; outbound path appears artificially paced", frameCount, elapsed)
	}
	for sequence := 0; sequence < frameCount; sequence++ {
		frame := readSocksTestFrame(t, connection.Send)
		if frame.Sequence != uint64(sequence) || frame.CloseConn || len(frame.Data) != 1 || frame.Data[0] != byte(sequence) {
			t.Fatalf("frame %d = sequence %d, close %t, data %x", sequence, frame.Sequence, frame.CloseConn, frame.Data)
		}
	}
}

// The scenario keeps the full-window block, cumulative acknowledgement, and
// resumed frame ordering in one connected flow-control lifecycle.
//
//nolint:gocyclo
func TestSocksFlowControlBlocksAtWindowAndAckResumes(t *testing.T) {
	const tunnelID = uint64(0xfeed6001)
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, socksTunnelFlowWindow+2)}
	t.Cleanup(connection.Cleanup)
	pool := socksTunnelPool{
		tunnels:           map[uint64]*socksTunnelState{},
		tombstoneDuration: time.Hour,
	}
	tunnel, created := pool.loadOrCreateForConnection(tunnelID, connection)
	if tunnel == nil || !created {
		t.Fatal("create flow-controlled SOCKS tunnel")
	}
	t.Cleanup(func() { tunnel.close() })
	tunnel.enableFlowControl(sliverpb.CapabilitySocksFlowControlV1)
	adapter := &socks{
		stream: &sliverpb.SocksData{TunnelID: tunnelID},
		conn:   connection,
		tunnel: tunnel,
	}

	for sequence := 0; sequence < socksTunnelFlowWindow; sequence++ {
		if count, err := adapter.Write([]byte{byte(sequence)}); count != 1 || err != nil {
			t.Fatalf("write frame %d = %d, %v, want 1, nil", sequence, count, err)
		}
	}
	writeResult := make(chan error, 1)
	go func() {
		_, err := adapter.Write([]byte("blocked"))
		writeResult <- err
	}()
	select {
	case err := <-writeResult:
		t.Fatalf("frame beyond flow-control window returned early: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	handleSocksReq(socksRequestEnvelope(t, &sliverpb.SocksData{
		TunnelID: tunnelID,
		Ack:      socksTunnelFlowAckBatch,
	}), connection, &pool)
	select {
	case err := <-writeResult:
		if err != nil {
			t.Fatalf("write after cumulative ACK: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cumulative ACK did not release outbound credit")
	}

	last := readSocksTestFrame(t, connection.Send)
	for len(connection.Send) > 0 {
		last = readSocksTestFrame(t, connection.Send)
	}
	if last.Sequence != socksTunnelFlowWindow || string(last.Data) != "blocked" || last.CloseConn || last.Ack != 0 {
		t.Fatalf("resumed frame = sequence %d data %q close %t ack %d", last.Sequence, last.Data, last.CloseConn, last.Ack)
	}
}

func TestSocksFlowControlCloseWakesBlockedWriter(t *testing.T) {
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, socksTunnelFlowWindow)}
	tunnel := newSocksTunnelState()
	tunnel.enableFlowControl(sliverpb.CapabilitySocksFlowControlV1)
	adapter := &socks{
		stream: &sliverpb.SocksData{TunnelID: 0xfeed6002},
		conn:   connection,
		tunnel: tunnel,
	}
	for sequence := 0; sequence < socksTunnelFlowWindow; sequence++ {
		if _, err := adapter.Write([]byte{byte(sequence)}); err != nil {
			t.Fatalf("fill flow-control window at frame %d: %v", sequence, err)
		}
	}
	writeResult := make(chan error, 1)
	go func() {
		_, err := adapter.Write([]byte("blocked"))
		writeResult <- err
	}()
	select {
	case err := <-writeResult:
		t.Fatalf("frame beyond flow-control window returned early: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	tunnel.close()
	select {
	case err := <-writeResult:
		if !errors.Is(err, transports.ErrTunnelClosed) {
			t.Fatalf("blocked writer after close = %v, want ErrTunnelClosed", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("tunnel close did not wake blocked flow-control writer")
	}
}

func TestSocksFlowControlRejectsAckBeyondSentSequence(t *testing.T) {
	const tunnelID = uint64(0xfeed6005)
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 3)}
	t.Cleanup(connection.Cleanup)
	pool := socksTunnelPool{
		tunnels:           map[uint64]*socksTunnelState{},
		tombstoneDuration: time.Hour,
		scheduleRemoval:   func(time.Duration, func()) {},
	}
	tunnel, created := pool.loadOrCreateForConnection(tunnelID, connection)
	if tunnel == nil || !created {
		t.Fatal("create flow-controlled SOCKS tunnel")
	}
	tunnel.enableFlowControl(sliverpb.CapabilitySocksFlowControlV1)
	adapter := &socks{
		stream: &sliverpb.SocksData{TunnelID: tunnelID},
		conn:   connection,
		tunnel: tunnel,
	}
	if _, err := adapter.Write([]byte("sent")); err != nil {
		t.Fatalf("write one outbound frame: %v", err)
	}

	handleSocksReq(socksRequestEnvelope(t, &sliverpb.SocksData{
		TunnelID: tunnelID,
		Ack:      2,
	}), connection, &pool)
	select {
	case <-tunnel.done:
	case <-time.After(2 * time.Second):
		t.Fatal("future cumulative ACK did not close exact tunnel")
	}
	if replacement, ok := pool.get(tunnelID); !ok || replacement != tunnel {
		t.Fatal("future ACK replaced or removed replay tombstone")
	}
}

// The scenario verifies both the acknowledgement batch boundary and partial
// frame consumption in one adapter lifecycle.
//
//nolint:gocyclo
func TestSocksFlowControlAcksOnlyFullyConsumedFrames(t *testing.T) {
	const tunnelID = uint64(0xfeed6003)
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 2)}
	tunnel := newSocksTunnelState()
	defer tunnel.close()
	adapter := &socks{
		stream: &sliverpb.SocksData{TunnelID: tunnelID},
		conn:   connection,
		tunnel: tunnel,
	}

	for sequence := uint64(0); sequence < socksTunnelFlowAckBatch-1; sequence++ {
		frame := socksTunnelFrame{sequence: sequence, data: []byte{byte(sequence)}}
		if sequence == 0 {
			frame.capabilities = sliverpb.CapabilitySocksFlowControlV1
		}
		if !tunnel.submit(frame) {
			t.Fatalf("submit inbound frame %d", sequence)
		}
		buffer := make([]byte, 1)
		if count, err := adapter.Read(buffer); count != 1 || err != nil {
			t.Fatalf("consume inbound frame %d = %d, %v", sequence, count, err)
		}
	}
	select {
	case envelope := <-connection.Send:
		t.Fatalf("ACK emitted before batch boundary: %+v", envelope)
	default:
	}

	if !tunnel.submit(socksTunnelFrame{sequence: socksTunnelFlowAckBatch - 1, data: []byte("abc")}) {
		t.Fatal("submit partial-read frame")
	}
	first := make([]byte, 1)
	if count, err := adapter.Read(first); count != 1 || err != nil || string(first) != "a" {
		t.Fatalf("partial inbound read = %d %q, %v", count, first, err)
	}
	select {
	case envelope := <-connection.Send:
		t.Fatalf("ACK emitted before whole frame consumption: %+v", envelope)
	default:
	}
	remainder := make([]byte, 2)
	if count, err := adapter.Read(remainder); count != 2 || err != nil || string(remainder) != "bc" {
		t.Fatalf("remainder inbound read = %d %q, %v", count, remainder, err)
	}
	ack := readSocksTestFrame(t, connection.Send)
	if ack.TunnelID != tunnelID || ack.Ack != socksTunnelFlowAckBatch || ack.Sequence != 0 || ack.CloseConn || len(ack.Data) != 0 || ack.Capabilities != 0 {
		t.Fatalf("batch ACK = tunnel %d ack %d sequence %d close %t data %x capabilities %d", ack.TunnelID, ack.Ack, ack.Sequence, ack.CloseConn, ack.Data, ack.Capabilities)
	}
}

func TestSocksFlowControlFlushesAckAtOrderedEOF(t *testing.T) {
	const tunnelID = uint64(0xfeed6004)
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 1)}
	tunnel := newSocksTunnelState()
	defer tunnel.close()
	adapter := &socks{
		stream: &sliverpb.SocksData{TunnelID: tunnelID},
		conn:   connection,
		tunnel: tunnel,
	}
	if !tunnel.submit(socksTunnelFrame{
		sequence:     0,
		data:         []byte("x"),
		capabilities: sliverpb.CapabilitySocksFlowControlV1,
	}) {
		t.Fatal("submit inbound frame")
	}
	if !tunnel.submit(socksTunnelFrame{sequence: 1, close: true}) {
		t.Fatal("submit ordered terminal")
	}
	waitForSocksTunnelLifecycle(t, tunnel, socksTunnelGraceful)
	buffer := make([]byte, 1)
	if count, err := adapter.Read(buffer); count != 1 || err != nil || string(buffer) != "x" {
		t.Fatalf("consume final inbound frame = %d %q, %v", count, buffer, err)
	}
	select {
	case envelope := <-connection.Send:
		t.Fatalf("short batch ACK emitted before EOF: %+v", envelope)
	default:
	}
	if count, err := adapter.Read(buffer); count != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("read ordered EOF = %d, %v, want 0, io.EOF", count, err)
	}
	ack := readSocksTestFrame(t, connection.Send)
	if ack.Ack != 1 || ack.TunnelID != tunnelID {
		t.Fatalf("terminal flush ACK = tunnel %d ack %d, want tunnel %d ack 1", ack.TunnelID, ack.Ack, tunnelID)
	}
}

// This integration test keeps the full greeting, connect, target ownership,
// and owner-cleanup lifecycle together to detect cross-stage leaks.
//
//nolint:gocyclo
func TestSocksServerOwnerCleanupClosesEstablishedTarget(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	accepted := make(chan net.Conn, 1)
	go func() {
		connection, _ := listener.Accept()
		accepted <- connection
	}()

	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 8)}
	defer connection.Cleanup()
	pool := socksTunnelPool{
		tunnels:           map[uint64]*socksTunnelState{},
		tombstoneDuration: time.Hour,
	}
	tunnel, _ := pool.loadOrCreate(53, connection.Done())
	if !tunnel.submit(socksTunnelFrame{sequence: 0, data: []byte{0x05, 0x01, 0x00}}) {
		t.Fatal("submit SOCKS greeting failed")
	}
	username, password, claimed := tunnel.takeServerCredentials()
	if !claimed {
		t.Fatal("canonical SOCKS greeting did not claim server credentials")
	}
	tunnel.handshakeTimeout = time.Second
	tunnel.startHandshakeLease()
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- newSocksServerForTunnel(username, password, tunnel).ServeConn(&socks{
			stream: &sliverpb.SocksData{TunnelID: 53},
			conn:   connection,
			tunnel: tunnel,
		})
	}()
	methodResponse := readSocksTestEnvelope(t, connection.Send)
	if string(methodResponse) != string([]byte{0x05, 0x00}) {
		t.Fatalf("SOCKS method response = %x, want 0500", methodResponse)
	}
	target := listener.Addr().(*net.TCPAddr)
	request := []byte{0x05, 0x01, 0x00, 0x01, 127, 0, 0, 1, 0, 0}
	binary.BigEndian.PutUint16(request[8:], uint16(target.Port))
	if !tunnel.submit(socksTunnelFrame{sequence: 1, data: request}) {
		t.Fatal("submit SOCKS CONNECT failed")
	}
	connectResponse := readSocksTestEnvelope(t, connection.Send)
	if len(connectResponse) < 2 || connectResponse[0] != 0x05 || connectResponse[1] != 0x00 {
		t.Fatalf("SOCKS CONNECT response = %x, want success", connectResponse)
	}
	select {
	case <-tunnel.established:
	case <-time.After(time.Second):
		t.Fatal("successful SOCKS target dial did not release the handshake lease")
	}

	var targetConnection net.Conn
	select {
	case targetConnection = <-accepted:
		if targetConnection == nil {
			t.Fatal("target listener did not accept SOCKS connection")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SOCKS target connection was not established")
	}
	defer func() { _ = targetConnection.Close() }()

	connection.Cleanup()
	select {
	case <-serverDone:
	case <-time.After(2 * time.Second):
		t.Fatal("owning connection cleanup did not stop SOCKS ServeConn")
	}
	if err := targetConnection.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 1)
	if _, err := targetConnection.Read(buffer); err == nil {
		t.Fatal("SOCKS target remained open after owning connection cleanup")
	} else if networkErr, ok := err.(net.Error); ok && networkErr.Timeout() {
		t.Fatalf("SOCKS target close timed out after owner cleanup: %v", err)
	}
}

func TestDialOwnedSocksTargetCancelsBlockedDial(t *testing.T) {
	ownerDone := make(chan struct{})
	dialStarted := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		_, err := dialOwnedSocksTargetWith(
			context.Background(),
			"tcp",
			"black-hole.invalid:443",
			ownerDone,
			func(ctx context.Context, _ string, _ string) (net.Conn, error) {
				close(dialStarted)
				<-ctx.Done()
				return nil, ctx.Err()
			},
		)
		result <- err
	}()
	select {
	case <-dialStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("owned SOCKS target dial did not start")
	}
	close(ownerDone)
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("owned SOCKS target dial error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("owner close did not cancel blocked SOCKS target dial")
	}
}

func TestOwnedSocksResolverWatchStopsAfterDial(t *testing.T) {
	ownerDone := make(chan struct{})
	resolver := ownedSocksResolver{
		ownerDone: ownerDone,
		lookupIPAddr: func(context.Context, string) ([]net.IPAddr, error) {
			address := net.IPAddr{IP: net.IPv4(127, 0, 0, 1)}
			return []net.IPAddr{address}, nil
		},
	}
	resolveContext, address, err := resolver.Resolve(context.Background(), "target.invalid")
	if err != nil {
		t.Fatalf("resolve test target: %v", err)
	}
	if !address.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Fatalf("resolved address = %v, want 127.0.0.1", address)
	}
	watch, ok := resolveContext.Value(socksResolverWatchContextKey{}).(*socksResolverWatch)
	if !ok || watch == nil {
		t.Fatal("resolver did not attach an owner watcher")
	}
	select {
	case <-watch.done:
		t.Fatal("resolver watcher stopped before dial consumed its context")
	default:
	}

	serverConnection, clientConnection := net.Pipe()
	owned, err := dialOwnedSocksTargetWith(
		resolveContext,
		"tcp",
		"target.invalid:443",
		ownerDone,
		func(context.Context, string, string) (net.Conn, error) {
			return serverConnection, nil
		},
	)
	if err != nil {
		t.Fatalf("dial resolved test target: %v", err)
	}
	defer func() { _ = clientConnection.Close() }()
	defer func() { _ = owned.Close() }()
	select {
	case <-watch.done:
	case <-time.After(time.Second):
		t.Fatal("successful dial left resolver owner watcher running")
	}
	select {
	case <-resolveContext.Done():
	default:
		t.Fatal("successful dial did not release resolver context")
	}
}

func TestOwnedSocksResolverWatchStopsAfterLookupFailure(t *testing.T) {
	ownerDone := make(chan struct{})
	wantErr := errors.New("lookup failed")
	resolver := ownedSocksResolver{
		ownerDone: ownerDone,
		lookupIPAddr: func(context.Context, string) ([]net.IPAddr, error) {
			return nil, wantErr
		},
	}
	resolveContext, _, err := resolver.Resolve(context.Background(), "target.invalid")
	if !errors.Is(err, wantErr) {
		t.Fatalf("resolver error = %v, want %v", err, wantErr)
	}
	watch, ok := resolveContext.Value(socksResolverWatchContextKey{}).(*socksResolverWatch)
	if !ok || watch == nil {
		t.Fatal("resolver failure did not expose its owner watcher lifecycle")
	}
	select {
	case <-watch.done:
	case <-time.After(time.Second):
		t.Fatal("failed lookup left resolver owner watcher running")
	}
}

func TestHostnameDialSocksRulePreservesPostRewriteHostname(t *testing.T) {
	resolved := &statute.AddrSpec{
		FQDN: "localhost",
		IP:   net.ParseIP("::1"),
		Port: 31337,
	}
	request := &socks5.Request{
		Request:  statute.Request{Command: statute.CommandConnect},
		DestAddr: resolved,
	}
	rule := hostnameDialSocksRule{delegate: &socks5.PermitCommand{EnableConnect: true}}
	ctx, allowed := rule.Allow(context.Background(), request)
	if !allowed {
		t.Fatal("CONNECT request was not allowed")
	}
	if got := socksTargetDialAddress(ctx, "[::1]:31337"); got != "localhost:31337" {
		t.Fatalf("hostname dial address = %q, want localhost:31337", got)
	}
	if request.DestAddr.IP.String() != "::1" {
		t.Fatalf("rule changed resolved IP to %v, want ::1", request.DestAddr.IP)
	}

	request.DestAddr = &statute.AddrSpec{IP: net.ParseIP("::1"), Port: 31337}
	ctx, allowed = rule.Allow(context.Background(), request)
	if !allowed {
		t.Fatal("literal CONNECT request was not allowed")
	}
	if got := socksTargetDialAddress(ctx, "[::1]:31337"); got != "[::1]:31337" {
		t.Fatalf("literal dial address = %q, want [::1]:31337", got)
	}

	request.Command = statute.CommandBind
	ctx, allowed = rule.Allow(context.Background(), request)
	if allowed {
		t.Fatal("BIND request was allowed")
	}
	if got := socksTargetDialAddress(ctx, "[::1]:31337"); got != "[::1]:31337" {
		t.Fatalf("denied request dial address = %q, want [::1]:31337", got)
	}
}

func readSocksTestEnvelope(t *testing.T, envelopes <-chan *sliverpb.Envelope) []byte {
	t.Helper()
	return readSocksTestFrame(t, envelopes).Data
}

func readSocksTestFrame(t *testing.T, envelopes <-chan *sliverpb.Envelope) *sliverpb.SocksData {
	t.Helper()
	select {
	case envelope := <-envelopes:
		if envelope == nil || envelope.Type != sliverpb.MsgSocksData {
			t.Fatalf("SOCKS response envelope = %+v", envelope)
		}
		data := &sliverpb.SocksData{}
		if err := proto.Unmarshal(envelope.Data, data); err != nil {
			t.Fatalf("decode SOCKS response envelope: %v", err)
		}
		return data
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SOCKS response envelope")
		return &sliverpb.SocksData{}
	}
}

func TestSocksServerAuthenticationPolicySequentialIsolation(t *testing.T) {
	policies := []socksAuthenticationPolicy{
		{username: "operator", password: "correct-horse", wantMethod: 0x02},
		{wantMethod: 0x00},
		{username: "analyst", password: "battery-staple", wantMethod: 0x02},
		{wantMethod: 0x00},
	}
	for index, policy := range policies {
		server := newSocksServer(policy.username, policy.password)
		if err := probeSocksAuthenticationPolicy(server.ServeConn, policy); err != nil {
			t.Fatalf("sequential policy %d: %v", index, err)
		}
	}
}

func TestSocksServerAuthenticationPolicyConcurrentIsolation(t *testing.T) {
	const probeCount = 24
	start := make(chan struct{})
	errs := make(chan error, probeCount)
	var workers sync.WaitGroup
	for index := 0; index < probeCount; index++ {
		index := index
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			policy := socksAuthenticationPolicy{wantMethod: 0x00}
			if index%2 == 0 {
				policy = socksAuthenticationPolicy{
					username:   fmt.Sprintf("operator-%d", index),
					password:   fmt.Sprintf("password-%d", index),
					wantMethod: 0x02,
				}
			}
			server := newSocksServer(policy.username, policy.password)
			if err := probeSocksAuthenticationPolicy(server.ServeConn, policy); err != nil {
				errs <- fmt.Errorf("concurrent policy %d: %w", index, err)
			}
		}()
	}
	close(start)
	workers.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func TestSocksServerRejectsUnsupportedUDPAssociateWithoutPanic(t *testing.T) {
	serverConnection, clientConnection := net.Pipe()
	deadline := time.Now().Add(2 * time.Second)
	if err := serverConnection.SetDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	if err := clientConnection.SetDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- newSocksServer("", "").ServeConn(serverConnection)
	}()
	defer func() { _ = clientConnection.Close() }()

	if err := writeSocksProbe(clientConnection, []byte{0x05, 0x01, 0x00}); err != nil {
		t.Fatalf("write no-auth greeting: %v", err)
	}
	methodResponse := make([]byte, 2)
	if _, err := io.ReadFull(clientConnection, methodResponse); err != nil {
		t.Fatalf("read no-auth response: %v", err)
	}
	if methodResponse[0] != 0x05 || methodResponse[1] != 0x00 {
		t.Fatalf("no-auth response = %x, want 0500", methodResponse)
	}
	// A syntactically valid UDP ASSOCIATE must be rejected by the command
	// rule before the upstream handler attempts to use this virtual net.Conn's
	// intentionally nil LocalAddr.
	request := []byte{0x05, 0x03, 0x00, 0x01, 0, 0, 0, 0, 0, 0}
	if err := writeSocksProbe(clientConnection, request); err != nil {
		t.Fatalf("write UDP ASSOCIATE request: %v", err)
	}
	reply := make([]byte, 10)
	if _, err := io.ReadFull(clientConnection, reply); err != nil {
		t.Fatalf("read UDP ASSOCIATE rejection: %v", err)
	}
	if reply[0] != 0x05 || reply[1] != 0x02 {
		t.Fatalf("UDP ASSOCIATE reply = %x, want SOCKS5 rule failure", reply)
	}
	select {
	case err := <-serverDone:
		if err == nil {
			t.Fatal("UDP ASSOCIATE unexpectedly succeeded")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SOCKS server did not finish after rejecting UDP ASSOCIATE")
	}
}

type socksAuthenticationPolicy struct {
	username   string
	password   string
	wantMethod byte
}

func probeSocksAuthenticationPolicy(serveConn func(net.Conn) error, policy socksAuthenticationPolicy) (resultErr error) {
	serverConnection, clientConnection := net.Pipe()
	deadline := time.Now().Add(2 * time.Second)
	if err := serverConnection.SetDeadline(deadline); err != nil {
		return err
	}
	if err := clientConnection.SetDeadline(deadline); err != nil {
		return err
	}
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- serveConn(serverConnection)
	}()
	defer func() {
		_ = clientConnection.Close()
		select {
		case <-serverDone:
		case <-time.After(2 * time.Second):
			resultErr = errors.Join(resultErr, errors.New("SOCKS authentication probe server did not stop"))
		}
	}()

	// Offer both methods so the response reflects this server instance's
	// policy rather than a limitation in the probe.
	if err := writeSocksProbe(clientConnection, []byte{0x05, 0x02, 0x00, 0x02}); err != nil {
		return fmt.Errorf("write method greeting: %w", err)
	}
	methodResponse := make([]byte, 2)
	if _, err := io.ReadFull(clientConnection, methodResponse); err != nil {
		return fmt.Errorf("read method response: %w", err)
	}
	if methodResponse[0] != 0x05 || methodResponse[1] != policy.wantMethod {
		return fmt.Errorf("method response = %x, want 05%02x", methodResponse, policy.wantMethod)
	}
	if policy.wantMethod != 0x02 {
		return nil
	}

	authRequest := []byte{0x01, byte(len(policy.username))}
	authRequest = append(authRequest, policy.username...)
	authRequest = append(authRequest, byte(len(policy.password)))
	authRequest = append(authRequest, policy.password...)
	if err := writeSocksProbe(clientConnection, authRequest); err != nil {
		return fmt.Errorf("write username/password request: %w", err)
	}
	authResponse := make([]byte, 2)
	if _, err := io.ReadFull(clientConnection, authResponse); err != nil {
		return fmt.Errorf("read username/password response: %w", err)
	}
	if authResponse[0] != 0x01 || authResponse[1] != 0x00 {
		return fmt.Errorf("username/password response = %x, want 0100", authResponse)
	}
	return nil
}

func writeSocksProbe(connection net.Conn, data []byte) error {
	for len(data) > 0 {
		written, err := connection.Write(data)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		data = data[written:]
	}
	return nil
}
