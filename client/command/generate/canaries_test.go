package generate

import (
	"strconv"
	"testing"

	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/spf13/cobra"
)

func TestCanariesCommandRegistered(t *testing.T) {
	root := &cobra.Command{Use: "sliver"}
	root.AddCommand(Commands(&console.SliverClient{})...)

	cmd, _, err := root.Find([]string{consts.CanariesStr})
	if err != nil {
		t.Fatalf("find canaries command: %v", err)
	}
	if cmd.Parent() != root {
		t.Fatal("canaries command is not registered at the root")
	}
	if cmd.Run == nil {
		t.Fatal("canaries command has no handler")
	}
	if cmd.GroupID != consts.PayloadsHelpGroup {
		t.Errorf("canaries group = %q, want %q", cmd.GroupID, consts.PayloadsHelpGroup)
	}

	for _, want := range []struct {
		name         string
		shorthand    string
		valueType    string
		defaultValue string
	}{
		{name: "burned", shorthand: "b", valueType: "bool", defaultValue: "false"},
		{name: "timeout", shorthand: "t", valueType: "int64", defaultValue: strconv.Itoa(flags.DefaultTimeout)},
	} {
		flag := cmd.Flags().Lookup(want.name)
		if flag == nil {
			t.Fatalf("canaries command is missing --%s", want.name)
		}
		if flag.Shorthand != want.shorthand {
			t.Errorf("--%s shorthand = %q, want %q", want.name, flag.Shorthand, want.shorthand)
		}
		if flag.Value.Type() != want.valueType {
			t.Errorf("--%s type = %q, want %q", want.name, flag.Value.Type(), want.valueType)
		}
		if flag.DefValue != want.defaultValue {
			t.Errorf("--%s default = %q, want %q", want.name, flag.DefValue, want.defaultValue)
		}
	}
}
