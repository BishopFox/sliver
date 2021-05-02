package console

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
	"context"
	"errors"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

var (
	// ClientHist - Client console history
	ClientHist = &ClientHistory{LinesSinceStart: 1}
	// UserHist - User-wide history
	UserHist = &UserHistory{LinesSinceStart: 1}
	// SessionHist - The current implant session history
	SessionHist = &SessionHistory{LinesSinceStart: 1}
)

// ClientHistory - Writes and queries only the Client's history
type ClientHistory struct {
	LinesSinceStart int // Keeps count of line since session
	items           []string
}

// Write - Sends the last command to the server for saving
func (h *ClientHistory) Write(s string) (int, error) {

	res, err := transport.RPC.AddToHistory(context.Background(),
		&clientpb.AddCmdHistoryRequest{Line: s})
	if err != nil {
		return 0, err
	}

	// The server sent us back the whole user history,
	// so we give it to the user history (the latter never
	// actually uses its Write() method.
	UserHist.cache = res.User

	h.items = append(h.items, s)
	return len(h.items), nil
}

// GetLine returns a line from history
func (h *ClientHistory) GetLine(i int) (string, error) {
	if len(h.items) == 0 {
		return "", nil
	}
	return h.items[i], nil
}

// Len returns the number of lines in history
func (h *ClientHistory) Len() int {
	return len(h.items)
}

// Dump returns the entire history
func (h *ClientHistory) Dump() interface{} {
	return h.items
}

// UserHistory - Only in charge of queries for the User's history
type UserHistory struct {
	LinesSinceStart int // Keeps count of line since session

	// cache - In order to avoid making hundreds of
	// requests to the server, we cache the user history.
	// This cache is refreshed each time we request the history.
	cache []string
}

// RefreshLines - Get the lines for the user command history
func (h *UserHistory) RefreshLines(lines []string) {
	h.cache = lines
}

// getUserHistory - On console startup, request
// the user-wide hitory and keep it in the cache.
func getUserHistory() {
	res, err := transport.RPC.GetHistory(context.Background(),
		&clientpb.HistoryRequest{
			AllConsoles: true,
		})
	if err != nil {
		return
	}
	UserHist.cache = res.User
}

// Write - Adds a line to user-wide command history.
// Due to readline functioning, this function is actually never
// called, and every new command is added through the client history.
func (h *UserHistory) Write(s string) (int, error) {

	res, err := transport.RPC.AddToHistory(context.Background(),
		&clientpb.AddCmdHistoryRequest{Line: s})
	if err != nil {
		return 0, err
	}

	// The server sent us back the whole user history,
	// which we cache for reading when GetLine()
	h.cache = res.User

	if !res.Doublon {
		h.LinesSinceStart++
	}
	return h.LinesSinceStart, nil
}

// GetLine returns a line from user-wide history, directly read from the cache
func (h *UserHistory) GetLine(i int) (string, error) {
	h.LinesSinceStart = len(h.cache)
	if i > len(h.cache) {
		return "", errors.New("index out of range")
	}
	return h.cache[i], nil
}

// Len returns the number of lines in history
func (h *UserHistory) Len() int {
	return len(h.cache)
}

// Dump returns the entire history
func (h *UserHistory) Dump() interface{} {
	return nil
}

// SessionHistory - Stores and asks the current Session history
type SessionHistory struct {
	LinesSinceStart int // Keeps count of line since session

	// cache - In order to avoid making hundreds of
	// requests to the server, we cache the user history.
	// This cache is refreshed each time we request the history.
	cache []string
}

// RefreshLines - Get the lines for the session command history
func (h *SessionHistory) RefreshLines(lines []string) {
	h.cache = lines
}

// Write - Adds a line to user-wide command history.
// Due to readline functioning, this function is actually never
// called, and every new command is added through the client history.
func (h *SessionHistory) Write(s string) (int, error) {

	res, err := transport.RPC.AddToHistory(context.Background(),
		&clientpb.AddCmdHistoryRequest{
			Line:    s,
			Session: core.ActiveSession.Session,
		})
	if err != nil {
		return 0, err
	}

	// The server sent us back the whole session history,
	// which we cache for reading when GetLine()
	h.cache = res.Sliver

	if !res.Doublon {
		h.LinesSinceStart++
	}
	return h.LinesSinceStart, nil
}

// GetLine returns a line from user-wide history, directly read from the cache
func (h *SessionHistory) GetLine(i int) (string, error) {

	// The session history cache is continuously refreshed, because several players
	// might be using the same implant at once, or one with many clients.
	// h.cache = core.GetSessionHistory()

	h.LinesSinceStart = len(h.cache)
	if i > len(h.cache) {
		return "", errors.New("index out of range")
	}
	return h.cache[i], nil
}

// Len returns the number of lines in history
func (h *SessionHistory) Len() int {
	return len(h.cache)
}

// Dump returns the entire history
func (h *SessionHistory) Dump() interface{} {
	return nil
}
