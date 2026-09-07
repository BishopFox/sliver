// Package carapace is a command argument completion generator for spf13/cobra
package carapace

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/carapace-sh/carapace/internal/shell"
	"github.com/carapace-sh/carapace/internal/shell/multi"
	"github.com/carapace-sh/carapace/pkg/uid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Carapace wraps cobra.Command to define completions.
type Carapace struct {
	cmd *cobra.Command
}

// Option configures completion for a command.
// Options are applied to the storage entry keyed by the command,
// so successive Gen() calls on the same command accumulate.
type Option func(*entry)

// WithSubcommands configures multi-completer support.
// The listed commands serve as independent completers within one binary.
// The binary name is automatically included as a pseudo-subcommand for self-completion.
// When set, Snippet() produces a multi-completer snippet and Execute() handles arg rewriting.
func WithSubcommands(cmds ...*cobra.Command) Option {
	return func(e *entry) {
		e.subcommands = cmds

		exe := uid.Executable()
		names := make([]string, 0, len(cmds)+1)
		names = append(names, exe)
		for _, cmd := range cmds {
			names = append(names, cmd.Name())
		}
		e.subcommandNames = names

		if e.defaultName == "" {
			e.defaultName = exe
		}
	}
}

// WithDefault sets the name used for the shell completer function
// in multi-completer snippets (e.g. _carapace_<name>_completer).
// Defaults to the executable name. No-op without WithSubcommands.
func WithDefault(name string) Option {
	return func(e *entry) {
		e.defaultName = name
	}
}

// WithSnippetFuncs adds custom shell code to snippets.
// The map key is the shell name; the value is the code to inject.
// Multiple calls accumulate per shell in order.
func WithSnippetFuncs(funcs map[string]string) Option {
	return func(e *entry) {
		if e.snippetFuncs == nil {
			e.snippetFuncs = make(map[string][]string)
		}
		for sh, code := range funcs {
			e.snippetFuncs[sh] = append(e.snippetFuncs[sh], code)
		}
	}
}

// Gen initializes Carapace for given command.
// Options configure snippet enrichment, multi-completer routing, etc.
func Gen(cmd *cobra.Command, opts ...Option) *Carapace {
	addCompletionCommand(cmd)
	storage.bridge(cmd)

	e := storage.get(cmd)
	for _, opt := range opts {
		opt(e)
	}

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
		maps.Copy(e.flag, actions)
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
// For multi-completers, produces a multi-completer snippet (all commands at once).
// For single completers, produces a standard single-command snippet.
func (c Carapace) Snippet(name string) (string, error) {
	entry := storage.get(c.cmd)
	if len(entry.subcommands) > 0 {
		return multi.Snippet(name, entry.subcommandNames, entry.defaultName, entry.snippetFuncs)
	}
	s, err := shell.Snippet(c.cmd, name)
	if err != nil {
		return s, err
	}
	if entry.snippetFuncs != nil {
		if extra, ok := entry.snippetFuncs[name]; ok && len(extra) > 0 {
			s = strings.Join(extra, "\n") + "\n" + s
		}
	}
	return s, nil
}

// Execute intercepts os.Args for multi-completer routing and then
// calls cmd.Execute(). Call this instead of cmd.Execute().
// For single completers (no WithSubcommands), this just calls cmd.Execute().
func (c Carapace) Execute() error {
	if entry := storage.get(c.cmd); len(entry.subcommands) > 0 {
		c.rewriteArgs(entry)
	}
	return c.cmd.Execute()
}

// IsCallback returns true if current program invocation is a callback.
func IsCallback() bool {
	return len(os.Args) > 1 && os.Args[1] == "_carapace"
}

// rewriteArgs intercepts os.Args for multi-completer routing.
// Directly adapted from carapace-magick's root.go Execute().
func (c Carapace) rewriteArgs(e *entry) {
	exe := uid.Executable()
	isCompleterSubcommand := func(name string) bool {
		return slices.Contains(e.subcommandNames, name)
	}
	isRootSubcommand := func(name string) bool {
		for _, cmd := range c.cmd.Commands() {
			if cmd.Name() == name {
				return true
			}
		}
		return false
	}

	// Root-level _carapace: multi-completer snippet + bridge routing
	if len(os.Args) > 1 && os.Args[1] == "_carapace" {
		if len(os.Args) < 4 {
			shell := ""
			if len(os.Args) > 2 {
				shell = os.Args[2]
			}
			fmt.Println(multi.SnippetOrEmpty(shell, e.subcommandNames, e.defaultName, e.snippetFuncs))
			os.Exit(0)
		}
		// Route completion/export requests to the correct subcommand.
		// bridge.ActionCarapace("binary", "identify") calls:
		//   binary _carapace export "" identify -verbose image.png
		// Rewrite to:
		//   binary identify _carapace export "" -verbose image.png
		//
		// bridge.ActionCarapace("binary") calls:
		//   binary _carapace export "" ""
		// Rewrite to:
		//   binary binary _carapace export "" ""
		// (pseudo-subcommand form for self-completion)
		if len(os.Args) > 4 && isCompleterSubcommand(os.Args[4]) && os.Args[4] != exe {
			os.Args = append(
				[]string{os.Args[0], os.Args[4], "_carapace", os.Args[2], os.Args[3]},
				os.Args[5:]...,
			)
		} else {
			os.Args = append(
				[]string{os.Args[0], exe, "_carapace"},
				os.Args[2:]...,
			)
		}
	}

	// Pseudo-subcommand "binary" — handle without a cobra command.
	// Provides root-level completion (subcommand names) and single-command snippet.
	if len(os.Args) > 2 && os.Args[1] == exe && os.Args[2] == "_carapace" {
		if len(os.Args) < 4 {
			return
		}
		if os.Args[3] == "export" {
			// Export format: binary binary _carapace export <shell> "" <args...>
			if len(os.Args) > 6 && isRootSubcommand(os.Args[6]) {
				// Route to the actual subcommand
				os.Args = append(
					[]string{os.Args[0], os.Args[6], "_carapace", os.Args[3], os.Args[4], os.Args[5]},
					os.Args[7:]...,
				)
			} else {
				// Root-level completion — strip pseudo-subcommand and let rootCmd handle it
				os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
			}
		} else {
			// Shell format: binary binary _carapace <shell> [args...]
			if len(os.Args) < 5 {
				// Snippet request only
				fmt.Println(multi.SingleSnippetOrEmpty(os.Args[3], exe, e.subcommandNames, e.defaultName, e.snippetFuncs))
				os.Exit(0)
			}
			if isRootSubcommand(os.Args[4]) {
				// Route to the actual subcommand
				os.Args = append(
					[]string{os.Args[0], os.Args[4], "_carapace", os.Args[3]},
					os.Args[5:]...,
				)
			} else {
				// Root-level completion — strip pseudo-subcommand and let rootCmd handle it
				os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
			}
		}
	}

	// Subcommand-level snippet request: binary <subcommand> _carapace [shell]
	if len(os.Args) > 2 && isCompleterSubcommand(os.Args[1]) && os.Args[2] == "_carapace" {
		if len(os.Args) < 5 {
			shell := ""
			if len(os.Args) > 3 {
				shell = os.Args[3]
			}
			fmt.Println(multi.SingleSnippetOrEmpty(shell, os.Args[1], e.subcommandNames, e.defaultName, e.snippetFuncs))
			os.Exit(0)
		}
	}
}

// Test verifies the configuration (e.g. flag name exists)
//
//	func TestCarapace(t *testing.T) {
//	    carapace.Test(t)
//	}
func Test(t interface{ Error(args ...any) }) {
	for _, e := range storage.check() {
		t.Error(e)
	}
}
