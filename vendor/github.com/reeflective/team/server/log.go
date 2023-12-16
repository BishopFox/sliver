package server

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
	"fmt"
	"path/filepath"

	"github.com/reeflective/team/internal/log"
	"github.com/sirupsen/logrus"
)

// NamedLogger returns a new logging "thread" with two fields (optional)
// to indicate the package/general domain, and a more precise flow/stream.
// The events are logged according to the teamclient logging backend setup.
func (ts *Server) NamedLogger(pkg, stream string) *logrus.Entry {
	return ts.log().WithFields(logrus.Fields{
		log.PackageFieldKey: pkg,
		"stream":            stream,
	})
}

// SetLogLevel sets the logging level of teamserver loggers (excluding audit ones).
func (ts *Server) SetLogLevel(level int) {
	if ts.stdioLog == nil {
		return
	}

	if uint32(level) > uint32(logrus.TraceLevel) {
		level = int(logrus.TraceLevel)
	}

	ts.stdioLog.SetLevel(logrus.Level(uint32(level)))

	// Also Change the file-based logging level:
	// - If they app runs a memfs, this wont have any effect.
	// - If the user wants to debug anyway, better two sources than one.
	if ts.fileLog != nil {
		ts.fileLog.SetLevel(logrus.Level(uint32(level)))
	}
}

// AuditLogger returns a special logger writing its event entries to an audit
// log file (default audit.json), distinct from other teamserver log files.
// Listener implementations will want to use this for logging various teamclient
// application requests, with this logger used somewhere in your listener middleware.
func (ts *Server) AuditLogger() (*logrus.Logger, error) {
	if ts.opts.inMemory || ts.opts.noLogs {
		return ts.log(), nil
	}

	// Generate a new audit logger
	auditLog, err := log.NewAudit(ts.fs, ts.LogsDir())
	if err != nil {
		return nil, ts.errorf("%w: %w", ErrLogging, err)
	}

	return auditLog, nil
}

// Initialize loggers in files/stdout according to options.
func (ts *Server) initLogging() (err error) {
	// If user supplied a logger, use it in place of the
	// file-based logger, since the file logger is optional.
	if ts.opts.logger != nil {
		ts.fileLog = ts.opts.logger
		return nil
	}

	logFile := filepath.Join(ts.LogsDir(), log.FileName(ts.Name(), true))

	// If the teamserver should log to a given file.
	if ts.opts.logFile != "" {
		logFile = ts.opts.logFile
	}

	level := logrus.Level(ts.opts.config.Log.Level)

	// Create any additional/configured logger and related/missing hooks.
	ts.fileLog, ts.stdioLog, err = log.Init(ts.fs, logFile, level)
	if err != nil {
		return err
	}

	return nil
}

// log returns a non-nil logger for the server:
// if file logging is disabled, it returns the stdout-only logger,
// otherwise returns the file logger equipped with a stdout hook.
func (ts *Server) log() *logrus.Logger {
	if ts.fileLog == nil {
		return ts.stdioLog
	}

	return ts.fileLog
}

func (ts *Server) errorf(msg string, format ...any) error {
	logged := fmt.Errorf(msg, format...)
	ts.log().Error(logged)

	return logged
}

func (ts *Server) errorWith(log *logrus.Entry, msg string, format ...any) error {
	logged := fmt.Errorf(msg, format...)

	if log != nil {
		log.Error(logged)
	} else {
		ts.log().Error(logged)
	}

	return logged
}
