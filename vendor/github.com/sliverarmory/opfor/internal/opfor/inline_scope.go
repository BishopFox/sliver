package opfor

import (
	"context"
	"errors"
	"strconv"

	"github.com/sliverarmory/opfor/internal/ast"
)

// invokeInline executes an inline declaration against the caller's active
// local level. The nil call key is retained for internal callers that do not
// originate directly from an AST call expression.
func (f *fiber) invokeInline(ctx context.Context, closure *scriptClosure, arguments []Argument) (Value, error) {
	return f.invokeInlineAt(ctx, nil, closure, arguments, true)
}

// invokeInlineAt executes either an inline declaration (initializeArguments)
// or the inline() builtin against the owning fiber. A yielded inline body is
// retained under its call expression so re-entering the same VM instruction
// resumes after yield instead of replaying the body from its beginning.
func (f *fiber) invokeInlineAt(ctx context.Context, call *ast.CallExpr, closure *scriptClosure, arguments []Argument, initializeArguments bool) (Value, error) {
	if f == nil || f.scope == nil || closure == nil || closure.function == nil {
		return Null(), ErrInvalidCallable
	}
	if suspended := f.inlineFiber(call); suspended != nil {
		return f.resumeInlineAt(ctx, call, suspended)
	}

	base := f.scope
	var oldArguments inlineArgumentState
	restoreCount := 0
	if initializeArguments {
		var err error
		oldArguments, err = captureInlineArgumentsAt(ctx, base, closure.function.Span)
		if err != nil {
			return Null(), err
		}
		restoreCount, err = f.initializeLocalAt(ctx, arguments, closure.function.Span)
		if err != nil {
			return Null(), err
		}
	}

	nested := &fiber{
		closure:         f.closure,
		function:        closure.function,
		scope:           f.scope,
		locals:          append([]*scope(nil), f.locals...),
		inline:          true,
		continuation:    f.continuation,
		yieldedInBinary: initializeArguments && f.binaryDepth != 0,
		swallowCallCC:   !initializeArguments || f.swallowCallCC,
		lastMatch:       append([]Value(nil), f.lastMatch...),
		regexCursors:    f.regexCursors,
	}
	result, yielded, err := nested.run(ctx)
	nested.swallowCallCC = false
	f.adoptInlineState(nested)
	var transfer *callCCTransfer
	if errors.As(err, &transfer) && transfer.fiber == nested && transfer.source == f.closure {
		if !initializeArguments {
			f.collectContinuation(nested)
			return Null(), nil
		}
		// The inline fiber is only the innermost portion of the continuation.
		// Retain it under the caller's instruction and park the owning fiber;
		// invoking the real source closure will re-enter this call site and then
		// resume the inline body after callcc. Sleep intentionally keeps the
		// continuation's replacement arguments in the caller's local level.
		f.setInlineFiber(call, nested)
		transfer.fiber = f
		return Null(), transfer
	}
	var control *loopControl
	if !initializeArguments && errors.As(err, &control) {
		// BasicUtilities.inline evaluates the closure Block directly, but the
		// surrounding FunctionCallRequest clears its flow request before the
		// caller resumes. A named inline declaration deliberately lacks this
		// boundary and continues propagating below.
		return Null(), nil
	}
	var restoreErr error
	if initializeArguments {
		restoreErr = restoreInlineArgumentsAt(ctx, oldArguments, restoreCount, closure.function.Span)
	}
	if err != nil {
		return result, errors.Join(err, restoreErr)
	}
	if restoreErr != nil {
		return result, restoreErr
	}
	if yielded {
		if !initializeArguments {
			f.collectContinuation(nested)
			return Null(), nil
		}
		f.setInlineFiber(call, nested)
		return result, &inlineYield{value: result}
	}
	if !initializeArguments {
		return Null(), nil
	}
	if nested.flow == inlineFlowReturn {
		return result, &inlineReturn{value: result}
	}
	return result, err
}

func (f *fiber) collectContinuation(suspended *fiber) {
	if f == nil || suspended == nil {
		return
	}
	if f.continuation == nil {
		f.continuation = &continuationCollector{}
	}
	f.continuation.append(suspended)
}

func (f *fiber) resumeInlineAt(ctx context.Context, call *ast.CallExpr, nested *fiber) (Value, error) {
	result, yielded, err := nested.run(ctx)
	f.adoptInlineState(nested)
	if err != nil {
		var transfer *callCCTransfer
		if errors.As(err, &transfer) && transfer.fiber == nested && transfer.source == f.closure {
			transfer.fiber = f
			return Null(), transfer
		}
		f.clearInlineFiber(call)
		return result, err
	}
	if yielded {
		return result, &inlineYield{value: result}
	}
	f.clearInlineFiber(call)
	if nested.yieldedInBinary {
		span := nested.function.Span
		if call != nil {
			span = call.Span()
		} else if f.pc >= 0 && f.pc < len(f.function.Instructions) {
			span = f.function.Instructions[f.pc].Span
		}
		return Null(), &inlineAbort{span: span}
	}
	if nested.flow == inlineFlowReturn {
		return result, &inlineReturn{value: result}
	}
	return result, nil
}

func (f *fiber) adoptInlineState(nested *fiber) {
	if f == nil || nested == nil {
		return
	}
	f.scope = nested.scope
	f.locals = nested.locals
	f.lastMatch = nested.lastMatch
	f.regexCursors = nested.regexCursors
}

// adoptContinuationState applies the one local-level/regex state saved for a
// SleepClosure top-level context to this fiber and every parked inline Block
// below it. Java serializes those Blocks as separate Context entries, but they
// all resume against the same final LinkedList of local levels. Updating only
// the carrier fiber would leave an inline body at its earlier yield-time scope.
func (f *fiber) adoptContinuationState(state *fiber) {
	if f == nil || state == nil {
		return
	}
	seen := make(map[*fiber]struct{})
	var adopt func(*fiber)
	adopt = func(current *fiber) {
		if current == nil {
			return
		}
		if _, duplicate := seen[current]; duplicate {
			return
		}
		seen[current] = struct{}{}
		if current != state {
			current.adoptInlineState(state)
		}
		for _, nested := range current.inlineAt {
			adopt(nested)
		}
	}
	adopt(f)
}

func (f *fiber) inlineFiber(call *ast.CallExpr) *fiber {
	if f == nil || f.inlineAt == nil {
		return nil
	}
	return f.inlineAt[call]
}

func (f *fiber) setInlineFiber(call *ast.CallExpr, nested *fiber) {
	if f.inlineAt == nil {
		f.inlineAt = make(map[*ast.CallExpr]*fiber)
	}
	f.inlineAt[call] = nested
}

func (f *fiber) clearInlineFiber(call *ast.CallExpr) {
	if f == nil || f.inlineAt == nil {
		return
	}
	delete(f.inlineAt, call)
}

func localScopeValue(frame *scope, name string) Value {
	if frame == nil {
		return Null()
	}
	name = normalizeVariableName(name)
	frame.mu.RLock()
	cell := frame.cells[name]
	frame.mu.RUnlock()
	if cell == nil {
		return Null()
	}
	return cell.Get()
}

type inlineArgumentState struct {
	frame     *scope
	arguments *Cell
}

func captureInlineArguments(frame *scope) inlineArgumentState {
	state, _ := captureInlineArgumentsAt(context.Background(), frame, Span{})
	return state
}

func captureInlineArgumentsAt(ctx context.Context, frame *scope, span Span) (inlineArgumentState, error) {
	state := inlineArgumentState{frame: frame}
	if frame == nil {
		return state, nil
	}
	cell, _, err := frame.ownCellAt(ctx, "@_", span)
	state.arguments = cell
	return state, err
}

func restoreInlineArguments(state inlineArgumentState, count int) {
	_ = restoreInlineArgumentsAt(context.Background(), state, count, Span{})
}

func restoreInlineArgumentsAt(ctx context.Context, state inlineArgumentState, count int, span Span) error {
	if state.frame == nil || state.arguments == nil {
		return nil
	}
	array, ok := state.arguments.Get().Array()
	if !ok {
		return nil
	}
	cells, err := array.snapshotCells()
	if err != nil {
		cells = array.cachedCells()
	}
	if count > len(cells) {
		count = len(cells)
	}
	if err := state.frame.putCellAt(ctx, "@_", state.arguments, span); err != nil {
		return err
	}
	for index := 0; index < count; index++ {
		if err := state.frame.putCellAt(ctx, "$"+strconv.Itoa(index+1), cells[index], span); err != nil {
			return err
		}
	}
	return nil
}
