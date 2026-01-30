package generate

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

var (
	minBeaconInterval         = 5 * time.Second
	ErrBeaconIntervalTooShort = fmt.Errorf("beacon interval must be %v or greater", minBeaconInterval)
)

// GenerateBeaconCmd - The main command used to generate implant binaries
func GenerateBeaconCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	if shouldRunGenerateBeaconForm(cmd, con, args) {
		compiler, _ := compilerTargets(con)
		result, err := forms.GenerateBeaconForm(compiler)
		if err != nil {
			if errors.Is(err, forms.ErrUserAborted) {
				return
			}
			con.PrintErrorf("Generate beacon form failed: %s\n", err)
			return
		}
		if err := applyGenerateBeaconForm(cmd, result); err != nil {
			con.PrintErrorf("Failed to apply generate beacon form values: %s\n", err)
			return
		}
	}

	name, config := parseCompileFlags(cmd, con)
	if config == nil {
		return
	}
	config.IsBeacon = true
	err := parseBeaconFlags(cmd, config)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	save, _ := cmd.Flags().GetString("save")
	if save == "" {
		save, _ = os.Getwd()
	}
	if external, _ := cmd.Flags().GetBool("external-builder"); !external {
		compile(name, config, save, con)
	} else {
		externalBuild(name, config, save, con)
	}
}

func parseBeaconFlags(cmd *cobra.Command, config *clientpb.ImplantConfig) error {
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

func shouldRunGenerateBeaconForm(cmd *cobra.Command, con *console.SliverClient, args []string) bool {
	if con == nil || con.IsCLI {
		return false
	}
	if len(args) != 0 {
		return false
	}
	return cmd.Flags().NFlag() == 0
}

func applyGenerateBeaconForm(cmd *cobra.Command, result *forms.GenerateBeaconFormResult) error {
	if err := cmd.Flags().Set("os", result.OS); err != nil {
		return err
	}
	if err := cmd.Flags().Set("arch", result.Arch); err != nil {
		return err
	}
	if err := cmd.Flags().Set("format", result.Format); err != nil {
		return err
	}
	name := strings.TrimSpace(result.Name)
	if name != "" {
		if err := cmd.Flags().Set("name", name); err != nil {
			return err
		}
	}
	save := strings.TrimSpace(result.Save)
	if save != "" {
		if err := cmd.Flags().Set("save", save); err != nil {
			return err
		}
	}

	c2Value := strings.TrimSpace(result.C2Value)
	switch result.C2Type {
	case "mtls":
		if err := cmd.Flags().Set("mtls", c2Value); err != nil {
			return err
		}
	case "wg":
		if err := cmd.Flags().Set("wg", c2Value); err != nil {
			return err
		}
	case "http":
		if err := cmd.Flags().Set("http", c2Value); err != nil {
			return err
		}
	case "dns":
		if err := cmd.Flags().Set("dns", c2Value); err != nil {
			return err
		}
	case "named-pipe":
		if err := cmd.Flags().Set("named-pipe", c2Value); err != nil {
			return err
		}
	case "tcp-pivot":
		if err := cmd.Flags().Set("tcp-pivot", c2Value); err != nil {
			return err
		}
	default:
		return errors.New("unsupported C2 transport selection")
	}

	if err := setOptionalInt64Flag(cmd, "days", result.Days); err != nil {
		return err
	}
	if err := setOptionalInt64Flag(cmd, "hours", result.Hours); err != nil {
		return err
	}
	if err := setOptionalInt64Flag(cmd, "minutes", result.Minutes); err != nil {
		return err
	}
	if err := setOptionalInt64Flag(cmd, "seconds", result.Seconds); err != nil {
		return err
	}
	if err := setOptionalInt64Flag(cmd, "jitter", result.Jitter); err != nil {
		return err
	}
	return nil
}

func setOptionalInt64Flag(cmd *cobra.Command, name, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return err
	}
	if parsed < 0 {
		return fmt.Errorf("%s must be 0 or greater", name)
	}
	return cmd.Flags().Set(name, strconv.FormatInt(parsed, 10))
}
