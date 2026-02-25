//go:build server

package testutils

import (
	"fmt"
	"os"
	"strings"

	"github.com/bishopfox/sliver/server/assets"
)

const (
	ServerRootEnv = "SLIVER_ROOT_DIR"
	ClientRootEnv = "SLIVER_CLIENT_ROOT_DIR"
)

// EnvOptions configures how SetupTestEnv prepares test roots.
type EnvOptions struct {
	ForceFreshServerRoot bool
	ForceFreshClientRoot bool
	UnpackServerAssets   bool
}

// EnvState tracks test root directories and what must be cleaned up.
type EnvState struct {
	ServerRoot  string
	ClientRoot  string
	cleanupDirs []string
}

// SetupTestEnv ensures SLIVER_ROOT_DIR/SLIVER_CLIENT_ROOT_DIR are set to usable
// directories for tests. It only removes directories it creates.
func SetupTestEnv(opts EnvOptions) (*EnvState, error) {
	state := &EnvState{}

	serverRoot, serverCreated, err := ensureRootEnv(ServerRootEnv, "sliver-test-root-*", opts.ForceFreshServerRoot)
	if err != nil {
		return nil, err
	}
	state.ServerRoot = serverRoot
	if serverCreated {
		state.cleanupDirs = append(state.cleanupDirs, serverRoot)
	}

	clientRoot, clientCreated, err := ensureRootEnv(ClientRootEnv, "sliver-test-client-root-*", opts.ForceFreshClientRoot)
	if err != nil {
		_ = state.Cleanup()
		return nil, err
	}
	state.ClientRoot = clientRoot
	if clientCreated {
		state.cleanupDirs = append(state.cleanupDirs, clientRoot)
	}

	if opts.UnpackServerAssets {
		assets.Setup(true, false)
	}

	return state, nil
}

// Cleanup removes any temporary directories created by SetupTestEnv.
func (s *EnvState) Cleanup() error {
	if s == nil {
		return nil
	}

	var errs []string
	for i := len(s.cleanupDirs) - 1; i >= 0; i-- {
		dir := s.cleanupDirs[i]
		if err := os.RemoveAll(dir); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", dir, err))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("test env cleanup failed: %s", strings.Join(errs, "; "))
}

func ensureRootEnv(envVar string, pattern string, forceFresh bool) (string, bool, error) {
	if !forceFresh {
		if current := os.Getenv(envVar); current != "" {
			if err := os.MkdirAll(current, 0o700); err != nil {
				return "", false, fmt.Errorf("create %s root %q: %w", envVar, current, err)
			}
			return current, false, nil
		}
	}

	root, err := os.MkdirTemp("", pattern)
	if err != nil {
		return "", false, fmt.Errorf("create %s temp root: %w", envVar, err)
	}
	if err := os.Setenv(envVar, root); err != nil {
		_ = os.RemoveAll(root)
		return "", false, fmt.Errorf("set %s: %w", envVar, err)
	}
	return root, true, nil
}
