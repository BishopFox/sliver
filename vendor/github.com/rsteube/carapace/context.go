package carapace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/rsteube/carapace/internal/common"
	"github.com/rsteube/carapace/internal/env"
	"github.com/rsteube/carapace/internal/shell/zsh"
	"github.com/rsteube/carapace/third_party/github.com/drone/envsubst"
	"github.com/rsteube/carapace/third_party/golang.org/x/sys/execabs"
)

// Context provides information during completion.
type Context struct {
	// Value contains the value currently being completed (or part of it during an ActionMultiParts).
	Value string
	// Args contains the positional arguments of current (sub)command (exclusive the one currently being completed).
	Args []string
	// Parts contains the splitted Value during an ActionMultiParts (exclusive the part currently being completed).
	Parts []string
	// Env contains environment variables for current context.
	Env []string
	// Dir contains the working directory for current context.
	Dir string

	mockedReplies map[string]string
}

// NewContext creates a new context for given arguments.
func NewContext(args ...string) Context {
	if len(args) == 0 {
		args = append(args, "")
	}

	context := Context{
		Value: args[len(args)-1],
		Args:  args[:len(args)-1],
		Env:   os.Environ(),
	}

	if wd, err := os.Getwd(); err == nil {
		context.Dir = wd
	}

	isGoRun := func() bool { return strings.HasPrefix(os.Args[0], os.TempDir()+"/go-build") }
	if sandbox := env.Sandbox(); sandbox != "" && isGoRun() {
		var m common.Mock
		_ = json.Unmarshal([]byte(sandbox), &m)
		context.Dir = m.Dir
		context.mockedReplies = m.Replies
	}
	return context
}

// LookupEnv retrieves the value of the environment variable named by the key.
func (c Context) LookupEnv(key string) (string, bool) {
	prefix := key + "="
	for i := len(c.Env) - 1; i >= 0; i-- {
		if env := c.Env[i]; strings.HasPrefix(env, prefix) {
			return strings.SplitN(env, "=", 2)[1], true
		}
	}
	return "", false
}

// Getenv retrieves the value of the environment variable named by the key.
func (c Context) Getenv(key string) string {
	v, _ := c.LookupEnv(key)
	return v
}

// Setenv sets the value of the environment variable named by the key.
func (c *Context) Setenv(key, value string) {
	if c.Env == nil {
		c.Env = []string{}
	}
	c.Env = append(c.Env, fmt.Sprintf("%v=%v", key, value))
}

// Envsubst replaces ${var} in the string based on environment variables in current context.
func (c Context) Envsubst(s string) (string, error) {
	return envsubst.Eval(s, c.Getenv)
}

// Command returns the Cmd struct to execute the named program with the given arguments.
// Env and Dir are set using the Context.
// See exec.Command for most details.
func (c Context) Command(name string, arg ...string) *execabs.Cmd {
	if c.mockedReplies != nil {
		if m, err := json.Marshal(append([]string{name}, arg...)); err == nil {
			if reply, exists := c.mockedReplies[string(m)]; exists {
				return execabs.Command("echo", reply)
			}
		}
	}

	cmd := execabs.Command(name, arg...)
	cmd.Env = c.Env
	cmd.Dir = c.Dir
	return cmd
}

func expandHome(s string) (string, error) {
	if strings.HasPrefix(s, "~") {
		if zsh.NamedDirectories.Matches(s) {
			return zsh.NamedDirectories.Replace(s), nil
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		s = strings.Replace(s, "~/", home+"/", 1)
	}
	return s, nil
}

func isWindowsVolume(path string) bool {
	if len(path) <= 1 {
		return false
	}

	// We need at least two characters,
	// of which the first must be a letter
	// and the second as colon.
	if unicode.IsLetter(rune(path[0])) && path[1] == ':' {
		return true
	}

	return false
}

// windowsDisplayTrimmed returns a trimmed display folder and true if
// the context value is a Windows volume (absolute path), or nothing and false.
func windowsDisplayTrimmed(abs, cValue, displayFolder string) (string, bool) {
	if !isWindowsVolume(cValue) {
		return displayFolder, false
	}

	// volume name such as C: => displayFolder then becomes C:.
	if !strings.HasSuffix(abs, ".") {
		displayFolder = strings.TrimSuffix(displayFolder, ".")
	}

	// If the context value is C:/, the display folder is still C:,
	// so we only add a trailing slash when the context value is C:
	// or if it's C:/Us (eg. longer than the volume root with slash).
	if len(cValue) == 2 || (len(displayFolder) > 3) && !strings.HasSuffix(displayFolder, "/") {
		displayFolder = displayFolder + "/"
	}

	return displayFolder, true
}

// Abs returns an absolute representation of path.
func (c Context) Abs(path string) (string, error) {
	if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "~") && !isWindowsVolume(path) { // path is relative
		switch c.Dir {
		case "":
			path = "./" + path
		default:
			path = c.Dir + "/" + path
		}
	}

	path, err := expandHome(path)
	if err != nil {
		return "", err
	}

	result, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	if strings.HasSuffix(path, "/") && !strings.HasSuffix(result, "/") {
		result += "/"
	} else if strings.HasSuffix(path, "/.") && !strings.HasSuffix(result, "/.") {
		result += "/."
	}
	return result, nil
}
