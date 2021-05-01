package core

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

	"github.com/maxlandon/gonsole"
	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/transport"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var (
	// ActiveSession - The Sliver session we are currently interacting with.
	ActiveSession *Session

	// Console At startup the console has passed itself to this package, so that
	// we can question the application parser for timeout/request options.
	Console *gonsole.Console

	//SessionHistoryFunc - Will pass the session history to the console package.
	// This is needed as we cannot import the console package, which contains histories.
	SessionHistoryFunc func(commands []string)
)

// Session - An implant session we are interacting with.
// This is a wrapper for some utility methods.
type Session struct {
	*clientpb.Session
	WorkingDir string // The implant working directory, stored to limit calls.
}

// SetActiveSession - Sets a session as active and
// pulls out all informations needed by the console.
func SetActiveSession(sess *clientpb.Session) {
	ActiveSession = &Session{Session: sess}

	// For the moment, we ask the current working directory to implant...
	pwd, err := transport.RPC.Pwd(context.Background(), &sliverpb.PwdReq{
		Request: ActiveSession.RequestTimeout(10),
	})
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	} else {
		ActiveSession.WorkingDir = pwd.Path
	}

	// Switch the console context
	Console.SwitchMenu(constants.SliverMenu)

	// Hide Windows commands if this implant is not Windows-based
	if ActiveSession.OS != "windows" {
		Console.HideCommands("windows")
	}

	// Then we get the history
	sessionHistory := GetActiveSessionHistory()
	SessionHistoryFunc(sessionHistory)
}

// RequestTimeout - Prepare a RPC request for the current Session.
func (s *Session) RequestTimeout(timeOut int) *commonpb.Request {
	if s.Session == nil {
		return nil
	}
	timeout := int(time.Second) * timeOut
	return &commonpb.Request{
		SessionID: s.ID,
		Timeout:   int64(timeout),
	}
}

// ActiveSessionRequest - Make a request for the active session
func ActiveSessionRequest() (req *commonpb.Request) {
	if ActiveSession != nil {
		return SessionRequest(ActiveSession.Session)
	}
	return SessionRequest(nil)
}

// SessionRequest - Forge a Request Protobuf metadata to be sent in a RPC request.
func SessionRequest(sess *clientpb.Session) (req *commonpb.Request) {
	req = &commonpb.Request{}

	if sess != nil {
		req.SessionID = sess.ID
	}

	// The current parser holds some data we want
	var parser = Console.CommandParser()
	if parser == nil {
		return req
	}

	// Get timeout
	if opt := parser.FindOptionByLongName("timeout"); opt != nil {
		if val, ok := opt.Value().(int64); ok {
			req.Timeout = val
		}
	}

	return
}

// GetSession - Get session by session ID or name
func GetSession(arg string) *clientpb.Session {
	sessions, err := transport.RPC.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}
	for _, session := range sessions.GetSessions() {
		if fmt.Sprintf("%d", session.ID) == arg {
			return session
		}
	}
	return nil
}

// GetActiveSessionHistory - Get the command history that matches all occurences for the user_UUID session.
func GetActiveSessionHistory() []string {
	res, err := transport.RPC.GetHistory(context.Background(),
		&clientpb.HistoryRequest{
			AllConsoles: true,
			Session:     ActiveSession.Session,
		})
	if err != nil {
		return []string{}
	}
	return res.Sliver
}

// IsUserAnAdult - This should be called for any dangerous (OPSEC-wise) functions
// Part of the core package because... well why not ?
// Please insert good reason here:
func IsUserAnAdult() bool {
	confirm := false
	prompt := &survey.Confirm{Message: "This action is bad OPSEC, are you an adult?"}
	survey.AskOne(prompt, &confirm, nil)
	return confirm
}
