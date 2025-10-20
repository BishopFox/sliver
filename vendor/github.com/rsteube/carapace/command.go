package carapace

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rsteube/carapace/pkg/style"
	"github.com/spf13/cobra"
)

func addCompletionCommand(cmd *cobra.Command) {
	for _, c := range cmd.Commands() {
		if c.Name() == "_carapace" {
			return
		}
	}

	carapaceCmd := &cobra.Command{
		Use:    "_carapace",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			LOG.Print(strings.Repeat("-", 80))
			LOG.Printf("%#v", os.Args)

			if len(args) > 2 && strings.HasPrefix(args[2], "_") {
				cmd.Hidden = false
			}

			if !cmd.HasParent() {
				panic("missing parent command") // this should never happen
			}

			parentCmd := cmd.Parent()
			if parentCmd.Annotations[annotation_standalone] == "true" {
				// TODO how to handle an explicit `_carapace` command?
				parentCmd.RemoveCommand(cmd) // don't complete local `_carapace` in standalone mode
			}

			if s, err := complete(parentCmd, args); err != nil {
				fmt.Fprintln(io.MultiWriter(parentCmd.OutOrStderr(), LOG.Writer()), err.Error())
			} else {
				fmt.Fprintln(io.MultiWriter(parentCmd.OutOrStdout(), LOG.Writer()), s)
			}
		},
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
		DisableFlagParsing: true,
	}

	cmd.AddCommand(carapaceCmd)

	Carapace{carapaceCmd}.PositionalCompletion(
		ActionStyledValues(
			"bash", "#d35673",
			"bash-ble", "#c2039a",
			"elvish", "#ffd6c9",
			"export", style.Default,
			"fish", "#7ea8fc",
			"ion", "#0e5d6d",
			"nushell", "#29d866",
			"oil", "#373a36",
			"powershell", "#e8a16f",
			"spec", style.Default,
			"tcsh", "#412f09",
			"xonsh", "#a8ffa9",
			"zsh", "#efda53",
		),
		ActionValues(cmd.Root().Name()),
	)
	Carapace{carapaceCmd}.PositionalAnyCompletion(
		ActionCallback(func(c Context) Action {
			cmd.RemoveCommand(carapaceCmd)
			action, _ := traverse(cmd, append(c.Args[2:], c.Value))
			return action
		}),
	)

	styleCmd := &cobra.Command{
		Use:  "style",
		Args: cobra.ExactArgs(1),
		Run:  func(cmd *cobra.Command, args []string) {},
	}
	carapaceCmd.AddCommand(styleCmd)

	styleSetCmd := &cobra.Command{
		Use:  "set",
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			for _, arg := range args {
				if splitted := strings.SplitN(arg, "=", 2); len(splitted) == 2 {
					if err := style.Set(splitted[0], splitted[1]); err != nil {
						fmt.Fprint(cmd.ErrOrStderr(), err.Error())
					}
				} else {
					fmt.Fprintf(cmd.ErrOrStderr(), "invalid format: '%v'", arg)
				}
			}
		},
	}
	styleCmd.AddCommand(styleSetCmd)
	Carapace{styleSetCmd}.PositionalAnyCompletion(
		ActionStyleConfig(),
	)
}
