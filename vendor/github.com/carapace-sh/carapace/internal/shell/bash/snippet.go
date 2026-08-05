package bash

import (
	"fmt"
	"strings"

	"github.com/carapace-sh/carapace/pkg/uid"
	"github.com/spf13/cobra"
)

// Snippet creates the bash completion script.
func Snippet(cmd *cobra.Command) string {
	return SnippetSingle(cmd.Name(), false)
}

// SnippetMulti creates a multi-completer bash completion script.
func SnippetMulti(names []string, defaultName string, snippetFuncs string) string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = fmt.Sprintf("%q", name)
	}
	return fmt.Sprintf(`#!/bin/bash
%[4]v_%[1]v_completer() {
  export COMP_LINE
  export COMP_POINT
  export COMP_TYPE
  export COMP_WORDBREAKS

  local nospace data compline="${COMP_LINE:0:${COMP_POINT}}"
  local command="${COMP_WORDS[0]}"

  data=$(echo "${compline}''" | xargs %[2]v "${command}" _carapace bash 2>/dev/null)
  if [ $? -eq 1 ]; then
    data=$(echo "${compline}'" | xargs %[2]v "${command}" _carapace bash 2>/dev/null)
    if [ $? -eq 1 ]; then
    	data=$(echo "${compline}\"" | xargs %[2]v "${command}" _carapace bash 2>/dev/null)
    fi
  fi

  IFS=$'\001' read -r -d '' nospace data <<<"${data}"
  mapfile -t COMPREPLY < <(echo "${data}")
  unset COMPREPLY[-1]

  [ "${nospace}" = true ] && compopt -o nospace
  local IFS=$'\n'
  [[ "${COMPREPLY[*]}" == "" ]] && COMPREPLY=() # fix for mapfile creating a non-empty array from empty command output
}

complete -o noquote -F _%[1]v_completer %[3]v
`, defaultName, uid.Executable(), strings.Join(quoted, " "), snippetFuncs)
}

// SnippetSingle creates a single-command bash completion script.
// When explicitCommand is true, the command name is included in the invocation
// (for multi-completer subcommands). When false, the executable is invoked
// without an explicit command (standalone mode).
func SnippetSingle(command string, explicitCommand bool) string {
	invocation := fmt.Sprintf("%v _carapace bash", uid.Executable())
	if explicitCommand {
		invocation = fmt.Sprintf("%v %v _carapace bash", uid.Executable(), command)
	}

	return fmt.Sprintf(`#!/bin/bash
_%[1]v_completion() {
  export COMP_LINE
  export COMP_POINT
  export COMP_TYPE
  export COMP_WORDBREAKS

  local nospace data compline="${COMP_LINE:0:${COMP_POINT}}"

  data=$(echo "${compline}''" | xargs %[2]v 2>/dev/null)
  if [ $? -eq 1 ]; then
    data=$(echo "${compline}'" | xargs %[2]v 2>/dev/null)
    if [ $? -eq 1 ]; then
    	data=$(echo "${compline}\"" | xargs %[2]v 2>/dev/null)
    fi
  fi

  IFS=$'\001' read -r -d '' nospace data <<<"${data}"
  mapfile -t COMPREPLY < <(echo "${data}")
  unset COMPREPLY[-1]

  [ "${nospace}" = true ] && compopt -o nospace
  local IFS=$'\n'
  [[ "${COMPREPLY[*]}" == "" ]] && COMPREPLY=() # fix for mapfile creating a non-empty array from empty command output
}

complete -o noquote -F _%[1]v_completion %[1]v
`, command, invocation)
}
