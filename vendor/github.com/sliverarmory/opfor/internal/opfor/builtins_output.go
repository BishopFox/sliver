package opfor

import (
	"context"
	"fmt"
	"io"
	"strings"
)

func (r *Runtime) writeWarning(message string, span Span) {
	if r == nil || r.stderr == nil {
		return
	}
	if span.Source != "" {
		_, _ = fmt.Fprintf(r.stderr, "Warning: %s at %s:%d\n", message, sleepSourceDisplayName(span.Source), sleepDisplayLine(span))
		return
	}
	_, _ = fmt.Fprintf(r.stderr, "Warning: %s\n", message)
}

func (r *Runtime) installCoreFunctions() {
	core := r.coreFunctions(r.ioFunctions())
	// Retain the exact stateful stock implementations independently of the
	// public resolution table. A Sleep Function object recovered by function()
	// remains authoritative after setf() installs it under another key, even if
	// that key names an importer override. The map is immutable after New.
	r.stockFunctions = make(map[string]NativeFunc, len(core))
	for name, function := range core {
		r.stockFunctions[name] = function
	}
	for name, function := range core {
		if _, overridden := r.functions[name]; !overridden {
			r.functions[name] = function
		}
	}
}

// coreFunctions composes the disjoint Sleep and Aggressor default inventories.
// The I/O map is supplied separately because a live Runtime snapshots its
// working directory when that bridge is constructed, while metadata
// enumeration must remain independent of process state.
func (r *Runtime) coreFunctions(ioFunctions map[string]NativeFunc) map[string]NativeFunc {
	core := r.sleepFunctions(ioFunctions)
	r.mergeAggressorFunctions(core)
	return core
}

// sleepFunctions returns the installed Sleep native inventory. Stateful bridge
// maps are materialized exactly once for each Runtime and their NativeFunc
// values flow unchanged into stockFunctions and the public resolution table.
func (r *Runtime) sleepFunctions(ioFunctions map[string]NativeFunc) map[string]NativeFunc {
	functions := map[string]NativeFunc{
		"print":    r.print,
		"println":  r.println,
		"printf":   r.println,
		"printAll": r.printAll,
		"warn":     r.warn,
	}
	for name, function := range r.collectionFunctions() {
		functions[name] = function
	}
	// BasicUtilities.add ultimately mutates a Java List. Its invalid-position
	// exception is translated by Sleep's Block into a soft warning, so wrap the
	// portable default at installation time while leaving direct bridge calls
	// and importer overrides untouched.
	if add := functions["add"]; add != nil {
		functions["add"] = wrapAddMutation(add)
	}
	for name, function := range r.mutationFunctions() {
		functions[name] = function
	}
	for name, function := range r.stringNumberFunctions() {
		functions[name] = function
	}
	for name, function := range r.mathExtraFunctions() {
		functions[name] = function
	}
	for name, function := range r.utilityExtraFunctions() {
		functions[name] = function
	}
	for name, function := range sleepBinaryFunctions() {
		functions[name] = function
	}
	for name, function := range r.sleepSequenceFunctions() {
		// reverse is intentionally installed after stringNumberFunctions. Both
		// historical tranches expose the name; the sequence implementation is
		// the existing authoritative default.
		functions[name] = function
	}
	for name, function := range ioFunctions {
		functions[name] = function
	}
	for name, function := range r.sleepTimeFunctions() {
		functions[name] = function
	}
	for name, function := range r.sleepRuntimeFunctions() {
		functions[name] = function
	}
	for name, function := range r.concurrencyFunctions() {
		functions[name] = function
	}
	for name, function := range r.dynamicSourceFunctions() {
		functions[name] = function
	}
	removeEvidenceGatedFunctions(functions)
	return functions
}

// aggressorFunctions returns the installed Aggressor native inventory. Each
// tranche must be disjoint so a newly duplicated name fails at construction
// instead of silently changing resolution precedence.
func (r *Runtime) aggressorFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc)
	r.mergeAggressorFunctions(functions)
	return functions
}

// mergeAggressorFunctions installs each Aggressor tranche directly into the
// destination. coreFunctions uses this path to avoid building and copying an
// intermediate full Aggressor map during every Runtime construction, while
// aggressorFunctions retains the independently inspectable inventory used by
// profile and regression tests.
func (r *Runtime) mergeAggressorFunctions(functions map[string]NativeFunc) {
	for _, inventory := range []map[string]NativeFunc{
		r.aggressorPortableUtilityFunctions(),
		aggressorBinaryFunctions(),
		aggressorSequenceFunctions(),
		r.aggressorTimeFunctions(),
		r.aggressorRuntimeFunctions(),
	} {
		mergeDisjointFunctionInventory(functions, inventory)
	}
	removeEvidenceGatedFunctions(functions)
}

func mergeDisjointFunctionInventory(destination, source map[string]NativeFunc) {
	for name, function := range source {
		if _, exists := destination[name]; exists {
			panic(fmt.Sprintf("opfor: duplicate native function %q across disjoint inventories", name))
		}
		destination[name] = function
	}
}

// removeEvidenceGatedFunctions keeps conveniences with no pinned Sleep or
// Aggressor namespace evidence out of each installable inventory. Their pure-
// Go implementations remain in focused bridge tranches and an importer may opt
// into the same spelling with WithFunction. Otherwise unresolved calls reach
// Host instead of being silently claimed by OPFOR.
func removeEvidenceGatedFunctions(functions map[string]NativeFunc) {
	for name := range evidenceGatedExtensionFunctionNames {
		delete(functions, name)
	}
}

// evidenceGatedExtensionFunctionNames are implemented conveniences which are
// deliberately not stock functions. Add a name to the default namespace only
// after pinned Sleep source/JAR or official Aggressor documentation/corpus
// evidence establishes that OPFOR should claim it.
var evidenceGatedExtensionFunctionNames = map[string]struct{}{
	"-d":          {},
	"-e":          {},
	"-f":          {},
	"contains":    {},
	"containsAll": {},
	"copyFile":    {},
	"dirname":     {},
	"grep":        {},
	"isEmpty":     {},
	"item":        {},
	"lastIndexOf": {},
	"mapValues":   {},
	"menu":        {},
	"move":        {},
	"pwd":         {},
	"trim":        {},
	"unshift":     {},
	"zip":         {},
}

func (r *Runtime) print(ctx context.Context, invocation Invocation) (Value, error) {
	writer, values, err := outputTarget(r.consoleOutputWriter(), invocation.Arguments)
	if err != nil {
		return Null(), outputWarning(ctx, err)
	}
	writer = runtimeOutputWriterFor(r.resources, writer)
	text := ""
	if len(values) != 0 {
		value := outputArgumentValue(values[0])
		if r.warnUnsafeArrayOutput(invocation, value) {
			return Null(), nil
		}
		text = sleepOutputString(value)
	}
	// IOObject.print("") still flushes its OutputStreamWriter. Keep the
	// zero-argument and handle-only forms observable to buffered destinations.
	if _, err := io.WriteString(writer, text); err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	return Null(), nil
}

func (r *Runtime) println(ctx context.Context, invocation Invocation) (Value, error) {
	writer, values, err := outputTarget(r.consoleOutputWriter(), invocation.Arguments)
	if err != nil {
		return Null(), outputWarning(ctx, err)
	}
	writer = runtimeOutputWriterFor(r.resources, writer)
	var line string
	if len(values) != 0 {
		value := outputArgumentValue(values[0])
		if r.warnUnsafeArrayOutput(invocation, value) {
			return Null(), nil
		}
		line = sleepOutputString(value)
	}
	// PrintWriter.println emits one synchronized record. Keeping the payload
	// and terminator in one Write prevents concurrent fork fibers from merging
	// two lines at the boundary between separate writes.
	_, err = io.WriteString(writer, line+"\n")
	if err != nil {
		err = preserveNativeBoundaryError(ctx, err)
	}
	return Null(), err
}

func (r *Runtime) printAll(ctx context.Context, invocation Invocation) (Value, error) {
	writer, values, err := outputTarget(r.consoleOutputWriter(), invocation.Arguments)
	if err != nil {
		return Null(), outputWarning(ctx, err)
	}
	writer = runtimeOutputWriterFor(r.resources, writer)
	if len(values) == 0 {
		return Null(), nil
	}
	value := outputArgumentValue(values[0])
	if r.warnUnsafeArrayOutput(invocation, value) {
		return Null(), nil
	}
	cursor, err := newSequenceCursor(value, invocation.Name)
	if err != nil {
		// BridgeUtilities.getIterator throws this message without the calling
		// function name. Block turns it into a recoverable bridge warning and
		// terminates only the active block.
		prefix := "&" + builtinName(invocation.Name) + ": "
		return Null(), outputWarning(ctx, fmt.Errorf("%s", strings.TrimPrefix(err.Error(), prefix)))
	}
	for {
		item, present, nextErr := cursor.next(ctx)
		if nextErr != nil {
			return Null(), nextErr
		}
		if !present {
			return Null(), nil
		}
		if _, err := fmt.Fprintln(writer, sleepOutputString(item)); err != nil {
			return Null(), preserveNativeBoundaryError(ctx, err)
		}
	}
}

// outputWarning preserves the boundary between Sleep's bridge execution and
// direct Go use. BasicIO.chooseSource throws IllegalArgumentException for an
// invalid handle; Sleep's Block catches that exception, emits a warning, and
// terminates only the currently executing block. A direct Runtime.Invoke has
// no Block around it, so callers still receive the original typed error.
func outputWarning(ctx context.Context, err error) error {
	if err != nil && currentFiber(ctx) != nil {
		return &uncaughtScriptWarning{err: err}
	}
	return err
}

func (r *Runtime) warnUnsafeArrayOutput(invocation Invocation, value Value) bool {
	array, ok := value.Array()
	if !ok || array.viewError() == nil {
		return false
	}
	r.writeWarning(unsafeArrayViewWarning, invocation.Span)
	return true
}

func (r *Runtime) warn(ctx context.Context, invocation Invocation) (Value, error) {
	if currentFiber(ctx) != nil && invocation.Span.Source != "" {
		return r.sleepBridgeWarn(ctx, invocation)
	}

	writer, values, err := outputTarget(r.stderr, invocation.Arguments)
	if err != nil {
		return Null(), err
	}
	writer = runtimeOutputWriterFor(r.resources, writer)
	var message strings.Builder
	message.WriteString("Warning: ")
	for _, value := range values {
		message.WriteString(sleepOutputString(outputArgumentValue(value)))
	}
	if invocation.Span.Source != "" {
		fmt.Fprintf(&message, " at %s:%d", sleepSourceDisplayName(invocation.Span.Source), sleepDisplayLine(invocation.Span))
	}
	message.WriteByte('\n')
	_, err = io.WriteString(writer, message.String())
	if err != nil {
		err = preserveNativeBoundaryError(ctx, err)
	}
	return Null(), err
}

// sleepBridgeWarn consumes the Stack observed by BasicUtilities.evaluate.
// Lexical call-site injection is performed before function resolution, while
// a direct Function object invocation reaches this bridge with no hidden line.
func (r *Runtime) sleepBridgeWarn(ctx context.Context, invocation Invocation) (Value, error) {
	line := int32(-1)
	message := "warning requested"
	if len(invocation.Arguments) != 0 {
		message = sleepOutputString(invocation.Arg(0))
		if len(invocation.Arguments) > 1 {
			line = sleepInt32(invocation.Arg(1))
		}
	}

	writer := runtimeOutputWriterFor(r.resources, r.stderr)
	var record strings.Builder
	record.WriteString("Warning: ")
	record.WriteString(message)
	if invocation.Span.Source != "" {
		fmt.Fprintf(&record, " at %s:%d", sleepSourceDisplayName(invocation.Span.Source), line)
	}
	record.WriteByte('\n')
	_, err := io.WriteString(writer, record.String())
	if err != nil {
		err = preserveNativeBoundaryError(ctx, err)
	}
	return Null(), err
}

func sleepOutputString(value Value) string {
	if value.Kind() == KindString {
		return sleepCanonicalString(value)
	}
	return value.String()
}

func outputTarget(fallback io.Writer, arguments []Argument) (io.Writer, []Argument, error) {
	if len(arguments) == 0 {
		return fallback, nil, nil
	}
	if object, ok := arguments[0].Resolve().Object(); ok {
		if handle, ok := object.(*sleepIOHandle); ok && handle != nil {
			return sleepIOTextWriter{handle: handle}, arguments[1:], nil
		}
		if writer, ok := object.(io.Writer); ok {
			return writer, arguments[1:], nil
		}
	}
	if len(arguments) >= 2 {
		return nil, nil, fmt.Errorf("expected I/O handle argument, received: %s", arguments[0].Resolve().Describe())
	}
	return fallback, arguments, nil
}

func (r *Runtime) consoleOutputWriter() io.Writer {
	if r != nil && r.console != nil {
		return sleepIOTextWriter{handle: r.console}
	}
	if r == nil {
		return io.Discard
	}
	return r.stdout
}

func outputArgumentValue(argument Argument) Value {
	if argument.Name != "" {
		return ObjectValue(sleepKeyValue{key: String(argument.Name), value: argument.Resolve()})
	}
	return argument.Resolve()
}
