package beacons

import (
	"context"
	"fmt"
	"strings"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	beaconsCmd := &cobra.Command{
		Use:     consts.BeaconsStr,
		Short:   "Manage beacons",
		Long:    help.GetHelpFor([]string{consts.BeaconsStr}),
		GroupID: consts.SliverHelpGroup,
		Run: func(cmd *cobra.Command, args []string) {
			BeaconsCmd(cmd, con, args)
		},
	}
	flags.Bind("beacons", true, beaconsCmd, func(f *pflag.FlagSet) {
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.Bind("beacons", false, beaconsCmd, func(f *pflag.FlagSet) {
		f.StringP("kill", "k", "", "kill the designated beacon")
		f.BoolP("kill-all", "K", false, "kill all beacons")
		f.StringP("interact", "i", "", "interact with a beacon")
		f.BoolP("force", "F", false, "force killing the beacon")

		f.StringP("filter", "f", "", "filter beacons by substring")
		f.StringP("filter-re", "e", "", "filter beacons by regular expression")
	})
	flags.BindFlagCompletions(beaconsCmd, func(comp *carapace.ActionMap) {
		(*comp)["kill"] = BeaconIDCompleter(con)
		(*comp)["interact"] = BeaconIDCompleter(con)
	})
	registerBeaconIDFlagCompletion(beaconsCmd, "interact", con)
	beaconsRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove a beacon",
		Long:  help.GetHelpFor([]string{consts.BeaconsStr, consts.RmStr}),
		Run: func(cmd *cobra.Command, args []string) {
			BeaconsRmCmd(cmd, con, args)
		},
	}
	carapace.Gen(beaconsRmCmd).PositionalCompletion(BeaconIDCompleter(con))
	beaconsCmd.AddCommand(beaconsRmCmd)

	beaconsWatchCmd := &cobra.Command{
		Use:   consts.WatchStr,
		Short: "Watch your beacons",
		Long:  help.GetHelpFor([]string{consts.BeaconsStr, consts.WatchStr}),
		Run: func(cmd *cobra.Command, args []string) {
			BeaconsWatchCmd(cmd, con, args)
		},
	}
	beaconsCmd.AddCommand(beaconsWatchCmd)

	beaconsPruneCmd := &cobra.Command{
		Use:   consts.PruneStr,
		Short: "Prune stale beacons automatically",
		Long:  help.GetHelpFor([]string{consts.BeaconsStr, consts.PruneStr}),
		Run: func(cmd *cobra.Command, args []string) {
			BeaconsPruneCmd(cmd, con, args)
		},
	}
	flags.Bind("beacons", false, beaconsPruneCmd, func(f *pflag.FlagSet) {
		f.StringP("duration", "d", "1h", "duration to prune beacons that have missed their last checkin")
	})
	beaconsCmd.AddCommand(beaconsPruneCmd)

	return []*cobra.Command{beaconsCmd}
}

// BeaconIDCompleter completes beacon IDs
func BeaconIDCompleter(con *console.SliverClient) carapace.Action {
	callback := func(_ carapace.Context) carapace.Action {
		return carapace.ActionValuesDescribed(beaconCompletionPairs(con)...).Tag("beacons")
	}

	return carapace.ActionCallback(callback)
}

func registerBeaconIDFlagCompletion(cmd *cobra.Command, name string, con *console.SliverClient) {
	if cmd == nil {
		return
	}
	if _, ok := cmd.GetFlagCompletionFunc(name); ok {
		return
	}
	if cmd.Flags().Lookup(name) == nil && cmd.PersistentFlags().Lookup(name) == nil {
		return
	}
	_ = cmd.RegisterFlagCompletionFunc(name, func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		values := describedValuesToTabs(beaconCompletionPairs(con))
		return filterCompletionValues(values, toComplete), cobra.ShellCompDirectiveNoFileComp
	})
}

func beaconCompletionPairs(con *console.SliverClient) []string {
	results := make([]string, 0)
	if con == nil || con.Rpc == nil {
		return results
	}

	beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
	if err != nil {
		return results
	}

	for _, b := range beacons.Beacons {
		link := fmt.Sprintf("[%s <- %s]", b.ActiveC2, b.RemoteAddress)
		id := fmt.Sprintf("%s (%d)", b.Name, b.PID)
		userHost := fmt.Sprintf("%s@%s", b.Username, b.Hostname)
		desc := strings.Join([]string{id, userHost, link}, " ")

		results = append(results, b.ID[:8], desc)
	}

	return results
}

func describedValuesToTabs(values []string) []string {
	tabbed := make([]string, 0, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		tabbed = append(tabbed, fmt.Sprintf("%s\t%s", values[i], values[i+1]))
	}
	return tabbed
}

func filterCompletionValues(values []string, prefix string) []string {
	if prefix == "" {
		return values
	}

	filtered := make([]string, 0, len(values))
	for _, value := range values {
		candidate := value
		if tab := strings.IndexByte(value, '\t'); tab >= 0 {
			candidate = value[:tab]
		}
		if strings.HasPrefix(candidate, prefix) {
			filtered = append(filtered, value)
		}
	}

	return filtered
}
