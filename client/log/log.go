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
	"sync"

	"github.com/evilsocket/islazy/tui"
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

// ClientLog - The client log printer
var (
	ClientLog *clientLogger
)

// clientLogger - A logger in charge of printing all messages coming from either the
// console components, or from the server.
type clientLogger struct {
	Comm   chan *logrus.Entry   // All logs of this client Comm subsystem.
	Server chan *clientpb.Event // All generic events (sessions, jobs, canaries, etc)
	rpc    rpcpb.SliverRPC_EventsClient
	mutex  sync.Mutex
}

// InitClientLogger - The console starts monitoring all event logs.
func InitClientLogger() error {

	// Listen for events on the RPC stream.
	if transport.RPC == nil {
		return errors.New("No connected RPC client")
	}

	eventStream, err := transport.RPC.Events(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.RPCError+"%s\n", err)
		return err
	}

	ClientLog = &clientLogger{
		Comm:   make(chan *logrus.Entry, eventBufferDefault),
		Server: make(chan *clientpb.Event, eventBufferDefault),
		rpc:    eventStream,
		mutex:  sync.Mutex{},
	}

	// The client logger starts monitoring.
	go ClientLog.ServeLogs()

	return nil
}

// ServeLogs - The client logger starts listening for incoming logs in background.
func (l *clientLogger) ServeLogs() {

	// Server logs
	go func() {
		for !isDone(l.rpc.Context()) {
			_, err := l.rpc.Recv()
			if err != nil {
				fmt.Printf(util.RPCError + tui.Dim(" server ") + tui.Red(err.Error()) + "\n")
				continue
			}
		}
	}()

	// Comm logs
	for log := range l.Comm {
		comp := log.Data["comm"].(string)    // Get component name
		line := logrusPrintLevels[log.Level] // Final status line to be printed

		// Print the component name in red if error
		if log.Level == logrus.ErrorLevel {
			line += fmt.Sprintf("%s%-10v %s-%s ", tui.RED, comp, tui.DIM, tui.RESET)
		} else {
			line += fmt.Sprintf("%s%-10v %s-%s ", tui.DIM, comp, tui.DIM, tui.RESET)
		}

		// Add the message and print
		line += log.Message
		fmt.Println(line)
	}
}

// All logs happening within the client binary use a classic text logger,
// which push the log messages to their appropriate channels.
func (l *clientLogger) Fire(entry *logrus.Entry) (err error) {

	// Switch on log component
	_, ok := entry.Data["comm"]
	if ok {
		l.Comm <- entry
	}
	return
}

// Levels - Function needed to implement the logrus.TxtLogger interface
func (l *clientLogger) Levels() (levels []logrus.Level) {
	return logrus.AllLevels
}

var logrusPrintLevels = map[logrus.Level]string{
	logrus.TraceLevel: fmt.Sprintf("%s[t] %s", tui.DIM, tui.RESET),
	logrus.DebugLevel: fmt.Sprintf("%s%s[_] %s", tui.DIM, tui.BLUE, tui.RESET),
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
