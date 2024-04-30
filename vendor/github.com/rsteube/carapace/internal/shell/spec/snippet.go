// Package spec provides spec file generation for use with carapace-bin
package spec

import (
	"github.com/rsteube/carapace/internal/pflagfork"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// Snippet generates the spec file.
func Snippet(cmd *cobra.Command) string {
	m, _ := yaml.Marshal(command(cmd))
	return string(m)
}

func command(cmd *cobra.Command) Command {
	c := Command{
		Name:            cmd.Use,
		Description:     cmd.Short,
		Aliases:         cmd.Aliases,
		Group:           cmd.GroupID,
		Hidden:          cmd.Hidden,
		Flags:           make(map[string]string),
		PersistentFlags: make(map[string]string),
		Commands:        make([]Command, 0),
	}

	// TODO mutually exclusive flags

	cmd.LocalFlags().VisitAll(func(flag *pflag.Flag) {
		if cmd.PersistentFlags().Lookup(flag.Name) != nil {
			return
		}

		f := pflagfork.Flag{Flag: flag}
		c.Flags[f.Definition()] = f.Usage

	})

	cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		f := pflagfork.Flag{Flag: flag}
		c.PersistentFlags[f.Definition()] = f.Usage

	})

	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() != "_carapace" && subcmd.Deprecated == "" {
			c.Commands = append(c.Commands, command(subcmd))
		}
	}

	return c
}
