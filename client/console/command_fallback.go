package console

import (
	"strings"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
)

func (con *SliverClient) allowServerRootCommands(args []string) ([]string, error) {
	con.applyCommandFallback(args)
	return args, nil
}

func (con *SliverClient) prepareCompletion(line []rune, cursor int) {
	if con == nil || con.App == nil {
		return
	}

	menu := con.App.ActiveMenu()
	if menu == nil {
		return
	}

	con.setMenuCommandForCompletion(menu, line, cursor)
}

func (con *SliverClient) applyCommandFallback(args []string) {
	if con == nil || con.App == nil {
		return
	}

	con.setMenuCommand(con.App.ActiveMenu(), args)
}

func (con *SliverClient) setMenuCommand(menu *console.Menu, args []string) {
	if menu == nil {
		return
	}

	if menu.Name() != consts.ImplantMenu {
		if menu.Command == nil && con.serverCmds != nil {
			menu.Command = con.serverCmds()
		}
		return
	}

	if menu.Command == nil {
		if con.sliverCmds == nil {
			return
		}
		menu.Command = con.sliverCmds()
	}

	if con.serverCmds == nil {
		return
	}

	name := firstNonFlagArg(args)
	if name == "" {
		return
	}

	if rootHasCommand(menu.Command, name) {
		return
	}

	serverRoot := con.serverCmds()
	if serverRoot == nil || !rootHasCommand(serverRoot, name) {
		return
	}

	menu.Command = serverRoot
}

func (con *SliverClient) setMenuCommandForCompletion(menu *console.Menu, line []rune, cursor int) {
	if menu == nil || menu.Name() != consts.ImplantMenu {
		return
	}

	if menu.Command == nil {
		if con.sliverCmds == nil {
			return
		}
		menu.Command = con.sliverCmds()
	}

	if con.serverCmds == nil {
		return
	}

	token := firstTokenFromLine(line, cursor)
	if token == "" {
		return
	}

	if rootHasCommandPrefix(menu.Command, token) {
		return
	}

	serverRoot := con.serverCmds()
	if serverRoot == nil || !rootHasCommandPrefix(serverRoot, token) {
		return
	}

	menu.Command = serverRoot
	hideFilteredCommands(menu, menu.Command)
}

func firstNonFlagArg(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return arg
	}
	return ""
}

func firstTokenFromLine(line []rune, cursor int) string {
	if cursor > len(line) {
		cursor = len(line)
	}
	if cursor < 0 {
		cursor = 0
	}

	pos := 0
	for pos < cursor && isSpace(line[pos]) {
		pos++
	}
	start := pos
	for pos < cursor && !isSpace(line[pos]) {
		pos++
	}
	if start == pos {
		return ""
	}
	return string(line[start:pos])
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

func rootHasCommand(root *cobra.Command, name string) bool {
	if root == nil || name == "" {
		return false
	}
	for _, cmd := range root.Commands() {
		if cmd.Name() == name || cmd.HasAlias(name) {
			return true
		}
	}
	return false
}

func rootHasCommandPrefix(root *cobra.Command, prefix string) bool {
	if root == nil || prefix == "" {
		return false
	}
	for _, cmd := range root.Commands() {
		if strings.HasPrefix(cmd.Name(), prefix) {
			return true
		}
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(alias, prefix) {
				return true
			}
		}
	}
	return false
}

func hideFilteredCommands(menu *console.Menu, root *cobra.Command) {
	if menu == nil || root == nil {
		return
	}
	for _, cmd := range root.Commands() {
		if cmd.Hidden {
			continue
		}
		if filters := menu.ActiveFiltersFor(cmd); len(filters) > 0 {
			cmd.Hidden = true
		}
	}
}
