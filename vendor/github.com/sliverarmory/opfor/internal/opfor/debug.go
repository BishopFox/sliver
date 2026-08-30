package opfor

import (
	"context"
	"fmt"
	"strings"

	"github.com/sliverarmory/opfor/internal/bytecode"
)

const (
	debugRequireStrict int32 = 4
	debugTraceCalls    int32 = 8
	debugTraceSuppress int32 = 16
)

// callTraceFrame retains the Java CallRequest state that remains observable
// while a Sleep call is executing. A callcc instruction changes the eventual
// trace from an ordinary return into two messages: the original call reports a
// -goto- transfer immediately, and the continuation target reports its result
// as a synthetic closure call named CALLCC after it finishes.
type callTraceFrame struct {
	call     string
	span     Span
	transfer *callCCTrace
	deferred bool
}

type callCCTrace struct {
	source *scriptClosure
	target *scriptClosure
	span   Span
}

func (f *fiber) callTraceEnabled() bool {
	if f == nil || f.closure == nil || f.closure.script == nil {
		return false
	}
	script := f.closure.script
	script.mu.RLock()
	flags := script.debug
	script.mu.RUnlock()
	return flags&debugTraceCalls == debugTraceCalls && flags&debugTraceSuppress != debugTraceSuppress
}

func (f *fiber) callTraceEmissionEnabled() bool {
	if f == nil || f.closure == nil || f.closure.script == nil {
		return false
	}
	script := f.closure.script
	script.mu.RLock()
	flags := script.debug
	script.mu.RUnlock()
	// ScriptInstance.fireWarning does not require DEBUG_TRACE_CALLS when it
	// receives an already-started trace. It only requires debugging to remain
	// nonzero and DEBUG_TRACE_SUPPRESS to remain clear.
	return flags != 0 && flags&debugTraceSuppress != debugTraceSuppress
}

func formatCall(name string, arguments []Argument) string {
	var builder strings.Builder
	builder.WriteByte('&')
	builder.WriteString(strings.TrimPrefix(name, "&"))
	builder.WriteByte('(')
	for index, argument := range arguments {
		if index != 0 {
			builder.WriteString(", ")
		}
		if argument.Name != "" {
			builder.WriteString(argument.Name)
			builder.WriteString(" => ")
		}
		builder.WriteString(describeTraceValue(argument.Resolve()))
	}
	builder.WriteByte(')')
	return builder.String()
}

func formatClosureCall(closure Value, message string, arguments []Argument) string {
	var builder strings.Builder
	builder.WriteByte('[')
	builder.WriteString(describeTraceValue(closure))
	if message != "" {
		builder.WriteByte(' ')
		builder.WriteString(message)
	}
	if len(arguments) != 0 {
		builder.WriteString(": ")
		for index, argument := range arguments {
			if index != 0 {
				builder.WriteString(", ")
			}
			if argument.Name != "" {
				builder.WriteString(argument.Name)
				builder.WriteString(" => ")
			}
			builder.WriteString(describeTraceValue(argument.Resolve()))
		}
	}
	builder.WriteByte(']')
	return builder.String()
}

func (f *fiber) beginCallTrace(call string, span Span) *callTraceFrame {
	if f == nil || !f.callTraceEnabled() {
		return nil
	}
	frame := &callTraceFrame{call: call, span: span}
	f.callTraces = append(f.callTraces, frame)
	return frame
}

func (f *fiber) finishCallTrace(frame *callTraceFrame, result Value, callErr error) {
	if f == nil || frame == nil {
		return
	}
	for index := len(f.callTraces) - 1; index >= 0; index-- {
		if f.callTraces[index] != frame {
			continue
		}
		copy(f.callTraces[index:], f.callTraces[index+1:])
		f.callTraces[len(f.callTraces)-1] = nil
		f.callTraces = f.callTraces[:len(f.callTraces)-1]
		break
	}
	if frame.deferred {
		return
	}
	// ScriptInstance.fireWarning checks the current flags when a completed
	// CallRequest emits its trace. A call such as debug(0) therefore suppresses
	// its own pending trace even though tracing was enabled when it began.
	if !f.callTraceEmissionEnabled() {
		return
	}
	if frame.transfer == nil {
		f.writeCallTrace(frame.call, result, callErr, frame.span)
		return
	}
	transfer := frame.transfer
	call := formatCallCCTrace(transfer.source, transfer.target)
	f.writeCallTrace(call, result, callErr, transfer.span)
}

func formatCallCCTrace(source, target *scriptClosure) string {
	return formatClosureCall(
		FunctionValue(target),
		"CALLCC",
		[]Argument{{Value: FunctionValue(source)}},
	)
}

func (f *fiber) traceCallCCTransfer(source, target *scriptClosure, span Span) {
	if f == nil || f.caller == nil || source == nil || target == nil {
		return
	}
	caller := f.caller
	if len(caller.callTraces) == 0 {
		return
	}
	frame := caller.callTraces[len(caller.callTraces)-1]
	if frame == nil || frame.transfer != nil {
		return
	}
	caller.writeTraceMessage(frame.call+" -goto- "+describeTraceValue(FunctionValue(target)), frame.span)
	frame.transfer = &callCCTrace{source: source, target: target, span: span}
}

func describeTraceValue(value Value) string {
	callable, ok := value.Function()
	if !ok {
		return value.Describe()
	}
	closure, ok := callable.(*scriptClosure)
	if !ok || closure == nil || closure.function == nil || closure.id == 0 {
		return value.Describe()
	}

	start, end := 0, 0
	for _, instruction := range closure.function.Instructions {
		if instruction.Op == bytecode.OpEnd {
			continue
		}
		line := instruction.Span.Start.Line
		if line > 0 {
			if start == 0 || line < start {
				start = line
			}
			if line > end {
				end = line
			}
		}
		// Sleep's block range includes the step that creates a callcc target,
		// but not the target closure's body. OPFOR keeps that closure as the
		// instruction operand, so account for its opening line explicitly.
		if instruction.Op == bytecode.OpCallCC && instruction.Expr != nil {
			line = instruction.Expr.Span().Start.Line
			if line > 0 && line > end {
				end = line
			}
		}
	}
	hasInstruction := start != 0
	if !hasInstruction {
		start, end = closure.function.Span.Start.Line, closure.function.Span.Start.Line
	}
	// A closure whose only compiled operation starts on one line may still
	// contain a multi-line parsed literal. The bytecode keeps the block span,
	// so use its last interior line when no instruction exposed the range. Do
	// not broaden partially executed continuations such as callcc sources,
	// which already have a distinct multi-line range above.
	if hasInstruction && start == end && closure.function.Span.End.Line > end+1 {
		end = closure.function.Span.End.Line - 1
	}
	source := closure.function.Span.Source
	start = sleepDisplayLineNumber(source, start)
	end = sleepDisplayLineNumber(source, end)
	location := fmt.Sprintf("%s:%d", sleepSourceDisplayName(source), start)
	if end > start {
		location = fmt.Sprintf("%s:%d-%d", sleepSourceDisplayName(source), start, end)
	}
	return fmt.Sprintf("&closure[%s]#%d", location, closure.id)
}

func (f *fiber) writeCallTrace(call string, result Value, callErr error, span Span) {
	if f == nil || f.closure == nil || f.closure.script == nil || f.closure.script.runtime == nil {
		return
	}
	writer := f.closure.script.runtime.stderr
	if writer == nil {
		return
	}
	message := call
	if callErr != nil {
		message += " - FAILED!"
	} else if !result.IsNull() {
		message += " = " + describeTraceValue(result)
	}
	f.writeTraceMessage(message, span)
}

func (f *fiber) writeTraceMessage(message string, span Span) {
	if f == nil || f.closure == nil || f.closure.script == nil || f.closure.script.runtime == nil {
		return
	}
	writer := f.closure.script.runtime.stderr
	if writer == nil {
		return
	}
	if span.Source != "" {
		_, _ = fmt.Fprintf(writer, "Trace: %s at %s:%d\n", message, sleepSourceDisplayName(span.Source), sleepDisplayLine(span))
		return
	}
	_, _ = fmt.Fprintf(writer, "Trace: %s\n", message)
}

func invokeTracedClosure(ctx context.Context, closureValue Value, message string, values []Value, callable Callable) (Value, error) {
	caller := currentFiber(ctx)
	if caller == nil || !caller.callTraceEnabled() {
		return callable.Invoke(ctx, values...)
	}
	arguments := make([]Argument, len(values))
	for index, value := range values {
		arguments[index] = Argument{Value: value}
	}
	frame := caller.beginCallTrace(
		formatClosureCall(closureValue, message, arguments),
		Span{Source: "<internal>", Start: Position{Line: -1}},
	)
	result, err := callable.Invoke(ctx, values...)
	caller.finishCallTrace(frame, result, err)
	return result, err
}
