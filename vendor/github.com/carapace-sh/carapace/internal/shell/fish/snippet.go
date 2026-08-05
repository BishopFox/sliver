// Package fish provides fish completion
package fish

import (
	"fmt"
	"strings"

	"github.com/carapace-sh/carapace/pkg/uid"
	"github.com/spf13/cobra"
)

// Snippet creates the fish completion script.
func Snippet(cmd *cobra.Command) string {
	return SnippetSingle(cmd.Name(), false)
}

// SnippetMulti creates a multi-completer fish completion script.
func SnippetMulti(names []string, defaultName string, snippetFuncs string) string {
	complete := make([]string, 0, len(names)*2)
	for _, name := range names {
		complete = append(complete,
			fmt.Sprintf(`complete -e %q`, name),
			fmt.Sprintf(`complete -c %[2]q -f -a '(_%[1]v_completer %[2]q)' -r`, defaultName, name),
		)
	}
	return fmt.Sprintf(`%[4]vfunction _%[1]v_completer
  set --local data
  IFS='' set data (echo (commandline -cp)'' | sed "s/ \$/ ''/" | xargs %[2]v $argv[1] _carapace fish 2>/dev/null)
  if [ $status -eq 1 ]
    IFS='' set data (echo (commandline -cp)"'" | sed "s/ \$/ ''/" | xargs %[2]v $argv[1] _carapace fish 2>/dev/null)
    if [ $status -eq 1 ]
      IFS='' set data (echo (commandline -cp)'"' | sed "s/ \$/ ''/" | xargs %[2]v $argv[1] _carapace fish 2>/dev/null)
    end
  end
  echo $data
end

%v
`, defaultName, uid.Executable(), strings.Join(complete, "\n"), snippetFuncs)
}

// SnippetSingle creates a single-command fish completion script.
// When explicitCommand is true, the command name is included in the invocation
// (for multi-completer subcommands). When false, the executable is invoked
// without an explicit command (standalone mode).
func SnippetSingle(command string, explicitCommand bool) string {
	invocation := fmt.Sprintf("%v _carapace fish", uid.Executable())
	if explicitCommand {
		invocation = fmt.Sprintf("%v %v _carapace fish", uid.Executable(), command)
	}

	return fmt.Sprintf(`function _%[1]v_completion
  set --local data
  IFS='' set data (echo (commandline -cp)'' | sed "s/ \$/ ''/" | xargs %[2]v 2>/dev/null)
  if [ $status -eq 1 ]
    IFS='' set data (echo (commandline -cp)"'" | sed "s/ \$/ ''/" | xargs %[2]v 2>/dev/null)
    if [ $status -eq 1 ]
      IFS='' set data (echo (commandline -cp)'"' | sed "s/ \$/ ''/" | xargs %[2]v 2>/dev/null)
    end
  end
  echo $data
end

complete -e '%[1]v'
complete -c '%[1]v' -f -a '(_%[1]v_completion)' -r
`, command, invocation)
}
