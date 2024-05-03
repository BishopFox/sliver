package readline

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"

	"github.com/reeflective/readline"
	"github.com/reeflective/readline/inputrc"
)

// Bind returns a command named `bind`, for manipulating readline keymaps and bindings.
func Bind(shell *readline.Shell) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bind",
		Short: "Display or modify readline key bindings",
		Long: `Manipulate readline keymaps and bindings.

Changing binds:
Note that the keymap name is optional, and if omitted, the default keymap is used.
The default keymap is 'vi' only if 'set editing-mode vi' is found in inputrc , and 
unless the -m option is used to set a different keymap.
Also, note that the bind [seq] [command] slightly differs from the original bash 'bind' command.

Exporting binds:
- Since all applications always look up to the same file for a given user,
  the export command does not allow to write and modify this file itself.
- Also, since saving the entire list of options and bindings in a different
  file for each application would also defeat the purpose of .inputrc.`,
		Example: `Changing binds:
    bind "\C-x\C-r": re-read-init-file          # C-x C-r to reload the inputrc file, in the default keymap.
    bind -m vi-insert "\C-l" clear-screen       # C-l to clear-screen in vi-insert mode
    bind -m menu-complete '\C-n' menu-complete  # C-n to cycle through choices in the completion keymap.

Exporting binds:
   bind --binds-rc --lib --changed # Only changed options/binds to stdout applying to all apps using this lib
   bind --app OtherApp -c          # Changed options, applying to an app other than our current shell one`,
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
	cmd.Flags().StringP("app", "A", "", "Optional application name (if empty/not used, the current app)")
	cmd.Flags().BoolP("changed", "c", false, "Only export options modified since app start: maybe not needed, since no use for it")
	cmd.Flags().BoolP("lib", "L", false, "Like 'app', but export options/binds for all apps using this specific library")
	cmd.Flags().BoolP("self-insert", "I", false, "If exporting bind sequences, also include the sequences mapped to self-insert")

	// Completions
	comps := carapace.Gen(cmd)
	flagComps := make(carapace.ActionMap)

	flagComps["keymap"] = completeKeymaps(shell, cmd)
	flagComps["query"] = completeCommands(shell, cmd)
	flagComps["unbind"] = completeCommands(shell, cmd)
	flagComps["remove"] = completeBindSequences(shell, cmd)
	flagComps["file"] = carapace.ActionFiles()

	comps.FlagCompletion(flagComps)

	comps.PositionalCompletion(
		carapace.ActionValues().Usage("key sequence"),
		completeCommands(shell, cmd),
	)

	// Run implementation
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Set keymap of interest
		keymap, _ := cmd.Flags().GetString("keymap")
		if keymap == "" {
			keymap = string(shell.Keymap.Main())
		}

		var name string
		reeflective := "reeflective"
		buf := &cfgBuilder{buf: &strings.Builder{}}

		// First prepare the branching strings for any
		// needed conditionals (App, Lib, keymap, etc)
		changed := cmd.Flags().Changed("changed")
		rm := cmd.Flags().Changed("remove")
		unbind := cmd.Flags().Changed("unbind")
		app := cmd.Flags().Changed("app")
		lib := cmd.Flags().Changed("lib")

		// 1 - SIMPLE QUERIES ------------------------------------------------

		// All flags and args that are "exiting the command
		// after run" are listed and evaluated first.

		// Function names
		if cmd.Flags().Changed("list") {
			for name := range shell.Keymap.Commands() {
				fmt.Println(name)
			}

			return nil
		}

		// 2 - Query binds for function
		if cmd.Flags().Changed("query") {
			listBinds(shell, buf, cmd, keymap)
			fmt.Fprint(cmd.OutOrStdout(), buf.buf.String())
			return nil
		}

		// From this point on, some flags don't exit after printing
		// their respective listings, since we can combine and output
		// various types of stuff at once, for configs or display.
		//
		// We can even read a file for binds, remove some of them,
		// and display all or specific sections of our config in
		// a single call, with multiple flags of all sorts.

		// 1 - Apply any changes we want from a file first.
		if cmd.Flags().Changed("file") {
			if err := readFileConfig(shell, cmd, keymap); err != nil {
				return err
			}
		}

		// Remove anything we might have been asked to.
		if unbind {
			unbindKeys(shell, cmd, keymap)
		}

		if rm {
			removeCommands(shell, cmd, keymap)
		}

		// 2 - COMPLEX QUERIES ------------------------------------------------

		// Write App/Lib headers for
		if app {
			fmt.Fprintf(buf, "# %s application (generated)\n", name)
			buf.newCond(name)
		} else if lib {
			fmt.Fprintf(buf, "# %s/readline library-specific (generated)\n", reeflective)
			buf.newCond(reeflective)
		}

		// Global option variables
		if cmd.Flags().Changed("vars") {
			listVars(shell, buf, cmd)
		} else if cmd.Flags().Changed("vars-rc") {
			listVarsRC(shell, buf, cmd)
		}

		// Sequences to function names
		if cmd.Flags().Changed("binds") {
			listBinds(shell, buf, cmd, keymap)
		} else if cmd.Flags().Changed("binds-rc") {
			listBindsRC(shell, buf, cmd, keymap)
		}

		// Macros
		if cmd.Flags().Changed("macros") {
			listMacros(shell, buf, cmd, keymap)
		} else if cmd.Flags().Changed("macros-rc") {
			listMacrosRC(shell, buf, cmd, keymap)
		}

		// Close any App/Lib conditional
		buf.endCond()

		// The command has performed an action, so any binding
		// with positional arguments is not considered or evaluated.
		if buf.buf.Len() > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), buf.buf.String())
			return nil
		} else if app || lib || changed || rm || unbind {
			return nil
		}

		// 3 - CREATE NEw BINDS -----------------------------------------------

		// Bind actions.
		// Some keymaps are aliases of others, so use either
		// all equivalents or fallback to the relevant keymap.
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

		// (Bind the key sequence to the command)
		applyToKeymap(keymap, bindkey)

		return nil
	}

	return cmd
}

//
// Binding & Unbinding functions ---------------------------------
//

func readFileConfig(sh *readline.Shell, cmd *cobra.Command, _ string) error {
	fileF, _ := cmd.Flags().GetString("file")

	file, err := os.Stat(fileF)
	if err != nil {
		return err
	}

	if err = inputrc.ParseFile(file.Name(), sh.Config, sh.Opts...); err != nil {
		return err
	}

	fmt.Printf("Read and parsed %s\n", file.Name())

	return nil
}

func unbindKeys(sh *readline.Shell, cmd *cobra.Command, keymap string) {
	command, _ := cmd.Flags().GetString("unbind")

	unbind := func(keymap string) {
		binds := sh.Config.Binds[keymap]
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
}

func removeCommands(sh *readline.Shell, cmd *cobra.Command, keymap string) {
	seq, _ := cmd.Flags().GetString("remove")

	removeBind := func(keymap string) {
		binds := sh.Config.Binds[keymap]
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
}
