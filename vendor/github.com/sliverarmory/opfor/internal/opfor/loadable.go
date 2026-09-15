package opfor

import (
	"context"
	"errors"
	"fmt"
)

// LoadableRequest identifies one Java Loadable class requested through
// Sleep's use() function. ClassName is always resolved from a string or class
// literal. Source is the original JAR/directory spelling supplied by the
// script, while ResolvedSource is the filesystem path checked by OPFOR. Both
// source fields are empty for class-only use(^Bridge) calls.
type LoadableRequest struct {
	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span

	ClassName      string
	Source         string
	ResolvedSource string
	HasSource      bool
	ClassLiteral   bool
}

// LoadableBridge is a pure-Go counterpart to sleep.interfaces.Loadable. A
// resolved bridge instance is cached for one source/class pair in one Script.
// ScriptLoaded is called for every matching use() invocation. ScriptUnloaded is
// paired with every begun ScriptLoaded call, in reverse order, when the owning
// portable ScriptLoader generation retires or the Script terminally unloads.
// Repeated use() calls may invoke ScriptLoaded concurrently, including for the
// same bridge and Script. ScriptUnloaded callbacks run only after active calls
// in their generation finish and are delivered sequentially.
// Implementations must be concurrency-safe and should observe ctx, but must not
// retain ctx after either callback returns.
//
// ScriptLoaded may install script-scoped functions with
// Script.RegisterFunction. Those functions are automatically revoked when the
// creating generation retires or the Script unloads. During logical
// ScriptLoader cleanup the supplied *Script remains active because its raw
// Sleep state survives; the generation-owned functions and opaque callbacks
// do not. A retained raw *Script is trusted authority, not a restricted
// generation capability.
type LoadableBridge interface {
	ScriptLoaded(context.Context, *Script) error
	ScriptUnloaded(context.Context, *Script) error
}

// LoadableBridgeFuncs adapts optional functions to LoadableBridge. A nil
// function is a no-op.
type LoadableBridgeFuncs struct {
	Loaded   func(context.Context, *Script) error
	Unloaded func(context.Context, *Script) error
}

// ScriptLoaded invokes Loaded when configured.
func (functions LoadableBridgeFuncs) ScriptLoaded(ctx context.Context, script *Script) error {
	if functions.Loaded == nil {
		return nil
	}
	return functions.Loaded(ctx, script)
}

// ScriptUnloaded invokes Unloaded when configured.
func (functions LoadableBridgeFuncs) ScriptUnloaded(ctx context.Context, script *Script) error {
	if functions.Unloaded == nil {
		return nil
	}
	return functions.Unloaded(ctx, script)
}

// LoadableProvider resolves Java Loadable class identities to pure-Go bridge
// instances. ResolveLoadable is called at most once concurrently for a given
// source/class pair in one Script; failures are not cached. A successful bridge
// is script-local even when the same Provider instance serves several scripts.
// Different identities and scripts may be resolved concurrently. The call is
// synchronous; implementations should observe ctx and must not retain it after
// returning. The detached LoadableRequest may be retained.
//
// UnsupportedError explicitly declines one identity and continues to OPFOR's
// canonical fixture/Host fallback. Other provider errors become Sleep's
// checkError() soft-error state, matching use(). Context cancellation and
// runtime/script closure remain hard execution errors.
type LoadableProvider interface {
	ResolveLoadable(context.Context, LoadableRequest) (LoadableBridge, error)
}

// LoadableProviderFunc adapts a function to LoadableProvider.
type LoadableProviderFunc func(context.Context, LoadableRequest) (LoadableBridge, error)

// ResolveLoadable invokes function.
func (function LoadableProviderFunc) ResolveLoadable(
	ctx context.Context,
	request LoadableRequest,
) (LoadableBridge, error) {
	if function == nil {
		return nil, errors.New("opfor: Loadable provider is nil")
	}
	return function(ctx, request)
}

// WithLoadableProvider installs the typed importer boundary for Sleep use().
// WithFunction("use", ...) retains precedence over this native wrapper. The
// provider follows source-backed portable ScriptLoader child runtimes.
func WithLoadableProvider(provider LoadableProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Loadable provider is nil")
		}
		config.loadableProvider = provider
		return nil
	}
}

type scriptLoadableResolution struct {
	ready   chan struct{}
	request LoadableRequest
	bridge  LoadableBridge
	err     error
}

type scriptLoadableUse struct {
	bridge     LoadableBridge
	generation *scriptGeneration
}

type loadableResolutionToken struct {
	script ScriptID
	key    string
	parent *loadableResolutionToken
}

type loadableResolutionContextKey struct{}

func loadableRequestKey(request LoadableRequest) string {
	return request.ResolvedSource + "\x00" + request.ClassName
}

func resolvingLoadable(ctx context.Context, script ScriptID, key string) bool {
	if ctx == nil {
		return false
	}
	token, _ := ctx.Value(loadableResolutionContextKey{}).(*loadableResolutionToken)
	for token != nil {
		if token.script == script && token.key == key {
			return true
		}
		token = token.parent
	}
	return false
}

func withLoadableResolution(ctx context.Context, script ScriptID, key string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	parent, _ := ctx.Value(loadableResolutionContextKey{}).(*loadableResolutionToken)
	return context.WithValue(ctx, loadableResolutionContextKey{}, &loadableResolutionToken{
		script: script,
		key:    key,
		parent: parent,
	})
}

func (r *Runtime) resolveScriptLoadable(
	ctx context.Context,
	script *Script,
	request LoadableRequest,
) (*scriptLoadableResolution, error) {
	if r == nil || script == nil {
		return nil, ErrScriptUnloaded
	}
	key := loadableRequestKey(request)
	generation := scriptGenerationFromContext(ctx, script)
	if generation == nil {
		generation = script.currentScriptGeneration()
	}

	script.mu.Lock()
	if !script.generationAdmissibleLocked(generation) {
		script.mu.Unlock()
		return nil, ErrScriptUnloaded
	}
	if script.loadables == nil {
		script.loadables = make(map[string]*scriptLoadableResolution)
	}
	resolution := script.loadables[key]
	if resolution != nil {
		ready := resolution.ready
		script.mu.Unlock()
		if resolvingLoadable(ctx, script.id, key) {
			return nil, fmt.Errorf("opfor: recursive Loadable resolution for %q", request.ClassName)
		}
		select {
		case <-ready:
			return resolution, resolution.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	resolution = &scriptLoadableResolution{ready: make(chan struct{}), request: request}
	script.loadables[key] = resolution
	script.mu.Unlock()

	providerCtx := withLoadableResolution(ctx, script.id, key)
	bridge, err := r.loadableProvider.ResolveLoadable(providerCtx, request)
	if err == nil && isNilInterface(bridge) {
		err = fmt.Errorf("opfor: Loadable provider returned a nil bridge for %q", request.ClassName)
	}
	resolution.bridge = bridge
	resolution.err = err
	if err != nil {
		script.mu.Lock()
		if script.loadables[key] == resolution {
			delete(script.loadables, key)
		}
		script.mu.Unlock()
	}
	close(resolution.ready)
	return resolution, err
}

func (r *Runtime) loadProviderBridge(
	ctx context.Context,
	invocation Invocation,
	request LoadableRequest,
) (Value, error) {
	script := r.script(invocation.Script)
	if script == nil {
		return Null(), ErrScriptUnloaded
	}
	resolution, err := r.resolveScriptLoadable(ctx, script, request)
	if err != nil {
		if executionErr := executionContextError(ctx); executionErr != nil {
			return Null(), executionErr
		}
		// A provider may implement only selected class identities. An explicit
		// UnsupportedError declines this request so useLoadable can continue to
		// the canonical portable adapter and exact Host compatibility path.
		var unsupported *UnsupportedError
		if errors.As(err, &unsupported) {
			return Null(), err
		}
		return r.flagSourceError(invocation, preserveNativeBoundaryError(ctx, err))
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}

	// Publish the matching unload call before entering importer code. This
	// mirrors ScriptLifecycleObserver's begun-callback rule: even a failing or
	// reentrantly unloading ScriptLoaded call receives cleanup exactly once.
	generation := invocation.generationToken()
	script.mu.Lock()
	if !script.generationAdmissibleLocked(generation) {
		script.mu.Unlock()
		return Null(), ErrScriptUnloaded
	}
	script.loadableUses = append(script.loadableUses, scriptLoadableUse{bridge: resolution.bridge, generation: generation})
	script.mu.Unlock()

	if err := resolution.bridge.ScriptLoaded(ctx, script); err != nil {
		if executionErr := executionContextError(ctx); executionErr != nil {
			return Null(), executionErr
		}
		return r.flagSourceError(invocation, preserveNativeBoundaryError(ctx, err))
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	if !script.generationAdmissible(generation) {
		return Null(), ErrScriptUnloaded
	}
	return Null(), nil
}
