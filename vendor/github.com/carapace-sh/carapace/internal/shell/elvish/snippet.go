// Package elvish provides elvish completion
package elvish

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/carapace-sh/carapace/pkg/uid"
	"github.com/spf13/cobra"
)

// Snippet creates the elvish completion script.
func Snippet(cmd *cobra.Command) string {
	return SnippetSingle(cmd.Name(), false)
}

// SnippetMulti creates a multi-completer elvish completion script.
func SnippetMulti(names []string, defaultName string, snippetFuncs string) string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = fmt.Sprintf("%q", name)
	}
	windowsSnippet := ""
	if runtime.GOOS == "windows" {
		windowsSnippet = "\n    set edit:completion:arg-completer[$c.exe] = $edit:completion:arg-completer[$c]\n"
	}
	return fmt.Sprintf(`%[5]vput %[3]v | each {|c|
    set edit:completion:arg-completer[$c] = {|@arg|
        %[2]v $c _carapace elvish (all $arg) | from-json | each {|completion|
    		put $completion[Messages] | all (one) | each {|m|
    			edit:notify (styled "error: " red)$m
    		}
    		if (not-eq $completion[Usage] "") {
    			edit:notify (styled "usage: " $completion[DescriptionStyle])$completion[Usage]
    		}
    		put $completion[Candidates] | all (one) | peach {|c|
    			if (eq $c[Description] "") {
    		    	edit:complex-candidate $c[Value] &display=(styled $c[Display] $c[Style]) &code-suffix=$c[CodeSuffix]
    			} else {
    		    	edit:complex-candidate $c[Value] &display=(styled $c[Display] $c[Style])(styled " " $completion[DescriptionStyle]" bg-default")(styled "("$c[Description]")" $completion[DescriptionStyle]) &code-suffix=$c[CodeSuffix]
    			}
    		}
        }
    }%[4]v
}
`, defaultName, uid.Executable(), strings.Join(quoted, " "), windowsSnippet, snippetFuncs)
}

// SnippetSingle creates a single-command elvish completion script.
// When explicitCommand is true, the command name is included in the invocation
// (for multi-completer subcommands). When false, the executable is invoked
// without an explicit command (standalone mode).
func SnippetSingle(command string, explicitCommand bool) string {
	invocation := fmt.Sprintf("%v _carapace elvish (all $arg)", uid.Executable())
	if explicitCommand {
		invocation = fmt.Sprintf("%v %v _carapace elvish (all $arg)", uid.Executable(), command)
	}

	windowsSnippet := ""
	if runtime.GOOS == "windows" {
		windowsSnippet = fmt.Sprintf("\nset edit:completion:arg-completer[%v.exe] = $edit:completion:arg-completer[%v]\n", command, command)
	}

	return fmt.Sprintf(`set edit:completion:arg-completer[%[1]v] = {|@arg|
    %[2]v | from-json | each {|completion|
		put $completion[Messages] | all (one) | each {|m|
			edit:notify (styled "error: " red)$m
		}
		if (not-eq $completion[Usage] "") {
			edit:notify (styled "usage: " $completion[DescriptionStyle])$completion[Usage]
		}
		put $completion[Candidates] | all (one) | peach {|c|
			if (eq $c[Description] "") {
		    	edit:complex-candidate $c[Value] &display=(styled $c[Display] $c[Style]) &code-suffix=$c[CodeSuffix]
			} else {
		    	edit:complex-candidate $c[Value] &display=(styled $c[Display] $c[Style])(styled " " $completion[DescriptionStyle]" bg-default")(styled "("$c[Description]")" $completion[DescriptionStyle]) &code-suffix=$c[CodeSuffix]
			}
		}
    }
}%[3]v
`, command, invocation, windowsSnippet)
}
