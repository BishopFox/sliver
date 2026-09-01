package handlers

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	------------------------------------------------------------------------

	WARNING: These functions can be invoked by remote implants without user interaction

*/

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/core/rtunnels"
)

type ServerHandler func(*core.ImplantConnection, []byte) *sliverpb.Envelope

var (
	tunnelHandlerMutex = &sync.Mutex{}
	// reverseTunnelOpenings lets tunnel frames wait for their own bounded broker
	// dial without holding the global tunnel handler mutex or blocking unrelated
	// tunnels. Values are *reverseTunnelOpening keyed by tunnel ID.
	reverseTunnelOpenings        sync.Map
	reverseTunnelOpeningAttempts = newReverseTunnelAdmission(
		maxReverseTunnelOpeningsPerSession,
		maxReverseTunnelOpeningsGlobal,
	)
	reverseTunnelOpeningWaiters = newReverseTunnelAdmission(
		maxReverseTunnelOpeningWaitersPerSession,
		maxReverseTunnelOpeningWaitersGlobal,
	)
	reverseTunnelRejectionSlots = make(chan struct{}, maxConcurrentReverseTunnelRejections)
)

const (
	// Match the full bounded tunnel reorder/transport concurrency window. A
	// legitimate source can fill every stream while the first broker dial is
	// still completing; admission below this value turns ordinary traffic into
	// cancellation of an otherwise authorized opening.
	maxReverseTunnelOpeningWaiters           = 128
	maxReverseTunnelOpeningsPerSession       = 32
	maxReverseTunnelOpeningsGlobal           = 128
	maxReverseTunnelOpeningWaitersPerSession = 128
	maxReverseTunnelOpeningWaitersGlobal     = 512
	maxTunnelDataMessageBytes                = 128 * 1024
	maxConcurrentReverseTunnelRejections     = 128
)

var (
	reverseTunnelOpeningWaitTimeout = rtunnels.DefaultDialTimeout + time.Second
	reverseTunnelRejectionTimeout   = rtunnels.DefaultDialTimeout
)

var (
	errReverseTunnelOpeningLimit       = errors.New("reverse tunnel opening limit reached")
	errReverseTunnelOpeningWaiterLimit = errors.New("reverse tunnel opening waiter limit reached")
	errReverseTunnelOpeningWaitTimeout = errors.New("reverse tunnel opening wait timed out")
)

type reverseTunnelOpening struct {
	sessionID string
	ready     chan struct{}
	waiters   chan struct{}
	cancel    func()
	closing   atomic.Bool
	terminal  atomic.Bool
}

func (opening *reverseTunnelOpening) claimTerminal() bool {
	return opening != nil && opening.terminal.CompareAndSwap(false, true)
}

func newReverseTunnelOpening(sessionID string, cancel func()) *reverseTunnelOpening {
	return &reverseTunnelOpening{
		sessionID: sessionID,
		ready:     make(chan struct{}),
		waiters:   make(chan struct{}, maxReverseTunnelOpeningWaiters),
		cancel:    cancel,
	}
}

func (opening *reverseTunnelOpening) requestClose() {
	if opening == nil {
		return
	}
	opening.closing.Store(true)
	if opening.cancel != nil {
		opening.cancel()
	}
}

func (opening *reverseTunnelOpening) wait(timeout time.Duration) error {
	if opening == nil {
		return errReverseTunnelOpeningWaitTimeout
	}
	waiterAdmission := reverseTunnelOpeningWaiters
	if !waiterAdmission.acquire(opening.sessionID) {
		return errReverseTunnelOpeningWaiterLimit
	}
	defer waiterAdmission.release(opening.sessionID)

	select {
	case opening.waiters <- struct{}{}:
		defer func() { <-opening.waiters }()
	default:
		return errReverseTunnelOpeningWaiterLimit
	}
	return opening.waitReady(timeout)
}

// waitReady is used after the opening's single terminal slot is claimed. That
// reserved slot lets a legitimate terminal wait even when every data-frame
// waiter is occupied without allowing duplicate terminals to create unbounded
// blocked handlers.
func (opening *reverseTunnelOpening) waitReady(timeout time.Duration) error {
	if opening == nil {
		return errReverseTunnelOpeningWaitTimeout
	}
	if timeout <= 0 {
		timeout = reverseTunnelOpeningWaitTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-opening.ready:
		return nil
	case <-timer.C:
		return errReverseTunnelOpeningWaitTimeout
	}
}

type reverseTunnelAdmission struct {
	mutex      sync.Mutex
	perSession map[string]int
	total      int
	sessionMax int
	globalMax  int
}

func newReverseTunnelAdmission(sessionMax int, globalMax int) *reverseTunnelAdmission {
	return &reverseTunnelAdmission{
		perSession: map[string]int{},
		sessionMax: sessionMax,
		globalMax:  globalMax,
	}
}

func (admission *reverseTunnelAdmission) acquire(sessionID string) bool {
	admission.mutex.Lock()
	defer admission.mutex.Unlock()
	if admission.total >= admission.globalMax || admission.perSession[sessionID] >= admission.sessionMax {
		return false
	}
	admission.total++
	admission.perSession[sessionID]++
	return true
}

func (admission *reverseTunnelAdmission) release(sessionID string) {
	admission.mutex.Lock()
	defer admission.mutex.Unlock()
	if admission.perSession[sessionID] == 0 {
		return
	}
	admission.perSession[sessionID]--
	admission.total--
	if admission.perSession[sessionID] == 0 {
		delete(admission.perSession, sessionID)
	}
}

// GetHandlers - Returns a map of server-side msg handlers
func GetHandlers() map[uint32]ServerHandler {
	return map[uint32]ServerHandler{
		// Sessions
		sliverpb.MsgRegister:    registerSessionHandler,
		sliverpb.MsgTunnelData:  tunnelDataHandler,
		sliverpb.MsgTunnelClose: tunnelCloseHandler,
		sliverpb.MsgPing:        pingHandler,
		sliverpb.MsgSocksData:   socksDataHandler,

		// Beacons
		sliverpb.MsgBeaconRegister: beaconRegisterHandler,
		sliverpb.MsgBeaconTasks:    beaconTasksHandler,

		// Pivots
		sliverpb.MsgPivotPeerEnvelope: pivotPeerEnvelopeHandler,
		sliverpb.MsgPivotPeerFailure:  pivotPeerFailureHandler,
	}
}

// GetNonPivotHandlers - Server handlers for pivot connections, its important
// to avoid a pivot handler from calling a pivot handler and causing a recursive
// call stack
func GetNonPivotHandlers() map[uint32]ServerHandler {
	return map[uint32]ServerHandler{
		// Sessions
		sliverpb.MsgRegister:    registerSessionHandler,
		sliverpb.MsgTunnelData:  tunnelDataHandler,
		sliverpb.MsgTunnelClose: tunnelCloseHandler,
		sliverpb.MsgPing:        pingHandler,
		sliverpb.MsgSocksData:   socksDataHandler,

		// Beacons - Not currently supported in pivots
	}
}
