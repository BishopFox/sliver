package carapace

import (
	"strings"

	"github.com/rsteube/carapace/internal/common"
	"github.com/rsteube/carapace/internal/env"
	"github.com/rsteube/carapace/internal/pflagfork"
	"github.com/rsteube/carapace/pkg/style"
	"github.com/spf13/cobra"
)

func traverse(cmd *cobra.Command, args []string) (Action, Context) {
	LOG.Printf("traverse called for %#v with args %#v\n", cmd.Name(), args)
	storage.preRun(cmd, args)

	if env.Lenient() {
		LOG.Printf("allowing unknown flags")
		cmd.FParseErrWhitelist.UnknownFlags = true
	}

	inArgs := []string{}        // args consumed by current command
	inPositionals := []string{} // positionals consumed by current command
	var inFlag *pflagfork.Flag  // last encountered flag that still expects arguments
	cmd.LocalFlags()            // TODO force  c.mergePersistentFlags() which is missing from c.Flags()
	fs := pflagfork.FlagSet{FlagSet: cmd.Flags()}

	context := NewContext(args...)
	context.cmd = cmd
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
		case !cmd.DisableFlagParsing && strings.HasPrefix(arg, "-") && (fs.IsInterspersed() || len(inPositionals) == 0):
			LOG.Printf("arg %#v is a flag\n", arg)
			inArgs = append(inArgs, arg)
			inFlag = fs.LookupArg(arg)

			if inFlag == nil {
				LOG.Printf("flag %#v is unknown", arg)
			}
			continue

		// subcommand
		case subcommand(cmd, arg) != nil:
			LOG.Printf("arg %#v is a subcommand\n", arg)

			switch {
			case cmd.DisableFlagParsing:
				LOG.Printf("flag parsing disabled for %#v\n", cmd.Name())

			default:
				LOG.Printf("parsing flags for %#v with args %#v\n", cmd.Name(), inArgs)
				if err := cmd.ParseFlags(inArgs); err != nil {
					return ActionMessage(err.Error()), context
				}
				context.Args = cmd.Flags().Args()
			}

			return traverse(subcommand(cmd, arg), args[i+1:])

		// positional
		default:
			LOG.Printf("arg %#v is a positional\n", arg)
			inArgs = append(inArgs, arg)
			inPositionals = append(inPositionals, arg)
		}
	}

	toParse := inArgs
	if inFlag != nil && len(inFlag.Args) == 0 && inFlag.Consumes("") {
		LOG.Printf("removing arg %#v since it is a flag missing its argument\n", toParse[len(toParse)-1])
		toParse = toParse[:len(toParse)-1]
	} else if (fs.IsInterspersed() || len(inPositionals) == 0) && fs.IsShorthandSeries(context.Value) { // TODO shorthand series isn't correct anymore (can have value attached)
		LOG.Printf("arg %#v is a shorthand flag series", context.Value) // TODO not aways correct
		localInFlag := fs.LookupArg(context.Value)

		if localInFlag != nil && (len(localInFlag.Args) == 0 || localInFlag.Args[0] == "") && (!localInFlag.IsOptarg() || strings.HasSuffix(localInFlag.Prefix, string(localInFlag.OptargDelimiter()))) { // TODO && len(context.Value) > 2 {
			// TODO check if empty prefix
			suffix := localInFlag.Prefix[strings.LastIndex(localInFlag.Prefix, localInFlag.Shorthand):]
			LOG.Printf("removing suffix %#v since it is a flag missing its argument\n", suffix)
			toParse = append(toParse, strings.TrimSuffix(localInFlag.Prefix, suffix))
		} else {
			LOG.Printf("adding shorthand flag %#v", context.Value)
			toParse = append(toParse, context.Value)
		}

	}

	// TODO duplicated code
	switch {
	case cmd.DisableFlagParsing:
		LOG.Printf("flag parsing is disabled for %#v\n", cmd.Name())

	default:
		LOG.Printf("parsing flags for %#v with args %#v\n", cmd.Name(), toParse)
		if err := cmd.ParseFlags(toParse); err != nil {
			return ActionMessage(err.Error()), context
		}
		context.Args = cmd.Flags().Args()
	}

	switch {
	// dash argument
	case common.IsDash(cmd):
		LOG.Printf("completing dash for arg %#v\n", context.Value)
		context.Args = cmd.Flags().Args()[cmd.ArgsLenAtDash():]
		LOG.Printf("context: %#v\n", context.Args)

		return storage.getPositional(cmd, len(context.Args)), context

	// flag argument
	case inFlag != nil && inFlag.Consumes(context.Value):
		LOG.Printf("completing flag argument of %#v for arg %#v\n", inFlag.Name, context.Value)
		context.Parts = inFlag.Args
		return storage.getFlag(cmd, inFlag.Name), context

	// flag
	case !cmd.DisableFlagParsing && strings.HasPrefix(context.Value, "-") && (fs.IsInterspersed() || len(inPositionals) == 0):
		if f := fs.LookupArg(context.Value); f != nil && len(f.Args) > 0 {
			LOG.Printf("completing optional flag argument for arg %#v with prefix %#v\n", context.Value, f.Prefix)

			switch f.Value.Type() {
			case "bool":
				return ActionValues("true", "false").StyleF(style.ForKeyword).Usage(f.Usage).Prefix(f.Prefix), context
			default:
				return storage.getFlag(cmd, f.Name).Prefix(f.Prefix), context
			}
		} else if f != nil && fs.IsPosix() && !strings.HasPrefix(context.Value, "--") && !f.IsOptarg() && f.Prefix == context.Value {
			LOG.Printf("completing attached flag argument for arg %#v with prefix %#v\n", context.Value, f.Prefix)
			return storage.getFlag(cmd, f.Name).Prefix(f.Prefix), context
		}
		LOG.Printf("completing flags for arg %#v\n", context.Value)
		return actionFlags(cmd), context

	// positional or subcommand
	default:
		LOG.Printf("completing positionals and subcommands for arg %#v\n", context.Value)
		batch := Batch(storage.getPositional(cmd, len(context.Args)))
		if cmd.HasAvailableSubCommands() && len(context.Args) == 0 {
			batch = append(batch, ActionCommands(cmd))
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
