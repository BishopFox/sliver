// Package elvish provides elvish completion
package elvish

import (
	"fmt"

	"github.com/rsteube/carapace/internal/uid"
	"github.com/spf13/cobra"
)

// Snippet creates the elvish completion script.
func Snippet(cmd *cobra.Command) string {
	return fmt.Sprintf(`set edit:completion:arg-completer[%v] = {|@arg|
    %v _carapace elvish (all $arg) | from-json | each {|completion|
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
}
`, cmd.Name(), uid.Executable())
}
