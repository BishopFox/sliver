package pivotclients

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
*/

import (
	"errors"
	"net"
	"net/url"
	"sync"
	"time"
)

var (
	ErrFailedWrite = errors.New("write failed")

	defaultDeadline = time.Second * 10
)

// ParseTCPPivotOptions - Parse the options for the TCP pivot from a C2 URL
func ParseTCPPivotOptions(uri *url.URL) *TCPPivotOptions {
	readDeadline, err := time.ParseDuration(uri.Query().Get("read-deadline"))
	if err != nil {
		readDeadline = defaultDeadline
	}
	writeDeadline, err := time.ParseDuration(uri.Query().Get("write-deadline"))
	if err != nil {
		writeDeadline = defaultDeadline
	}
	return &TCPPivotOptions{
		ReadDeadline:  readDeadline,
		WriteDeadline: writeDeadline,
	}
}

// TCPPivotOptions - Options for the TCP pivot
type TCPPivotOptions struct {
	ReadDeadline  time.Duration
	WriteDeadline time.Duration
}

// TCPPivotStartSession - Start a TCP pivot session with a peer
func TCPPivotStartSession(peer string, opts *TCPPivotOptions) (*NetConnPivotClient, error) {
	conn, err := net.Dial("tcp", peer)
	if err != nil {
		return nil, err
	}
	pivot := &NetConnPivotClient{
		conn:       conn,
		readMutex:  &sync.Mutex{},
		writeMutex: &sync.Mutex{},

		readDeadline:  opts.ReadDeadline,
		writeDeadline: opts.WriteDeadline,
	}
	err = pivot.KeyExchange()
	if err != nil {
		conn.Close()
		return nil, err
	}
	return pivot, nil
}
