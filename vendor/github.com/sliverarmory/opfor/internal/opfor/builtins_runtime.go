package opfor

import (
	"context"
	"fmt"
	"strings"
)

// sleepRuntimeFunctions returns the stock Sleep runtime helpers that depend on
// the active Script or Runtime rather than one of the narrower bridge tranches.
func (r *Runtime) sleepRuntimeFunctions() map[string]NativeFunc {
	return map[string]NativeFunc{
		"checkError": r.checkError,
		"debug":      r.debug,
		"exit":       builtinExit,
		"profile":    r.profile,
		"use":        r.useLoadable,
		"watch":      r.watch,
	}
}

// aggressorRuntimeFunctions returns the complete Aggressor runtime surface,
// including portable helpers and typed-provider wrappers. The narrower
// client-independent Aggressor tranches are composed by aggressorFunctions.
func (r *Runtime) aggressorRuntimeFunctions() map[string]NativeFunc {
	functions := map[string]NativeFunc{
		"alias":                              r.registerAlias,
		"alias_clear":                        r.clearAlias,
		"beacon_command_describe":            r.beaconCommandDescribe,
		"beacon_command_detail":              r.beaconCommandDetail,
		"beacon_command_group":               r.beaconCommandGroup,
		"beacon_command_register":            r.beaconCommandRegister,
		"beacon_commands":                    r.beaconCommands,
		"beacon_elevator_describe":           r.beaconElevatorDescribe,
		"beacon_elevator_register":           r.beaconElevatorRegister,
		"beacon_elevators":                   r.beaconElevators,
		"beacon_exploit_describe":            r.beaconExploitDescribe,
		"beacon_exploit_register":            r.beaconExploitRegister,
		"beacon_exploits":                    r.beaconExploits,
		"beacon_remote_exec_method_describe": r.beaconRemoteExecMethodDescribe,
		"beacon_remote_exec_method_register": r.beaconRemoteExecMethodRegister,
		"beacon_remote_exec_methods":         r.beaconRemoteExecMethods,
		"beacon_remote_exploit_arch":         r.beaconRemoteExploitArch,
		"beacon_remote_exploit_describe":     r.beaconRemoteExploitDescribe,
		"beacon_remote_exploit_register":     r.beaconRemoteExploitRegister,
		"beacon_remote_exploits":             r.beaconRemoteExploits,
		"berror":                             r.berror,
		"belevate":                           r.belevate,
		"belevate_command":                   r.belevateCommand,
		"binput":                             r.binput,
		"bjoberror":                          r.bjoberror,
		"bjoblog":                            r.bjoblog,
		"blog":                               r.blog,
		"blog2":                              r.blog2,
		"bof_pack":                           r.builtinAggressorBOFPack,
		"brk":                                r.aggressorBreakpoint,
		"bjump":                              r.bjump,
		"bremote_exec":                       r.bremoteExec,
		"btask":                              r.btask,
		"btaskcompleted":                     r.btaskcompleted,
		"dispatch_event":                     r.dispatchEvent,
		"fireAlias":                          r.fireAlias,
		"fireEvent":                          r.fireEvent,
		"fire_event":                         r.fireEvent,
		"getAggressorClientType":             r.getAggressorClientType,
		"bind":                               r.registerKeyBinding,
		"insert_menu":                        r.insertMenu,
		"item":                               r.registerMenuItem,
		"menu":                               r.registerSubmenu,
		"on":                                 r.registerEvent,
		"ssh_alias":                          r.registerSSHAlias,
		"ssh_command_describe":               r.sshCommandDescribe,
		"ssh_command_detail":                 r.sshCommandDetail,
		"ssh_command_group":                  r.sshCommandGroup,
		"ssh_command_register":               r.sshCommandRegister,
		"ssh_commands":                       r.sshCommands,
		"unbind":                             r.clearKeyBinding,
		"when":                               r.registerWhen,
	}
	for name, function := range r.aggressorPEFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorPEProviderFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorCodeTransformFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorBOFExtractionFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorArtifactFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorPayloadFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorListenerFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorPayloadStoreFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorSiteFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorTeamServerRPCFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorSessionQueryFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorDataModelQueryFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorDataStoreFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorPreferenceFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorProcessInjectionFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorProfileFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorVPNFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorClientServiceFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorBeaconActionFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorBeaconExecutionFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorClientUIFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorDialogFunctions() {
		functions[name] = function
	}
	for name, function := range r.aggressorPromptFunctions() {
		functions[name] = function
	}
	return functions
}

// fireEvent emits one runtime-local Aggressor event. fire_event is retained as
// the legacy spelling used by the official checkit.cna example.
func (r *Runtime) fireEvent(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 0 {
		return Null(), fmt.Errorf("&%s: expected an event name", invocation.Name)
	}
	values := invocation.Values()
	_, err := r.DispatchEvent(ctx, values[0].String(), values[1:]...)
	return Null(), err
}

func (r *Runtime) debug(_ context.Context, invocation Invocation) (Value, error) {
	script := r.script(invocation.Script)
	if script == nil {
		return Int(0), nil
	}
	script.mu.Lock()
	if len(invocation.Arguments) != 0 {
		// BasicUtilities routes debug's flag through BridgeUtilities.getInt.
		// StringValue.intValue accepts decimal integer text only; the broader
		// host-facing Value.Int32 conversion also accepts hex and floats.
		script.debug = sleepInt32(invocation.Arg(0))
	}
	flags := script.debug
	script.mu.Unlock()
	return Int(flags), nil
}

func builtinExit(_ context.Context, invocation Invocation) (Value, error) {
	exit := &scriptExit{span: invocation.Span}
	if len(invocation.Arguments) != 0 && !invocation.Arg(0).IsNull() {
		exit.message = invocation.Arg(0).String()
		exit.warn = true
	}
	return Null(), exit
}

func (r *Runtime) checkError(_ context.Context, invocation Invocation) (Value, error) {
	script := r.script(invocation.Script)
	if script == nil {
		return Null(), nil
	}
	script.mu.Lock()
	value := script.lastError
	script.lastError = Null()
	script.mu.Unlock()
	if len(invocation.Arguments) != 0 {
		invocation.Arguments[0].Set(value)
	}
	return value, nil
}

func (r *Runtime) watch(ctx context.Context, invocation Invocation) (Value, error) {
	fiber := currentFiber(ctx)
	variables := strings.Split(invocation.Arg(0).String(), " ")
	for _, variable := range variables {
		valid := len(variable) > 1 && (variable[0] == '$' || variable[0] == '@' || variable[0] == '%')
		var cell *Cell
		var exists bool
		if fiber != nil && fiber.scope != nil && valid {
			var err error
			cell, exists, err = fiber.scope.lookupAt(ctx, variable, invocation.Span)
			if err != nil {
				return Null(), err
			}
		}
		if !exists {
			err := fmt.Errorf("%s must already exist in a scope prior to watching", variable)
			if fiber != nil {
				return Null(), &uncaughtScriptWarning{err: err}
			}
			return Null(), err
		}
		name := variable
		cell.setWatcher(func(value Value, span Span) {
			r.writeWarning("watch(): "+name+" = "+value.Describe(), span)
		})
	}
	return Null(), nil
}

func (r *Runtime) script(id ScriptID) *Script {
	if r == nil || id == 0 {
		return nil
	}
	r.mu.RLock()
	script := r.scripts[id]
	r.mu.RUnlock()
	return script
}
