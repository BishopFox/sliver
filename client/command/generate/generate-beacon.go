package generate

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

var (
	minBeaconInterval         = 5 * time.Second
	ErrBeaconIntervalTooShort = fmt.Errorf("beacon interval must be %v or greater", minBeaconInterval)
)

// GenerateBeaconCmd - The main command used to generate implant binaries
func GenerateBeaconCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
	config := parseCompileFlags(cmd, con)
	if config == nil {
		return
	}
	config.IsBeacon = true
	err := parseBeaconFlags(cmd, con, config)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	save, _ := cmd.Flags().GetString("save")
	if save == "" {
		save, _ = os.Getwd()
	}
	if external, _ := cmd.Flags().GetBool("external-builder"); !external {
		disableSGN, _ := cmd.Flags().GetBool("disable-sgn")
		compile(config, disableSGN, save, con)
	} else {
		externalBuild(config, save, con)
	}
}

func parseBeaconFlags(cmd *cobra.Command, con *console.SliverConsoleClient, config *clientpb.ImplantConfig) error {
	days, _ := cmd.Flags().GetInt64("days")
	hours, _ := cmd.Flags().GetInt64("hours")
	minutes, _ := cmd.Flags().GetInt64("minutes")
	interval := time.Duration(days) * time.Hour * 24
	interval += time.Duration(hours) * time.Hour
	interval += time.Duration(minutes) * time.Minute

	/*
		If seconds has not been specified but any of the other time units have, then do not add
		the default 60 seconds to the interval.

		If seconds have been specified, then add them regardless.
	*/
	if (!cmd.Flags().Changed("seconds") && interval.Seconds() == 0) || (cmd.Flags().Changed("seconds")) {
		// if (ctx.Flags["seconds"].IsDefault && interval.Seconds() == 0) || (!ctx.Flags["seconds"].IsDefault) {
		seconds, _ := cmd.Flags().GetInt64("seconds")
		interval += time.Duration(seconds) * time.Second
	}

	if interval < minBeaconInterval {
		return ErrBeaconIntervalTooShort
	}

	beaconJitter, _ := cmd.Flags().GetInt64("jitter")
	config.BeaconInterval = int64(interval)
	config.BeaconJitter = int64(time.Duration(beaconJitter) * time.Second)
	return nil
}
