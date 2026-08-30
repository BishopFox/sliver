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
	"bytes"
	"errors"
	"io"
	"log"
	"sync"
)

const tunnelRecvBufferSize = 16

var errClosedTunnel = errors.New("closed tunnel")

// TunnelIO - Duplex data tunnel, compatible with both io.ReadWriter
type TunnelIO struct {
	ID        uint64
	SessionID string

	Send chan []byte
	Recv chan []byte

	done      chan struct{}
	closeOnce sync.Once
	bound     chan struct{}
	boundOnce sync.Once
	isOpen    bool
	mutex     *sync.RWMutex
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
		Recv:  make(chan []byte, tunnelRecvBufferSize),
		done:  make(chan struct{}),
		bound: make(chan struct{}),
		mutex: &sync.RWMutex{},
	}
}

// Write - Writer method for interface
func (tun *TunnelIO) Write(data []byte) (int, error) {
	if !tun.IsOpen() {
		log.Printf("Warning: Write on closed tunnel %d", tun.ID)
		return 0, io.EOF
	}

	// This is necessary to avoid any race conditions on thay byte array
	dataCopy := make([]byte, len(data))
	n := copy(dataCopy, data)

	log.Printf("Write %d bytes", n)
	log.Printf("This bytes is: %s", dataCopy)

	select {
	case <-tun.done:
		return 0, io.EOF
	default:
	}

	select {
	case tun.Send <- dataCopy:
		return n, nil
	case <-tun.done:
		return 0, io.EOF
	}
}

// Read - Reader method for interface
func (tun *TunnelIO) Read(data []byte) (int, error) {
	// Prefer already queued output over the close signal so callers can drain
	// the final frames before observing EOF.
	select {
	case recvData := <-tun.Recv:
		return tun.copyRecvData(data, recvData), nil
	default:
	}

	select {
	case recvData := <-tun.Recv:
		return tun.copyRecvData(data, recvData), nil
	case <-tun.done:
		log.Printf("Warning: Read on closed tunnel %d", tun.ID)
		return 0, io.EOF
	}
}

func (tun *TunnelIO) copyRecvData(data []byte, recvData []byte) int {
	var buff bytes.Buffer
	log.Printf("Read %d bytes", len(recvData))
	buff.Write(recvData)

	return copy(data, buff.Bytes())
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

// RecvData - safely queues data on the bounded internal Recv channel.
// It blocks only when that startup buffer is full.
func (tun *TunnelIO) RecvData(data []byte) error {
	if !tun.IsOpen() {
		return errClosedTunnel
	}

	select {
	case <-tun.done:
		return errClosedTunnel
	default:
	}

	select {
	case tun.Recv <- data:
		return nil
	case <-tun.done:
		return errClosedTunnel
	}
}
