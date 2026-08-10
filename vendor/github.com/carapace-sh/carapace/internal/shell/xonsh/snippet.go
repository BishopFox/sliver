package xonsh

import (
	"fmt"
	"strings"

	"github.com/carapace-sh/carapace/pkg/uid"
	"github.com/spf13/cobra"
)

// Snippet creates the xonsh completion script.
func Snippet(cmd *cobra.Command) string {
	return SnippetSingle(cmd.Name(), false)
}

// SnippetMulti creates a multi-completer xonsh completion script.
func SnippetMulti(names []string, defaultName string, snippetFuncs string) string {
	complete := make([]string, len(names))
	for i, name := range names {
		complete[i] = fmt.Sprintf("'%v'", name)
	}
	return fmt.Sprintf(`%[5]vfrom xonsh.completers.completer import add_one_completer
from xonsh.completers.tools import contextual_command_completer

@contextual_command_completer
def _%[1]v_completer(context):
    """carapace multi-completer"""
    if context.command not in [%[3]v]:
        return

    from json import loads
    from xonsh.completers.tools import sub_proc_get_output, RichCompletion

    def fix_prefix(s):
        """quick fix for partially quoted prefix completion ('prefix',<TAB>)"""
        return s.translate(str.maketrans('', '', '\'"'))

    output, _ = sub_proc_get_output(
        '%[2]v', context.command, '_carapace', 'xonsh', *[a.value for a in context.args], fix_prefix(context.prefix)
    )

    try:
        result = {RichCompletion(c["Value"], display=c["Display"], description=c["Description"], prefix_len=len(context.raw_prefix), append_closing_quote=False, style=c["Style"]) for c in loads(output)}
    except:
        result = {}
    if len(result) == 0:
        result = {RichCompletion(context.prefix, display=context.prefix, description='', prefix_len=len(context.raw_prefix), append_closing_quote=False)}
    return result

add_one_completer('%[1]v', _%[1]v_completer, 'start')
`, strings.ReplaceAll(defaultName, "-", "__"), uid.Executable(), strings.Join(complete, ", "), "", snippetFuncs)
}

// SnippetSingle creates a single-command xonsh completion script.
// When explicitCommand is true, the command name is included in the invocation
// (for multi-completer subcommands). When false, the executable is invoked
// without an explicit command (standalone mode).
func SnippetSingle(command string, explicitCommand bool) string {
	functionName := strings.ReplaceAll(command, "-", "__")

	invocation := fmt.Sprintf("'%v', '_carapace', 'xonsh'", uid.Executable())
	if explicitCommand {
		invocation = fmt.Sprintf("'%v', '%v', '_carapace', 'xonsh'", uid.Executable(), command)
	}

	return fmt.Sprintf(`from xonsh.completers.completer import add_one_completer
from xonsh.completers.tools import contextual_command_completer

@contextual_command_completer
def _%v_completer(context):
    """carapace completer for %v"""
    if context.completing_command('%v'):
        from json import loads
        from xonsh.completers.tools import sub_proc_get_output, RichCompletion
        
        def fix_prefix(s):
            """quick fix for partially quoted prefix completion ('prefix',<TAB>)"""
            return s.translate(str.maketrans('', '', '\'"'))

        output, _ = sub_proc_get_output(
            %v, *[a.value for a in context.args], fix_prefix(context.prefix)
        )

        try:
            result = {RichCompletion(c["Value"], display=c["Display"], description=c["Description"], prefix_len=len(context.raw_prefix), append_closing_quote=False, style=c["Style"]) for c in loads(output)}
        except:
            result = {}
        if len(result) == 0:
            result = {RichCompletion(context.prefix, display=context.prefix, description='', prefix_len=len(context.raw_prefix), append_closing_quote=False)}
        return result

add_one_completer('%v', _%v_completer, 'start')
`, functionName, command, command, invocation, command, functionName)
}
