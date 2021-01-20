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

	"github.com/evilsocket/islazy/tui"
	"github.com/sirupsen/logrus"
)

var (
	// ClientLogger - Logger used by console binary components only
	ClientLogger = NewClientLogger("")
)

// NewClientLogger - A text logger being passed to any
// component the client binary (only) for logging events/info.
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
	name string // The name of the component (comm, module, etc.)
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
		line += fmt.Sprintf("%s%-10v %s-%s ", tui.RED, component, tui.DIM, tui.RESET)
	} else {
		line += fmt.Sprintf("%s%-10v %s-%s ", tui.DIM, component, tui.DIM, tui.RESET)
	}

	// Add the message and print
	line += log.Message
	fmt.Println(line)

	// By default, refresh the prompt
	// shell.RefreshMultiline(promptRender(), 0, false)

	return nil
}

// Levels - Function needed to implement the logrus.TxtLogger interface
func (l *clientHook) Levels() (levels []logrus.Level) {
	return logrus.AllLevels
}
