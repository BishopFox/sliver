package gonsole

import (
	"os"
	"sort"
	"strings"

	"github.com/maxlandon/readline"
)

// EnvironmentVariables - Returns all environment variables as suggestions.
func (c *CommandCompleter) EnvironmentVariables() (completions []*readline.CompletionGroup) {

	grp := &readline.CompletionGroup{
		Name:          "console OS environment",
		MaxLength:     5, // Should be plenty enough
		DisplayType:   readline.TabDisplayGrid,
		TrimSlash:     true, // Some variables can be paths
		PathSeparator: getOSPathSeparator(),
		Descriptions:  map[string]string{},
	}

	var clientEnv = map[string]string{}
	env := os.Environ()

	for _, kv := range env {
		key := strings.Split(kv, "=")[0]
		value := strings.Split(kv, "=")[1]
		clientEnv[key] = value
	}

	keys := make([]string, 0, len(clientEnv))
	for k := range clientEnv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		grp.Suggestions = append(grp.Suggestions, key)
		value := clientEnv[key]
		grp.Descriptions[key] = value
	}

	// Add some special ones
	grp.Aliases = map[string]string{
		"~": "HOME",
	}

	completions = append(completions, grp)

	return
}

func (c *CommandCompleter) getDefaultExpansions() {

	// ~ . .. *

	// OS: windows, darwin, linux

	// windows: os.GetEnv(HOMEPATH)
	// hardcoded: \Users\<usename>
}
