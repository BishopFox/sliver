package opfor

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// The String stream sources in this file follow OpenJDK 17u commit
// 352633b5cef98ef3de7e562751222c38d76bb319. They retain the immutable String
// scalar and materialize only when a terminal or iterator operation consumes
// the stream, matching the source methods' lazy traversal boundary.

type portableJavaStringStreamKind uint8

const (
	portableJavaStringLinesStream portableJavaStringStreamKind = iota
	portableJavaStringCharsStream
	portableJavaStringCodePointsStream
)

type portableJavaStringStream struct {
	mu          sync.Mutex
	head        *portableJavaStringStreamHead
	kind        portableJavaStringStreamKind
	linked      bool
	ordered     bool
	sourceStage bool
}

type portableJavaStringStreamHead struct {
	mu       sync.Mutex
	source   Value
	consumed bool
	parallel bool
}

func newPortableJavaStringStream(source Value, kind portableJavaStringStreamKind) Value {
	return ObjectValue(&portableJavaStringStream{
		head:        &portableJavaStringStreamHead{source: sleepStringCoercion(source)},
		kind:        kind,
		ordered:     true,
		sourceStage: true,
	})
}

func (stream *portableJavaStringStream) className() string {
	if stream != nil && stream.kind != portableJavaStringLinesStream {
		return "java.util.stream.IntPipeline$Head"
	}
	return "java.util.stream.ReferencePipeline$Head"
}

func (stream *portableJavaStringStream) String() string {
	if stream == nil {
		return "null"
	}
	return stream.className()
}

func (stream *portableJavaStringStream) invoke(
	ctx context.Context,
	invocation ObjectInvocation,
) (Value, bool, error) {
	if stream == nil {
		return Null(), false, nil
	}
	if invocation.Op == ObjectTypeCheck {
		return Bool(portableJavaAssignable(stream.className(), invocation.Class)), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "isParallel":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, stream.className()), true, nil
		}
		stream.head.mu.Lock()
		parallel := stream.head.parallel
		stream.head.mu.Unlock()
		return portableJavaBooleanValue(parallel), true, nil
	case "sequential", "parallel":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, stream.className()), true, nil
		}
		stream.head.mu.Lock()
		stream.head.parallel = invocation.Message == "parallel"
		stream.head.mu.Unlock()
		// AbstractPipeline returns this even after the stream was consumed.
		return invocation.Target, true, nil
	case "close":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, stream.className()), true, nil
		}
		stream.mu.Lock()
		stream.linked = true
		sourceStage := stream.sourceStage
		stream.mu.Unlock()
		if sourceStage {
			stream.head.mu.Lock()
			stream.head.consumed = true
			stream.head.source = Null()
			stream.head.mu.Unlock()
		}
		return Null(), true, nil
	case "unordered":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, stream.className()), true, nil
		}
		return stream.unordered(invocation)
	case "count":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, stream.className()), true, nil
		}
		values, err := stream.consume(ctx, nil)
		if err != nil {
			return Null(), true, err
		}
		return Long(int64(len(values))), true, nil
	case "toArray":
		if len(invocation.Arguments) != 0 {
			// Stream.toArray(IntFunction) requires a JVM functional object whose
			// array construction contract is intentionally not emulated here.
			return portableNoMatchingMethod(invocation, stream.className()), true, nil
		}
		values, err := stream.consume(ctx, invocation.Runtime)
		if err != nil {
			return Null(), true, err
		}
		return ArrayValue(NewArray(values...)), true, nil
	case "iterator":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, stream.className()), true, nil
		}
		values, err := stream.consume(ctx, invocation.Runtime)
		if err != nil {
			return Null(), true, err
		}
		return ObjectValue(&portableJavaStringStreamIterator{
			values:    values,
			intStream: stream.kind != portableJavaStringLinesStream,
		}), true, nil
	case "sum":
		if len(invocation.Arguments) != 0 || stream.kind == portableJavaStringLinesStream {
			return portableNoMatchingMethod(invocation, stream.className()), true, nil
		}
		values, err := stream.consume(ctx, nil)
		if err != nil {
			return Null(), true, err
		}
		var sum int32
		for _, value := range values {
			sum += sleepInt32(value)
		}
		return Int(sum), true, nil
	}
	return Null(), false, nil
}

func (stream *portableJavaStringStream) unordered(invocation ObjectInvocation) (Value, bool, error) {
	stream.mu.Lock()
	defer stream.mu.Unlock()
	if stream.linked {
		return Null(), true, errors.New("java.lang.IllegalStateException: stream has already been operated upon or closed")
	}
	if !stream.ordered {
		return invocation.Target, true, nil
	}
	stream.linked = true
	return ObjectValue(&portableJavaStringStream{
		head:    stream.head,
		kind:    stream.kind,
		ordered: false,
	}), true, nil
}

func (stream *portableJavaStringStream) consume(ctx context.Context, runtime *Runtime) ([]Value, error) {
	stream.mu.Lock()
	defer stream.mu.Unlock()
	if stream.linked {
		return nil, errors.New("java.lang.IllegalStateException: stream has already been operated upon or closed")
	}
	// AbstractPipeline marks a terminal operation consumed before evaluating
	// it. Cancellation and instruction-limit failures therefore consume this
	// portable stream too.
	stream.linked = true
	stream.head.mu.Lock()
	if stream.head.consumed {
		stream.head.mu.Unlock()
		return nil, errors.New("java.lang.IllegalStateException: source already consumed or closed")
	}
	stream.head.consumed = true
	source := stream.head.source
	stream.head.source = Null()
	stream.head.mu.Unlock()
	values, err := portableJavaStringStreamValues(ctx, runtime, source, stream.kind)
	return values, err
}

func portableJavaStringStreamValues(
	ctx context.Context,
	runtime *Runtime,
	source Value,
	kind portableJavaStringStreamKind,
) ([]Value, error) {
	work := &portableJavaStringWork{ctx: ctx}
	units := sleepStringUnits(source)
	switch kind {
	case portableJavaStringLinesStream:
		var values []Value
		for cursor := 0; cursor < len(units); {
			start := cursor
			for cursor < len(units) && units[cursor] != '\n' && units[cursor] != '\r' {
				if err := work.advance(1); err != nil {
					return nil, err
				}
				cursor++
			}
			if err := reserveCollectionEntries(runtime, 1); err != nil {
				return nil, err
			}
			values = append(values, sleepStringValueSlice(source, start, cursor))
			if cursor == len(units) {
				break
			}
			separator := units[cursor]
			if err := work.advance(1); err != nil {
				return nil, err
			}
			cursor++
			if separator == '\r' && cursor < len(units) && units[cursor] == '\n' {
				if err := work.advance(1); err != nil {
					return nil, err
				}
				cursor++
			}
		}
		if err := work.finish(); err != nil {
			return nil, err
		}
		return values, nil
	case portableJavaStringCharsStream:
		var values []Value
		for _, unit := range units {
			if err := work.advance(1); err != nil {
				return nil, err
			}
			if err := reserveCollectionEntries(runtime, 1); err != nil {
				return nil, err
			}
			values = append(values, Int(int32(unit)))
		}
		if err := work.finish(); err != nil {
			return nil, err
		}
		return values, nil
	case portableJavaStringCodePointsStream:
		var values []Value
		for index := 0; index < len(units); {
			codePoint, width := sleepUTF16CodePointAt(units, index)
			if err := work.advance(width); err != nil {
				return nil, err
			}
			if err := reserveCollectionEntries(runtime, 1); err != nil {
				return nil, err
			}
			values = append(values, Int(codePoint))
			index += width
		}
		if err := work.finish(); err != nil {
			return nil, err
		}
		return values, nil
	default:
		return nil, fmt.Errorf("opfor: invalid portable String stream kind %d", kind)
	}
}

type portableJavaStringStreamIterator struct {
	mu        sync.Mutex
	values    []Value
	position  int
	intStream bool
}

func (iterator *portableJavaStringStreamIterator) className() string {
	if iterator != nil && iterator.intStream {
		return "java.util.Spliterators$2Adapter"
	}
	return "java.util.Spliterators$1Adapter"
}

func (iterator *portableJavaStringStreamIterator) String() string {
	if iterator == nil {
		return "null"
	}
	return iterator.className()
}

func (iterator *portableJavaStringStreamIterator) invoke(
	ctx context.Context,
	invocation ObjectInvocation,
) (Value, bool, error) {
	if iterator == nil {
		return Null(), false, nil
	}
	if invocation.Op == ObjectTypeCheck {
		return Bool(portableJavaAssignable(iterator.className(), invocation.Class)), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	if len(invocation.Arguments) != 0 {
		switch invocation.Message {
		case "hasNext", "next", "nextInt", "remove":
			return portableNoMatchingMethod(invocation, iterator.className()), true, nil
		default:
			return Null(), false, nil
		}
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), true, err
	}
	switch invocation.Message {
	case "hasNext":
		iterator.mu.Lock()
		result := iterator.position < len(iterator.values)
		iterator.mu.Unlock()
		return portableJavaBooleanValue(result), true, nil
	case "next", "nextInt":
		if invocation.Message == "nextInt" && !iterator.intStream {
			return portableNoMatchingMethod(invocation, iterator.className()), true, nil
		}
		iterator.mu.Lock()
		defer iterator.mu.Unlock()
		if iterator.position >= len(iterator.values) {
			return Null(), true, errors.New("java.util.NoSuchElementException")
		}
		value := iterator.values[iterator.position]
		iterator.position++
		return value, true, nil
	case "remove":
		return Null(), true, errors.New("java.lang.UnsupportedOperationException: remove")
	}
	return Null(), false, nil
}

func portableJavaStringTransform(
	ctx context.Context,
	invocation ObjectInvocation,
	target Value,
) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	function := invocation.Arg(0)
	if function.IsNull() {
		return Null(), true, errors.New(`java.lang.NullPointerException: Cannot invoke "java.util.function.Function.apply(Object)" because "f" is null`)
	}
	if object, ok := function.Object(); ok {
		proxy, ok := object.(*portableJavaProxy)
		if !ok || proxy == nil || !proxy.implements("java.util.function.Function") {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		value, err := proxy.call(ctx, "apply", []Argument{{Value: target}}, true)
		return value, true, err
	}
	if _, ok := function.Function(); !ok {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	// Sleep's ObjectUtilities automatically converts a function scalar to a
	// java.util.function.Function proxy. Reuse the same exact $0/$1 call shape
	// as an explicit newInstance(^Function, ...) proxy without retaining it.
	proxy := &portableJavaProxy{closure: function, runtime: invocation.Runtime}
	value, err := proxy.call(ctx, "apply", []Argument{{Value: target}}, true)
	return value, true, err
}

func (proxy *portableJavaProxy) implements(class string) bool {
	if proxy == nil {
		return false
	}
	class = resolvePortableClassName(class)
	for _, implemented := range proxy.interfaces {
		if resolvePortableClassName(implemented) == class {
			return true
		}
	}
	return false
}
