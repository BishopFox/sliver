package core

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
*/

import (
	"errors"
	"io"
	"log"
	"sync"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

const (
	// HTTP C2 may deliver up to 64 concurrent responses while a command-specific
	// reader is starting or briefly backpressured. Match the 128-frame generic
	// server/implant transport window so that ordinary bursts remain isolated and
	// bounded instead of resetting an otherwise healthy port forward.
	tunnelRecvBufferSize = 128
	tunnelRecvByteLimit  = tunnelRecvBufferSize * sliverpb.MaxTunnelFrameBytes
)

var (
	errClosedTunnel               = errors.New("closed tunnel")
	errTunnelReceiveQueueFull     = errors.New("tunnel receive queue capacity exceeded")
	errTunnelReceiveFrameTooLarge = errors.New("tunnel receive frame exceeds payload limit")
)

type tunnelWriteRequest struct {
	data   []byte
	result chan error
}

// TunnelIO - Duplex data tunnel, compatible with both io.ReadWriter
type TunnelIO struct {
	ID        uint64
	SessionID string

	Send chan []byte
	Recv chan []byte

	writeRequests  chan tunnelWriteRequest
	done           chan struct{}
	closeOnce      sync.Once
	recvFailed     chan struct{}
	recvFailedOnce sync.Once
	bound          chan struct{}
	boundOnce      sync.Once
	readMu         sync.Mutex
	remainder      []byte

	recvBudgetMu   sync.Mutex
	recvFrames     int
	recvBytes      int
	recvFrameLimit int
	recvByteLimit  int
	isOpen         bool
	mutex          *sync.RWMutex
	// readSelectHook is an unexported scheduling seam for deterministic tests of
	// the close-vs-final-frame select. Production instances leave it nil.
	readSelectHook func(doneSelected bool)
}

// NewTunnelIO - Single entry point for creating instance of new TunnelIO
func NewTunnelIO(tunnelID uint64, sessionID string) *TunnelIO {
	log.Printf("New tunnel!: %d", tunnelID)

	return &TunnelIO{
		ID:        tunnelID,
		SessionID: sessionID,
		Send:      make(chan []byte),
		// A tunnel can receive its first frames before the command-specific
		// reader is running. Keep that short startup burst from blocking the
		// single client TunnelLoop while retaining bounded backpressure.
		Recv:           make(chan []byte, tunnelRecvBufferSize),
		writeRequests:  make(chan tunnelWriteRequest),
		done:           make(chan struct{}),
		recvFailed:     make(chan struct{}),
		bound:          make(chan struct{}),
		recvFrameLimit: tunnelRecvBufferSize,
		recvByteLimit:  tunnelRecvByteLimit,
		mutex:          &sync.RWMutex{},
	}
}

// Write - Writer method for interface
func (tun *TunnelIO) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if !tun.IsOpen() {
		log.Printf("Warning: Write on closed tunnel %d", tun.ID)
		return 0, io.EOF
	}

	// This is necessary to avoid any race conditions on thay byte array
	dataCopy := make([]byte, len(data))
	n := copy(dataCopy, data)

	log.Printf("Write %d byte(s) on tunnel %d", n, tun.ID)

	select {
	case <-tun.done:
		return 0, io.EOF
	default:
	}

	request := tunnelWriteRequest{
		data:   dataCopy,
		result: make(chan error, 1),
	}
	select {
	case tun.writeRequests <- request:
	case <-tun.done:
		return 0, io.EOF
	}

	// A successful Write means the exact tunnel generation's gRPC stream Send
	// completed. This prevents a full-close owner from deleting the generation
	// after a channel handoff but before the final payload reaches the stream.
	select {
	case err := <-request.result:
		if err != nil {
			return 0, err
		}
		return n, nil
	case <-tun.done:
		// Prefer an already-published Send result when completion and close race.
		select {
		case err := <-request.result:
			if err != nil {
				return 0, err
			}
			return n, nil
		default:
			return 0, io.EOF
		}
	}
}

// Read - Reader method for interface
func (tun *TunnelIO) Read(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	tun.readMu.Lock()
	defer tun.readMu.Unlock()
	if len(tun.remainder) > 0 {
		return tun.copyRecvData(data, nil), nil
	}
	// Prefer already queued output over the close signal so callers can drain
	// the final frames before observing EOF.
	select {
	case recvData := <-tun.Recv:
		return tun.copyRecvData(data, recvData), nil
	default:
	}

	if tun.readSelectHook != nil {
		tun.readSelectHook(false)
	}
	select {
	case recvData := <-tun.Recv:
		return tun.copyRecvData(data, recvData), nil
	case <-tun.done:
		if tun.readSelectHook != nil {
			tun.readSelectHook(true)
		}
		// RecvData that won admission immediately before Close may have queued
		// after the first poll. Prefer that final admitted frame over EOF.
		select {
		case recvData := <-tun.Recv:
			return tun.copyRecvData(data, recvData), nil
		default:
			log.Printf("Warning: Read on closed tunnel %d", tun.ID)
			return 0, io.EOF
		}
	}
}

func (tun *TunnelIO) copyRecvData(data []byte, recvData []byte) int {
	if recvData != nil {
		tun.remainder = recvData
		log.Printf("Read frame with %d byte(s) on tunnel %d", len(recvData), tun.ID)
	}
	n := copy(data, tun.remainder)
	tun.remainder = tun.remainder[n:]
	tun.releaseRecvBudget(n, len(tun.remainder) == 0)
	if len(tun.remainder) == 0 {
		tun.remainder = nil
	}
	return n
}

func (tun *TunnelIO) releaseRecvBudget(bytes int, frameComplete bool) {
	tun.recvBudgetMu.Lock()
	if bytes >= tun.recvBytes {
		tun.recvBytes = 0
	} else {
		tun.recvBytes -= bytes
	}
	if frameComplete && tun.recvFrames > 0 {
		tun.recvFrames--
	}
	tun.recvBudgetMu.Unlock()
}

// Close - Close tunnel IO operations
func (tun *TunnelIO) Close() error {
	tun.mutex.Lock()
	wasOpen := tun.isOpen
	tun.isOpen = false
	tun.closeOnce.Do(func() {
		close(tun.done)
	})
	tun.mutex.Unlock()

	if !wasOpen {
		log.Printf("Warning: Close on closed tunnel %d", tun.ID)

		// I guess we can ignore it and don't return any error
		return nil
	}

	log.Printf("Close tunnel %d", tun.ID)

	return nil
}

// Done is closed when the tunnel is closed. The public data channels remain
// open so concurrent producers cannot panic by racing a channel close.
func (tun *TunnelIO) Done() <-chan struct{} {
	return tun.done
}

// receiveFailed is closed only for local receive-admission failure. Ordinary
// peer close leaves it open so owners can still drain already admitted final
// frames before closing their local endpoint.
func (tun *TunnelIO) receiveFailed() <-chan struct{} {
	return tun.recvFailed
}

func (tun *TunnelIO) failReceive() {
	tun.recvFailedOnce.Do(func() {
		close(tun.recvFailed)
	})
	_ = tun.Close()
}

// Bound is closed after the server acknowledges the tunnel's streaming bind.
func (tun *TunnelIO) Bound() <-chan struct{} {
	return tun.bound
}

func (tun *TunnelIO) markBound() {
	tun.boundOnce.Do(func() {
		close(tun.bound)
	})
}

func (tun *TunnelIO) IsOpen() bool {
	tun.mutex.RLock()
	defer tun.mutex.RUnlock()

	return tun.isOpen
}

func (tun *TunnelIO) Open() error {
	tun.mutex.Lock()
	defer tun.mutex.Unlock()

	if tun.isOpen {
		return errors.New("tunnel relady in open state")
	}
	select {
	case <-tun.done:
		return errClosedTunnel
	default:
	}

	log.Printf("Open tunnel %d", tun.ID)

	tun.isOpen = true

	return nil
}

// RecvData admits data to the bounded receive queue without blocking the
// process-wide TunnelLoop. The reservation includes unread short-buffer
// remainder, so both the frame count and byte footprint remain bounded.
func (tun *TunnelIO) RecvData(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if len(data) > sliverpb.MaxTunnelFrameBytes {
		return errTunnelReceiveFrameTooLarge
	}

	tun.mutex.RLock()
	defer tun.mutex.RUnlock()
	if !tun.isOpen {
		return errClosedTunnel
	}

	return tun.queueRecvData(data)
}

// queueRecvData completes admission after the caller has established that the
// tunnel is open. Keeping this step separate also lets tests hold the Read done
// branch at the exact historical race point without relying on scheduler luck.
func (tun *TunnelIO) queueRecvData(data []byte) error {
	dataCopy := append([]byte(nil), data...)
	tun.recvBudgetMu.Lock()
	defer tun.recvBudgetMu.Unlock()
	if tun.recvFrames >= tun.recvFrameLimit || tun.recvBytes+len(dataCopy) > tun.recvByteLimit {
		return errTunnelReceiveQueueFull
	}

	tun.recvFrames++
	tun.recvBytes += len(dataCopy)
	select {
	case tun.Recv <- dataCopy:
		return nil
	default:
		tun.recvFrames--
		tun.recvBytes -= len(dataCopy)
		return errTunnelReceiveQueueFull
	}
}
