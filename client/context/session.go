package context

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

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// Session - An implant session we are interacting with.
// This is a wrapper for some utility methods.
type Session struct {
	*clientpb.Session
	WorkingDir string // The implant working directory, stored to limit calls.
}

// Request - Prepare a RPC request for the current Session.
func (s *Session) Request(timeOut int) *commonpb.Request {
	if s.Session == nil {
		return nil
	}
	timeout := int(time.Second) * timeOut
	return &commonpb.Request{
		SessionID: s.ID,
		Timeout:   int64(timeout),
	}
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
			return session
		}
	}
	return nil
}
