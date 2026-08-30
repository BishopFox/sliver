//go:build client

package opfor

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	opforengine "github.com/sliverarmory/opfor"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type testRPC struct {
	rpcpb.SliverRPCClient

	sessions *clientpb.Sessions
	beacons  *clientpb.Beacons
	call     func(context.Context, *sliverpb.CallExtensionReq) (*sliverpb.CallExtension, error)
	task     func(context.Context, *clientpb.BeaconTask) (*clientpb.BeaconTask, error)

	mu        sync.Mutex
	calls     []*sliverpb.CallExtensionReq
	taskCalls int
}

func (rpc *testRPC) GetSessions(context.Context, *commonpb.Empty, ...grpc.CallOption) (*clientpb.Sessions, error) {
	return rpc.sessions, nil
}

func (rpc *testRPC) GetBeacons(context.Context, *commonpb.Empty, ...grpc.CallOption) (*clientpb.Beacons, error) {
	return rpc.beacons, nil
}

func (rpc *testRPC) CallExtension(ctx context.Context, request *sliverpb.CallExtensionReq, _ ...grpc.CallOption) (*sliverpb.CallExtension, error) {
	rpc.mu.Lock()
	rpc.calls = append(rpc.calls, proto.Clone(request).(*sliverpb.CallExtensionReq))
	rpc.mu.Unlock()
	if rpc.call == nil {
		return nil, fmt.Errorf("unexpected CallExtension")
	}
	return rpc.call(ctx, request)
}

func (rpc *testRPC) GetBeaconTaskContent(ctx context.Context, task *clientpb.BeaconTask, _ ...grpc.CallOption) (*clientpb.BeaconTask, error) {
	rpc.mu.Lock()
	rpc.taskCalls++
	rpc.mu.Unlock()
	if rpc.task == nil {
		return nil, fmt.Errorf("unexpected GetBeaconTaskContent")
	}
	return rpc.task(ctx, task)
}

func (rpc *testRPC) snapshotCalls() ([]*sliverpb.CallExtensionReq, int) {
	rpc.mu.Lock()
	defer rpc.mu.Unlock()
	return append([]*sliverpb.CallExtensionReq(nil), rpc.calls...), rpc.taskCalls
}

type testOutput struct {
	mu      sync.Mutex
	output  strings.Builder
	info    strings.Builder
	success strings.Builder
	errors  strings.Builder
}

func (output *testOutput) Printf(format string, values ...any) {
	output.mu.Lock()
	fmt.Fprintf(&output.output, format, values...)
	output.mu.Unlock()
}

func (output *testOutput) PrintInfof(format string, values ...any) {
	output.mu.Lock()
	fmt.Fprintf(&output.info, format, values...)
	output.mu.Unlock()
}

func (output *testOutput) PrintSuccessf(format string, values ...any) {
	output.mu.Lock()
	fmt.Fprintf(&output.success, format, values...)
	output.mu.Unlock()
}

func (output *testOutput) PrintErrorf(format string, values ...any) {
	output.mu.Lock()
	fmt.Fprintf(&output.errors, format, values...)
	output.mu.Unlock()
}

func (output *testOutput) stdout() string {
	output.mu.Lock()
	defer output.mu.Unlock()
	return output.output.String()
}

func (output *testOutput) stderr() string {
	output.mu.Lock()
	defer output.mu.Unlock()
	return output.errors.String()
}

func (output *testOutput) infoOutput() string {
	output.mu.Lock()
	defer output.mu.Unlock()
	return output.info.String()
}

func (output *testOutput) successOutput() string {
	output.mu.Lock()
	defer output.mu.Unlock()
	return output.success.String()
}

type testPromptUI struct {
	confirmAnswer bool
	inputAnswer   string
	confirmTitle  string
	inputTitle    string
	inputDefault  string
}

func (ui *testPromptUI) Confirm(title string, _ bool) (bool, error) {
	ui.confirmTitle = title
	return ui.confirmAnswer, nil
}

func (ui *testPromptUI) Input(title, defaultValue string) (string, error) {
	ui.inputTitle = title
	ui.inputDefault = defaultValue
	return ui.inputAnswer, nil
}

func newTestManager(t *testing.T, rpc *testRPC) (*Manager, *testOutput) {
	t.Helper()
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", t.TempDir())
	client := console.NewConsole(false)
	client.Rpc = rpc
	output := &testOutput{}
	manager, err := newManagerWithOutput(client, &testPromptUI{}, output)
	if err != nil {
		t.Fatalf("newManagerWithOutput: %v", err)
	}
	t.Cleanup(func() { _ = manager.runtime.Close(context.Background()) })
	return manager, output
}

func installTestManager(t *testing.T, manager *Manager) {
	t.Helper()
	clientManagersMu.Lock()
	previous, existed := clientManagers[manager.client]
	clientManagers[manager.client] = managerEntry{manager: manager}
	clientManagersMu.Unlock()
	t.Cleanup(func() {
		clientManagersMu.Lock()
		if existed {
			clientManagers[manager.client] = previous
		} else {
			delete(clientManagers, manager.client)
		}
		clientManagersMu.Unlock()
	})
}

func TestManagerLoadsRegistersAndInvokesSyncBOF(t *testing.T) {
	rpc := &testRPC{
		sessions: &clientpb.Sessions{Sessions: []*clientpb.Session{{
			ID: "session-1", Arch: "amd64", Capabilities: sliverpb.CapabilityBOFV1,
		}}},
		beacons: &clientpb.Beacons{},
		call: func(context.Context, *sliverpb.CallExtensionReq) (*sliverpb.CallExtension, error) {
			return &sliverpb.CallExtension{Output: []byte("sync output")}, nil
		},
	}
	manager, output := newTestManager(t, rpc)

	namespace := &cobra.Command{Use: "opfor"}
	manager.mu.Lock()
	manager.roots[namespace] = map[string]*cobra.Command{}
	manager.mu.Unlock()

	scriptPath := filepath.Join(t.TempDir(), "example.cna")
	source := `
alias runbof {
    $packed = bof_pack($1, "iz", 16909060, $2);
    beacon_inline_execute($1, "OBJ", "go", $packed);
}
beacon_command_register("runbof", "Run the example BOF", "Runs exact BOF bytes");
`
	if err := os.WriteFile(scriptPath, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	absolute, err := manager.Load(context.Background(), scriptPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if absolute != scriptPath || !reflect.DeepEqual(manager.Paths(), []string{scriptPath}) {
		t.Fatalf("loaded paths = %q / %q, want %q", absolute, manager.Paths(), scriptPath)
	}
	assertSyncBOFAlias(t, directChild(namespace, "runbof"))

	if err := manager.invokeAlias(context.Background(), "runbof", "runbof fox", []string{"fox"}, "session-1"); err != nil {
		t.Fatalf("invokeAlias: %v", err)
	}
	calls, _ := rpc.snapshotCalls()
	if len(calls) != 1 {
		t.Fatalf("CallExtension calls = %d, want 1", len(calls))
	}
	assertSyncBOFCall(t, calls[0])
	if got := output.stdout(); got != "sync output\n" {
		t.Fatalf("stdout = %q, want %q", got, "sync output\\n")
	}

	if _, err := manager.Unload(context.Background(), filepath.Base(scriptPath)); err != nil {
		t.Fatalf("Unload: %v", err)
	}
	assertSyncBOFUnloaded(t, manager, namespace)
}

func assertSyncBOFAlias(t *testing.T, command *cobra.Command) {
	t.Helper()
	if command == nil {
		t.Fatal("CNA alias was not registered beneath the opfor namespace")
	}
	if command.GroupID != "" {
		t.Fatalf("nested CNA alias group = %q, want empty", command.GroupID)
	}
	if command.Short != "Run the example BOF" || !strings.Contains(command.Long, "Runs exact BOF bytes") {
		t.Fatalf("alias help = short:%q long:%q", command.Short, command.Long)
	}
}

func assertSyncBOFCall(t *testing.T, call *sliverpb.CallExtensionReq) {
	t.Helper()
	digest := sha256.Sum256([]byte("OBJ"))
	wantArguments := []byte{
		12, 0, 0, 0,
		4, 3, 2, 1,
		4, 0, 0, 0, 'f', 'o', 'x', 0,
	}
	want := &sliverpb.CallExtensionReq{
		Name:           hex.EncodeToString(digest[:]),
		Args:           wantArguments,
		Export:         "go",
		BOFData:        []byte("OBJ"),
		IsBOF:          true,
		WantBOFOutputs: true,
		Request: &commonpb.Request{
			Timeout:   int64(bofRequestFallback),
			SessionID: "session-1",
		},
	}
	if !proto.Equal(call, want) {
		t.Fatalf("CallExtension request mismatch\n got: %s\nwant: %s", call, want)
	}
}

func assertSyncBOFUnloaded(t *testing.T, manager *Manager, namespace *cobra.Command) {
	t.Helper()
	if directChild(namespace, "runbof") != nil || len(manager.Paths()) != 0 {
		t.Fatalf("alias/path survived unload: command=%p paths=%q", directChild(namespace, "runbof"), manager.Paths())
	}
}

func TestManagerCheckCompilesWithoutExecuting(t *testing.T) {
	manager, output := newTestManager(t, &testRPC{})
	scriptPath := filepath.Join(t.TempDir(), "check.cna")
	if err := os.WriteFile(scriptPath, []byte(`
println("check must not execute");
alias checkonly {
    println("alias must not persist");
}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	absolute, err := manager.Check(scriptPath)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if absolute != scriptPath {
		t.Fatalf("checked path = %q, want %q", absolute, scriptPath)
	}
	if got := output.stdout(); got != "" {
		t.Fatalf("check executed top-level output: %q", got)
	}
	if manager.hasAlias("checkonly") || len(manager.Paths()) != 0 {
		t.Fatalf("check retained script state: alias=%v paths=%q", manager.hasAlias("checkonly"), manager.Paths())
	}

	invalidPath := filepath.Join(t.TempDir(), "invalid.cna")
	if err := os.WriteFile(invalidPath, []byte(`alias broken {`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Check(invalidPath); err == nil || !strings.Contains(err.Error(), "opfor: compile "+invalidPath) {
		t.Fatalf("invalid check error = %v, want compile path context", err)
	}
}

func TestRunCommandExecutesOnceWithARGVAndRetiresBindings(t *testing.T) {
	manager, output := newTestManager(t, &testRPC{})
	installTestManager(t, manager)
	scriptPath := filepath.Join(t.TempDir(), "run.cna")
	if err := os.WriteFile(scriptPath, []byte(`
println("argv=" . @ARGV[0] . "|" . @ARGV[1]);
alias ephemeral {
    println("ephemeral alias");
}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	namespace := Commands(manager.client)[0]
	manager.mu.Lock()
	manager.roots[namespace] = map[string]*cobra.Command{}
	manager.mu.Unlock()
	root := &cobra.Command{Use: "implant", TraverseChildren: true}
	root.AddGroup(&cobra.Group{ID: consts.SliverCoreHelpGroup, Title: "Core"})
	root.AddCommand(namespace)
	root.SetArgs([]string{consts.OpforStr, "run", scriptPath, "alpha", "--literal-flag"})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	if _, err := root.ExecuteC(); err != nil {
		t.Fatalf("opfor run: %v", err)
	}
	if got := output.stdout(); got != "argv=alpha|--literal-flag\n" {
		t.Fatalf("run output = %q, want exact @ARGV output", got)
	}
	if manager.hasAlias("ephemeral") || directChild(namespace, "ephemeral") != nil || len(manager.Paths()) != 0 {
		t.Fatalf("run retained one-shot state: alias=%v command=%p paths=%q", manager.hasAlias("ephemeral"), directChild(namespace, "ephemeral"), manager.Paths())
	}
}

func TestManagerRunPreservesLoadedSameNameAlias(t *testing.T) {
	manager, _ := newTestManager(t, &testRPC{})
	namespace := &cobra.Command{Use: "opfor"}
	manager.mu.Lock()
	manager.roots[namespace] = map[string]*cobra.Command{}
	manager.mu.Unlock()

	persistentPath := filepath.Join(t.TempDir(), "persistent.cna")
	if err := os.WriteFile(persistentPath, []byte(`alias shared { println("persistent"); }`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Load(context.Background(), persistentPath); err != nil {
		t.Fatalf("Load persistent alias: %v", err)
	}
	persistentCommand := directChild(namespace, "shared")
	if persistentCommand == nil {
		t.Fatal("persistent alias was not attached")
	}

	oneShotPath := filepath.Join(t.TempDir(), "oneshot.cna")
	if err := os.WriteFile(oneShotPath, []byte(`alias shared { println("one-shot"); }`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Run(context.Background(), oneShotPath, nil); err != nil {
		t.Fatalf("Run one-shot alias: %v", err)
	}
	if !manager.hasAlias("shared") || directChild(namespace, "shared") != persistentCommand {
		t.Fatalf("one-shot run displaced persistent alias: alias=%v command=%p want=%p", manager.hasAlias("shared"), directChild(namespace, "shared"), persistentCommand)
	}
	if got := manager.Paths(); !reflect.DeepEqual(got, []string{persistentPath}) {
		t.Fatalf("loaded paths after one-shot run = %q, want %q", got, []string{persistentPath})
	}
}

func TestCheckAndLoadCommandsReportTheirEffects(t *testing.T) {
	manager, output := newTestManager(t, &testRPC{})
	installTestManager(t, manager)
	scriptPath := filepath.Join(t.TempDir(), "reported.cna")
	if err := os.WriteFile(scriptPath, []byte(`
beacon_command_register("zulu", "Zulu short", "Zulu detail");
beacon_command_register("alpha", "Alpha short", "Alpha detail");
alias zulu { println("zulu"); }
alias alpha { println("alpha"); }
`), 0o600); err != nil {
		t.Fatal(err)
	}
	command := managementCommand(manager.client, consts.SliverCoreHelpGroup)
	assertCheckCommandEffects(t, command, output, scriptPath)
	wantOutput := assertLoadCommandEffects(t, command, output, scriptPath)
	assertAliasHelpEffects(t, command, output, wantOutput)
}

func assertCheckCommandEffects(t *testing.T, command *cobra.Command, output *testOutput, scriptPath string) {
	t.Helper()
	check := requireManagementRunE(t, command, "check")
	if err := check.RunE(check, []string{scriptPath}); err != nil {
		t.Fatalf("opfor check: %v", err)
	}
	if got := output.stdout(); got != scriptPath+": ok\n" {
		t.Fatalf("check result = %q, want %q", got, scriptPath+": ok\\n")
	}
}

func assertLoadCommandEffects(t *testing.T, command *cobra.Command, output *testOutput, scriptPath string) string {
	t.Helper()
	load := requireManagementRunE(t, command, "load")
	if err := load.RunE(load, []string{scriptPath}); err != nil {
		t.Fatalf("opfor load: %v", err)
	}
	if got := output.successOutput(); !strings.Contains(got, "Loaded CNA script "+scriptPath) {
		t.Fatalf("load success output = %q", got)
	}
	if got, want := output.infoOutput(), "Registered CNA aliases:\nView CNA alias help:\n"; got != want {
		t.Fatalf("load info output = %q, want %q", got, want)
	}
	wantOutput := scriptPath + ": ok\n" +
		"  opfor alpha\n" +
		"  opfor zulu\n" +
		"  opfor help alpha\n" +
		"  opfor help zulu\n"
	if got := output.stdout(); got != wantOutput {
		t.Fatalf("load command output = %q, want %q", got, wantOutput)
	}
	if strings.Contains(output.stdout(), "--help") {
		t.Fatalf("load command suggested unsafe alias flag help: %q", output.stdout())
	}
	return wantOutput
}

func assertAliasHelpEffects(t *testing.T, command *cobra.Command, output *testOutput, wantOutput string) {
	t.Helper()
	aliasHelp := requireManagementRunE(t, command, "help")
	if err := aliasHelp.RunE(aliasHelp, []string{"alpha"}); err != nil {
		t.Fatalf("opfor help alpha: %v", err)
	}
	wantOutput += "Command: opfor alpha [arguments...]\n\nAlpha short\n\nAlpha detail\n"
	if got := output.stdout(); got != wantOutput {
		t.Fatalf("alias help output = %q, want %q", got, wantOutput)
	}
	if err := aliasHelp.RunE(aliasHelp, []string{"missing"}); err == nil || !strings.Contains(err.Error(), `unknown CNA alias "missing"`) {
		t.Fatalf("missing alias help error = %v", err)
	}
}

func requireManagementRunE(t *testing.T, command *cobra.Command, name string) *cobra.Command {
	t.Helper()
	child := directChild(command, name)
	if child == nil || child.RunE == nil {
		t.Fatalf("opfor %s command does not expose RunE", name)
	}
	return child
}

func TestLoadCommandReportsOnlyNewInvokableAliases(t *testing.T) {
	t.Run("does not repeat aliases from earlier scripts", testLoadCommandDoesNotRepeatEarlierAliases)
	t.Run("does not advertise a management-command collision", testLoadCommandDoesNotAdvertiseCollision)
}

func testLoadCommandDoesNotRepeatEarlierAliases(t *testing.T) {
	manager, output := newTestManager(t, &testRPC{})
	installTestManager(t, manager)
	preloadedPath := filepath.Join(t.TempDir(), "preloaded.cna")
	if err := os.WriteFile(preloadedPath, []byte(`alias earlier { println("earlier"); }`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Load(context.Background(), preloadedPath); err != nil {
		t.Fatalf("preload: %v", err)
	}

	scriptPath := filepath.Join(t.TempDir(), "no-alias.cna")
	if err := os.WriteFile(scriptPath, []byte(`$value = 1;`), 0o600); err != nil {
		t.Fatal(err)
	}
	command := managementCommand(manager.client, consts.SliverCoreHelpGroup)
	load := requireManagementRunE(t, command, "load")
	if err := load.RunE(load, []string{scriptPath}); err != nil {
		t.Fatalf("load no-alias script: %v", err)
	}
	if got := output.infoOutput(); !strings.Contains(got, "no invokable Beacon aliases") {
		t.Fatalf("load info output = %q", got)
	}
	if got := output.stdout(); strings.Contains(got, "opfor earlier") {
		t.Fatalf("load repeated an earlier script's alias: %q", got)
	}
}

func testLoadCommandDoesNotAdvertiseCollision(t *testing.T) {
	manager, output := newTestManager(t, &testRPC{})
	installTestManager(t, manager)
	scriptPath := filepath.Join(t.TempDir(), "collision.cna")
	if err := os.WriteFile(scriptPath, []byte(`alias run { println("collision"); }`), 0o600); err != nil {
		t.Fatal(err)
	}
	command := managementCommand(manager.client, consts.SliverCoreHelpGroup)
	staticRun := directChild(command, "run")
	load := requireManagementRunE(t, command, "load")
	if err := load.RunE(load, []string{scriptPath}); err != nil {
		t.Fatalf("load colliding alias: %v", err)
	}
	if directChild(command, "run") != staticRun {
		t.Fatal("CNA alias displaced the static run command")
	}
	if got := output.infoOutput(); !strings.Contains(got, "no invokable Beacon aliases") {
		t.Fatalf("collision load info output = %q", got)
	}
	if got := output.stdout(); strings.Contains(got, "opfor run") {
		t.Fatalf("load advertised a colliding alias: %q", got)
	}
	aliasHelp := requireManagementRunE(t, command, "help")
	if err := aliasHelp.RunE(aliasHelp, []string{"run"}); err == nil || !strings.Contains(err.Error(), `unknown CNA alias "run"`) {
		t.Fatalf("colliding alias help error = %v", err)
	}
}

func TestInvokeAliasPreservesExactParsedArguments(t *testing.T) {
	rpc := &testRPC{
		sessions: &clientpb.Sessions{Sessions: []*clientpb.Session{{
			ID: "session-arguments", Arch: "amd64", Capabilities: sliverpb.CapabilityBOFV1,
		}}},
		beacons: &clientpb.Beacons{},
		call: func(context.Context, *sliverpb.CallExtensionReq) (*sliverpb.CallExtension, error) {
			return &sliverpb.CallExtension{}, nil
		},
	}
	manager, _ := newTestManager(t, rpc)
	scriptPath := filepath.Join(t.TempDir(), "arguments.cna")
	if err := os.WriteFile(scriptPath, []byte(`
alias exactargs {
    $packed = bof_pack($1, "zzz", $2, $3, $4);
    beacon_inline_execute($1, "OBJ", "go", $packed);
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Load(context.Background(), scriptPath); err != nil {
		t.Fatalf("Load: %v", err)
	}

	arguments := []string{`C:\Program Files\x`, "", `literal"quote`}
	rawInput := "exactargs " + strings.Join(arguments, " ")
	if err := manager.invokeAlias(context.Background(), "exactargs", rawInput, arguments, "session-arguments"); err != nil {
		t.Fatalf("invokeAlias with exact parsed arguments: %v", err)
	}
	calls, _ := rpc.snapshotCalls()
	if len(calls) != 1 {
		t.Fatalf("CallExtension calls = %d, want 1", len(calls))
	}
	want := expectedPackedStringArguments(arguments...)
	if !bytes.Equal(calls[0].Args, want) {
		t.Fatalf("packed whitespace/empty/quote arguments = %x, want %x", calls[0].Args, want)
	}
}

func TestStaticManagementCommandDispatchesAliasLoadedAfterConstruction(t *testing.T) {
	session := &clientpb.Session{
		ID: "session-static", Arch: "amd64", Capabilities: sliverpb.CapabilityBOFV1,
	}
	rpc := &testRPC{
		sessions: &clientpb.Sessions{Sessions: []*clientpb.Session{session}},
		beacons:  &clientpb.Beacons{},
		call: func(context.Context, *sliverpb.CallExtensionReq) (*sliverpb.CallExtension, error) {
			return &sliverpb.CallExtension{}, nil
		},
	}
	manager, _ := newTestManager(t, rpc)
	installTestManager(t, manager)
	manager.client.IsCLI = true
	manager.client.ActiveTarget.Set(session, nil)

	// One-shot CLI command trees are constructed before --rc scripts run.
	namespace := Commands(manager.client)[0]
	scriptPath := filepath.Join(t.TempDir(), "late.cna")
	if err := os.WriteFile(scriptPath, []byte(`
alias latealias {
    $packed = bof_pack($1, "zzzz", $2, $3, $4, $5);
    beacon_inline_execute($1, "OBJ", "go", $packed);
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Load(context.Background(), scriptPath); err != nil {
		t.Fatalf("load after command construction: %v", err)
	}
	if directChild(namespace, "latealias") != nil {
		t.Fatal("late alias unexpectedly mutated the static command tree")
	}
	for _, name := range []string{"check", "help", "load", "unload", "list", "run"} {
		if directChild(namespace, name) == nil {
			t.Fatalf("management subcommand %q was not preserved", name)
		}
	}

	root := &cobra.Command{Use: "implant"}
	root.AddGroup(&cobra.Group{ID: consts.SliverCoreHelpGroup, Title: "Core"})
	root.AddCommand(namespace)
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	tests := []struct {
		name      string
		arguments []string
		want      []string
	}{
		{name: "exact ordinary arguments", arguments: []string{`value with whitespace`, ""}, want: []string{`value with whitespace`, ""}},
		{name: "post-alias timeout tokens", arguments: []string{"payload", "--timeout", "2"}, want: []string{"payload", "--timeout", "2"}},
		{name: "unknown flag-like CNA argument", arguments: []string{"--foo"}, want: []string{"--foo"}},
		{name: "separator preserves literal timeout", arguments: []string{"--", "--timeout"}, want: []string{"--timeout"}},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root.SetArgs(append([]string{consts.OpforStr, "latealias"}, test.arguments...))
			if _, err := root.ExecuteC(); err != nil {
				t.Fatalf("execute late-loaded alias through static command: %v", err)
			}
			calls, _ := rpc.snapshotCalls()
			if len(calls) != index+1 {
				t.Fatalf("CallExtension calls = %d, want %d", len(calls), index+1)
			}
			padded := append(append([]string(nil), test.want...), make([]string, 4-len(test.want))...)
			if want := expectedPackedStringArguments(padded...); !bytes.Equal(calls[index].Args, want) {
				t.Fatalf("late alias arguments = %x, want %x", calls[index].Args, want)
			}
		})
	}
}

func expectedPackedStringArguments(values ...string) []byte {
	packed := make([]byte, 0)
	for _, value := range values {
		start := len(packed)
		packed = append(packed, make([]byte, 4+len(value)+1)...)
		binary.LittleEndian.PutUint32(packed[start:start+4], uint32(len(value)+1))
		copy(packed[start+4:], value)
	}
	arguments := make([]byte, 4+len(packed))
	binary.LittleEndian.PutUint32(arguments[:4], uint32(len(packed)))
	copy(arguments[4:], packed)
	return arguments
}

func TestCommandFactoriesUseValidMenuGroups(t *testing.T) {
	client := console.NewConsole(false)
	tests := []struct {
		name      string
		factory   func(*console.SliverClient) []*cobra.Command
		wantGroup string
	}{
		{name: "implant", factory: Commands, wantGroup: consts.SliverCoreHelpGroup},
		{name: "server", factory: ServerCommands, wantGroup: consts.GenericHelpGroup},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commands := test.factory(client)
			if len(commands) != 1 {
				t.Fatalf("command count = %d, want 1", len(commands))
			}
			if commands[0].Name() != consts.OpforStr || commands[0].GroupID != test.wantGroup {
				t.Fatalf("command name/group = %q/%q, want %q/%q", commands[0].Name(), commands[0].GroupID, consts.OpforStr, test.wantGroup)
			}
			timeoutFlag := commands[0].PersistentFlags().Lookup("timeout")
			if timeoutFlag == nil || timeoutFlag.Shorthand != "t" || timeoutFlag.DefValue != "600" {
				t.Fatalf("persistent timeout flag = %#v, want -t/--timeout default 600", timeoutFlag)
			}
			if !strings.Contains(commands[0].Long, "before the alias name") {
				t.Fatalf("opfor help does not explain timeout placement: %q", commands[0].Long)
			}
			if !strings.Contains(commands[0].Long, "run executes top-level code once") || !strings.Contains(commands[0].Long, "Most BOF CNAs") {
				t.Fatalf("opfor help does not explain run/load semantics: %q", commands[0].Long)
			}

			root := &cobra.Command{Use: test.name}
			root.AddGroup(&cobra.Group{ID: test.wantGroup, Title: test.wantGroup})
			root.AddCommand(commands...)
			root.SetArgs([]string{consts.OpforStr, "--help"})
			root.SetOut(&bytes.Buffer{})
			root.SetErr(&bytes.Buffer{})
			if _, err := root.ExecuteC(); err != nil {
				t.Fatalf("execute grouped command tree: %v", err)
			}
		})
	}
}

func TestDynamicAliasTimeoutOnInteractiveAndCLIRoots(t *testing.T) {
	tests := []struct {
		name         string
		traverse     bool
		arguments    []string
		wantArgs     []string
		wantTimeout  time.Duration
		minimumSlack time.Duration
	}{
		{name: "interactive long separated", arguments: []string{"--timeout", "2", "scriptalias", "payload"}, wantArgs: []string{"payload"}, wantTimeout: 2 * time.Second, minimumSlack: 500 * time.Millisecond},
		{name: "interactive long equals", arguments: []string{"--timeout=2", "scriptalias", "payload"}, wantArgs: []string{"payload"}, wantTimeout: 2 * time.Second, minimumSlack: 500 * time.Millisecond},
		{name: "interactive short separated", arguments: []string{"-t", "2", "scriptalias", "payload"}, wantArgs: []string{"payload"}, wantTimeout: 2 * time.Second, minimumSlack: 500 * time.Millisecond},
		{name: "interactive short equals", arguments: []string{"-t=2", "scriptalias", "payload"}, wantArgs: []string{"payload"}, wantTimeout: 2 * time.Second, minimumSlack: 500 * time.Millisecond},
		{name: "CLI parses persistent flag", traverse: true, arguments: []string{"--timeout", "2", "scriptalias", "payload"}, wantArgs: []string{"payload"}, wantTimeout: 2 * time.Second, minimumSlack: 500 * time.Millisecond},
		{name: "separator preserves literal timeout", arguments: []string{"scriptalias", "--", "--timeout"}, wantArgs: []string{"--timeout"}, wantTimeout: aliasInvocationTimeout, minimumSlack: time.Second},
		{name: "host parsing stops at first CNA argument", arguments: []string{"scriptalias", "payload", "--timeout", "2"}, wantArgs: []string{"payload", "--timeout", "2"}, wantTimeout: aliasInvocationTimeout, minimumSlack: time.Second},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := &cobra.Command{Use: "implant", TraverseChildren: test.traverse}
			root.AddGroup(&cobra.Group{ID: consts.SliverCoreHelpGroup, Title: "Core"})
			namespace := managementCommand(console.NewConsole(false), consts.SliverCoreHelpGroup)
			var captured []string
			var remaining time.Duration
			alias := &cobra.Command{
				Use:                "scriptalias",
				DisableFlagParsing: true,
				RunE: func(command *cobra.Command, args []string) error {
					cleaned, override, err := consumeAliasHostArguments(args)
					if err != nil {
						return err
					}
					captured = append([]string(nil), cleaned...)
					ctx, cancel, err := commandTimeoutContextWithOverride(command, override)
					if err != nil {
						return err
					}
					defer cancel()
					deadline, ok := ctx.Deadline()
					if !ok {
						return fmt.Errorf("timeout context has no deadline")
					}
					remaining = time.Until(deadline)
					return nil
				},
			}
			namespace.AddCommand(alias)
			root.AddCommand(namespace)
			root.SetArgs(append([]string{consts.OpforStr}, test.arguments...))
			root.SetOut(&bytes.Buffer{})
			root.SetErr(&bytes.Buffer{})
			if _, err := root.ExecuteC(); err != nil {
				t.Fatalf("execute dynamic alias: %v", err)
			}
			if !reflect.DeepEqual(captured, test.wantArgs) {
				t.Fatalf("CNA arguments = %#v, want %#v", captured, test.wantArgs)
			}
			if remaining < test.wantTimeout-test.minimumSlack || remaining > test.wantTimeout {
				t.Fatalf("dynamic alias timeout remaining = %s, want approximately %s", remaining, test.wantTimeout)
			}
		})
	}
}

func TestInvalidPersistentTimeoutsReturnErrors(t *testing.T) {
	overflow := math.MaxInt64/int64(time.Second) + 1
	tests := []struct {
		name  string
		value string
	}{
		{name: "zero", value: "0"},
		{name: "negative", value: "-1"},
		{name: "duration overflow", value: fmt.Sprint(overflow)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager, _ := newTestManager(t, &testRPC{})
			installTestManager(t, manager)
			command := Commands(manager.client)[0]
			command.SetArgs([]string{"--timeout=" + test.value, "list"})
			command.SetOut(&bytes.Buffer{})
			command.SetErr(&bytes.Buffer{})
			if _, err := command.ExecuteC(); err == nil || !strings.Contains(err.Error(), "invalid timeout") {
				t.Fatalf("persistent timeout %q error = %v, want invalid-timeout failure", test.value, err)
			}
		})
	}
}

func TestTargetRequestTimeoutUsesDeadlineAndBackgroundFallback(t *testing.T) {
	target := resolvedTarget{session: &clientpb.Session{ID: "session-1"}}
	background := targetRequest(context.Background(), target)
	if time.Duration(background.Timeout) != bofRequestFallback {
		t.Fatalf("background request timeout = %s, want %s", time.Duration(background.Timeout), bofRequestFallback)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	deadlineRequest := targetRequest(ctx, target)
	got := time.Duration(deadlineRequest.Timeout)
	if got < 3500*time.Millisecond || got > 4*time.Second {
		t.Fatalf("deadline-derived request timeout = %s, want approximately 4s", got)
	}
}

func TestManagementCommandsPropagateErrorsAndSucceedForHelpAndList(t *testing.T) {
	manager, _ := newTestManager(t, &testRPC{})
	installTestManager(t, manager)
	command := managementCommand(manager.client, consts.SliverCoreHelpGroup)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})

	missing := filepath.Join(t.TempDir(), "missing.cna")
	if err := command.RunE(command, []string{missing}); err == nil || !strings.Contains(err.Error(), "read") {
		t.Fatalf("direct missing script error = %v, want read failure", err)
	}

	unload := directChild(command, "unload")
	if unload == nil || unload.RunE == nil {
		t.Fatal("opfor unload command does not expose RunE")
	}
	if err := unload.RunE(unload, []string{missing}); err == nil || !strings.Contains(err.Error(), "not loaded") {
		t.Fatalf("unload missing script error = %v, want not-loaded failure", err)
	}

	if err := command.RunE(command, nil); err != nil {
		t.Fatalf("opfor help: %v", err)
	}
	list := directChild(command, "list")
	if list == nil || list.RunE == nil {
		t.Fatal("opfor list command does not expose RunE")
	}
	if err := list.RunE(list, nil); err != nil {
		t.Fatalf("opfor list: %v", err)
	}
}

func TestExecuteBeaconRejectsMissingCapability(t *testing.T) {
	rpc := &testRPC{
		sessions: &clientpb.Sessions{Sessions: []*clientpb.Session{{ID: "old-session", Arch: "amd64"}}},
		beacons:  &clientpb.Beacons{},
	}
	manager, _ := newTestManager(t, rpc)
	_, err := manager.executeBeacon(context.Background(), opforengine.AggressorBeaconExecutionRequest{
		Kind:            opforengine.AggressorBeaconInlineExecute,
		Name:            "beacon_inline_execute",
		BeaconID:        opforengine.String("old-session"),
		Content:         opforengine.BinaryString([]byte("OBJ")),
		EntryPoint:      opforengine.String("go"),
		PackedArguments: opforengine.BinaryString(nil),
	})
	if err == nil || !strings.Contains(err.Error(), "bof_v1") {
		t.Fatalf("executeBeacon error = %v, want bof_v1 rejection", err)
	}
	if calls, _ := rpc.snapshotCalls(); len(calls) != 0 {
		t.Fatalf("CallExtension called %d time(s) after capability rejection", len(calls))
	}
}

func TestExecuteBeaconRejectsUnsupportedArchitecture(t *testing.T) {
	rpc := &testRPC{
		sessions: &clientpb.Sessions{Sessions: []*clientpb.Session{{
			ID: "arm-session", Arch: "arm64", Capabilities: sliverpb.CapabilityBOFV1,
		}}},
		beacons: &clientpb.Beacons{},
	}
	manager, _ := newTestManager(t, rpc)
	_, err := manager.executeBeacon(context.Background(), opforengine.AggressorBeaconExecutionRequest{
		Kind:            opforengine.AggressorBeaconInlineExecute,
		Name:            "beacon_inline_execute",
		BeaconID:        opforengine.String("arm-session"),
		Content:         opforengine.BinaryString([]byte("OBJ")),
		EntryPoint:      opforengine.String("go"),
		PackedArguments: opforengine.BinaryString(nil),
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported BOF target architecture") {
		t.Fatalf("executeBeacon error = %v, want unsupported architecture rejection", err)
	}
	if calls, _ := rpc.snapshotCalls(); len(calls) != 0 {
		t.Fatalf("CallExtension called %d time(s) after architecture rejection", len(calls))
	}
}

func TestDynamicAliasRunEPropagatesBOFFailure(t *testing.T) {
	session := &clientpb.Session{ID: "old-session", OS: "windows", Arch: "amd64"}
	rpc := &testRPC{
		sessions: &clientpb.Sessions{Sessions: []*clientpb.Session{session}},
		beacons:  &clientpb.Beacons{},
	}
	manager, _ := newTestManager(t, rpc)
	manager.client.IsCLI = true
	manager.client.ActiveTarget.Set(session, nil)

	namespace := &cobra.Command{Use: "opfor"}
	manager.mu.Lock()
	manager.roots[namespace] = map[string]*cobra.Command{}
	manager.mu.Unlock()
	scriptPath := filepath.Join(t.TempDir(), "rejected.cna")
	if err := os.WriteFile(scriptPath, []byte(`
alias rejected {
    beacon_inline_execute($1, "OBJ", "go", $null);
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Load(context.Background(), scriptPath); err != nil {
		t.Fatalf("Load: %v", err)
	}
	command := directChild(namespace, "rejected")
	if command == nil || command.RunE == nil || command.Run != nil {
		t.Fatalf("dynamic alias handlers = command:%p Run:%v RunE:%v", command, command != nil && command.Run != nil, command != nil && command.RunE != nil)
	}
	if err := command.RunE(command, nil); err == nil || !strings.Contains(err.Error(), "bof_v1") {
		t.Fatalf("dynamic alias RunE error = %v, want propagated bof_v1 failure", err)
	}
}

type callbackRecorder struct {
	calls [][]opforengine.Value
}

func (recorder *callbackRecorder) call(_ context.Context, values ...opforengine.Value) (opforengine.Value, error) {
	recorder.calls = append(recorder.calls, append([]opforengine.Value(nil), values...))
	return opforengine.Null(), nil
}

func newAsyncBeaconTestRPC(t *testing.T, completed []byte) *testRPC {
	t.Helper()
	return &testRPC{
		sessions: &clientpb.Sessions{},
		beacons: &clientpb.Beacons{Beacons: []*clientpb.Beacon{{
			ID: "beacon-1", Arch: "amd64", Capabilities: sliverpb.CapabilityBOFV1,
		}}},
		call: func(context.Context, *sliverpb.CallExtensionReq) (*sliverpb.CallExtension, error) {
			return &sliverpb.CallExtension{Response: &commonpb.Response{
				Async: true, BeaconID: "beacon-1", TaskID: "task-1",
			}}, nil
		},
		task: func(_ context.Context, request *clientpb.BeaconTask) (*clientpb.BeaconTask, error) {
			if request.ID != "task-1" {
				t.Fatalf("task ID = %q, want task-1", request.ID)
			}
			return &clientpb.BeaconTask{ID: request.ID, State: "completed", Response: completed}, nil
		},
	}
}

func TestExecuteBeaconPollsAsyncTaskAndInvokesOrderedLegacyCallbacks(t *testing.T) {
	completed, err := proto.Marshal(&sliverpb.CallExtension{Output: []byte{'A', 0, 'B'}})
	if err != nil {
		t.Fatal(err)
	}
	rpc := newAsyncBeaconTestRPC(t, completed)
	manager, output := newTestManager(t, rpc)
	manager.client.Settings.BeaconAutoResults = false

	recorder := &callbackRecorder{}
	_, err = manager.executeBeacon(context.Background(), opforengine.AggressorBeaconExecutionRequest{
		Kind:            opforengine.AggressorBeaconInlineExecute,
		Name:            "beacon_inline_execute",
		BeaconID:        opforengine.String("beacon-1"),
		Content:         opforengine.BinaryString([]byte("OBJ")),
		EntryPoint:      opforengine.String("go"),
		PackedArguments: opforengine.BinaryString([]byte{1, 2}),
		Callback:        opforengine.CallableFunc(recorder.call),
	})
	if err != nil {
		t.Fatalf("executeBeacon: %v", err)
	}
	calls, taskCalls := rpc.snapshotCalls()
	assertAsyncBeaconRequest(t, calls, taskCalls, output)
	if len(recorder.calls) != 2 {
		t.Fatalf("callback calls = %d, want data plus terminal", len(recorder.calls))
	}
	assertLegacyDataCallback(t, recorder.calls[0])
	assertLegacyTerminalCallback(t, recorder.calls[1])
}

func assertAsyncBeaconRequest(t *testing.T, calls []*sliverpb.CallExtensionReq, taskCalls int, output *testOutput) {
	t.Helper()
	if len(calls) != 1 || taskCalls != 1 {
		t.Fatalf("CallExtension/task polls = %d/%d, want 1/1", len(calls), taskCalls)
	}
	if !calls[0].Request.Async || calls[0].Request.BeaconID != "beacon-1" || calls[0].Request.SessionID != "" {
		t.Fatalf("asynchronous target request = %#v", calls[0].Request)
	}
	if !bytes.Equal(calls[0].Args, []byte{2, 0, 0, 0, 1, 2}) {
		t.Fatalf("prefixed async arguments = %x", calls[0].Args)
	}
	if got := output.stdout(); got != "" {
		t.Fatalf("stdout = %q, want callback-owned output", got)
	}
}

func assertLegacyDataCallback(t *testing.T, dataCall []opforengine.Value) {
	t.Helper()
	if len(dataCall) != 3 || dataCall[0].String() != "beacon-1" {
		t.Fatalf("data callback values = %#v", dataCall)
	}
	callbackOutput, ok := dataCall[1].Bytes()
	if !ok || !dataCall[1].IsBinaryString() || !bytes.Equal(callbackOutput, []byte{'A', 0, 'B'}) {
		t.Fatalf("callback output = %x/binary:%v", callbackOutput, dataCall[1].IsBinaryString())
	}
	information, ok := dataCall[2].Hash()
	if !ok {
		t.Fatalf("callback information = %s, want hash", dataCall[2].Describe())
	}
	outputType, found := information.Get("type")
	if !found || outputType.String() != "output" {
		t.Fatalf("callback information type = %q/found:%v", outputType.String(), found)
	}
	assertCallbackInteger(t, information, "type_id", 0)
	assertCallbackInteger(t, information, "chunk_num", 1)
	assertCallbackFinal(t, information, true)
}

func assertLegacyTerminalCallback(t *testing.T, terminalCall []opforengine.Value) {
	t.Helper()
	if len(terminalCall) != 3 || terminalCall[0].String() != "beacon-1" {
		t.Fatalf("terminal callback values = %#v", terminalCall)
	}
	terminalOutput, ok := terminalCall[1].Bytes()
	if !ok || len(terminalOutput) != 0 {
		t.Fatalf("terminal callback output = %x/string:%v, want empty string", terminalOutput, ok)
	}
	terminalInformation, ok := terminalCall[2].Hash()
	if !ok {
		t.Fatalf("terminal callback information = %s, want hash", terminalCall[2].Describe())
	}
	terminalType, found := terminalInformation.Get("type")
	if !found || terminalType.String() != "task_completed" {
		t.Fatalf("terminal callback type = %q/found:%v", terminalType.String(), found)
	}
	taskID, found := terminalInformation.Get("taskid")
	if !found || taskID.String() != "task-1" {
		t.Fatalf("terminal callback taskid = %q/found:%v", taskID.String(), found)
	}
	if _, found := terminalInformation.Get("type_id"); found {
		t.Fatal("terminal callback unexpectedly contains type_id")
	}
	if _, found := terminalInformation.Get("chunk_num"); found {
		t.Fatal("terminal callback unexpectedly contains chunk_num")
	}
	if _, found := terminalInformation.Get("is_final"); found {
		t.Fatal("terminal callback unexpectedly contains is_final")
	}
}

func assertCallbackInteger(t *testing.T, information *opforengine.Hash, name string, want int32) {
	t.Helper()
	value, found := information.Get(name)
	if !found || value.String() != fmt.Sprint(want) {
		t.Fatalf("callback information %s = %q/found:%v, want %d", name, value.String(), found, want)
	}
}

func assertCallbackFinal(t *testing.T, information *opforengine.Hash, want bool) {
	t.Helper()
	value, found := information.Get("is_final")
	if !found {
		t.Fatal("callback information is_final is missing")
	}
	if got := !value.IsNull(); got != want {
		t.Fatalf("callback information is_final = %v, want %v", got, want)
	}
}

func TestExecuteBeaconInvokesTypedOutputBeforeLifecycleError(t *testing.T) {
	records := []*sliverpb.BOFOutput{
		{Type: 0x00, Data: []byte("first")},
		{Type: 0x0d, Data: []byte{'A', 0, 'B'}},
		{Type: 0x1e, Data: []byte{0xff}},
		{Type: 0x7f, Data: []byte("unknown")},
	}
	rpc := &testRPC{
		sessions: &clientpb.Sessions{Sessions: []*clientpb.Session{{
			ID: "session-typed", Arch: "amd64", Capabilities: sliverpb.CapabilityBOFV1,
		}}},
		beacons: &clientpb.Beacons{},
		call: func(context.Context, *sliverpb.CallExtensionReq) (*sliverpb.CallExtension, error) {
			return &sliverpb.CallExtension{
				Output:     []byte("legacy output must be ignored"),
				BOFOutputs: records,
				Response:   &commonpb.Response{Err: "entry point failed"},
			}, nil
		},
	}
	manager, output := newTestManager(t, rpc)
	recorder := &callbackRecorder{}

	_, err := manager.executeBeacon(context.Background(), opforengine.AggressorBeaconExecutionRequest{
		Kind:            opforengine.AggressorBeaconInlineExecute,
		Name:            "beacon_inline_execute",
		BeaconID:        opforengine.String("session-typed"),
		Content:         opforengine.BinaryString([]byte("OBJ")),
		EntryPoint:      opforengine.String("go"),
		PackedArguments: opforengine.BinaryString(nil),
		Callback:        opforengine.CallableFunc(recorder.call),
	})
	if err == nil || !strings.Contains(err.Error(), "entry point failed") {
		t.Fatalf("executeBeacon error = %v, want entry point failure", err)
	}
	if output.stdout() != "" || output.stderr() != "" {
		t.Fatalf("callback-owned output was also rendered: stdout=%q stderr=%q", output.stdout(), output.stderr())
	}
	assertTypedCallbacks(t, recorder.calls, records)
}

func assertTypedCallbacks(t *testing.T, calls [][]opforengine.Value, records []*sliverpb.BOFOutput) {
	t.Helper()
	if len(calls) != len(records)+1 {
		t.Fatalf("callback calls = %d, want %d records plus lifecycle error", len(calls), len(records)+1)
	}
	wantKinds := []string{"output", "error", "output", "output"}
	for index, record := range records {
		assertTypedOutputCallback(t, calls[index], record, wantKinds[index], index, len(records))
	}
	assertTypedLifecycleCallback(t, calls[len(records)])
}

func assertTypedOutputCallback(t *testing.T, call []opforengine.Value, record *sliverpb.BOFOutput, wantKind string, index, recordCount int) {
	t.Helper()
	if len(call) != 3 || call[0].String() != "session-typed" {
		t.Fatalf("callback %d values = %#v", index, call)
	}
	data, ok := call[1].Bytes()
	if !ok || !call[1].IsBinaryString() || !bytes.Equal(data, record.Data) {
		t.Fatalf("callback %d data = %x/binary:%v, want %x", index, data, call[1].IsBinaryString(), record.Data)
	}
	information, ok := call[2].Hash()
	if !ok {
		t.Fatalf("callback %d information = %s, want hash", index, call[2].Describe())
	}
	kind, found := information.Get("type")
	if !found || kind.String() != wantKind {
		t.Fatalf("callback %d type = %q/found:%v, want %q", index, kind.String(), found, wantKind)
	}
	assertCallbackInteger(t, information, "type_id", record.Type)
	assertCallbackInteger(t, information, "chunk_num", int32(index+1))
	assertCallbackFinal(t, information, index == recordCount-1)
}

func assertTypedLifecycleCallback(t *testing.T, lifecycle []opforengine.Value) {
	t.Helper()
	if len(lifecycle) != 3 || lifecycle[0].String() != "session-typed" {
		t.Fatalf("lifecycle callback values = %#v", lifecycle)
	}
	result, ok := lifecycle[1].Bytes()
	if !ok || !lifecycle[1].IsBinaryString() || !bytes.Contains(result, []byte("entry point failed")) {
		t.Fatalf("lifecycle error result = %q/binary:%v", result, lifecycle[1].IsBinaryString())
	}
	information, ok := lifecycle[2].Hash()
	if !ok {
		t.Fatalf("lifecycle information = %s, want hash", lifecycle[2].Describe())
	}
	kind, found := information.Get("type")
	if !found || kind.String() != "error" {
		t.Fatalf("lifecycle type = %q/found:%v, want error", kind.String(), found)
	}
	for _, name := range []string{"type_id", "chunk_num", "is_final", "taskid"} {
		if _, found := information.Get(name); found {
			t.Fatalf("lifecycle error unexpectedly contains %s", name)
		}
	}
}

func TestExecuteBeaconRendersTypedChannelsSafelyWithoutCallback(t *testing.T) {
	rpc := &testRPC{
		sessions: &clientpb.Sessions{Sessions: []*clientpb.Session{{
			ID: "session-render", Arch: "amd64", Capabilities: sliverpb.CapabilityBOFV1,
		}}},
		beacons: &clientpb.Beacons{},
		call: func(context.Context, *sliverpb.CallExtensionReq) (*sliverpb.CallExtension, error) {
			return &sliverpb.CallExtension{
				Output: []byte("legacy output must be ignored"),
				BOFOutputs: []*sliverpb.BOFOutput{
					{Type: 0x00, Data: []byte("plain")},
					{Type: 0x0d, Data: []byte("bad\x1b[31m")},
					{Type: 0x1e, Data: []byte{0xff, 0}},
					{Type: 0x20, Data: []byte(" snow \u2603\n")},
				},
			}, nil
		},
	}
	manager, output := newTestManager(t, rpc)
	_, err := manager.executeBeacon(context.Background(), opforengine.AggressorBeaconExecutionRequest{
		Kind:            opforengine.AggressorBeaconInlineExecute,
		Name:            "beacon_inline_execute",
		BeaconID:        opforengine.String("session-render"),
		Content:         opforengine.BinaryString([]byte("OBJ")),
		EntryPoint:      opforengine.String("go"),
		PackedArguments: opforengine.BinaryString(nil),
	})
	if err != nil {
		t.Fatalf("executeBeacon: %v", err)
	}
	if got, want := output.stdout(), "plain\\xff\\x00 snow \u2603\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got, want := output.stderr(), "bad\\x1b[31m"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestExecuteBeaconSyncEmptyResultInvokesSuccessLifecycle(t *testing.T) {
	rpc := &testRPC{
		sessions: &clientpb.Sessions{Sessions: []*clientpb.Session{{
			ID: "session-empty", Arch: "amd64", Capabilities: sliverpb.CapabilityBOFV1,
		}}},
		beacons: &clientpb.Beacons{},
		call: func(context.Context, *sliverpb.CallExtensionReq) (*sliverpb.CallExtension, error) {
			return &sliverpb.CallExtension{}, nil
		},
	}
	manager, output := newTestManager(t, rpc)
	var callbackCalls [][]opforengine.Value
	callback := opforengine.CallableFunc(func(_ context.Context, values ...opforengine.Value) (opforengine.Value, error) {
		callbackCalls = append(callbackCalls, append([]opforengine.Value(nil), values...))
		return opforengine.Null(), nil
	})
	_, err := manager.executeBeacon(context.Background(), opforengine.AggressorBeaconExecutionRequest{
		Kind:            opforengine.AggressorBeaconInlineExecute,
		Name:            "beacon_inline_execute",
		BeaconID:        opforengine.String("session-empty"),
		Content:         opforengine.BinaryString([]byte("OBJ")),
		EntryPoint:      opforengine.String("go"),
		PackedArguments: opforengine.BinaryString(nil),
		Callback:        callback,
	})
	if err != nil {
		t.Fatalf("executeBeacon: %v", err)
	}
	if output.stdout() != "" || output.stderr() != "" {
		t.Fatalf("empty result rendered output: stdout=%q stderr=%q", output.stdout(), output.stderr())
	}
	if len(callbackCalls) != 1 {
		t.Fatalf("callback calls = %d, want one success lifecycle", len(callbackCalls))
	}
	result, ok := callbackCalls[0][1].Bytes()
	if !ok || len(result) != 0 {
		t.Fatalf("success result = %x/string:%v, want empty", result, ok)
	}
	information, ok := callbackCalls[0][2].Hash()
	if !ok {
		t.Fatalf("success information = %s, want hash", callbackCalls[0][2].Describe())
	}
	kind, found := information.Get("type")
	if !found || kind.String() != "success" {
		t.Fatalf("success type = %q/found:%v", kind.String(), found)
	}
	for _, name := range []string{"type_id", "chunk_num", "is_final", "taskid"} {
		if _, found := information.Get(name); found {
			t.Fatalf("success lifecycle unexpectedly contains %s", name)
		}
	}
}

func TestExecuteBeaconDoesNotReinvokeFailingDataCallback(t *testing.T) {
	rpc := &testRPC{
		sessions: &clientpb.Sessions{Sessions: []*clientpb.Session{{
			ID: "session-callback-error", Arch: "amd64", Capabilities: sliverpb.CapabilityBOFV1,
		}}},
		beacons: &clientpb.Beacons{},
		call: func(context.Context, *sliverpb.CallExtensionReq) (*sliverpb.CallExtension, error) {
			return &sliverpb.CallExtension{
				BOFOutputs: []*sliverpb.BOFOutput{{Data: []byte("partial")}},
				Response:   &commonpb.Response{Err: "implant failure"},
			}, nil
		},
	}
	manager, output := newTestManager(t, rpc)
	callbackCalls := 0
	callback := opforengine.CallableFunc(func(context.Context, ...opforengine.Value) (opforengine.Value, error) {
		callbackCalls++
		return opforengine.Null(), errors.New("script callback failure")
	})
	_, err := manager.executeBeacon(context.Background(), opforengine.AggressorBeaconExecutionRequest{
		Kind:            opforengine.AggressorBeaconInlineExecute,
		Name:            "beacon_inline_execute",
		BeaconID:        opforengine.String("session-callback-error"),
		Content:         opforengine.BinaryString([]byte("OBJ")),
		EntryPoint:      opforengine.String("go"),
		PackedArguments: opforengine.BinaryString(nil),
		Callback:        callback,
	})
	if err == nil || !strings.Contains(err.Error(), "implant failure") || !strings.Contains(err.Error(), "script callback failure") {
		t.Fatalf("executeBeacon error = %v, want joined implant and callback failures", err)
	}
	if callbackCalls != 1 {
		t.Fatalf("callback calls = %d, want no retry after data callback failure", callbackCalls)
	}
	if output.stdout() != "" || output.stderr() != "" {
		t.Fatalf("callback-owned output was also rendered: stdout=%q stderr=%q", output.stdout(), output.stderr())
	}
}

func TestNotifyBOFLifecycleErrorSkipsCanceledCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	callbackCalls := 0
	callback := opforengine.CallableFunc(func(context.Context, ...opforengine.Value) (opforengine.Value, error) {
		callbackCalls++
		return opforengine.Null(), nil
	})
	executionErr := errors.New("polling canceled")
	got := notifyBOFLifecycleError(ctx, opforengine.AggressorBeaconExecutionRequest{
		BeaconID: opforengine.String("beacon-canceled"),
		Callback: callback,
	}, "task-canceled", executionErr)
	if !errors.Is(got, executionErr) {
		t.Fatalf("notifyBOFLifecycleError = %v, want original cancellation error", got)
	}
	if callbackCalls != 0 {
		t.Fatalf("callback calls = %d, want none after context cancellation", callbackCalls)
	}
}

type testPromptResponder struct {
	accepted   [][]opforengine.Value
	dismissed  bool
	done       chan struct{}
	acceptFunc func(context.Context, ...opforengine.Value) (opforengine.Value, error)
}

func (responder *testPromptResponder) Accept(ctx context.Context, values ...opforengine.Value) (opforengine.Value, error) {
	if responder.acceptFunc != nil {
		return responder.acceptFunc(ctx, values...)
	}
	responder.accepted = append(responder.accepted, append([]opforengine.Value(nil), values...))
	return opforengine.Null(), nil
}

func (responder *testPromptResponder) Dismiss() error {
	responder.dismissed = true
	return nil
}

func (responder *testPromptResponder) Done() <-chan struct{} {
	if responder.done == nil {
		responder.done = make(chan struct{})
	}
	return responder.done
}

func TestPromptAdapterUsesInjectedUI(t *testing.T) {
	t.Run("confirm accepts without callback values", func(t *testing.T) {
		ui := &testPromptUI{confirmAnswer: true}
		responder := &testPromptResponder{}
		manager := &Manager{ui: ui}
		err := manager.presentPrompt(context.Background(), opforengine.AggressorPromptPresentation{
			Kind: opforengine.AggressorPromptConfirm, Title: "Confirm", Text: "Continue?",
		}, responder)
		if err != nil {
			t.Fatal(err)
		}
		if ui.confirmTitle != "Confirm: Continue?" || len(responder.accepted) != 1 || len(responder.accepted[0]) != 0 || responder.dismissed {
			t.Fatalf("confirm UI/responder = title:%q accepted:%#v dismissed:%v", ui.confirmTitle, responder.accepted, responder.dismissed)
		}
	})

	t.Run("confirm false dismisses", func(t *testing.T) {
		ui := &testPromptUI{}
		responder := &testPromptResponder{}
		manager := &Manager{ui: ui}
		if err := manager.presentPrompt(context.Background(), opforengine.AggressorPromptPresentation{
			Kind: opforengine.AggressorPromptConfirm, Text: "Continue?",
		}, responder); err != nil {
			t.Fatal(err)
		}
		if !responder.dismissed || len(responder.accepted) != 0 {
			t.Fatalf("responder = accepted:%#v dismissed:%v", responder.accepted, responder.dismissed)
		}
	})

	t.Run("text preserves default and accepts answer", func(t *testing.T) {
		ui := &testPromptUI{inputAnswer: "answer"}
		responder := &testPromptResponder{}
		manager := &Manager{ui: ui}
		if err := manager.presentPrompt(context.Background(), opforengine.AggressorPromptPresentation{
			Kind: opforengine.AggressorPromptText, Text: "Value", Default: opforengine.String("default"), HasDefault: true,
		}, responder); err != nil {
			t.Fatal(err)
		}
		if ui.inputTitle != "Value" || ui.inputDefault != "default" || len(responder.accepted) != 1 || len(responder.accepted[0]) != 1 || responder.accepted[0][0].String() != "answer" {
			t.Fatalf("text UI/responder = title:%q default:%q accepted:%#v", ui.inputTitle, ui.inputDefault, responder.accepted)
		}
	})
}

func TestPromptAdapterReleasesUILockBeforeResponder(t *testing.T) {
	manager := &Manager{ui: &testPromptUI{inputAnswer: "answer"}}
	inner := &testPromptResponder{}
	outer := &testPromptResponder{acceptFunc: func(ctx context.Context, _ ...opforengine.Value) (opforengine.Value, error) {
		err := manager.presentPrompt(ctx, opforengine.AggressorPromptPresentation{
			Kind: opforengine.AggressorPromptText,
			Text: "Nested prompt",
		}, inner)
		return opforengine.Null(), err
	}}

	done := make(chan error, 1)
	go func() {
		done <- manager.presentPrompt(context.Background(), opforengine.AggressorPromptPresentation{
			Kind: opforengine.AggressorPromptText,
			Text: "Outer prompt",
		}, outer)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
		if len(inner.accepted) != 1 {
			t.Fatalf("nested responder accepted %d time(s), want 1", len(inner.accepted))
		}
	case <-time.After(time.Second):
		t.Fatal("nested prompt deadlocked while responder resumed the script")
	}
}
