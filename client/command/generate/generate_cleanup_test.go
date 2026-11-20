package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

func cleanupCommand(t *testing.T, beacon bool) *cobra.Command {
	t.Helper()
	cmd := commandByUse(Commands(&console.SliverClient{}), consts.GenerateStr)
	if cmd == nil {
		t.Fatal("missing generate command")
	}
	if beacon {
		cmd = commandByUse(cmd.Commands(), consts.BeaconStr)
		if cmd == nil {
			t.Fatal("missing generate beacon command")
		}
	}
	return cmd
}

func TestGenerateCleanupFlagSpaceFormAndRouting(t *testing.T) {
	for _, beacon := range []bool{false, true} {
		name := "session"
		if beacon {
			name = "beacon"
		}
		t.Run(name, func(t *testing.T) {
			defaultCmd := cleanupCommand(t, beacon)
			if flag := defaultCmd.Flags().Lookup("cleanup"); flag == nil || flag.DefValue != "true" || flag.Value.Type() != "string" {
				t.Fatalf("--cleanup is not a string flag defaulting to true: %+v", flag)
			}
			if retain, err := retainWorkspaceFromFlags(defaultCmd); err != nil || retain {
				t.Fatalf("default cleanup: retain=%t err=%v", retain, err)
			}

			cmd := cleanupCommand(t, beacon)
			if err := cmd.Flags().Parse([]string{"--cleanup", "false"}); err != nil {
				t.Fatal(err)
			}
			if len(cmd.Flags().Args()) != 0 {
				t.Fatalf("space-separated false was treated as a positional argument: %v", cmd.Flags().Args())
			}
			if retain, err := retainWorkspaceFromFlags(cmd); err != nil || !retain {
				t.Fatalf("--cleanup false: retain=%t err=%v", retain, err)
			}

			for _, tc := range []struct {
				name, want string
				args       []string
			}{
				{"invalid value", "invalid --cleanup value", []string{"--cleanup", "sometimes"}},
				{"assigned remote", "only for local generation", []string{"--cleanup", "false", "--remote-step"}},
				{"assigned remote shorthand", "only for local generation", []string{"--cleanup", "false", "-E"}},
				{"pinned builder", "only for local generation", []string{"--cleanup", "false", "--builder", "worker-a"}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					cmd := cleanupCommand(t, beacon)
					if err := cmd.Flags().Parse(tc.args); err != nil {
						t.Fatal(err)
					}
					if _, err := retainWorkspaceFromFlags(cmd); err == nil || !strings.Contains(err.Error(), tc.want) {
						t.Fatalf("flag error = %v, want %q", err, tc.want)
					}
					rpc := &generateCompatibilityRPC{}
					con := generateCompatibilityConsole(t, rpc)
					if beacon {
						GenerateBeaconCmd(cmd, con, nil)
					} else {
						GenerateCmd(cmd, con, nil)
					}
					if rpc.generateReq != nil {
						t.Fatal("invalid cleanup or remote retention reached Generate RPC")
					}
				})
			}
		})
	}
}

func TestGenerateWorkspaceRetention(t *testing.T) {
	rpc := &generateCompatibilityRPC{generateName: "payload", workspacePath: "/server/slivers/linux/amd64/payload/workspace"}
	con := generateCompatibilityConsole(t, rpc)
	path := filepath.Join(t.TempDir(), "payload")
	file, err := compileWithWorkspaceRetention("payload", &clientpb.ImplantConfig{
		GOOS: "linux", GOARCH: "amd64", Format: clientpb.OutputFormat_EXECUTABLE, Debug: true,
	}, nil, path, true, con)
	if err != nil {
		t.Fatal(err)
	}
	if rpc.generateReq == nil || !rpc.generateReq.GetRetainWorkspace() {
		t.Fatal("Generate RPC did not request workspace retention")
	}
	if file == nil || string(file.Data) != "raw" {
		t.Fatalf("generated file = %+v", file)
	}
	if saved, err := os.ReadFile(path); err != nil || string(saved) != "raw" {
		t.Fatalf("saved artifact = %q, err = %v", saved, err)
	}
}

func TestGenerateWorkspaceRetentionRequiresServerPath(t *testing.T) {
	rpc := &generateCompatibilityRPC{generateName: "payload"}
	con := generateCompatibilityConsole(t, rpc)
	path := filepath.Join(t.TempDir(), "payload")
	_, err := compileWithWorkspaceRetention("payload", &clientpb.ImplantConfig{
		GOOS: "linux", GOARCH: "amd64", Format: clientpb.OutputFormat_EXECUTABLE, Debug: true,
	}, nil, path, true, con)
	if err == nil || !strings.Contains(err.Error(), "server did not return a retained workspace path") {
		t.Fatalf("old-server retention error = %v", err)
	}
	if rpc.generateReq == nil || !rpc.generateReq.GetRetainWorkspace() {
		t.Fatal("Generate RPC did not request workspace retention")
	}
	if saved, err := os.ReadFile(path); err != nil || string(saved) != "raw" {
		t.Fatalf("completed artifact was not saved: %q, err = %v", saved, err)
	}
}
