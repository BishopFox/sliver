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
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/reeflective/readline"
	"github.com/reeflective/readline/inputrc"
)

const (
	printOn  = "on"
	printOff = "off"
)

// listVars prints the readline global option variables in human-readable format.
func listVars(shell *readline.Shell, buf *cfgBuilder, cmd *cobra.Command) {
	var vars map[string]interface{}

	// Apply other filters to our current list of vars.
	if cmd.Flags().Changed("changed") {
		vars = cfgChanged.Vars
	} else {
		vars = shell.Config.Vars
	}

	if len(vars) == 0 {
		return
	}

	variables := make([]string, len(shell.Config.Vars))

	for variable := range shell.Config.Vars {
		variables = append(variables, variable)
	}

	sort.Strings(variables)

	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "======= Global Variables =========")
	fmt.Fprintln(buf)

	for _, variable := range variables {
		value := shell.Config.Vars[variable]
		if value == nil || variable == "" {
			continue
		}

		fmt.Fprintf(buf, "%s is set to `%v'\n", variable, value)
	}
}

// listVarsRC returns the readline global options, split according to which are
// supported by which library, and output in .inputrc compliant format.
func listVarsRC(shell *readline.Shell, buf *cfgBuilder, cmd *cobra.Command) {
	var vars map[string]interface{}

	// Apply other filters to our current list of vars.
	if cmd.Flags().Changed("changed") {
		vars = cfgChanged.Vars
	} else {
		vars = shell.Config.Vars
	}

	if len(vars) == 0 {
		return
	}

	// Include print all legacy options.
	// Filter them in a separate groups only if NOT being used with --app/--lib
	if !cmd.Flags().Changed("app") && !cmd.Flags().Changed("lib") {
		var legacy []string
		for variable := range filterLegacyVars(vars) {
			legacy = append(legacy, variable)
		}

		sort.Strings(legacy)

		fmt.Fprintln(buf, "# General/legacy Options (generated from reeflective/readline)")

		for _, variable := range legacy {
			value := shell.Config.Vars[variable]
			var printVal string

			if on, ok := value.(bool); ok {
				if on {
					printVal = "on"
				} else {
					printVal = "off"
				}
			} else {
				printVal = fmt.Sprintf("%v", value)
			}

			fmt.Fprintf(buf, "set %s %s\n", variable, printVal)
		}

		// Now we print the App/lib specific.
		var reef []string

		for variable := range filterAppLibVars(vars) {
			reef = append(reef, variable)
		}

		sort.Strings(reef)

		fmt.Fprintln(buf)
		fmt.Fprintln(buf, "# reeflective/readline specific options (generated)")
		fmt.Fprintln(buf, "# The following block is not implemented in GNU C Readline.")
		buf.newCond("reeflective")

		for _, variable := range reef {
			value := shell.Config.Vars[variable]
			var printVal string

			if on, ok := value.(bool); ok {
				if on {
					printVal = printOn
				} else {
					printVal = printOff
				}
			} else {
				printVal = fmt.Sprintf("%v", value)
			}

			fmt.Fprintf(buf, "set %s %s\n", variable, printVal)
		}

		buf.endCond()

		return
	}

	fmt.Fprintln(buf, "# General options (legacy and reeflective)")

	var all []string
	for variable := range vars {
		all = append(all, variable)
	}
	sort.Strings(all)

	for _, variable := range all {
		value := shell.Config.Vars[variable]
		var printVal string

		if on, ok := value.(bool); ok {
			if on {
				printVal = "on"
			} else {
				printVal = "off"
			}
		} else {
			printVal = fmt.Sprintf("%v", value)
		}

		fmt.Fprintf(buf, "set %s %s\n", variable, printVal)
	}
}

// listBinds prints the bind sequences for a given keymap,
// according to command filter flags, in human-readable format.
func listBinds(shell *readline.Shell, buf *cfgBuilder, cmd *cobra.Command, keymap string) {
	var binds map[string]inputrc.Bind

	// Apply other filters to our current list of vars.
	if cmd.Flags().Changed("changed") {
		binds = cfgChanged.Binds[keymap]
	} else {
		binds = shell.Config.Binds[keymap]
	}

	// Get all the commands, used to sort the displays.
	commands := make([]string, len(shell.Keymap.Commands()))
	for command := range shell.Keymap.Commands() {
		commands = append(commands, command)
	}

	sort.Strings(commands)

	query, _ := cmd.Flags().GetString("query")
	mustMatchQuery := query != "" && cmd.Flags().Changed("query")

	// Make a list of all sequences bound to each command.
	allBinds := make(map[string][]string)

	for _, command := range commands {
		for key, bind := range binds {
			if bind.Action != command {
				continue
			}

			// If we are querying a specific command
			if bind.Action != query && mustMatchQuery {
				continue
			}

			commandBinds := allBinds[command]
			commandBinds = append(commandBinds, inputrc.Escape(key))
			allBinds[command] = commandBinds
		}
	}

	if len(commands) == 0 {
		return
	}

	fmt.Fprintln(buf)
	fmt.Fprintf(buf, "===== Command Binds (%s) =======\n", keymap)
	fmt.Fprintln(buf)

	for _, command := range commands {
		commandBinds := allBinds[command]
		sort.Strings(commandBinds)

		switch {
		case len(commandBinds) == 0:
		case len(commandBinds) > 5:
			var firstBinds []string

			for i := 0; i < 5; i++ {
				firstBinds = append(firstBinds, "\""+commandBinds[i]+"\"")
			}

			bindsStr := strings.Join(firstBinds, ", ")
			fmt.Fprintf(buf, "%s can be found on %s ...\n", command, bindsStr)

		default:
			var firstBinds []string

			for _, bind := range commandBinds {
				firstBinds = append(firstBinds, "\""+bind+"\"")
			}

			bindsStr := strings.Join(firstBinds, ", ")
			fmt.Fprintf(buf, "%s can be found on %s\n", command, bindsStr)
		}
	}
}

// listBindsRC prints the bind sequences for a given keymap,
// according to command filter flags, in .inputrc compliant format.
func listBindsRC(shell *readline.Shell, buf *cfgBuilder, cmd *cobra.Command, keymap string) {
	var binds map[string]inputrc.Bind
	selfInsert, _ := cmd.Flags().GetBool("self-insert")

	// Apply other filters to our current list of vars.
	if cmd.Flags().Changed("changed") {
		binds = cfgChanged.Binds[keymap]
	} else {
		binds = shell.Config.Binds[keymap]
	}

	if len(binds) == 0 {
		return
	}

	// Get all the commands, used to sort the displays.
	commands := make([]string, len(shell.Keymap.Commands()))
	for command := range shell.Keymap.Commands() {
		commands = append(commands, command)
	}

	sort.Strings(commands)

	// Make a list of all sequences bound to each command.
	allBinds := make(map[string][]string)

	for _, command := range commands {
		for key, bind := range binds {
			if bind.Action != command {
				continue
			}

			commandBinds := allBinds[command]
			commandBinds = append(commandBinds, inputrc.Escape(key))
			allBinds[command] = commandBinds
		}
	}

	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "# Command binds (generated from reeflective/readline)")
	fmt.Fprintf(buf, "set keymap %s\n\n", keymap)

	for _, command := range commands {
		commandBinds := allBinds[command]
		sort.Strings(commandBinds)

		if command == "self-insert" && !selfInsert {
			continue
		}

		if len(commandBinds) > 0 {
			for _, bind := range commandBinds {
				fmt.Fprintf(buf, "\"%s\": %s\n", bind, command)
			}
		}
	}
}

// listMacros prints the recorded/existing macros for a given keymap, in human-readable format.
func listMacros(shell *readline.Shell, buf *cfgBuilder, cmd *cobra.Command, keymap string) {
	var binds map[string]inputrc.Bind

	// Apply other filters to our current list of vars.
	if cmd.Flags().Changed("changed") {
		binds = cfgChanged.Binds[keymap]
	} else {
		binds = shell.Config.Binds[keymap]
	}

	var macroBinds []string

	for keys, bind := range binds {
		if bind.Macro {
			macroBinds = append(macroBinds, inputrc.Escape(keys))
		}
	}

	if len(macroBinds) == 0 {
		return
	}

	sort.Strings(macroBinds)

	fmt.Fprintln(buf)
	fmt.Fprintf(buf, "====== Macros (%s) ======\n", keymap)
	fmt.Fprintln(buf)

	for _, key := range macroBinds {
		action := inputrc.Escape(binds[inputrc.Unescape(key)].Action)
		fmt.Printf("%s outputs %s\n", key, action)
	}
}

// listMacros prints the recorded/existing macros for a given keymap, in .inputrc compliant format.
func listMacrosRC(shell *readline.Shell, buf *cfgBuilder, cmd *cobra.Command, keymap string) {
	var binds map[string]inputrc.Bind

	// Apply other filters to our current list of vars.
	if cmd.Flags().Changed("changed") {
		binds = cfgChanged.Binds[keymap]
	} else {
		binds = shell.Config.Binds[keymap]
	}

	var macroBinds []string

	for keys, bind := range binds {
		if bind.Macro {
			macroBinds = append(macroBinds, inputrc.Escape(keys))
		}
	}

	if len(macroBinds) == 0 {
		return
	}

	sort.Strings(macroBinds)

	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "# Macro binds (generated from reeflective/readline)")
	fmt.Fprintf(buf, "set keymap %s\n\n", keymap)

	for _, key := range macroBinds {
		action := inputrc.Escape(binds[inputrc.Unescape(key)].Action)
		fmt.Fprintf(buf, "\"%s\": \"%s\"\n", key, action)
	}
}
