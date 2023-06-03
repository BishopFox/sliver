// Package bash provides bash completion
package bash

import (
	"fmt"

	"github.com/rsteube/carapace/internal/uid"
	"github.com/spf13/cobra"
)

// Snippet creates the bash completion script.
func Snippet(cmd *cobra.Command) string {
	result := fmt.Sprintf(`#!/bin/bash
_%v_completion() {
  export COMP_WORDBREAKS

  local compline="${COMP_LINE:0:${COMP_POINT}}"
  local IFS=$'\n'
  mapfile -t COMPREPLY < <(echo "$compline" | sed -e "s/ \$/ ''/" -e 's/"/\"/g' | xargs %v _carapace bash)
  [[ "${COMPREPLY[*]}" == "" ]] && COMPREPLY=() # fix for mapfile creating a non-empty array from empty command output

  compopt -o nospace
}

complete -F _%v_completion %v
`, cmd.Name(), uid.Executable(), cmd.Name(), cmd.Name())

	return result
}
