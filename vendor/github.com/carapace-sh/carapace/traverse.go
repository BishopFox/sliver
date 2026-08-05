package carapace

import (
	"strings"

	"github.com/carapace-sh/carapace/internal/common"
	"github.com/carapace-sh/carapace/internal/env"
	"github.com/carapace-sh/carapace/internal/pflagfork"
	"github.com/carapace-sh/carapace/pkg/style"
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
	var dash bool // dash encountered

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
		case arg == string(fs.Prefix())+string(fs.Prefix()):
			LOG.Printf("arg %#v is dash\n", arg)
			inArgs = append(inArgs, context.Args[i:]...)
			dash = true
			break loop

		// flag
		case !cmd.DisableFlagParsing && strings.HasPrefix(arg, string(fs.Prefix())) && (fs.IsInterspersed() || len(inPositionals) == 0):
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
	switch {
	case dash: // skip looking for flags in dash arguments
	case inFlag != nil && len(inFlag.Args) == 0 && inFlag.Consumes(""):
		LOG.Printf("removing arg %#v since it is a flag missing its argument\n", toParse[len(toParse)-1])
		toParse = toParse[:len(toParse)-1]
	case (fs.IsInterspersed() || len(inPositionals) == 0) && fs.IsShorthandSeries(context.Value): // TODO shorthand series isn't correct anymore (can have value attached)
		LOG.Printf("arg %#v is a shorthand flag series", context.Value) // TODO not aways correct
		localInFlag := fs.LookupArg(context.Value)

		if localInFlag != nil && (len(localInFlag.Args) == 0 || localInFlag.Args[0] == "") && (!localInFlag.IsOptarg() || strings.HasSuffix(localInFlag.ArgPrefix, string(localInFlag.OptargDelimiter()))) { // TODO && len(context.Value) > 2 {
			// TODO check if empty prefix
			suffix := localInFlag.ArgPrefix[strings.LastIndex(localInFlag.ArgPrefix, localInFlag.Shorthand):]
			LOG.Printf("removing suffix %#v since it is a flag missing its argument\n", suffix)
			toParse = append(toParse, strings.TrimSuffix(localInFlag.ArgPrefix, suffix))
		} else if localInFlag == nil {
			// shorthand lookup failed (e.g. due to ArgumentStyle restriction)
			// context.Value is not a valid flag form; skip adding it to toParse
			// so that flag completions are shown instead
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

	// flag argument (only when the flag accepts the next-arg style)
	case inFlag != nil && inFlag.Consumes(context.Value) && inFlag.AcceptsNext():
		LOG.Printf("completing flag argument of %#v for arg %#v\n", inFlag.Name, context.Value)
		context.Parts = inFlag.Args
		return storage.getFlag(cmd, inFlag.Name), context

	// flag
	case !cmd.DisableFlagParsing && strings.HasPrefix(context.Value, string(fs.Prefix())) && (fs.IsInterspersed() || len(inPositionals) == 0):
		if f := fs.LookupArg(context.Value); f != nil && len(f.Args) > 0 {
			LOG.Printf("completing optional flag argument for arg %#v with prefix %#v\n", context.Value, f.ArgPrefix)

			switch f.Value.Type() {
			case "bool":
				//nolint:govet
				return ActionValues("true", "false").StyleF(style.ForKeyword).Usage(f.Usage).Prefix(f.ArgPrefix), context
			default:
				return storage.getFlag(cmd, f.Name).Prefix(f.ArgPrefix), context
			}
		} else if f != nil && fs.IsPosix() && !strings.HasPrefix(context.Value, string(fs.Prefix())+string(fs.Prefix())) && !f.IsOptarg() && f.ArgPrefix == context.Value && f.AcceptsAttached() {
			LOG.Printf("completing attached flag argument for arg %#v with prefix %#v\n", context.Value, f.ArgPrefix)
			return storage.getFlag(cmd, f.Name).Prefix(f.ArgPrefix), context
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
