package opfor

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/sliverarmory/opfor/internal/ast"
	"github.com/sliverarmory/opfor/internal/bytecode"
	"github.com/sliverarmory/opfor/internal/envspec"
)

type bindingInvocationContextKey struct{}

func withBindingInvocation(ctx context.Context, invocation *BindingInvocation) context.Context {
	return context.WithValue(ctx, bindingInvocationContextKey{}, invocation)
}

func currentBindingInvocation(ctx context.Context) *BindingInvocation {
	if ctx == nil {
		return nil
	}
	invocation, _ := ctx.Value(bindingInvocationContextKey{}).(*BindingInvocation)
	return invocation
}

func cloneBindingInvocation(invocation *BindingInvocation) *BindingInvocation {
	if invocation == nil {
		return nil
	}
	return &BindingInvocation{
		BindingID: invocation.BindingID,
		Kind:      invocation.Kind,
		Keyword:   invocation.Keyword,
		Name:      invocation.Name,
		Script:    invocation.Script,
		Arguments: append([]Value(nil), invocation.Arguments...),
		Parent:    cloneBindingInvocation(invocation.Parent),
	}
}

func cloneBinding(binding Binding) Binding {
	binding.Selectors = append([]BindingSelector(nil), binding.Selectors...)
	binding.Parent = cloneBindingInvocation(binding.Parent)
	return binding
}

func bindingInvocation(binding Binding, arguments []Value) *BindingInvocation {
	return &BindingInvocation{
		BindingID: binding.ID,
		Kind:      binding.Kind,
		Keyword:   binding.Keyword,
		Name:      binding.Name,
		Script:    binding.Script,
		Arguments: append([]Value(nil), arguments...),
		Parent:    cloneBindingInvocation(binding.Parent),
	}
}

func (r *Runtime) registeredEnvironment(keyword string) (EnvironmentKind, bool) {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if r != nil {
		r.mu.RLock()
		kind, ok := r.environments[keyword]
		r.mu.RUnlock()
		if ok {
			return kind, true
		}
	}
	if knownBindingEnvironment(keyword) {
		return EnvironmentOrdinary, true
	}
	return EnvironmentOrdinary, false
}

func environmentKindFromAST(form ast.EnvironmentForm) EnvironmentKind {
	switch form {
	case ast.FilterEnvironment:
		return EnvironmentFilter
	case ast.PredicateEnvironment:
		return EnvironmentPredicate
	default:
		return EnvironmentOrdinary
	}
}

type scriptPredicateEvaluator struct {
	script     *Script
	generation *scriptGeneration
	expression ast.Expr
	captured   *scope
	span       Span
}

func (predicate *scriptPredicateEvaluator) Evaluate(ctx context.Context, arguments ...Value) (result bool, resultErr error) {
	if predicate == nil || predicate.script == nil {
		return false, ErrScriptUnloaded
	}
	ctx, release, err := predicate.script.acquireGenerationExecution(ctx, predicate.generation)
	if err != nil {
		return false, err
	}
	defer func() { resultErr = joinExecutionError(resultErr, release) }()
	if predicate.expression == nil || predicate.captured == nil {
		return false, errors.New("opfor: invalid predicate environment")
	}
	ctx = withExecutionMeter(ctx, predicate.script.runtime)
	function := &bytecode.Function{Name: "<predicate>", Span: predicate.span}
	closure := &scriptClosure{script: predicate.script, function: function, captured: predicate.captured}
	bound := make([]Argument, len(arguments))
	for index, argument := range arguments {
		bound[index] = Argument{Value: argument}
	}
	frame, err := predicate.captured.localChildAt(ctx, predicate.span)
	if err != nil {
		return false, err
	}
	fiber, err := newFiberAt(ctx, closure, frame, bound, predicate.span)
	if err != nil {
		return false, err
	}
	ctx = withCurrentFiber(ctx, fiber)
	value, err := fiber.evalPredicate(ctx, predicate.expression)
	if err != nil {
		return false, err
	}
	return value.Truth(), nil
}

func bindingHasAncestor(binding Binding, ancestor Binding) bool {
	for parent := binding.Parent; parent != nil; parent = parent.Parent {
		if parent.BindingID == ancestor.ID && parent.Script == ancestor.Script {
			return true
		}
	}
	return false
}

func isCompositionBinding(kind BindingKind) bool {
	return envspec.RecomposesDescendants(string(kind))
}

// clearBindingDescendants retires the previous ephemeral popup/menu tree just
// before the parent is composed again. The newest tree stays invokable until
// the next composition, while repeated composition does not grow permanent
// Runtime registrations.
func (r *Runtime) clearBindingDescendants(ctx context.Context, binding Binding) error {
	if r == nil || binding.ID == 0 || !isCompositionBinding(binding.Kind) {
		return nil
	}
	r.mu.RLock()
	parentOwner := r.scripts[binding.Script]
	if parentOwner == nil {
		r.mu.RUnlock()
		return ErrScriptUnloaded
	}
	var descendants []Binding
	for _, ordered := range r.bindingOrder {
		for _, candidate := range ordered {
			if bindingHasAncestor(candidate, binding) {
				descendants = append(descendants, cloneBinding(candidate))
			}
		}
	}
	r.mu.RUnlock()
	if !parentOwner.Active() {
		return ErrScriptUnloaded
	}
	// Descendants can cross script ownership through insert_menu. Remove deeper
	// nodes first, then use stable script-local identities for deterministic
	// teardown where no process-wide binding sequence exists.
	sort.SliceStable(descendants, func(left, right int) bool {
		leftDepth := bindingInvocationDepth(descendants[left].Parent)
		rightDepth := bindingInvocationDepth(descendants[right].Parent)
		if leftDepth != rightDepth {
			return leftDepth > rightDepth
		}
		if descendants[left].Script != descendants[right].Script {
			return descendants[left].Script > descendants[right].Script
		}
		return descendants[left].ID > descendants[right].ID
	})

	var result error
	for _, candidate := range descendants {
		r.mu.RLock()
		owner := r.scripts[candidate.Script]
		r.mu.RUnlock()
		if owner == nil || !owner.removeBindingIfPresent(candidate) {
			continue
		}
		if r.observer != nil {
			if err := r.observer.Unregistered(ctx, cloneBinding(candidate)); err != nil {
				result = errors.Join(result, preserveNativeBoundaryError(ctx, err))
			}
		}
	}
	return result
}

func bindingInvocationDepth(invocation *BindingInvocation) int {
	depth := 0
	for current := invocation; current != nil; current = current.Parent {
		depth++
	}
	return depth
}

func (r *Runtime) prepareBindingInvocation(ctx context.Context, binding Binding, arguments []Value) (context.Context, func() error, error) {
	if r == nil {
		return ctx, nil, errors.New("opfor: runtime is nil")
	}
	r.mu.RLock()
	script := r.scripts[binding.Script]
	r.mu.RUnlock()
	if script == nil {
		return ctx, nil, ErrScriptUnloaded
	}
	generation := generationForBinding(binding)
	if generation == nil {
		return ctx, nil, ErrScriptUnloaded
	}
	executionCtx, release, err := script.acquireGenerationExecution(ctx, generation)
	if err != nil {
		return ctx, nil, err
	}
	handedOff := false
	defer func() {
		if !handedOff {
			_ = release()
		}
	}()
	if err := r.clearBindingDescendants(executionCtx, binding); err != nil {
		return ctx, nil, joinExecutionError(err, release)
	}
	handedOff = true
	return withBindingInvocation(executionCtx, bindingInvocation(binding, arguments)), release, nil
}
