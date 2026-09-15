package opfor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sliverarmory/opfor/internal/bytecode"
)

// portableJavaPrimitive is the boxed Java value produced by BasicUtilities'
// casti function. Keeping the Java class alongside the scalar is observable
// through getClass(), isa, and Java object serialization.
type portableJavaPrimitive struct {
	class string
	value Value
}

func (p *portableJavaPrimitive) className() string {
	if p == nil {
		return "java.lang.Object"
	}
	return p.class
}

func (p *portableJavaPrimitive) sleepValue() Value {
	if p == nil {
		return Null()
	}
	return p.value
}

func (p *portableJavaPrimitive) String() string {
	if p == nil {
		return "null"
	}
	switch p.class {
	case "java.lang.Boolean":
		return strconv.FormatBool(p.value.Int32() != 0)
	case "java.lang.Float":
		return formatJavaFloat(float32(p.value.Float64()))
	default:
		return p.value.String()
	}
}

func formatJavaFloat(value float32) string {
	magnitude := value
	if magnitude < 0 {
		magnitude = -magnitude
	}
	if magnitude != 0 && (magnitude < 1e-3 || magnitude >= 1e7) {
		text := strconv.FormatFloat(float64(value), 'e', -1, 32)
		separator := strings.IndexByte(text, 'e')
		mantissa, exponent := text[:separator], text[separator+1:]
		if !strings.Contains(mantissa, ".") {
			mantissa += ".0"
		}
		if parsed, err := strconv.Atoi(exponent); err == nil {
			exponent = strconv.Itoa(parsed)
		}
		return mantissa + "E" + exponent
	}
	text := strconv.FormatFloat(float64(value), 'f', -1, 32)
	if !strings.Contains(text, ".") {
		text += ".0"
	}
	return text
}

func builtinCastImmediate(ctx context.Context, invocation Invocation) (Value, error) {
	value := invocation.Arg(0)
	typeName, primitive, ok := portableCastType(invocation.Arg(1))
	if !ok || !primitive {
		return portableUtilityWarning(ctx, invocation,
			fmt.Errorf("&casti: '%s' is an invalid primitive cast identifier", invocation.Arg(1).String()))
	}

	converted, err := coercePortableJavaValue(value, typeName)
	if err != nil {
		return portableUtilityWarning(ctx, invocation, err)
	}
	if typeName == "java.lang.Object" {
		return portableBoxedObjectAtRuntime(invocation.Runtime, converted)
	}
	return ObjectValue(&portableJavaPrimitive{class: portableBoxedClass(typeName), value: converted}), nil
}

func portableBoxedObject(value Value) Value {
	result, _ := portableBoxedObjectAtRuntime(nil, value)
	return result
}

func portableBoxedObjectAtRuntime(runtime *Runtime, value Value) (Value, error) {
	if value.IsNull() || value.Kind() == KindObject {
		return value, nil
	}
	if array, ok := value.Array(); ok {
		values := array.Values()
		if err := reserveCollectionEntries(runtime, len(values)); err != nil {
			return Null(), err
		}
		return ObjectValue(newPortableJavaCollection("LinkedList", values)), nil
	}
	if hash, ok := value.Hash(); ok {
		entries, _, err := portableMapEntriesReserved(HashValue(hash), func(count int) error {
			return reserveCollectionEntries(runtime, count)
		})
		if err != nil {
			return Null(), err
		}
		return ObjectValue(newPortableJavaMapFromEntries("HashMap", entries)), nil
	}
	class, ok := portableObjectClass(value)
	if !ok {
		class = "java.lang.Object"
	}
	return ObjectValue(&portableJavaPrimitive{class: class, value: value}), nil
}

func portableBoxedClass(class string) string {
	switch class {
	case "boolean":
		return "java.lang.Boolean"
	case "byte":
		return "java.lang.Byte"
	case "char":
		return "java.lang.Character"
	case "short":
		return "java.lang.Short"
	case "int":
		return "java.lang.Integer"
	case "long":
		return "java.lang.Long"
	case "float":
		return "java.lang.Float"
	case "double":
		return "java.lang.Double"
	default:
		return class
	}
}

type portableJavaArrayType struct {
	name       string
	descriptor string
	primitive  bool
}

const (
	portableJavaArrayMaximumDimensions = 255
	// Java's real limit is heap-dependent. OPFOR needs a deterministic ceiling
	// so an untrusted script cannot terminate the importing Go process with an
	// unrecoverable allocation. Count every recursively materialized Sleep-array
	// entry, not only the leaf product.
	portableJavaArrayMaximumElements = 1_000_000
)

var portableJavaArraySerial atomic.Uint64

type portableJavaArray struct {
	mu         sync.RWMutex
	typeInfo   portableJavaArrayType
	dimensions []int
	values     []Value
	identity   uint64
}

func newPortableJavaArray(typeInfo portableJavaArrayType, dimensions []int, values []Value) *portableJavaArray {
	return &portableJavaArray{
		typeInfo:   typeInfo,
		dimensions: append([]int(nil), dimensions...),
		values:     append([]Value(nil), values...),
		identity:   portableJavaArraySerial.Add(1),
	}
}

func (a *portableJavaArray) className() string {
	if a == nil {
		return "[Ljava.lang.Object;"
	}
	component := a.typeInfo.descriptor
	if component == "" {
		component = "Ljava.lang.Object;"
	}
	return strings.Repeat("[", len(a.dimensions)) + component
}

func (a *portableJavaArray) String() string {
	if a == nil {
		return "null"
	}
	return fmt.Sprintf("%s@%x", a.className(), a.identity)
}

func (a *portableJavaArray) length() int {
	if a == nil || len(a.dimensions) == 0 {
		return 0
	}
	return a.dimensions[0]
}

func (a *portableJavaArray) get(index int) (Value, error) {
	return a.getAtRuntime(nil, index)
}

func (a *portableJavaArray) getAtRuntime(runtime *Runtime, index int) (Value, error) {
	if a == nil || index < 0 || index >= a.length() {
		return Null(), fmt.Errorf("java.lang.ArrayIndexOutOfBoundsException: %d", index)
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.dimensions) == 1 {
		return a.values[index], nil
	}
	stride := portableDimensionProduct(a.dimensions[1:])
	start := index * stride
	return portableJavaArraySnapshotToSleepValue(runtime, a.typeInfo, a.dimensions[1:], a.values[start:start+stride])
}

func (a *portableJavaArray) toSleepValue() Value {
	value, _ := a.toSleepValueAtRuntime(nil)
	return value
}

func (a *portableJavaArray) toSleepValueAtRuntime(runtime *Runtime) (Value, error) {
	if a == nil {
		return Null(), nil
	}
	typeInfo, dimensions, values := a.snapshot()
	return portableJavaArraySnapshotToSleepValue(runtime, typeInfo, dimensions, values)
}

func (a *portableJavaArray) snapshot() (portableJavaArrayType, []int, []Value) {
	if a == nil {
		return portableJavaArrayType{}, nil, nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.typeInfo, append([]int(nil), a.dimensions...), append([]Value(nil), a.values...)
}

func (a *portableJavaArray) set(index int, value Value) error {
	if a == nil {
		return fmt.Errorf("java.lang.ArrayIndexOutOfBoundsException: %d", index)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if index < 0 || index >= len(a.values) || len(a.dimensions) != 1 {
		return fmt.Errorf("java.lang.ArrayIndexOutOfBoundsException: %d", index)
	}
	a.values[index] = value
	return nil
}

func portableJavaArraySnapshotToSleepValue(runtime *Runtime, typeInfo portableJavaArrayType, dimensions []int, values []Value) (Value, error) {
	return portableJavaArraySnapshotToSleepValueWithPrepaidLeaves(runtime, typeInfo, dimensions, values, false)
}

func portableJavaArraySnapshotToSleepValueWithPrepaidLeaves(
	runtime *Runtime,
	typeInfo portableJavaArrayType,
	dimensions []int,
	values []Value,
	prepaidLeaves bool,
) (Value, error) {
	if len(dimensions) == 1 && typeInfo.name == "byte" {
		data := make([]byte, len(values))
		for index, value := range values {
			data[index] = byte(int8(value.Int32()))
		}
		return BinaryString(data), nil
	}
	if len(dimensions) == 1 && typeInfo.name == "char" {
		units := make([]uint16, len(values))
		for index, value := range values {
			encoded := sleepStringUnits(value)
			if len(encoded) != 0 {
				units[index] = encoded[0]
			}
		}
		return sleepStringValueFromUnits(units, nil), nil
	}
	result := make([]Value, 0)
	if len(dimensions) != 0 {
		result = make([]Value, dimensions[0])
	}
	if len(dimensions) == 1 {
		copy(result, values)
	} else {
		stride := portableDimensionProduct(dimensions[1:])
		for index := range result {
			start := index * stride
			converted, err := portableJavaArraySnapshotToSleepValueWithPrepaidLeaves(
				runtime, typeInfo, dimensions[1:], values[start:start+stride], prepaidLeaves,
			)
			if err != nil {
				return Null(), err
			}
			result[index] = converted
		}
	}
	if prepaidLeaves && len(dimensions) == 1 {
		return ArrayValue(NewArray(result...)), nil
	}
	array, err := newRuntimeArray(runtime, result...)
	if err != nil {
		return Null(), err
	}
	return ArrayValue(array), nil
}

func builtinCast(ctx context.Context, invocation Invocation) (Value, error) {
	value := invocation.Arg(0)
	if _, ok := value.Array(); !ok {
		switch invocation.Arg(1).String() {
		case "c":
			units := sleepStringUnits(value)
			if err := reserveCollectionEntries(invocation.Runtime, len(units)); err != nil {
				return Null(), err
			}
			values := make([]Value, len(units))
			for index, unit := range units {
				values[index] = sleepUTF16CharacterValue(unit)
			}
			typeInfo, _, _ := portableCastType(String("c"))
			return ObjectValue(newPortableJavaArray(portableArrayType(typeInfo), []int{len(values)}, values)), nil
		case "b":
			data := sleepStringLowBytes(value)
			if err := reserveCollectionEntries(invocation.Runtime, len(data)); err != nil {
				return Null(), err
			}
			values := make([]Value, len(data))
			for index, octet := range data {
				values[index] = Int(int32(int8(octet)))
			}
			typeInfo, _, _ := portableCastType(String("b"))
			return ObjectValue(newPortableJavaArray(portableArrayType(typeInfo), []int{len(values)}, values)), nil
		default:
			return Null(), nil
		}
	}

	array, _ := value.Array()
	dimensionCapacity := len(invocation.Arguments) - 2
	if dimensionCapacity < 0 {
		dimensionCapacity = 0
	}
	dimensions := make([]int, 0, dimensionCapacity)
	for index := 2; index < len(invocation.Arguments); index++ {
		dimensions = append(dimensions, int(sleepInt32(invocation.Arg(index))))
	}
	if len(dimensions) == 0 {
		dimensions = []int{array.Len()}
	}
	for _, dimension := range dimensions {
		if dimension < 0 {
			return portableUtilityWarning(ctx, invocation,
				fmt.Errorf("java.lang.NegativeArraySizeException: %d", dimension))
		}
	}

	var flat []Value
	visiting := make(map[*Array]bool)
	if err := visitIteratorValues(ctx, value, invocation.Name, func(element Value) error {
		return flattenValueAtExecution(ctx, element, &flat, visiting, func() error {
			return reserveCollectionEntries(invocation.Runtime, 1)
		})
	}); err != nil {
		if isExecutionResourceError(err) {
			return Null(), err
		}
		return portableUtilityWarning(ctx, invocation, err)
	}
	if total := portableDimensionProduct(dimensions); total != len(flat) {
		return portableUtilityWarning(ctx, invocation, fmt.Errorf(
			"&cast: specified dimensions %d is not equal to total array elements %d", total, len(flat)))
	}

	typeInfo := portableArrayTypeForValue(invocation.Arg(1), value)
	converted := make([]Value, len(flat))
	for index, element := range flat {
		coerced, err := coercePortableJavaValue(element, typeInfo.name)
		if err != nil {
			return portableUtilityWarning(ctx, invocation, err)
		}
		converted[index] = coerced
	}
	return ObjectValue(newPortableJavaArray(typeInfo, dimensions, converted)), nil
}

func portableArrayTypeForValue(description, source Value) portableJavaArrayType {
	if name, _, ok := portableCastType(description); ok {
		return portableArrayType(name)
	}
	return portableArrayType(inferPortableArrayClass(source))
}

func portableCastType(description Value) (name string, primitive bool, ok bool) {
	if class, classOK := portableClassOperand(description); classOK && class != "" {
		return resolvePortableClassName(class), false, true
	}
	if description.Kind() != KindString || len(description.String()) != 1 {
		return "", false, false
	}
	switch description.String()[0] {
	case 'z':
		return "boolean", true, true
	case 'c':
		return "char", true, true
	case 'b':
		return "byte", true, true
	case 'h':
		return "short", true, true
	case 'i':
		return "int", true, true
	case 'l':
		return "long", true, true
	case 'f':
		return "float", true, true
	case 'd':
		return "double", true, true
	case 'o':
		return "java.lang.Object", true, true
	default:
		return "", false, false
	}
}

func portableArrayType(name string) portableJavaArrayType {
	switch name {
	case "boolean":
		return portableJavaArrayType{name: name, descriptor: "Z", primitive: true}
	case "char":
		return portableJavaArrayType{name: name, descriptor: "C", primitive: true}
	case "byte":
		return portableJavaArrayType{name: name, descriptor: "B", primitive: true}
	case "short":
		return portableJavaArrayType{name: name, descriptor: "S", primitive: true}
	case "int":
		return portableJavaArrayType{name: name, descriptor: "I", primitive: true}
	case "long":
		return portableJavaArrayType{name: name, descriptor: "J", primitive: true}
	case "float":
		return portableJavaArrayType{name: name, descriptor: "F", primitive: true}
	case "double":
		return portableJavaArrayType{name: name, descriptor: "D", primitive: true}
	default:
		name = resolvePortableClassName(name)
		if name == "" {
			name = "java.lang.Object"
		}
		return portableJavaArrayType{name: name, descriptor: "L" + name + ";"}
	}
}

func inferPortableArrayClass(value Value) string {
	array, ok := value.Array()
	if !ok || array == nil {
		return "java.lang.Object"
	}
	for _, element := range array.Values() {
		if nested, ok := element.Array(); ok {
			return inferPortableArrayClass(ArrayValue(nested))
		}
		if element.IsNull() {
			continue
		}
		switch element.Kind() {
		case KindInt:
			return "int"
		case KindLong:
			return "long"
		case KindDouble:
			return "double"
		case KindString:
			return "java.lang.String"
		case KindObject:
			if class, ok := portableObjectClass(element); ok {
				return class
			}
		}
	}
	return "java.lang.Object"
}

func coercePortableJavaValue(value Value, class string) (Value, error) {
	if primitive, ok := value.Object(); ok {
		if boxed, boxedOK := primitive.(*portableJavaPrimitive); boxedOK && boxed != nil {
			value = boxed.sleepValue()
		}
	}
	switch class {
	case "boolean":
		if sleepInt32(value) != 0 {
			return Int(1), nil
		}
		return Int(0), nil
	case "byte":
		return Int(int32(int8(sleepInt32(value)))), nil
	case "char":
		units := sleepStringUnits(value)
		if len(units) == 0 {
			return Null(), errors.New("java.lang.StringIndexOutOfBoundsException: String index out of range: 0")
		}
		return sleepUTF16CharacterValue(units[0]), nil
	case "short":
		return Int(int32(int16(sleepInt32(value)))), nil
	case "int":
		return Int(sleepInt32(value)), nil
	case "long":
		return Long(sleepInt64(value)), nil
	case "float":
		return Double(float64(float32(sleepFloat64(value)))), nil
	case "double":
		return Double(sleepFloat64(value)), nil
	case "java.lang.Object":
		return value, nil
	case "java.lang.String":
		if value.IsNull() {
			return Null(), nil
		}
		return sleepStringCoercion(value), nil
	default:
		if value.IsNull() {
			return Null(), nil
		}
		actual, ok := portableObjectClass(value)
		if ok && portableJavaAssignable(actual, class) {
			return value, nil
		}
		return Null(), fmt.Errorf("%s is not compatible with %s", value.Describe(), class)
	}
}

func portableDimensionProduct(dimensions []int) int {
	total := 1
	maxInt := int(^uint(0) >> 1)
	for _, dimension := range dimensions {
		if dimension == 0 {
			return 0
		}
		if dimension > 0 && total > maxInt/dimension {
			return maxInt
		}
		total *= dimension
	}
	return total
}

// portableJavaProxy is an inert java.lang.reflect.Proxy counterpart. Method
// calls execute the backing Sleep closure with $0 set to the Java method name.
type portableJavaProxy struct {
	closure    Value
	interfaces []string
	class      *portableJavaProxyClass
	runtime    *Runtime
	stringMu   sync.Mutex
	stringing  bool
}

type portableJavaProxyClass struct {
	name       string
	interfaces []string
}

func (c *portableJavaProxyClass) String() string {
	if c == nil {
		return "class com.sun.proxy.$Proxy0"
	}
	return "class " + c.name
}

func (p *portableJavaProxy) className() string {
	if p == nil || p.class == nil {
		return "com.sun.proxy.$Proxy0"
	}
	return p.class.name
}

func (p *portableJavaProxy) SleepDescribe() string {
	if p == nil {
		return "$null"
	}
	return "[" + describeTraceValue(p.closure) + " as " + strings.Join(p.interfaces, ", ") + "]"
}

func (p *portableJavaProxy) String() string {
	if p == nil {
		return "null"
	}
	p.stringMu.Lock()
	if p.stringing {
		p.stringMu.Unlock()
		return p.SleepDescribe()
	}
	p.stringing = true
	p.stringMu.Unlock()
	defer func() {
		p.stringMu.Lock()
		p.stringing = false
		p.stringMu.Unlock()
	}()
	ctx := context.Background()
	if p.runtime != nil {
		ctx = withExecutionMeter(ctx, p.runtime)
	}
	value, err := p.call(ctx, "toString", nil, false)
	if err != nil || value.IsNull() {
		return p.SleepDescribe()
	}
	return value.String()
}

func (r *Runtime) newPortableInstance(ctx context.Context, invocation Invocation) (Value, error) {
	interfaces := make([]string, 0, 1)
	target := invocation.Arg(0)
	if array, ok := target.Array(); ok {
		for _, value := range array.Values() {
			class, ok := portableClassOperand(value)
			if !ok || class == "" {
				return portableUtilityWarning(ctx, invocation, newPortableInvalidClassCast(value))
			}
			interfaces = append(interfaces, resolvePortableClassName(class))
		}
	} else {
		class, ok := portableClassOperand(target)
		if !ok || class == "" {
			return portableUtilityWarning(ctx, invocation, newPortableInvalidClassCast(target))
		}
		interfaces = append(interfaces, resolvePortableClassName(class))
	}
	if len(interfaces) == 0 {
		return portableUtilityWarning(ctx, invocation, errors.New("&newInstance: no interfaces specified"))
	}
	closure := invocation.Arg(1)
	if _, ok := closure.Function(); !ok {
		return portableUtilityWarning(ctx, invocation, errors.New("&newInstance: expected a function"))
	}
	return ObjectValue(r.newPortableJavaProxy(closure, interfaces)), nil
}

// newPortableJavaProxy creates a proxy whose class identity remains visible if
// Java code retains and later returns an automatically adapted Sleep closure.
// Transient adapters may omit this metadata, but retained wrappers such as
// Collections.reverseOrder(Comparator) must behave like newInstance proxies.
func (r *Runtime) newPortableJavaProxy(closure Value, interfaces []string) *portableJavaProxy {
	resolved := make([]string, len(interfaces))
	for index, implemented := range interfaces {
		resolved[index] = resolvePortableClassName(implemented)
	}
	serial := uint64(0)
	if r != nil {
		r.mu.Lock()
		serial = r.nextProxy
		r.nextProxy++
		r.mu.Unlock()
	}
	class := &portableJavaProxyClass{
		name:       fmt.Sprintf("com.sun.proxy.$Proxy%d", serial),
		interfaces: append([]string(nil), resolved...),
	}
	return &portableJavaProxy{
		closure: closure, interfaces: resolved, class: class, runtime: r,
	}
}

func (p *portableJavaProxy) invoke(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		target := resolvePortableClassName(invocation.Class)
		if target == "java.lang.Object" || target == "java.lang.reflect.Proxy" || target == "java.io.Serializable" {
			return Int(1), true, nil
		}
		for _, implemented := range p.interfaces {
			if implemented == target {
				return Int(1), true, nil
			}
		}
		return Int(0), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	if invocation.Message == "getClass" && len(invocation.Arguments) == 0 {
		return ObjectValue(p.class), true, nil
	}
	result, err := p.call(ctx, invocation.Message, invocation.Arguments, true)
	if err != nil {
		return Null(), true, err
	}
	switch invocation.Message {
	case "add", "addAll", "contains", "containsAll", "equals", "isEmpty", "remove", "removeAll", "retainAll":
		if sleepInt32(result) != 0 {
			return Int(1), true, nil
		}
		return Int(0), true, nil
	case "compareTo", "hashCode", "size":
		return Int(sleepInt32(result)), true, nil
	case "toString":
		return String(result.String()), true, nil
	default:
		return result, true, nil
	}
}

func (p *portableJavaProxy) call(ctx context.Context, message string, arguments []Argument, trace bool) (Value, error) {
	callable, ok := p.closure.Function()
	if !ok {
		return Null(), ErrInvalidCallable
	}
	bound := append([]Argument(nil), arguments...)
	bound = append(bound, Argument{Name: "$0", Value: String(message)})
	var result Value
	var err error
	if closure, ok := callable.(*scriptClosure); ok && closure != nil {
		result, err = closure.invokeArguments(ctx, bound)
	} else if target, ok := callable.(interface {
		invokeArguments(context.Context, []Argument) (Value, error)
	}); ok {
		result, err = target.invokeArguments(ctx, bound)
	} else {
		result, err = callable.Invoke(ctx, resolvedArguments(arguments)...)
	}
	caller := currentFiber(ctx)
	if trace && caller != nil && caller.callTraceEnabled() {
		call := formatClosureCall(p.closure, message, arguments)
		if err != nil {
			call += " - FAILED!"
		} else {
			call += " = " + describeTraceValue(result)
		}
		caller.writeTraceMessage(call, Span{Source: "<Java>", Start: Position{Line: -1}})
	}
	return result, err
}

func (c *portableJavaProxyClass) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		class := resolvePortableClassName(invocation.Class)
		return Bool(class == "java.lang.Class" || class == "java.lang.Object"), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "getInterfaces":
		values := make([]Value, len(c.interfaces))
		for index, class := range c.interfaces {
			values[index] = ObjectValue(classReference(class))
		}
		array, err := newRuntimeArray(invocation.Runtime, values...)
		if err != nil {
			return Null(), true, err
		}
		return ArrayValue(array), true, nil
	case "getName":
		return String(c.name), true, nil
	case "toString":
		return String(c.String()), true, nil
	}
	return Null(), false, nil
}

type portableJavaStringBuffer struct {
	// Both portable classes use the mutex so Go importers may inspect them
	// without data races. This is an implementation-safety guarantee, not a
	// promise that java.lang.StringBuilder has Java StringBuffer's synchronized
	// method contract; scripts must still treat StringBuilder as single-threaded.
	mu       sync.RWMutex
	class    string
	units    []uint16
	raw      []bool
	capacity int
}

type portableJavaMessageDigest struct {
	algorithm string
	state     *sleepDigestState
}

func (digest *portableJavaMessageDigest) String() string {
	if digest == nil {
		return "null"
	}
	return digest.algorithm + " Message Digest"
}

func (digest *portableJavaMessageDigest) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		class := resolvePortableClassName(invocation.Class)
		return Bool(class == "java.security.MessageDigest" || class == "java.security.MessageDigestSpi" || class == "java.lang.Object"), true, nil
	}
	if invocation.Op != ObjectInvoke || digest == nil || digest.state == nil {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "update":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.security.MessageDigest"), true, nil
		}
		data, ok := portableJavaByteArray(invocation.Arg(0))
		if !ok {
			return portableNoMatchingMethod(invocation, "java.security.MessageDigest"), true, nil
		}
		digest.state.write(data)
		return Null(), true, nil
	case "digest":
		if len(invocation.Arguments) > 1 {
			return portableNoMatchingMethod(invocation, "java.security.MessageDigest"), true, nil
		}
		if len(invocation.Arguments) == 1 {
			data, ok := portableJavaByteArray(invocation.Arg(0))
			if !ok {
				return portableNoMatchingMethod(invocation, "java.security.MessageDigest"), true, nil
			}
			digest.state.write(data)
		}
		return BinaryString(digest.state.sumAndReset()), true, nil
	case "reset":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.security.MessageDigest"), true, nil
		}
		digest.state.mu.Lock()
		digest.state.digest.Reset()
		digest.state.mu.Unlock()
		return Null(), true, nil
	case "getAlgorithm":
		if len(invocation.Arguments) == 0 {
			return String(digest.algorithm), true, nil
		}
	}
	return Null(), false, nil
}

func portableJavaByteArray(value Value) ([]byte, bool) {
	if value.Kind() == KindString {
		return sleepStringLowBytes(value), true
	}
	object, ok := value.Object()
	if !ok {
		return nil, false
	}
	array, ok := object.(*portableJavaArray)
	if !ok || array == nil || array.typeInfo.name != "byte" || len(array.dimensions) != 1 {
		return nil, false
	}
	_, _, values := array.snapshot()
	data := make([]byte, len(values))
	for index, element := range values {
		data[index] = byte(int8(element.Int32()))
	}
	return data, true
}

func (b *portableJavaStringBuffer) String() string {
	if b == nil {
		return "null"
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return sleepRenderStringUnits(b.units, b.raw)
}

var portableJavaPrintStreamSerial atomic.Uint64

type portableJavaPrintStream struct {
	runtime  *Runtime
	identity uint64
}

func newPortableJavaPrintStream(runtime *Runtime) *portableJavaPrintStream {
	return &portableJavaPrintStream{runtime: runtime, identity: portableJavaPrintStreamSerial.Add(1)}
}

func (r *Runtime) portableSystemOut() *portableJavaPrintStream {
	if r == nil {
		return newPortableJavaPrintStream(nil)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.printStream == nil {
		r.printStream = newPortableJavaPrintStream(r)
	}
	return r.printStream
}

func (stream *portableJavaPrintStream) String() string {
	if stream == nil {
		return "null"
	}
	return fmt.Sprintf("java.io.PrintStream@%x", stream.identity)
}

func (stream *portableJavaPrintStream) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		class := resolvePortableClassName(invocation.Class)
		return Bool(class == "java.io.PrintStream" || class == "java.io.FilterOutputStream" || class == "java.io.OutputStream" || class == "java.lang.Object"), true, nil
	}
	if invocation.Op != ObjectInvoke || (invocation.Message != "print" && invocation.Message != "println") || len(invocation.Arguments) > 1 {
		return Null(), false, nil
	}
	text := ""
	if len(invocation.Arguments) == 1 {
		value := invocation.Arg(0)
		if value.IsNull() {
			text = "null"
		} else if object, ok := value.Object(); ok {
			if array, ok := object.(*portableJavaArray); ok && array != nil && array.typeInfo.name == "char" && len(array.dimensions) == 1 {
				text = array.toSleepValue().String()
			} else {
				text = value.String()
			}
		} else {
			text = sleepOutputString(value)
		}
	}
	if stream.runtime == nil || stream.runtime.stdout == nil {
		return Null(), true, nil
	}
	var err error
	if invocation.Message == "println" {
		_, err = fmt.Fprintln(stream.runtime.stdout, text)
	} else {
		_, err = fmt.Fprint(stream.runtime.stdout, text)
	}
	return Null(), true, err
}

type portableJavaPoint struct {
	mu sync.RWMutex
	x  int32
	y  int32
}

func (p *portableJavaPoint) String() string {
	if p == nil {
		return "java.awt.Point[x=0,y=0]"
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return fmt.Sprintf("java.awt.Point[x=%d,y=%d]", p.x, p.y)
}

func (p *portableJavaPoint) invoke(invocation ObjectInvocation) (Value, bool, error) {
	switch invocation.Op {
	case ObjectGet:
		p.mu.RLock()
		defer p.mu.RUnlock()
		switch invocation.Message {
		case "x":
			return Int(p.x), true, nil
		case "y":
			return Int(p.y), true, nil
		}
	case ObjectSet:
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.awt.Point"), true, nil
		}
		p.mu.Lock()
		defer p.mu.Unlock()
		switch invocation.Message {
		case "x":
			p.x = sleepInt32(invocation.Arg(0))
			return Null(), true, nil
		case "y":
			p.y = sleepInt32(invocation.Arg(0))
			return Null(), true, nil
		}
	case ObjectInvoke:
		if invocation.Message == "toString" && len(invocation.Arguments) == 0 {
			return String(p.String()), true, nil
		}
	case ObjectTypeCheck:
		class := resolvePortableClassName(invocation.Class)
		return Bool(class == "java.awt.Point" || class == "java.awt.geom.Point2D" || class == "java.lang.Object"), true, nil
	}
	return Null(), false, nil
}

func builtinSetField(ctx context.Context, invocation Invocation) (Value, error) {
	target := invocation.Arg(0)
	if target.IsNull() {
		return portableUtilityWarning(ctx, invocation, errors.New("&setField: can not set field on a null object"))
	}
	if invocation.Runtime == nil || invocation.Runtime.objectHost == nil {
		return Null(), errors.New("&setField: object host is unavailable")
	}
	for index := 1; index < len(invocation.Arguments); index++ {
		field := invocation.Arguments[index]
		name, value, ok := sleepNamedArgument(field)
		if !ok {
			return portableUtilityWarning(ctx, invocation, errors.New("&setField: expected a field => value pair"))
		}
		request := ObjectInvocation{
			Runtime: invocation.Runtime,
			Script:  invocation.Script,
			Op:      ObjectSet,
			Target:  target,
			Message: name,
			Span:    invocation.Span,
			Arguments: []Argument{{
				Value: value, Reference: sleepArgumentReference(field),
			}},
		}
		if class, ok := portableClassOperand(target); ok && class != "" {
			request.Class = class
			request.Target = Null()
		}
		if _, err := invocation.Runtime.objectHost.Object(ctx, request); err != nil {
			var fieldErr *portableFieldError
			if errors.As(err, &fieldErr) {
				return portableUtilityWarning(ctx, invocation, fieldErr)
			}
			var unsupported *UnsupportedError
			if errors.As(err, &unsupported) {
				class := request.Class
				if class == "" {
					class, _ = portableObjectClass(target)
				}
				if class == "" {
					class = "java.lang.Object"
				}
				return portableUtilityWarning(ctx, invocation,
					fmt.Errorf("no field named %s in class %s", name, class))
			}
			return Null(), preserveNativeBoundaryError(ctx, err)
		}
	}
	return Null(), nil
}

func portableUtilityWarning(ctx context.Context, invocation Invocation, err error) (Value, error) {
	if fiber := currentFiber(ctx); fiber != nil {
		if fiber.skipEnclosingConditionalBlock() {
			if invocation.Runtime != nil {
				invocation.Runtime.writeWarning(err.Error(), invocation.Span)
			}
			return Null(), nil
		}
		return Null(), &uncaughtScriptWarning{err: err}
	}
	return Null(), err
}

// skipEnclosingConditionalBlock models Sleep's Block-level exception boundary
// for a bridge RuntimeException raised inside if/while/for bodies. The first
// containing conditional jump is the innermost compiled block. OpEval advances
// the program counter after the builtin returns, hence destination-1 here.
func (f *fiber) skipEnclosingConditionalBlock() bool {
	if f == nil || f.function == nil || f.pc < 0 || f.pc >= len(f.function.Instructions) {
		return false
	}
	for index := f.pc - 1; index >= 0; index-- {
		instruction := f.function.Instructions[index]
		if instruction.Op != bytecode.OpJumpFalse && instruction.Op != bytecode.OpAssignWhile {
			continue
		}
		if instruction.Target <= f.pc || instruction.Target > len(f.function.Instructions) {
			continue
		}
		destination := instruction.Target
		// An if/else has a forward jump immediately before the else target;
		// a loop's corresponding jump points backward and is left untouched.
		if before := instruction.Target - 1; before > f.pc {
			jump := f.function.Instructions[before]
			if jump.Op == bytecode.OpJump && jump.Target > instruction.Target {
				destination = jump.Target
			}
		}
		f.pc = destination - 1
		return true
	}
	return false
}

func portableJavaUtilityTarget(ctx context.Context, target any, invocation ObjectInvocation) (Value, bool, error) {
	switch object := target.(type) {
	case *portableJavaProxy:
		if object != nil {
			return object.invoke(ctx, invocation)
		}
	case *portableJavaProxyClass:
		if object != nil {
			return object.invoke(invocation)
		}
	case *portableJavaPoint:
		if object != nil {
			return object.invoke(invocation)
		}
	case *portableJavaPrintStream:
		if object != nil {
			return object.invoke(invocation)
		}
	case *portableJavaMessageDigest:
		if object != nil {
			return object.invoke(invocation)
		}
	case *portableJavaStringBuffer:
		if object != nil {
			return object.invoke(invocation)
		}
	}
	return Null(), false, nil
}

func portableJavaUtilityClass(object any) (string, bool) {
	switch value := object.(type) {
	case *portableJavaPrimitive:
		return value.className(), value != nil
	case *portableJavaArray:
		return value.className(), value != nil
	case *portableJavaProxy:
		return value.className(), value != nil
	case *portableJavaProxyClass:
		return "java.lang.Class", value != nil
	case *portableJavaStringBuffer:
		if value != nil {
			return value.class, true
		}
	case *portableJavaPoint:
		return "java.awt.Point", value != nil
	case *portableJavaPrintStream:
		return "java.io.PrintStream", value != nil
	case *portableJavaMessageDigest:
		return "java.security.MessageDigest", value != nil
	}
	return "", false
}

func portableJavaArrays(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "equals":
		if len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, "java.util.Arrays"), true, nil
		}
		left, leftType, leftOK := portableArrayComparableValues(invocation.Arg(0))
		right, rightType, rightOK := portableArrayComparableValues(invocation.Arg(1))
		if !leftOK || !rightOK || len(left) != len(right) || leftType.descriptor != rightType.descriptor {
			return Int(0), true, nil
		}
		for index := range left {
			if !portableJavaEqual(left[index], right[index]) {
				return Int(0), true, nil
			}
		}
		return Int(1), true, nil
	case "binarySearch":
		if len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, "java.util.Arrays"), true, nil
		}
		values, typeInfo, ok := portableArrayComparableValues(invocation.Arg(0))
		if !ok {
			return portableNoMatchingMethod(invocation, "java.util.Arrays"), true, nil
		}
		needle, err := coercePortableJavaValue(invocation.Arg(1), typeInfo.name)
		if err != nil {
			return Null(), true, err
		}
		position := sort.Search(len(values), func(index int) bool {
			return portableJavaCompare(values[index], needle) >= 0
		})
		if position < len(values) && portableJavaCompare(values[position], needle) == 0 {
			return Int(int32(position)), true, nil
		}
		return Int(int32(-position - 1)), true, nil
	}
	return Null(), false, nil
}

func portableArrayComparableValues(value Value) ([]Value, portableJavaArrayType, bool) {
	if object, ok := value.Object(); ok {
		if array, ok := object.(*portableJavaArray); ok && array != nil && len(array.dimensions) == 1 {
			typeInfo, _, values := array.snapshot()
			return values, typeInfo, true
		}
	}
	array, ok := value.Array()
	if !ok || array == nil {
		return nil, portableJavaArrayType{}, false
	}
	typeInfo := portableArrayType(inferPortableArrayClass(value))
	values := array.Values()
	for index, element := range values {
		converted, err := coercePortableJavaValue(element, typeInfo.name)
		if err != nil {
			return nil, portableJavaArrayType{}, false
		}
		values[index] = converted
	}
	return values, typeInfo, true
}

func portableJavaReflectArray(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	if invocation.Message == "newInstance" {
		if len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, "java.lang.reflect.Array"), true, nil
		}
		component, ok := portableClassOperand(invocation.Arg(0))
		if !ok || component == "" {
			return portableNoMatchingMethod(invocation, "java.lang.reflect.Array"), true, nil
		}
		dimensions, err := portableJavaArrayDimensions(invocation.Arg(1))
		if err != nil {
			return Null(), true, err
		}
		if len(dimensions)+portableJavaArrayComponentDimensions(component) > portableJavaArrayMaximumDimensions {
			return Null(), true, errors.New("java.lang.IllegalArgumentException")
		}
		typeInfo := portableArrayType(component)
		length, err := portableJavaArrayAllocationLength(dimensions)
		if err != nil {
			return Null(), true, err
		}
		prepaidLeaves := len(dimensions) != 0 && typeInfo.name != "byte" && typeInfo.name != "char"
		if prepaidLeaves {
			if err := reserveCollectionEntries(invocation.Runtime, length); err != nil {
				return Null(), true, err
			}
		}
		values := make([]Value, length)
		for index := range values {
			values[index] = portableJavaArrayZero(typeInfo)
		}
		// java.lang.reflect.Array.newInstance returns through Sleep's ordinary
		// Java-method bridge. ObjectUtilities.BuildScalar eagerly converts that
		// result: byte[]/char[] become strings and every other array becomes a
		// fresh (recursively converted) ListContainer. Arrays intentionally
		// produced by Sleep's cast() remain opaque portableJavaArray objects.
		array := newPortableJavaArray(typeInfo, dimensions, values)
		arrayType, arrayDimensions, arrayValues := array.snapshot()
		value, err := portableJavaArraySnapshotToSleepValueWithPrepaidLeaves(
			invocation.Runtime, arrayType, arrayDimensions, arrayValues, prepaidLeaves,
		)
		return value, true, err
	}
	if len(invocation.Arguments) == 0 {
		return portableNoMatchingMethod(invocation, "java.lang.reflect.Array"), true, nil
	}
	target := invocation.Arg(0)
	if object, ok := target.Object(); ok {
		if array, ok := object.(*portableJavaArray); ok && array != nil {
			switch invocation.Message {
			case "getLength":
				if len(invocation.Arguments) != 1 {
					return portableNoMatchingMethod(invocation, "java.lang.reflect.Array"), true, nil
				}
				return Int(int32(array.length())), true, nil
			case "get":
				if len(invocation.Arguments) != 2 {
					return portableNoMatchingMethod(invocation, "java.lang.reflect.Array"), true, nil
				}
				value, err := array.getAtRuntime(invocation.Runtime, int(sleepInt32(invocation.Arg(1))))
				return value, true, err
			case "set":
				if len(invocation.Arguments) != 3 {
					return portableNoMatchingMethod(invocation, "java.lang.reflect.Array"), true, nil
				}
				index := int(sleepInt32(invocation.Arg(1)))
				if index < 0 || index >= array.length() {
					return Null(), true, fmt.Errorf("java.lang.ArrayIndexOutOfBoundsException: %d", index)
				}
				if len(array.dimensions) != 1 {
					return portableNoMatchingMethod(invocation, "java.lang.reflect.Array"), true, nil
				}
				converted, err := coercePortableJavaReflectArrayValue(invocation.Arg(2), array.typeInfo.name)
				if err != nil {
					return Null(), true, err
				}
				if err := array.set(index, converted); err != nil {
					return Null(), true, err
				}
				return Null(), true, nil
			}
		}
	}
	switch invocation.Message {
	case "getLength":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.lang.reflect.Array"), true, nil
		}
		return Null(), true, errors.New("java.lang.IllegalArgumentException: Argument is not an array")
	case "get":
		if len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, "java.lang.reflect.Array"), true, nil
		}
		return Null(), true, errors.New("java.lang.IllegalArgumentException: Argument is not an array")
	case "set":
		if len(invocation.Arguments) != 3 {
			return portableNoMatchingMethod(invocation, "java.lang.reflect.Array"), true, nil
		}
		return Null(), true, errors.New("java.lang.IllegalArgumentException: Argument is not an array")
	}
	return portableNoMatchingMethod(invocation, "java.lang.reflect.Array"), true, nil
}

func portableJavaArrayComponentDimensions(component string) int {
	component = resolvePortableClassName(component)
	dimensions := 0
	for dimensions < len(component) && component[dimensions] == '[' {
		dimensions++
	}
	if dimensions != 0 {
		return dimensions
	}
	for strings.HasSuffix(component, "[]") {
		dimensions++
		component = strings.TrimSuffix(component, "[]")
	}
	return dimensions
}

func portableJavaArrayAllocationLength(dimensions []int) (int, error) {
	leafProduct := 1
	materializedEntries := 0
	for _, dimension := range dimensions {
		if dimension < 0 {
			return 0, fmt.Errorf("java.lang.NegativeArraySizeException: %d", dimension)
		}
		if dimension == 0 {
			return 0, nil
		}
		if leafProduct > portableJavaArrayMaximumElements/dimension {
			return 0, errors.New("java.lang.OutOfMemoryError: Required array size exceeds implementation limit")
		}
		leafProduct *= dimension
		if materializedEntries > portableJavaArrayMaximumElements-leafProduct {
			return 0, errors.New("java.lang.OutOfMemoryError: Required array size exceeds implementation limit")
		}
		materializedEntries += leafProduct
	}
	return leafProduct, nil
}

func portableJavaArrayDimensions(value Value) ([]int, error) {
	if array, ok := value.Array(); ok && array != nil {
		values := array.Values()
		dimensions := make([]int, len(values))
		for index, value := range values {
			dimensions[index] = int(sleepInt32(value))
			if dimensions[index] < 0 {
				return nil, fmt.Errorf("java.lang.NegativeArraySizeException: %d", dimensions[index])
			}
		}
		if len(dimensions) == 0 {
			return nil, errors.New("java.lang.IllegalArgumentException: dimensions array is empty")
		}
		return dimensions, nil
	}
	dimension := int(sleepInt32(value))
	if dimension < 0 {
		return nil, fmt.Errorf("java.lang.NegativeArraySizeException: %d", dimension)
	}
	return []int{dimension}, nil
}

func portableJavaArrayZero(typeInfo portableJavaArrayType) Value {
	switch typeInfo.name {
	case "char":
		return sleepUTF16CharacterValue(0)
	case "long":
		return Long(0)
	case "float", "double":
		return Double(0)
	case "boolean", "byte", "short", "int":
		return Int(0)
	default:
		return Null()
	}
}

func coercePortableJavaReflectArrayValue(value Value, class string) (Value, error) {
	if class != "char" {
		return coercePortableJavaValue(value, class)
	}
	object, ok := value.Object()
	if !ok {
		return Null(), errors.New("java.lang.IllegalArgumentException: argument type mismatch")
	}
	character, ok := object.(*portableJavaPrimitive)
	if !ok || character == nil || character.className() != "java.lang.Character" {
		return Null(), errors.New("java.lang.IllegalArgumentException: argument type mismatch")
	}
	return coercePortableJavaValue(character.sleepValue(), class)
}
