// Package uid provides unique identifiers
package uid

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/carapace-sh/carapace/internal/pflagfork"
	"github.com/spf13/cobra"
)

type Context interface {
	Abs(s string) (string, error)
	Getenv(key string) string
	LookupEnv(key string) (string, bool)
}

// UidF TODO experimental
func UidF(scheme, host string, opts ...string) func(v string, uc Context) (*url.URL, error) {
	return func(v string, uc Context) (*url.URL, error) {
		if length := len(opts); length%2 != 0 {
			return nil, fmt.Errorf("invalid amount of arguments [Uid]: %v", length)
		}

		uid := &url.URL{
			Scheme: scheme,
			Host:   url.PathEscape(host),
			Path:   PathEscape(v),
		}
		if len(opts) > 0 {
			values := uid.Query()
			for i := 0; i < len(opts); i += 2 {
				if opts[i+1] != "" { // implicitly skip empty values
					values.Set(opts[i], opts[i+1])
				}
			}
			uid.RawQuery = values.Encode()
		}
		return uid, nil
	}
}

// Command creates a uid for given command.
func Command(cmd *cobra.Command) *url.URL {
	path := []string{cmd.Name()}
	for parent := cmd.Parent(); parent != nil; parent = parent.Parent() {
		path = append(path, url.PathEscape(parent.Name()))
	}
	reverse(path) // TODO slices.Reverse
	return &url.URL{
		Scheme: "cmd",
		Host:   path[0],
		Path:   strings.Join(path[1:], "/"),
	}
}

// reverse reverses the elements of the slice in place.
func reverse(s []string) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

var mLocalFlags sync.Mutex

// Flag creates a uid for given flag.
func Flag(cmd *cobra.Command, flag *pflagfork.Flag) *url.URL {
	mLocalFlags.Lock()
	defer mLocalFlags.Unlock()
	return flagRecursive(cmd, flag)
}

func flagRecursive(cmd *cobra.Command, flag *pflagfork.Flag) *url.URL {
	_ = cmd.LocalFlags() // Force flag merge; not thread-safe internally

	if cmd.LocalFlags().Lookup(flag.Name) == nil && cmd.HasParent() {
		return flagRecursive(cmd.Parent(), flag)
	}
	uid := Command(cmd)
	values := uid.Query()
	values.Set("flag", flag.Name)
	uid.RawQuery = values.Encode()
	return uid
}

// callerModuleContains checks if any caller's source path contains the given substring.
func callerModuleContains(substr string) bool {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(0, pcs)
	for _, pc := range pcs[:n] {
		if fn := runtime.FuncForPC(pc); fn != nil {
			if strings.Contains(fn.Name(), substr) {
				return true
			}
		}
	}
	return false
}

// Executable returns the name of the executable.
func Executable() string {
	executable, err := os.Executable()
	if err != nil {
		return "echo" // safe fallback that should never happen
	}
	switch base := filepath.Base(executable); base {
	case "cmd.test":
		if callerModuleContains("example-multi") {
			return "example-multi" // for `go test -v ./...` in example-multi
		}
		return "example" // for `go test -v ./...` in example
	case "ld-musl-x86_64.so.1":
		return filepath.Base(os.Args[0]) // alpine container workaround (gcompat)
	default:
		return base
	}
}

// Map maps values to uids to simplify testing.
//
//	Map(
//	    "go.mod", "file://path/to/go.mod",
//	    "go.sum", "file://path/to/go.sum",
//	)
func Map(uids ...string) func(s string) (*url.URL, error) {
	return func(s string) (*url.URL, error) {
		for i := 0; i < len(uids); i += 2 {
			if uids[i] == s {
				return url.Parse(uids[i+1])
			}
		}
		return &url.URL{}, nil
	}
}
