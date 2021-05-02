package sliver

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
	"regexp"
	"time"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// Background - Exit from implant context.
type Background struct{}

// Execute - Exit from implant context.
func (b *Background) Execute(args []string) (err error) {

	// Takes care of menu switching, unregistering session refreshing history, etc...
	core.UnsetActiveSession()
	fmt.Printf(Info + "Background ...\n")

	return
}

// Set - Set an environment value for the current session.
type Set struct {
	Options struct {
		Name      string `long:"name" description:"set agent name"`
		Reconnect int32  `long:"reconnect" short:"r" description:"reconnect interval for agent" default:"-1"`
		Poll      int32  `long:"poll" short:"p" description:"poll interval for agent" default:"-1"`
	} `group:"session values"`
}

// Execute - Set an environment value for the current session.
func (s *Set) Execute(args []string) (err error) {

	// Option to change the agent name
	name := s.Options.Name
	if name != "" {
		isAlphanumeric := regexp.MustCompile(`^[[:alnum:]]+$`).MatchString
		if !isAlphanumeric(name) {
			fmt.Printf(Error + "Name must be in alphanumeric only\n")
			return
		}
	}

	session, err := transport.RPC.UpdateSession(context.Background(), &clientpb.UpdateSession{
		SessionID:         core.ActiveSession.ID,
		Name:              name,
		ReconnectInterval: s.Options.Reconnect,
		PollInterval:      s.Options.Poll,
	})
	if err != nil {
		fmt.Printf(Error+"Error: %v", err)
		return
	}
	core.ActiveSession = session // Will be noticed by all components in need.

	// For the moment, we ask the current working directory to implant...
	pwd, err := transport.RPC.Pwd(context.Background(), &sliverpb.PwdReq{
		Request: core.RequestTimeout(10),
	})
	if err != nil {
		fmt.Printf(Error+"%s\n", err)
	} else {
		core.ActiveSession.WorkingDirectory = pwd.Path
	}

	return
}

// Kill - Kill the active session.
// Therefore this command is different from the one in Sessions struct.
type Kill struct {
	Options struct {
		Force bool `long:"force" short:"f" description:"force kill, does not clean up"`
	} `group:"kill options"`
}

// Execute - Kill the active session.
func (k *Kill) Execute(args []string) (err error) {

	session := core.ActiveSession
	err = killSession(session, transport.RPC)
	if err != nil {
		fmt.Printf(Error+"%s\n", err)
		return
	}

	core.UnsetActiveSession()
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

	ctrl := make(chan bool)
	go spin.Until(Info+"Waiting for confirmation...", ctrl)
	time.Sleep(time.Second * 1)
	ctrl <- true
	<-ctrl
	fmt.Printf(Info+"Killed %s (%d)\n", session.Name, session.ID)

	return nil
}
