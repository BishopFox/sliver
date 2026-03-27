package mcp

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestResolveAuthInfoCreatesAndPersistsToken(t *testing.T) {
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", t.TempDir())

	authInfo, created, err := ResolveAuthInfo()
	if err != nil {
		t.Fatalf("ResolveAuthInfo returned error: %v", err)
	}
	if !created {
		t.Fatalf("expected first ResolveAuthInfo call to create a token")
	}
	if authInfo.Header != AuthHeaderName() {
		t.Fatalf("expected header %q, got %q", AuthHeaderName(), authInfo.Header)
	}
	if len(authInfo.Token) != authTokenBytes*2 {
		t.Fatalf("expected %d hex characters, got %d", authTokenBytes*2, len(authInfo.Token))
	}

	data, err := os.ReadFile(AuthConfigPath())
	if err != nil {
		t.Fatalf("failed to read persisted auth config: %v", err)
	}
	if !strings.Contains(string(data), "token: "+authInfo.Token) {
		t.Fatalf("persisted config did not contain generated token: %s", string(data))
	}

	authInfoAgain, createdAgain, err := ResolveAuthInfo()
	if err != nil {
		t.Fatalf("second ResolveAuthInfo returned error: %v", err)
	}
	if createdAgain {
		t.Fatalf("expected second ResolveAuthInfo call to reuse the existing token")
	}
	if authInfoAgain.Token != authInfo.Token {
		t.Fatalf("expected persisted token %q, got %q", authInfo.Token, authInfoAgain.Token)
	}
}

func TestResolveAuthInfoRejectsInvalidExistingToken(t *testing.T) {
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", t.TempDir())

	if err := os.WriteFile(AuthConfigPath(), []byte("token: short\n"), 0o600); err != nil {
		t.Fatalf("failed to seed auth config: %v", err)
	}

	_, _, err := ResolveAuthInfo()
	if err == nil {
		t.Fatal("expected invalid token error")
	}
	if !strings.Contains(err.Error(), "at least 8 characters") {
		t.Fatalf("expected length validation error, got %v", err)
	}
}

func TestAuthMiddlewareRequiresExactToken(t *testing.T) {
	const token = "0123456789abcdef"

	handler := authMiddleware(token, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	testCases := []struct {
		name       string
		header     string
		statusCode int
	}{
		{name: "missing", statusCode: http.StatusUnauthorized},
		{name: "wrong", header: "wrong-token", statusCode: http.StatusUnauthorized},
		{name: "prefixed", header: "Bearer " + token, statusCode: http.StatusUnauthorized},
		{name: "exact", header: token, statusCode: http.StatusNoContent},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
			if testCase.header != "" {
				req.Header.Set(AuthHeaderName(), testCase.header)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != testCase.statusCode {
				t.Fatalf("expected status %d, got %d", testCase.statusCode, recorder.Code)
			}
		})
	}
}
