package multi

import (
	"fmt"
	"sort"
	"strings"

	"github.com/carapace-sh/carapace/internal/shell/bash"
	"github.com/carapace-sh/carapace/internal/shell/bash_ble"
	"github.com/carapace-sh/carapace/internal/shell/elvish"
	"github.com/carapace-sh/carapace/internal/shell/fish"
	"github.com/carapace-sh/carapace/internal/shell/nushell"
	"github.com/carapace-sh/carapace/internal/shell/oil"
	"github.com/carapace-sh/carapace/internal/shell/powershell"
	"github.com/carapace-sh/carapace/internal/shell/tcsh"
	"github.com/carapace-sh/carapace/internal/shell/xonsh"
	"github.com/carapace-sh/carapace/internal/shell/zsh"
	"github.com/carapace-sh/carapace/pkg/ps"
)

// Snippet generates a multi-completer shell snippet that registers all
// command completers at once.
func Snippet(shell string, names []string, defaultName string, snippetFuncs map[string][]string) (string, error) {
	if shell == "" {
		shell = ps.DetermineShell()
	}
	snippetFuncsStr := snippetFuncsForShell(shell, snippetFuncs)
	switch shell {
	case "bash":
		return bash.SnippetMulti(names, defaultName, snippetFuncsStr), nil
	case "bash-ble":
		return bash_ble.SnippetMulti(names, defaultName, snippetFuncsStr), nil
	case "zsh":
		return zsh.SnippetMulti(names, defaultName, snippetFuncsStr), nil
	case "fish":
		return fish.SnippetMulti(names, defaultName, snippetFuncsStr), nil
	case "elvish":
		return elvish.SnippetMulti(names, defaultName, snippetFuncsStr), nil
	case "nushell":
		return nushell.SnippetMulti(names, defaultName, snippetFuncsStr), nil
	case "powershell":
		return powershell.SnippetMulti(names, defaultName, snippetFuncsStr), nil
	case "xonsh":
		return xonsh.SnippetMulti(names, defaultName, snippetFuncsStr), nil
	case "oil":
		return oil.SnippetMulti(names, defaultName, snippetFuncsStr), nil
	case "tcsh":
		return tcsh.SnippetMulti(names, defaultName, snippetFuncsStr), nil
	case "export":
		return "", nil
	case "cmd-clink", "ion":
		return "", fmt.Errorf("unsupported shell for multi-completer: '%v'", shell)
	default:
		supported := []string{"bash", "bash-ble", "elvish", "fish", "nushell", "oil", "powershell", "tcsh", "xonsh", "zsh"}
		sort.Strings(supported)
		return "", fmt.Errorf("unsupported shell: '%v' [expected one of '%v']", shell, strings.Join(supported, "', '"))
	}
}

// SnippetOrEmpty is like Snippet but returns the snippet string on success
// or the error message on failure (for use in os.Exit flows).
func SnippetOrEmpty(shell string, names []string, defaultName string, snippetFuncs map[string][]string) string {
	s, err := Snippet(shell, names, defaultName, snippetFuncs)
	if err != nil {
		return err.Error()
	}
	return s
}

// SingleSnippet generates a shell snippet that registers completion for
// a single command (e.g. `binary identify _carapace bash`).
func SingleSnippet(shell string, command string, names []string, defaultName string, snippetFuncs map[string][]string) (string, error) {
	if shell == "" {
		shell = ps.DetermineShell()
	}
	switch shell {
	case "bash":
		return bash.SnippetSingle(command, true), nil
	case "bash-ble":
		return bash_ble.SnippetSingle(command, true), nil
	case "zsh":
		return zsh.SnippetSingle(command, true), nil
	case "fish":
		return fish.SnippetSingle(command, true), nil
	case "elvish":
		return elvish.SnippetSingle(command, true), nil
	case "nushell":
		return nushell.SnippetSingle(command, true), nil
	case "powershell":
		return powershell.SnippetSingle(command, true), nil
	case "xonsh":
		return xonsh.SnippetSingle(command, true), nil
	case "oil":
		return oil.SnippetSingle(command, true), nil
	case "tcsh":
		return tcsh.SnippetSingle(command, true), nil
	case "export":
		return "", nil
	case "cmd-clink", "ion":
		return "", fmt.Errorf("unsupported shell for multi-completer: '%v'", shell)
	default:
		supported := []string{"bash", "bash-ble", "elvish", "fish", "nushell", "oil", "powershell", "tcsh", "xonsh", "zsh"}
		sort.Strings(supported)
		return "", fmt.Errorf("unsupported shell: '%v' [expected one of '%v']", shell, strings.Join(supported, "', '"))
	}
}

// SingleSnippetOrEmpty is like SingleSnippet but returns the snippet string
// on success or the error message on failure.
func SingleSnippetOrEmpty(shell string, command string, names []string, defaultName string, snippetFuncs map[string][]string) string {
	s, err := SingleSnippet(shell, command, names, defaultName, snippetFuncs)
	if err != nil {
		return err.Error()
	}
	return s
}

// snippetFuncsForShell joins the snippet funcs for a given shell.
func snippetFuncsForShell(shell string, snippetFuncs map[string][]string) string {
	if snippetFuncs == nil {
		return ""
	}
	parts := snippetFuncs[shell]
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n") + "\n"
}
