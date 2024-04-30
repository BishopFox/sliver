// Package nushell provides Nushell completion
package nushell

import (
	"fmt"

	"github.com/rsteube/carapace/internal/uid"
	"github.com/spf13/cobra"
)

// Snippet creates the nushell completion script.
func Snippet(cmd *cobra.Command) string {
	return fmt.Sprintf(`let %v_completer = {|spans| 
    %v _carapace nushell $spans | from json
}`, cmd.Name(), uid.Executable())
}
