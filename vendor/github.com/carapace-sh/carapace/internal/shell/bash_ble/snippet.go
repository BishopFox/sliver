package bash_ble

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/carapace-sh/carapace/internal/shell/bash"
	"github.com/carapace-sh/carapace/pkg/uid"

	"github.com/spf13/cobra"
)

// Snippet creates the bash-ble completion script.
func Snippet(cmd *cobra.Command) string {
	return SnippetSingle(cmd.Name(), false)
}

// SnippetMulti creates a multi-completer bash-ble completion script.
func SnippetMulti(names []string, defaultName string, snippetFuncs string) string {
	bashSnip := bash.SnippetMulti(names, defaultName, snippetFuncs)
	bashSnip = regexp.MustCompile(`complete -o noquote -F [^\n]+`).ReplaceAllString(bashSnip, "")

	quotedJoined := strings.Join(quoteAll(names), " ")

	return fmt.Sprintf(`%v
_%[2]v_completer_ble() {
  if [[ ${BLE_ATTACHED-} ]]; then
    [[ :$comp_type: == *:auto:* ]] && return

    compopt -o ble/no-default
    bleopt complete_menu_style=desc

    local command="${COMP_WORDS[0]}"
    local compline="${COMP_LINE:0:${COMP_POINT}}"
    local IFS=$'\n'
    local c
    mapfile -t c < <(echo "$compline" | sed -e "s/ \$/ ''/" -e 's/"/\"/g' | xargs %[3]v "${command}" _carapace bash-ble)
    [[ "${c[*]}" == "" ]] && c=() # fix for mapfile creating a non-empty array from empty command output

    local cand
    for cand in "${c[@]}"; do
      [ ! -z "$cand" ] && ble/complete/cand/yield mandb "${cand%%$'\t'*}" "${cand##*$'\t'}"
    done
  else
    complete -F _%[2]v_completer %[4]v
  fi
}

complete -F _%[2]v_completer_ble %[4]v
`, bashSnip, defaultName, uid.Executable(), quotedJoined)
}

// SnippetSingle creates a single-command bash-ble completion script.
// When explicitCommand is true, the command name is included in the invocation
// (for multi-completer subcommands). When false, the executable is invoked
// without an explicit command (standalone mode).
func SnippetSingle(command string, explicitCommand bool) string {
	bashSnip := bash.SnippetSingle(command, explicitCommand)
	if explicitCommand {
		bashSnip = regexp.MustCompile(`complete -o noquote -F [^\n]+`).ReplaceAllString(bashSnip, "")
	} else {
		bashSnip = regexp.MustCompile(`complete -F [^\n]+`).ReplaceAllString(bashSnip, "")
	}

	invocation := fmt.Sprintf("%v _carapace bash-ble", uid.Executable())
	if explicitCommand {
		invocation = fmt.Sprintf("%v %v _carapace bash-ble", uid.Executable(), command)
	}

	return fmt.Sprintf(`%v
_%[2]v_completion_ble() {
  if [[ ${BLE_ATTACHED-} ]]; then
    [[ :$comp_type: == *:auto:* ]] && return

    compopt -o ble/no-default
    bleopt complete_menu_style=desc

    local compline="${COMP_LINE:0:${COMP_POINT}}"
    local IFS=$'\n'
    local c
    mapfile -t c < <(echo "$compline" | sed -e "s/ \$/ ''/" -e 's/"/\"/g' | xargs %[3]v)
    [[ "${c[*]}" == "" ]] && c=() # fix for mapfile creating a non-empty array from empty command output

    local cand
    for cand in "${c[@]}"; do
      [ ! -z "$cand" ] && ble/complete/cand/yield mandb "${cand%%$'\t'*}" "${cand##*$'\t'}"
    done
  else
    complete -F _%[2]v_completion %[2]v
  fi
}

complete -F _%[2]v_completion_ble %[2]v
`, bashSnip, command, invocation)
}

func quoteAll(names []string) []string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = fmt.Sprintf("%q", name)
	}
	return quoted
}
