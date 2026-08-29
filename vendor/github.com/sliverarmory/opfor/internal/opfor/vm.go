package opfor

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/sliverarmory/opfor/internal/ast"
	"github.com/sliverarmory/opfor/internal/bytecode"
	"github.com/sliverarmory/opfor/internal/envspec"
)

type fiber struct {
	closure                *scriptClosure
	function               *bytecode.Function
	scope                  *scope
	locals                 []*scope
	caller                 *fiber
	inline                 bool
	flow                   inlineFlow
	inlineAt               map[*ast.CallExpr]*fiber
	dynamicSource          *dynamicSourceExecution
	continuation           *continuationCollector
	continuationTail       []*fiber
	continuedLoop          *bytecode.LoopRecovery
	binaryDepth            int
	yieldedInBinary        bool
	swallowCallCC          bool
	serializedForeach      *sleepSerializedForeachResume
	serializedMoreHandlers bool
	serializedReturn       bool
	pc                     int
	last                   Value

	iterators    []valueIterator
	tries        []tryFrame
	pending      Value
	lastMatch    []Value
	regexCursors map[string]*regexCursor
	callTraces   []*callTraceFrame
}

type tryFrame struct {
	handler int
	depth   int
}

type scriptThrow struct {
	value  Value
	frames []string
}

type inlineFlow uint8

const (
	inlineFlowNone inlineFlow = iota
	inlineFlowReturn
)

type loopControlKind uint8

const (
	loopControlBreak loopControlKind = iota
	loopControlContinue
)

// loopControl is Sleep's FLOW_CONTROL_BREAK/FLOW_CONTROL_CONTINUE request.
// It is deliberately distinct from an error: try/catch and Block warning
// recovery must not consume it. Named inline Blocks may carry the request into
// their caller, where the nearest active Goto Step handles it.
type loopControl struct{ kind loopControlKind }

func (*loopControl) Error() string { return "opfor: loop control" }

// continuationCollector is the ScriptEnvironment context stack accumulated
// during one top-level SleepClosure invocation. Dynamic eval/expr/include
// blocks may suspend without interrupting their caller, so more than one
// independently resumable fiber can belong to the same invocation. Sleep
// resumes those contexts in FIFO order as one group.
type continuationCollector struct {
	contexts []*fiber
}

func (c *continuationCollector) append(fibers ...*fiber) {
	if c == nil {
		return
	}
	for _, fiber := range fibers {
		if fiber != nil {
			c.contexts = append(c.contexts, fiber)
		}
	}
}

type inlineReturn struct{ value Value }

func (*inlineReturn) Error() string { return "opfor: inline return" }

type inlineYield struct{ value Value }

func (*inlineYield) Error() string { return "opfor: inline yield" }

// scriptExit is Sleep's empty FLOW_CONTROL_THROW marker. Unlike a thrown
// script value it deliberately bypasses every installed try/catch frame and
// unwinds only the current script invocation. A non-null argument also causes
// the Block containing exit() to emit one warning at the call site.
type scriptExit struct {
	message  string
	span     Span
	warn     bool
	reported bool
}

func (e *scriptExit) Error() string {
	if e != nil && e.warn {
		return e.message
	}
	return "script requested exit"
}

// callCCTransfer is an internal, non-exceptional control marker. A callcc
// instruction stops the current closure, parks this exact fiber, and hands the
// source closure to another script closure. The marker must reach the owning
// scriptClosure unchanged: try/catch and warning conversion are not involved
// in Sleep's continuation handoff.
type callCCTransfer struct {
	source          *scriptClosure
	fiber           *fiber
	target          *scriptClosure
	span            Span
	standaloneTrace bool
}

func (*callCCTransfer) Error() string { return "opfor: callcc transfer" }

// inlineAbort preserves Sleep 2.1's observable failure when an inline body
// yields from an ordinary binary expression: its Java operand frame is not
// resumable, so the owning block warns and returns the empty scalar.
type inlineAbort struct{ span Span }

func (*inlineAbort) Error() string { return "internal error - class java.util.EmptyStackException" }

func (e *scriptThrow) Error() string {
	if e == nil {
		return ""
	}
	return e.value.String()
}

func (e *scriptThrow) addFrame(frame string) {
	if e == nil || frame == "" {
		return
	}
	e.frames = append([]string{frame}, e.frames...)
}

func newFiber(closure *scriptClosure, frame *scope, arguments []Argument) *fiber {
	fiber, _ := newFiberAt(context.Background(), closure, frame, arguments, Span{})
	return fiber
}

func newFiberAt(ctx context.Context, closure *scriptClosure, frame *scope, arguments []Argument, span Span) (*fiber, error) {
	f := &fiber{
		closure:      closure,
		function:     closure.function,
		scope:        frame,
		continuation: &continuationCollector{},
	}
	if err := f.resetArgumentsAt(ctx, arguments, span); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *fiber) resetArguments(arguments []Argument) {
	_ = f.resetArgumentsAt(context.Background(), arguments, Span{})
}

func (f *fiber) resetArgumentsAt(ctx context.Context, arguments []Argument, span Span) error {
	if f == nil || f.scope == nil {
		return nil
	}
	positional := make([]*Cell, 0, len(arguments))
	position := 0
	for _, argument := range arguments {
		if argument.Name != "" {
			continue
		}
		position++
		cell := argument.Reference
		if cell == nil {
			cell = NewCell(argument.Resolve())
		}
		if err := f.scope.putCellAt(ctx, "$"+strconv.Itoa(position), cell, span); err != nil {
			return err
		}
		positional = append(positional, cell)
	}
	argumentsCell, err := f.scope.localAt(ctx, "@_", span)
	if err != nil {
		return err
	}
	var runtime *Runtime
	if f.closure != nil && f.closure.script != nil {
		runtime = f.closure.script.runtime
	}
	argumentsArray, err := newRuntimeArrayFromCells(runtime, positional)
	if err != nil {
		return err
	}
	argumentsCell.Set(ArrayValue(argumentsArray))
	if f.closure != nil {
		if err := f.closure.ensureStateAt(ctx, span); err != nil {
			return err
		}
		self, err := f.scope.localAt(ctx, "$this", span)
		if err != nil {
			return err
		}
		self.Set(FunctionValue(f.closure))
	}
	for _, argument := range arguments {
		if argument.Name == "" {
			continue
		}
		variable := normalizeVariableName(argument.Name)
		if argument.Reference != nil {
			if err := f.scope.putCellAt(ctx, variable, argument.Reference, span); err != nil {
				return err
			}
		} else {
			cell, err := f.scope.localAt(ctx, variable, span)
			if err != nil {
				return err
			}
			cell.Set(argument.Resolve())
		}
	}
	return nil
}

func (f *fiber) pushLocal(arguments []Argument) {
	_ = f.pushLocalAt(context.Background(), arguments, Span{})
}

func (f *fiber) pushLocalAt(ctx context.Context, arguments []Argument, span Span) error {
	previous := f.scope
	next, err := previous.nextLocalAt(ctx, span)
	if err != nil {
		return err
	}
	f.locals = append(f.locals, previous)
	f.scope = next
	if len(arguments) != 0 {
		if _, err := f.initializeLocalAt(ctx, arguments, span); err != nil {
			f.scope = previous
			f.locals[len(f.locals)-1] = nil
			f.locals = f.locals[:len(f.locals)-1]
			return err
		}
	}
	return nil
}

func (f *fiber) popLocal(arguments []Argument) bool {
	ok, _ := f.popLocalAt(context.Background(), arguments, Span{})
	return ok
}

func (f *fiber) popLocalAt(ctx context.Context, arguments []Argument, span Span) (bool, error) {
	if f == nil || len(f.locals) == 0 {
		return false, nil
	}
	last := len(f.locals) - 1
	f.scope = f.locals[last]
	f.locals[last] = nil
	f.locals = f.locals[:last]
	if len(arguments) != 0 {
		if _, err := f.initializeLocalAt(ctx, arguments, span); err != nil {
			return true, err
		}
	}
	return true, nil
}

func (f *fiber) initializeLocal(arguments []Argument) int {
	count, _ := f.initializeLocalAt(context.Background(), arguments, Span{})
	return count
}

func (f *fiber) initializeLocalAt(ctx context.Context, arguments []Argument, span Span) (int, error) {
	if f == nil || f.scope == nil {
		return 0, nil
	}
	positional := make([]*Cell, 0, len(arguments))
	for _, argument := range arguments {
		if argument.Name == "" {
			cell := argument.Reference
			if cell == nil {
				cell = NewCell(argument.Resolve())
			}
			positional = append(positional, cell)
			if err := f.scope.putCellAt(ctx, "$"+strconv.Itoa(len(positional)), cell, span); err != nil {
				return 0, err
			}
			continue
		}

		name := normalizeVariableName(argument.Name)
		cell := argument.Reference
		if cell == nil {
			cell = NewCell(argument.Resolve())
		}
		if err := f.scope.putCellAt(ctx, name, cell, span); err != nil {
			return 0, err
		}
	}
	var runtime *Runtime
	if f.closure != nil && f.closure.script != nil {
		runtime = f.closure.script.runtime
	}
	argumentsArray, err := newRuntimeArrayFromCells(runtime, positional)
	if err != nil {
		return 0, err
	}
	if err := f.scope.putCellAt(ctx, "@_", NewCell(ArrayValue(argumentsArray)), span); err != nil {
		return 0, err
	}
	return len(positional) + 1, nil
}

func (f *fiber) run(ctx context.Context) (Value, bool, error) {
	if f == nil || f.function == nil || f.closure == nil || f.closure.script == nil {
		return Null(), false, errors.New("opfor: invalid execution fiber")
	}
	if !f.closure.script.Active() {
		return Null(), false, ErrScriptUnloaded
	}
	if missing := f.serializedForeach; missing != nil {
		// SleepClosure.writeObject deliberately omits ScriptEnvironment metadata.
		// A yielded foreach therefore has its saved Block/Goto handles but not the
		// `iterators` Stack used by Iterate NEXT. Match the official 2.1 failure
		// instead of manufacturing a cursor that was never serialized.
		f.serializedForeach = nil
		f.closure.script.runtime.writeWarning("null value error", missing.span)
		f.closure.script.runtime.writeWarning("internal error - class java.util.EmptyStackException", missing.span)
		return Null(), false, nil
	}
	caller := currentFiber(ctx)
	previousCaller := f.caller
	if caller != f {
		f.caller = caller
	}
	defer func() { f.caller = previousCaller }()
	ctx = withCurrentFiber(ctx, f)
	meter, outputAccount := vmExecutionLimits(ctx)
	if meter == nil && outputAccount == nil {
		return f.runInstructionLoop(ctx, nil, nil)
	}
	return f.runInstructionLoop(ctx, meter, outputAccount)
}

// runInstructionLoop receives pre-resolved accounting state. In the ordinary
// unlimited case both pointers are nil, so the loop retains cancellation and
// unload cancellation checks without context lookups or atomic accounting on
// every instruction.
func (f *fiber) runInstructionLoop(ctx context.Context, meter *executionMeter, outputAccount *runtimeResourceAccount) (Value, bool, error) {
	done := ctx.Done()
	for f.pc >= 0 && f.pc < len(f.function.Instructions) {
		f.resetContinuedLoop()
		if done != nil {
			select {
			case <-done:
				return Null(), false, ctx.Err()
			default:
			}
		}
		if err := consumeInstructionLimits(meter, outputAccount); err != nil {
			return Null(), false, &RuntimeError{Script: f.closure.script.id, Span: f.function.Instructions[f.pc].Span, Cause: err}
		}
		instruction := f.function.Instructions[f.pc]
		value, yielded, finished, err := f.step(ctx, instruction)
		// A diagnostic or asynchronous producer may latch output exhaustion
		// while the final instruction is executing. Check before every error,
		// yield, and finished return so a terminal control transfer cannot escape
		// the same-execution fatal boundary without reaching another VM safe
		// point.
		if outputAccount != nil {
			if outputErr := outputAccount.outputLimitError(); outputErr != nil {
				return Null(), false, &RuntimeError{
					Script: f.closure.script.id,
					Span:   instruction.Span,
					Cause:  errors.Join(outputErr, err),
				}
			}
		}
		if err != nil {
			if isImportCompileError(err) {
				return Null(), false, err
			}
			// Resource exhaustion is a host-enforced execution boundary, not a
			// Sleep exception. Letting a script catch it would make quotas
			// advisory and permit execution to continue after the family budget
			// has been exhausted.
			if errors.Is(err, ErrResourceLimit) {
				var runtimeError *RuntimeError
				if errors.As(err, &runtimeError) {
					return Null(), false, err
				}
				return Null(), false, &RuntimeError{
					Script: f.closure.script.id,
					Span:   instruction.Span,
					Cause:  err,
				}
			}
			var control *loopControl
			if errors.As(err, &control) {
				if f.recoverLoopControl(control.kind) {
					continue
				}
				return Null(), false, control
			}
			var invalidClassCast *portableInvalidClassCastError
			if errors.As(err, &invalidClassCast) {
				f.closure.script.runtime.writeWarning(invalidClassCast.Error(), instruction.Span)
				f.pc++
				continue
			}
			var exited *scriptExit
			if errors.As(err, &exited) {
				if !exited.reported {
					exited.reported = true
					if exited.warn {
						span := exited.span
						if span.Source == "" {
							span = instruction.Span
						}
						f.closure.script.runtime.writeWarning(exited.message, span)
					}
				}
				if f.caller == nil {
					return Null(), false, nil
				}
				return Null(), false, exited
			}
			var aborted *inlineAbort
			if errors.As(err, &aborted) {
				f.closure.script.runtime.writeWarning(aborted.Error(), aborted.span)
				return Null(), false, nil
			}
			var returned *inlineReturn
			if errors.As(err, &returned) {
				f.flow = inlineFlowReturn
				return returned.value, false, nil
			}
			var yieldedInline *inlineYield
			if errors.As(err, &yieldedInline) {
				return yieldedInline.value, true, nil
			}
			var transfer *callCCTransfer
			if errors.As(err, &transfer) {
				return Null(), false, transfer
			}
			// Java bridge exceptions are consumed by Sleep's Block.evaluate
			// before a Try-installed script exception handler can observe them.
			// Preserve that boundary for warnings which model those exceptions:
			// they terminate only the active block and cannot be caught by Sleep
			// try/catch.
			var warning *uncaughtScriptWarning
			if errors.As(err, &warning) {
				span := instruction.Span
				var nested *RuntimeError
				if errors.As(err, &nested) && nested.Span.Source != "" {
					span = nested.Span
				}
				f.closure.script.runtime.writeWarning(warning.Error(), span)
				if f.recoverBlockWarning() {
					continue
				}
				return Null(), false, nil
			}
			if f.catch(err) {
				continue
			}
			var thrown *scriptThrow
			if errors.As(err, &thrown) && f.caller == nil && !f.serializedMoreHandlers {
				span := instruction.Span
				var nested *RuntimeError
				if errors.As(err, &nested) && nested.Span.Source != "" {
					span = nested.Span
				}
				f.closure.script.runtime.writeWarning("Uncaught exception: "+thrown.value.String(), span)
				return Null(), false, nil
			}
			var runtimeError *RuntimeError
			if errors.As(err, &runtimeError) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrScriptUnloaded) {
				return Null(), false, err
			}
			return Null(), false, &RuntimeError{Script: f.closure.script.id, Span: instruction.Span, Cause: err}
		}
		if yielded {
			return value, true, nil
		}
		if finished {
			return value, false, nil
		}
	}
	return f.last, false, nil
}

type uncaughtScriptWarning struct {
	err error
}

func (e *uncaughtScriptWarning) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *uncaughtScriptWarning) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (f *fiber) step(ctx context.Context, instruction bytecode.Instruction) (Value, bool, bool, error) {
	switch instruction.Op {
	case bytecode.OpEval:
		value, err := f.eval(ctx, instruction.Expr)
		if err == nil {
			f.last = value
			f.pc++
		}
		return value, false, false, err
	case bytecode.OpJump:
		if instruction.ClearResult {
			f.last = Null()
		}
		f.pc = instruction.Target
		return Null(), false, false, nil
	case bytecode.OpJumpFalse:
		value, err := f.evalBlockPredicate(ctx, instruction.Expr)
		if err != nil {
			return Null(), false, false, err
		}
		if !value.Truth() {
			f.pc = instruction.Target
		} else {
			f.pc++
		}
		return value, false, false, nil
	case bytecode.OpAssignWhile:
		value, err := f.eval(ctx, instruction.Expr)
		if err != nil {
			return Null(), false, false, f.predicateSetupError(err, instruction.Expr)
		}
		cell, err := f.variableCell(ctx, instruction.Name, instruction.Span)
		if err != nil {
			return Null(), false, false, err
		}
		if err := f.setCellAtExecution(ctx, cell, value, instruction.Span); err != nil {
			return Null(), false, false, err
		}
		// Sleep's extended `while $value (producer())` form is an iterator
		// loop: only $null ends it. Numeric zero and an empty-but-present
		// string are valid yielded/read values.
		present := !value.IsNull()
		f.tracePresencePredicate(value, present, instruction.Span)
		if !present {
			f.pc = instruction.Target
		} else {
			f.pc++
		}
		return value, false, false, nil
	case bytecode.OpIterInit:
		value, err := f.eval(ctx, instruction.Expr)
		if err != nil {
			return Null(), false, false, err
		}
		keyLevel, valueLevel, err := f.foreachDestinationLevels(ctx, instruction.Name, instruction.Name2, instruction.Span)
		if err != nil {
			return Null(), false, false, err
		}
		iterator, err := f.iterator(ctx, value, instruction.Span)
		if err != nil {
			if errors.Is(err, ErrUnsafeArrayView) {
				f.closure.script.runtime.writeWarning(unsafeArrayViewWarning, instruction.Span)
				f.iterators = append(f.iterators, &foreachScopeIterator{
					valueIterator: &sliceIterator{}, keyLevel: keyLevel, valueLevel: valueLevel,
				})
				f.pc++
				return Null(), false, false, nil
			}
			return Null(), false, false, err
		}
		f.iterators = append(f.iterators, &foreachScopeIterator{
			valueIterator: iterator, keyLevel: keyLevel, valueLevel: valueLevel,
		})
		f.pc++
		return Null(), false, false, nil
	case bytecode.OpIterNext:
		if len(f.iterators) == 0 {
			return Null(), false, false, errors.New("opfor: foreach iterator stack is empty")
		}
		iterator := f.iterators[len(f.iterators)-1]
		item, ok, err := iterator.next(ctx)
		if err != nil {
			// Importer iterators own their error contract. In a nested stock-native
			// callback importerValueIterator carries a nativeBoundaryError to the
			// outer call; at top level its raw error must likewise bypass Sleep's
			// internal concurrent-modification warning translations.
			if isImporterIterator(iterator) {
				return Null(), false, false, err
			}
			// Sleep's Iterate step turns a ConcurrentModificationException into
			// an exhausted iterator before the surrounding Block reports it as a
			// warning. The Block then advances to the loop body, which runs once
			// more with the previous key/value bindings before the exhausted
			// iterator ends the loop on its next visit.
			if errors.Is(err, ErrArrayChangedDuringIteration) {
				f.closure.script.runtime.writeWarning("unsafe data modification: @array changed during iteration", instruction.Span)
				f.pc++
				return Null(), false, false, nil
			}
			if errors.Is(err, ErrUnsafeArrayView) {
				f.closure.script.runtime.writeWarning(unsafeArrayViewWarning, instruction.Span)
				f.pc++
				return Null(), false, false, nil
			}
			if errors.Is(err, ErrHashChangedDuringIteration) {
				f.closure.script.runtime.writeWarning("detected unsafe data modification", instruction.Span)
				f.pc++
				return Null(), false, false, nil
			}
			return Null(), false, false, err
		}
		if !ok {
			f.pc = instruction.Target
			return Null(), false, false, nil
		}
		scoped, _ := iterator.(*foreachScopeIterator)
		keyLevel, valueLevel := f.scope.root, f.scope.root
		if scoped != nil {
			keyLevel, valueLevel = scoped.keyLevel, scoped.valueLevel
		}
		if instruction.Name != "" {
			if err := keyLevel.putCellAt(ctx, instruction.Name, NewCell(item.key), instruction.Span); err != nil {
				return Null(), false, false, err
			}
		}
		if instruction.Name == "" && item.single != nil {
			if err := valueLevel.putCellAt(ctx, instruction.Name2, NewCell(*item.single), instruction.Span); err != nil {
				return Null(), false, false, err
			}
		} else if item.cell != nil {
			if err := valueLevel.putCellAt(ctx, instruction.Name2, item.cell, instruction.Span); err != nil {
				return Null(), false, false, err
			}
		} else {
			if err := valueLevel.putCellAt(ctx, instruction.Name2, NewCell(item.value), instruction.Span); err != nil {
				return Null(), false, false, err
			}
		}
		f.pc++
		return item.value, false, false, nil
	case bytecode.OpIterDestroy:
		if len(f.iterators) == 0 {
			return Null(), false, false, errors.New("opfor: foreach iterator stack is empty")
		}
		f.iterators[len(f.iterators)-1] = nil
		f.iterators = f.iterators[:len(f.iterators)-1]
		f.pc++
		return Null(), false, false, nil
	case bytecode.OpEnterTry:
		f.tries = append(f.tries, tryFrame{handler: instruction.Target, depth: len(f.iterators)})
		f.pc++
		return Null(), false, false, nil
	case bytecode.OpLeaveTry:
		if len(f.tries) != 0 {
			f.tries = f.tries[:len(f.tries)-1]
		}
		f.pc++
		return Null(), false, false, nil
	case bytecode.OpCatch:
		if instruction.Name != "" {
			cell, err := f.scope.resolveAt(ctx, instruction.Name, instruction.Span)
			if err != nil {
				return Null(), false, false, err
			}
			if err := f.setCellAtExecution(ctx, cell, f.pending, instruction.Span); err != nil {
				return Null(), false, false, err
			}
		}
		f.pending = Null()
		f.pc++
		return Null(), false, false, nil
	case bytecode.OpReturn:
		value, err := f.evalOptional(ctx, instruction.Expr)
		if err == nil {
			f.flow = inlineFlowReturn
		}
		return value, false, err == nil, err
	case bytecode.OpBreak, bytecode.OpContinue:
		if _, err := f.evalOptional(ctx, instruction.Expr); err != nil {
			return Null(), false, false, err
		}
		kind := loopControlBreak
		if instruction.Op == bytecode.OpContinue {
			kind = loopControlContinue
		}
		return Null(), false, false, &loopControl{kind: kind}
	case bytecode.OpYield:
		value, err := f.evalOptional(ctx, instruction.Expr)
		if err == nil {
			f.pc++
		}
		return value, err == nil, false, err
	case bytecode.OpThrow:
		value, err := f.evalOptional(ctx, instruction.Expr)
		if err != nil {
			return Null(), false, false, err
		}
		// Sleep's Return atom only raises FLOW_CONTROL_THROW for a non-empty
		// scalar. A bare throw or an explicit $null therefore consumes the
		// return frame and lets the current block continue normally.
		if value.IsNull() {
			f.last = Null()
			f.pc++
			return Null(), false, false, nil
		}
		origin := ""
		if instruction.Span.Source != "" {
			origin = fmt.Sprintf("   %s:%d <origin of exception>", instruction.Span.Source, sleepDisplayLine(instruction.Span))
		}
		return Null(), false, false, &scriptThrow{value: value, frames: []string{origin}}
	case bytecode.OpAssert:
		condition, err := f.evalBlockPredicate(ctx, instruction.Expr)
		if err != nil {
			return Null(), false, false, err
		}
		if !condition.Truth() {
			message := "assertion failed"
			if instruction.Message != nil {
				value, messageErr := f.eval(ctx, instruction.Message)
				if messageErr != nil {
					return Null(), false, false, messageErr
				}
				message = value.String()
			}
			exit := &scriptExit{message: message, span: instruction.Span, warn: true}
			if f.callTraceEnabled() {
				f.writeCallTrace(formatCall("exit", []Argument{{Value: String(message)}}), Null(), exit, instruction.Span)
			}
			return Null(), false, false, exit
		}
		f.pc++
		return condition, false, false, nil
	case bytecode.OpBind:
		environment := environmentKindFromAST(instruction.Environment)
		selectors := make([]BindingSelector, 0, len(instruction.Selectors))
		for _, selector := range instruction.Selectors {
			bound := BindingSelector{Raw: selector.Raw, Span: selector.Span()}
			if environment == EnvironmentOrdinary {
				bound.Evaluated = true
				bound.Value = String(selector.Value)
				if selector.Expr != nil {
					value, err := f.eval(ctx, selector.Expr)
					if err != nil {
						return Null(), false, false, err
					}
					bound.Value = value
				}
			} else if environment == EnvironmentFilter {
				// BindFilter passes both identifier and parameter strings to the
				// bridge verbatim; quoted text retains its quote delimiters.
				bound.Value = String(selector.Raw)
			}
			selectors = append(selectors, bound)
		}
		name := ""
		if len(selectors) != 0 && environment != EnvironmentPredicate {
			name = selectors[0].Value.String()
		}
		if name == "" && environment != EnvironmentPredicate {
			return Null(), false, false, fmt.Errorf("opfor: %s declaration has no name", instruction.Keyword)
		}
		registeredKind, registered := f.closure.script.runtime.registeredEnvironment(instruction.Keyword)
		if registered && registeredKind != environment {
			registered = false
		}
		if !registered && f.closure.script.runtime.observer == nil {
			message := "Attempting to bind code to non-existent environment: " + instruction.Keyword
			if environment != EnvironmentOrdinary {
				message = "Attempting to bind code to non-existent predicate environment: " + instruction.Keyword
			}
			f.closure.script.runtime.writeWarning(message, instruction.Span)
			if name != "" && f.closure.script.resolveFunction(name) == nil {
				if err := f.closure.script.setFunction(name, nil); err != nil {
					return Null(), false, false, err
				}
			}
			f.pc++
			return Null(), false, false, nil
		}
		closureMode := envspec.ClosureCurrent
		if spec, ok := envspec.LookupFold(instruction.Keyword); ok {
			closureMode = spec.Closure
		}
		var closure *scriptClosure
		switch closureMode {
		case envspec.ClosureInline:
			closure = f.closure.script.newInline(instruction.Body, f.scope)
		case envspec.ClosureRoot:
			// DefaultEnvironment wraps a sub body in a new SleepClosure. That
			// closure receives an independent internal scope and does not capture
			// the declaration site's active local/this levels.
			closure = f.closure.script.newClosure(instruction.Body, f.scope.root)
		default:
			// Importer-defined and Aggressor environments may deliberately retain
			// their composition frame. In particular nested popup/menu/item bodies
			// inherit the parent invocation arguments. That is separate from
			// DefaultEnvironment's stock sub contract above.
			closure = f.closure.script.newClosure(instruction.Body, f.scope)
		}
		binding := Binding{
			Kind: bindingKind(instruction.Keyword), Keyword: instruction.Keyword,
			Environment: environment, Name: name, Span: instruction.Span, Selectors: selectors,
		}
		if environment == EnvironmentFilter && len(selectors) > 1 {
			binding.Filter = selectors[1].Raw
		}
		if environment == EnvironmentPredicate {
			binding.Predicate = &scriptPredicateEvaluator{
				script: f.closure.script, expression: instruction.Predicate,
				captured: f.scope, span: instruction.Span,
			}
		}
		err := f.closure.script.registerBinding(ctx, binding, closure)
		if err == nil {
			f.last = FunctionValue(closure)
			f.pc++
		}
		return f.last, false, false, err
	case bytecode.OpImport:
		if instruction.ImportFrom == nil {
			f.closure.script.addImport(instruction.ImportTarget)
			f.pc++
			return Null(), false, false, nil
		}

		from, err := f.eval(ctx, instruction.ImportFrom)
		if err != nil {
			return Null(), false, false, err
		}
		runtime := f.closure.script.runtime
		inspection, err := runtime.inspectImportArchive(ctx, instruction.ImportTarget, from.String())
		if err != nil {
			return Null(), false, false, err
		}
		diagnosticSpan := importInstructionDiagnosticSpan(f.closure.script, instruction.Span)

		_, err = runtime.invoke(ctx, Invocation{
			Script: f.closure.script.id,
			Name:   "import",
			Span:   instruction.Span,
			Arguments: []Argument{
				{Value: String(instruction.ImportTarget)},
				{Value: from},
			},
		})
		if err != nil {
			var unsupported *UnsupportedError
			if !errors.As(err, &unsupported) {
				return Null(), false, false, err
			}
			if inspection.local && !inspection.exists {
				return Null(), false, false, importCompileError(
					diagnosticImportArchiveNotFound,
					messageImportArchiveNotFound,
					diagnosticSpan,
				)
			}
			supported := portableFixtureImport(runtime, f.closure.script.id, instruction.ImportTarget, from.String())
			if !supported && inspection.entryChecked && !inspection.hasEntry {
				return Null(), false, false, importCompileError(
					diagnosticImportedClassNotFound,
					messageImportedClassNotFound,
					diagnosticSpan,
				)
			}
			if !supported && !runtime.hasImportObjectDelegate() {
				return Null(), false, false, err
			}
		}
		f.closure.script.addImport(instruction.ImportTarget)
		f.pc++
		return Null(), false, false, nil
	case bytecode.OpHalt:
		f.flow = inlineFlowReturn
		return Int(2), false, true, nil
	case bytecode.OpDone:
		f.flow = inlineFlowReturn
		return Int(1), false, true, nil
	case bytecode.OpCallCC:
		callableValue, err := f.eval(ctx, instruction.Expr)
		if err != nil {
			return Null(), false, false, err
		}
		callable, ok := callableValue.Function()
		target, scriptClosure := callable.(*scriptClosure)
		if !ok || !scriptClosure || target == nil {
			f.closure.script.runtime.writeWarning("callcc requires a function: "+callableValue.Describe(), instruction.Span)
			f.pc++
			return callableValue, true, false, nil
		}
		if !f.swallowCallCC {
			f.traceCallCCTransfer(f.closure, target, instruction.Span)
		}
		standaloneTrace := !f.swallowCallCC && f.caller == nil && f.callTraceEnabled()
		f.pc++
		return Null(), false, false, &callCCTransfer{
			source:          f.closure,
			fiber:           f,
			target:          target,
			span:            instruction.Span,
			standaloneTrace: standaloneTrace,
		}
	case bytecode.OpEnd:
		// A script body exposes its last expression to Runtime.Eval/Execute.
		// Sleep closures and named functions, however, return the empty scalar
		// when control reaches the end without an explicit return. This is also
		// the sentinel that terminates a yielded closure used as an iterator.
		if f.function.Name == "<main>" {
			return f.last, false, true, nil
		}
		return Null(), false, true, nil
	default:
		return Null(), false, false, fmt.Errorf("opfor: unknown opcode %d", instruction.Op)
	}
}

// foreachDestinationLevels mirrors Iterate.iterator_create. Unlike a normal
// Get, checking an empty destination does not install a scalar: Iterate keeps
// the selected Variable container and only putScalar creates/replaces the
// binding when an item arrives. The current local, closure, and global levels
// are checked once at iterator creation, even when strict diagnostics are off.
func (f *fiber) foreachDestinationLevels(ctx context.Context, key, value string, span Span) (*scope, *scope, error) {
	if f == nil || f.scope == nil {
		return nil, nil, errors.New("opfor: foreach has no variable scope")
	}
	missing := ""
	valueLevel, valueExists, err := f.scope.levelAt(ctx, value, span)
	if err != nil {
		return nil, nil, err
	}
	if value != "" {
		if !valueExists {
			missing = value
		}
	}
	var keyLevel *scope
	if key != "" {
		var keyExists bool
		keyLevel, keyExists, err = f.scope.levelAt(ctx, key, span)
		if err != nil {
			return nil, nil, err
		}
		if !keyExists {
			missing = key
		}
	}
	if missing != "" && f.strictVariablesEnabled() && f.closure.script.runtime != nil {
		f.closure.script.runtime.writeWarning("variable '"+normalizeVariableName(missing)+"' not declared", span)
	}
	return keyLevel, valueLevel, nil
}

func (f *fiber) evalOptional(ctx context.Context, expression ast.Expr) (Value, error) {
	if expression == nil {
		return Null(), nil
	}
	return f.eval(ctx, expression)
}

func (f *fiber) catch(err error) bool {
	if len(f.tries) == 0 {
		return false
	}
	frame := f.tries[len(f.tries)-1]
	f.tries = f.tries[:len(f.tries)-1]
	// A deserialized Goto context may carry the handler responsible for its
	// owning try Block. Once an earlier saved context propagates a throw here,
	// the handler supersedes the metadata-omitted foreach resume.
	f.serializedForeach = nil
	if frame.depth < len(f.iterators) {
		clear(f.iterators[frame.depth:])
		f.iterators = f.iterators[:frame.depth]
	}
	var thrown *scriptThrow
	if errors.As(err, &thrown) {
		f.pending = thrown.value
		if f.closure != nil && f.closure.script != nil {
			f.closure.script.setStackTrace(thrown.frames)
		}
	} else {
		f.pending = String(err.Error())
	}
	f.pc = frame.handler
	return true
}

// recoverBlockWarning applies the innermost compiler-recorded Sleep Block
// boundary. Block.evaluate consumes Java bridge exceptions before a script
// try/catch can observe them, returns the empty scalar, and lets its owning
// Decide/Goto/Try Step continue. OPFOR's flattened bytecode restores that
// behavior by resuming at the corresponding target and unwinding only state
// owned by the abandoned Block.
func (f *fiber) recoverBlockWarning() bool {
	if f == nil || f.function == nil {
		return false
	}
	best := -1
	bestWidth := 0
	for index, recovery := range f.function.BlockRecoveries {
		if f.pc < recovery.Start || f.pc >= recovery.End {
			continue
		}
		if recovery.Target < 0 || recovery.Target >= len(f.function.Instructions) {
			continue
		}
		width := recovery.End - recovery.Start
		if best < 0 || width < bestWidth {
			best = index
			bestWidth = width
		}
	}
	if best < 0 {
		return false
	}
	recovery := f.function.BlockRecoveries[best]
	if recovery.TryDepth < len(f.tries) {
		clear(f.tries[recovery.TryDepth:])
		f.tries = f.tries[:recovery.TryDepth]
	}
	if recovery.IteratorDepth < len(f.iterators) {
		clear(f.iterators[recovery.IteratorDepth:])
		f.iterators = f.iterators[:recovery.IteratorDepth]
	}
	f.pending = Null()
	f.last = Null()
	f.pc = recovery.Target
	return true
}

func (f *fiber) resetContinuedLoop() {
	if f == nil || f.continuedLoop == nil {
		return
	}
	recovery := f.continuedLoop
	if f.pc == recovery.BodyStart || f.pc < recovery.Start || f.pc >= recovery.End {
		f.continuedLoop = nil
	}
}

// recoverLoopControl applies the innermost Goto boundary containing the
// currently executing instruction. Break is caught across the whole Goto;
// continue is caught only while its true-body Block is running. Once caught,
// continue's condition/increment executes outside that one boundary, allowing
// a second continue there to propagate to an enclosing loop just as Sleep's
// Goto.evaluate does.
func (f *fiber) recoverLoopControl(kind loopControlKind) bool {
	if f == nil || f.function == nil {
		return false
	}
	best := -1
	bestWidth := 0
	for index := range f.function.LoopRecoveries {
		recovery := &f.function.LoopRecoveries[index]
		start, end := recovery.Start, recovery.End
		if kind == loopControlContinue {
			if recovery == f.continuedLoop {
				continue
			}
			start, end = recovery.BodyStart, recovery.BodyEnd
		}
		if f.pc < start || f.pc >= end {
			continue
		}
		width := end - start
		if best < 0 || width < bestWidth {
			best = index
			bestWidth = width
		}
	}
	if best < 0 {
		return false
	}
	recovery := &f.function.LoopRecoveries[best]
	if recovery.TryDepth < len(f.tries) {
		clear(f.tries[recovery.TryDepth:])
		f.tries = f.tries[:recovery.TryDepth]
	}
	if recovery.IteratorDepth < len(f.iterators) {
		clear(f.iterators[recovery.IteratorDepth:])
		f.iterators = f.iterators[:recovery.IteratorDepth]
	}
	f.pending = Null()
	f.last = Null()
	if kind == loopControlBreak {
		f.continuedLoop = nil
		f.pc = recovery.BreakTarget
	} else {
		f.continuedLoop = recovery
		f.pc = recovery.ContinueTarget
	}
	return true
}

type valueIterator interface {
	next(context.Context) (item iteratorItem, ok bool, err error)
	remove(context.Context) error
	sourceValue() Value
}

type foreachScopeIterator struct {
	valueIterator
	keyLevel   *scope
	valueLevel *scope
}

func isImporterIterator(iterator valueIterator) bool {
	if scoped, ok := iterator.(*foreachScopeIterator); ok && scoped != nil {
		iterator = scoped.valueIterator
	}
	_, ok := iterator.(*importerValueIterator)
	return ok
}

type iteratorItem struct {
	key    Value
	value  Value
	cell   *Cell
	single *Value
}

type sliceIterator struct {
	source Value
	keys   []Value
	values []Value
	cells  []*Cell
	index  int
}

func (i *sliceIterator) next(context.Context) (iteratorItem, bool, error) {
	if i == nil || i.index >= len(i.values) {
		return iteratorItem{}, false, nil
	}
	index := i.index
	i.index++
	key := Int(int32(index))
	if index < len(i.keys) {
		key = i.keys[index]
	}
	item := iteratorItem{key: key, value: i.values[index]}
	if index < len(i.cells) {
		item.cell = i.cells[index]
	}
	return item, true, nil
}

func (i *sliceIterator) remove(context.Context) error {
	return errors.New("opfor: iterator does not support removal")
}

func (i *sliceIterator) sourceValue() Value {
	if i == nil {
		return Null()
	}
	return i.source
}

type arrayIterator struct {
	source           Value
	storage          *arrayStorage
	window           *arrayWindow
	expectedModCount uint64
	nextIndex        int
	count            int
	lastIndex        int
	canRemove        bool
	stopped          bool
}

func newArrayIterator(value Value, array *Array) (*arrayIterator, error) {
	storage, window := array.arrayStorage()
	if storage == nil || window == nil {
		return &arrayIterator{source: value, lastIndex: -1}, nil
	}
	storage.mu.RLock()
	defer storage.mu.RUnlock()
	if !window.valid {
		return nil, unsafeArrayViewError()
	}
	return &arrayIterator{
		source:           value,
		storage:          storage,
		window:           window,
		expectedModCount: storage.modCount,
		lastIndex:        -1,
	}, nil
}

func (i *arrayIterator) next(context.Context) (iteratorItem, bool, error) {
	if i == nil || i.storage == nil || i.window == nil || i.stopped {
		return iteratorItem{}, false, nil
	}
	i.storage.mu.RLock()
	if !i.window.valid {
		i.stopped = true
		i.storage.mu.RUnlock()
		return iteratorItem{}, false, unsafeArrayViewError()
	}
	// MyLinkedList.MyListIterator.hasNext() compares its cursor with the
	// current list size before next() performs the modification check. Thus a
	// mutation that makes the cursor exactly equal to the new size ends the
	// loop quietly, while other structural mutations reach next() and warn.
	length := i.window.end - i.window.start
	if i.nextIndex == length {
		i.storage.mu.RUnlock()
		return iteratorItem{}, false, nil
	}
	if i.expectedModCount != i.storage.modCount {
		i.stopped = true
		i.storage.mu.RUnlock()
		return iteratorItem{}, false, ErrArrayChangedDuringIteration
	}
	if i.nextIndex < 0 || i.nextIndex >= length {
		i.storage.mu.RUnlock()
		return iteratorItem{}, false, nil
	}
	index := i.nextIndex
	cell := i.storage.items[i.window.start+index]
	i.nextIndex++
	i.lastIndex = index
	i.canRemove = true
	key := Int(int32(i.count))
	i.count++
	i.storage.mu.RUnlock()
	return iteratorItem{key: key, value: cell.Get(), cell: cell}, true, nil
}

func (i *arrayIterator) remove(context.Context) error {
	if i == nil || i.storage == nil || i.window == nil {
		return errors.New("opfor: iterator does not support removal")
	}
	i.storage.mu.Lock()
	defer i.storage.mu.Unlock()
	if !i.window.valid {
		return unsafeArrayViewError()
	}
	if i.expectedModCount != i.storage.modCount {
		return ErrArrayChangedDuringIteration
	}
	if !i.canRemove || i.lastIndex < 0 || i.lastIndex >= i.window.end-i.window.start {
		return errors.New("opfor: iterator has no current element")
	}

	absolute := i.window.start + i.lastIndex
	copy(i.storage.items[absolute:], i.storage.items[absolute+1:])
	i.storage.items[len(i.storage.items)-1] = nil
	i.storage.items = i.storage.items[:len(i.storage.items)-1]
	i.storage.modCount++
	for candidate := range i.storage.views {
		if candidate != i.storage.root && candidate != i.window {
			candidate.valid = false
		}
	}
	i.window.end--
	i.storage.root.start = 0
	i.storage.root.end = len(i.storage.items)
	i.storage.root.valid = true
	i.storage.syncCachesLocked()

	i.expectedModCount = i.storage.modCount
	i.nextIndex--
	i.count--
	i.lastIndex = -1
	i.canRemove = false
	return nil
}

func (i *arrayIterator) sourceValue() Value {
	if i == nil {
		return Null()
	}
	return i.source
}

type hashIterator struct {
	source           Value
	hash             *Hash
	keys             []string
	expectedModCount uint64
	index            int
	current          string
	canRemove        bool
	stopped          bool
}

// ErrHashChangedDuringIteration reports a structural hash mutation made
// outside the active foreach iterator. Sleep surfaces the corresponding
// ConcurrentModificationException as "detected unsafe data modification".
var ErrHashChangedDuringIteration = errors.New("opfor: hash changed during iteration")

func newHashIterator(value Value, hash *Hash) *hashIterator {
	iterator := &hashIterator{source: value, hash: hash}
	if hash == nil {
		return iterator
	}
	hash.mu.RLock()
	iterator.keys = hash.compatibleKeysLocked()
	iterator.expectedModCount = hash.modCount
	hash.mu.RUnlock()
	return iterator
}

func (i *hashIterator) next(context.Context) (iteratorItem, bool, error) {
	if i == nil || i.hash == nil || i.stopped {
		return iteratorItem{}, false, nil
	}
	for i.index < len(i.keys) {
		i.hash.mu.RLock()
		if i.expectedModCount != i.hash.modCount {
			i.stopped = true
			i.hash.mu.RUnlock()
			return iteratorItem{}, false, ErrHashChangedDuringIteration
		}
		key := i.keys[i.index]
		i.index++
		cell, ok := i.hash.items[key]
		keyValue := i.hash.keyValueLocked(key)
		if ok && i.hash.readOnly {
			cell = NewCell(cell.Get())
		}
		i.hash.mu.RUnlock()
		if !ok || cell.Get().IsNull() {
			continue
		}
		i.current = key
		i.canRemove = true
		return iteratorItem{key: keyValue, value: cell.Get(), cell: cell, single: &keyValue}, true, nil
	}
	return iteratorItem{}, false, nil
}

func (i *hashIterator) remove(context.Context) error {
	if i == nil || i.hash == nil || !i.canRemove {
		return errors.New("opfor: iterator has no current element")
	}
	i.hash.mu.Lock()
	defer i.hash.mu.Unlock()
	if i.hash.readOnly {
		// Sleep foreach iterates MapWrapper.getData(), which is a detached
		// HashMap snapshot. Removing the current entry therefore succeeds for
		// iteration bookkeeping without changing the wrapped map.
		i.canRemove = false
		return nil
	}
	if i.expectedModCount != i.hash.modCount {
		i.stopped = true
		return ErrHashChangedDuringIteration
	}
	if _, ok := i.hash.items[i.current]; !ok {
		return errors.New("opfor: hash changed during iteration")
	}
	delete(i.hash.items, i.current)
	delete(i.hash.keyValues, i.current)
	for index, candidate := range i.hash.order {
		if candidate == i.current {
			i.hash.order = append(i.hash.order[:index], i.hash.order[index+1:]...)
			break
		}
	}
	i.hash.modCount++
	i.expectedModCount = i.hash.modCount
	i.canRemove = false
	return nil
}

func (i *hashIterator) sourceValue() Value {
	if i == nil {
		return Null()
	}
	return i.source
}

type closureIterator struct {
	source  Value
	closure Callable
	index   int
	done    bool
}

func (i *closureIterator) next(ctx context.Context) (iteratorItem, bool, error) {
	if i == nil || i.done {
		return iteratorItem{}, false, nil
	}
	value, err := invokeTracedClosure(ctx, i.source, "eval", nil, i.closure)
	if err != nil {
		return iteratorItem{}, false, err
	}
	if value.IsNull() {
		i.done = true
		return iteratorItem{}, false, nil
	}
	key := Int(int32(i.index))
	i.index++
	return iteratorItem{key: key, value: value}, true, nil
}

func (i *closureIterator) remove(context.Context) error {
	if i != nil && i.index > 0 {
		i.index--
	}
	return nil
}

func (i *closureIterator) sourceValue() Value {
	if i == nil {
		return Null()
	}
	return i.source
}

type javaValueIterator struct {
	source   Value
	iterator *portableJavaIterator
	index    int
}

type importerValueIterator struct {
	source   Value
	iterator Iterator
	index    int
}

func (i *importerValueIterator) next(ctx context.Context) (iteratorItem, bool, error) {
	if i == nil || i.iterator == nil {
		return iteratorItem{}, false, nil
	}
	if err := ctx.Err(); err != nil {
		return iteratorItem{}, false, err
	}
	value, present, err := i.iterator.Next(ctx)
	if err != nil {
		err = preserveNativeBoundaryError(ctx, err)
	}
	if err != nil || !present {
		return iteratorItem{}, false, err
	}
	key := Int(int32(i.index))
	i.index++
	return iteratorItem{key: key, value: value}, true, nil
}

func (i *importerValueIterator) remove(ctx context.Context) error {
	if i == nil || i.iterator == nil {
		return errors.New("opfor: iterator does not support removal")
	}
	mutable, ok := i.iterator.(MutableIterator)
	if !ok {
		return errors.New("opfor: iterator does not support removal")
	}
	if err := mutable.Remove(ctx); err != nil {
		return preserveNativeBoundaryError(ctx, err)
	}
	if i.index > 0 {
		i.index--
	}
	return nil
}

func (i *importerValueIterator) sourceValue() Value {
	if i == nil {
		return Null()
	}
	return i.source
}

func (i *javaValueIterator) next(ctx context.Context) (iteratorItem, bool, error) {
	if i == nil || i.iterator == nil {
		return iteratorItem{}, false, nil
	}
	if err := ctx.Err(); err != nil {
		return iteratorItem{}, false, err
	}
	hasNext, _, err := i.iterator.invoke(ObjectInvocation{Op: ObjectInvoke, Message: "hasNext"})
	if err != nil || !hasNext.Truth() {
		return iteratorItem{}, false, err
	}
	value, _, err := i.iterator.invoke(ObjectInvocation{Op: ObjectInvoke, Message: "next"})
	if err != nil {
		return iteratorItem{}, false, err
	}
	key := Int(int32(i.index))
	i.index++
	return iteratorItem{key: key, value: value}, true, nil
}

func (i *javaValueIterator) remove(context.Context) error {
	if i == nil || i.iterator == nil {
		return errors.New("opfor: iterator does not support removal")
	}
	_, _, err := i.iterator.invoke(ObjectInvocation{Op: ObjectInvoke, Message: "remove"})
	if err == nil && i.index > 0 {
		i.index--
	}
	return err
}

func (i *javaValueIterator) sourceValue() Value {
	if i == nil {
		return Null()
	}
	return i.source
}

func (f *fiber) iterator(ctx context.Context, value Value, span Span) (valueIterator, error) {
	switch value.Kind() {
	case KindArray:
		array, _ := value.Array()
		if array != nil && array.backend != nil {
			return array.backend.iterator(value), nil
		}
		return newArrayIterator(value, array)
	case KindHash:
		hash, _ := value.Hash()
		if hash != nil && hash.backend != nil {
			// Iterate.iterator_create calls ScalarHash.getData(), whose MapWrapper
			// implementation builds a detached HashMap before iteration begins.
			snapshot, err := hash.backend.dataSnapshotReserved(func(count int) error {
				return reserveCollectionEntriesAtExecution(ctx, f.closure.script, count)
			})
			if err != nil {
				return nil, err
			}
			return newHashIterator(value, snapshot), nil
		}
		return newHashIterator(value, hash), nil
	case KindFunction:
		callable, _ := value.Function()
		if _, closure := callable.(sleepSequenceClosure); closure {
			return &closureIterator{source: value, closure: callable}, nil
		}
	case KindObject:
		object, _ := value.Object()
		if iterator, ok := object.(*portableJavaIterator); ok && iterator != nil {
			return &javaValueIterator{source: value, iterator: iterator}, nil
		}
		if iterator, ok := object.(Iterator); ok && iterator != nil {
			return &importerValueIterator{source: value, iterator: iterator}, nil
		}
	}
	if f != nil && f.closure != nil && f.closure.script != nil {
		f.closure.script.runtime.writeWarning(
			fmt.Sprintf("Attempted to use foreach on non-array: '%s'", value.String()),
			span,
		)
	}
	return &sliceIterator{source: value}, nil
}

func variableDisplay(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "$null"
	}
	return name
}
