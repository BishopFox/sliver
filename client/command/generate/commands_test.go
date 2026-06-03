package generate

import (
	"testing"

	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/spf13/cobra"
)

func TestGenerateSaveFlagCompletionRegistered(t *testing.T) {
	cmds := Commands(&console.SliverClient{})

	generateCmd := commandByUse(cmds, consts.GenerateStr)
	if generateCmd == nil {
		t.Fatalf("missing %q command", consts.GenerateStr)
	}
	assertFlagHasDefaultFileCompletion(t, generateCmd, "save")

	beaconCmd := commandByUse(generateCmd.Commands(), consts.BeaconStr)
	if beaconCmd == nil {
		t.Fatalf("missing %q subcommand", consts.BeaconStr)
	}
	assertFlagHasDefaultFileCompletion(t, beaconCmd, "save")

	triggerCmd := commandByUse(generateCmd.Commands(), consts.TriggerStr)
	if triggerCmd == nil {
		t.Fatalf("missing %q subcommand", consts.TriggerStr)
	}
	assertFlagHasDefaultFileCompletion(t, triggerCmd, "save")
}

func TestGenerateTriggerSubcommandRegistered(t *testing.T) {
	cmds := Commands(&console.SliverClient{})

	generateCmd := commandByUse(cmds, consts.GenerateStr)
	if generateCmd == nil {
		t.Fatalf("missing %q command", consts.GenerateStr)
	}

	triggerCmd := commandByUse(generateCmd.Commands(), consts.TriggerStr)
	if triggerCmd == nil {
		t.Fatalf("missing %q subcommand under %q", consts.TriggerStr, consts.GenerateStr)
	}

	// Trigger subcommand must have trigger-wake flags from coreImplantFlags.
	for _, flagName := range []string{
		"trigger-wake-bind",
		"trigger-wake-secret-env",
		"trigger-wake-secret",
		"trigger-wake-allowed-client",
		"ttl",
	} {
		if triggerCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("trigger subcommand missing flag %q", flagName)
		}
	}

	// Trigger subcommand must have standard implant flags.
	for _, flagName := range []string{
		"mtls", "os", "arch", "name", "format", "http", "dns",
	} {
		if triggerCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("trigger subcommand missing standard flag %q", flagName)
		}
	}

	// Trigger subcommand must NOT have beacon flags -- trigger implants
	// always use session mode, never beacon mode.
	for _, flagName := range []string{
		"seconds", "jitter", "days", "hours", "minutes",
	} {
		if triggerCmd.Flags().Lookup(flagName) != nil {
			t.Errorf("trigger subcommand should NOT have beacon flag %q", flagName)
		}
	}
}

func assertFlagHasDefaultFileCompletion(t *testing.T, cmd *cobra.Command, flag string) {
	t.Helper()

	completionFn, ok := cmd.GetFlagCompletionFunc(flag)
	if !ok {
		t.Fatalf("%q missing %q flag completion func", cmd.Use, flag)
	}

	values, directive := completionFn(cmd, nil, "")
	if len(values) != 0 {
		t.Fatalf("%q %q completion should return no explicit values, got %d", cmd.Use, flag, len(values))
	}
	if directive != cobra.ShellCompDirectiveDefault {
		t.Fatalf("%q %q completion directive mismatch: got=%v want=%v", cmd.Use, flag, directive, cobra.ShellCompDirectiveDefault)
	}
}

func commandByUse(cmds []*cobra.Command, use string) *cobra.Command {
	for _, cmd := range cmds {
		if cmd.Use == use {
			return cmd
		}
	}
	return nil
}
