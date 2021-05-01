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
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/maxlandon/gonsole"
	"github.com/maxlandon/readline"
	"github.com/sirupsen/logrus"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

const (
	eventBufferDefault = 200
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

	// ensure that nothing remains when we refresh the prompt
	seqClearScreenBelow = "\x1b[0J"
)

var (
	// ClientLogger - Logger used by console binary components only,
	// like the client-specific part of shell tunnels.
	ClientLogger = NewClientLogger("")

	// References to console components, used by all loggers.
	console *gonsole.Console

	// Mappings between logrus log levels and their associated console print icon.
	logrusPrintLevels = map[logrus.Level]string{
		logrus.TraceLevel: fmt.Sprintf("%s[t] %s", readline.DIM, readline.RESET),
		logrus.DebugLevel: fmt.Sprintf("%s%s[_] %s", readline.DIM, readline.BLUE, readline.RESET),
		logrus.InfoLevel:  Info,
		logrus.WarnLevel:  fmt.Sprintf("%s[!] %s", readline.YELLOW, readline.RESET),
		logrus.ErrorLevel: Warn,
	}
)

// Init - The client starts monitoring all event logs coming from itself, or the server
func Init(c *gonsole.Console, rpc rpcpb.SliverRPCClient) error {
	if transport.RPC == nil {
		return errors.New("No connected RPC client")
	}
	// Keep references for loggers
	console = c

	return nil
}

// NewClientLogger - A text logger being passed to any component
// running on the client binary (only) for logging events/info.
func NewClientLogger(name string) *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	logger.Out = ioutil.Discard

	// Text hook
	clientHook := &clientHook{name: name}
	logger.AddHook(clientHook)
	return logger
}

// Client Components use this hook to get their logs dispatched.
type clientHook struct {
	name string // (comm, module, etc.)
}

// All logs happening within the client binary use a classic text logger,
// which push the log messages to their appropriate channels.
func (l *clientHook) Fire(log *logrus.Entry) (err error) {

	// Get the component name, and dispatch to central log printer.
	component, ok := log.Data[l.name].(string)
	if !ok {
		component = l.name
	}

	// Maybe can switch on different printing behavior depending on name & component.

	// Final status line to be printed
	line := logrusPrintLevels[log.Level]

	// Print the component name in red if error
	if log.Level == logrus.ErrorLevel {
		line += fmt.Sprintf("%s%-10v %s-%s %s \n",
			readline.RED, component, readline.DIM, readline.RESET, log.Message)
	} else {
		line += fmt.Sprintf("%s%-10v %s-%s %s \n",
			readline.DIM, component, readline.DIM, readline.RESET, log.Message)
	}

	// If we are in the middle of a command, we just print the log without refreshing prompt
	// Else, we pass the log to the shell, which will handle wrapping computing, and so on.
	// The gonsole.Console takes care of print synchronicity.
	console.RefreshPromptLog(line)

	return nil
}

// Levels - Function needed to implement the logrus.TxtLogger interface
func (l *clientHook) Levels() (levels []logrus.Level) {
	return logrus.AllLevels
}
