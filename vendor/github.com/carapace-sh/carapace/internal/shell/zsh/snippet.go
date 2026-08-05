// Package zsh provides zsh completion
package zsh

import (
	"fmt"
	"strings"

	"github.com/carapace-sh/carapace/pkg/uid"
	"github.com/spf13/cobra"
)

// Snippet creates the zsh completion script.
func Snippet(cmd *cobra.Command) string {
	return SnippetSingle(cmd.Name(), false)
}

// SnippetMulti creates a multi-completer zsh completion script.
func SnippetMulti(names []string, defaultName string, snippetFuncs string) string {
	return fmt.Sprintf(`#compdef %[3]v
%[4]vfunction _%[1]v_completer {
  local command="$(basename $words[1])"
  local compline=${words[@]:0:$CURRENT}
  local IFS=$'\n'
  local lines

  # shellcheck disable=SC2086,SC2154,SC2155
  lines="$(echo "${compline}''" | CARAPACE_COMPLINE="${compline}" CARAPACE_ZSH_HASH_DIRS="$(hash -d)" xargs %[2]v "${command}" _carapace zsh 2>/dev/null)"
  if [ $? -eq 1 ]; then
    lines="$(echo "${compline}'" | CARAPACE_COMPLINE="${compline}" CARAPACE_ZSH_HASH_DIRS="$(hash -d)" xargs %[2]v "${command}" _carapace zsh 2>/dev/null)"
    if [ $? -eq 1 ]; then
      lines="$(echo "${compline}\"" | CARAPACE_COMPLINE="${compline}" CARAPACE_ZSH_HASH_DIRS="$(hash -d)" xargs %[2]v "${command}" _carapace zsh 2>/dev/null)"
    fi
  fi

  local zstyle message noprefix data
  IFS=$'\001' read -r -d '' zstyle message noprefix data <<<"${lines}"
  # shellcheck disable=SC2154
  zstyle ":completion:${curcontext}:*" list-colors "${zstyle}"
  zstyle ":completion:${curcontext}:*" group-name ''
  [ -z "$message" ] || _message -r "${message}"
  [[ "${noprefix}" = "true" ]] && compstate[insert]=menu
  
  local block tag displays values displaysArr valuesArr
  while IFS=$'\002' read -r -d $'\002' block; do
    IFS=$'\003' read -r -d '' tag displays values <<<"${block}"
    # shellcheck disable=SC2034
    IFS=$'\n' read -r -d $'\004' -A displaysArr <<<"${displays}"$'\004'
    IFS=$'\n' read -r -d $'\004' -A valuesArr <<<"${values}"$'\004'
  
    [[ ${#valuesArr[@]} -gt 1 ]] && _describe -t "${tag}" "${tag}" displaysArr valuesArr -Q -S ''
  done <<<"${data}"
}
compquote '' 2>/dev/null && _%[1]v_completer
compdef _%[1]v_completer %[3]v
`, defaultName, uid.Executable(), strings.Join(names, " "), snippetFuncs)
}

// SnippetSingle creates a single-command zsh completion script.
// When explicitCommand is true, the command name is included in the invocation
// (for multi-completer subcommands). When false, the executable is invoked
// without an explicit command (standalone mode).
func SnippetSingle(command string, explicitCommand bool) string {
	invocation := fmt.Sprintf("%v _carapace zsh", uid.Executable())
	if explicitCommand {
		invocation = fmt.Sprintf("%v %v _carapace zsh", uid.Executable(), command)
	}

	return fmt.Sprintf(`#compdef %[1]v
function _%[1]v_completion {
  local compline=${words[@]:0:$CURRENT}
  local IFS=$'\n'
  local lines

  # shellcheck disable=SC2086,SC2154,SC2155
  lines="$(echo "${compline}''" | CARAPACE_COMPLINE="${compline}" CARAPACE_ZSH_HASH_DIRS="$(hash -d)" xargs %[2]v 2>/dev/null)"
  if [ $? -eq 1 ]; then
    lines="$(echo "${compline}'" | CARAPACE_COMPLINE="${compline}" CARAPACE_ZSH_HASH_DIRS="$(hash -d)" xargs %[2]v 2>/dev/null)"
    if [ $? -eq 1 ]; then
      lines="$(echo "${compline}\"" | CARAPACE_COMPLINE="${compline}" CARAPACE_ZSH_HASH_DIRS="$(hash -d)" xargs %[2]v 2>/dev/null)"
    fi
  fi

  local zstyle message noprefix data
  IFS=$'\001' read -r -d '' zstyle message noprefix data <<<"${lines}"
  # shellcheck disable=SC2154
  zstyle ":completion:${curcontext}:*" list-colors "${zstyle}"
  zstyle ":completion:${curcontext}:*" group-name ''
  [ -z "$message" ] || _message -r "${message}"
  [[ "${noprefix}" = "true" ]] && compstate[insert]=menu
  
  local block tag displays values displaysArr valuesArr
  while IFS=$'\002' read -r -d $'\002' block; do
    IFS=$'\003' read -r -d '' tag displays values <<<"${block}"
    # shellcheck disable=SC2034
    IFS=$'\n' read -r -d $'\004' -A displaysArr <<<"${displays}"$'\004'
    IFS=$'\n' read -r -d $'\004' -A valuesArr <<<"${values}"$'\004'
  
    [[ ${#valuesArr[@]} -gt 1 ]] && _describe -t "${tag}" "${tag}" displaysArr valuesArr -Q -S ''
  done <<<"${data}"
}
compquote '' 2>/dev/null && _%[1]v_completion
compdef _%[1]v_completion %[1]v`, command, invocation)
}
