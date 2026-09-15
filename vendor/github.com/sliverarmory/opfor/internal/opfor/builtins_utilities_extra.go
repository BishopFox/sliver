package opfor

import (
	"context"
	"fmt"
	"os"
	goruntime "runtime"
)

// utilityExtraFunctions returns the portable utility functions whose behavior
// can be expressed entirely through OPFOR's public value and callable model.
// The semantic reference is Sleep 2.1's pinned BasicStrings and
// BasicUtilities bridges.
//
// Scope-stack functions recover the active fiber from the execution context;
// ordered-hash constructors and policies live with the collection functions.
func (r *Runtime) utilityExtraFunctions() map[string]NativeFunc {
	return map[string]NativeFunc{
		"cast":             builtinCast,
		"casti":            builtinCastImmediate,
		"mid":              builtinSleepMid,
		"newInstance":      r.newPortableInstance,
		"popl":             builtinPopLocal,
		"pushl":            builtinPushLocal,
		"putAll":           builtinPutAll,
		"search":           builtinSearch,
		"setField":         builtinSetField,
		"systemProperties": r.systemProperties,
		"taint":            builtinTaint,
		"untaint":          builtinUntaint,
	}
}

// systemProperties returns the portable subset of the JVM properties exposed
// by Sleep 2.1's BasicUtilities.systemProperties bridge. The reference bridge
// wraps System.getProperties(), so its keys and values are strings and a
// missing property reads as Sleep's empty scalar.
//
// OPFOR has no JVM and deliberately does not invent java.*, VM, vendor, class
// path, encoding, or arbitrary -D properties. It takes a detached, read-only
// snapshot of the small set of host facts with direct pure-Go counterparts.
// This exposes host paths and platform identity, just like the reference
// function; it is introspection rather than a security boundary. Embedders
// needing a fixed, redacted, or JVM-shaped property set can override
// systemProperties with WithFunction.
func (r *Runtime) systemProperties(_ context.Context, invocation Invocation) (Value, error) {
	properties := make(map[string]Value)
	set := func(key, value string) {
		if value != "" {
			properties[key] = String(value)
		}
	}

	// NewReadOnlyHash sorts these keys before applying the same stable Java-7
	// bucket traversal model as every other ordinary OPFOR hash.
	set("file.separator", string(os.PathSeparator))
	set("path.separator", string(os.PathListSeparator))
	if goruntime.GOOS == "windows" {
		set("line.separator", "\r\n")
	} else {
		set("line.separator", "\n")
	}
	set("os.name", goruntime.GOOS)
	set("os.arch", goruntime.GOARCH)
	set("java.io.tmpdir", os.TempDir())
	if home, err := os.UserHomeDir(); err == nil {
		set("user.home", home)
	}
	if goruntime.GOOS == "windows" {
		set("user.name", os.Getenv("USERNAME"))
	} else {
		set("user.name", os.Getenv("USER"))
	}

	// chdir is runtime-local in OPFOR. Its state is shared with the default
	// FileSourceResolver, so use that synchronized value instead of the Go
	// process cwd. A custom SourceResolver owns its own path model; omitting
	// user.dir in that case avoids reporting a silently divergent directory.
	activeRuntime := invocation.Runtime
	if activeRuntime == nil {
		activeRuntime = r
	}
	if activeRuntime != nil && activeRuntime.defaultFileResolver != nil {
		set("user.dir", activeRuntime.defaultFileResolver.BaseDirectory())
	}

	hash, err := newRuntimeReadOnlyHash(activeRuntime, properties)
	if err != nil {
		return Null(), err
	}
	return HashValue(hash), nil
}

func builtinTaint(_ context.Context, invocation Invocation) (Value, error) {
	value := invocation.Arg(0)
	if invocation.Runtime != nil {
		value = invocation.Runtime.TaintAll(value)
	}
	if len(invocation.Arguments) != 0 {
		if invocation.Arguments[0].Reference != nil {
			invocation.Arguments[0].Reference.setTaintValue(value)
		}
	}
	return value, nil
}

func builtinUntaint(_ context.Context, invocation Invocation) (Value, error) {
	value := invocation.Arg(0)
	if invocation.Runtime != nil {
		value = invocation.Runtime.Untaint(value)
	}
	if len(invocation.Arguments) != 0 {
		invocation.Arguments[0].Set(value)
	}
	return value, nil
}

func builtinPushLocal(ctx context.Context, invocation Invocation) (Value, error) {
	fiber, err := invocationFiber(ctx, invocation)
	if err != nil {
		return Null(), err
	}
	if err := fiber.pushLocalAt(ctx, sleepExtractedArguments(invocation.Arguments), invocation.Span); err != nil {
		return Null(), err
	}
	return Null(), nil
}

func builtinPopLocal(ctx context.Context, invocation Invocation) (Value, error) {
	fiber, err := invocationFiber(ctx, invocation)
	if err != nil {
		return Null(), err
	}
	popped, popErr := fiber.popLocalAt(ctx, sleepExtractedArguments(invocation.Arguments), invocation.Span)
	if popErr != nil {
		return Null(), popErr
	}
	if popped {
		return Null(), nil
	}
	return Null(), sleepBridgeIllegalArgument("&popl: no more local frames exist")
}

func invocationFiber(ctx context.Context, invocation Invocation) (*fiber, error) {
	fiber := currentFiber(ctx)
	if fiber == nil || fiber.closure == nil || fiber.closure.script == nil {
		return nil, fmt.Errorf("&%s: local scope management requires an active script", invocation.Name)
	}
	if invocation.Script != 0 && invocation.Script != fiber.closure.script.id {
		return nil, fmt.Errorf("&%s: active script does not match invocation owner", invocation.Name)
	}
	return fiber, nil
}

func builtinSleepMid(_ context.Context, invocation Invocation) (Value, error) {
	value := sleepStringCoercion(invocation.Arg(0))
	start := sleepInt32(invocation.Arg(1))
	count := int32(sleepStringLength(value)) - start
	if len(invocation.Arguments) > 2 {
		count = sleepInt32(invocation.Arg(2))
	}

	// BasicStrings performs this arithmetic using Java ints. Keeping it in
	// int32 preserves the same overflow behavior before index normalization.
	stop := start + count
	return sleepValueSubstring(invocation.Name, value, int(start), int(stop))
}

func builtinSearch(ctx context.Context, invocation Invocation) (Value, error) {
	array, err := invocationArray(invocation, 0)
	if err != nil {
		return Null(), err
	}
	callable, err := invocationCallable(invocation, 1)
	if err != nil {
		return Null(), err
	}
	length := array.Len()
	cursor, err := newSequenceCursor(ArrayValue(array), invocation.Name)
	if err != nil {
		return Null(), err
	}

	start := sleepInt32(invocation.Arg(2))
	if start < 0 {
		// BridgeUtilities.normalize adds the length once; values still below
		// zero consequently search from the beginning.
		start += int32(length)
	}
	for index := 0; ; index++ {
		value, present, err := cursor.next(ctx)
		if err != nil {
			return Null(), err
		}
		if !present {
			return Null(), nil
		}
		if start > 0 {
			start--
			continue
		}
		result, err := callable.Invoke(ctx, value, Int(int32(index)))
		if err != nil {
			return Null(), err
		}
		// Sleep's search tests for the empty scalar, not truthiness: integer
		// zero and the empty string are legitimate successful results.
		if !result.IsNull() {
			return result, nil
		}
	}
}

func builtinPutAll(ctx context.Context, invocation Invocation) (Value, error) {
	target := invocation.Arg(0)
	if hash, ok := target.Hash(); ok {
		keys, err := invocationSequenceCursor(invocation, 1)
		if err != nil {
			return Null(), err
		}
		values := keys
		if len(invocation.Arguments) > 2 {
			values, err = invocationSequenceCursor(invocation, 2)
			if err != nil {
				return Null(), err
			}
		}

		for {
			key, ok, err := keys.next(ctx)
			if err != nil {
				return Null(), err
			}
			if !ok {
				return target, nil
			}
			value, present, err := values.next(ctx)
			if err != nil {
				return Null(), err
			}
			if !present {
				value = Null()
			}
			if err := hash.SetContext(ctx, key.String(), value); err != nil {
				return Null(), err
			}
		}
	}

	if array, ok := target.Array(); ok {
		values, err := invocationSequenceCursor(invocation, 1)
		if err != nil {
			return Null(), err
		}
		for {
			value, ok, err := values.next(ctx)
			if err != nil {
				return Null(), err
			}
			if !ok {
				return target, nil
			}
			if err := array.appendValuesAtExecution(ctx, invocation, value); err != nil {
				return Null(), err
			}
		}
	}

	// BasicUtilities leaves non-collection targets untouched and does not
	// inspect the remaining arguments.
	return target, nil
}
