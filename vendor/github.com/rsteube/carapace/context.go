package carapace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rsteube/carapace/internal/env"
	"github.com/rsteube/carapace/internal/shell/zsh"
	"github.com/rsteube/carapace/pkg/util"
	"github.com/rsteube/carapace/third_party/github.com/drone/envsubst"
	"github.com/rsteube/carapace/third_party/golang.org/x/sys/execabs"
	"github.com/spf13/cobra"
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
	cmd           *cobra.Command // needed for ActionCobra
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

	if m, err := env.Sandbox(); err == nil {
		context.Dir = m.WorkDir()
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
		home = filepath.ToSlash(home)
		s = strings.Replace(s, "~/", home+"/", 1)
	}
	return s, nil
}

// Abs returns an absolute representation of path.
func (c Context) Abs(path string) (string, error) {
	path = filepath.ToSlash(path)
	if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "~") && !util.HasVolumePrefix(path) { // path is relative
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

	if len(path) == 2 && util.HasVolumePrefix(path) {
		path += "/" // prevent `C:` -> `C:./current/working/directory`
	}
	result, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	result = filepath.ToSlash(result)

	if strings.HasSuffix(path, "/") && !strings.HasSuffix(result, "/") {
		result += "/"
	} else if strings.HasSuffix(path, "/.") && !strings.HasSuffix(result, "/.") {
		result += "/."
	}
	return result, nil
}
