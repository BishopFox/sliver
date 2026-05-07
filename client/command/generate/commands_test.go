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
