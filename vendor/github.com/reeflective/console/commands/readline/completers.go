package readline

/*
   console - Closed-loop console application for cobra commands
   Copyright (C) 2023 Reeflective

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"fmt"

	"github.com/carapace-sh/carapace"
	"github.com/spf13/cobra"

	"github.com/reeflective/readline"
	"github.com/reeflective/readline/inputrc"
)

func completeKeymaps(sh *readline.Shell, _ *cobra.Command) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		results := make([]string, 0)

		for name := range sh.Config.Binds {
			results = append(results, name)
		}

		return carapace.ActionValues(results...).Tag("keymaps").Usage("keymap")
	})
}

func completeBindSequences(sh *readline.Shell, cmd *cobra.Command) carapace.Action {
	return carapace.ActionCallback(func(ctx carapace.Context) carapace.Action {
		// Get the keymap.
		var keymap string

		if cmd.Flags().Changed("keymap") {
			keymap, _ = cmd.Flags().GetString("keymap")
		}

		if keymap == "" {
			keymap = string(sh.Keymap.Main())
		}

		// Get the binds.
		binds := sh.Config.Binds[keymap]
		if binds == nil {
			return carapace.ActionValues().Usage("sequence")
		}

		// Make a list of all sequences bound to each command, with descriptions.
		var cmdBinds, insertBinds []string

		for key, bind := range binds {
			val := inputrc.Escape(key)

			if bind.Action == "self-insert" {
				insertBinds = append(insertBinds, val)
			} else {
				cmdBinds = append(cmdBinds, val)
				cmdBinds = append(cmdBinds, bind.Action)
			}
		}

		// Build the list of bind sequences bompletions
		completions := carapace.Batch(
			carapace.ActionValues(insertBinds...).Tag(fmt.Sprintf("self-insert binds (%s)", keymap)).Usage("sequence"),
			carapace.ActionValuesDescribed(cmdBinds...).Tag(fmt.Sprintf("non-insert binds (%s)", keymap)).Usage("sequence"),
		).ToA().Suffix("\"")

		// We're lucky and be particularly cautious about completion here:
		// Look for the current argument and check whether or not it's quoted.
		// If yes, only include quotes at the end of the inserted value.
		// If no quotes, include them in both.
		if ctx.Value == "" {
			completions = completions.Prefix("\"")
		}

		return completions
	})
}

func completeCommands(sh *readline.Shell, _ *cobra.Command) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		results := make([]string, 0)

		for name := range sh.Keymap.Commands() {
			results = append(results, name)
		}

		return carapace.ActionValues(results...).Tag("commands").Usage("command")
	})
}

func applyToKeymap(keymap string, bind func(keymap string)) {
	switch keymap {
	case "emacs", "emacs-standard":
		for _, km := range []string{"emacs", "emacs-standard"} {
			bind(km)
		}
	case "emacs-ctlx":
		for _, km := range []string{"emacs-ctlx", "emacs-standard", "emacs"} {
			bind(km)
		}
	case "emacs-meta":
		for _, km := range []string{"emacs-meta", "emacs-standard", "emacs"} {
			bind(km)
		}
	case "vi", "vi-move", "vi-command":
		for _, km := range []string{"vi", "vi-move", "vi-command"} {
			bind(km)
		}
	default:
		bind(keymap)
	}
}
