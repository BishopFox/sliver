package opfor

import (
	"context"
	"errors"
	"fmt"
)

type aggressorClientUISpec struct {
	operation     AggressorClientUIOperation
	minimum       int
	maximum       int
	callback      int
	discardResult bool
}

var aggressorClientUISpecs = map[string]aggressorClientUISpec{
	"addTab":            exactAggressorClientUISpec(AggressorClientUIAddTab, 3),
	"addVisualization":  exactAggressorClientUISpec(AggressorClientUIAddVisualization, 2),
	"showVisualization": exactAggressorClientUISpec(AggressorClientUIShowVisualization, 1),
	"show_popup":        exactAggressorClientUISpec(AggressorClientUIShowPopup, 3),
	"menubar":           exactAggressorClientUISpec(AggressorClientUIMenubar, 2),
	"popup_clear":       effectAggressorClientUISpec(AggressorClientUIPopupClear, 1),
	"separator":         exactAggressorClientUISpec(AggressorClientUISeparator, 0),
	"removeTab":         exactAggressorClientUISpec(AggressorClientUIRemoveTab, 0),
	"nextTab":           exactAggressorClientUISpec(AggressorClientUINextTab, 0),
	"previousTab":       exactAggressorClientUISpec(AggressorClientUIPreviousTab, 0),
	"add_to_clipboard":  exactAggressorClientUISpec(AggressorClientUIAddToClipboard, 1),
	"url_open":          exactAggressorClientUISpec(AggressorClientUIOpenURL, 1),
	"show_error":        exactAggressorClientUISpec(AggressorClientUIShowError, 1),
	"show_message":      exactAggressorClientUISpec(AggressorClientUIShowMessage, 1),
	"bbrowser":          exactAggressorClientUISpec(AggressorClientUIGenerateBeaconBrowser, 0),
	"colorMenu":         exactAggressorClientUISpec(AggressorClientUIColorMenu, 2),
	"file_browser":      effectAggressorClientUISpec(AggressorClientUIFileBrowser, 0),
	"insert_color_menu": effectAggressorClientUISpec(AggressorClientUIInsertColorMenu, 1),
	"insert_component":  effectAggressorClientUISpec(AggressorClientUIInsertComponent, 1),
	"pgraph":            exactAggressorClientUISpec(AggressorClientUIGeneratePivotGraph, 0),
	"process_browser":   effectAggressorClientUISpec(AggressorClientUIProcessBrowser, 0),
	"sbrowser":          exactAggressorClientUISpec(AggressorClientUIGenerateSessionBrowser, 0),
	"tbrowser":          exactAggressorClientUISpec(AggressorClientUIGenerateTargetBrowser, 0),

	"openAboutDialog":                     exactAggressorClientUISpec(AggressorClientUIOpenAboutDialog, 0),
	"openApplicationManager":              exactAggressorClientUISpec(AggressorClientUIOpenApplicationManager, 0),
	"openAutoRunDialog":                   exactAggressorClientUISpec(AggressorClientUIOpenAutoRunDialog, 0),
	"openBeaconBrowser":                   exactAggressorClientUISpec(AggressorClientUIOpenBeaconBrowser, 0),
	"openBeaconConsole":                   exactAggressorClientUISpec(AggressorClientUIOpenBeaconConsole, 1),
	"openBrowserPivotSetup":               exactAggressorClientUISpec(AggressorClientUIOpenBrowserPivotSetup, 1),
	"openCloneSiteDialog":                 exactAggressorClientUISpec(AggressorClientUIOpenCloneSiteDialog, 0),
	"openConnectDialog":                   exactAggressorClientUISpec(AggressorClientUIOpenConnectDialog, 0),
	"openCovertVPNSetup":                  exactAggressorClientUISpec(AggressorClientUIOpenCovertVPNSetup, 1),
	"openCredentialManager":               exactAggressorClientUISpec(AggressorClientUIOpenCredentialManager, 0),
	"openDefaultShortcutsDialog":          exactAggressorClientUISpec(AggressorClientUIOpenDefaultShortcutsDialog, 0),
	"openDownloadBrowser":                 exactAggressorClientUISpec(AggressorClientUIOpenDownloadBrowser, 0),
	"openElevateDialog":                   exactAggressorClientUISpec(AggressorClientUIOpenElevateDialog, 1),
	"openEventLog":                        exactAggressorClientUISpec(AggressorClientUIOpenEventLog, 0),
	"openFileBrowser":                     exactAggressorClientUISpec(AggressorClientUIOpenFileBrowser, 1),
	"openGoldenTicketDialog":              exactAggressorClientUISpec(AggressorClientUIOpenGoldenTicketDialog, 1),
	"openHTMLApplicationDialog":           exactAggressorClientUISpec(AggressorClientUIOpenHTMLApplicationDialog, 0),
	"openHostFileDialog":                  exactAggressorClientUISpec(AggressorClientUIOpenHostFileDialog, 0),
	"openInterfaceManager":                exactAggressorClientUISpec(AggressorClientUIOpenInterfaceManager, 0),
	"openJavaSignedAppletDialog":          exactAggressorClientUISpec(AggressorClientUIOpenJavaSignedAppletDialog, 0),
	"openJavaSmartAppletDialog":           exactAggressorClientUISpec(AggressorClientUIOpenJavaSmartAppletDialog, 0),
	"openJobBrowser":                      {operation: AggressorClientUIOpenJobBrowser, minimum: 0, maximum: 1, callback: -1},
	"openJobConsole":                      exactAggressorClientUISpec(AggressorClientUIOpenJobConsole, 2),
	"openJumpDialog":                      exactAggressorClientUISpec(AggressorClientUIOpenJumpDialog, 2),
	"openKeystrokeBrowser":                exactAggressorClientUISpec(AggressorClientUIOpenKeystrokeBrowser, 0),
	"openListenerManager":                 exactAggressorClientUISpec(AggressorClientUIOpenListenerManager, 0),
	"openMakeTokenDialog":                 exactAggressorClientUISpec(AggressorClientUIOpenMakeTokenDialog, 1),
	"openMalleableProfileDialog":          exactAggressorClientUISpec(AggressorClientUIOpenMalleableProfileDialog, 0),
	"openNewCredentialDialog":             exactAggressorClientUISpec(AggressorClientUIOpenNewCredentialDialog, 1),
	"openOfficeMacroDialog":               exactAggressorClientUISpec(AggressorClientUIOpenOfficeMacroDialog, 0),
	"openOneLinerDialog":                  exactAggressorClientUISpec(AggressorClientUIOpenOneLinerDialog, 1),
	"openOrActivate":                      exactAggressorClientUISpec(AggressorClientUIOpenOrActivate, 1),
	"openPayloadGeneratorDialog":          exactAggressorClientUISpec(AggressorClientUIOpenPayloadGeneratorDialog, 0),
	"openPayloadGeneratorStageDialog":     exactAggressorClientUISpec(AggressorClientUIOpenPayloadGeneratorStageDialog, 0),
	"openPayloadHelper":                   callbackAggressorClientUISpec(AggressorClientUIOpenPayloadHelper, 0),
	"openPayloadStoreManager":             exactAggressorClientUISpec(AggressorClientUIOpenPayloadStoreManager, 0),
	"openPivotListenerSetup":              exactAggressorClientUISpec(AggressorClientUIOpenPivotListenerSetup, 1),
	"openPortScanner":                     exactAggressorClientUISpec(AggressorClientUIOpenPortScanner, 1),
	"openPortScannerLocal":                exactAggressorClientUISpec(AggressorClientUIOpenPortScannerLocal, 1),
	"openPowerShellWebDialog":             exactAggressorClientUISpec(AggressorClientUIOpenPowerShellWebDialog, 0),
	"openPreferencesDialog":               exactAggressorClientUISpec(AggressorClientUIOpenPreferencesDialog, 0),
	"openProcessBrowser":                  exactAggressorClientUISpec(AggressorClientUIOpenProcessBrowser, 1),
	"openSOCKSBrowser":                    exactAggressorClientUISpec(AggressorClientUIOpenSOCKSBrowser, 0),
	"openSOCKSSetup":                      exactAggressorClientUISpec(AggressorClientUIOpenSOCKSSetup, 1),
	"openScreenshotBrowser":               exactAggressorClientUISpec(AggressorClientUIOpenScreenshotBrowser, 0),
	"openScriptConsole":                   exactAggressorClientUISpec(AggressorClientUIOpenScriptConsole, 0),
	"openScriptManager":                   exactAggressorClientUISpec(AggressorClientUIOpenScriptManager, 0),
	"openScriptedWebDialog":               exactAggressorClientUISpec(AggressorClientUIOpenScriptedWebDialog, 0),
	"openServiceBrowser":                  exactAggressorClientUISpec(AggressorClientUIOpenServiceBrowser, 1),
	"openSiteManager":                     exactAggressorClientUISpec(AggressorClientUIOpenSiteManager, 0),
	"openSpawnAsDialog":                   exactAggressorClientUISpec(AggressorClientUIOpenSpawnAsDialog, 1),
	"openSpawnDialog":                     exactAggressorClientUISpec(AggressorClientUIOpenSpawnDialog, 1),
	"openSpearPhishDialog":                exactAggressorClientUISpec(AggressorClientUIOpenSpearPhishDialog, 0),
	"openSystemInformationDialog":         exactAggressorClientUISpec(AggressorClientUIOpenSystemInformationDialog, 0),
	"openSystemProfilerDialog":            exactAggressorClientUISpec(AggressorClientUIOpenSystemProfilerDialog, 0),
	"openTargetBrowser":                   exactAggressorClientUISpec(AggressorClientUIOpenTargetBrowser, 0),
	"openUserDefinedBrowser":              {operation: AggressorClientUIOpenUserDefinedBrowser, minimum: 4, maximum: 6, callback: -1},
	"openWebLog":                          exactAggressorClientUISpec(AggressorClientUIOpenWebLog, 0),
	"openWindowsExecutableDialog":         exactAggressorClientUISpec(AggressorClientUIOpenWindowsExecutableDialog, 0),
	"openWindowsExecutableStageAllDialog": exactAggressorClientUISpec(AggressorClientUIOpenWindowsExecutableStageAllDialog, 0),
	"openWindowsExecutableStageDialog":    exactAggressorClientUISpec(AggressorClientUIOpenWindowsExecutableStageDialog, 0),
}

func exactAggressorClientUISpec(operation AggressorClientUIOperation, arguments int) aggressorClientUISpec {
	return aggressorClientUISpec{operation: operation, minimum: arguments, maximum: arguments, callback: -1}
}

func callbackAggressorClientUISpec(operation AggressorClientUIOperation, index int) aggressorClientUISpec {
	return aggressorClientUISpec{
		operation:     operation,
		minimum:       index + 1,
		maximum:       index + 1,
		callback:      index,
		discardResult: true,
	}
}

func effectAggressorClientUISpec(operation AggressorClientUIOperation, arguments int) aggressorClientUISpec {
	spec := exactAggressorClientUISpec(operation, arguments)
	spec.discardResult = true
	return spec
}

// aggressorClientUIFunctions returns native wrappers around the
// importer-owned client UI boundary. With no provider, every valid call
// preserves the original reference-bearing Host invocation exactly once.
func (r *Runtime) aggressorClientUIFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorClientUISpecs))
	for name := range aggressorClientUISpecs {
		functions[name] = r.aggressorClientUI
	}
	return functions
}

func (r *Runtime) aggressorClientUI(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorClientUISpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor client UI",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	if err := requireAggressorClientUIArguments(invocation, spec.minimum, spec.maximum); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	provider := r.aggressorClientUIProvider
	if isNilInterface(provider) {
		// Keep the original Arguments intact on the compatibility route. Host
		// retains pass-by-name and ordinary bare-variable Cell capabilities. The
		// one locally meaningful fallback, popup_clear, snapshots its target before
		// Host so a Host mutation cannot redirect the clear.
		clearName := ""
		if spec.operation == AggressorClientUIPopupClear {
			clearName = invocation.Arg(0).String()
		}
		result, err := r.host.Call(ctx, invocation)
		if boundaryErr := preserveNativeBoundaryError(ctx, err); boundaryErr != nil {
			return result, boundaryErr
		}
		if spec.operation == AggressorClientUIPopupClear {
			if err := r.clearAggressorPopupBindings(ctx, clearName); err != nil {
				return result, err
			}
		}
		return result, nil
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}

	values := invocation.Values()
	request := AggressorClientUIRequest{
		Operation: spec.operation,
		Name:      invocation.Name,
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
		Bindings:  invocation.Bindings(),
		Arguments: values,
	}
	if spec.callback >= 0 {
		callback, err := invocation.RetainCallback(values[spec.callback])
		if err != nil {
			if errors.Is(err, ErrInvalidCallable) {
				return Null(), fmt.Errorf("&%s: argument %d is not callable: %w",
					builtinName(invocation.Name), spec.callback+1, err)
			}
			return Null(), err
		}
		request.Callback = callback
		request.Arguments = append([]Value(nil), values[:spec.callback]...)
		request.Arguments = append(request.Arguments, values[spec.callback+1:]...)
	}
	if spec.operation == AggressorClientUIShowPopup || spec.operation == AggressorClientUIMenubar {
		bindings := r.Bindings(BindingPopup, values[1].String())
		if len(bindings) != 0 {
			arguments := []Value(nil)
			if spec.operation == AggressorClientUIShowPopup {
				arguments = []Value{values[2]}
			}
			request.Popup = newAggressorPopupComposer(r, invocation, bindings, arguments, nil)
		}
	}
	if spec.operation == AggressorClientUISeparator ||
		spec.operation == AggressorClientUIInsertColorMenu ||
		spec.operation == AggressorClientUIInsertComponent {
		request.Composition = cloneBindingInvocation(currentBindingInvocation(ctx))
	}

	result, err := provider.HandleAggressorClientUI(ctx, request)
	err = preserveNativeBoundaryError(ctx, err)
	err = joinExecutionContextError(ctx, err)
	if err != nil {
		return Null(), err
	}
	if spec.operation == AggressorClientUIPopupClear {
		if err := r.clearAggressorPopupBindings(ctx, values[0].String()); err != nil {
			return Null(), err
		}
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	if spec.discardResult {
		return Null(), nil
	}
	return result, nil
}

func requireAggressorClientUIArguments(invocation Invocation, minimum, maximum int) error {
	count := len(invocation.Arguments)
	if count >= minimum && count <= maximum {
		return nil
	}
	if minimum == maximum {
		return requireExactAggressorClientArguments(invocation, minimum)
	}
	return fmt.Errorf("&%s: expected %d to %d argument(s), received %d",
		builtinName(invocation.Name), minimum, maximum, count)
}

type aggressorPopupComposer struct {
	runtime    *Runtime
	creator    ScriptID
	generation *scriptGeneration
	bindings   []aggressorPopupBindingReference
	arguments  []Value
	parent     *BindingInvocation
}

type aggressorPopupBindingReference struct {
	script  ScriptID
	binding uint64
}

func newAggressorPopupComposer(
	runtime *Runtime,
	invocation Invocation,
	bindings []Binding,
	arguments []Value,
	parent *BindingInvocation,
) *aggressorPopupComposer {
	selected := make([]aggressorPopupBindingReference, len(bindings))
	for index, binding := range bindings {
		selected[index] = aggressorPopupBindingReference{script: binding.Script, binding: binding.ID}
	}
	return &aggressorPopupComposer{
		runtime:    runtime,
		creator:    invocation.Script,
		generation: invocation.generationToken(),
		bindings:   selected,
		arguments:  append([]Value(nil), arguments...),
		parent:     cloneBindingInvocation(parent),
	}
}

func (composer *aggressorPopupComposer) Compose(ctx context.Context) (resultErr error) {
	if composer == nil || composer.runtime == nil {
		return errors.New("opfor: Aggressor popup composer is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if composer.creator != 0 {
		creator := composer.runtime.script(composer.creator)
		if creator == nil {
			return errors.Join(ErrAggressorPopupStale, ErrScriptUnloaded)
		}
		executionCtx, release, err := creator.acquireGenerationExecution(ctx, composer.generation)
		if err != nil {
			if errors.Is(err, ErrScriptUnloaded) {
				return errors.Join(ErrAggressorPopupStale, err)
			}
			return err
		}
		ctx = executionCtx
		defer func() { resultErr = joinExecutionError(resultErr, release) }()
	}
	// Pin the whole captured generation under one registry snapshot before
	// invoking any member. A clear followed by a same-name registration can
	// therefore neither retarget this capability nor intentionally produce a
	// partial tree. A generation cleared before this snapshot is stale; a clear
	// after it does not revoke callbacks already admitted to this composition.
	bindings, ok := composer.runtime.captureAggressorPopupBindings(composer.bindings)
	if !ok {
		return ErrAggressorPopupStale
	}
	var result error
	for _, binding := range bindings {
		arguments := append([]Value(nil), composer.arguments...)
		_, err := composer.runtime.invokePinnedAggressorPopupBinding(ctx, binding, arguments, composer.parent)
		result = errors.Join(result, err)
	}
	return result
}

// captureAggressorPopupBindings resolves a retained generation atomically and
// returns the registrations in the composer's original registration order.
func (r *Runtime) captureAggressorPopupBindings(references []aggressorPopupBindingReference) ([]Binding, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	active := make(map[aggressorPopupBindingReference]Binding, len(references))
	for _, binding := range r.bindingOrder[BindingPopup] {
		key := aggressorPopupBindingReference{script: binding.Script, binding: binding.ID}
		active[key] = binding
	}
	bindings := make([]Binding, len(references))
	for index, reference := range references {
		binding, exists := active[reference]
		if !exists {
			return nil, false
		}
		bindings[index] = cloneBinding(binding)
	}
	return bindings, true
}

func (r *Runtime) invokePinnedAggressorPopupBinding(
	ctx context.Context,
	binding Binding,
	arguments []Value,
	parent *BindingInvocation,
) (Value, error) {
	if parent != nil {
		binding.Parent = cloneBindingInvocation(parent)
	}
	value, err, claimed := r.invokeRegisteredBinding(ctx, binding, arguments)
	if !claimed {
		return Null(), ErrAggressorPopupStale
	}
	return value, err
}

// clearAggressorPopupBindings retires every exact popup registration in
// reverse registration order, mirroring other layered binding clear
// operations. A composer retains only IDs, so removal makes its captured
// generation stale and later declarations cannot be substituted.
func (r *Runtime) clearAggressorPopupBindings(ctx context.Context, name string) error {
	bindings := r.Bindings(BindingPopup, name)
	var result error
	for index := len(bindings) - 1; index >= 0; index-- {
		binding := bindings[index]
		// A composed popup owns ephemeral menu/item descendants. Retire that
		// complete tree before removing the root registration itself.
		result = errors.Join(result, r.clearBindingDescendants(ctx, binding))
		r.mu.RLock()
		owner := r.scripts[binding.Script]
		r.mu.RUnlock()
		if owner == nil || !owner.removeBindingIfPresent(binding) {
			continue
		}
		if r.observer != nil {
			if err := r.observer.Unregistered(ctx, cloneBinding(binding)); err != nil {
				result = errors.Join(result, preserveNativeBoundaryError(ctx, err))
			}
		}
	}
	return result
}
