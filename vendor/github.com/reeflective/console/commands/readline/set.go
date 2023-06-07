package readline

import (
	"errors"
	"strconv"

	"github.com/reeflective/readline"
	"github.com/rsteube/carapace"
	"github.com/rsteube/carapace/pkg/style"
	"github.com/spf13/cobra"
)

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
			return shell.Config.Set(args[0], value)
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
		val := c.Args[len(c.Args)-1]

		option := shell.Config.Get(val)
		if option == nil {
			return carapace.ActionValues()
		}

		switch option.(type) {
		case bool:
			return carapace.ActionValues("on", "off", "true", "false").StyleF(style.ForKeyword)
		case int:
			return carapace.ActionValues().Usage("option value (int)")
		default:
			carapace.ActionValues().Usage("option value (string)")
		}

		return carapace.ActionValues().Usage("option value")
	}

	carapace.Gen(cmd).PositionalCompletion(
		carapace.ActionCallback(varComps),
		carapace.ActionCallback(argComp),
	)

	return cmd
}
