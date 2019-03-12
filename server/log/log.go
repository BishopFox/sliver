package log

import (
	"fmt"
	golog "log"
	"os"
	"os/user"
	"path"

	"github.com/sirupsen/logrus"
)

var (
	// RootLoggerName - Root logger name, contains all log data
	RootLoggerName = "root"
	// RootLoggers - The actual loggers for the "root" stream
	RootLoggers = rootLoggers()
)

// Stream - Struct that writes logs to multiple files
type Stream struct {
	Name        string
	loggers     []*logrus.Logger
	rootLoggers []*logrus.Logger
}

func (s Stream) allLoggers() []*logrus.Logger {
	return append(s.loggers, s.rootLoggers...)
}

// Debug - Write to debug logs
func (s Stream) Debug(msg ...interface{}) {
	for _, logger := range s.allLoggers() {
		logger.WithFields(logrus.Fields{
			"stream": s.Name,
		}).Debug(msg...)
	}
}

// Debugf - Write to Debug logs
func (s Stream) Debugf(format string, msg ...interface{}) {
	for _, logger := range s.allLoggers() {
		logger.WithFields(logrus.Fields{
			"stream": s.Name,
		}).Debugf(format, msg...)
	}
}

// Info - Write to Info logs
func (s Stream) Info(msg ...interface{}) {
	for _, logger := range s.allLoggers() {
		logger.WithFields(logrus.Fields{
			"stream": s.Name,
		}).Info(msg...)
	}
}

// Infof - Write to Info logs
func (s Stream) Infof(format string, msg ...interface{}) {
	for _, logger := range s.allLoggers() {
		logger.WithFields(logrus.Fields{
			"stream": s.Name,
		}).Infof(format, msg...)
	}
}

// Warn - Write to debug logs
func (s Stream) Warn(msg ...interface{}) {
	for _, logger := range s.allLoggers() {
		logger.WithFields(logrus.Fields{
			"stream": s.Name,
		}).Warn(msg...)
	}
}

// Warnf - Write to Warn logs
func (s Stream) Warnf(format string, msg ...interface{}) {
	for _, logger := range s.allLoggers() {
		logger.WithFields(logrus.Fields{
			"stream": s.Name,
		}).Warnf(format, msg...)
	}
}

// Error - Write to debug logs
func (s Stream) Error(msg ...interface{}) {
	for _, logger := range s.allLoggers() {
		logger.WithFields(logrus.Fields{
			"stream": s.Name,
		}).Error(msg...)
	}
}

// Errorf - Write to Error logs
func (s Stream) Errorf(format string, msg ...interface{}) {
	for _, logger := range s.allLoggers() {
		logger.WithFields(logrus.Fields{
			"stream": s.Name,
		}).Errorf(format, msg...)
	}
}

// Fatal - Write to debug logs
func (s Stream) Fatal(msg ...interface{}) {
	for _, logger := range s.allLoggers() {
		logger.WithFields(logrus.Fields{
			"stream": s.Name,
		}).Fatal(msg...)
	}
}

// Fatalf - Write to Fatal logs
func (s Stream) Fatalf(format string, msg ...interface{}) {
	for _, logger := range s.allLoggers() {
		logger.WithFields(logrus.Fields{
			"stream": s.Name,
		}).Fatalf(format, msg...)
	}
}

// SetLevel - Set the level of logging
func (s Stream) SetLevel(level logrus.Level) {
	for _, logger := range s.loggers {
		logger.SetLevel(level)
	}
}

// GetLogDir - Return the log dir
func GetLogDir() string {
	user, _ := user.Current()
	dir := path.Join(user.HomeDir, ".sliver")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
	logDir := path.Join(dir, "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err = os.MkdirAll(logDir, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
	return logDir
}

// RootLogger - Returns the root logger
func rootLoggers() []*logrus.Logger {

	jsonLogger := logrus.New()
	jsonLogger.Formatter = &logrus.JSONFormatter{}
	jsonFilePath := path.Join(GetLogDir(), "root.json")
	jsonFile, err := os.OpenFile(jsonFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file %v", err))
	}
	jsonLogger.Out = jsonFile
	jsonLogger.SetLevel(logrus.DebugLevel)

	txtLogger := logrus.New()
	txtLogger.Formatter = &logrus.TextFormatter{}
	txtFilePath := path.Join(GetLogDir(), "root.log")
	txtFile, err := os.OpenFile(txtFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file %v", err))
	}
	txtLogger.Out = txtFile
	txtLogger.SetLevel(logrus.DebugLevel)

	return []*logrus.Logger{jsonLogger, txtLogger}

}

// NewLogger - Get a new log stream by name
func NewLogger(name string) *Stream {

	logger := &Stream{Name: name}

	jsonLogger := logrus.New()
	jsonLogger.Formatter = &logrus.JSONFormatter{}
	jsonFilePath := path.Join(GetLogDir(), path.Base(name)+".json")
	jsonFile, err := os.OpenFile(jsonFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		golog.Fatalf("Failed to open log file %v", err)
	}
	jsonLogger.Out = jsonFile
	jsonLogger.SetLevel(logrus.InfoLevel)

	txtLogger := logrus.New()
	txtLogger.Formatter = &logrus.TextFormatter{}
	txtFilePath := path.Join(GetLogDir(), path.Base(name)+".log")
	txtFile, err := os.OpenFile(txtFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file %v", err))
	}
	txtLogger.Out = txtFile
	txtLogger.SetLevel(logrus.InfoLevel)

	logger.loggers = []*logrus.Logger{jsonLogger, txtLogger}
	logger.loggers = append(logger.loggers, RootLoggers...)

	return logger
}
