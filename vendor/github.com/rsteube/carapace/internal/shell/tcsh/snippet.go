// Package tcsh provides tcsh completion
package tcsh

import (
	"fmt"

	"github.com/rsteube/carapace/internal/uid"
	"github.com/spf13/cobra"
)

// Snippet creates the tcsh completion script.
func Snippet(cmd *cobra.Command) string {
	// TODO initial version - needs to handle open quotes
	return fmt.Sprintf("complete \"%v\" 'p@*@`echo \"$COMMAND_LINE'\"''\"'\" | xargs %v _carapace tcsh `@@' ;", cmd.Name(), uid.Executable())
}
