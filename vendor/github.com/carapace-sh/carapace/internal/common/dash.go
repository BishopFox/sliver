package common

import "github.com/spf13/cobra"

// IsDash checks if command contains a dash disabling flag parsing
//
//	example action positional1 -- dash1 dash2
func IsDash(cmd *cobra.Command) bool {
	return cmd.ArgsLenAtDash() != -1
}
