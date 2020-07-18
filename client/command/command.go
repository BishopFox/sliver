package command

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
	"fmt"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/desertbit/grumble"

	"gopkg.in/AlecAivazis/survey.v1"
)

const (
	// ANSI Colors
	normal    = "\033[0m"
	black     = "\033[30m"
	red       = "\033[31m"
	green     = "\033[32m"
	orange    = "\033[33m"
	blue      = "\033[34m"
	purple    = "\033[35m"
	cyan      = "\033[36m"
	gray      = "\033[37m"
	bold      = "\033[1m"
	clearln   = "\r\x1b[2K"
	upN       = "\033[%dA"
	downN     = "\033[%dB"
	underline = "\033[4m"

	// Info - Display colorful information
	Info = bold + cyan + "[*] " + normal
	// Warn - Warn a user
	Warn = bold + red + "[!] " + normal
	// Debug - Display debug information
	Debug = bold + purple + "[-] " + normal
	// Woot - Display success
	Woot = bold + green + "[$] " + normal
)

var (

	// ActiveSession - The current sliver we're interacting with
	ActiveSession = &activeSession{
		observers:  map[int]Observer{},
		observerID: 0,
	}

	stdinReadTimeout = 10 * time.Millisecond
)

// Observer - A function to call when the sessions changes
type Observer func(*clientpb.Session)

type activeSession struct {
	session    *clientpb.Session
	observers  map[int]Observer
	observerID int
}

// GetInteractive - GetInteractive the active session
func (s *activeSession) GetInteractive() *clientpb.Session {
	if s.session == nil {
		fmt.Printf(Warn + "Please select an active session via `use`\n")
		return nil
	}
	return s.session
}

// Get - Same as Get() but doesn't print a warning
func (s *activeSession) Get() *clientpb.Session {
	if s.session == nil {
		return nil
	}
	return s.session
}

// AddObserver - Observers to notify when the active session changes
func (s *activeSession) AddObserver(observer Observer) int {
	s.observerID++
	s.observers[s.observerID] = observer
	return s.observerID
}

func (s *activeSession) RemoveObserver(observerID int) {
	if _, ok := s.observers[observerID]; ok {
		delete(s.observers, observerID)
	}
}

func (s *activeSession) Request(ctx *grumble.Context) *commonpb.Request {
	if s.session == nil {
		return nil
	}
	timeout := int(time.Second) * ctx.Flags.Int("timeout")
	return &commonpb.Request{
		SessionID: s.session.ID,
		Timeout:   int64(timeout),
	}
}

// Set - Change the active session
func (s *activeSession) Set(session *clientpb.Session) {
	s.session = session
	for _, observer := range s.observers {
		observer(s.session)
	}
}

// Background - Background the active session
func (s *activeSession) Background() {
	s.session = nil
	for _, observer := range s.observers {
		observer(nil)
	}
}

// GetSession - Get session by session ID or name
func GetSession(arg string, rpc rpcpb.SliverRPCClient) *clientpb.Session {
	sessions, err := rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return nil
	}
	for _, session := range sessions.GetSessions() {
		if session.Name == arg || fmt.Sprintf("%d", session.ID) == arg {
			return session
		}
	}
	return nil
}

// GetSessionsByName - Return all sessions for an Implant by name
func GetSessionsByName(name string, rpc rpcpb.SliverRPCClient) []*clientpb.Session {
	sessions, err := rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return nil
	}
	matched := []*clientpb.Session{}
	for _, session := range sessions.GetSessions() {
		if session.Name == name {
			matched = append(matched, session)
		}
	}
	return matched
}

// This should be called for any dangerous (OPSEC-wise) functions
func isUserAnAdult() bool {
	confirm := false
	prompt := &survey.Confirm{Message: "This action is bad OPSEC, are you an adult?"}
	survey.AskOne(prompt, &confirm, nil)
	return confirm
}
