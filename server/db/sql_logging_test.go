//go:build go_sqlite || cgo_sqlite

package db

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/server/configs"
	"github.com/sirupsen/logrus"
)

func TestSQLiteClientLogsInitializationWithoutDSN(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	const sensitiveValue = "synthetic-sqlite-password-sentinel"
	config := &configs.DatabaseConfig{
		Dialect:  configs.Sqlite,
		LogLevel: "silent",
		Params:   map[string]string{"password": sensitiveValue},
	}
	dsn, err := config.DSN()
	if err != nil {
		t.Fatalf("create SQLite DSN: %v", err)
	}
	if !strings.Contains(dsn, sensitiveValue) {
		t.Fatal("test DSN omitted the sensitive parameter")
	}

	var output bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&output)
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableQuote: true})
	originalLog := clientLog
	clientLog = logrus.NewEntry(logger)
	t.Cleanup(func() { clientLog = originalLog })

	client := sqliteClient(config)
	sqlDB, err := client.DB()
	if err != nil {
		t.Fatalf("get SQLite connection: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close SQLite connection: %v", err)
		}
	})
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("connect to temporary SQLite database: %v", err)
	}

	logged := output.String()
	if !strings.Contains(logged, "Connecting to SQLite database") {
		t.Error("SQLite initialization message was not logged")
	}
	if strings.Contains(logged, sensitiveValue) {
		t.Error("SQLite initialization logged the sensitive parameter")
	}
	if strings.Contains(logged, dsn) {
		t.Error("SQLite initialization logged the full DSN")
	}
}
