// Package nushell provides Nushell completion
package nushell

import (
	"fmt"

	"github.com/rsteube/carapace/internal/uid"
	"github.com/spf13/cobra"
)

// Snippet creates the nushell completion script.
func Snippet(cmd *cobra.Command) string {
	return fmt.Sprintf(`let external_completer = {|spans| 
  {
    $spans.0: { } # default
    %v: { %v _carapace nushell $spans | from json }
  } | get $spans.0 | each {|it| do $it}
}

let-env config = {
  completions: {
    external: {
      enable: true
      completer: $external_completer
    }
  }
}`, cmd.Name(), uid.Executable())
}
