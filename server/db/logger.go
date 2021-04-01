package db

import (
	"strings"
	"time"

	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/log"
	"gorm.io/gorm/logger"
)

var (
	gormLog = log.NamedLogger("db", "gorm")
)

type gormWriter struct {
}

func (w gormWriter) Printf(format string, args ...interface{}) {
	gormLog.Printf(format, args...)
}

func getGormLogger(dbConfig *configs.DatabaseConfig) logger.Interface {
	logConfig := logger.Config{
		SlowThreshold: time.Second,
		Colorful:      true,
		LogLevel:      logger.Info,
	}
	switch strings.ToLower(dbConfig.LogLevel) {
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

	return logger.New(gormWriter{}, logConfig)
}
