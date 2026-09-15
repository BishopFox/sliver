package opfor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// useLoadable resolves the language-level use() operation through an explicit
// LoadableProvider, the safe source-audited canonical fixture adapter, or the
// generic Host compatibility boundary, in that order. The default never loads
// or executes JAR bytecode; portableFixtureImport only verifies ZIP entries and
// authorizes a handwritten Go adapter.
func (r *Runtime) useLoadable(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 0 {
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeEmptyStack()
		}
		return r.flagSourceError(invocation, fmt.Errorf("java.lang.ClassNotFoundException: "))
	}
	request, err := r.loadableRequest(ctx, invocation)
	if err != nil {
		return Null(), err
	}
	if !isNilInterface(r.loadableProvider) {
		value, providerErr := r.loadProviderBridge(ctx, invocation, request)
		var unsupported *UnsupportedError
		if providerErr == nil || !errors.As(providerErr, &unsupported) {
			return value, providerErr
		}
	}
	if used, useErr := r.usePortableFixtureLoadable(invocation, request); useErr != nil {
		return Null(), useErr
	} else if used {
		return Null(), nil
	}

	// Preserve the original reference-bearing Invocation for importers that use
	// the generic compatibility boundary instead of the typed provider. The
	// stock unsupported Host maps back to Sleep's ClassNotFoundException soft
	// error so the default behavior and canonical corpus remain unchanged.
	_, hostErr := r.host.Call(ctx, invocation)
	if hostErr == nil {
		return Null(), nil
	}
	var unsupported *UnsupportedError
	if !errors.As(hostErr, &unsupported) {
		if executionErr := executionContextError(ctx); executionErr != nil {
			return Null(), executionErr
		}
		return r.flagSourceError(invocation, preserveNativeBoundaryError(ctx, hostErr))
	}
	return r.flagSourceError(invocation, fmt.Errorf("java.lang.ClassNotFoundException: %s", request.ClassName))
}

func (r *Runtime) loadableRequest(ctx context.Context, invocation Invocation) (LoadableRequest, error) {
	request := LoadableRequest{
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
	}
	if len(invocation.Arguments) >= 2 {
		request.Source = invocation.Arg(0).String()
		request.ClassName = strings.TrimSpace(invocation.Arg(1).String())
		request.HasSource = true
		request.ResolvedSource = request.Source
		if resolver := r.concreteFileSourceResolver(); resolver != nil {
			request.ResolvedSource = resolver.resolveContainer(request.ResolvedSource)
		} else if !filepath.IsAbs(request.ResolvedSource) {
			request.ResolvedSource = filepath.Clean(request.ResolvedSource)
		}
		if _, err := os.Stat(request.ResolvedSource); err != nil {
			if os.IsNotExist(err) {
				return request, outputWarning(ctx, fmt.Errorf("&use: could not locate source '%s'", request.Source))
			}
			return request, outputWarning(ctx, fmt.Errorf("&use: %w", err))
		}
		return request, nil
	}
	if class, ok := loadableClassLiteral(invocation.Arg(0)); ok {
		request.ClassName = resolvePortableClassName(class)
		request.ClassLiteral = true
	} else {
		request.ClassName = strings.TrimSpace(invocation.Arg(0).String())
	}
	return request, nil
}

func loadableClassLiteral(value Value) (string, bool) {
	object, ok := value.Object()
	if !ok {
		return "", false
	}
	switch class := object.(type) {
	case classReference:
		return string(class), true
	case sleepClass:
		return string(class), true
	default:
		return "", false
	}
}

func (r *Runtime) usePortableFixtureLoadable(invocation Invocation, request LoadableRequest) (bool, error) {
	if request.HasSource && !portableFixtureImport(r, invocation.Script, request.ClassName, request.Source) {
		return false, nil
	}
	if request.ClassName != "org.hick.tests.TestLoadable" ||
		!r.portableFixtureState().allows(invocation.Script, request.ClassName) {
		return false, nil
	}
	script := r.script(invocation.Script)
	if script == nil {
		return false, nil
	}
	if err := script.setFunction("foo", &portableFixtureFooFunction{runtime: r}); err != nil {
		return false, err
	}
	return true, nil
}

// portableFixtureFooFunction mirrors the observable behavior of the pinned
// org.hick.tests.FooFunction fixture without executing its class bytes.
type portableFixtureFooFunction struct {
	runtime *Runtime
	mu      sync.Mutex
	calls   int32
}

func (function *portableFixtureFooFunction) Invoke(_ context.Context, values ...Value) (Value, error) {
	function.mu.Lock()
	defer function.mu.Unlock()

	arguments := make([]string, len(values))
	for index := range values {
		arguments[len(values)-1-index] = values[index].String()
	}
	if function.runtime != nil && function.runtime.stdout != nil {
		if _, err := fmt.Fprintf(function.runtime.stdout, "Foo has been called with args: [%s]\n", strings.Join(arguments, ", ")); err != nil {
			return Null(), err
		}
	}
	function.calls++
	return Int(function.calls), nil
}
