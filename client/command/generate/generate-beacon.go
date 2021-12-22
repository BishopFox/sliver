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
	compile(config, save, con)
}

func parseBeaconFlags(ctx *grumble.Context, con *console.SliverConsoleClient, config *clientpb.ImplantConfig) error {
	interval := time.Duration(ctx.Flags.Int64("days")) * time.Hour * 24
	interval += time.Duration(ctx.Flags.Int64("hours")) * time.Hour
	interval += time.Duration(ctx.Flags.Int64("minutes")) * time.Minute
	interval += time.Duration(ctx.Flags.Int64("seconds")) * time.Second
	if interval < minBeaconInterval {
		return ErrBeaconIntervalTooShort
	}
	config.BeaconInterval = int64(interval)
	config.BeaconJitter = int64(time.Duration(ctx.Flags.Int64("jitter")) * time.Second)
	return nil
}
