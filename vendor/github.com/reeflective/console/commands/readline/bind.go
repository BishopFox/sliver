package readline

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/reeflective/readline"
	"github.com/reeflective/readline/inputrc"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// Bind returns a command named `bind`, for manipulating readline keymaps and bindings.
func Bind(shell *readline.Shell) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bind",
		Short: "Display or modify readline key bindings",
		Long: `Manipulate readline keymaps and bindings.

Basic binding examples:
    bind "\C-x\C-r": re-read-init-file          // C-x C-r to reload the inputrc file, in the default keymap.
    bind -m vi-insert "\C-l" clear-screen       // C-l to clear-screen in vi-insert mode
    bind -m menu-complete '\C-n' menu-complete  // C-n to cycle through choices in the completion keymap.

Note that the keymap name is optional, and if omitted, the default keymap is used.
The default keymap is 'vi' only if 'set editing-mode vi' is found in inputrc , and 
unless the -m option is used to set a different keymap.
Also, note that the bind [seq] [command] slightly differs from the original bash 'bind' command.`,
	}

	// Flags
	cmd.Flags().StringP("keymap", "m", "", "Specify the keymap")
	cmd.Flags().BoolP("list", "l", false, "List names of functions")
	cmd.Flags().BoolP("binds", "P", false, "List function names and bindings")
	cmd.Flags().BoolP("binds-rc", "p", false, "List functions and bindings in a form that can be reused as input")
	cmd.Flags().BoolP("macros", "S", false, "List key sequences that invoke macros and their values")
	cmd.Flags().BoolP("macros-rc", "s", false, "List key sequences that invoke macros and their values in a form that can be reused as input")
	cmd.Flags().BoolP("vars", "V", false, "List variables names and values")
	cmd.Flags().BoolP("vars-rc", "v", false, "List variables names and values in a form that can be reused as input")
	cmd.Flags().StringP("query", "q", "", "Query about which keys invoke the named function")
	cmd.Flags().StringP("unbind", "u", "", "Unbind all keys which are bound to the named function")
	cmd.Flags().StringP("remove", "r", "", "Remove the bindings for KEYSEQ")
	cmd.Flags().StringP("file", "f", "", "Read key bindings from FILENAME")
	// cmd.Flags().StringP("execute", "x", "", "Cause SHELL-COMMAND to be executed whenever KEYSEQ is entered")
	// cmd.Flags().BoolP("execute-rc", "X", false, "List key sequences bound with -x and associated commands in a form that can be reused as input")

	// Run implementation
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Set keymap of interest
		keymap, _ := cmd.Flags().GetString("keymap")
		if keymap == "" {
			keymap = string(shell.Keymap.Main())
		}

		// Listing actions
		switch {
		// Function names
		case cmd.Flags().Changed("list"):
			for name := range shell.Keymap.Commands() {
				fmt.Println(name)
			}

			// Sequences to function names
		case cmd.Flags().Changed("binds"):
			shell.Keymap.PrintBinds(keymap, false)
			return nil

		case cmd.Flags().Changed("binds-rc"):
			shell.Keymap.PrintBinds(keymap, true)
			return nil

			// Macros
		case cmd.Flags().Changed("macros"):
			binds := shell.Config.Binds[keymap]
			if len(binds) == 0 {
				return nil
			}
			var macroBinds []string

			for keys, bind := range binds {
				if bind.Macro {
					macroBinds = append(macroBinds, inputrc.Escape(keys))
				}
			}

			sort.Strings(macroBinds)

			for _, key := range macroBinds {
				action := inputrc.Escape(binds[inputrc.Unescape(key)].Action)
				fmt.Printf("%s outputs %s\n", key, action)
			}

			return nil

		case cmd.Flags().Changed("macros-rc"):
			binds := shell.Config.Binds[keymap]
			if len(binds) == 0 {
				return nil
			}
			var macroBinds []string

			for keys, bind := range binds {
				if bind.Macro {
					macroBinds = append(macroBinds, inputrc.Escape(keys))
				}
			}

			sort.Strings(macroBinds)

			for _, key := range macroBinds {
				action := inputrc.Escape(binds[inputrc.Unescape(key)].Action)
				fmt.Printf("\"%s\": \"%s\"\n", key, action)
			}

			return nil

			// Global readline options
		case cmd.Flags().Changed("vars"):
			var variables []string

			for variable := range shell.Config.Vars {
				variables = append(variables, variable)
			}

			sort.Strings(variables)

			for _, variable := range variables {
				value := shell.Config.Vars[variable]
				fmt.Printf("%s is set to `%v'\n", variable, value)
			}

			return nil

		case cmd.Flags().Changed("vars-rc"):
			var variables []string

			for variable := range shell.Config.Vars {
				variables = append(variables, variable)
			}

			sort.Strings(variables)

			for _, variable := range variables {
				value := shell.Config.Vars[variable]
				fmt.Printf("set %s %v\n", variable, value)
			}

			return nil

			// Query binds for function
		case cmd.Flags().Changed("query"):
			binds := shell.Config.Binds[keymap]
			if binds == nil {
				return nil
			}

			command, _ := cmd.Flags().GetString("query")

			// Make a list of all sequences bound to each command.
			cmdBinds := make([]string, 0)

			for key, bind := range binds {
				if bind.Action != command {
					continue
				}

				cmdBinds = append(cmdBinds, inputrc.Escape(key))
			}

			sort.Strings(cmdBinds)

			switch {
			case len(cmdBinds) == 0:
			case len(cmdBinds) > 5:
				var firstBinds []string

				for i := 0; i < 5; i++ {
					firstBinds = append(firstBinds, "\""+cmdBinds[i]+"\"")
				}

				bindsStr := strings.Join(firstBinds, ", ")
				fmt.Printf("%s can be found on %s ...\n", command, bindsStr)

			default:
				var firstBinds []string

				for _, bind := range cmdBinds {
					firstBinds = append(firstBinds, "\""+bind+"\"")
				}

				bindsStr := strings.Join(firstBinds, ", ")
				fmt.Printf("%s can be found on %s\n", command, bindsStr)
			}

			return nil

			// case cmd.Flags().Changed("execute-rc"):
			// return nil
		}

		// Bind actions.
		// Some keymaps are aliases of others, so use either all equivalents or fallback to the relevant keymap.
		switch {
		case cmd.Flags().Changed("unbind"):
			command, _ := cmd.Flags().GetString("unbind")

			unbind := func(keymap string) {
				binds := shell.Config.Binds[keymap]
				if binds == nil {
					return
				}

				cmdBinds := make([]string, 0)

				for key, bind := range binds {
					if bind.Action != command {
						continue
					}

					cmdBinds = append(cmdBinds, key)
				}

				for _, key := range cmdBinds {
					delete(binds, key)
				}
			}

			applyToKeymap(keymap, unbind)

		case cmd.Flags().Changed("remove"):
			seq, _ := cmd.Flags().GetString("remove")

			removeBind := func(keymap string) {
				binds := shell.Config.Binds[keymap]
				if binds == nil {
					return
				}

				cmdBinds := make([]string, 0)

				for key := range binds {
					if key != seq {
						continue
					}

					cmdBinds = append(cmdBinds, key)
				}

				for _, key := range cmdBinds {
					delete(binds, key)
				}
			}

			applyToKeymap(keymap, removeBind)

		case cmd.Flags().Changed("file"):
			fileF, _ := cmd.Flags().GetString("file")

			file, err := os.Stat(fileF)
			if err != nil {
				return err
			}

			if err = inputrc.ParseFile(file.Name(), shell.Config, shell.Opts...); err != nil {
				return err
			}

			fmt.Printf("Read %s\n", file.Name())
			// case cmd.Flags().Changed("execute"):

			// Else if sufficient arguments, bind the key sequence to the command.
		default:
			if len(args) < 2 {
				return errors.New("Usage: bind [-m keymap] [keyseq] [command]")
			}

			// The key sequence is an escaped string, so unescape it.
			seq := inputrc.Unescape(args[0])

			var found bool

			for command := range shell.Keymap.Commands() {
				if command == args[1] {
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("Unknown command: %s", args[1])
			}

			// If the keymap doesn't exist, create it.
			if shell.Config.Binds[keymap] == nil {
				shell.Config.Binds[keymap] = make(map[string]inputrc.Bind)
			}

			// Adjust some keymaps (aliases of each other).
			bindkey := func(keymap string) {
				shell.Config.Binds[keymap][seq] = inputrc.Bind{Action: args[1]}
			}

			// (Bind the key sequence to the command.)
			applyToKeymap(keymap, bindkey)
		}

		return nil
	}

	// *** Completions ***
	comps := carapace.Gen(cmd)
	flagComps := make(carapace.ActionMap)

	// Flags
	flagComps["keymap"] = carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		results := make([]string, 0)

		for name := range shell.Config.Binds {
			results = append(results, name)
		}

		return carapace.ActionValues(results...).Tag("keymaps").Usage("keymap")
	})

	functionsComps := carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		results := make([]string, 0)

		for name := range shell.Keymap.Commands() {
			results = append(results, name)
		}

		return carapace.ActionValues(results...).Tag("commands").Usage("command")
	})

	bindSequenceComps := carapace.ActionCallback(func(ctx carapace.Context) carapace.Action {
		// Get the keymap.
		var keymap string

		if cmd.Flags().Changed("keymap") {
			keymap, _ = cmd.Flags().GetString("keymap")
		}

		if keymap == "" {
			keymap = string(shell.Keymap.Main())
		}

		// Get the binds.
		binds := shell.Config.Binds[keymap]
		if binds == nil {
			return carapace.ActionValues().Usage("sequence")
		}

		// Make a list of all sequences bound to each command, with descriptions.
		cmdBinds := make([]string, 0)
		insertBinds := make([]string, 0)

		for key, bind := range binds {
			if bind.Action == "self-insert" {
				insertBinds = append(insertBinds, "\""+inputrc.Escape(key)+"\"")
			} else {
				cmdBinds = append(cmdBinds, "\""+inputrc.Escape(key)+"\"")
				cmdBinds = append(cmdBinds, bind.Action)
			}
		}

		return carapace.Batch(
			carapace.ActionValues(insertBinds...).Tag(fmt.Sprintf("self-insert binds (%s)", keymap)).Usage("sequence"),
			carapace.ActionValuesDescribed(cmdBinds...).Tag(fmt.Sprintf("non-insert binds (%s)", keymap)).Usage("sequence"),
		).ToA()
	})

	flagComps["query"] = functionsComps
	flagComps["unbind"] = functionsComps
	flagComps["file"] = carapace.ActionFiles()
	flagComps["remove"] = bindSequenceComps

	comps.FlagCompletion(flagComps)

	// Positional arguments
	comps.PositionalCompletion(
		carapace.ActionValues().Usage("sequence"),
		functionsComps,
	)

	return cmd
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
