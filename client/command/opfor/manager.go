//go:build client

package opfor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/console"
	opforengine "github.com/sliverarmory/opfor"
	"github.com/spf13/cobra"
)

// Manager owns one long-lived OPFOR runtime for a Sliver client. Keeping the
// runtime alive is important: CNA aliases, hooks, and callbacks are scoped to
// the Script returned by Runtime.Load.
type Manager struct {
	client *console.SliverClient
	output clientOutput

	runtime *opforengine.Runtime
	ui      promptUI

	loadMu    sync.Mutex
	commandMu sync.Mutex
	promptMu  sync.Mutex
	mu        sync.RWMutex
	scripts   map[string]*opforengine.Script
	aliases   map[string][]opforengine.Binding
	roots     map[*cobra.Command]map[string]*cobra.Command
}

type managerEntry struct {
	manager *Manager
	err     error
}

const oneShotRuntimeCloseTimeout = 5 * time.Second

var (
	clientManagersMu sync.Mutex
	clientManagers   = map[*console.SliverClient]managerEntry{}
)

func managerFor(client *console.SliverClient) (*Manager, error) {
	if client == nil {
		return nil, errors.New("opfor: Sliver client is nil")
	}

	clientManagersMu.Lock()
	defer clientManagersMu.Unlock()
	if entry, found := clientManagers[client]; found {
		return entry.manager, entry.err
	}

	manager, err := newManager(client, formsPromptUI{})
	clientManagers[client] = managerEntry{manager: manager, err: err}
	return manager, err
}

func newManager(client *console.SliverClient, ui promptUI) (*Manager, error) {
	return newManagerWithOutput(client, ui, client)
}

func newManagerWithOutput(client *console.SliverClient, ui promptUI, output clientOutput) (*Manager, error) {
	if client == nil {
		return nil, errors.New("opfor: Sliver client is nil")
	}
	if ui == nil {
		return nil, errors.New("opfor: prompt UI is nil")
	}
	if output == nil {
		return nil, errors.New("opfor: output sink is nil")
	}

	manager := &Manager{
		client:  client,
		output:  output,
		ui:      ui,
		scripts: map[string]*opforengine.Script{},
		aliases: map[string][]opforengine.Binding{},
		roots:   map[*cobra.Command]map[string]*cobra.Command{},
	}

	runtime, err := manager.newRuntime(true)
	if err != nil {
		return nil, fmt.Errorf("opfor: create runtime: %w", err)
	}
	manager.runtime = runtime
	return manager, nil
}

func (manager *Manager) newRuntime(observeBindings bool) (*opforengine.Runtime, error) {
	stdout := clientWriter{output: manager.output}
	stderr := clientWriter{output: manager.output, stderr: true}
	options := []opforengine.Option{
		opforengine.WithBOFPackByteOrder(opforengine.BOFPackLittleEndian),
		opforengine.WithAggressorBeaconExecutionProvider(
			opforengine.AggressorBeaconExecutionProviderFunc(manager.executeBeacon),
		),
		opforengine.WithAggressorSessionQueryProvider(
			opforengine.AggressorSessionQueryProviderFunc(manager.querySession),
		),
		opforengine.WithAggressorBeaconTranscriptSink(
			opforengine.AggressorBeaconTranscriptSinkFunc(manager.publishTranscript),
		),
		opforengine.WithAggressorPromptProvider(
			opforengine.AggressorPromptProviderFunc(manager.presentPrompt),
		),
		opforengine.WithStdout(stdout),
		opforengine.WithStderr(stderr),
	}
	if observeBindings {
		options = append(options, opforengine.WithBindingObserver(manager))
	}
	return opforengine.New(options...)
}

// Registered observes declarations published by loaded scripts. Only Beacon
// aliases become Sliver commands; other binding families remain available to
// OPFOR internally for hooks, events, and callback composition.
func (manager *Manager) Registered(_ context.Context, binding opforengine.Binding) error {
	if binding.Kind != opforengine.BindingAlias {
		return nil
	}
	manager.mu.Lock()
	manager.aliases[binding.Name] = append(manager.aliases[binding.Name], binding)
	manager.mu.Unlock()
	return nil
}

// Unregistered removes the exact binding generation which retired. An older
// alias with the same name remains visible when scripts were layered.
func (manager *Manager) Unregistered(_ context.Context, binding opforengine.Binding) error {
	if binding.Kind != opforengine.BindingAlias {
		return nil
	}
	manager.mu.Lock()
	bindings := manager.aliases[binding.Name]
	for index := len(bindings) - 1; index >= 0; index-- {
		if bindings[index].ID == binding.ID && bindings[index].Script == binding.Script {
			bindings = append(bindings[:index], bindings[index+1:]...)
			break
		}
	}
	if len(bindings) == 0 {
		delete(manager.aliases, binding.Name)
	} else {
		manager.aliases[binding.Name] = bindings
	}
	manager.mu.Unlock()
	return nil
}

// Load compiles and executes a CNA, retaining its registrations until unload.
func (manager *Manager) Load(ctx context.Context, path string) (string, error) {
	absolute, content, err := readCNAFile(path)
	if err != nil {
		return "", err
	}

	manager.loadMu.Lock()
	defer manager.loadMu.Unlock()

	manager.mu.RLock()
	_, exists := manager.scripts[absolute]
	manager.mu.RUnlock()
	if exists {
		return "", fmt.Errorf("opfor: script is already loaded: %s", absolute)
	}

	program, err := manager.runtime.Compile(opforengine.NewSource(absolute, content))
	if err != nil {
		return "", fmt.Errorf("opfor: compile %s: %w", absolute, err)
	}
	script, err := manager.runtime.Load(ctx, program)
	if err != nil {
		return "", fmt.Errorf("opfor: load %s: %w", absolute, err)
	}

	manager.mu.Lock()
	manager.scripts[absolute] = script
	manager.mu.Unlock()
	manager.syncAllRoots()
	return absolute, nil
}

// Check compiles a CNA without creating a Script or executing top-level code.
// Standalone compilation intentionally mirrors the OPFOR CLI check command and
// avoids charging the long-lived Sliver runtime's source budget.
func (manager *Manager) Check(path string) (string, error) {
	absolute, _, err := compileCNAFile(path)
	return absolute, err
}

// Run executes a CNA once with arguments exposed through Sleep's @ARGV. Any
// aliases, hooks, callbacks, or events registered by the script are retired
// before this method returns, matching the OPFOR CLI run command.
func (manager *Manager) Run(ctx context.Context, path string, arguments []string) (string, error) {
	absolute, program, err := compileCNAFile(path)
	if err != nil {
		return "", err
	}
	values := make([]opforengine.Value, len(arguments))
	for index, argument := range arguments {
		values[index] = opforengine.String(argument)
	}

	// A one-shot runtime matches the standalone OPFOR CLI and prevents
	// temporary registrations from becoming visible to the persistent Sliver
	// manager. It retains all Sliver providers but deliberately omits the
	// binding observer used to publish persistent aliases.
	runtime, err := manager.newRuntime(false)
	if err != nil {
		return "", fmt.Errorf("opfor: create runtime: %w", err)
	}
	_, executeErr := runtime.Execute(ctx, program, values...)
	closeCtx, cancel := context.WithTimeout(context.Background(), oneShotRuntimeCloseTimeout)
	closeErr := runtime.Close(closeCtx)
	cancel()
	if executeErr != nil || closeErr != nil {
		var runErr error
		if executeErr != nil {
			runErr = fmt.Errorf("opfor: execute %s: %w", absolute, executeErr)
		}
		if closeErr != nil {
			runErr = errors.Join(runErr, fmt.Errorf("opfor: close runtime: %w", closeErr))
		}
		return "", runErr
	}
	return absolute, nil
}

// Unload retires a loaded CNA and all registrations owned by that script.
func (manager *Manager) Unload(ctx context.Context, path string) (string, error) {
	manager.loadMu.Lock()
	defer manager.loadMu.Unlock()

	absolute, script, err := manager.findScript(path)
	if err != nil {
		return "", err
	}
	if err := script.Unload(ctx); err != nil {
		return "", fmt.Errorf("opfor: unload %s: %w", absolute, err)
	}
	manager.mu.Lock()
	delete(manager.scripts, absolute)
	manager.mu.Unlock()
	manager.syncAllRoots()
	return absolute, nil
}

func (manager *Manager) findScript(path string) (string, *opforengine.Script, error) {
	absolute, absoluteErr := filepath.Abs(path)
	if absoluteErr == nil {
		absolute = filepath.Clean(absolute)
	}

	manager.mu.RLock()
	defer manager.mu.RUnlock()
	if absoluteErr == nil {
		if script := manager.scripts[absolute]; script != nil {
			return absolute, script, nil
		}
	}

	base := filepath.Base(path)
	var matchedPath string
	var matchedScript *opforengine.Script
	for candidate, script := range manager.scripts {
		if filepath.Base(candidate) != base {
			continue
		}
		if matchedScript != nil {
			return "", nil, fmt.Errorf("opfor: script name %q is ambiguous; use an absolute path", path)
		}
		matchedPath, matchedScript = candidate, script
	}
	if matchedScript == nil {
		return "", nil, fmt.Errorf("opfor: script is not loaded: %s", path)
	}
	return matchedPath, matchedScript, nil
}

// Paths returns the sorted absolute paths of all loaded CNA scripts.
func (manager *Manager) Paths() []string {
	manager.mu.RLock()
	paths := make([]string, 0, len(manager.scripts))
	for path := range manager.scripts {
		paths = append(paths, path)
	}
	manager.mu.RUnlock()
	sort.Strings(paths)
	return paths
}

func (manager *Manager) aliasNames() []string {
	manager.mu.RLock()
	names := make([]string, 0, len(manager.aliases))
	for name, bindings := range manager.aliases {
		if len(bindings) != 0 {
			names = append(names, name)
		}
	}
	manager.mu.RUnlock()
	sort.Strings(names)
	return names
}

func (manager *Manager) aliasBindingCounts() map[string]int {
	manager.mu.RLock()
	counts := make(map[string]int, len(manager.aliases))
	for name, bindings := range manager.aliases {
		if len(bindings) != 0 {
			counts[name] = len(bindings)
		}
	}
	manager.mu.RUnlock()
	return counts
}

func (manager *Manager) hasAlias(name string) bool {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	return len(manager.aliases[name]) != 0
}

func (manager *Manager) aliasMetadata(name string) (opforengine.AggressorCommandMetadata, error) {
	if reservedManagementAlias(name) || !manager.hasAlias(name) {
		return opforengine.AggressorCommandMetadata{}, fmt.Errorf("opfor: unknown CNA alias %q", name)
	}
	catalog, err := manager.runtime.SnapshotAggressorCommandCatalog(opforengine.AggressorCommandBeacon)
	if err != nil {
		return opforengine.AggressorCommandMetadata{}, fmt.Errorf("opfor: snapshot Beacon command help: %w", err)
	}
	for _, metadata := range catalog.Commands {
		if metadata.Name == name {
			return metadata, nil
		}
	}
	return opforengine.AggressorCommandMetadata{Name: name}, nil
}

func absoluteCNAPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("opfor: CNA script path is empty")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("opfor: resolve %s: %w", path, err)
	}
	absolute = filepath.Clean(absolute)
	if !strings.EqualFold(filepath.Ext(absolute), ".cna") {
		return "", fmt.Errorf("opfor: script must have a .cna extension: %s", absolute)
	}
	return absolute, nil
}

func readCNAFile(path string) (string, []byte, error) {
	absolute, err := absoluteCNAPath(path)
	if err != nil {
		return "", nil, err
	}
	content, err := os.ReadFile(absolute)
	if err != nil {
		return "", nil, fmt.Errorf("opfor: read %s: %w", absolute, err)
	}
	return absolute, content, nil
}

func compileCNAFile(path string) (string, *opforengine.Program, error) {
	absolute, content, err := readCNAFile(path)
	if err != nil {
		return "", nil, err
	}
	program, err := opforengine.Compile(opforengine.NewSource(absolute, content))
	if err != nil {
		return "", nil, fmt.Errorf("opfor: compile %s: %w", absolute, err)
	}
	return absolute, program, nil
}

type clientWriter struct {
	output clientOutput
	stderr bool
}

func (writer clientWriter) Write(data []byte) (int, error) {
	if writer.output == nil || len(data) == 0 {
		return len(data), nil
	}
	if writer.stderr {
		writer.output.PrintErrorf("%s", data)
	} else {
		writer.output.Printf("%s", data)
	}
	return len(data), nil
}

type clientOutput interface {
	Printf(string, ...any)
	PrintInfof(string, ...any)
	PrintSuccessf(string, ...any)
	PrintErrorf(string, ...any)
}

var _ io.Writer = clientWriter{}
var _ opforengine.BindingObserver = (*Manager)(nil)
