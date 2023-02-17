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

// TunnelIO - Duplex data tunnel, compatible with both io.ReadWriter
type TunnelIO struct {
	ID        uint64
	SessionID string

	Send chan []byte
	Recv chan []byte

	isOpen bool
	mutex  *sync.RWMutex
}

// NewTunnelIO - Single entry point for creating instance of new TunnelIO
func NewTunnelIO(tunnelID uint64, sessionID string) *TunnelIO {
	log.Printf("New tunnel!: %d", tunnelID)

	return &TunnelIO{
		ID:        tunnelID,
		SessionID: sessionID,
		Send:      make(chan []byte),
		Recv:      make(chan []byte),
		isOpen:    false,
		mutex:     &sync.RWMutex{},
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

	tun.Send <- dataCopy

	return n, nil
}

// Read - Reader method for interface
func (tun *TunnelIO) Read(data []byte) (int, error) {
	recvData, ok := <-tun.Recv
	if !ok {
		log.Printf("Warning: Read on closed tunnel %d", tun.ID)
		return 0, io.EOF
	}

	var buff bytes.Buffer
	log.Printf("Read %d bytes", len(recvData))
	buff.Write(recvData)

	n := copy(data, buff.Bytes())
	return n, nil
}

// Close - Close tunnel IO operations
func (tun *TunnelIO) Close() error {
	tun.mutex.Lock()
	defer tun.mutex.Unlock()

	if !tun.isOpen {
		log.Printf("Warning: Close on closed tunnel %d", tun.ID)

		// I guess we can ignore it and don't return any error
		return nil
	}

	log.Printf("Close tunnel %d", tun.ID)

	tun.isOpen = false

	close(tun.Recv)
	close(tun.Send)

	return nil
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

	log.Printf("Open tunnel %d", tun.ID)

	tun.isOpen = true

	return nil
}

// RecvData - safe way to send data to internal Recv channel. Blocking.
func (tun *TunnelIO) RecvData(data []byte) error {
	tun.mutex.Lock()
	defer tun.mutex.Unlock()

	if !tun.isOpen {
		return errors.New("closed tunnel")
	}

	tun.Recv <- data

	return nil
}
