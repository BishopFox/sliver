package log

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

	"github.com/maxlandon/readline"
	"github.com/sirupsen/logrus"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

const (
	eventBufferDefault = 200
)

var (
	// References to console components, used by all loggers.
	shell        *readline.Instance
	promptRender func() string

	isSynchronized bool // If true, the log printing beahvior will vary
)

// Init - The client starts monitoring all event logs coming from itself, or the server
func Init(sh *readline.Instance, render func() string, rpc rpcpb.SliverRPCClient) error {
	if sh == nil || render == nil {
		return errors.New("missing shell instance or prompt rendering function")
	}
	if transport.RPC == nil {
		return errors.New("No connected RPC client")
	}
	// Keep references for loggers
	shell = sh
	promptRender = render

	// Here all client text loggers will work out of the box.
	// Now we start monitoring server events in a separate loop
	// go handleServerLogs(rpc)

	return nil
}

// SynchronizeLogs - A command has to be executed, and we don't want any refresh of the prompt
// while waiting for it finish execution. We pass the component that might produce such logs.
func SynchronizeLogs(component string) {
	isSynchronized = true
}

// ResetLogSynchroniser - Log are again printed with a prompt refresh each time.
func ResetLogSynchroniser() {
	isSynchronized = false
}

// IsSynchronized - Before the console prints some stuffs not handled by the logger,
// it checks if it needs to refresh the prompt on its own.
func IsSynchronized() bool {
	return isSynchronized
}

var logrusPrintLevels = map[logrus.Level]string{
	logrus.TraceLevel: fmt.Sprintf("%s[t] %s", readline.DIM, readline.RESET),
	logrus.DebugLevel: fmt.Sprintf("%s%s[_] %s", readline.DIM, readline.BLUE, readline.RESET),
	logrus.InfoLevel:  util.Info,
	logrus.WarnLevel:  util.Warn,
	logrus.ErrorLevel: util.Error,
}

func isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// getSessionsByName - Return all sessions for an Implant by name
func getSessionsByName(name string, rpc rpcpb.SliverRPCClient) []*clientpb.Session {
	sessions, err := rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
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
