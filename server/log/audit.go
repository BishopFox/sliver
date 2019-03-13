package log

import (
	"fmt"
	"os"
	"path"

	"github.com/sirupsen/logrus"
)

var (
	// AuditLogger - Single audit log
	AuditLogger = newAuditLogger()
)

func newAuditLogger() *logrus.Logger {
	auditLogger := logrus.New()
	auditLogger.Formatter = &logrus.JSONFormatter{}
	jsonFilePath := path.Join(GetLogDir(), "audit.json")
	jsonFile, err := os.OpenFile(jsonFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file %v", err))
	}
	auditLogger.Out = jsonFile
	auditLogger.SetLevel(logrus.DebugLevel)

	return auditLogger
}
