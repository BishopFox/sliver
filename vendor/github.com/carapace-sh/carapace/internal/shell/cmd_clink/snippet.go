package cmd_clink

import (
	"fmt"

	"github.com/carapace-sh/carapace/pkg/uid"
	"github.com/spf13/cobra"
)

func Snippet(cmd *cobra.Command) string {
	result := fmt.Sprintf(`local function %v_completion(word, word_index, line_state, match_builder)
  local compline = string.sub(line_state:getline(), 1, line_state:getcursor())

  local output = io.popen("env CARAPACE_COMPLINE=" .. string.format("%%q", compline) .. " %v _carapace cmd-clink \"\""):read("*a")
  for line in string.gmatch(output, '[^\r\n]+') do
    local matches = {}
    for m in string.gmatch(line, '[^\t]+') do
      table.insert(matches, m)
    end
    match_builder:addmatch({
      match = matches[1],
      display = matches[2],
      description = matches[3],
      type = "word",
      appendchar = matches[4],
      suppressappend = false
    })
  end
  return true
end

clink.argmatcher("%v"):addarg({nowordbreakchars="'&backprime;+;,", %v_completion}):loop(1)
`, cmd.Name(), uid.Executable(), cmd.Name(), cmd.Name())
	return result
}
