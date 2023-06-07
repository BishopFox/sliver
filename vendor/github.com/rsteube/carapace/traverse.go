package carapace

import (
	"strings"

	"github.com/rsteube/carapace/internal/common"
	"github.com/rsteube/carapace/internal/env"
	"github.com/rsteube/carapace/internal/pflagfork"
	"github.com/rsteube/carapace/pkg/style"
	"github.com/spf13/cobra"
)

type _inFlag struct { // TODO rename or integrate into pflagfork.Flag?
	*pflagfork.Flag
	// currently consumed args since encountered flag
	Args []string
}

func (f _inFlag) Consumes(arg string) bool {
	switch {
	case f.Flag == nil:
		return false
	case !f.TakesValue():
		return false
	case f.IsOptarg():
		return false
	case len(f.Args) == 0:
		return true
	case f.Nargs() > 1 && len(f.Args) < f.Nargs():
		return true
	case f.Nargs() < 0 && !strings.HasPrefix(arg, "-"):
		return true
	default:
		return false
	}
}

func traverse(c *cobra.Command, args []string) (Action, Context) {
	LOG.Printf("traverse called for %#v with args %#v\n", c.Name(), args)
	storage.preRun(c, args)

	if env.Lenient() {
		LOG.Printf("allowing unknown flags")
		c.FParseErrWhitelist.UnknownFlags = true
	}

	inArgs := []string{} // args consumed by current command
	var inFlag *_inFlag  // last encountered flag that still expects arguments
	c.LocalFlags()       // TODO force  c.mergePersistentFlags() which is missing from c.Flags()
	fs := pflagfork.FlagSet{FlagSet: c.Flags()}

	context := NewContext(args...)
loop:
	for i, arg := range context.Args {
		switch {
		// flag argument
		case inFlag != nil && inFlag.Consumes(arg):
			LOG.Printf("arg %#v is a flag argument\n", arg)
			inArgs = append(inArgs, arg)
			inFlag.Args = append(inFlag.Args, arg)

			if !inFlag.Consumes("") {
				inFlag = nil // no more args expected
			}
			continue

		// dash
		case arg == "--":
			LOG.Printf("arg %#v is dash\n", arg)
			inArgs = append(inArgs, context.Args[i:]...)
			break loop

		// flag
		case !c.DisableFlagParsing && strings.HasPrefix(arg, "-"):
			LOG.Printf("arg %#v is a flag\n", arg)
			inArgs = append(inArgs, arg)
			inFlag = &_inFlag{
				Flag: fs.LookupArg(arg),
				Args: []string{},
			}

			if inFlag.Flag == nil {
				LOG.Printf("flag %#v is unknown", arg)
			}
			continue

		// subcommand
		case subcommand(c, arg) != nil:
			LOG.Printf("arg %#v is a subcommand\n", arg)

			switch {
			case c.DisableFlagParsing:
				LOG.Printf("flag parsing disabled for %#v\n", c.Name())

			default:
				LOG.Printf("parsing flags for %#v with args %#v\n", c.Name(), inArgs)
				if err := c.ParseFlags(inArgs); err != nil {
					return ActionMessage(err.Error()), context
				}
				context.Args = c.Flags().Args()
			}

			return traverse(subcommand(c, arg), args[i+1:])

		// positional
		default:
			LOG.Printf("arg %#v is a positional\n", arg)
			inArgs = append(inArgs, arg)
		}
	}

	toParse := inArgs
	if inFlag != nil && len(inFlag.Args) == 0 && inFlag.Consumes("") {
		LOG.Printf("removing arg %#v since it is a flag missing its argument\n", toParse[len(toParse)-1])
		toParse = toParse[:len(toParse)-1]
	} else if fs.IsShorthandSeries(context.Value) {
		LOG.Printf("arg %#v is a shorthand flag series", context.Value)
		localInFlag := &_inFlag{
			Flag: fs.LookupArg(context.Value),
			Args: []string{},
		}
		if localInFlag.Consumes("") && len(context.Value) > 2 {
			LOG.Printf("removing shorthand %#v from flag series since it is missing its argument\n", localInFlag.Shorthand)
			toParse = append(toParse, strings.TrimSuffix(context.Value, localInFlag.Shorthand))
		} else {
			toParse = append(toParse, context.Value)
		}

	}

	// TODO duplicated code
	switch {
	case c.DisableFlagParsing:
		LOG.Printf("flag parsing is disabled for %#v\n", c.Name())

	default:
		LOG.Printf("parsing flags for %#v with args %#v\n", c.Name(), toParse)
		if err := c.ParseFlags(toParse); err != nil {
			return ActionMessage(err.Error()), context
		}
		context.Args = c.Flags().Args()
	}

	switch {
	// dash argument
	case common.IsDash(c):
		LOG.Printf("completing dash for arg %#v\n", context.Value)
		context.Args = c.Flags().Args()[c.ArgsLenAtDash():]
		LOG.Printf("context: %#v\n", context.Args)

		return storage.getPositional(c, len(context.Args)), context

	// flag argument
	case inFlag != nil && inFlag.Consumes(context.Value):
		LOG.Printf("completing flag argument of %#v for arg %#v\n", inFlag.Name, context.Value)
		context.Parts = inFlag.Args
		return storage.getFlag(c, inFlag.Name), context

	// flag
	case !c.DisableFlagParsing && strings.HasPrefix(context.Value, "-"):
		if f := fs.LookupArg(context.Value); f != nil && f.IsOptarg() && strings.Contains(context.Value, string(f.OptargDelimiter())) {
			LOG.Printf("completing optional flag argument for arg %#v\n", context.Value)
			prefix, optarg := f.Split(context.Value)
			context.Value = optarg

			switch f.Value.Type() {
			case "bool":
				return ActionValues("true", "false").StyleF(style.ForKeyword).Prefix(prefix), context
			default:
				return storage.getFlag(c, f.Name).Prefix(prefix), context
			}
		}
		LOG.Printf("completing flags for arg %#v\n", context.Value)
		return actionFlags(c), context

	// positional or subcommand
	default:
		LOG.Printf("completing positionals and subcommands for arg %#v\n", context.Value)
		batch := Batch(storage.getPositional(c, len(context.Args)))
		if c.HasAvailableSubCommands() && len(context.Args) == 0 {
			batch = append(batch, actionSubcommands(c))
		}
		return batch.ToA(), context
	}
}

func subcommand(cmd *cobra.Command, arg string) *cobra.Command {
	if subcommand, _, _ := cmd.Find([]string{arg}); subcommand != cmd {
		return subcommand
	}
	return nil
}
