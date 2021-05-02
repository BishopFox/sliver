package server

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

	"github.com/maxlandon/readline"
	"github.com/sirupsen/logrus"

	"github.com/bishopfox/sliver/client/log"
)

// Log - Log management commands. Sets log level by default.
type Log struct {
	Positional struct {
		Level      string   `description:"log level to filter by" required:"1-1"`
		Components []string `description:"components on which to apply log filter" required:"1"`
	} `positional-args:"yes" required:"true"`
}

// Execute - Set the log level of one or more components
func (l *Log) Execute(args []string) (err error) {
	// Check level
	level, valid := logrusLevels[l.Positional.Level]
	if !valid {
		fmt.Printf(Error + "Invalid log level (trace, debug, info, warn, error)\n")
		return
	}

	for _, comp := range l.Positional.Components {
		for name, logger := range log.Loggers {
			if name == comp {
				fmt.Printf(Info+"Log level (%s): %s\n", readline.Blue(name), readline.Yellow(level.String()))
				logger.SetLevel(level)
			}
		}
	}

	return
}

var logrusLevels = map[string]logrus.Level{
	"trace":   logrus.TraceLevel,
	"debug":   logrus.DebugLevel,
	"info":    logrus.InfoLevel,
	"warning": logrus.WarnLevel,
	"error":   logrus.ErrorLevel,
}
