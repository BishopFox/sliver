// Package xonsh provides Xonsh completion
package xonsh

import (
	"fmt"
	"strings"

	"github.com/rsteube/carapace/internal/uid"
	"github.com/spf13/cobra"
)

// Snippet creates the xonsh completion script.
func Snippet(cmd *cobra.Command) string {
	functionName := strings.Replace(cmd.Name(), "-", "__", -1)
	return fmt.Sprintf(`from xonsh.completers.tools import *

@contextual_command_completer
def _%v_completer(context):
    """carapace completer for %v"""
    if context.completing_command('%v'):
        from json import loads
        from subprocess import Popen, PIPE
        from xonsh.completers.tools import RichCompletion
        
        def fix_prefix(s):
            """quick fix for partially quoted prefix completion ('prefix',<TAB>)"""
            return s.translate(str.maketrans('', '', '\'"'))

        output, _ = Popen(['%v', '_carapace', 'xonsh', *[a.value for a in context.args], fix_prefix(context.prefix)], stdout=PIPE, stderr=PIPE).communicate()
        try:
            result = {RichCompletion(c["Value"], display=c["Display"], description=c["Description"], prefix_len=len(context.raw_prefix), append_closing_quote=False, style=c["Style"]) for c in loads(output)}
        except:
            result = {}
        if len(result) == 0:
            result = {RichCompletion(context.prefix, display=context.prefix, description='', prefix_len=len(context.raw_prefix), append_closing_quote=False)}
        return result


from xonsh.completers._aliases import _add_one_completer
_add_one_completer('%v', _%v_completer, 'start')
`, functionName, cmd.Name(), cmd.Name(), uid.Executable(), cmd.Name(), functionName)
}
