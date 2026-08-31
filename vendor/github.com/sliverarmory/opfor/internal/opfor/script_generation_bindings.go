package opfor

import (
	"context"
	"fmt"
)

// scriptGenerationCallable is the importer-facing view of a raw Sleep
// callable. The raw callable deliberately survives portable ScriptLoader
// unload; this wrapper belongs to exactly one registration generation.
type scriptGenerationCallable struct {
	owner      *Script
	generation *scriptGeneration
	callable   Callable
}

func (callable *scriptGenerationCallable) String() string {
	if callable == nil || callable.callable == nil {
		return "&closure"
	}
	return fmt.Sprint(callable.callable)
}

func (callable *scriptGenerationCallable) Invoke(
	ctx context.Context,
	arguments ...Value,
) (result Value, resultErr error) {
	return callable.invokeNamed(ctx, "", arguments...)
}

func (callable *scriptGenerationCallable) invokeNamed(
	ctx context.Context,
	name string,
	arguments ...Value,
) (result Value, resultErr error) {
	if callable == nil || callable.owner == nil || callable.callable == nil {
		return Null(), ErrScriptUnloaded
	}
	executionCtx, release, err := callable.owner.acquireGenerationExecution(ctx, callable.generation)
	if err != nil {
		return Null(), err
	}
	defer func() { resultErr = joinExecutionError(resultErr, release) }()
	if name != "" {
		if named, ok := callable.callable.(namedBindingCallable); ok {
			result, resultErr = named.invokeNamed(executionCtx, name, arguments...)
		} else {
			result, resultErr = callable.callable.Invoke(executionCtx, arguments...)
		}
	} else {
		result, resultErr = callable.callable.Invoke(executionCtx, arguments...)
	}
	resultErr = joinExecutionContextError(executionCtx, resultErr)
	return result, resultErr
}

func generationForBinding(binding Binding) *scriptGeneration {
	if callable, ok := binding.Callback.(*scriptGenerationCallable); ok && callable != nil {
		return callable.generation
	}
	if predicate, ok := binding.Predicate.(*scriptPredicateEvaluator); ok && predicate != nil {
		return predicate.generation
	}
	return nil
}

func bindScriptGenerationCallable(
	owner *Script,
	generation *scriptGeneration,
	callable Callable,
) Callable {
	if callable == nil {
		return nil
	}
	return &scriptGenerationCallable{owner: owner, generation: generation, callable: callable}
}

// scriptLifetimeCallback is reserved for portable Sleep facilities whose raw
// callbacks survive ScriptLoader registry unload (notably BasicIO sockets).
// Importer-facing callbacks must use invocationCallback instead.
type scriptLifetimeCallback struct {
	owner    *Script
	callable Callable
}

func (callback *scriptLifetimeCallback) Invoke(
	ctx context.Context,
	arguments ...Value,
) (result Value, resultErr error) {
	return callback.invokeNamed(ctx, "", arguments...)
}

func (callback *scriptLifetimeCallback) invokeNamed(
	ctx context.Context,
	name string,
	arguments ...Value,
) (result Value, resultErr error) {
	if callback == nil || callback.owner == nil {
		return Null(), ErrScriptUnloaded
	}
	if callback.callable == nil {
		return Null(), ErrInvalidCallable
	}
	executionCtx, release, err := callback.owner.acquireExecution(ctx)
	if err != nil {
		return Null(), err
	}
	defer func() { resultErr = joinExecutionError(resultErr, release) }()
	if name != "" {
		if named, ok := callback.callable.(interface {
			invokeNamed(context.Context, string, ...Value) (Value, error)
		}); ok {
			result, resultErr = named.invokeNamed(executionCtx, name, arguments...)
		} else {
			result, resultErr = callback.callable.Invoke(executionCtx, arguments...)
		}
	} else {
		result, resultErr = callback.callable.Invoke(executionCtx, arguments...)
	}
	resultErr = joinExecutionContextError(executionCtx, resultErr)
	return result, resultErr
}

func retainScriptLifetimeCallback(invocation Invocation, value Value) (Callable, error) {
	callable, ok := value.Function()
	if !ok {
		return nil, ErrInvalidCallable
	}
	if invocation.Runtime == nil {
		return nil, fmt.Errorf("opfor: invocation has no originating runtime")
	}
	owner := invocation.Runtime.script(invocation.Script)
	if owner == nil || !owner.Active() {
		return nil, ErrScriptUnloaded
	}
	return &scriptLifetimeCallback{owner: owner, callable: callable}, nil
}
