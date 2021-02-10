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
	"fmt"
	insecureRand "math/rand"

	"github.com/evilsocket/islazy/tui"

	consts "github.com/bishopfox/sliver/client/constants"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// Info - Show Session information
type Info struct {
	Positional struct {
		SessionID string `description:"session ID"`
	} `positional-args:"yes"`
}

// Execute - Show Session information
func (i *Info) Execute(args []string) (err error) {

	var session *clientpb.Session
	if cctx.Context.Sliver != nil {
		session = cctx.Context.Sliver.Session
	} else if i.Positional.SessionID != "" {
		session = GetSession(i.Positional.SessionID)
	}

	if session != nil {
		fmt.Printf(bold+"            ID: %s%d\n", normal, session.ID)
		fmt.Printf(bold+"          Name: %s%s\n", normal, session.Name)
		fmt.Printf(bold+"      Hostname: %s%s\n", normal, session.Hostname)
		fmt.Printf(bold+"          UUID: %s%s\n", normal, session.UUID)
		fmt.Printf(bold+"      Username: %s%s\n", normal, session.Username)
		fmt.Printf(bold+"           UID: %s%s\n", normal, session.UID)
		fmt.Printf(bold+"           GID: %s%s\n", normal, session.GID)
		fmt.Printf(bold+"           PID: %s%d\n", normal, session.PID)
		fmt.Printf(bold+"            OS: %s%s\n", normal, session.OS)
		fmt.Printf(bold+"       Version: %s%s\n", normal, session.Version)
		fmt.Printf(bold+"          Arch: %s%s\n", normal, session.Arch)
		fmt.Printf(bold+"Remote Address: %s%s\n", normal, session.RemoteAddress)
		fmt.Printf(bold+"     Proxy URL: %s%s\n", normal, session.ProxyURL)
	} else {
		fmt.Printf(util.Error+"No target session, see `help %s`\n", consts.InfoStr)
	}
	return
}

// Ping - Ping a session
type Ping struct{}

// Execute - Command
func (p *Ping) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}
	nonce := insecureRand.Intn(999999)
	fmt.Printf(util.Info+"Ping %d\n", nonce)
	pong, err := transport.RPC.Ping(context.Background(), &sliverpb.Ping{
		Nonce:   int32(nonce),
		Request: ContextRequest(cctx.Context.Sliver.Session),
	})
	if err != nil {
		fmt.Printf(util.Warn+"%s\n", err)
	} else {
		fmt.Printf(util.Info+"Pong %d\n", pong.Nonce)
	}
	return nil
}

// PID - Get session Process ID
type PID struct{}

// Execute - Command
func (p *PID) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}
	fmt.Printf(util.Info+"Process ID: %d\n", session.PID)
	return
}

// UID - Get session User ID
type UID struct{}

// Execute - Command
func (u *UID) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}
	fmt.Printf(util.Info+"User ID: %s\n", tui.Bold(session.UID))
	return
}

// GID - Get session User Group ID
type GID struct{}

// Execute - Command
func (p *GID) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}
	fmt.Printf(util.Info+"User group ID: %s\n", tui.Bold(session.GID))
	return
}

// Whoami - Whoami command
type Whoami struct{}

// Execute - Command
func (w *Whoami) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}
	fmt.Printf(util.Info+"User: %s\n", tui.Bold(session.Username))
	return
}
