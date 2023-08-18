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
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

type ActiveTarget struct {
	session    *clientpb.Session
	beacon     *clientpb.Beacon
	observers  map[int]Observer
	observerID int
	con        *SliverClient
	hist       *implantHistory
}

func newActiveTarget() *ActiveTarget {
	at := &ActiveTarget{
		observers:  map[int]Observer{},
		observerID: 0,
	}

	return at
}

// GetSession returns the session matching an ID, either by prefix or strictly.
func (con *SliverClient) GetSession(arg string) *clientpb.Session {
	sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintWarnf("%s", err)
		return nil
	}
	for _, session := range sessions.GetSessions() {
		if session.Name == arg || strings.HasPrefix(session.ID, arg) {
			return session
		}
	}
	return nil
}

// GetBeacon returns the beacon matching an ID, either by prefix or strictly.
func (con *SliverClient) GetBeacon(arg string) *clientpb.Beacon {
	beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintWarnf("%s", err)
		return nil
	}
	for _, session := range beacons.GetBeacons() {
		if session.Name == arg || strings.HasPrefix(session.ID, arg) {
			return session
		}
	}
	return nil
}

// GetSessionsByName - Return all sessions for an Implant by name.
func (con *SliverClient) GetSessionsByName(name string) []*clientpb.Session {
	sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
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

// GetActiveSessionConfig - Get the active sessions's config
// TODO: Switch to query config based on ConfigID.
func (con *SliverClient) GetActiveSessionConfig() *clientpb.ImplantConfig {
	session := con.ActiveTarget.GetSession()
	if session == nil {
		return nil
	}
	c2s := []*clientpb.ImplantC2{}
	c2s = append(c2s, &clientpb.ImplantC2{
		URL:      session.GetActiveC2(),
		Priority: uint32(0),
	})
	config := &clientpb.ImplantConfig{
		Name:    session.GetName(),
		GOOS:    session.GetOS(),
		GOARCH:  session.GetArch(),
		Debug:   true,
		Evasion: session.GetEvasion(),

		MaxConnectionErrors: uint32(1000),
		ReconnectInterval:   int64(60),
		Format:              clientpb.OutputFormat_SHELLCODE,
		IsSharedLib:         true,
		C2:                  c2s,
	}
	return config
}

// GetSessionInteractive - Get the active target(s).
func (s *ActiveTarget) GetInteractive() (*clientpb.Session, *clientpb.Beacon) {
	if s.session == nil && s.beacon == nil {
		fmt.Printf(Warn + "Please select a session or beacon via `use`\n")
		return nil, nil
	}
	return s.session, s.beacon
}

// GetSessionInteractive - Get the active target(s).
func (s *ActiveTarget) Get() (*clientpb.Session, *clientpb.Beacon) {
	return s.session, s.beacon
}

// GetSessionInteractive - GetSessionInteractive the active session.
func (s *ActiveTarget) GetSessionInteractive() *clientpb.Session {
	if s.session == nil {
		fmt.Printf(Warn + "Please select a session via `use`\n")
		return nil
	}
	return s.session
}

// GetSession - Same as GetSession() but doesn't print a warning.
func (s *ActiveTarget) GetSession() *clientpb.Session {
	return s.session
}

// GetBeaconInteractive - Get beacon interactive the active session.
func (s *ActiveTarget) GetBeaconInteractive() *clientpb.Beacon {
	if s.beacon == nil {
		fmt.Printf(Warn + "Please select a beacon via `use`\n")
		return nil
	}
	return s.beacon
}

// GetBeacon - Same as GetBeacon() but doesn't print a warning.
func (s *ActiveTarget) GetBeacon() *clientpb.Beacon {
	return s.beacon
}

// IsSession - Is the current target a session?
func (s *ActiveTarget) IsSession() bool {
	return s.session != nil
}

// AddObserver - Observers to notify when the active session changes.
func (s *ActiveTarget) AddObserver(observer Observer) int {
	s.observerID++
	s.observers[s.observerID] = observer
	return s.observerID
}

func (s *ActiveTarget) RemoveObserver(observerID int) {
	delete(s.observers, observerID)
}

func (s *ActiveTarget) Request(cmd *cobra.Command) *commonpb.Request {
	if s.session == nil && s.beacon == nil {
		return nil
	}

	// One less than the gRPC timeout so that the server should timeout first
	timeOutF := int64(defaultTimeout) - 1
	if cmd != nil {
		timeOutF, _ = cmd.Flags().GetInt64("timeout")
	}
	timeout := (int64(time.Second) * timeOutF) - 1

	req := &commonpb.Request{}
	req.Timeout = timeout

	if s.session != nil {
		req.Async = false
		req.SessionID = s.session.ID
	}
	if s.beacon != nil {
		req.Async = true
		req.BeaconID = s.beacon.ID
	}
	return req
}

// Set - Change the active session.
func (s *ActiveTarget) Set(session *clientpb.Session, beacon *clientpb.Beacon) {
	if session != nil && beacon != nil {
		s.con.PrintErrorf("cannot set both an active beacon and an active session")
		return
	}

	defer s.con.FilterCommands(s.con.App.ActiveMenu().Command)

	// Backgrounding
	if session == nil && beacon == nil {
		s.session = nil
		s.beacon = nil
		for _, observer := range s.observers {
			observer(s.session, s.beacon)
		}

		if s.con.isCLI {
			return
		}

		// Switch back to server menu.
		if s.con.App.ActiveMenu().Name() == consts.ImplantMenu {
			s.con.App.SwitchMenu(consts.ServerMenu)
		}

		return
	}

	// Foreground
	if session != nil {
		s.session = session
		s.beacon = nil
		for _, observer := range s.observers {
			observer(s.session, s.beacon)
		}
	} else if beacon != nil {
		s.beacon = beacon
		s.session = nil
		for _, observer := range s.observers {
			observer(s.session, s.beacon)
		}
	}

	// Update menus, prompts and commands
	if s.con.App.ActiveMenu().Name() != consts.ImplantMenu {
		s.con.App.SwitchMenu(consts.ImplantMenu)
	}
}

// Background - Background the active session.
func (s *ActiveTarget) Background() {
	defer s.con.App.ShowCommands()

	s.session = nil
	s.beacon = nil
	for _, observer := range s.observers {
		observer(nil, nil)
	}

	// Switch back to server menu.
	if !s.con.isCLI && s.con.App.ActiveMenu().Name() == consts.ImplantMenu {
		s.con.App.SwitchMenu(consts.ServerMenu)
	}
}
