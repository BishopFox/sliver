package opfor

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type intrinsicFunctionFamily uint8

const (
	intrinsicFamilyScope intrinsicFunctionFamily = iota + 1
	intrinsicFamilyLambda
	intrinsicFamilyFunction
	intrinsicFamilyUtilities
	intrinsicFamilyMatched
	intrinsicFamilyMatches
	intrinsicFamilyFind
)

// intrinsicNamedArgumentCallable models Sleep's actual bridge identity. A
// function("&name") handle is not itself a closure, but setf may install that
// bridge object under another key. CallRequest then passes the called key—not
// the key the handle came from—to Function.evaluate.
type intrinsicNamedArgumentCallable interface {
	Callable
	invokeNamedArgumentsAt(context.Context, string, Span, []Argument) (Value, error)
}

func newIntrinsicFunctionCallable(name string) *intrinsicFunctionCallable {
	name = strings.TrimPrefix(name, "&")
	family := intrinsicFamilyUtilities
	switch name {
	case "local", "global", "this":
		family = intrinsicFamilyScope
	case "lambda", "let", "compile_closure":
		family = intrinsicFamilyLambda
	case "function", "setf":
		family = intrinsicFamilyFunction
	case "matched":
		family = intrinsicFamilyMatched
	case "matches":
		family = intrinsicFamilyMatches
	case "find":
		family = intrinsicFamilyFind
	}
	return &intrinsicFunctionCallable{name: name, family: family}
}

func (c *intrinsicFunctionCallable) invokeNamedArgumentsAt(ctx context.Context, calledName string, span Span, arguments []Argument) (Value, error) {
	if c == nil || c.name == "" {
		return Null(), ErrInvalidCallable
	}
	fiber := currentFiber(ctx)
	if fiber == nil || fiber.closure == nil || fiber.closure.script == nil || fiber.closure.script.runtime == nil {
		return Null(), fmt.Errorf("opfor: intrinsic function &%s requires an active script", c.name)
	}
	calledName = strings.TrimPrefix(calledName, "&")
	if c.family == intrinsicFamilyUtilities && !intrinsicUtilitiesFunction(calledName) {
		if family, ok := sleepBuiltinFamilyForFunction(calledName); ok && family == sleepBuiltinFamilyUtilities {
			bridge := newSleepBuiltinFunctionCallable(c.name, family, fiber.closure.script.runtime, fiber.closure.script.id, span)
			return bridge.invokeNamedArgumentsAt(ctx, calledName, span, arguments)
		}
	}
	if c.family != intrinsicFamilyUtilities || calledName != "invoke" {
		arguments = sleepBridgeArguments(arguments)
	}
	switch c.family {
	case intrinsicFamilyScope:
		return fiber.invokeScopeBridgeAt(ctx, calledName, span, arguments)
	case intrinsicFamilyLambda:
		return fiber.invokeLambdaBridgeAt(ctx, calledName, span, arguments)
	case intrinsicFamilyFunction:
		return fiber.invokeFunctionBridgeAt(ctx, calledName, span, arguments)
	case intrinsicFamilyUtilities:
		return fiber.invokeUtilitiesBridgeAt(ctx, calledName, span, arguments)
	case intrinsicFamilyMatched:
		return fiber.intrinsicMatched()
	case intrinsicFamilyMatches:
		return fiber.intrinsicMatches(ctx, arguments)
	case intrinsicFamilyFind:
		return fiber.intrinsicFind(ctx, arguments)
	default:
		return Null(), ErrInvalidCallable
	}
}

func (f *fiber) invokeScopeBridgeAt(ctx context.Context, calledName string, span Span, arguments []Argument) (Value, error) {
	if len(arguments) == 0 {
		return Null(), sleepBridgeEmptyStack()
	}
	if calledName != "local" && calledName != "global" && calledName != "this" {
		return Null(), nil
	}
	declaration := arguments[0].Resolve().String()
	for _, variable := range strings.Fields(declaration) {
		if variable == "" || (variable[0] != '$' && variable[0] != '@' && variable[0] != '%') {
			return Null(), sleepBridgeIllegalArgument(fmt.Sprintf("&%s: malformed variable name '%s' from '%s'", calledName, variable, declaration))
		}
		switch calledName {
		case "local":
			if _, err := f.scope.localAt(ctx, variable, span); err != nil {
				return Null(), err
			}
		case "global":
			if _, err := f.scope.globalAt(ctx, variable, span); err != nil {
				return Null(), err
			}
		case "this":
			if err := f.closure.declareThis(ctx, variable); err != nil {
				return Null(), err
			}
		}
	}
	return Null(), nil
}

func (f *fiber) invokeLambdaBridgeAt(ctx context.Context, calledName string, span Span, arguments []Argument) (Value, error) {
	if len(arguments) == 0 {
		if calledName == "compile_closure" {
			return Null(), sleepBridgeEmptyStack()
		}
		return Null(), sleepBridgeIllegalArgument("expected &closure--received: $null")
	}

	var closure *scriptClosure
	if calledName == "compile_closure" {
		code := arguments[0].Resolve()
		if err := f.closure.script.runtime.rejectTaintedCall(ctx, calledName, arguments[:1]); err != nil {
			return Null(), err
		}
		program, err := f.closure.script.runtime.CompileString("eval", code.String())
		if err != nil {
			invocation := Invocation{Runtime: f.closure.script.runtime, Script: f.closure.script.id, Name: calledName, Span: span}
			return f.closure.script.runtime.flagSourceError(invocation, err)
		}
		function := *program.function
		function.Name = "<closure>"
		closure = f.closure.script.newClosure(&function, f.scope.root)
	} else {
		value := arguments[0].Resolve()
		callable, ok := value.Function()
		if !ok {
			return Null(), sleepBridgeIllegalArgument("expected &closure--received: " + value.Describe())
		}
		template, ok := callable.(*scriptClosure)
		if !ok || template == nil {
			return Null(), errors.New("opfor: expected a script closure")
		}
		if calledName == "lambda" {
			closure = f.closure.script.newClosure(template.function, f.scope.root)
		} else {
			closure = template
		}
	}
	if err := f.bindIntrinsicClosureArgumentsAt(ctx, closure, span, arguments[1:]); err != nil {
		return Null(), err
	}
	return FunctionValue(closure), nil
}

func (f *fiber) bindIntrinsicClosureArgumentsAt(ctx context.Context, closure *scriptClosure, span Span, arguments []Argument) error {
	if closure == nil {
		return errors.New("opfor: closure is nil")
	}
	for _, argument := range arguments {
		name, value, ok := sleepNamedArgument(argument)
		if !ok {
			return fmt.Errorf("opfor: malformed closure binding %s", argument.Resolve().Describe())
		}
		if name == "$this" {
			callable, ok := value.Function()
			if !ok {
				return fmt.Errorf("opfor: $this closure binding expected a closure, received %s", value.Describe())
			}
			source, ok := callable.(*scriptClosure)
			if !ok || source == nil {
				return errors.New("opfor: $this closure binding expected a script closure")
			}
			if source != closure {
				if err := source.ensureStateAt(ctx, span); err != nil {
					return err
				}
				source.mu.Lock()
				sharedState, sharedThis := source.state, source.thisHash
				source.mu.Unlock()
				closure.mu.Lock()
				closure.state, closure.thisHash = sharedState, sharedThis
				closure.mu.Unlock()
			}
			continue
		}
		if err := closure.ensureStateAt(ctx, span); err != nil {
			return err
		}
		closure.mu.Lock()
		state := closure.state
		closure.mu.Unlock()
		cell, err := state.localAt(ctx, name, span)
		if err != nil {
			return err
		}
		if err := f.setCellAtExecution(ctx, cell, value, span); err != nil {
			return err
		}
	}
	return nil
}

func (f *fiber) invokeFunctionBridgeAt(ctx context.Context, calledName string, span Span, arguments []Argument) (Value, error) {
	switch calledName {
	case "function":
		if len(arguments) == 0 {
			return Null(), sleepBridgeIllegalArgument("&function: requested function name must begin with '&'")
		}
		value := arguments[0].Resolve()
		if err := f.closure.script.runtime.rejectTaintedCall(ctx, calledName, arguments[:1]); err != nil {
			return Null(), err
		}
		functionName := value.String()
		if !strings.HasPrefix(functionName, "&") || len(functionName) == 1 {
			return Null(), sleepBridgeIllegalArgument("&function: requested function name must begin with '&'")
		}
		return FunctionValue(f.callable(functionName, span)), nil
	case "setf":
		return f.setIntrinsicFunction(arguments)
	default:
		return Null(), nil
	}
}

func (f *fiber) setIntrinsicFunction(arguments []Argument) (Value, error) {
	functionName := "&eh"
	if len(arguments) != 0 {
		functionName = arguments[0].Resolve().String()
	}
	callableValue := Null()
	if len(arguments) > 1 {
		callableValue = arguments[1].Resolve()
	}
	if !strings.HasPrefix(functionName, "&") || len(functionName) == 1 {
		return Null(), sleepBridgeIllegalArgument(fmt.Sprintf("&setf: invalid function name '%s'", functionName))
	}
	if callableValue.IsNull() {
		return Null(), f.closure.script.setFunction(functionName, nil)
	}
	callable, ok := callableValue.Function()
	if !ok {
		class := "sleep.engine.types.ObjectValue"
		if resolved, exists := portableObjectClass(callableValue); exists {
			class = resolved
		}
		return Null(), sleepBridgeIllegalArgument(fmt.Sprintf("&setf: can not set function %s to a class %s", functionName, class))
	}
	return Null(), f.closure.script.setFunction(functionName, callable)
}

func (f *fiber) invokeUtilitiesBridgeAt(ctx context.Context, calledName string, span Span, arguments []Argument) (Value, error) {
	switch calledName {
	case "checkError":
		invocation := Invocation{Runtime: f.closure.script.runtime, Script: f.closure.script.id, Name: calledName, Span: span}
		if len(arguments) != 0 {
			invocation.Arguments = arguments[:1]
		}
		return f.closure.script.runtime.checkError(ctx, invocation)
	case "getStackTrace":
		frames := f.closure.script.getStackTrace()
		values := make([]Value, len(frames))
		for index, frame := range frames {
			values[index] = ObjectValue(portableSleepStackElement{text: frame})
		}
		array, err := newRuntimeArray(f.closure.script.runtime, values...)
		if err != nil {
			return Null(), err
		}
		return ArrayValue(array), nil
	case "invoke":
		return f.invokeIntrinsicClosureAt(ctx, span, arguments)
	case "inline":
		if len(arguments) == 0 {
			return Null(), sleepBridgeIllegalArgument("expected &closure--received: $null")
		}
		value := arguments[0].Resolve()
		callable, ok := value.Function()
		if !ok {
			return Null(), sleepBridgeIllegalArgument("expected &closure--received: " + value.Describe())
		}
		closure, ok := callable.(*scriptClosure)
		if !ok || closure == nil {
			return Null(), errors.New("opfor: inline expected a script closure")
		}
		return f.invokeInlineAt(ctx, nil, closure, nil, false)
	default:
		return Null(), nil
	}
}

func (f *fiber) invokeIntrinsicClosureAt(ctx context.Context, span Span, arguments []Argument) (Value, error) {
	if len(arguments) == 0 {
		return Null(), sleepBridgeIllegalArgument("expected &closure--received: $null")
	}
	callee := arguments[0].Resolve()
	positional, named := extractSleepNamedArguments(arguments[1:])

	invokeArguments := make([]Argument, 0)
	if len(positional) != 0 {
		values, err := iteratorValues(ctx, positional[0].Resolve(), "invoke")
		if err != nil {
			return Null(), err
		}
		for _, value := range values {
			invokeArguments = append(invokeArguments, Argument{Value: value})
		}
	}
	message := Null()
	if len(positional) > 1 {
		message = String(positional[1].Resolve().String())
	}
	if argument, ok := named["message"]; ok {
		message = String(argument.Resolve().String())
	}
	if argument, ok := named["parameters"]; ok {
		hash, ok := argument.Resolve().Hash()
		if !ok || hash == nil {
			return Null(), errors.New("opfor: invoke parameters expected a hash")
		}
		keys, err := activeHashKeysAtExecution(ctx, f.closure.script, hash, true)
		if err != nil {
			return Null(), err
		}
		for _, key := range keys {
			value, err := hash.HashAtValue(ctx, key)
			if err != nil {
				return Null(), err
			}
			invokeArguments = append(invokeArguments, Argument{Name: key.String(), Value: value})
		}
	}
	invokeArguments = append(invokeArguments, Argument{Name: "$0", Value: message})
	environment := Null()
	if argument, ok := named["$this"]; ok {
		environment = argument.Resolve()
	}
	value, err := f.invokeCallableWithEnvironment(ctx, callee, invokeArguments, environment)
	var thrown *scriptThrow
	if errors.As(err, &thrown) {
		thrown.addFrame(fmt.Sprintf("   <internal>:-1 %s", describeTraceValue(callee)))
		frame := "&invoke()"
		if span.Source != "" {
			frame = fmt.Sprintf("   %s:%d %s", span.Source, sleepDisplayLine(span), frame)
		}
		thrown.addFrame(frame)
	}
	return value, err
}

func (f *fiber) intrinsicMatched() (Value, error) {
	array, err := newRuntimeArray(f.closure.script.runtime, f.lastMatch...)
	if err != nil {
		return Null(), err
	}
	return ArrayValue(array), nil
}

func (f *fiber) intrinsicMatches(ctx context.Context, arguments []Argument) (Value, error) {
	if len(arguments) < 2 {
		return Null(), sleepBridgeEmptyStack()
	}
	values := resolvedArguments(arguments)
	expression, err := f.closure.script.runtime.compileSleepRegexBridge(sleepCanonicalString(values[1]), false)
	if err != nil {
		return Null(), err
	}
	first, last := int32(-1), int32(-1)
	if len(values) > 2 {
		first = values[2].Int32()
		last = first
	}
	if len(values) > 3 {
		last = values[3].Int32()
	}
	text := sleepCanonicalString(values[0])
	allMatches, err := expression.FindAllStringSubmatchIndexContext(ctx, text, -1)
	if err != nil {
		return Null(), fmt.Errorf("opfor: regular expression match: %w", err)
	}
	selectedCount := 0
	for index, match := range allMatches {
		if int32(index) == first {
			selectedCount = 0
		}
		if len(match) >= 2 {
			selectedCount += len(match)/2 - 1
		}
		if int32(index) == last {
			break
		}
	}
	if err := reserveCollectionEntries(f.closure.script.runtime, selectedCount); err != nil {
		return Null(), err
	}
	matches := make([]Value, 0, selectedCount)
	for index, match := range allMatches {
		if int32(index) == first {
			matches = nil
		}
		matches = append(matches, sleepRegexCaptures(text, match)...)
		if int32(index) == last {
			break
		}
	}
	result := ArrayValue(NewArray(matches...))
	if len(f.closure.script.runtime.taintedValues(values[0], values[1])) != 0 {
		result = f.closure.script.runtime.TaintAll(result)
	}
	return result, nil
}

func (f *fiber) intrinsicFind(ctx context.Context, arguments []Argument) (Value, error) {
	values := make([]Value, len(arguments))
	if len(values) < 2 {
		values = make([]Value, 2)
	}
	for index := range values {
		values[index] = Null()
	}
	for index, argument := range arguments {
		values[index] = argument.Resolve()
	}
	text := sleepCanonicalString(values[0])
	expression, err := f.closure.script.runtime.compileSleepRegexBridge(sleepCanonicalString(values[1]), false)
	if err != nil {
		return Null(), err
	}
	start := 0
	if len(values) > 2 {
		start = int(values[2].Int32())
		if start < 0 {
			start += sleepUTF16Length(text)
		}
	}
	byteStart, ok := sleepUTF16ToByteIndex(text, start)
	if !ok {
		return Null(), fmt.Errorf("opfor: find start %d is outside the UTF-16 string boundary", start)
	}
	match, err := expression.FindStringSubmatchIndexAtContext(ctx, text, byteStart)
	if err != nil {
		return Null(), fmt.Errorf("opfor: regular expression match: %w", err)
	}
	if match == nil {
		f.lastMatch = nil
		return Null(), nil
	}
	f.lastMatch = sleepRegexCaptures(text, match)
	f.applyRegexTaint(values[0], values[1])
	return Int(int32(sleepUTF16Length(text[:match[0]]))), nil
}
