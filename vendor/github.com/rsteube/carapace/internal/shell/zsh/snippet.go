// Package zsh provides zsh completion
package zsh

import (
	"fmt"

	"github.com/rsteube/carapace/internal/uid"
	"github.com/spf13/cobra"
)

// Snippet creates the zsh completion script
func Snippet(cmd *cobra.Command) string {
	return fmt.Sprintf(`#compdef %v
function _%v_completion {
  local IFS=$'\n'
  
  # shellcheck disable=SC2086,SC2154,SC2155
  if echo ${words}"''" | xargs echo 2>/dev/null > /dev/null; then
    local lines="$(echo ${words}"''" | CARAPACE_ZSH_HASH_DIRS="$(hash -d)" xargs %v _carapace zsh )"
  elif echo ${words} | sed "s/\$/'/" | xargs echo 2>/dev/null > /dev/null; then
    local lines="$(echo ${words} | sed "s/\$/'/" | CARAPACE_ZSH_HASH_DIRS="$(hash -d)" xargs %v _carapace zsh)"
  else
    local lines="$(echo ${words} | sed 's/$/"/' | CARAPACE_ZSH_HASH_DIRS="$(hash -d)" xargs %v _carapace zsh)"
  fi

  local zstyle message data
  IFS=$'\001' read -r -d '' zstyle message data <<<"${lines}"
  # shellcheck disable=SC2154
  zstyle ":completion:${curcontext}:*" list-colors "${zstyle}"
  zstyle ":completion:${curcontext}:*" group-name ''
  [ -z "$message" ] || _message -r "${message}"
  
  local block tag displays values displaysArr valuesArr
  while IFS=$'\002' read -r -d $'\002' block; do
    IFS=$'\003' read -r -d '' tag displays values <<<"${block}"
    # shellcheck disable=SC2034
    IFS=$'\n' read -r -d $'\004' -A displaysArr <<<"${displays}"$'\004'
    IFS=$'\n' read -r -d $'\004' -A valuesArr <<<"${values}"$'\004'
  
    [[ ${#valuesArr[@]} -gt 1 ]] && _describe -t "${tag}" "${tag}" displaysArr valuesArr -Q -S ''
  done <<<"${data}"
}
compquote '' 2>/dev/null && _%v_completion
compdef _%v_completion %v
`, cmd.Name(), cmd.Name(), uid.Executable(), uid.Executable(), uid.Executable(), cmd.Name(), cmd.Name(), cmd.Name())
}
