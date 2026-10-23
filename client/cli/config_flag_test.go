package cli

import (
	"testing"
)

func TestConfigFlagRoutesBeforeAndAfterConsoleAndMCP(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup(ConsoleConfigFlagName)
	if flag == nil {
		t.Fatal("root --config flag is missing")
	}
	oldValue, oldChanged := flag.Value.String(), flag.Changed
	t.Cleanup(func() {
		_ = flag.Value.Set(oldValue)
		flag.Changed = oldChanged
	})

	for _, command := range []string{"console", "mcp"} {
		for _, before := range []bool{true, false} {
			name := command + " after --config"
			args := []string{"--config", "selected.cfg", command}
			if !before {
				name = command + " before --config"
				args = []string{command, "--config", "selected.cfg"}
			}
			t.Run(name, func(t *testing.T) {
				_ = flag.Value.Set("")
				flag.Changed = false
				cmd, remaining, err := rootCmd.Traverse(args)
				if err != nil {
					t.Fatalf("traverse %q: %v", args, err)
				}
				if cmd.Name() != command {
					t.Fatalf("command = %q, want %q", cmd.Name(), command)
				}
				if err := cmd.ParseFlags(remaining); err != nil {
					t.Fatalf("parse %q: %v", remaining, err)
				}
				got, err := cmd.Flags().GetString(ConsoleConfigFlagName)
				if err != nil || got != "selected.cfg" {
					t.Fatalf("--config = %q, error = %v", got, err)
				}
			})
		}
	}
}
