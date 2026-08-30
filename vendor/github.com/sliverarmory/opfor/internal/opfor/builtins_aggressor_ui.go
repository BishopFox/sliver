package opfor

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type aggressorUIResource interface {
	revokeAggressorUI()
	aggressorUIGeneration() *scriptGeneration
}

type aggressorUIState uint8

const (
	aggressorUIBuilding aggressorUIState = iota
	aggressorUIPresenting
	aggressorUIOpen
	aggressorUICompleted
	aggressorUIDismissed
	aggressorUIFailed
	aggressorUIRevoked
)

func (state aggressorUIState) terminal() bool {
	return state >= aggressorUICompleted
}

type aggressorDialogRowSpec struct {
	kind       AggressorDialogRowKind
	minimum    int
	maximum    int
	checkbox   bool
	combobox   bool
	width      bool
	deprecated bool
	equivalent AggressorDialogRowKind
}

var aggressorDialogRowSpecs = map[string]aggressorDialogRowSpec{
	"drow_beacon":         {kind: AggressorDialogRowBeacon, minimum: 3, maximum: 3},
	"drow_checkbox":       {kind: AggressorDialogRowCheckbox, minimum: 4, maximum: 4, checkbox: true},
	"drow_combobox":       {kind: AggressorDialogRowCombobox, minimum: 4, maximum: 4, combobox: true},
	"drow_exploits":       {kind: AggressorDialogRowExploits, minimum: 3, maximum: 3},
	"drow_file":           {kind: AggressorDialogRowFile, minimum: 3, maximum: 3},
	"drow_interface":      {kind: AggressorDialogRowInterface, minimum: 3, maximum: 3},
	"drow_krbtgt":         {kind: AggressorDialogRowKRBTGT, minimum: 3, maximum: 3},
	"drow_listener":       {kind: AggressorDialogRowListener, minimum: 3, maximum: 3},
	"drow_listener_smb":   {kind: AggressorDialogRowListenerStage, minimum: 3, maximum: 3, deprecated: true, equivalent: AggressorDialogRowListenerStage},
	"drow_listener_stage": {kind: AggressorDialogRowListenerStage, minimum: 3, maximum: 3},
	"drow_mailserver":     {kind: AggressorDialogRowMailServer, minimum: 3, maximum: 3},
	"drow_proxyserver":    {kind: AggressorDialogRowProxyServer, minimum: 3, maximum: 3, deprecated: true},
	"drow_site":           {kind: AggressorDialogRowSite, minimum: 3, maximum: 3},
	"drow_text":           {kind: AggressorDialogRowText, minimum: 3, maximum: 4, width: true},
	"drow_text_big":       {kind: AggressorDialogRowTextBig, minimum: 3, maximum: 3},
}

type aggressorPromptSpec struct {
	kind          AggressorPromptKind
	arguments     int
	callbackIndex int
	callbackArity int
}

var aggressorPromptSpecs = map[string]aggressorPromptSpec{
	"prompt_confirm":        {kind: AggressorPromptConfirm, arguments: 3, callbackIndex: 2},
	"prompt_directory_open": {kind: AggressorPromptDirectoryOpen, arguments: 4, callbackIndex: 3, callbackArity: 1},
	"prompt_file_open":      {kind: AggressorPromptFileOpen, arguments: 4, callbackIndex: 3, callbackArity: 1},
	"prompt_file_save":      {kind: AggressorPromptFileSave, arguments: 2, callbackIndex: 1, callbackArity: 1},
	"prompt_text":           {kind: AggressorPromptText, arguments: 3, callbackIndex: 2, callbackArity: 1},
}

func (r *Runtime) aggressorDialogFunctions() map[string]NativeFunc {
	functions := map[string]NativeFunc{
		"dialog":             r.aggressorDialogCreate,
		"dialog_description": r.aggressorDialogDescription,
		"dialog_show":        r.aggressorDialogShow,
		"dbutton_action":     r.aggressorDialogButton,
		"dbutton_help":       r.aggressorDialogButton,
	}
	for name := range aggressorDialogRowSpecs {
		functions[name] = r.aggressorDialogRow
	}
	return functions
}

func (r *Runtime) aggressorPromptFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorPromptSpecs))
	for name := range aggressorPromptSpecs {
		functions[name] = r.aggressorPrompt
	}
	return functions
}

func requireAggressorUIArguments(invocation Invocation, minimum, maximum int) error {
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

func callAggressorUIHost(ctx context.Context, runtime *Runtime, invocation Invocation) (Value, error) {
	if runtime == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	result, err := runtime.host.Call(ctx, invocation)
	return result, preserveNativeBoundaryError(ctx, err)
}

func retainAggressorUICallback(invocation Invocation, value Value, index int) (Callable, error) {
	callback, err := invocation.RetainCallback(value)
	if err == nil {
		return callback, nil
	}
	if errors.Is(err, ErrInvalidCallable) {
		return nil, fmt.Errorf("&%s: argument %d is not callable: %w",
			builtinName(invocation.Name), index+1, err)
	}
	return nil, err
}

func (r *Runtime) nextAggressorDialogIdentity() (AggressorDialogID, error) {
	if r == nil {
		return 0, errors.New("opfor: runtime is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.closed {
		return 0, ErrRuntimeClosed
	}
	for {
		r.nextAggressorDialog++
		if r.nextAggressorDialog != 0 {
			return r.nextAggressorDialog, nil
		}
	}
}

func (r *Runtime) nextAggressorPromptIdentity() (AggressorPromptID, error) {
	if r == nil {
		return 0, errors.New("opfor: runtime is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.closed {
		return 0, ErrRuntimeClosed
	}
	for {
		r.nextAggressorPrompt++
		if r.nextAggressorPrompt != 0 {
			return r.nextAggressorPrompt, nil
		}
	}
}

func registerAggressorUIResource(invocation Invocation, owner *Script, resource aggressorUIResource) error {
	if owner == nil || resource == nil {
		return ErrScriptUnloaded
	}
	generation := invocation.generationToken()
	if generation == nil || resource.aggressorUIGeneration() != generation {
		return ErrScriptUnloaded
	}
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if !owner.generationAdmissibleLocked(generation) {
		return ErrScriptUnloaded
	}
	if owner.aggressorUIResources == nil {
		owner.aggressorUIResources = make(map[aggressorUIResource]struct{})
	}
	owner.aggressorUIResources[resource] = struct{}{}
	return nil
}

func unregisterAggressorUIResource(owner *Script, resource aggressorUIResource) {
	if owner == nil || resource == nil {
		return
	}
	owner.mu.Lock()
	delete(owner.aggressorUIResources, resource)
	owner.mu.Unlock()
}

// takeAggressorUIResourcesForGenerationLocked removes and returns the UI
// resources owned by one exact execution generation. The caller must hold
// owner.mu. Revocation must happen after releasing owner.mu because responder
// cleanup may otherwise contend with script lifecycle work.
func takeAggressorUIResourcesForGenerationLocked(
	owner *Script,
	generation *scriptGeneration,
) []aggressorUIResource {
	if owner == nil || generation == nil {
		return nil
	}
	resources := make([]aggressorUIResource, 0)
	for resource := range owner.aggressorUIResources {
		if resource != nil && resource.aggressorUIGeneration() == generation {
			resources = append(resources, resource)
			delete(owner.aggressorUIResources, resource)
		}
	}
	return resources
}

func revokeAggressorUIResources(resources []aggressorUIResource) {
	for _, resource := range resources {
		if resource != nil {
			resource.revokeAggressorUI()
		}
	}
}

func snapshotAggressorDialogDefaults(value Value) ([]AggressorDialogDefault, error) {
	hash, ok := value.Hash()
	if !ok || hash == nil {
		return nil, errors.New("opfor: Aggressor dialog defaults must be a hash")
	}
	keys := hash.KeyValues()
	defaults := make([]AggressorDialogDefault, 0, len(keys))
	for _, key := range keys {
		cell, exists := hash.CellValue(key)
		if !exists || cell == nil {
			continue
		}
		defaults = append(defaults, AggressorDialogDefault{Name: key.String(), Value: cell.Get()})
	}
	return defaults, nil
}

// aggressorUIPresentationWindow distinguishes an inline provider response from
// a responder retained beyond PresentAggressor*'s return. Inline callbacks are
// ordinary nested script execution: they share the presenter's instruction
// meter and execution ancestry. Retained callbacks are detached UI events and
// receive a fresh top-level meter from scriptClosure.Invoke.
//
// finish waits for callbacks which entered while the provider call was live.
// Besides making the boundary deterministic, this keeps the outer execution
// token active until a reentrant Unload or Close in the callback can reserve it
// as the lifecycle-error recipient instead of waiting on its own inner lease.
type aggressorUIPresentationWindow struct {
	mu sync.Mutex

	context  context.Context
	open     bool
	inFlight int
	drained  chan struct{}
}

func newAggressorUIPresentationWindow(ctx context.Context) *aggressorUIPresentationWindow {
	if ctx == nil {
		ctx = context.Background()
	}
	return &aggressorUIPresentationWindow{
		context: ctx,
		open:    true,
		drained: make(chan struct{}),
	}
}

func (window *aggressorUIPresentationWindow) callbackContext(ctx context.Context) (context.Context, func()) {
	if ctx == nil {
		ctx = context.Background()
	}
	if window == nil {
		return snapshotCallbackInvocationContext(ctx, nil), func() {}
	}
	window.mu.Lock()
	if !window.open || window.context == nil {
		window.mu.Unlock()
		return snapshotCallbackInvocationContext(ctx, nil), func() {}
	}
	presentationContext := window.context
	window.inFlight++
	window.mu.Unlock()

	caller, releaseCaller := captureExecutionCallerLease(ctx)
	ancestry := &aggressorUICallbackAncestry{
		context: presentationContext,
		caller:  caller,
	}
	ancestry.active.Store(true)
	callbackContext := context.Context(aggressorUISynchronousCallbackContext{
		Context:      ctx,
		presentation: presentationContext,
		ancestry:     ancestry,
	})
	return callbackContext, func() {
		ancestry.active.Store(false)
		releaseCaller()
		window.mu.Lock()
		if window.inFlight > 0 {
			window.inFlight--
		}
		if !window.open && window.inFlight == 0 && !channelClosed(window.drained) {
			window.context = nil
			close(window.drained)
		}
		window.mu.Unlock()
	}
}

func (window *aggressorUIPresentationWindow) close() <-chan struct{} {
	if window == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	window.mu.Lock()
	window.open = false
	if window.inFlight == 0 && !channelClosed(window.drained) {
		window.context = nil
		close(window.drained)
	}
	drained := window.drained
	window.mu.Unlock()
	return drained
}

// aggressorUISynchronousCallbackContext takes cancellation from the context
// supplied to Activate/Accept while preserving the live presenter's private
// evaluator state. Non-private values prefer the responder context and fall
// back to the presentation context, allowing a provider to add request-scoped
// values without losing importer values already attached to Present's ctx.
type aggressorUISynchronousCallbackContext struct {
	context.Context
	presentation context.Context
	ancestry     *aggressorUICallbackAncestry
}

func (ctx aggressorUISynchronousCallbackContext) AfterFunc(function func()) func() bool {
	return context.AfterFunc(ctx.Context, function)
}

func (ctx aggressorUISynchronousCallbackContext) Value(key any) any {
	switch key.(type) {
	case aggressorUICallbackAncestryContextKey:
		return ctx.ancestry
	case executionMeterKey:
		if ctx.ancestry == nil || !ctx.ancestry.active.Load() || ctx.presentation == nil {
			return nil
		}
		return ctx.presentation.Value(key)
	case currentFiberContextKey,
		includeChainContextKey,
		bindingInvocationContextKey,
		nativeDispatchStateContextKey,
		portableScriptInstanceRunContextKey,
		scriptExecutionContextKey,
		runtimeExecutionContextKey,
		scriptUnloadContextKey,
		runtimeCloseContextKey:
		// A private ancestry marker, rather than the raw tokens, gives lifecycle
		// classification the live outer execution without letting a callable
		// retain those tokens after this responder invocation returns.
		return nil
	default:
		if ctx.Context != nil {
			if value := ctx.Context.Value(key); value != nil {
				return value
			}
		}
		if ctx.presentation == nil {
			return nil
		}
		return ctx.presentation.Value(key)
	}
}

type aggressorDialog struct {
	mu sync.Mutex

	owner      *Script
	generation *scriptGeneration
	runtimeID  RuntimeID
	id         AggressorDialogID
	value      Value
	callback   Callable
	state      aggressorUIState
	done       chan struct{}
	window     *aggressorUIPresentationWindow

	creatorScript ScriptID
	creationSpan  Span
	title         string
	defaults      []AggressorDialogDefault

	description       string
	descriptionLines  int32
	hasDescription    bool
	descriptionScript ScriptID
	descriptionSpan   Span

	nextRow    AggressorDialogRowID
	nextButton AggressorDialogButtonID
	rows       []AggressorDialogRow
	buttons    []AggressorDialogButton
}

func (dialog *aggressorDialog) String() string {
	if dialog == nil {
		return "<Aggressor dialog>"
	}
	return fmt.Sprintf("<Aggressor dialog %d>", dialog.id)
}

func (dialog *aggressorDialog) Done() <-chan struct{} {
	if dialog == nil || dialog.done == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return dialog.done
}

func (dialog *aggressorDialog) stateErrorLocked() error {
	if dialog.state == aggressorUIRevoked {
		return ErrScriptUnloaded
	}
	return fmt.Errorf("%w: dialog %d", ErrAggressorUIClosed, dialog.id)
}

func (dialog *aggressorDialog) requireBuildingLocked() error {
	if dialog.state == aggressorUIRevoked {
		return ErrScriptUnloaded
	}
	if dialog.state != aggressorUIBuilding {
		return fmt.Errorf("opfor: Aggressor dialog %d is no longer being built", dialog.id)
	}
	return nil
}

func (dialog *aggressorDialog) finishLocked(state aggressorUIState) (*Script, Callable) {
	owner := dialog.owner
	callback := dialog.callback
	dialog.owner = nil
	dialog.callback = nil
	dialog.state = state
	// The presentation snapshot and any callback arguments already hold the
	// exact Values they need. Drop the runtime-owned construction graph so a UI
	// provider retaining only a terminal responder does not also retain every
	// default, option, private dialog self-reference, and owning closure.
	dialog.value = Null()
	dialog.title = ""
	dialog.defaults = nil
	dialog.description = ""
	dialog.rows = nil
	dialog.buttons = nil
	if dialog.done != nil {
		select {
		case <-dialog.done:
		default:
			close(dialog.done)
		}
	}
	return owner, callback
}

func (dialog *aggressorDialog) fail() {
	if dialog == nil {
		return
	}
	dialog.mu.Lock()
	if dialog.state.terminal() {
		dialog.mu.Unlock()
		return
	}
	owner, _ := dialog.finishLocked(aggressorUIFailed)
	dialog.mu.Unlock()
	unregisterAggressorUIResource(owner, dialog)
}

func (dialog *aggressorDialog) revokeAggressorUI() {
	if dialog == nil {
		return
	}
	dialog.mu.Lock()
	if dialog.state.terminal() {
		dialog.mu.Unlock()
		return
	}
	dialog.finishLocked(aggressorUIRevoked)
	dialog.mu.Unlock()
}

func (dialog *aggressorDialog) aggressorUIGeneration() *scriptGeneration {
	if dialog == nil {
		return nil
	}
	return dialog.generation
}

func (dialog *aggressorDialog) Dismiss() error {
	if dialog == nil {
		return errors.New("opfor: Aggressor dialog responder is nil")
	}
	dialog.mu.Lock()
	if dialog.state != aggressorUIPresenting && dialog.state != aggressorUIOpen {
		err := dialog.stateErrorLocked()
		dialog.mu.Unlock()
		return err
	}
	owner, _ := dialog.finishLocked(aggressorUIDismissed)
	dialog.mu.Unlock()
	unregisterAggressorUIResource(owner, dialog)
	return nil
}

func (dialog *aggressorDialog) Activate(
	ctx context.Context,
	buttonID AggressorDialogButtonID,
	rowValues ...AggressorDialogRowValue,
) (Value, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	if dialog == nil {
		return Null(), errors.New("opfor: Aggressor dialog responder is nil")
	}

	dialog.mu.Lock()
	if dialog.state != aggressorUIPresenting && dialog.state != aggressorUIOpen {
		err := dialog.stateErrorLocked()
		dialog.mu.Unlock()
		return Null(), err
	}
	var button AggressorDialogButton
	buttonFound := false
	for _, candidate := range dialog.buttons {
		if candidate.ID == buttonID {
			button = candidate
			buttonFound = true
			break
		}
	}
	if !buttonFound {
		dialog.mu.Unlock()
		return Null(), fmt.Errorf("opfor: Aggressor dialog %d has no button %d", dialog.id, buttonID)
	}
	if button.Kind != AggressorDialogButtonAction {
		dialog.mu.Unlock()
		return Null(), fmt.Errorf("opfor: Aggressor dialog button %d is not an action button", buttonID)
	}

	knownRows := make(map[AggressorDialogRowID]AggressorDialogRow, len(dialog.rows))
	for _, row := range dialog.rows {
		knownRows[row.ID] = row
	}
	provided := make(map[AggressorDialogRowID]Value, len(rowValues))
	for _, response := range rowValues {
		if _, exists := knownRows[response.RowID]; !exists {
			dialog.mu.Unlock()
			return Null(), fmt.Errorf("opfor: Aggressor dialog %d has no row %d", dialog.id, response.RowID)
		}
		if _, duplicate := provided[response.RowID]; duplicate {
			dialog.mu.Unlock()
			return Null(), fmt.Errorf("opfor: Aggressor dialog response repeats row %d", response.RowID)
		}
		provided[response.RowID] = response.Value
	}
	rowNames := make(map[string]struct{}, len(dialog.rows))
	for _, row := range dialog.rows {
		rowNames[sleepCanonicalString(String(row.Name))] = struct{}{}
	}
	var runtime *Runtime
	if dialog.owner != nil {
		runtime = dialog.owner.runtime
	}
	if err := reserveCollectionEntries(runtime, len(rowNames)); err != nil {
		dialog.mu.Unlock()
		return Null(), err
	}
	values := NewHash()
	for _, row := range dialog.rows {
		value, supplied := provided[row.ID]
		if !supplied {
			value = Null()
			if row.HasDefault {
				value = row.Default
			}
		}
		values.Set(row.Name, value)
	}
	callbackValue := dialog.value
	callbackContext, callbackDone := dialog.window.callbackContext(ctx)
	owner, callback := dialog.finishLocked(aggressorUICompleted)
	dialog.mu.Unlock()
	defer callbackDone()
	unregisterAggressorUIResource(owner, dialog)
	if callback == nil {
		return Null(), ErrInvalidCallable
	}
	return callback.Invoke(
		callbackContext,
		callbackValue,
		String(button.Label),
		HashValue(values),
	)
}

func (dialog *aggressorDialog) presentation(ctx context.Context, invocation Invocation) (AggressorDialogPresentation, error) {
	if dialog == nil {
		return AggressorDialogPresentation{}, errors.New("opfor: Aggressor dialog is nil")
	}
	dialog.mu.Lock()
	defer dialog.mu.Unlock()
	if err := dialog.requireBuildingLocked(); err != nil {
		return AggressorDialogPresentation{}, err
	}
	dialog.state = aggressorUIPresenting
	dialog.window = newAggressorUIPresentationWindow(ctx)
	defaults := append([]AggressorDialogDefault(nil), dialog.defaults...)
	rows := make([]AggressorDialogRow, len(dialog.rows))
	for index, row := range dialog.rows {
		rows[index] = row
		rows[index].Options = append([]Value(nil), row.Options...)
	}
	buttons := append([]AggressorDialogButton(nil), dialog.buttons...)
	return AggressorDialogPresentation{
		ID:                dialog.id,
		RuntimeID:         dialog.runtimeID,
		CreatorScript:     dialog.creatorScript,
		CreationSpan:      dialog.creationSpan,
		PresenterScript:   invocation.Script,
		PresentationSpan:  invocation.Span,
		Title:             dialog.title,
		Defaults:          defaults,
		Description:       dialog.description,
		DescriptionLines:  dialog.descriptionLines,
		HasDescription:    dialog.hasDescription,
		DescriptionScript: dialog.descriptionScript,
		DescriptionSpan:   dialog.descriptionSpan,
		Rows:              rows,
		Buttons:           buttons,
	}, nil
}

func (dialog *aggressorDialog) finishPresentationWindow(failed bool) {
	if dialog == nil {
		return
	}
	dialog.mu.Lock()
	window := dialog.window
	drained := window.close()
	var owner *Script
	if failed && !dialog.state.terminal() {
		owner, _ = dialog.finishLocked(aggressorUIFailed)
	} else if !failed && dialog.state == aggressorUIPresenting {
		dialog.state = aggressorUIOpen
	}
	if dialog.window == window {
		dialog.window = nil
	}
	dialog.mu.Unlock()
	unregisterAggressorUIResource(owner, dialog)
	<-drained
}

func (r *Runtime) aggressorDialogCreate(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireExactAggressorClientArguments(invocation, 3); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	if isNilInterface(r.aggressorDialogProvider) {
		return callAggressorUIHost(ctx, r, invocation)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	values := invocation.Values()
	defaults, err := snapshotAggressorDialogDefaults(values[1])
	if err != nil {
		return Null(), fmt.Errorf("&%s: argument 2: %w", builtinName(invocation.Name), err)
	}
	callback, err := retainAggressorUICallback(invocation, values[2], 2)
	if err != nil {
		return Null(), err
	}
	id, err := r.nextAggressorDialogIdentity()
	if err != nil {
		return Null(), err
	}
	owner := r.script(invocation.Script)
	dialog := &aggressorDialog{
		owner:         owner,
		generation:    invocation.generationToken(),
		runtimeID:     r.ID(),
		id:            id,
		callback:      callback,
		state:         aggressorUIBuilding,
		done:          make(chan struct{}),
		creatorScript: invocation.Script,
		creationSpan:  invocation.Span,
		title:         values[0].String(),
		defaults:      defaults,
	}
	dialogValue := ObjectValue(dialog)
	dialog.value = dialogValue
	if err := registerAggressorUIResource(invocation, owner, dialog); err != nil {
		dialog.revokeAggressorUI()
		return Null(), err
	}
	return dialogValue, nil
}

func (r *Runtime) dialogFromValue(value Value) (*aggressorDialog, error) {
	object, ok := value.Object()
	if !ok {
		return nil, errors.New("opfor: argument 1 is not an Aggressor dialog")
	}
	dialog, ok := object.(*aggressorDialog)
	if !ok || dialog == nil {
		return nil, errors.New("opfor: argument 1 is not an Aggressor dialog")
	}
	if r == nil || dialog.runtimeID != r.ID() {
		return nil, errors.New("opfor: Aggressor dialog belongs to another runtime")
	}
	return dialog, nil
}

func (r *Runtime) aggressorDialogDescription(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireAggressorUIArguments(invocation, 2, 3); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	if isNilInterface(r.aggressorDialogProvider) {
		return callAggressorUIHost(ctx, r, invocation)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	values := invocation.Values()
	dialog, err := r.dialogFromValue(values[0])
	if err != nil {
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
	}
	lines := int32(2)
	if len(values) == 3 {
		lines = values[2].Int32()
		if lines > 20 {
			lines = 20
		}
	}
	dialog.mu.Lock()
	if err := dialog.requireBuildingLocked(); err != nil {
		dialog.mu.Unlock()
		return Null(), err
	}
	dialog.description = values[1].String()
	dialog.descriptionLines = lines
	dialog.hasDescription = true
	dialog.descriptionScript = invocation.Script
	dialog.descriptionSpan = invocation.Span
	dialog.mu.Unlock()
	return Null(), nil
}

func (r *Runtime) aggressorDialogRow(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorDialogRowSpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{Operation: "Aggressor dialog row", Name: invocation.Name, Span: invocation.Span}
	}
	if err := requireAggressorUIArguments(invocation, spec.minimum, spec.maximum); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	if isNilInterface(r.aggressorDialogProvider) {
		return callAggressorUIHost(ctx, r, invocation)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	values := invocation.Values()
	dialog, err := r.dialogFromValue(values[0])
	if err != nil {
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
	}
	row := AggressorDialogRow{
		Kind:       spec.kind,
		Function:   invocation.Name,
		Name:       values[1].String(),
		Label:      values[2].String(),
		Default:    Null(),
		Deprecated: spec.deprecated,
		Equivalent: spec.equivalent,
		Script:     invocation.Script,
		Span:       invocation.Span,
	}
	if spec.checkbox {
		row.CheckboxText = values[3].String()
	}
	if spec.combobox {
		options, ok := values[3].Array()
		if !ok || options == nil {
			return Null(), fmt.Errorf("&%s: argument 4 must be an array", builtinName(invocation.Name))
		}
		row.Options = options.Values()
	}
	if spec.width && len(values) == 4 {
		row.Width = values[3].Int32()
		row.HasWidth = true
	}
	dialog.mu.Lock()
	if err := dialog.requireBuildingLocked(); err != nil {
		dialog.mu.Unlock()
		return Null(), err
	}
	for _, candidate := range dialog.defaults {
		if candidate.Name == row.Name {
			row.Default = candidate.Value
			row.HasDefault = true
			break
		}
	}
	dialog.nextRow++
	row.ID = dialog.nextRow
	dialog.rows = append(dialog.rows, row)
	dialog.mu.Unlock()
	return Null(), nil
}

func (r *Runtime) aggressorDialogButton(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireExactAggressorClientArguments(invocation, 2); err != nil {
		return Null(), err
	}
	if invocation.Name != "dbutton_action" && invocation.Name != "dbutton_help" {
		return Null(), &UnsupportedError{Operation: "Aggressor dialog button", Name: invocation.Name, Span: invocation.Span}
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	if isNilInterface(r.aggressorDialogProvider) {
		return callAggressorUIHost(ctx, r, invocation)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	values := invocation.Values()
	dialog, err := r.dialogFromValue(values[0])
	if err != nil {
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
	}
	button := AggressorDialogButton{Script: invocation.Script, Span: invocation.Span}
	if invocation.Name == "dbutton_action" {
		button.Kind = AggressorDialogButtonAction
		button.Label = values[1].String()
	} else {
		button.Kind = AggressorDialogButtonHelp
		button.Label = "Help"
		button.URL = values[1].String()
	}
	dialog.mu.Lock()
	if err := dialog.requireBuildingLocked(); err != nil {
		dialog.mu.Unlock()
		return Null(), err
	}
	dialog.nextButton++
	button.ID = dialog.nextButton
	dialog.buttons = append(dialog.buttons, button)
	dialog.mu.Unlock()
	return Null(), nil
}

func (r *Runtime) aggressorDialogShow(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireExactAggressorClientArguments(invocation, 1); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	provider := r.aggressorDialogProvider
	if isNilInterface(provider) {
		return callAggressorUIHost(ctx, r, invocation)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	values := invocation.Values()
	dialog, err := r.dialogFromValue(values[0])
	if err != nil {
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
	}
	presentation, err := dialog.presentation(ctx, invocation)
	if err != nil {
		return Null(), err
	}
	providerErr := provider.PresentAggressorDialog(ctx, presentation, dialog)
	contextErr := executionContextError(ctx)
	dialog.finishPresentationWindow(providerErr != nil || contextErr != nil)
	if contextErr == nil {
		// An enrolled concurrent callback may have unloaded its owner while the
		// presentation window drained. Observe that cancellation at the native
		// boundary instead of returning a stale success.
		contextErr = executionContextError(ctx)
	}
	if providerErr != nil {
		return Null(), preserveNativeBoundaryError(ctx, providerErr)
	}
	if contextErr != nil {
		return Null(), contextErr
	}
	return Null(), nil
}

type aggressorPrompt struct {
	mu sync.Mutex

	owner      *Script
	generation *scriptGeneration
	id         AggressorPromptID
	kind       AggressorPromptKind
	callback   Callable
	state      aggressorUIState
	done       chan struct{}
	window     *aggressorUIPresentationWindow
}

func (prompt *aggressorPrompt) Done() <-chan struct{} {
	if prompt == nil || prompt.done == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return prompt.done
}

func (prompt *aggressorPrompt) stateErrorLocked() error {
	if prompt.state == aggressorUIRevoked {
		return ErrScriptUnloaded
	}
	return fmt.Errorf("%w: prompt %d", ErrAggressorUIClosed, prompt.id)
}

func (prompt *aggressorPrompt) finishLocked(state aggressorUIState) (*Script, Callable) {
	owner := prompt.owner
	callback := prompt.callback
	prompt.owner = nil
	prompt.callback = nil
	prompt.state = state
	if prompt.done != nil {
		select {
		case <-prompt.done:
		default:
			close(prompt.done)
		}
	}
	return owner, callback
}

func (prompt *aggressorPrompt) fail() {
	if prompt == nil {
		return
	}
	prompt.mu.Lock()
	if prompt.state.terminal() {
		prompt.mu.Unlock()
		return
	}
	owner, _ := prompt.finishLocked(aggressorUIFailed)
	prompt.mu.Unlock()
	unregisterAggressorUIResource(owner, prompt)
}

func (prompt *aggressorPrompt) revokeAggressorUI() {
	if prompt == nil {
		return
	}
	prompt.mu.Lock()
	if prompt.state.terminal() {
		prompt.mu.Unlock()
		return
	}
	prompt.finishLocked(aggressorUIRevoked)
	prompt.mu.Unlock()
}

func (prompt *aggressorPrompt) aggressorUIGeneration() *scriptGeneration {
	if prompt == nil {
		return nil
	}
	return prompt.generation
}

func (prompt *aggressorPrompt) Dismiss() error {
	if prompt == nil {
		return errors.New("opfor: Aggressor prompt responder is nil")
	}
	prompt.mu.Lock()
	if prompt.state != aggressorUIPresenting && prompt.state != aggressorUIOpen {
		err := prompt.stateErrorLocked()
		prompt.mu.Unlock()
		return err
	}
	owner, _ := prompt.finishLocked(aggressorUIDismissed)
	prompt.mu.Unlock()
	unregisterAggressorUIResource(owner, prompt)
	return nil
}

func (prompt *aggressorPrompt) Accept(ctx context.Context, values ...Value) (Value, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	if prompt == nil {
		return Null(), errors.New("opfor: Aggressor prompt responder is nil")
	}
	prompt.mu.Lock()
	if prompt.state != aggressorUIPresenting && prompt.state != aggressorUIOpen {
		err := prompt.stateErrorLocked()
		prompt.mu.Unlock()
		return Null(), err
	}
	want := 1
	if prompt.kind == AggressorPromptConfirm {
		want = 0
	}
	if len(values) != want {
		prompt.mu.Unlock()
		return Null(), fmt.Errorf("opfor: Aggressor %s prompt response expects %d value(s), received %d", prompt.kind, want, len(values))
	}
	callbackContext, callbackDone := prompt.window.callbackContext(ctx)
	owner, callback := prompt.finishLocked(aggressorUICompleted)
	prompt.mu.Unlock()
	defer callbackDone()
	unregisterAggressorUIResource(owner, prompt)
	if callback == nil {
		return Null(), ErrInvalidCallable
	}
	return callback.Invoke(callbackContext, values...)
}

func (prompt *aggressorPrompt) finishPresentationWindow(failed bool) {
	if prompt == nil {
		return
	}
	prompt.mu.Lock()
	window := prompt.window
	drained := window.close()
	var owner *Script
	if failed && !prompt.state.terminal() {
		owner, _ = prompt.finishLocked(aggressorUIFailed)
	} else if !failed && prompt.state == aggressorUIPresenting {
		prompt.state = aggressorUIOpen
	}
	if prompt.window == window {
		prompt.window = nil
	}
	prompt.mu.Unlock()
	unregisterAggressorUIResource(owner, prompt)
	<-drained
}

func (r *Runtime) aggressorPrompt(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorPromptSpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{Operation: "Aggressor prompt", Name: invocation.Name, Span: invocation.Span}
	}
	if err := requireExactAggressorClientArguments(invocation, spec.arguments); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	provider := r.aggressorPromptProvider
	if isNilInterface(provider) {
		return callAggressorUIHost(ctx, r, invocation)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	values := invocation.Values()
	callback, err := retainAggressorUICallback(invocation, values[spec.callbackIndex], spec.callbackIndex)
	if err != nil {
		return Null(), err
	}
	id, err := r.nextAggressorPromptIdentity()
	if err != nil {
		return Null(), err
	}
	owner := r.script(invocation.Script)
	prompt := &aggressorPrompt{
		owner: owner, generation: invocation.generationToken(), id: id, kind: spec.kind, callback: callback,
		state: aggressorUIPresenting, done: make(chan struct{}),
		window: newAggressorUIPresentationWindow(ctx),
	}
	if err := registerAggressorUIResource(invocation, owner, prompt); err != nil {
		prompt.revokeAggressorUI()
		return Null(), err
	}
	presentation := AggressorPromptPresentation{
		ID: id, Kind: spec.kind, Name: invocation.Name, RuntimeID: r.ID(),
		Script: invocation.Script, Span: invocation.Span,
		Default: Null(), Multiple: Null(),
	}
	switch spec.kind {
	case AggressorPromptConfirm:
		presentation.Text = values[0].String()
		presentation.Title = values[1].String()
	case AggressorPromptText:
		presentation.Text = values[0].String()
		presentation.Default = values[1]
		presentation.HasDefault = true
	case AggressorPromptDirectoryOpen, AggressorPromptFileOpen:
		presentation.Title = values[0].String()
		presentation.Default = values[1]
		presentation.HasDefault = true
		presentation.Multiple = values[2]
		presentation.HasMultiple = true
		presentation.AllowMultiple = values[2].Truth()
	case AggressorPromptFileSave:
		presentation.Default = values[0]
		presentation.HasDefault = true
	}
	providerErr := provider.PresentAggressorPrompt(ctx, presentation, prompt)
	contextErr := executionContextError(ctx)
	prompt.finishPresentationWindow(providerErr != nil || contextErr != nil)
	if contextErr == nil {
		contextErr = executionContextError(ctx)
	}
	if providerErr != nil {
		return Null(), preserveNativeBoundaryError(ctx, providerErr)
	}
	if contextErr != nil {
		return Null(), contextErr
	}
	return Null(), nil
}
