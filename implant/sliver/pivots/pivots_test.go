package pivots

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

func framePrefix(n uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, n)
	return b
}

// readWithTimeout runs read() with a deadline so a regression (an unbounded
// read that blocks waiting for a bogus multi-GB body) fails fast instead of
// hanging the whole test binary.
func readWithTimeout(t *testing.T, read func() ([]byte, error)) ([]byte, error) {
	t.Helper()
	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		d, e := read()
		ch <- result{d, e}
	}()
	select {
	case r := <-ch:
		return r.data, r.err
	case <-time.After(3 * time.Second):
		t.Fatal("read() did not return in time (possible unbounded read)")
		return nil, nil
	}
}

// A corrupted/desynced 4-byte length prefix that reads as a huge value must be
// rejected before allocating, instead of triggering a multi-GB allocation that
// crashes the implant (BishopFox/sliver#1452).
func TestNetConnPivotReadRejectsOversizedFrame(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	p := &NetConnPivot{conn: server, readMutex: &sync.Mutex{}}

	go func() {
		// Only the length prefix is ever sent; no body follows.
		_, _ = client.Write(framePrefix(uint32(MaxFrameLength) + 1))
	}()

	if _, err := readWithTimeout(t, p.read); !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("expected ErrFrameTooLarge, got %v", err)
	}
}

// A well-formed frame within the bound must still round-trip unchanged.
func TestNetConnPivotReadAcceptsValidFrame(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	p := &NetConnPivot{conn: server, readMutex: &sync.Mutex{}}

	payload := []byte("hello pivot")
	go func() {
		_, _ = client.Write(framePrefix(uint32(len(payload))))
		_, _ = client.Write(payload)
	}()

	data, err := readWithTimeout(t, p.read)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != string(payload) {
		t.Fatalf("got %q, want %q", data, payload)
	}
}
