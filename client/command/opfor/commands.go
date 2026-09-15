//go:build client

// Package opfor integrates OPFOR-compatible CNA scripting with the Sliver client.
package opfor

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	opforengine "github.com/sliverarmory/opfor"
	"github.com/spf13/cobra"
)

const aliasInvocationTimeout = 10 * time.Minute

// Commands returns the OPFOR script manager for the implant menu.
func Commands(client *console.SliverClient) []*cobra.Command {
	return []*cobra.Command{managementCommand(client, consts.SliverCoreHelpGroup)}
}

// ServerCommands returns the same persistent script manager in the server
// menu. CNA Beacon aliases themselves are registered only on implant roots.
func ServerCommands(client *console.SliverClient) []*cobra.Command {
	return []*cobra.Command{managementCommand(client, consts.GenericHelpGroup)}
}

func managementCommand(client *console.SliverClient, groupID string) *cobra.Command {
	command := &cobra.Command{
		Use:     consts.OpforStr + " [script.cna]",
		Short:   "Load and manage OPFOR CNA scripts",
		Long:    help.GetHelpFor([]string{consts.OpforStr}),
		GroupID: groupID,
		Args:    cobra.ArbitraryArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runManagementCommand(command, client, args)
		},
	}
	command.PersistentFlags().Int64P(
		"timeout",
		"t",
		int64(aliasInvocationTimeout/time.Second),
		"OPFOR operation timeout in seconds (place before a dynamic alias name)",
	)
	// The first non-flag token is a CNA alias (or convenience script path).
	// Everything after it belongs to the script, including flag-like tokens.
	command.Flags().SetInterspersed(false)

	load := &cobra.Command{
		Use:   "load <script.cna>",
		Short: "Execute and retain a CNA's aliases, hooks, and callbacks",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			return loadScriptCommand(command, client, args[0])
		},
	}
	run := &cobra.Command{
		Use:   "run <script.cna> [arguments...]",
		Short: "Execute a CNA once and discard its registrations",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			return runScriptCommand(command, client, args[0], args[1:])
		},
	}
	// Tokens after the script path belong to @ARGV, including flag-like ones.
	run.Flags().SetInterspersed(false)
	check := &cobra.Command{
		Use:   "check <script.cna>",
		Short: "Compile a CNA without executing it",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			return checkScriptCommand(command, client, args[0])
		},
	}
	aliasHelp := &cobra.Command{
		Use:   "help <alias>",
		Short: "Show help registered for a loaded CNA alias",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			return helpAliasCommand(command, client, args[0])
		},
	}
	unload := &cobra.Command{
		Use:   "unload <script.cna>",
		Short: "Unload a CNA script and retire its aliases",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			return unloadScriptCommand(command, client, args[0])
		},
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "List loaded CNA scripts by absolute path",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return listScriptsCommand(command, client)
		},
	}
	command.AddCommand(check, aliasHelp, load, unload, list, run)
	return command
}

func runManagementCommand(command *cobra.Command, client *console.SliverClient, args []string) error {
	if _, err := configuredCommandTimeout(command); err != nil {
		return err
	}
	if len(args) == 0 {
		return command.Help()
	}
	if len(args) == 1 && strings.EqualFold(filepath.Ext(args[0]), ".cna") {
		return loadScriptCommand(command, client, args[0])
	}

	manager, err := managerFor(client)
	if err != nil {
		return err
	}
	if manager.hasAlias(args[0]) {
		return manager.runAlias(command, args[0], args[1:])
	}
	return fmt.Errorf("opfor: unknown CNA alias %q", args[0])
}

func loadScriptCommand(command *cobra.Command, client *console.SliverClient, path string) error {
	manager, err := managerFor(client)
	if err != nil {
		return err
	}
	previousAliases := manager.aliasBindingCounts()
	ctx, cancel, err := commandTimeoutContext(command)
	if err != nil {
		return err
	}
	defer cancel()
	absolute, err := manager.Load(ctx, path)
	if err != nil {
		return err
	}
	manager.output.PrintSuccessf("Loaded CNA script %s\n", absolute)
	currentAliases := manager.aliasBindingCounts()
	aliases := make([]string, 0)
	for _, alias := range manager.aliasNames() {
		if currentAliases[alias] > previousAliases[alias] && !reservedManagementAlias(alias) {
			aliases = append(aliases, alias)
		}
	}
	if len(aliases) == 0 {
		manager.output.PrintInfof("The script is retained for this client process; it registered no invokable Beacon aliases\n")
		return nil
	}
	manager.output.PrintInfof("Registered CNA aliases:\n")
	for _, alias := range aliases {
		manager.output.Printf("  opfor %s\n", alias)
	}
	manager.output.PrintInfof("View CNA alias help:\n")
	for _, alias := range aliases {
		manager.output.Printf("  opfor help %s\n", alias)
	}
	return nil
}

func reservedManagementAlias(name string) bool {
	switch name {
	case "check", "help", "list", "load", "run", "unload":
		return true
	default:
		return false
	}
}

func runScriptCommand(command *cobra.Command, client *console.SliverClient, path string, arguments []string) error {
	manager, err := managerFor(client)
	if err != nil {
		return err
	}
	ctx, cancel, err := commandTimeoutContext(command)
	if err != nil {
		return err
	}
	defer cancel()
	_, err = manager.Run(ctx, path, arguments)
	return err
}

func checkScriptCommand(command *cobra.Command, client *console.SliverClient, path string) error {
	if _, err := configuredCommandTimeout(command); err != nil {
		return err
	}
	manager, err := managerFor(client)
	if err != nil {
		return err
	}
	if _, err := manager.Check(path); err != nil {
		return err
	}
	manager.output.Printf("%s: ok\n", path)
	return nil
}

func helpAliasCommand(command *cobra.Command, client *console.SliverClient, name string) error {
	if _, err := configuredCommandTimeout(command); err != nil {
		return err
	}
	manager, err := managerFor(client)
	if err != nil {
		return err
	}
	metadata, err := manager.aliasMetadata(name)
	if err != nil {
		return err
	}
	description := strings.TrimSpace(metadata.Description)
	if description == "" {
		description = "CNA-defined Beacon alias"
	}
	manager.output.Printf("Command: opfor %s [arguments...]\n\n%s\n", name, description)
	if detail := strings.TrimSpace(metadata.Detail); detail != "" {
		manager.output.Printf("\n%s\n", detail)
	}
	return nil
}

func unloadScriptCommand(command *cobra.Command, client *console.SliverClient, path string) error {
	manager, err := managerFor(client)
	if err != nil {
		return err
	}
	ctx, cancel, err := commandTimeoutContext(command)
	if err != nil {
		return err
	}
	defer cancel()
	absolute, err := manager.Unload(ctx, path)
	if err != nil {
		return err
	}
	manager.output.PrintSuccessf("Unloaded CNA script %s\n", absolute)
	return nil
}

func listScriptsCommand(command *cobra.Command, client *console.SliverClient) error {
	if _, err := configuredCommandTimeout(command); err != nil {
		return err
	}
	manager, err := managerFor(client)
	if err != nil {
		return err
	}
	paths := manager.Paths()
	if len(paths) == 0 {
		manager.output.PrintInfof("No CNA scripts are loaded\n")
		return nil
	}
	manager.output.PrintInfof("Loaded CNA scripts:\n")
	for _, path := range paths {
		manager.output.Printf("  %s\n", path)
	}
	return nil
}

// RegisterCommands rehydrates the active CNA Beacon aliases beneath the
// implant menu's opfor namespace. Keeping script declarations namespaced avoids
// collisions with built-ins such as Sliver's own cat command.
func RegisterCommands(root *cobra.Command, client *console.SliverClient) {
	if root == nil || client == nil {
		return
	}
	manager, err := managerFor(client)
	if err != nil {
		client.PrintErrorf("%s\n", err)
		return
	}
	namespace := directChild(root, consts.OpforStr)
	if namespace == nil {
		return
	}
	manager.commandMu.Lock()
	manager.mu.Lock()
	for previous := range manager.roots {
		if previous == namespace {
			continue
		}
		// A replaced menu root may still be executing asynchronously. Stop
		// retaining it, but leave its Cobra tree intact for that execution.
		delete(manager.roots, previous)
	}
	if manager.roots[namespace] == nil {
		manager.roots[namespace] = make(map[string]*cobra.Command)
	}
	manager.mu.Unlock()
	manager.syncRootLocked(namespace)
	manager.commandMu.Unlock()
}

func (manager *Manager) syncAllRoots() {
	manager.commandMu.Lock()
	defer manager.commandMu.Unlock()

	manager.mu.RLock()
	roots := make([]*cobra.Command, 0, len(manager.roots))
	for root := range manager.roots {
		roots = append(roots, root)
	}
	manager.mu.RUnlock()
	for _, root := range roots {
		manager.syncRootLocked(root)
	}
}

func (manager *Manager) syncRootLocked(root *cobra.Command) {
	names := manager.aliasNames()
	wanted := make(map[string]struct{}, len(names))
	for _, name := range names {
		wanted[name] = struct{}{}
	}

	catalog, _ := manager.runtime.SnapshotAggressorCommandCatalog(opforengine.AggressorCommandBeacon)
	help := make(map[string]opforengine.AggressorCommandMetadata, len(catalog.Commands))
	for _, metadata := range catalog.Commands {
		help[metadata.Name] = metadata
	}

	manager.mu.Lock()
	attached := manager.roots[root]
	if attached == nil {
		attached = map[string]*cobra.Command{}
		manager.roots[root] = attached
	}
	for name, command := range attached {
		if _, keep := wanted[name]; keep {
			continue
		}
		root.RemoveCommand(command)
		delete(attached, name)
	}
	manager.mu.Unlock()

	for _, name := range names {
		metadata := help[name]
		manager.mu.RLock()
		existing := attached[name]
		manager.mu.RUnlock()
		if existing != nil {
			applyAliasHelp(existing, metadata)
			continue
		}
		if directChild(root, name) != nil {
			// A built-in/Armory command wins over an untrusted script declaration.
			continue
		}

		aliasName := name
		command := &cobra.Command{
			Use:                aliasName + " [arguments...]",
			DisableFlagParsing: true,
			RunE: func(command *cobra.Command, args []string) error {
				return manager.runAlias(command, aliasName, args)
			},
		}
		applyAliasHelp(command, metadata)
		root.AddCommand(command)
		manager.mu.Lock()
		attached[aliasName] = command
		manager.mu.Unlock()
	}
}

func directChild(root *cobra.Command, name string) *cobra.Command {
	for _, child := range root.Commands() {
		if child.Name() == name {
			return child
		}
	}
	return nil
}

func applyAliasHelp(command *cobra.Command, metadata opforengine.AggressorCommandMetadata) {
	command.Short = metadata.Description
	if command.Short == "" {
		command.Short = "CNA-defined Beacon alias"
	}
	command.Long = strings.TrimSpace(strings.Join([]string{metadata.Description, metadata.Detail}, "\n\n"))
	if command.Long == "" {
		command.Long = command.Short
	}
}

func (manager *Manager) runAlias(command *cobra.Command, name string, args []string) error {
	args, timeoutOverride, err := consumeAliasHostArguments(args)
	if err != nil {
		return err
	}
	ctx, cancel, err := commandTimeoutContextWithOverride(command, timeoutOverride)
	if err != nil {
		return err
	}
	defer cancel()
	session, beacon := manager.client.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return errorsNewTargetRequired(name)
	}
	targetID := ""
	if session != nil {
		targetID = session.ID
	} else {
		targetID = beacon.ID
	}

	rawInput := name
	if len(args) != 0 {
		rawInput += " " + strings.Join(args, " ")
	}
	if err := manager.invokeAlias(ctx, name, rawInput, args, targetID); err != nil {
		return fmt.Errorf("OPFOR alias %s failed: %w", name, err)
	}
	return nil
}

func consumeAliasHostArguments(args []string) ([]string, time.Duration, error) {
	remaining := args
	var timeout time.Duration
	for len(remaining) != 0 {
		var value string
		switch argument := remaining[0]; {
		case argument == "--":
			return remaining[1:], timeout, nil
		case argument == "--timeout" || argument == "-t":
			if len(remaining) < 2 {
				return nil, 0, fmt.Errorf("opfor: %s requires a timeout in seconds", argument)
			}
			value = remaining[1]
			remaining = remaining[2:]
		case strings.HasPrefix(argument, "--timeout="):
			value = strings.TrimPrefix(argument, "--timeout=")
			remaining = remaining[1:]
		case strings.HasPrefix(argument, "-t="):
			value = strings.TrimPrefix(argument, "-t=")
			remaining = remaining[1:]
		default:
			return remaining, timeout, nil
		}

		parsed, err := parseAliasTimeout(value)
		if err != nil {
			return nil, 0, err
		}
		timeout = parsed
	}
	return remaining, timeout, nil
}

func parseAliasTimeout(value string) (time.Duration, error) {
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || seconds <= 0 || seconds > math.MaxInt64/int64(time.Second) {
		return 0, fmt.Errorf("opfor: invalid timeout %q; expected positive seconds", value)
	}
	return time.Duration(seconds) * time.Second, nil
}

func (manager *Manager) invokeAlias(ctx context.Context, name, rawInput string, arguments []string, targetID string) error {
	if targetID == "" {
		return errorsNewTargetRequired(name)
	}
	parsedArguments := make([]string, len(arguments))
	copy(parsedArguments, arguments)
	_, err := manager.runtime.InvokeConsole(ctx, opforengine.ConsoleInvocation{
		Kind:            opforengine.BindingAlias,
		Name:            name,
		RawInput:        rawInput,
		ParsedArguments: parsedArguments,
		SessionID:       opforengine.String(targetID),
	})
	return err
}

func errorsNewTargetRequired(name string) error {
	return fmt.Errorf("opfor: alias %q requires an active target", name)
}

func commandTimeoutContext(command *cobra.Command) (context.Context, context.CancelFunc, error) {
	return commandTimeoutContextWithOverride(command, 0)
}

func commandTimeoutContextWithOverride(command *cobra.Command, override time.Duration) (context.Context, context.CancelFunc, error) {
	parent := context.Background()
	if command != nil && command.Context() != nil {
		parent = command.Context()
	}
	timeout, err := configuredCommandTimeout(command)
	if err != nil {
		return nil, nil, err
	}
	if override > 0 {
		timeout = override
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	return ctx, cancel, nil
}

func configuredCommandTimeout(command *cobra.Command) (time.Duration, error) {
	if command == nil {
		return aliasInvocationTimeout, nil
	}
	flag := command.Flag("timeout")
	if flag == nil {
		return aliasInvocationTimeout, nil
	}
	return parseAliasTimeout(flag.Value.String())
}
