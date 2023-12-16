package log

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

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
	"io"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// txtHook - Hook in a textual version of the logs.
type txtHook struct {
	Name   string
	app    string
	logger *logrus.Logger
}

// newTxtHook - returns a new txt hook.
func newTxtHook(fs io.Writer, level logrus.Level, log *logrus.Logger) *txtHook {
	hook := &txtHook{}

	logger, err := NewText(fs)
	if err != nil {
		log.Error(err)
	}

	hook.logger = logger
	hook.logger.SetLevel(level)

	return hook
}

// Fire - Implements the fire method of the Logrus hook.
func (hook *txtHook) Fire(entry *logrus.Entry) error {
	if hook.logger == nil {
		return errors.New("no txt logger")
	}

	// Determine the caller (filename/line number)
	srcFile := "<no caller>"
	if entry.HasCaller() {
		wiregostIndex := strings.Index(entry.Caller.File, hook.app)
		srcFile = entry.Caller.File
		if wiregostIndex != -1 {
			srcFile = srcFile[wiregostIndex:]
		}
	}

	// Tream the useless prefix path, containing where it was compiled on the host...
	paths := strings.Split(srcFile, "/mod/")
	if len(paths) > 1 && paths[1] != "" {
		srcFile = filepath.Join(paths[1:]...)
	}

	switch entry.Level {
	case logrus.PanicLevel:
		hook.logger.Panicf(" [%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.FatalLevel:
		hook.logger.Fatalf(" [%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.ErrorLevel:
		hook.logger.Errorf(" [%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.WarnLevel:
		hook.logger.Warnf(" [%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.InfoLevel:
		hook.logger.Infof(" [%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.DebugLevel:
		hook.logger.Debugf(" [%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.TraceLevel:
		hook.logger.Tracef(" [%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	}

	return nil
}

// Levels - Hook all levels.
func (hook *txtHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
