package nushell

import (
	"fmt"
	"strings"

	"github.com/carapace-sh/carapace/pkg/uid"
	"github.com/spf13/cobra"
)

func sanitizeName(s string) string {
	return strings.ReplaceAll(s, "-", "__")
}

// Snippet creates the nushell completion script.
func Snippet(cmd *cobra.Command) string {
	return SnippetSingle(cmd.Name(), false)
}

// SnippetMulti creates a multi-completer nushell completion script.
func SnippetMulti(names []string, defaultName string, snippetFuncs string) string {
	return fmt.Sprintf(`%[4]vlet %[1]v_completer = {|spans|
    %[2]v $spans.0 _carapace nushell ...$spans | from json
}

mut current = (($env | default {} config).config | default {} completions)
$current.completions = ($current.completions | default {} external)
$current.completions.external = ($current.completions.external
    | default true enable
    |# backwards compatible workaround for default, see nushell #15654
    | upsert completer { if $in == null { $%[1]v_completer } else { $in } })

$env.config = $current
`, sanitizeName(defaultName), uid.Executable(), "", snippetFuncs)
}

// SnippetSingle creates a single-command nushell completion script.
// When explicitCommand is true, the command name is included in the invocation
// (for multi-completer subcommands) and the full config setup is included.
// When false, only the minimal completer function is generated (standalone mode).
func SnippetSingle(command string, explicitCommand bool) string {
	if explicitCommand {
		return fmt.Sprintf(`let %[2]v_completer = {|spans|
    %[1]v %[3]v _carapace nushell ...$spans | from json
}

mut current = (($env | default {} config).config | default {} completions)
$current.completions = ($current.completions | default {} external)
$current.completions.external = ($current.completions.external
    | default true enable
    |# backwards compatible workaround for default, see nushell #15654
    | upsert completer { if $in == null { $%[2]v_completer } else { $in } })

$env.config = $current
`, uid.Executable(), sanitizeName(command), command)
	}

	return fmt.Sprintf(`let %v_completer = {|spans| 
    %v _carapace nushell ...$spans | from json
}`, sanitizeName(command), uid.Executable())
}
