package log

import (
	golog "log"
	"os"
	"path"

	"github.com/sirupsen/logrus"
)

var (
	// Audit - Single audit log
	Audit = newAuditLogger()
)

func newAuditLogger() *logrus.Logger {
	auditLogger := logrus.New()
	auditLogger.Formatter = &logrus.JSONFormatter{}
	jsonFilePath := path.Join(GetLogDir(), "audit.json")
	jsonFile, err := os.OpenFile(jsonFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		golog.Fatalf("Failed to open log file %v", err)
	}
	auditLogger.Out = jsonFile
	auditLogger.SetLevel(logrus.InfoLevel)

	return auditLogger
}
