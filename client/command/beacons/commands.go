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
		f.BoolP("force", "F", false, "force killing the beacon")

		f.StringP("filter", "f", "", "filter beacons by substring")
		f.StringP("filter-re", "e", "", "filter beacons by regular expression")
	})
	flags.BindFlagCompletions(beaconsCmd, func(comp *carapace.ActionMap) {
		(*comp)["kill"] = BeaconIDCompleter(con)
	})
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
		results := make([]string, 0)

		beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
		if err == nil {
			for _, b := range beacons.Beacons {
				link := fmt.Sprintf("[%s <- %s]", b.ActiveC2, b.RemoteAddress)
				id := fmt.Sprintf("%s (%d)", b.Name, b.PID)
				userHost := fmt.Sprintf("%s@%s", b.Username, b.Hostname)
				desc := strings.Join([]string{id, userHost, link}, " ")

				results = append(results, b.ID[:8])
				results = append(results, desc)
			}
		}
		return carapace.ActionValuesDescribed(results...).Tag("beacons")
	}

	return carapace.ActionCallback(callback)
}
