// Package carapace is a command argument completion generator for spf13/cobra
package carapace

import (
	"os"

	"github.com/rsteube/carapace/internal/shell"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Carapace wraps cobra.Command to define completions.
type Carapace struct {
	cmd *cobra.Command
}

// Gen initialized Carapace for given command.
func Gen(cmd *cobra.Command) *Carapace {
	addCompletionCommand(cmd)
	storage.bridge(cmd)

	return &Carapace{
		cmd: cmd,
	}
}

// PreRun sets a function to be run before completion.
func (c Carapace) PreRun(f func(cmd *cobra.Command, args []string)) {
	if entry := storage.get(c.cmd); entry.prerun != nil {
		_f := entry.prerun
		entry.prerun = func(cmd *cobra.Command, args []string) {
			// TODO yuck - probably best to append to a slice in storage
			_f(cmd, args)
			f(cmd, args)
		}
	} else {
		entry.prerun = f
	}
}

// PreInvoke sets a function to alter actions before they are invoked.
func (c Carapace) PreInvoke(f func(cmd *cobra.Command, flag *pflag.Flag, action Action) Action) {
	if entry := storage.get(c.cmd); entry.preinvoke != nil {
		_f := entry.preinvoke
		entry.preinvoke = func(cmd *cobra.Command, flag *pflag.Flag, action Action) Action {
			return f(cmd, flag, _f(cmd, flag, action))
		}
	} else {
		entry.preinvoke = f
	}
}

// PositionalCompletion defines completion for positional arguments using a list of Actions.
func (c Carapace) PositionalCompletion(action ...Action) {
	storage.get(c.cmd).positional = action
}

// PositionalAnyCompletion defines completion for any positional arguments not already defined.
func (c Carapace) PositionalAnyCompletion(action Action) {
	storage.get(c.cmd).positionalAny = &action
}

// DashCompletion defines completion for positional arguments after dash (`--`) using a list of Actions.
func (c Carapace) DashCompletion(action ...Action) {
	storage.get(c.cmd).dash = action
}

// DashAnyCompletion defines completion for any positional arguments after dash (`--`) not already defined.
func (c Carapace) DashAnyCompletion(action Action) {
	storage.get(c.cmd).dashAny = &action
}

// FlagCompletion defines completion for flags using a map consisting of name and Action.
func (c Carapace) FlagCompletion(actions ActionMap) {
	e := storage.get(c.cmd)
	e.flagMutex.Lock()
	defer e.flagMutex.Unlock()

	if e.flag == nil {
		e.flag = actions
	} else {
		for name, action := range actions {
			e.flag[name] = action
		}
	}
}

const annotation_standalone = "carapace_standalone"

// Standalone prevents cobra defaults interfering with standalone mode (e.g. implicit help command).
func (c Carapace) Standalone() {
	c.cmd.CompletionOptions = cobra.CompletionOptions{
		DisableDefaultCmd: true,
	}

	if c.cmd.Annotations == nil {
		c.cmd.Annotations = make(map[string]string)
	}
	c.cmd.Annotations[annotation_standalone] = "true"

	c.PreRun(func(cmd *cobra.Command, args []string) {
		if f := cmd.Flag("help"); f == nil {
			cmd.Flags().Bool("help", false, "")
			cmd.Flag("help").Hidden = true
		} else if f.Annotations != nil {
			if _, ok := f.Annotations[cobra.FlagSetByCobraAnnotation]; ok {
				cmd.Flag("help").Hidden = true
			}
		}
	})
	c.cmd.SetHelpCommand(&cobra.Command{Use: "_carapace_help", Hidden: true, Deprecated: "fake help command to prevent default"})
}

// Snippet creates completion script for given shell.
func (c Carapace) Snippet(name string) (string, error) {
	return shell.Snippet(c.cmd, name)
}

// IsCallback returns true if current program invocation is a callback.
func IsCallback() bool {
	return len(os.Args) > 1 && os.Args[1] == "_carapace"
}

// Test verifies the configuration (e.g. flag name exists)
//
//	func TestCarapace(t *testing.T) {
//	    carapace.Test(t)
//	}
func Test(t interface{ Error(args ...interface{}) }) {
	for _, e := range storage.check() {
		t.Error(e)
	}
}
