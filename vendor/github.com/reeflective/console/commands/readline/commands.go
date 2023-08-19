package readline

import (
	"github.com/spf13/cobra"

	"github.com/reeflective/readline"
)

// Commands returns a command named `readline`, with subcommands dedicated
// to setting up readline keybindings, keymaps, and global options. It is
// intended to be used as a subcommand of the root command.
// You can freely change the use name of this command, or any of its properties.
func Commands(shell *readline.Shell) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "readline",
		Short: "Manipulate readline options, keymaps and bindings",
		Long:  `Manipulate readline options, keymaps and bindings.`,
	}

	// Subcommands
	cmd.AddCommand(Set(shell))
	cmd.AddCommand(Bind(shell))

	return cmd
}
