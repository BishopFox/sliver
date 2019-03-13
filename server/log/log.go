package log

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"

	"github.com/sirupsen/logrus"
)

var (
	// RootLoggerName - Root logger name, contains all log data
	RootLoggerName = "root"
	// RootLogger - Root Logger
	RootLogger = rootLogger()
)

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
func rootLogger() *logrus.Logger {
	rootLogger := logrus.New()
	rootLogger.Formatter = &logrus.JSONFormatter{}
	jsonFilePath := path.Join(GetLogDir(), "sliver.json")
	jsonFile, err := os.OpenFile(jsonFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file %v", err))
	}
	rootLogger.Out = jsonFile
	rootLogger.SetLevel(logrus.DebugLevel)
	rootLogger.SetReportCaller(true)
	rootLogger.AddHook(NewTxtHook("root"))
	return rootLogger
}

// RootLogger - Returns the root logger
func txtLogger() *logrus.Logger {
	txtLogger := logrus.New()
	txtLogger.Formatter = &logrus.TextFormatter{ForceColors: true}
	txtFilePath := path.Join(GetLogDir(), "sliver.log")
	txtFile, err := os.OpenFile(txtFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file %v", err))
	}
	txtLogger.Out = txtFile
	txtLogger.SetLevel(logrus.DebugLevel)
	return txtLogger
}

// TxtHook - Hook in a textual version of the logs
type TxtHook struct {
	Name   string
	logger *logrus.Logger
}

// NewTxtHook - returns a new txt hook
func NewTxtHook(name string) *TxtHook {
	hook := &TxtHook{
		Name:   name,
		logger: txtLogger(),
	}
	return hook
}

// Fire - Implements the fire method of the Logrus hook
func (hook *TxtHook) Fire(entry *logrus.Entry) error {
	if hook.logger == nil {
		return errors.New("No txt logger")
	}
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
	case logrus.DebugLevel, logrus.TraceLevel:
		hook.logger.Debug(entry.Message)
	}
	return nil
}

// Levels - Hook all levels
func (hook *TxtHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
