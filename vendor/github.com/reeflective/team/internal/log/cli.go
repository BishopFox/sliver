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
	"fmt"
	"os"
	"strings"

	"github.com/rsteube/carapace/pkg/style"
	"github.com/sirupsen/logrus"
)

// Text effects.
const (
	sgrStart = "\x1b["
	fg       = "38;05;"
	bg       = "48;05;"
	sgrEnd   = "m"
)

const (
	fieldTimestamp = "timestamp"
	fieldPackage   = "logger"
	fieldMessage   = "message"

	minimumPackagePad = 11
)

// PackageFieldKey is used to identify the name of the package
// specified by teamclients and teamservers named loggers.
const PackageFieldKey = "teamserver_pkg"

// stdioHook combines a stdout hook (info/debug/trace),
// and a stderr hook (warn/error/fatal/panic).
type stdioHook struct {
	logger *logrus.Logger
}

func newStdioHook() *stdioHook {
	hook := &stdioHook{
		logger: NewStdio(logrus.WarnLevel),
	}

	return hook
}

// The stdout hooks only outputs info, debug and trace.
func (hook *stdioHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire - Implements the fire method of the Logrus hook.
func (hook *stdioHook) Fire(entry *logrus.Entry) error {
	switch entry.Level {
	case logrus.PanicLevel:
		hook.logger.Panic(entry.Message)
	case logrus.FatalLevel:
		hook.logger.Fatal(entry.Message)
	case logrus.ErrorLevel:
		hook.logger.Error(entry.Message)
	case logrus.WarnLevel:
		hook.logger.Warn(entry.Message)
	case logrus.InfoLevel:
		hook.logger.Info(entry.Message)
	case logrus.DebugLevel:
		hook.logger.Debug(entry.Message)
	case logrus.TraceLevel:
		hook.logger.Trace(entry.Message)
	}

	return nil
}

func newLoggerStdout() *stdoutHook {
	stdLogger := logrus.New()
	stdLogger.SetReportCaller(true)
	stdLogger.Out = os.Stdout

	stdLogger.Formatter = &stdoutHook{
		DisableColors: false,
		ShowTimestamp: false,
		Colors:        defaultFieldsFormat(),
	}

	hook := &stdoutHook{
		logger: stdLogger,
	}

	return hook
}

// stderrHook only logs info events and less.
type stdoutHook struct {
	DisableColors   bool
	ShowTimestamp   bool
	TimestampFormat string
	Colors          map[string]string
	logger          *logrus.Logger
}

// The stdout hooks only outputs info, debug and trace.
func (hook *stdoutHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.InfoLevel,
		logrus.DebugLevel,
		logrus.TraceLevel,
	}
}

// Fire - Implements the fire method of the Logrus hook.
func (hook *stdoutHook) Fire(entry *logrus.Entry) error {
	switch entry.Level {
	case logrus.PanicLevel:
		hook.logger.Panic(entry.Message)
	case logrus.FatalLevel:
		hook.logger.Fatal(entry.Message)
	case logrus.ErrorLevel:
		hook.logger.Error(entry.Message)
	case logrus.WarnLevel:
		hook.logger.Warn(entry.Message)
	case logrus.InfoLevel:
		hook.logger.Info(entry.Message)
	case logrus.DebugLevel:
		hook.logger.Debug(entry.Message)
	case logrus.TraceLevel:
		hook.logger.Trace(entry.Message)
	}

	return nil
}

// Format is a custom formatter for all stdout/text logs, with better format and coloring.
func (hook *stdoutHook) Format(entry *logrus.Entry) ([]byte, error) {
	// Basic information.
	sign, signColor := hook.getLevelFieldColor(entry.Level)
	levelLog := fmt.Sprintf("%s%s%s", color(signColor), sign, color(style.Default))

	timestamp := entry.Time.Format(hook.TimestampFormat)
	timestampLog := fmt.Sprintf("%s%s%s", color(hook.Colors[fieldTimestamp]), timestamp, color(style.Default))

	var pkgLogF string

	pkg := entry.Data[PackageFieldKey]
	if pkg != nil {
		pkgLog := fmt.Sprintf(" %v ", pkg)
		pkgLog = fmt.Sprintf("%-*s", minimumPackagePad, pkgLog)
		pkgLogF = strings.ReplaceAll(pkgLog, fmt.Sprintf("%s", pkg), fmt.Sprintf("%s%s%s", color(hook.Colors[fieldPackage]), pkg, color(style.Default)))
	}

	// Always try to unwrap the error at least once, and colorize it.
	message := entry.Message
	if err := errors.Unwrap(errors.New(message)); err != nil {
		if err.Error() != message {
			message = color(style.Red) + message + color(style.Of(style.Default, style.White)) + err.Error() + color(style.Default)
		}
	}

	messageLog := fmt.Sprintf("%s%s%s", color(hook.Colors[fieldMessage]), message, color(style.Default))

	// Assemble the log message
	var logMessage string

	if hook.ShowTimestamp {
		logMessage += timestampLog + " "
	}

	logMessage += pkgLogF + " "
	logMessage += levelLog + " "
	logMessage += messageLog + "\n"

	return []byte(logMessage), nil
}

func (hook *stdoutHook) getLevelFieldColor(level logrus.Level) (string, string) {
	// Builtin configurations.
	signs := defaultLevelFields()
	colors := defaultLevelFieldsColored()

	if sign, ok := signs[level]; ok {
		if color, ok := colors[sign]; ok {
			return sign, color
		}

		return sign, style.Default
	}

	return signs[logrus.InfoLevel], style.Default
}

// stderrHook only logs warning events and worst.
type stderrHook struct {
	DisableColors   bool
	ShowTimestamp   bool
	TimestampFormat string
	Colors          map[string]string
	logger          *logrus.Logger
}

func newLoggerStderr() *stderrHook {
	stdLogger := logrus.New()
	stdLogger.SetLevel(logrus.WarnLevel)
	stdLogger.SetReportCaller(true)
	stdLogger.Out = os.Stderr

	stdLogger.Formatter = &stdoutHook{
		DisableColors: false,
		ShowTimestamp: false,
		Colors:        defaultFieldsFormat(),
	}

	hook := &stderrHook{
		logger: stdLogger,
	}

	return hook
}

// Fire - Implements the fire method of the Logrus hook.
func (hook *stderrHook) Fire(entry *logrus.Entry) error {
	switch entry.Level {
	case logrus.PanicLevel:
		hook.logger.Panic(entry.Message)
	case logrus.FatalLevel:
		hook.logger.Fatal(entry.Message)
	case logrus.ErrorLevel:
		hook.logger.Error(entry.Message)
	case logrus.WarnLevel:
		hook.logger.Warn(entry.Message)
	case logrus.InfoLevel:
		hook.logger.Info(entry.Message)
	case logrus.DebugLevel:
		hook.logger.Debug(entry.Message)
	case logrus.TraceLevel:
		hook.logger.Trace(entry.Message)
	}

	return nil
}

// The stderr hooks only outputs errors and worst.
func (hook *stderrHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	}
}

func defaultFieldsFormat() map[string]string {
	return map[string]string{
		fieldTimestamp: style.BrightBlack,
		fieldPackage:   style.Dim,
		fieldMessage:   style.BrightWhite,
	}
}

func defaultLevelFields() map[logrus.Level]string {
	return map[logrus.Level]string{
		logrus.TraceLevel: "▪",
		logrus.DebugLevel: "▫",
		logrus.InfoLevel:  "○",
		logrus.WarnLevel:  "▲",
		logrus.ErrorLevel: "✖",
		logrus.FatalLevel: "☠",
		logrus.PanicLevel: "!!",
	}
}

func defaultLevelFieldsColored() map[string]string {
	return map[string]string{
		"▪":  style.BrightBlack,
		"▫":  style.Dim,
		"○":  style.BrightBlue,
		"▲":  style.Yellow,
		"✖":  style.BrightRed,
		"☠":  style.BgBrightCyan,
		"!!": style.BgBrightMagenta,
	}
}

func color(color string) string {
	return sgrStart + style.SGR(color) + sgrEnd
}
