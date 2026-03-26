package mcp

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/client/assets"
	"gopkg.in/yaml.v3"
)

const (
	authConfigFileName = "mcp.yaml"
	authHeaderName     = "Authorization"
	minAuthTokenLength = 8
	authTokenBytes     = 16
	sseMessagePath     = "/message"
)

// AuthInfo describes the persisted MCP HTTP/SSE authentication settings.
type AuthInfo struct {
	Header     string
	Token      string
	ConfigPath string
}

// AuthHeaderName returns the required header name for MCP HTTP/SSE requests.
func AuthHeaderName() string {
	return authHeaderName
}

// AuthConfigPath returns the MCP auth config path in the client app directory.
func AuthConfigPath() string {
	rootDir, _ := filepath.Abs(assets.GetRootAppDir())
	return filepath.Join(rootDir, authConfigFileName)
}

type authConfig struct {
	Token string `yaml:"token"`
}

// LoadAuthInfo loads the persisted MCP auth configuration without creating it.
func LoadAuthInfo() (AuthInfo, error) {
	cfg, exists, err := loadAuthConfig()
	if err != nil {
		return AuthInfo{}, err
	}
	if !exists {
		return AuthInfo{}, os.ErrNotExist
	}
	cfg.Token = strings.TrimSpace(cfg.Token)
	if err := validateAuthToken(cfg.Token); err != nil {
		return AuthInfo{}, fmt.Errorf("invalid mcp token in %s: %w", AuthConfigPath(), err)
	}
	return AuthInfo{
		Header:     authHeaderName,
		Token:      cfg.Token,
		ConfigPath: AuthConfigPath(),
	}, nil
}

// ResolveAuthInfo loads the persisted auth token or creates and saves one on first use.
func ResolveAuthInfo() (AuthInfo, bool, error) {
	cfg, exists, err := loadAuthConfig()
	if err != nil {
		return AuthInfo{}, false, err
	}
	if exists {
		cfg.Token = strings.TrimSpace(cfg.Token)
		if err := validateAuthToken(cfg.Token); err != nil {
			return AuthInfo{}, false, fmt.Errorf("invalid mcp token in %s: %w", AuthConfigPath(), err)
		}
		return AuthInfo{
			Header:     authHeaderName,
			Token:      cfg.Token,
			ConfigPath: AuthConfigPath(),
		}, false, nil
	}

	token, err := generateAuthToken()
	if err != nil {
		return AuthInfo{}, false, err
	}
	cfg = &authConfig{Token: token}
	if err := saveAuthConfig(cfg); err != nil {
		return AuthInfo{}, false, err
	}
	return AuthInfo{
		Header:     authHeaderName,
		Token:      token,
		ConfigPath: AuthConfigPath(),
	}, true, nil
}

func loadAuthConfig() (*authConfig, bool, error) {
	data, err := os.ReadFile(AuthConfigPath())
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	cfg := &authConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, true, err
	}
	return cfg, true, nil
}

func saveAuthConfig(cfg *authConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(AuthConfigPath(), data, 0o600)
}

func generateAuthToken() (string, error) {
	tokenBytes := make([]byte, authTokenBytes)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("generate mcp auth token: %w", err)
	}
	return hex.EncodeToString(tokenBytes), nil
}

func validateAuthToken(token string) error {
	if len(strings.TrimSpace(token)) < minAuthTokenLength {
		return fmt.Errorf("token must be at least %d characters", minAuthTokenLength)
	}
	return nil
}

func authMiddleware(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !authTokenMatches(r.Header.Get(authHeaderName), token) {
			http.Error(w, fmt.Sprintf("missing or invalid %s header", authHeaderName), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func authTokenMatches(actual string, expected string) bool {
	if len(expected) == 0 || len(actual) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

type authenticatedHTTPTransport struct {
	server *http.Server
}

func newAuthenticatedTransport(cfg Config, handler http.Handler, token string) serverTransport {
	mux := http.NewServeMux()
	authenticatedHandler := authMiddleware(token, handler)

	switch cfg.Transport {
	case TransportHTTP:
		mux.Handle("/mcp", authenticatedHandler)
	case TransportSSE:
		mux.Handle("/sse", authenticatedHandler)
		mux.Handle(sseMessagePath, authenticatedHandler)
	}

	return &authenticatedHTTPTransport{
		server: &http.Server{
			Handler: mux,
		},
	}
}

func (t *authenticatedHTTPTransport) Start(addr string) error {
	if t.server.Addr == "" {
		t.server.Addr = addr
	} else if t.server.Addr != addr {
		return fmt.Errorf("conflicting listen address: %q vs %q", t.server.Addr, addr)
	}
	return t.server.ListenAndServe()
}

func (t *authenticatedHTTPTransport) Shutdown(ctx context.Context) error {
	return t.server.Shutdown(ctx)
}
