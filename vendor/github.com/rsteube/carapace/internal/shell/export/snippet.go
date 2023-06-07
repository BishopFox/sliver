// Package export provides command structure export
package export

import (
	"encoding/json"

	"github.com/rsteube/carapace/internal/pflagfork"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type command struct {
	Name            string
	Short           string
	Long            string    `json:",omitempty"`
	Aliases         []string  `json:",omitempty"`
	Commands        []command `json:",omitempty"`
	LocalFlags      []flag    `json:",omitempty"`
	PersistentFlags []flag    `json:",omitempty"`
}

type flag struct {
	Longhand    string `json:",omitempty"`
	Shorthand   string `json:",omitempty"`
	Usage       string
	Type        string
	NoOptDefVal string `json:",omitempty"`
}

func convertFlag(f *pflag.Flag) flag {
	longhand := ""
	if (pflagfork.Flag{Flag: f}).Mode() != pflagfork.ShorthandOnly {
		longhand = f.Name
	}

	noOptDefVal := ""
	if f.Value.Type() != "bool" {
		noOptDefVal = f.NoOptDefVal
	}
	return flag{
		Longhand:    longhand,
		Shorthand:   f.Shorthand,
		Usage:       f.Usage,
		Type:        f.Value.Type(),
		NoOptDefVal: noOptDefVal,
	}
}

func convert(cmd *cobra.Command) command {
	c := command{
		Name:    cmd.Name(),
		Short:   cmd.Short,
		Long:    cmd.Long,
		Aliases: cmd.Aliases,
	}

	lflags := make([]flag, 0)
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		lflags = append(lflags, convertFlag(f))
	})
	c.LocalFlags = lflags

	pflags := make([]flag, 0)
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		pflags = append(pflags, convertFlag(f))
	})
	c.PersistentFlags = pflags

	subcommands := make([]command, 0)
	for _, s := range cmd.Commands() {
		if !s.Hidden {
			subcommands = append(subcommands, convert(s))
		}
	}
	c.Commands = subcommands
	return c
}

// Snippet exports the command structure as json.
func Snippet(cmd *cobra.Command) string {
	out, err := json.Marshal(convert(cmd))
	if err == nil {
		return string(out)
	}
	return err.Error()
}
