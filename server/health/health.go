package health

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/bishopfox/sliver/server/log"
)

const (
	healthzPath = "/healthz"

	readTimeout  = 5 * time.Second
	writeTimeout = 5 * time.Second
	idleTimeout  = 15 * time.Second
)

var healthLog = log.NamedLogger("daemon", "health")

// Server - Daemon-local HTTP health endpoint
type Server struct {
	listener net.Listener
	server   *http.Server
}

// Start - Bind addr and serve the health endpoint in the background
func Start(addr string) (*Server, error) {
	if !isLoopbackAddress(addr) {
		healthLog.Warnf("Health endpoint %s is NOT bound to a loopback interface and may be reachable from other hosts", addr)
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("health endpoint listen %s: %w", addr, err)
	}
	server := &http.Server{
		Handler:      newHandler(),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			healthLog.Errorf("Health endpoint server error: %s", serveErr)
		}
	}()
	return &Server{listener: listener, server: server}, nil
}

// Addr - The bound listen address; useful when addr used port 0
func (s *Server) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Shutdown - Gracefully stop serving; safe to call on a nil Server
func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+healthzPath, handleHealthz)
	return mux
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(struct {
		Status string `json:"status"`
	}{Status: "ok"}); err != nil {
		healthLog.Errorf("Failed to write health response: %s", err)
	}
}

func isLoopbackAddress(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil || host == "" {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
