package cmd_clink

import (
	"fmt"
	"strings"

	"github.com/carapace-sh/carapace/pkg/uid"
	"github.com/spf13/cobra"
)

func sanitize(s string) string {
	return strings.ReplaceAll(s, "-", "__")
}

func Snippet(cmd *cobra.Command) string {
	result := fmt.Sprintf(`local function %[1]s_completion(word, word_index, line_state, match_builder)
    match_builder:setnosort()
    match_builder:setvolatile()
    os.setenv('CARAPACE_COMPLINE', line_state:getline():sub(1, line_state:getcursor()))

    local file, pclose = io.popenyield('%[3]s _carapace cmd-clink %[2]s')

    if not file then
        return false
    end

    for line in file:lines() do
        local matches = string.explode(line, '\t')

        if matches[1] then
            match_builder:addmatch({
                match       = matches[1],
                display     = matches[2],
                description = matches[3],
                type        = 'word',
                appendchar  = matches[4] or ''
            })
        end
    end

    if pclose then
        pclose()
    else
        file:close()
    end

    return not match_builder:isempty()
end

clink.argmatcher(50, '%[2]s', '%[2]s.exe'):addarg({nowordbreakchars="'`+"`"+`=+;,", %[1]s_completion}):loop(1)
`, sanitize(cmd.Name()), cmd.Name(), uid.Executable())
	return result
}
