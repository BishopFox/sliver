package opfor

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const debugTraceTaint int32 = 128

// TaintPolicy classifies a bridge using Sleep 2.1's taint contract. Functions
// are permeable unless explicitly classified: tainted input taints a scalar
// result. Sources always taint their result, sanitizers remove the top-level
// scalar marker, and sensitive functions reject tainted input.
type TaintPolicy uint8

const (
	// TaintPermeable applies Sleep's default input-to-result propagation.
	TaintPermeable TaintPolicy = iota
	// TaintSource recursively marks every successful result.
	TaintSource
	// TaintSanitizer removes the successful result's top scalar marker.
	TaintSanitizer
	// TaintSensitive rejects calls containing any tainted argument.
	TaintSensitive
	// TaintSensitiveSource combines sensitive input checking with a tainted
	// successful result.
	// TaintSensitiveSource is used for effects such as backtick execution:
	// tainted commands are rejected and successful output is tainted.
	TaintSensitiveSource
)

// WithTaintMode enables or disables Sleep 2.1 compatible taint tracking for
// this Runtime. It is disabled by default and is isolated per Runtime.
func WithTaintMode(enabled bool) Option {
	return func(config *runtimeConfig) error {
		config.taintMode = enabled
		return nil
	}
}

// TaintMode reports whether this Runtime tracks and enforces Sleep taint.
func (r *Runtime) TaintMode() bool { return r != nil && r.taintMode }

// WithTaintPolicy classifies a native or Host-resolved function. The name may
// include Sleep's leading ampersand. It may be combined with WithFunction or
// used for a function supplied by Host.
func WithTaintPolicy(name string, policy TaintPolicy) Option {
	return func(config *runtimeConfig) error {
		normalized, err := normalizeFunctionName(name)
		if err != nil {
			return err
		}
		if err := validateTaintPolicy(policy); err != nil {
			return err
		}
		if config.taintPolicies == nil {
			config.taintPolicies = make(map[string]TaintPolicy)
		}
		config.taintPolicies[normalized] = policy
		return nil
	}
}

// WithTaintFunction installs a native function and its taint classification.
func WithTaintFunction(name string, function NativeFunc, policy TaintPolicy) Option {
	return func(config *runtimeConfig) error {
		if err := WithFunction(name, function)(config); err != nil {
			return err
		}
		return WithTaintPolicy(name, policy)(config)
	}
}

// RegisterTaintPolicy changes the classification of a native or Host-resolved
// function. Existing calls in progress keep the policy they started with.
func (r *Runtime) RegisterTaintPolicy(name string, policy TaintPolicy) error {
	if r == nil {
		return errors.New("opfor: runtime is nil")
	}
	normalized, err := normalizeFunctionName(name)
	if err != nil {
		return err
	}
	if err := validateTaintPolicy(policy); err != nil {
		return err
	}
	r.mu.Lock()
	if r.taintPolicies == nil {
		r.taintPolicies = make(map[string]TaintPolicy)
	}
	r.taintPolicies[normalized] = policy
	r.mu.Unlock()
	return nil
}

// RegisterTaintFunction installs a native function and its taint
// classification atomically with respect to subsequent calls.
func (r *Runtime) RegisterTaintFunction(name string, function NativeFunc, policy TaintPolicy) error {
	if r == nil {
		return errors.New("opfor: runtime is nil")
	}
	normalized, err := normalizeFunctionName(name)
	if err != nil {
		return err
	}
	if function == nil {
		return fmt.Errorf("opfor: function %q is nil", name)
	}
	if err := validateTaintPolicy(policy); err != nil {
		return err
	}
	r.mu.Lock()
	r.functions[normalized] = function
	if r.taintPolicies == nil {
		r.taintPolicies = make(map[string]TaintPolicy)
	}
	r.taintPolicies[normalized] = policy
	r.mu.Unlock()
	return nil
}

func validateTaintPolicy(policy TaintPolicy) error {
	if policy > TaintSensitiveSource {
		return fmt.Errorf("opfor: invalid taint policy %d", policy)
	}
	return nil
}

// Taint marks a scalar value when taint mode is enabled. Arrays and hashes are
// intentionally not traversed; use TaintAll for a bridge source that returns a
// container. The returned Value retains the same kind, data, and identity.
func (r *Runtime) Taint(value Value) Value {
	if r == nil || !r.taintMode || value.IsNull() || value.Kind() == KindArray || value.Kind() == KindHash {
		return value
	}
	value.tainted = true
	return value
}

// TaintAll marks a scalar result or recursively marks the elements of an array
// or hash. Unlike Sleep 2.1's original helper, traversal is cycle-safe.
func (r *Runtime) TaintAll(value Value) Value {
	if r == nil || !r.taintMode {
		return value
	}
	return taintAllValue(value, make(map[any]struct{}))
}

// Untaint removes only the top-level scalar marker, matching
// TaintUtils.untaint. It does not recursively sanitize array/hash elements.
func (r *Runtime) Untaint(value Value) Value {
	if r == nil || !r.taintMode {
		return value
	}
	value.tainted = false
	return value
}

func taintAllValue(value Value, seen map[any]struct{}) Value {
	switch value.Kind() {
	case KindNull:
		return value
	case KindArray:
		array, _ := value.Array()
		if array == nil {
			return value
		}
		if _, exists := seen[array]; exists {
			return value
		}
		seen[array] = struct{}{}
		if array.backend != nil {
			array.backend.taintAll()
			return value
		}
		cells, err := array.snapshotCells()
		if err != nil {
			cells = array.cachedCells()
		}
		for _, cell := range cells {
			cell.setTaintValue(taintAllValue(cell.Get(), seen))
		}
		return value
	case KindHash:
		hash, _ := value.Hash()
		if hash == nil {
			return value
		}
		if _, exists := seen[hash]; exists {
			return value
		}
		seen[hash] = struct{}{}
		if hash.backend != nil {
			hash.backend.taintAll()
			return value
		}
		for _, key := range hash.KeyValues() {
			if cell, ok := hash.CellValue(key); ok {
				cell.setTaintValue(taintAllValue(cell.Get(), seen))
			}
		}
		return value
	case KindObject:
		object, _ := value.Object()
		switch pair := object.(type) {
		case sleepKeyValue:
			pair.value = taintAllValue(pair.value, seen)
			value.data = pair
			return value
		case *sleepKeyValue:
			if pair != nil {
				copy := *pair
				copy.value = taintAllValue(copy.value, seen)
				value.data = &copy
				return value
			}
		}
	}
	value.tainted = true
	return value
}

func valueIsTainted(value Value) bool {
	return valueIsTaintedSeen(value, make(map[any]struct{}))
}

func valueIsTaintedSeen(value Value, seen map[any]struct{}) bool {
	switch value.Kind() {
	case KindNull:
		return false
	case KindArray:
		array, _ := value.Array()
		if array == nil {
			return false
		}
		if _, exists := seen[array]; exists {
			return false
		}
		seen[array] = struct{}{}
		for _, item := range array.Values() {
			if valueIsTaintedSeen(item, seen) {
				return true
			}
		}
		return false
	case KindHash:
		hash, _ := value.Hash()
		if hash == nil {
			return false
		}
		if _, exists := seen[hash]; exists {
			return false
		}
		seen[hash] = struct{}{}
		for _, key := range hash.KeyValues() {
			if cell, ok := hash.CellValue(key); ok && valueIsTaintedSeen(cell.Get(), seen) {
				return true
			}
		}
		return false
	default:
		return value.tainted
	}
}

func taintedArgumentValues(arguments []Argument) []Value {
	values := make([]Value, 0, len(arguments))
	for _, argument := range arguments {
		value := argument.Resolve()
		if value.IsTainted() {
			values = append(values, value)
		}
	}
	return values
}

// taintedArguments returns the tainted values reachable from arguments only
// when this runtime has taint tracking enabled. Keep the mode check outside
// taintedArgumentValues so the default runtime never recursively walks array
// or hash arguments merely to discover that propagation is disabled.
func (r *Runtime) taintedArguments(arguments []Argument) []Value {
	if r == nil || !r.taintMode {
		return nil
	}
	return taintedArgumentValues(arguments)
}

// taintedValues returns the tainted values reachable from values only when
// this runtime has taint tracking enabled.
func (r *Runtime) taintedValues(values ...Value) []Value {
	if r == nil || !r.taintMode {
		return nil
	}
	return taintedValues(values...)
}

func describeTaintedValues(values []Value) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = value.Describe()
	}
	return strings.Join(parts, ", ")
}

func (r *Runtime) taintPolicy(name string) TaintPolicy {
	if r == nil {
		return TaintPermeable
	}
	r.mu.RLock()
	policy := r.taintPolicies[name]
	r.mu.RUnlock()
	return policy
}

func (r *Runtime) rejectTaintedCall(ctx context.Context, name string, arguments []Argument) error {
	if r == nil || !r.taintMode {
		return nil
	}
	values := taintedArgumentValues(arguments)
	if len(values) == 0 {
		return nil
	}
	display := name
	if !strings.HasPrefix(display, "&") && display != "__EXEC__" {
		display = "&" + display
	}
	err := fmt.Errorf("Insecure %s: %s is tainted", display, describeTaintedValues(values))
	if currentFiber(ctx) != nil {
		return &uncaughtScriptWarning{err: err}
	}
	return err
}

func (r *Runtime) permeateResult(ctx context.Context, result Value, arguments []Argument, span Span) Value {
	if r == nil || !r.taintMode {
		return result
	}
	return r.permeateResultFrom(ctx, result, taintedArgumentValues(arguments), span)
}

func (r *Runtime) permeateResultFrom(ctx context.Context, result Value, values []Value, span Span) Value {
	if r == nil || !r.taintMode || result.IsNull() || result.Kind() == KindArray || result.Kind() == KindHash {
		return result
	}
	if len(values) == 0 {
		return result
	}
	return r.permeateResultWithDescription(ctx, result, describeTaintedValues(values), span)
}

func (r *Runtime) permeateResultWithDescription(ctx context.Context, result Value, description string, span Span) Value {
	if r == nil || !r.taintMode || result.IsNull() || result.Kind() == KindArray || result.Kind() == KindHash || description == "" {
		return result
	}
	result = r.Taint(result)
	r.traceTaint(ctx, "tainted value: "+result.Describe()+" from: "+description, span)
	return result
}

func (r *Runtime) applyTaintPolicy(ctx context.Context, name string, policy TaintPolicy, arguments []Argument, span Span, call func() (Value, error)) (Value, error) {
	if r == nil || !r.taintMode {
		return call()
	}
	if policy == TaintSensitive || policy == TaintSensitiveSource {
		if err := r.rejectTaintedCall(ctx, name, arguments); err != nil {
			return Null(), err
		}
	}
	taintedInputs := taintedArgumentValues(arguments)
	taintedDescription := describeTaintedValues(taintedInputs)
	result, err := call()
	if err != nil {
		return result, err
	}
	switch policy {
	case TaintSource, TaintSensitiveSource:
		return r.TaintAll(result), nil
	case TaintSanitizer:
		return r.Untaint(result), nil
	default:
		return r.permeateResultWithDescription(ctx, result, taintedDescription, span), nil
	}
}

func (r *Runtime) traceTaint(ctx context.Context, message string, span Span) {
	if r == nil || !r.taintMode {
		return
	}
	fiber := currentFiber(ctx)
	if fiber == nil || fiber.closure == nil || fiber.closure.script == nil {
		return
	}
	script := fiber.closure.script
	script.mu.RLock()
	enabled := script.debug&debugTraceTaint == debugTraceTaint
	script.mu.RUnlock()
	if enabled {
		r.writeWarning(message, span)
	}
}

func (r *Runtime) installCoreTaintPolicies() {
	if r == nil {
		return
	}
	sources := []string{"readln", "readAll", "readc", "readb", "bread", "readObject", "readAsObject"}
	sensitive := []string{"openf", "connect", "exec", "use", "include", "compile_closure", "function", "eval", "expr"}
	sanitizers := []string{
		"abs", "acos", "asin", "atan", "atan2", "ceil", "cos", "log", "round", "sin", "sqrt", "tan",
		"radians", "degrees", "exp", "floor", "sum", "double", "int", "uint", "long", "parseNumber", "formatNumber",
		"not", "untaint",
	}
	for _, name := range sources {
		if _, explicit := r.taintPolicies[name]; !explicit {
			r.taintPolicies[name] = TaintSource
		}
	}
	for _, name := range sensitive {
		if _, explicit := r.taintPolicies[name]; !explicit {
			r.taintPolicies[name] = TaintSensitive
		}
	}
	for _, name := range sanitizers {
		if _, explicit := r.taintPolicies[name]; !explicit {
			r.taintPolicies[name] = TaintSanitizer
		}
	}
	if _, explicit := r.taintPolicies["taint"]; !explicit {
		r.taintPolicies["taint"] = TaintSource
	}
	if _, explicit := r.taintPolicies["__EXEC__"]; !explicit {
		r.taintPolicies["__EXEC__"] = TaintSensitiveSource
	}
}
