package commands

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
	"fmt"
	"sort"
	"strconv"
	"time"

	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/evilsocket/islazy/tui"
)

// Main & Sliver context available commands
// ----------------------------------------------------------------------------------------------------------

// Sessions - Root command for managing sessions. Prints registered sessions by default.
type Sessions struct{}

// Execute - Prints registered sessions if no sub commands invoked.
func (s *Sessions) Execute(args []string) (err error) {

	// Get a map of all sessions
	sessions, err := transport.RPC.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return
	}
	sessionsMap := map[uint32]*clientpb.Session{}
	for _, session := range sessions.GetSessions() {
		sessionsMap[session.ID] = session
	}

	// Print all sessions
	if 0 < len(sessionsMap) {
		printSessions(sessionsMap)
	} else {
		fmt.Printf(util.Info + "No sessions \n")
	}

	return
}

// SessionsKill - Kill one or more sessions that are not mandatorily the current one.
type SessionsKill struct {
	Positional struct {
		SessionID []uint32 `description:"Session ID (multiple values accepted)" `
	} `positional-args:"yes" required:"true"`
}

// Execute - Kill one or more sessions.
func (sk *SessionsKill) Execute(args []string) (err error) {

	// Get a map of all sessions
	sessions, err := transport.RPC.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return
	}
	sessionsMap := map[uint32]*clientpb.Session{}
	for _, session := range sessions.GetSessions() {
		sessionsMap[session.ID] = session
	}

	// Kill each ID
	for _, id := range sk.Positional.SessionID {
		sess, ok := sessionsMap[id]
		if !ok || sess == nil {
			fmt.Printf(util.Error+"Invalid session ID: %d\n", id)
		}

		// Kill session
		err = killSession(sess, transport.RPC)

		// Change context if we are killing the current session
		active := cctx.Context.Sliver
		if active != nil && sess.ID == active.ID {
			cctx.Context.Menu = cctx.Server
			cctx.Context.Sliver = nil
		}
	}
	return
}

// SessionsKillAll - Kill all sessions
type SessionsKillAll struct{}

// Execute - Kill all sessions
func (ka *SessionsKillAll) Execute(args []string) (err error) {

	// Get a map of all sessions
	sessions, err := transport.RPC.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return
	}
	sessionsMap := map[uint32]*clientpb.Session{}
	for _, session := range sessions.GetSessions() {
		sessionsMap[session.ID] = session
	}

	// Kill all IDs
	for i := range sessionsMap {
		sess, ok := sessionsMap[i]
		if !ok || sess == nil {
			fmt.Printf(util.Error+"Invalid session ID: %d\n", i)
		}

		// Kill session
		err = killSession(sess, transport.RPC)

		// Change context if we are killing the current session
		active := cctx.Context.Sliver
		if active != nil && sess.ID == active.ID {
			cctx.Context.Menu = cctx.Server
			cctx.Context.Sliver = nil
		}
	}

	return
}

// SessionsClean - Clean sessions marked dead
type SessionsClean struct{}

// Execute - Clean sessions marked dead
func (ka *SessionsClean) Execute(args []string) (err error) {

	// Get a map of all sessions
	sessions, err := transport.RPC.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return
	}
	sessionsMap := map[uint32]*clientpb.Session{}
	for _, session := range sessions.GetSessions() {
		sessionsMap[session.ID] = session
	}

	// Kill all IDs
	for i := range sessionsMap {
		sess, ok := sessionsMap[i]
		if !ok || sess == nil {
			fmt.Printf(util.Error+"Invalid session ID: %d\n", i)
		}

		if sess.IsDead {
			// Kill session
			err = killSession(sess, transport.RPC)

			// Change context if we are killing the current session
			active := cctx.Context.Sliver
			if active != nil && sess.ID == active.ID {
				cctx.Context.Menu = cctx.Server
				cctx.Context.Sliver = nil
			}
		}
	}

	return
}

// Interact - Interact with a Sliver implant. This commands changes the console
// context, with different commands and completions.
type Interact struct {
	Positional struct {
		ImplantID string `description:"Session ID, Name or Name/ID"` // Name or ID, command will say.
	} `positional-args:"yes" required:"yes"`
}

// Execute - Interact with a Sliver implant.
func (i *Interact) Execute(args []string) (err error) {

	session := GetSession(i.Positional.ImplantID)
	if session != nil {
		cctx.Context.Sliver = &cctx.Session{Session: session} // Will be noticed by all components in need.
		cctx.Context.Menu = cctx.Sliver                       // Except this one.
		fmt.Printf(util.Info+"Active session %s (%d)\n", session.Name, session.ID)
	} else {
		fmt.Printf(util.Error+"Invalid session name or session number '%s'\n", i.Positional.ImplantID)
	}

	// For the moment, we ask the current working directory to implant...
	pwd, err := transport.RPC.Pwd(context.Background(), &sliverpb.PwdReq{
		Request: cctx.Context.Sliver.Request(10),
	})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
	} else {
		cctx.Context.Sliver.WorkingDir = pwd.Path
	}

	return
}

func printSessions(sessions map[uint32]*clientpb.Session) {

	// Sort keys
	var keys []int
	for k := range sessions {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	table := util.NewTable(tui.Bold(tui.Yellow("Sessions")))
	headers := []string{"ID", "Name", "OS/Arch", "Remote Address", "User", "Hostname", "Last Check-in", "Status"}
	headLen := []int{0, 0, 0, 15, 0, 0, 0, 0}
	table.SetColumns(headers, headLen)

	for _, k := range keys {
		s := sessions[uint32(k)]
		osArch := fmt.Sprintf("%s/%s", s.OS, s.Arch)

		var status string
		if s.IsDead {
			status = "Dead"
		} else {
			status = "Alive"
		}
		row := []string{strconv.Itoa(int(s.ID)), s.Name, osArch, s.RemoteAddress, s.Username,
			s.Hostname, s.LastCheckin, status}

		table.AppendRow(row)
	}
	table.Output()
}

// Sliver-only context available commands
// ----------------------------------------------------------------------------------------------------------

// Background - Exit from implant context.
type Background struct{}

// Execute - Exit from implant context.
func (b *Background) Execute(args []string) (err error) {

	cctx.Context.Menu = cctx.Server // Coming back to server main menu
	cctx.Context.Sliver = nil
	fmt.Printf(util.Info + "Background ...\n")

	return
}

// GetSession - Get session by session ID or name
func GetSession(arg string) *clientpb.Session {
	sessions, err := transport.RPC.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return nil
	}
	for _, session := range sessions.GetSessions() {
		if fmt.Sprintf("%d", session.ID) == arg {
			// if session.Name == arg || fmt.Sprintf("%d", session.ID) == arg {
			return session
		}
	}
	return nil
}

// Kill - Kill the active session.
// Therefore this command is different from the one in Sessions struct.
type Kill struct {
	Force   bool `long:"force" description:"Force kill, does not clean up"`
	Timeout int  `long:"timeout" description:"Command timeout in seconds" default:"60"`
}

// Execute - Kill the active session.
func (k *Kill) Execute(args []string) (err error) {

	session := cctx.Context.Sliver.Session
	err = killSession(session, transport.RPC)
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return
	}
	// Change context
	cctx.Context.Menu = cctx.Server // Coming back to server main menu
	cctx.Context.Sliver = nil

	return
}

func killSession(session *clientpb.Session, rpc rpcpb.SliverRPCClient) error {
	if session == nil {
		return errors.New("Session does not exist")
	}
	_, err := rpc.KillSession(context.Background(), &sliverpb.KillSessionReq{
		Request: &commonpb.Request{
			SessionID: session.ID,
		},
		Force: true,
	})
	if err != nil {
		return err
	}

	fmt.Printf(util.Info+"Killed %s (%d)\n", session.Name, session.ID)
	// fmt.Printf(util.Info + "Waiting for confirmation...\n")
	ctrl := make(chan bool)
	go spin.Until(util.Info+"Waiting for confirmation...", ctrl)
	time.Sleep(time.Second * 1)
	ctrl <- true
	<-ctrl

	return nil
}
