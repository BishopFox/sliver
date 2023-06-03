// Package bash_ble provides bash-ble completion
package bash_ble

import (
	"fmt"
	"regexp"

	"github.com/rsteube/carapace/internal/shell/bash"
	"github.com/rsteube/carapace/internal/uid"
	"github.com/spf13/cobra"
)

// Snippet creates the bash-ble completion script.
func Snippet(cmd *cobra.Command) string {
	bashSnippet := bash.Snippet(cmd)
	bashSnippet = regexp.MustCompile("complete -F [^\n]+").ReplaceAllString(bashSnippet, "")

	result := fmt.Sprintf(`
_%v_completion_ble() {
  if [[ ${BLE_ATTACHED-} ]]; then
    [[ :$comp_type: == *:auto:* ]] && return

    compopt -o ble/no-default
    bleopt complete_menu_style=desc

    local compline="${COMP_LINE:0:${COMP_POINT}}"
    local IFS=$'\n'
    local c
    mapfile -t c < <(echo "$compline" | sed -e "s/ \$/ ''/" -e 's/"/\"/g' | xargs %v _carapace bash-ble)
    [[ "${c[*]}" == "" ]] && c=() # fix for mapfile creating a non-empty array from empty command output

    local cand
    for cand in "${c[@]}"; do
      [ ! -z "$cand" ] && ble/complete/cand/yield mandb "${cand%%$'\t'*}" "${cand##*$'\t'}"
    done
  else
    complete -F _%v_completion %v
  fi
}

complete -F _%v_completion_ble %v
`, cmd.Name(), uid.Executable(), cmd.Name(), cmd.Name(), cmd.Name(), cmd.Name())

	return bashSnippet + result
}
