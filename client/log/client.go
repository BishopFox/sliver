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
	"fmt"
	"io/ioutil"

	"github.com/maxlandon/readline"
	"github.com/sirupsen/logrus"
)

var (
	// ClientLogger - Logger used by console binary components only,
	// like the client-specific part of shell tunnels.
	ClientLogger = NewClientLogger("")
)

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
	if isSynchronized {
		fmt.Print(line)
	}

	// Else, we pass the log to the shell, which will handle wrapping computing, and so on.
	if !isSynchronized {
		shell.RefreshPromptLog(line)
	}

	return nil
}

// Levels - Function needed to implement the logrus.TxtLogger interface
func (l *clientHook) Levels() (levels []logrus.Level) {
	return logrus.AllLevels
}
