package tcsh

import (
	"fmt"
	"strings"

	"github.com/carapace-sh/carapace/pkg/uid"
	"github.com/spf13/cobra"
)

// Snippet creates the tcsh completion script.
func Snippet(cmd *cobra.Command) string {
	return SnippetSingle(cmd.Name(), false)
}

// SnippetMulti creates a multi-completer tcsh completion script.
func SnippetMulti(names []string, defaultName string, snippetFuncs string) string {
	lines := make([]string, len(names))
	for i, name := range names {
		lines[i] = fmt.Sprintf("complete \"%v\" 'p@*@`echo \"$COMMAND_LINE'\"''\"'\" | xargs %v \"%v\" _carapace tcsh `@@' ;", name, uid.Executable(), name)
	}
	return strings.Join(lines, "\n")
}

// SnippetSingle creates a single-command tcsh completion script.
// When explicitCommand is true, the command name is included in the invocation
// (for multi-completer subcommands). When false, the executable is invoked
// without an explicit command (standalone mode).
func SnippetSingle(command string, explicitCommand bool) string {
	// TODO initial version - needs to handle open quotes
	invocation := fmt.Sprintf("%v _carapace tcsh", uid.Executable())
	if explicitCommand {
		invocation = fmt.Sprintf("%v \"%v\" _carapace tcsh", uid.Executable(), command)
	}

	return fmt.Sprintf("complete \"%v\" 'p@*@`echo \"$COMMAND_LINE'\"''\"'\" | xargs %v `@@' ;", command, invocation)
}
