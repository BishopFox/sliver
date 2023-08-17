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
	"fmt"
	"io"
	"path/filepath"

	"github.com/reeflective/team/internal/assets"
	"github.com/sirupsen/logrus"
)

const (
	// ClientLogFileExt is used as extension by all main teamclients log files by default.
	ClientLogFileExt = "teamclient.log"
	// ServerLogFileExt is used as extension by all teamservers core log files by default.
	ServerLogFileExt = "teamserver.log"
)

// Init is the main constructor that is (and should be) used for teamserver and teamclient logging.
// It hooks a normal logger with a sublogger writing to a file in text version, and another logger
// writing to stdout/stderr with enhanced formatting/coloring support.
func Init(fs *assets.FS, file string, level logrus.Level) (*logrus.Logger, *logrus.Logger, error) {
	logFile, err := fs.OpenFile(file, assets.FileWriteOpenMode, assets.FileWritePerm)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to open log file %w", err)
	}

	// Text-format logger, writing to file.
	textLogger := logrus.New()
	textLogger.Formatter = &stdoutHook{
		DisableColors: false,
		ShowTimestamp: false,
		Colors:        defaultFieldsFormat(),
	}
	textLogger.Out = io.Discard

	textLogger.SetLevel(logrus.InfoLevel)
	textLogger.SetReportCaller(true)

	// File output
	textLogger.AddHook(newTxtHook(logFile, level, textLogger))

	// Stdout/err output, with special formatting.
	stdioHook := newStdioHook()
	textLogger.AddHook(stdioHook)

	return textLogger, stdioHook.logger, nil
}

// NewStdio returns a logger configured to output its events to the system stdio:
// - Info/Debug/Trace logs are written to os.Stdout.
// - Warn/Error/Fatal/Panic are written to os.Stderr.
func NewStdio(level logrus.Level) *logrus.Logger {
	stdLogger := logrus.New()
	stdLogger.Formatter = &stdoutHook{
		DisableColors: false,
		ShowTimestamp: false,
		Colors:        defaultFieldsFormat(),
	}

	stdLogger.SetLevel(level)
	stdLogger.SetReportCaller(true)
	stdLogger.Out = io.Discard

	// Info/debug/trace is given to a stdout logger.
	stdoutHook := newLoggerStdout()
	stdLogger.AddHook(stdoutHook)

	// Warn/error/panics/fatals are given to stderr.
	stderrHook := newLoggerStderr()
	stdLogger.AddHook(stderrHook)

	return stdLogger
}

// NewJSON returns a logger writing to the central log file of the teamserver, JSON-encoded.
func NewJSON(fs *assets.FS, file string, level logrus.Level) (*logrus.Logger, error) {
	rootLogger := logrus.New()
	rootLogger.Formatter = &logrus.JSONFormatter{}
	jsonFilePath := fmt.Sprintf("%s.json", file)

	logFile, err := fs.OpenFile(jsonFilePath, assets.FileWriteOpenMode, assets.FileWritePerm)
	if err != nil {
		return nil, fmt.Errorf("Failed to open log file %w", err)
	}

	rootLogger.Out = logFile
	rootLogger.SetLevel(logrus.InfoLevel)
	rootLogger.SetReportCaller(true)
	rootLogger.AddHook(newTxtHook(logFile, level, rootLogger))

	return rootLogger, nil
}

// NewAudit returns a logger writing to an audit file in JSON format.
func NewAudit(fs *assets.FS, logDir string) (*logrus.Logger, error) {
	auditLogger := logrus.New()
	auditLogger.Formatter = &logrus.JSONFormatter{}
	jsonFilePath := filepath.Join(logDir, "audit.json")

	logFile, err := fs.OpenFile(jsonFilePath, assets.FileWriteOpenMode, assets.FileWritePerm)
	if err != nil {
		return nil, fmt.Errorf("Failed to open log file %w", err)
	}

	auditLogger.Out = logFile
	auditLogger.SetLevel(logrus.DebugLevel)

	return auditLogger, nil
}

// NewText returns a new logger writing to a given file.
// The formatting is enhanced for informative debugging and call
// stack reporting, but without any special coloring/formatting.
func NewText(file io.Writer) (*logrus.Logger, error) {
	txtLogger := logrus.New()
	txtLogger.Formatter = &logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	}

	txtLogger.Out = file
	txtLogger.SetLevel(logrus.InfoLevel)

	return txtLogger, nil
}

// LevelFrom - returns level from int.
func LevelFrom(level int) logrus.Level {
	switch level {
	case int(logrus.PanicLevel):
		return logrus.PanicLevel
	case int(logrus.FatalLevel):
		return logrus.FatalLevel
	case int(logrus.ErrorLevel):
		return logrus.ErrorLevel
	case int(logrus.WarnLevel):
		return logrus.WarnLevel
	case int(logrus.InfoLevel):
		return logrus.InfoLevel
	case int(logrus.DebugLevel):
		return logrus.DebugLevel
	case int(logrus.TraceLevel):
		return logrus.TraceLevel
	}

	return logrus.DebugLevel
}

// FileName take a filename without extension and adds
// the corresponding teamserver/teamclient logfile extension.
func FileName(name string, server bool) string {
	if server {
		return fmt.Sprintf("%s.%s", name, ServerLogFileExt)
	}

	return fmt.Sprintf("%s.%s", name, ClientLogFileExt)
}
