package opfor

import (
	"context"
	"errors"
	"math"
	"sync"
)

// The primitive streams in this file follow java.util.Random's overrides in
// OpenJDK at commit 42e3c842ae2684265c794868fc76eb0ff2dea3d9. Random uses
// RandomSupport.AbstractSpliteratorGenerator rather than the interface's
// generate/limit defaults, so the source is lazy, SIZED, and shares the exact
// Random instance whose state it advances.

const portableJavaRandomStreamNativeLoopChunk = 32 * 1024

var errPortableJavaRandomStreamMaterializationLimit = errors.New("java.lang.OutOfMemoryError: Required length exceeds implementation limit")

type portableJavaRandomStreamKind uint8

const (
	portableJavaRandomIntStream portableJavaRandomStreamKind = iota
	portableJavaRandomLongStream
	portableJavaRandomDoubleStream
)

type portableJavaRandomStream struct {
	mu      sync.Mutex
	head    *portableJavaRandomStreamHead
	kind    portableJavaRandomStreamKind
	size    int64
	bounded bool

	intOrigin    int32
	intBound     int32
	longOrigin   int64
	longBound    int64
	doubleOrigin float64
	doubleBound  float64

	linked bool
}

type portableJavaRandomStreamHead struct {
	mu       sync.Mutex
	random   *portableJavaRandom
	consumed bool
	parallel bool
}

func (stream *portableJavaRandomStream) className() string {
	if stream == nil {
		return "java.util.stream.IntPipeline$Head"
	}
	switch stream.kind {
	case portableJavaRandomLongStream:
		return "java.util.stream.LongPipeline$Head"
	case portableJavaRandomDoubleStream:
		return "java.util.stream.DoublePipeline$Head"
	default:
		return "java.util.stream.IntPipeline$Head"
	}
}

func (stream *portableJavaRandomStream) String() string {
	if stream == nil {
		return "null"
	}
	return stream.className()
}

func portableJavaRandomStreamForInvocation(
	random *portableJavaRandom,
	invocation ObjectInvocation,
) (Value, bool, error) {
	var kind portableJavaRandomStreamKind
	switch invocation.Message {
	case "ints":
		kind = portableJavaRandomIntStream
	case "longs":
		kind = portableJavaRandomLongStream
	case "doubles":
		kind = portableJavaRandomDoubleStream
	default:
		return Null(), false, nil
	}

	stream := &portableJavaRandomStream{
		head: &portableJavaRandomStreamHead{random: random},
		kind: kind,
		size: math.MaxInt64,
	}
	switch len(invocation.Arguments) {
	case 0:
	case 1:
		size, ok := portableJavaRandomLongArgument(invocation.Arg(0))
		if !ok {
			return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
		}
		stream.size = size
		if stream.size < 0 {
			return Null(), true, portableJavaRandomArgumentException(
				"size must be non-negative", portableJavaRandomStreamFrame(kind, 1),
			)
		}
	case 2:
		stream.bounded = true
		matched, err := stream.setBounds(invocation.Arg(0), invocation.Arg(1), 2)
		if !matched {
			return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
		}
		if err != nil {
			return Null(), true, err
		}
	case 3:
		size, ok := portableJavaRandomLongArgument(invocation.Arg(0))
		if !ok {
			return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
		}
		stream.size = size
		if stream.size < 0 {
			return Null(), true, portableJavaRandomArgumentException(
				"size must be non-negative", portableJavaRandomStreamFrame(kind, 3),
			)
		}
		stream.bounded = true
		matched, err := stream.setBounds(invocation.Arg(1), invocation.Arg(2), 3)
		if !matched {
			return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
		}
		if err != nil {
			return Null(), true, err
		}
	default:
		return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
	}
	return ObjectValue(stream), true, nil
}

func (stream *portableJavaRandomStream) setBounds(originValue, boundValue Value, arguments int) (bool, error) {
	switch stream.kind {
	case portableJavaRandomIntStream:
		origin, originOK := portableJavaRandomIntArgument(originValue)
		bound, boundOK := portableJavaRandomIntArgument(boundValue)
		if !originOK || !boundOK {
			return false, nil
		}
		stream.intOrigin = origin
		stream.intBound = bound
		if stream.intOrigin >= stream.intBound {
			return true, portableJavaRandomArgumentException(
				"bound must be greater than origin", portableJavaRandomStreamFrame(stream.kind, arguments),
			)
		}
	case portableJavaRandomLongStream:
		origin, originOK := portableJavaRandomLongArgument(originValue)
		bound, boundOK := portableJavaRandomLongArgument(boundValue)
		if !originOK || !boundOK {
			return false, nil
		}
		stream.longOrigin = origin
		stream.longBound = bound
		if stream.longOrigin >= stream.longBound {
			return true, portableJavaRandomArgumentException(
				"bound must be greater than origin", portableJavaRandomStreamFrame(stream.kind, arguments),
			)
		}
	case portableJavaRandomDoubleStream:
		origin, originOK := portableJavaRandomDoubleArgument(originValue)
		bound, boundOK := portableJavaRandomDoubleArgument(boundValue)
		if !originOK || !boundOK {
			return false, nil
		}
		stream.doubleOrigin = origin
		stream.doubleBound = bound
		if !(math.Inf(-1) < stream.doubleOrigin && stream.doubleOrigin < stream.doubleBound && stream.doubleBound < math.Inf(1)) {
			return true, portableJavaRandomArgumentException(
				"bound must be greater than origin", portableJavaRandomStreamFrame(stream.kind, arguments),
			)
		}
	}
	return true, nil
}

func portableJavaRandomIntArgument(value Value) (int32, bool) {
	if value.IsNull() {
		return 0, true
	}
	switch value.Kind() {
	case KindInt, KindLong, KindDouble:
		return sleepInt32(value), true
	case KindObject:
		primitive, ok := value.data.(*portableJavaPrimitive)
		if ok && primitive != nil && primitive.className() == "java.lang.Integer" {
			return sleepInt32(primitive.sleepValue()), true
		}
	}
	return 0, false
}

func portableJavaRandomLongArgument(value Value) (int64, bool) {
	if value.IsNull() {
		return 0, true
	}
	switch value.Kind() {
	case KindInt, KindLong, KindDouble:
		return sleepInt64(value), true
	case KindObject:
		primitive, ok := value.data.(*portableJavaPrimitive)
		if ok && primitive != nil && primitive.className() == "java.lang.Long" {
			return sleepInt64(primitive.sleepValue()), true
		}
	}
	return 0, false
}

func portableJavaRandomDoubleArgument(value Value) (float64, bool) {
	if value.IsNull() {
		return 0, true
	}
	switch value.Kind() {
	case KindInt, KindLong, KindDouble:
		return sleepFloat64(value), true
	case KindObject:
		primitive, ok := value.data.(*portableJavaPrimitive)
		if ok && primitive != nil && primitive.className() == "java.lang.Double" {
			return sleepFloat64(primitive.sleepValue()), true
		}
	}
	return 0, false
}

func portableJavaRandomStreamFrame(kind portableJavaRandomStreamKind, arguments int) string {
	switch kind {
	case portableJavaRandomLongStream:
		switch arguments {
		case 1:
			return "public java.util.stream.LongStream java.util.Random.longs(long)"
		case 2:
			return "public java.util.stream.LongStream java.util.Random.longs(long,long)"
		default:
			return "public java.util.stream.LongStream java.util.Random.longs(long,long,long)"
		}
	case portableJavaRandomDoubleStream:
		switch arguments {
		case 1:
			return "public java.util.stream.DoubleStream java.util.Random.doubles(long)"
		case 2:
			return "public java.util.stream.DoubleStream java.util.Random.doubles(double,double)"
		default:
			return "public java.util.stream.DoubleStream java.util.Random.doubles(long,double,double)"
		}
	default:
		switch arguments {
		case 1:
			return "public java.util.stream.IntStream java.util.Random.ints(long)"
		case 2:
			return "public java.util.stream.IntStream java.util.Random.ints(int,int)"
		default:
			return "public java.util.stream.IntStream java.util.Random.ints(long,int,int)"
		}
	}
}

func (stream *portableJavaRandomStream) invoke(
	ctx context.Context,
	invocation ObjectInvocation,
) (Value, bool, error) {
	if stream == nil || stream.head == nil {
		return Null(), false, nil
	}
	if invocation.Op == ObjectTypeCheck {
		return Bool(portableJavaAssignable(stream.className(), invocation.Class)), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "isParallel", "sequential", "parallel", "close", "unordered", "count", "toArray", "iterator", "sum":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, stream.className()), true, nil
		}
	default:
		return Null(), false, nil
	}

	switch invocation.Message {
	case "isParallel":
		stream.head.mu.Lock()
		parallel := stream.head.parallel
		stream.head.mu.Unlock()
		return portableJavaBooleanValue(parallel), true, nil
	case "sequential", "parallel":
		stream.head.mu.Lock()
		stream.head.parallel = invocation.Message == "parallel"
		stream.head.mu.Unlock()
		return invocation.Target, true, nil
	case "close":
		stream.close()
		return Null(), true, nil
	case "unordered":
		// RandomSupport.RandomSpliterator deliberately omits ORDERED. The JDK
		// pipeline therefore returns this exact stage without linking it.
		return invocation.Target, true, nil
	case "count":
		if err := stream.beginTerminal(); err != nil {
			return Null(), true, err
		}
		if err := executionContextError(ctx); err != nil {
			return Null(), true, err
		}
		// The source spliterator is SIZED, so ReduceOps returns its exact size
		// without drawing a random value.
		return Long(stream.size), true, nil
	case "toArray":
		if err := stream.beginTerminal(); err != nil {
			return Null(), true, err
		}
		if stream.size > int64(^uint(0)>>1) {
			return Null(), true, errPortableCollectionsMaterializationLimit
		}
		if err := reserveCollectionEntries(invocation.Runtime, int(stream.size)); err != nil {
			return Null(), true, err
		}
		values, err := stream.materialize(ctx)
		if err != nil {
			return Null(), true, err
		}
		return ArrayValue(NewArray(values...)), true, nil
	case "iterator":
		if err := stream.beginTerminal(); err != nil {
			return Null(), true, err
		}
		return ObjectValue(&portableJavaRandomStreamIterator{
			stream:    stream,
			remaining: stream.size,
		}), true, nil
	case "sum":
		if err := stream.beginTerminal(); err != nil {
			return Null(), true, err
		}
		return stream.sum(ctx)
	default:
		return Null(), false, nil
	}
}

func (stream *portableJavaRandomStream) close() {
	stream.mu.Lock()
	stream.linked = true
	stream.mu.Unlock()
	stream.head.mu.Lock()
	stream.head.consumed = true
	stream.head.mu.Unlock()
}

func (stream *portableJavaRandomStream) beginTerminal() error {
	stream.mu.Lock()
	if stream.linked {
		stream.mu.Unlock()
		return errors.New("java.lang.IllegalStateException: stream has already been operated upon or closed")
	}
	stream.linked = true
	stream.mu.Unlock()

	stream.head.mu.Lock()
	defer stream.head.mu.Unlock()
	if stream.head.consumed {
		return errors.New("java.lang.IllegalStateException: source already consumed or closed")
	}
	stream.head.consumed = true
	return nil
}

func (stream *portableJavaRandomStream) materialize(ctx context.Context) ([]Value, error) {
	if stream.size > portableCollectionsMaximumMaterializedElements {
		return nil, errPortableJavaRandomStreamMaterializationLimit
	}
	values := make([]Value, int(stream.size))
	work := &portableJavaRandomStreamWork{ctx: ctx}
	for index := range values {
		if err := work.advance(); err != nil {
			return nil, err
		}
		values[index] = stream.nextValue()
	}
	if err := work.finish(); err != nil {
		return nil, err
	}
	return values, nil
}

func (stream *portableJavaRandomStream) sum(ctx context.Context) (Value, bool, error) {
	work := &portableJavaRandomStreamWork{ctx: ctx}
	switch stream.kind {
	case portableJavaRandomIntStream:
		var sum int32
		for remaining := stream.size; remaining > 0; remaining-- {
			if err := work.advance(); err != nil {
				return Null(), true, err
			}
			sum += stream.nextValue().Int32()
		}
		if err := work.finish(); err != nil {
			return Null(), true, err
		}
		return Int(sum), true, nil
	case portableJavaRandomLongStream:
		var sum int64
		for remaining := stream.size; remaining > 0; remaining-- {
			if err := work.advance(); err != nil {
				return Null(), true, err
			}
			sum += stream.nextValue().Int64()
		}
		if err := work.finish(); err != nil {
			return Null(), true, err
		}
		return Long(sum), true, nil
	case portableJavaRandomDoubleStream:
		// DoublePipeline.sum uses compensated summation and separately tracks
		// the simple sum for same-signed infinities.
		var high, low, simple float64
		for remaining := stream.size; remaining > 0; remaining-- {
			if err := work.advance(); err != nil {
				return Null(), true, err
			}
			value := stream.nextValue().Float64()
			tmp := float64(value - low)
			combined := float64(high + tmp)
			low = float64(float64(combined-high) - tmp)
			high = combined
			simple = float64(simple + value)
		}
		if err := work.finish(); err != nil {
			return Null(), true, err
		}
		result := float64(high - low)
		if math.IsNaN(result) && math.IsInf(simple, 0) {
			result = simple
		}
		return Double(result), true, nil
	default:
		return Null(), false, nil
	}
}

func (stream *portableJavaRandomStream) nextValue() Value {
	random := stream.head.random
	random.mu.Lock()
	defer random.mu.Unlock()
	switch stream.kind {
	case portableJavaRandomLongStream:
		if stream.bounded {
			return Long(random.boundedNextLong(stream.longOrigin, stream.longBound))
		}
		return Long(random.nextLong())
	case portableJavaRandomDoubleStream:
		if stream.bounded {
			return Double(random.boundedNextDouble(stream.doubleOrigin, stream.doubleBound))
		}
		return Double(random.random.nextDouble())
	default:
		if stream.bounded {
			return Int(random.boundedNextInt(stream.intOrigin, stream.intBound))
		}
		return Int(random.nextInt())
	}
}

type portableJavaRandomStreamWork struct {
	ctx        context.Context
	iterations uint64
}

func (work *portableJavaRandomStreamWork) advance() error {
	if work.iterations%portableJavaRandomStreamNativeLoopChunk == 0 {
		if err := executionContextError(work.ctx); err != nil {
			return err
		}
		if err := consumeInstruction(work.ctx); err != nil {
			return err
		}
	}
	work.iterations++
	return nil
}

func (work *portableJavaRandomStreamWork) finish() error {
	return executionContextError(work.ctx)
}

type portableJavaRandomStreamIterator struct {
	mu         sync.Mutex
	stream     *portableJavaRandomStream
	remaining  int64
	valueReady bool
	next       Value
	iterations uint64
}

func (iterator *portableJavaRandomStreamIterator) className() string {
	if iterator == nil || iterator.stream == nil {
		return "java.util.Spliterators$2Adapter"
	}
	switch iterator.stream.kind {
	case portableJavaRandomLongStream:
		return "java.util.Spliterators$3Adapter"
	case portableJavaRandomDoubleStream:
		return "java.util.Spliterators$4Adapter"
	default:
		return "java.util.Spliterators$2Adapter"
	}
}

func (iterator *portableJavaRandomStreamIterator) String() string {
	if iterator == nil {
		return "null"
	}
	return iterator.className()
}

func (iterator *portableJavaRandomStreamIterator) invoke(
	ctx context.Context,
	invocation ObjectInvocation,
) (Value, bool, error) {
	if iterator == nil || iterator.stream == nil {
		return Null(), false, nil
	}
	if invocation.Op == ObjectTypeCheck {
		return Bool(portableJavaAssignable(iterator.className(), invocation.Class)), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "hasNext", "next", "nextInt", "nextLong", "nextDouble", "remove":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, iterator.className()), true, nil
		}
	default:
		return Null(), false, nil
	}

	iterator.mu.Lock()
	defer iterator.mu.Unlock()
	switch invocation.Message {
	case "hasNext":
		ready, err := iterator.prepare(ctx)
		return portableJavaBooleanValue(ready), true, err
	case "next", "nextInt", "nextLong", "nextDouble":
		if !iterator.accepts(invocation.Message) {
			return portableNoMatchingMethod(invocation, iterator.className()), true, nil
		}
		ready, err := iterator.prepare(ctx)
		if err != nil {
			return Null(), true, err
		}
		if !ready {
			return Null(), true, errors.New("java.util.NoSuchElementException")
		}
		value := iterator.next
		iterator.next = Null()
		iterator.valueReady = false
		return value, true, nil
	case "remove":
		return Null(), true, errors.New("java.lang.UnsupportedOperationException: remove")
	default:
		return Null(), false, nil
	}
}

func (iterator *portableJavaRandomStreamIterator) accepts(message string) bool {
	if message == "next" {
		return true
	}
	switch iterator.stream.kind {
	case portableJavaRandomLongStream:
		return message == "nextLong"
	case portableJavaRandomDoubleStream:
		return message == "nextDouble"
	default:
		return message == "nextInt"
	}
}

func (iterator *portableJavaRandomStreamIterator) prepare(ctx context.Context) (bool, error) {
	if iterator.valueReady {
		return true, nil
	}
	if iterator.remaining == 0 {
		return false, executionContextError(ctx)
	}
	if iterator.iterations%portableJavaRandomStreamNativeLoopChunk == 0 {
		if err := executionContextError(ctx); err != nil {
			return false, err
		}
		if err := consumeInstruction(ctx); err != nil {
			return false, err
		}
	}
	iterator.iterations++
	iterator.next = iterator.stream.nextValue()
	iterator.remaining--
	iterator.valueReady = true
	return true, nil
}
