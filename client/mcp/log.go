package mcp

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/assets"
)

const mcpLogTimeFormat = "2006-01-02_15-04-05"

func newMCPLogger() (*log.Logger, *os.File, error) {
	logsDir := assets.GetMCPLogsDir()
	logPath := filepath.Join(logsDir, fmt.Sprintf("%s.log", time.Now().Format(mcpLogTimeFormat)))
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, nil, err
	}
	logger := log.New(logFile, "", log.LstdFlags|log.Lshortfile)
	return logger, logFile, nil
}

func (m *mcpManager) logf(format string, args ...any) {
	if m == nil {
		return
	}
	m.mu.Lock()
	logger := m.logger
	m.mu.Unlock()
	if logger != nil {
		logger.Printf(format, args...)
	}
}

func (s *SliverMCPServer) logf(format string, args ...any) {
	if s == nil || s.logger == nil {
		return
	}
	s.logger.Printf(format, args...)
}

func (s *SliverMCPServer) logToolCall(tool, sessionID, beaconID string, extras ...string) {
	if s == nil || s.logger == nil {
		return
	}
	parts := make([]string, 0, 2+len(extras))
	if sessionID != "" {
		parts = append(parts, fmt.Sprintf("session_id=%s", sessionID))
	}
	if beaconID != "" {
		parts = append(parts, fmt.Sprintf("beacon_id=%s", beaconID))
	}
	for _, extra := range extras {
		if extra != "" {
			parts = append(parts, extra)
		}
	}
	if len(parts) == 0 {
		s.logger.Printf("mcp tool=%s", tool)
		return
	}
	s.logger.Printf("mcp tool=%s %s", tool, strings.Join(parts, " "))
}
