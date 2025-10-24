package readline

import (
	"errors"
	"strconv"
	"strings"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace/pkg/style"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"github.com/reeflective/readline"
	"github.com/reeflective/readline/inputrc"
)

// We here must assume that all bind changes during the lifetime
// of the binary are all made by a single readline application.
// This config only stores the vars/binds that have been changed.
var cfgChanged = inputrc.NewConfig()

// Set returns a command named `set`, for manipulating readline global options.
func Set(shell *readline.Shell) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Manipulate readline global options",
		Long:  `Manipulate readline global options.`,
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// First argument is the key.
			// It must be one of the options available in the shell.
			option := shell.Config.Get(args[0])
			if option == nil {
				return errors.New("Unknown readline option: " + args[0])
			}

			// Second argument is the value.
			var value interface{}
			var err error

			// It must be a valid value for the option.
			switch option.(type) {
			case bool:
				if args[1] != "on" && args[1] != "off" && args[1] != "true" && args[1] != "false" {
					return errors.New("Invalid value for boolean option " + args[0] + ": " + args[1])
				}
				value = args[1] == "on" || args[1] == "true"

			case int:
				value, err = strconv.Atoi(args[1])
				if err != nil {
					return errors.New("Invalid value for integer option " + args[0] + ": " + args[1])
				}
			}

			// Set the option.
			if err = shell.Config.Set(args[0], value); err != nil {
				return err
			}

			return cfgChanged.Set(args[0], value)
		},
	}

	// Completions
	varComps := func(_ carapace.Context) carapace.Action {
		results := make([]string, 0)

		for varName := range shell.Config.Vars {
			results = append(results, varName)
		}

		return carapace.ActionValues(results...).Tag("global options").Usage("option name")
	}

	argComp := func(c carapace.Context) carapace.Action {
		val := strings.TrimSpace(c.Args[len(c.Args)-1])

		option := shell.Config.Get(val)
		if option == nil {
			return carapace.ActionMessage("No var named %v", option)
		}

		switch val {
		case "cursor-style":
			return carapace.ActionValues("block", "beam", "underline", "blinking-block", "blinking-underline", "blinking-beam", "default")
		case "editing-mode":
			return carapace.ActionValues("vi", "emacs")
		case "keymap":
			return completeKeymaps(shell, cmd)
		}

		switch option.(type) {
		case bool:
			return carapace.ActionValues("on", "off", "true", "false").StyleF(style.ForKeyword)
		case int:
			return carapace.ActionValues().Usage("option value (int)")
		case string:
			return carapace.ActionValues().Usage("option value (string)")
		}

		return carapace.ActionValues().Usage("option value")
	}

	carapace.Gen(cmd).PositionalCompletion(
		carapace.ActionCallback(varComps),
		carapace.ActionCallback(argComp),
	)

	return cmd
}

// Returns the subset of inputrc variables that are specific
// to this library and application/binary.
func filterAppLibVars(cfgVars map[string]interface{}) map[string]interface{} {
	appVars := make(map[string]interface{})

	defCfg := inputrc.DefaultVars()
	defVars := maps.Keys(defCfg)

	for name, val := range cfgVars {
		if slices.Contains(defVars, name) {
			continue
		}

		appVars[name] = val
	}

	return appVars
}

// Returns the subset of inputrc variables that are specific
// to this library and application/binary.
func filterLegacyVars(cfgVars map[string]interface{}) map[string]interface{} {
	appVars := make(map[string]interface{})

	defCfg := inputrc.DefaultVars()
	defVars := maps.Keys(defCfg)

	for name, val := range cfgVars {
		if !slices.Contains(defVars, name) {
			continue
		}

		appVars[name] = val
	}

	return appVars
}

// Filters out all configuration variables that have not been changed.
func filterChangedVars(allVars map[string]interface{}) map[string]interface{} {
	if allVars == nil {
		return cfgChanged.Vars
	}

	appVars := make(map[string]interface{})
	defVars := maps.Keys(appVars)

	for name, val := range allVars {
		if slices.Contains(defVars, name) {
			appVars[name] = val
		}
	}

	return appVars
}
