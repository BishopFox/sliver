package health

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealthzMethodRouting(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		expected int
	}{
		{name: "GET returns 200", method: http.MethodGet, expected: http.StatusOK},
		{name: "HEAD returns 200", method: http.MethodHead, expected: http.StatusOK},
		{name: "POST returns 405", method: http.MethodPost, expected: http.StatusMethodNotAllowed},
		{name: "PUT returns 405", method: http.MethodPut, expected: http.StatusMethodNotAllowed},
		{name: "PATCH returns 405", method: http.MethodPatch, expected: http.StatusMethodNotAllowed},
		{name: "DELETE returns 405", method: http.MethodDelete, expected: http.StatusMethodNotAllowed},
		{name: "OPTIONS returns 405", method: http.MethodOptions, expected: http.StatusMethodNotAllowed},
	}
	handler := newHandler()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(test.method, healthzPath, nil))
			if recorder.Code != test.expected {
				t.Fatalf("status = %d, want %d", recorder.Code, test.expected)
			}
			if test.expected == http.StatusMethodNotAllowed {
				if allow := recorder.Header().Get("Allow"); !strings.Contains(allow, http.MethodGet) {
					t.Fatalf("Allow header %q does not contain %q", allow, http.MethodGet)
				}
			}
		})
	}
}

func TestHealthzResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	newHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, healthzPath, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", contentType, "application/json")
	}
	if cacheControl := recorder.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("Cache-Control = %q, want %q", cacheControl, "no-store")
	}
	body, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Fatalf("failed to read body: %s", err)
	}
	if string(body) != "{\"status\":\"ok\"}\n" {
		t.Fatalf("body = %q, want %q", string(body), "{\"status\":\"ok\"}\n")
	}
}

func TestHealthzUnknownPath(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "root path", path: "/"},
		{name: "health path", path: "/health"},
		{name: "trailing slash", path: "/healthz/"},
		{name: "nested path", path: "/healthz/extra"},
		{name: "metrics path", path: "/metrics"},
	}
	handler := newHandler()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
			if recorder.Code != http.StatusNotFound {
				t.Fatalf("path %s status = %d, want %d", test.path, recorder.Code, http.StatusNotFound)
			}
		})
	}
}

func TestIsLoopbackAddress(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		expected bool
	}{
		{name: "ipv4 loopback", addr: "127.0.0.1:31339", expected: true},
		{name: "localhost", addr: "localhost:31339", expected: true},
		{name: "ipv6 loopback", addr: "[::1]:31339", expected: true},
		{name: "full ipv4 loopback range", addr: "127.200.0.1:1", expected: true},
		{name: "all interfaces ipv4", addr: ":31339", expected: false},
		{name: "wildcard ipv4", addr: "0.0.0.0:31339", expected: false},
		{name: "wildcard ipv6", addr: "[::]:31339", expected: false},
		{name: "private ipv4", addr: "192.168.1.10:31339", expected: false},
		{name: "hostname", addr: "example.com:80", expected: false},
		{name: "empty addr", addr: "", expected: false},
		{name: "missing port", addr: "127.0.0.1", expected: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := isLoopbackAddress(test.addr); actual != test.expected {
				t.Fatalf("isLoopbackAddress(%q) = %v, want %v", test.addr, actual, test.expected)
			}
		})
	}
}

func TestServerStartServeShutdown(t *testing.T) {
	server, err := Start("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Start failed: %s", err)
	}
	t.Cleanup(func() {
		if shutdownErr := server.Shutdown(context.Background()); shutdownErr != nil {
			t.Errorf("shutdown failed: %s", shutdownErr)
		}
	})

	addr := server.Addr()
	if addr == "" {
		t.Fatal("Addr() is empty after Start")
	}
	if host, _, err := net.SplitHostPort(addr); err != nil || host != "127.0.0.1" {
		t.Fatalf("Addr() = %q, want host 127.0.0.1", addr)
	}

	client := &http.Client{Timeout: time.Second}
	resp, err := client.Get("http://" + addr + healthzPath)
	if err != nil {
		t.Fatalf("GET %s failed: %s", healthzPath, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to decode body %q: %s", string(body), err)
	}
	if payload.Status != "ok" {
		t.Fatalf("status field = %q, want %q", payload.Status, "ok")
	}

	if shutdownErr := server.Shutdown(context.Background()); shutdownErr != nil {
		t.Fatalf("Shutdown failed: %s", shutdownErr)
	}
	if _, err := client.Get("http://" + addr + healthzPath); err == nil {
		t.Fatal("GET after Shutdown succeeded, want connection error")
	}
}

func TestServerStartBindError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve listener: %s", err)
	}
	defer listener.Close()

	server, err := Start(listener.Addr().String())
	if err == nil {
		server.Shutdown(context.Background())
		t.Fatal("Start succeeded on an already-bound address, want error")
	}
	if server != nil {
		t.Fatal("Start returned a non-nil Server with an error")
	}
}
