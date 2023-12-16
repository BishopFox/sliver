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
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm/logger"
)

// gorm middleware for database queries/results logging.
type gormWriter struct {
	log *logrus.Entry
}

func (w gormWriter) Printf(format string, args ...interface{}) {
	w.log.Printf(format, args...)
}

// NewDatabase returns a logger suitable as logrus database logging middleware.
func NewDatabase(log *logrus.Entry, level string) logger.Interface {
	logConfig := logger.Config{
		SlowThreshold: time.Second,
		Colorful:      true,
		LogLevel:      logger.Info,
	}
	switch strings.ToLower(level) {
	case "silent":
		logConfig.LogLevel = logger.Silent
	case "err":
		fallthrough
	case "error":
		logConfig.LogLevel = logger.Error
	case "warning":
		fallthrough
	case "warn":
		logConfig.LogLevel = logger.Warn
	case "info":
		fallthrough
	default:
		logConfig.LogLevel = logger.Info
	}

	return logger.New(gormWriter{log: log}, logConfig)
}
