package generate

import (
	"fmt"
	"os"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

var (
	minBeaconInterval         = 5 * time.Second
	ErrBeaconIntervalTooShort = fmt.Errorf("beacon interval must be %v or greater", minBeaconInterval)
)

// GenerateBeaconCmd - The main command used to generate implant binaries
func GenerateBeaconCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	config := parseCompileFlags(ctx, con)
	if config == nil {
		return
	}
	config.IsBeacon = true
	err := parseBeaconFlags(ctx, con, config)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	save := ctx.Flags.String("save")
	if save == "" {
		save, _ = os.Getwd()
	}
	if !ctx.Flags.Bool("external-builder") {
		compile(config, ctx.Flags.Bool("disable-sgn"), save, con)
	} else {
		externalBuild(config, save, con)
	}
}

func parseBeaconFlags(ctx *grumble.Context, con *console.SliverConsoleClient, config *clientpb.ImplantConfig) error {
	interval := time.Duration(ctx.Flags.Int64("days")) * time.Hour * 24
	interval += time.Duration(ctx.Flags.Int64("hours")) * time.Hour
	interval += time.Duration(ctx.Flags.Int64("minutes")) * time.Minute

	/*
		If seconds has not been specified but any of the other time units have, then do not add
		the default 60 seconds to the interval.

		If seconds have been specified, then add them regardless.
	*/
	if (ctx.Flags["seconds"].IsDefault && interval.Seconds() == 0) || (!ctx.Flags["seconds"].IsDefault) {
		interval += time.Duration(ctx.Flags.Int64("seconds")) * time.Second
	}

	if interval < minBeaconInterval {
		return ErrBeaconIntervalTooShort
	}
	config.BeaconInterval = int64(interval)
	config.BeaconJitter = int64(time.Duration(ctx.Flags.Int64("jitter")) * time.Second)
	return nil
}
