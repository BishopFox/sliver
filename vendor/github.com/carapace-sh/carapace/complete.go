package carapace

import (
	"os"

	"github.com/carapace-sh/carapace/internal/config"
	"github.com/carapace-sh/carapace/internal/shell/bash"
	"github.com/carapace-sh/carapace/internal/shell/cmd_clink"
	"github.com/carapace-sh/carapace/internal/shell/nushell"
	"github.com/carapace-sh/carapace/pkg/ps"
	"github.com/spf13/cobra"
)

func complete(cmd *cobra.Command, args []string) (string, error) {
	switch len(args) {
	case 0:
		return Gen(cmd).Snippet(ps.DetermineShell())
	case 1:
		return Gen(cmd).Snippet(args[0])
	default:
		initHelpCompletion(cmd)

		switch ps.DetermineShell() {
		case "nushell":
			args = nushell.Patch(args) // handle open quotes
			LOG.Printf("patching args to %#v", args)
		case "bash": // TODO what about oil and such?
			LOG.Printf("COMP_LINE is %#v", os.Getenv("COMP_LINE"))
			LOG.Printf("COMP_POINT is %#v", os.Getenv("COMP_POINT"))
			LOG.Printf("COMP_WORDBREAKS is %#v", os.Getenv("COMP_WORDBREAKS"))
			var err error
			args, err = bash.Patch(args) // handle redirects
			LOG.Printf("patching args to %#v", args)
			if err != nil {
				context := NewContext(args...)
				if _, ok := err.(bash.RedirectError); ok {
					LOG.Printf("completing redirect target for %#v", args)
					return ActionFiles().Invoke(context).value(args[0], args[len(args)-1]), nil
				}
				return ActionMessage(err.Error()).Invoke(context).value(args[0], args[len(args)-1]), nil
			}
		case "cmd-clink":
			var err error
			args, err = cmd_clink.Patch(args)
			LOG.Printf("patching args to %#v", args)
			if err != nil {
				context := NewContext(args...)
				return ActionMessage(err.Error()).Invoke(context).value(args[0], args[len(args)-1]), nil
			}
		}

		action, context := traverse(cmd, args[2:])
		if err := config.Load(); err != nil {
			action = ActionMessage("failed to load config: " + err.Error())
		}
		return action.Invoke(context).value(args[0], args[len(args)-1]), nil
	}
}
