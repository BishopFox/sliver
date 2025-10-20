package shell

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rsteube/carapace/internal/common"
	"github.com/rsteube/carapace/internal/env"
	"github.com/rsteube/carapace/internal/shell/bash"
	"github.com/rsteube/carapace/internal/shell/bash_ble"
	"github.com/rsteube/carapace/internal/shell/elvish"
	"github.com/rsteube/carapace/internal/shell/export"
	"github.com/rsteube/carapace/internal/shell/fish"
	"github.com/rsteube/carapace/internal/shell/ion"
	"github.com/rsteube/carapace/internal/shell/nushell"
	"github.com/rsteube/carapace/internal/shell/oil"
	"github.com/rsteube/carapace/internal/shell/powershell"
	"github.com/rsteube/carapace/internal/shell/spec"
	"github.com/rsteube/carapace/internal/shell/tcsh"
	"github.com/rsteube/carapace/internal/shell/xonsh"
	"github.com/rsteube/carapace/internal/shell/zsh"
	"github.com/rsteube/carapace/pkg/ps"
	"github.com/rsteube/carapace/pkg/style"
	"github.com/spf13/cobra"
)

// Snippet creates completion script for given shell.
func Snippet(cmd *cobra.Command, shell string) (string, error) {
	if shell == "" {
		shell = ps.DetermineShell()
	}
	shellSnippets := map[string]func(cmd *cobra.Command) string{
		"bash":       bash.Snippet,
		"bash-ble":   bash_ble.Snippet,
		"export":     export.Snippet,
		"fish":       fish.Snippet,
		"elvish":     elvish.Snippet,
		"ion":        ion.Snippet,
		"nushell":    nushell.Snippet,
		"oil":        oil.Snippet,
		"powershell": powershell.Snippet,
		"spec":       spec.Snippet,
		"tcsh":       tcsh.Snippet,
		"xonsh":      xonsh.Snippet,
		"zsh":        zsh.Snippet,
	}
	if s, ok := shellSnippets[shell]; ok {
		return s(cmd.Root()), nil
	}

	expected := make([]string, 0)
	for key := range shellSnippets {
		expected = append(expected, key)
	}
	sort.Strings(expected)
	return "", fmt.Errorf("expected one of '%v' [was: %v]", strings.Join(expected, "', '"), shell)
}

func Value(shell string, value string, meta common.Meta, values common.RawValues) string { // TODO use context instead?
	shellFuncs := map[string]func(currentWord string, meta common.Meta, values common.RawValues) string{
		"bash":       bash.ActionRawValues,
		"bash-ble":   bash_ble.ActionRawValues,
		"fish":       fish.ActionRawValues,
		"elvish":     elvish.ActionRawValues,
		"export":     export.ActionRawValues,
		"ion":        ion.ActionRawValues,
		"nushell":    nushell.ActionRawValues,
		"oil":        oil.ActionRawValues,
		"powershell": powershell.ActionRawValues,
		"tcsh":       tcsh.ActionRawValues,
		"xonsh":      xonsh.ActionRawValues,
		"zsh":        zsh.ActionRawValues,
	}
	if f, ok := shellFuncs[shell]; ok {
		if env.ColorDisabled() {
			style.Carapace.Value = style.Default
			style.Carapace.Description = style.Default
			style.Carapace.Error = style.Underlined
			style.Carapace.Usage = style.Italic
			values = values.Decolor()
		}
		filtered := values.FilterPrefix(value)
		switch shell {
		case "elvish", "export", "zsh": // shells with support for showing messages
		default:
			filtered = meta.Messages.Integrate(filtered, value)
		}

		if !meta.Messages.IsEmpty() && shell != "export" {
			meta.Nospace.Add('*')
		}

		sort.Sort(common.ByDisplay(filtered))
		return f(value, meta, filtered)
	}
	return ""
}
