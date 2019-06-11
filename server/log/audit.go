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
