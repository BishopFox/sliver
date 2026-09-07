package handlers

import (
	"bytes"
	"errors"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/core/rtunnels"
)

// Reverse-port-forward data leaves an implant on independent transport
// streams. A follower can therefore reach the server before the sequence-zero
// CreateReverse frame. This registry owns those inert precursors until their
// exact connection generation supplies the authorized create frame.
//
// The first precursor permanently claims its tunnel ID on the C2 connection.
// Consequently, expiry or resource rejection leaves a bounded replay tombstone
// and a late create can never open a truncated stream.
const (
	maxReverseTunnelPreCreateEntriesPerSession = 32
	maxReverseTunnelPreCreateEntriesGlobal     = 128
	maxReverseTunnelPreCreateFramesPerEntry    = 128
	maxReverseTunnelPreCreateFramesPerSession  = 128
	maxReverseTunnelPreCreateFramesGlobal      = 512
	maxReverseTunnelPreCreateBytesPerEntry     = sliverpb.MaxTunnelFrameBytes * maxReverseTunnelPreCreateFramesPerEntry
	maxReverseTunnelPreCreateBytesPerSession   = 32 * 1024 * 1024
	maxReverseTunnelPreCreateBytesGlobal       = 128 * 1024 * 1024
	maxReverseTunnelPreCreateTerminalSequence  = maxReverseTunnelPreCreateFramesPerEntry + 1
)

var (
	reverseTunnelPreCreateTTL = 10 * time.Second

	errReverseTunnelPreCreateInvalid        = errors.New("invalid reverse tunnel precursor")
	errReverseTunnelPreCreateConflict       = errors.New("conflicting reverse tunnel precursor")
	errReverseTunnelPreCreateEntryLimit     = errors.New("reverse tunnel precursor entry limit reached")
	errReverseTunnelPreCreateFrameLimit     = errors.New("reverse tunnel precursor frame limit reached")
	errReverseTunnelPreCreatePendingBytes   = errors.New("reverse tunnel precursor pending-data limit reached")
	errReverseTunnelPreCreateExpired        = errors.New("reverse tunnel precursor expired before create")
	errReverseTunnelPreCreateAlreadyClaimed = errors.New("reverse tunnel precursor ID is already claimed")
	errReverseTunnelPreCreateCollision      = errors.New("reverse tunnel precursor collides with generic tunnel")
)

type reverseTunnelPreCreateKey struct {
	connection *core.ImplantConnection
	sessionID  string
	tunnelID   uint64
}

type reverseTunnelPreCreateUsage struct {
	entries int
	frames  int
	bytes   int
}

type reverseTunnelPreCreateEntry struct {
	key          reverseTunnelPreCreateKey
	frames       map[uint64]*sliverpb.TunnelData
	terminal     *sliverpb.TunnelData
	bytes        int
	invalid      error
	claimReady   chan struct{}
	claimResult  core.ReverseTunnelIDClaimResult
	rejectionSet atomic.Bool
	stop         chan struct{}
	detached     bool
	released     bool
}

type reverseTunnelPreCreateRegistry struct {
	// routeMutex is the linearization point between the first precursor and
	// CreateReverse publication. Resource cleanup uses only mutex, so a C2
	// capacity failure cannot deadlock its Done-driven cache cleanup.
	routeMutex sync.Mutex
	mutex      sync.Mutex
	entries    map[reverseTunnelPreCreateKey]*reverseTunnelPreCreateEntry
	perSession map[string]*reverseTunnelPreCreateUsage
	total      reverseTunnelPreCreateUsage
	ttl        time.Duration
	generic    func(uint64) *core.Tunnel
}

type reverseTunnelPreCreateStageResult uint8

const (
	reverseTunnelPreCreateStaged reverseTunnelPreCreateStageResult = iota
	reverseTunnelPreCreateKnown
	reverseTunnelPreCreateGenericKnown
	reverseTunnelPreCreateRejected
	reverseTunnelPreCreateTombstoned
)

var reverseTunnelPreCreates = newReverseTunnelPreCreateRegistry(reverseTunnelPreCreateTTL)

func newReverseTunnelPreCreateRegistry(ttl time.Duration) *reverseTunnelPreCreateRegistry {
	if ttl <= 0 {
		ttl = reverseTunnelPreCreateTTL
	}
	return &reverseTunnelPreCreateRegistry{
		entries:    map[reverseTunnelPreCreateKey]*reverseTunnelPreCreateEntry{},
		perSession: map[string]*reverseTunnelPreCreateUsage{},
		ttl:        ttl,
		generic:    core.Tunnels.Get,
	}
}

func canonicalReverseTunnelPreCreateData(frame *sliverpb.TunnelData) bool {
	if frame == nil || frame.CreateReverse || frame.Closed || frame.Resend || frame.Sequence == 0 ||
		frame.Sequence > maxReverseTunnelPreCreateFramesPerEntry || len(frame.Data) == 0 ||
		len(frame.Data) > sliverpb.MaxTunnelFrameBytes || frame.SessionID != "" || frame.Rportfwd == nil {
		return false
	}
	return emptyReverseTunnelMarker(frame.Rportfwd)
}

func emptyReverseTunnelMarker(marker *sliverpb.RPortfwd) bool {
	if marker == nil {
		return false
	}
	return marker.Port == 0 && marker.Protocol == 0 && marker.Host == "" &&
		marker.AuthorizationID == "" && marker.TunnelID == 0 && marker.Response == nil
}

func canonicalReverseTunnelCreate(frame *sliverpb.TunnelData) bool {
	return frame != nil && frame.CreateReverse && !frame.Closed && !frame.Resend &&
		frame.Sequence == 0 && frame.Ack == 0 && len(frame.Data) <= sliverpb.MaxTunnelFrameBytes &&
		frame.SessionID == "" && frame.Rportfwd != nil
}

func canonicalReverseTunnelPreCreateTerminal(frame *sliverpb.TunnelData) bool {
	return frame != nil && frame.Closed && !frame.CreateReverse && !frame.Resend &&
		len(frame.Data) == 0 && frame.Ack == 0 && frame.SessionID == "" && emptyReverseTunnelMarker(frame.Rportfwd) &&
		frame.Sequence <= maxReverseTunnelPreCreateTerminalSequence
}

func canonicalKnownReverseTunnelTerminal(frame *sliverpb.TunnelData) bool {
	return frame != nil && frame.Closed && !frame.CreateReverse && !frame.Resend &&
		len(frame.Data) == 0 && frame.Ack == 0 && frame.SessionID == "" &&
		(frame.Rportfwd == nil || emptyReverseTunnelMarker(frame.Rportfwd)) &&
		frame.Sequence <= maxReverseTunnelPreCreateTerminalSequence
}

func cloneReverseTunnelPreCreateData(frame *sliverpb.TunnelData) *sliverpb.TunnelData {
	return &sliverpb.TunnelData{
		Data:     append([]byte(nil), frame.Data...),
		Sequence: frame.Sequence,
		Ack:      frame.Ack,
		TunnelID: frame.TunnelID,
		Rportfwd: &sliverpb.RPortfwd{},
	}
}

func cloneReverseTunnelPreCreateTerminal(frame *sliverpb.TunnelData) *sliverpb.TunnelData {
	return &sliverpb.TunnelData{Closed: true, Sequence: frame.Sequence, TunnelID: frame.TunnelID}
}

func equalReverseTunnelPreCreateData(left *sliverpb.TunnelData, right *sliverpb.TunnelData) bool {
	return left != nil && right != nil && left.Sequence == right.Sequence && left.Ack == right.Ack &&
		left.TunnelID == right.TunnelID && bytes.Equal(left.Data, right.Data)
}

func equalReverseTunnelPreCreateTerminal(left *sliverpb.TunnelData, right *sliverpb.TunnelData) bool {
	return left != nil && right != nil && left.Sequence == right.Sequence && left.TunnelID == right.TunnelID
}

func (registry *reverseTunnelPreCreateRegistry) stageData(connection *core.ImplantConnection, sessionID string, frame *sliverpb.TunnelData) (reverseTunnelPreCreateStageResult, error) {
	var validationErr error
	if !canonicalReverseTunnelPreCreateData(frame) {
		validationErr = errReverseTunnelPreCreateInvalid
	}
	return registry.stage(connection, sessionID, frame, false, validationErr, true, true)
}

func (registry *reverseTunnelPreCreateRegistry) stageTerminal(connection *core.ImplantConnection, sessionID string, frame *sliverpb.TunnelData) (reverseTunnelPreCreateStageResult, error) {
	var validationErr error
	if !canonicalKnownReverseTunnelTerminal(frame) {
		validationErr = errReverseTunnelPreCreateInvalid
	}
	return registry.stage(connection, sessionID, frame, true, validationErr, frame != nil && frame.Rportfwd != nil, true)
}

func (registry *reverseTunnelPreCreateRegistry) rejectMalformedCreate(connection *core.ImplantConnection, sessionID string, frame *sliverpb.TunnelData) (reverseTunnelPreCreateStageResult, error) {
	return registry.stage(connection, sessionID, frame, false, errReverseTunnelPreCreateInvalid, true, false)
}

func (registry *reverseTunnelPreCreateRegistry) rejectForeignCreate(connection *core.ImplantConnection, sessionID string, frame *sliverpb.TunnelData) (reverseTunnelPreCreateStageResult, error) {
	return registry.stage(connection, sessionID, frame, false, errReverseTunnelPreCreateCollision, true, false)
}

func (registry *reverseTunnelPreCreateRegistry) rejectBeforePromotion(connection *core.ImplantConnection, sessionID string, frame *sliverpb.TunnelData, reason error) (reverseTunnelPreCreateStageResult, error) {
	return registry.stage(connection, sessionID, frame, false, reason, true, false)
}

func (registry *reverseTunnelPreCreateRegistry) stage(connection *core.ImplantConnection, sessionID string, frame *sliverpb.TunnelData, terminal bool, validationErr error, allowUnknown bool, rejectKnownInvalid bool) (reverseTunnelPreCreateStageResult, error) {
	if registry == nil || connection == nil || frame == nil || sessionID == "" {
		return reverseTunnelPreCreateRejected, errReverseTunnelPreCreateInvalid
	}
	key := reverseTunnelPreCreateKey{connection: connection, sessionID: sessionID, tunnelID: frame.TunnelID}

	registry.routeMutex.Lock()
	knownOpening, knownTunnel, known, owned := registry.reverseOwnership(frame.TunnelID, connection, sessionID)
	if known && owned && validationErr != nil && rejectKnownInvalid {
		registry.routeMutex.Unlock()
		if knownOpening != nil {
			knownOpening.fail(frame.TunnelID, validationErr)
		} else if knownTunnel != nil {
			firstFailure := knownTunnel.ClaimProtocolFailure()
			_ = closeReverseTunnelRemote(knownTunnel)
			if firstFailure {
				rejectReverseTunnel(connection, frame.TunnelID, validationErr)
			}
		}
		return reverseTunnelPreCreateTombstoned, validationErr
	}
	if known && (owned || !allowUnknown) {
		registry.routeMutex.Unlock()
		return reverseTunnelPreCreateKnown, nil
	}
	if known && validationErr == nil {
		validationErr = errReverseTunnelPreCreateCollision
	}

	registry.mutex.Lock()
	entry := registry.entries[key]
	if entry != nil {
		if validationErr != nil {
			entry.invalid = validationErr
		} else if err := registry.appendLocked(entry, frame, terminal); err != nil {
			entry.invalid = err
		}
		invalid := entry.invalid
		registry.mutex.Unlock()
		registry.routeMutex.Unlock()
		<-entry.claimReady
		if invalid != nil {
			registry.release(entry)
		}
		return registry.stageOutcome(entry)
	}
	if !allowUnknown {
		registry.mutex.Unlock()
		registry.routeMutex.Unlock()
		return reverseTunnelPreCreateGenericKnown, nil
	}

	generic := registry.generic
	if generic == nil {
		generic = core.Tunnels.Get
	}
	if generic(frame.TunnelID) != nil && validationErr == nil {
		validationErr = errReverseTunnelPreCreateCollision
	}

	usage := registry.perSession[sessionID]
	if (usage != nil && usage.entries >= maxReverseTunnelPreCreateEntriesPerSession) || registry.total.entries >= maxReverseTunnelPreCreateEntriesGlobal {
		claimResult, cleanup := connection.TryClaimReverseTunnelIDDeferredCleanup(frame.TunnelID)
		registry.mutex.Unlock()
		registry.routeMutex.Unlock()
		if cleanup != nil {
			cleanup()
		}
		if claimResult == core.ReverseTunnelIDClaimed {
			if validationErr != nil {
				return reverseTunnelPreCreateRejected, validationErr
			}
			return reverseTunnelPreCreateRejected, errReverseTunnelPreCreateEntryLimit
		}
		// A prior claim is the lightweight connection-lifetime tombstone. If the
		// connection's own ID history is exhausted, the deferred cleanup above
		// fails only that exact C2 generation closed after registry locks release.
		return reverseTunnelPreCreateTombstoned, errReverseTunnelPreCreateEntryLimit
	}
	entry = &reverseTunnelPreCreateEntry{
		key:        key,
		frames:     map[uint64]*sliverpb.TunnelData{},
		invalid:    validationErr,
		claimReady: make(chan struct{}),
		stop:       make(chan struct{}),
	}
	if usage == nil {
		usage = &reverseTunnelPreCreateUsage{}
		registry.perSession[sessionID] = usage
	}
	registry.entries[key] = entry
	usage.entries++
	registry.total.entries++
	if entry.invalid == nil {
		if err := registry.appendLocked(entry, frame, terminal); err != nil {
			entry.invalid = err
		}
	}
	registry.mutex.Unlock()
	registry.routeMutex.Unlock()

	// The provisional actor is visible before the claim. Promotion can detach
	// it, but must await claimReady; no registry lock is held while a capacity
	// failure synchronously closes this connection.
	claimResult, cleanup := connection.TryClaimReverseTunnelIDDeferredCleanup(frame.TunnelID)
	if registry.finishClaim(entry, claimResult) {
		go registry.expireOrDiscardOnClose(entry)
	}
	if cleanup != nil {
		cleanup()
	}
	return registry.stageOutcome(entry)
}

func (registry *reverseTunnelPreCreateRegistry) finishClaim(entry *reverseTunnelPreCreateEntry, result core.ReverseTunnelIDClaimResult) bool {
	registry.mutex.Lock()
	entry.claimResult = result
	select {
	case <-entry.key.connection.Done():
		entry.claimResult = core.ReverseTunnelIDConnectionClosed
	default:
	}
	close(entry.claimReady)
	retained := registry.entries[entry.key] == entry && !entry.detached &&
		entry.claimResult == core.ReverseTunnelIDClaimed && entry.invalid == nil
	var frames []*sliverpb.TunnelData
	if !retained && !entry.detached {
		frames = registry.detachAndReleaseLocked(entry)
	}
	registry.mutex.Unlock()
	scrubReverseTunnelPreCreateFrames(frames)
	return retained
}

func (registry *reverseTunnelPreCreateRegistry) stageOutcome(entry *reverseTunnelPreCreateEntry) (reverseTunnelPreCreateStageResult, error) {
	registry.mutex.Lock()
	claimResult := entry.claimResult
	invalid := entry.invalid
	registry.mutex.Unlock()
	if claimResult != core.ReverseTunnelIDClaimed {
		return reverseTunnelPreCreateTombstoned, errReverseTunnelPreCreateAlreadyClaimed
	}
	if invalid != nil {
		if entry.rejectionSet.CompareAndSwap(false, true) {
			return reverseTunnelPreCreateRejected, invalid
		}
		return reverseTunnelPreCreateTombstoned, invalid
	}
	return reverseTunnelPreCreateStaged, nil
}

func (registry *reverseTunnelPreCreateRegistry) reverseOwnership(tunnelID uint64, connection *core.ImplantConnection, sessionID string) (*reverseTunnelOpening, *rtunnels.RTunnel, bool, bool) {
	if value, ok := reverseTunnelOpenings.Load(tunnelID); ok {
		opening, _ := value.(*reverseTunnelOpening)
		return opening, nil, true, opening.ownedBy(connection, sessionID)
	}
	if tunnel := rtunnels.GetRTunnel(tunnelID); tunnel != nil {
		return nil, tunnel, true, connection != nil && tunnel.OwnedBy(sessionID, connection.ID)
	}
	return nil, nil, false, false
}

func (registry *reverseTunnelPreCreateRegistry) genericTunnel(tunnelID uint64) *core.Tunnel {
	if registry == nil {
		return core.Tunnels.Get(tunnelID)
	}
	generic := registry.generic
	if generic == nil {
		generic = core.Tunnels.Get
	}
	return generic(tunnelID)
}

func (registry *reverseTunnelPreCreateRegistry) appendLocked(entry *reverseTunnelPreCreateEntry, frame *sliverpb.TunnelData, terminal bool) error {
	if terminal {
		if entry.terminal != nil {
			if equalReverseTunnelPreCreateTerminal(entry.terminal, frame) {
				return nil
			}
			return errReverseTunnelPreCreateConflict
		}
		if frame.Sequence != 0 {
			for sequence := range entry.frames {
				if sequence >= frame.Sequence {
					return errReverseTunnelPreCreateConflict
				}
			}
		}
		entry.terminal = cloneReverseTunnelPreCreateTerminal(frame)
		return nil
	}

	if existing := entry.frames[frame.Sequence]; existing != nil {
		if equalReverseTunnelPreCreateData(existing, frame) {
			return nil
		}
		return errReverseTunnelPreCreateConflict
	}
	if entry.terminal != nil && entry.terminal.Sequence != 0 && frame.Sequence >= entry.terminal.Sequence {
		return errReverseTunnelPreCreateConflict
	}
	usage := registry.perSession[entry.key.sessionID]
	if len(entry.frames) >= maxReverseTunnelPreCreateFramesPerEntry || usage.frames >= maxReverseTunnelPreCreateFramesPerSession || registry.total.frames >= maxReverseTunnelPreCreateFramesGlobal {
		return errReverseTunnelPreCreateFrameLimit
	}
	frameBytes := len(frame.Data)
	if entry.bytes+frameBytes > maxReverseTunnelPreCreateBytesPerEntry || usage.bytes+frameBytes > maxReverseTunnelPreCreateBytesPerSession || registry.total.bytes+frameBytes > maxReverseTunnelPreCreateBytesGlobal {
		return errReverseTunnelPreCreatePendingBytes
	}
	entry.frames[frame.Sequence] = cloneReverseTunnelPreCreateData(frame)
	entry.bytes += frameBytes
	usage.frames++
	usage.bytes += frameBytes
	registry.total.frames++
	registry.total.bytes += frameBytes
	return nil
}

// promote atomically publishes the opening and detaches precursors for its
// exact connection generation. Quota remains reserved until drain/rejection
// finishes, so promotion cannot temporarily exceed the global memory bound.
func (registry *reverseTunnelPreCreateRegistry) promote(connection *core.ImplantConnection, sessionID string, tunnelID uint64, opening *reverseTunnelOpening) (*reverseTunnelPreCreateEntry, *reverseTunnelOpening, bool, error) {
	registry.routeMutex.Lock()
	key := reverseTunnelPreCreateKey{connection: connection, sessionID: sessionID, tunnelID: tunnelID}
	registry.mutex.Lock()
	entry := registry.entries[key]
	if entry != nil && registry.genericTunnel(tunnelID) != nil {
		entry.invalid = errReverseTunnelPreCreateCollision
		delete(registry.entries, key)
		entry.detached = true
		close(entry.stop)
		registry.mutex.Unlock()
		registry.routeMutex.Unlock()
		return entry, nil, false, errReverseTunnelPreCreateCollision
	}
	registry.mutex.Unlock()
	actual, loaded := reverseTunnelOpenings.LoadOrStore(tunnelID, opening)
	if loaded {
		existing, _ := actual.(*reverseTunnelOpening)
		if existing.ownedBy(connection, sessionID) {
			registry.routeMutex.Unlock()
			return nil, existing, true, nil
		}

		// A different connection generation won publication after this create's
		// initial ownership check. Permanently poison the losing generation before
		// releasing the routing linearization point; otherwise its cached follower
		// (or an unclaimed create) could be replayed after the winner closes.
		registry.mutex.Lock()
		entry = registry.entries[key]
		if entry != nil {
			entry.invalid = errReverseTunnelPreCreateCollision
			delete(registry.entries, key)
			entry.detached = true
			close(entry.stop)
		}
		registry.mutex.Unlock()
		if entry != nil {
			registry.routeMutex.Unlock()
			return entry, existing, true, errReverseTunnelPreCreateCollision
		}

		claimResult, cleanup := connection.TryClaimReverseTunnelIDDeferredCleanup(tunnelID)
		registry.routeMutex.Unlock()
		if cleanup != nil {
			cleanup()
		}
		if claimResult == core.ReverseTunnelIDClaimed {
			return nil, existing, true, errReverseTunnelPreCreateCollision
		}
		return nil, existing, true, errReverseTunnelPreCreateAlreadyClaimed
	}
	registry.mutex.Lock()
	entry = registry.entries[key]
	if entry != nil {
		delete(registry.entries, key)
		entry.detached = true
		close(entry.stop)
	}
	registry.mutex.Unlock()
	registry.routeMutex.Unlock()
	return entry, nil, false, nil
}

func (registry *reverseTunnelPreCreateRegistry) expireOrDiscardOnClose(entry *reverseTunnelPreCreateEntry) {
	timer := time.NewTimer(registry.ttl)
	defer timer.Stop()
	var reason error
	select {
	case <-entry.stop:
		return
	case <-entry.key.connection.Done():
	case <-timer.C:
		reason = errReverseTunnelPreCreateExpired
	}
	if registry.remove(entry) && reason != nil {
		rejectReverseTunnel(entry.key.connection, entry.key.tunnelID, reason)
	}
}

func (registry *reverseTunnelPreCreateRegistry) remove(entry *reverseTunnelPreCreateEntry) bool {
	if registry == nil || entry == nil {
		return false
	}
	registry.mutex.Lock()
	if registry.entries[entry.key] != entry {
		registry.mutex.Unlock()
		return false
	}
	frames := registry.detachAndReleaseLocked(entry)
	registry.mutex.Unlock()
	scrubReverseTunnelPreCreateFrames(frames)
	return true
}

func (registry *reverseTunnelPreCreateRegistry) release(entry *reverseTunnelPreCreateEntry) {
	if registry == nil || entry == nil {
		return
	}
	registry.mutex.Lock()
	frames := registry.detachAndReleaseLocked(entry)
	registry.mutex.Unlock()
	scrubReverseTunnelPreCreateFrames(frames)
}

func (registry *reverseTunnelPreCreateRegistry) detachAndReleaseLocked(entry *reverseTunnelPreCreateEntry) []*sliverpb.TunnelData {
	if entry.released {
		return nil
	}
	if !entry.detached {
		if registry.entries[entry.key] == entry {
			delete(registry.entries, entry.key)
		}
		entry.detached = true
		close(entry.stop)
	}
	entry.released = true
	usage := registry.perSession[entry.key.sessionID]
	if usage != nil {
		usage.entries--
		usage.frames -= len(entry.frames)
		usage.bytes -= entry.bytes
		if usage.entries == 0 && usage.frames == 0 && usage.bytes == 0 {
			delete(registry.perSession, entry.key.sessionID)
		}
	}
	registry.total.entries--
	registry.total.frames -= len(entry.frames)
	registry.total.bytes -= entry.bytes
	frames := make([]*sliverpb.TunnelData, 0, len(entry.frames)+1)
	for _, frame := range entry.frames {
		frames = append(frames, frame)
	}
	if entry.terminal != nil {
		frames = append(frames, entry.terminal)
	}
	entry.frames = nil
	entry.terminal = nil
	entry.bytes = 0
	return frames
}

func scrubReverseTunnelPreCreateFrames(frames []*sliverpb.TunnelData) {
	for _, frame := range frames {
		if frame == nil {
			continue
		}
		clear(frame.Data)
		frame.Data = nil
		frame.Rportfwd = nil
	}
}

func (entry *reverseTunnelPreCreateEntry) orderedData() []*sliverpb.TunnelData {
	if entry == nil {
		return nil
	}
	sequences := make([]uint64, 0, len(entry.frames))
	for sequence := range entry.frames {
		sequences = append(sequences, sequence)
	}
	sort.Slice(sequences, func(left int, right int) bool { return sequences[left] < sequences[right] })
	frames := make([]*sliverpb.TunnelData, 0, len(sequences))
	for _, sequence := range sequences {
		frames = append(frames, entry.frames[sequence])
	}
	return frames
}
