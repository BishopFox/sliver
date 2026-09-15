package opfor

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"unicode/utf16"

	"github.com/sliverarmory/opfor/internal/bytecode"
	"github.com/sliverarmory/opfor/internal/javaser"
)

const sleepSerializationProfileName = "sleep-2.1-official"

var (
	javaStringType       = javaser.NewString("Ljava/lang/String;")
	javaObjectType       = javaser.NewString("Ljava/lang/Object;")
	javaListType         = javaser.NewString("Ljava/util/List;")
	javaMapType          = javaser.NewString("Ljava/util/Map;")
	sleepScalarArrayType = javaser.NewString("Lsleep/runtime/ScalarArray;")
	sleepScalarHashType  = javaser.NewString("Lsleep/runtime/ScalarHash;")
	sleepScalarType      = javaser.NewString("Lsleep/runtime/ScalarType;")
	sleepOrderedHashType = javaser.NewString("Lsleep/engine/types/OrderedHashContainer;")

	sleepScalarDescriptor = &javaser.ClassDesc{
		Name:             "sleep.runtime.Scalar",
		SerialVersionUID: -4850599259538399162,
		Flags:            javaser.SCSerializable | javaser.SCWriteMethod,
		Fields: []javaser.FieldDesc{
			{TypeCode: javaser.TypeObject, Name: "array", ClassName: sleepScalarArrayType},
			{TypeCode: javaser.TypeObject, Name: "hash", ClassName: sleepScalarHashType},
			{TypeCode: javaser.TypeObject, Name: "value", ClassName: sleepScalarType},
		},
	}
	sleepStringValueDescriptor = &javaser.ClassDesc{
		Name:             "sleep.engine.types.StringValue",
		SerialVersionUID: 1979570663676146016,
		Flags:            javaser.SCSerializable,
		Fields: []javaser.FieldDesc{
			{TypeCode: javaser.TypeObject, Name: "value", ClassName: javaStringType},
		},
	}
	sleepIntValueDescriptor = &javaser.ClassDesc{
		Name:             "sleep.engine.types.IntValue",
		SerialVersionUID: 58957799476976928,
		Flags:            javaser.SCSerializable,
		Fields:           []javaser.FieldDesc{{TypeCode: javaser.TypeInt, Name: "value"}},
	}
	sleepLongValueDescriptor = &javaser.ClassDesc{
		Name:             "sleep.engine.types.LongValue",
		SerialVersionUID: -8590950665834023129,
		Flags:            javaser.SCSerializable,
		Fields:           []javaser.FieldDesc{{TypeCode: javaser.TypeLong, Name: "value"}},
	}
	sleepDoubleValueDescriptor = &javaser.ClassDesc{
		Name:             "sleep.engine.types.DoubleValue",
		SerialVersionUID: -3878817456786564555,
		Flags:            javaser.SCSerializable,
		Fields:           []javaser.FieldDesc{{TypeCode: javaser.TypeDouble, Name: "value"}},
	}
	sleepObjectValueDescriptor = &javaser.ClassDesc{
		Name:             "sleep.engine.types.ObjectValue",
		SerialVersionUID: -5081985781831374967,
		Flags:            javaser.SCSerializable,
		Fields: []javaser.FieldDesc{
			{TypeCode: javaser.TypeObject, Name: "value", ClassName: javaObjectType},
		},
	}
	sleepListContainerDescriptor = &javaser.ClassDesc{
		Name:             "sleep.engine.types.ListContainer",
		SerialVersionUID: -3953378005425853106,
		Flags:            javaser.SCSerializable,
		Fields: []javaser.FieldDesc{
			{TypeCode: javaser.TypeObject, Name: "values", ClassName: javaListType},
		},
	}
	sleepMyLinkedListDescriptor = &javaser.ClassDesc{
		Name:             "sleep.engine.types.MyLinkedList",
		SerialVersionUID: -5628420710725558840,
		Flags:            javaser.SCSerializable | javaser.SCWriteMethod,
	}
	sleepHashContainerDescriptor = &javaser.ClassDesc{
		Name:             "sleep.engine.types.HashContainer",
		SerialVersionUID: -3142263564746093352,
		Flags:            javaser.SCSerializable,
		Fields: []javaser.FieldDesc{
			{TypeCode: javaser.TypeObject, Name: "values", ClassName: javaMapType},
		},
	}
	sleepOrderedHashContainerDescriptor = &javaser.ClassDesc{
		Name:             "sleep.engine.types.OrderedHashContainer",
		SerialVersionUID: 7596106700353511577,
		Flags:            javaser.SCSerializable,
		Fields:           []javaser.FieldDesc{{TypeCode: javaser.TypeBoolean, Name: "shouldClean"}},
		Super:            sleepHashContainerDescriptor,
	}
	javaHashMapDescriptor = &javaser.ClassDesc{
		Name:             "java.util.HashMap",
		SerialVersionUID: 362498820763181265,
		Flags:            javaser.SCSerializable | javaser.SCWriteMethod,
		Fields: []javaser.FieldDesc{
			{TypeCode: javaser.TypeFloat, Name: "loadFactor"},
			{TypeCode: javaser.TypeInt, Name: "threshold"},
		},
	}
	javaLinkedHashMapDescriptor = &javaser.ClassDesc{
		Name:             "java.util.LinkedHashMap",
		SerialVersionUID: 3801124242820219131,
		Flags:            javaser.SCSerializable,
		Fields:           []javaser.FieldDesc{{TypeCode: javaser.TypeBoolean, Name: "accessOrder"}},
		Super:            javaHashMapDescriptor,
	}
	sleepOrderedHashDescriptor = &javaser.ClassDesc{
		Name:             "sleep.engine.types.OrderedHashContainer$OrderedHash",
		SerialVersionUID: -8996339986379220353,
		Flags:            javaser.SCSerializable,
		Fields: []javaser.FieldDesc{
			{TypeCode: javaser.TypeObject, Name: "this$0", ClassName: sleepOrderedHashType},
		},
		Super: javaLinkedHashMapDescriptor,
	}
	javaNumberDescriptor = &javaser.ClassDesc{
		Name:             "java.lang.Number",
		SerialVersionUID: -8742448824652078965,
		Flags:            javaser.SCSerializable,
	}
	javaIntegerDescriptor = &javaser.ClassDesc{
		Name:             "java.lang.Integer",
		SerialVersionUID: 1360826667806852920,
		Flags:            javaser.SCSerializable,
		Fields:           []javaser.FieldDesc{{TypeCode: javaser.TypeInt, Name: "value"}},
		Super:            javaNumberDescriptor,
	}
	javaLongDescriptor = &javaser.ClassDesc{
		Name:             "java.lang.Long",
		SerialVersionUID: 4290774380558885855,
		Flags:            javaser.SCSerializable,
		Fields:           []javaser.FieldDesc{{TypeCode: javaser.TypeLong, Name: "value"}},
		Super:            javaNumberDescriptor,
	}
	javaDoubleDescriptor = &javaser.ClassDesc{
		Name:             "java.lang.Double",
		SerialVersionUID: -9172774392245257468,
		Flags:            javaser.SCSerializable,
		Fields:           []javaser.FieldDesc{{TypeCode: javaser.TypeDouble, Name: "value"}},
		Super:            javaNumberDescriptor,
	}
	javaBooleanDescriptor = &javaser.ClassDesc{
		Name:             "java.lang.Boolean",
		SerialVersionUID: -3665804199014368530,
		Flags:            javaser.SCSerializable,
		Fields:           []javaser.FieldDesc{{TypeCode: javaser.TypeBoolean, Name: "value"}},
	}
	javaByteDescriptor = &javaser.ClassDesc{
		Name:             "java.lang.Byte",
		SerialVersionUID: -7183698231559129828,
		Flags:            javaser.SCSerializable,
		Fields:           []javaser.FieldDesc{{TypeCode: javaser.TypeByte, Name: "value"}},
		Super:            javaNumberDescriptor,
	}
	javaCharacterDescriptor = &javaser.ClassDesc{
		Name:             "java.lang.Character",
		SerialVersionUID: 3786198910865385080,
		Flags:            javaser.SCSerializable,
		Fields:           []javaser.FieldDesc{{TypeCode: javaser.TypeChar, Name: "value"}},
	}
	javaShortDescriptor = &javaser.ClassDesc{
		Name:             "java.lang.Short",
		SerialVersionUID: 7515723908773894738,
		Flags:            javaser.SCSerializable,
		Fields:           []javaser.FieldDesc{{TypeCode: javaser.TypeShort, Name: "value"}},
		Super:            javaNumberDescriptor,
	}
	javaFloatDescriptor = &javaser.ClassDesc{
		Name:             "java.lang.Float",
		SerialVersionUID: -2671257302660747028,
		Flags:            javaser.SCSerializable,
		Fields:           []javaser.FieldDesc{{TypeCode: javaser.TypeFloat, Name: "value"}},
		Super:            javaNumberDescriptor,
	}
	javaStringClassDescriptor = &javaser.ClassDesc{
		Name:             "java.lang.String",
		SerialVersionUID: -6849794470754667710,
		Flags:            javaser.SCSerializable,
	}
)

type serializedSleepScalar struct {
	value Value
}

func (value *serializedSleepScalar) String() string {
	if value == nil {
		return ""
	}
	return value.value.String()
}

func (value *serializedSleepScalar) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if value == nil || invocation.Op != ObjectInvoke || len(invocation.Arguments) != 0 {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "toString", "stringValue":
		return String(value.String()), true, nil
	case "intValue":
		return Int(value.value.Int32()), true, nil
	case "longValue":
		return Long(value.value.Int64()), true, nil
	case "doubleValue":
		return Double(value.value.Float64()), true, nil
	}
	return Null(), false, nil
}

type serializedJavaObject struct {
	class string
	value Value
	graph javaser.Value
	text  string
}

func newSerializedJavaObject(class string, value Value, graph javaser.Value) *serializedJavaObject {
	serialized := &serializedJavaObject{class: class, value: value, graph: graph}
	if class == javaBooleanDescriptor.Name {
		if value.Truth() {
			serialized.text = "true"
		} else {
			serialized.text = "false"
		}
	}
	return serialized
}

func (value *serializedJavaObject) String() string {
	if value == nil {
		return ""
	}
	if value.text != "" {
		return value.text
	}
	return value.value.String()
}

func (value *serializedJavaObject) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if value == nil || invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	if value.class == javaStringClassDescriptor.Name {
		stringInvocation := invocation
		stringInvocation.Target = value.value
		if result, handled, err := portableString(stringInvocation); handled {
			return result, true, err
		}
	}
	if len(invocation.Arguments) != 0 {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "toString":
		return String(value.String()), true, nil
	case "intValue", "byteValue", "shortValue":
		return Int(value.value.Int32()), true, nil
	case "longValue":
		return Long(value.value.Int64()), true, nil
	case "floatValue", "doubleValue":
		return Double(value.value.Float64()), true, nil
	case "booleanValue":
		return Bool(value.value.Truth()), true, nil
	}
	return Null(), false, nil
}

func sleepSerializationClassData(descriptor *javaser.ClassDesc) (javaser.ClassDataLayout, error) {
	if descriptor == nil {
		return javaser.ClassDataAuto, errors.New("sleep serialization: nil class descriptor")
	}
	if layout, handled, err := sleepClosureClassData(descriptor); handled || err != nil {
		return layout, err
	}
	switch descriptor.Name {
	case sleepScalarDescriptor.Name:
		if err := validateSleepDescriptor(descriptor, sleepScalarDescriptor); err != nil {
			return javaser.ClassDataAuto, err
		}
		return javaser.ClassDataAnnotationOnly, nil
	case sleepMyLinkedListDescriptor.Name:
		if err := validateSleepDescriptor(descriptor, sleepMyLinkedListDescriptor); err != nil {
			return javaser.ClassDataAuto, err
		}
		return javaser.ClassDataAnnotationOnly, nil
	case javaHashMapDescriptor.Name:
		if err := validateSleepDescriptor(descriptor, javaHashMapDescriptor); err != nil {
			return javaser.ClassDataAuto, err
		}
		return javaser.ClassDataDefaultFieldsAndAnnotation, nil
	default:
		return javaser.ClassDataAuto, nil
	}
}

func validateSleepDescriptor(got, want *javaser.ClassDesc) error {
	if got == nil || want == nil {
		return errors.New("sleep serialization: missing class descriptor")
	}
	if got.IsProxy || got.Name != want.Name || got.SerialVersionUID != want.SerialVersionUID || got.Flags != want.Flags {
		return fmt.Errorf("sleep serialization: descriptor for %q does not match profile %s", got.Name, sleepSerializationProfileName)
	}
	if len(got.Annotation) != 0 || len(got.Fields) != len(want.Fields) {
		return fmt.Errorf("sleep serialization: descriptor schema for %q does not match profile %s", got.Name, sleepSerializationProfileName)
	}
	for index := range want.Fields {
		actual := got.Fields[index]
		expected := want.Fields[index]
		if actual.TypeCode != expected.TypeCode || actual.Name != expected.Name || javaFieldClassName(actual) != javaFieldClassName(expected) {
			return fmt.Errorf("sleep serialization: field schema for %q does not match profile %s", got.Name, sleepSerializationProfileName)
		}
	}
	if (got.Super == nil) != (want.Super == nil) {
		return fmt.Errorf("sleep serialization: superclass for %q does not match profile %s", got.Name, sleepSerializationProfileName)
	}
	if want.Super != nil {
		return validateSleepDescriptor(got.Super, want.Super)
	}
	return nil
}

func javaFieldClassName(field javaser.FieldDesc) string {
	if field.ClassName == nil {
		return ""
	}
	return field.ClassName.Value
}

type sleepSerializationEncoder struct {
	arrays          map[*Array]*javaser.Object
	hashes          map[*Hash]*javaser.Object
	closures        map[*scriptClosure]*javaser.Object
	scalarCells     map[*Cell]*javaser.Object
	legacyFunctions map[*bytecode.Function]*sleepLegacyEncodedBlock
}

func encodeSleepScalarStream(value Value) ([]byte, error) {
	state := &sleepSerializationEncoder{
		arrays: make(map[*Array]*javaser.Object),
		hashes: make(map[*Hash]*javaser.Object),
	}
	root, err := state.scalar(value)
	if err != nil {
		return nil, err
	}
	return encodeSleepJavaStream(root)
}

func encodeSleepRawStream(value Value) ([]byte, error) {
	state := &sleepSerializationEncoder{
		arrays: make(map[*Array]*javaser.Object),
		hashes: make(map[*Hash]*javaser.Object),
	}
	root, err := state.raw(value)
	if err != nil {
		return nil, err
	}
	return encodeSleepJavaStream(root)
}

func encodeSleepJavaStream(root javaser.Value) ([]byte, error) {
	var output bytes.Buffer
	encoder := javaser.NewEncoder(&output)
	if err := encoder.Encode(root); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func (state *sleepSerializationEncoder) scalar(value Value) (*javaser.Object, error) {
	root := &javaser.Object{Descriptor: sleepScalarDescriptor}
	if err := state.fillScalar(root, value); err != nil {
		return nil, err
	}
	return root, nil
}

func (state *sleepSerializationEncoder) scalarType(value Value) (javaser.Value, error) {
	switch value.Kind() {
	case KindInt:
		return javaObjectWithField(sleepIntValueDescriptor, "value", javaser.Int(value.Int32())), nil
	case KindLong:
		return javaObjectWithField(sleepLongValueDescriptor, "value", javaser.Long(value.Int64())), nil
	case KindDouble:
		return javaObjectWithField(sleepDoubleValueDescriptor, "value", javaser.Double(value.Float64())), nil
	case KindString:
		return javaObjectWithField(sleepStringValueDescriptor, "value", sleepJavaStringValue(value)), nil
	case KindObject:
		object, _ := value.Object()
		raw, err := state.rawObject(object)
		if err != nil {
			return nil, err
		}
		return javaObjectWithField(sleepObjectValueDescriptor, "value", raw), nil
	case KindFunction:
		callable, _ := value.Function()
		closure, ok := callable.(*scriptClosure)
		if !ok || closure == nil {
			return nil, &UnsupportedError{Operation: "serialization", Name: fmt.Sprintf("callable %T", callable)}
		}
		raw, err := state.closure(closure)
		if err != nil {
			return nil, err
		}
		return javaObjectWithField(sleepObjectValueDescriptor, "value", raw), nil
	default:
		return nil, fmt.Errorf("sleep serialization: %s is not a scalar type", value.Kind())
	}
}

func javaObjectWithField(descriptor *javaser.ClassDesc, name string, value javaser.Element) *javaser.Object {
	field := descriptor.Fields[0]
	if field.Name != name {
		panic("opfor: invalid serialization descriptor field")
	}
	return &javaser.Object{
		Descriptor: descriptor,
		Data: []javaser.ClassData{{
			Descriptor: descriptor,
			Fields:     []javaser.FieldValue{{Field: field, Value: value}},
		}},
	}
}

func (state *sleepSerializationEncoder) array(array *Array) (*javaser.Object, error) {
	if existing := state.arrays[array]; existing != nil {
		return existing, nil
	}
	container := &javaser.Object{Descriptor: sleepListContainerDescriptor}
	state.arrays[array] = container
	values := array.Values()
	list := &javaser.Object{Descriptor: sleepMyLinkedListDescriptor}
	annotation := []javaser.Content{javaIntBlock(len(values))}
	for _, value := range values {
		scalar, err := state.scalar(value)
		if err != nil {
			return nil, err
		}
		annotation = append(annotation, scalar)
	}
	list.Data = []javaser.ClassData{{Descriptor: sleepMyLinkedListDescriptor, Annotation: annotation}}
	container.Data = []javaser.ClassData{{
		Descriptor: sleepListContainerDescriptor,
		Fields: []javaser.FieldValue{{
			Field: sleepListContainerDescriptor.Fields[0], Value: list,
		}},
	}}
	return container, nil
}

type sleepHashEntry struct {
	key   Value
	value Value
}

func snapshotSleepHash(hash *Hash) ([]sleepHashEntry, bool, bool, bool) {
	if hash == nil {
		return nil, false, false, false
	}
	if hash.backend != nil {
		// ScalarHash serialization observes getData(). MapWrapper defines that
		// operation as a detached HashMap snapshot with null entries omitted.
		return snapshotSleepHash(hash.backend.dataSnapshot())
	}
	hash.mu.RLock()
	keys := hash.compatibleKeysLocked()
	entries := make([]sleepHashEntry, 0, len(keys))
	for _, key := range keys {
		if cell := hash.items[key]; cell != nil {
			entries = append(entries, sleepHashEntry{key: hash.keyValueLocked(key), value: cell.Get()})
		}
	}
	ordered := hash.ordered
	accessOrdered := hash.accessOrdered
	shouldClean := hash.shouldClean
	hash.mu.RUnlock()
	return entries, ordered, accessOrdered, shouldClean
}

func (state *sleepSerializationEncoder) hash(hash *Hash) (*javaser.Object, error) {
	if existing := state.hashes[hash]; existing != nil {
		return existing, nil
	}
	entries, ordered, accessOrdered, shouldClean := snapshotSleepHash(hash)
	if !ordered {
		container := &javaser.Object{Descriptor: sleepHashContainerDescriptor}
		state.hashes[hash] = container
		mapping := &javaser.Object{Descriptor: javaHashMapDescriptor}
		data, err := state.hashMapData(entries)
		if err != nil {
			return nil, err
		}
		mapping.Data = []javaser.ClassData{data}
		container.Data = []javaser.ClassData{{
			Descriptor: sleepHashContainerDescriptor,
			Fields: []javaser.FieldValue{{
				Field: sleepHashContainerDescriptor.Fields[0], Value: mapping,
			}},
		}}
		return container, nil
	}

	container := &javaser.Object{Descriptor: sleepOrderedHashContainerDescriptor}
	state.hashes[hash] = container
	mapping := &javaser.Object{Descriptor: sleepOrderedHashDescriptor}
	hashMapData, err := state.hashMapData(entries)
	if err != nil {
		return nil, err
	}
	mapping.Data = []javaser.ClassData{
		hashMapData,
		{
			Descriptor: javaLinkedHashMapDescriptor,
			Fields: []javaser.FieldValue{{
				Field: javaLinkedHashMapDescriptor.Fields[0], Value: javaser.Boolean(accessOrdered),
			}},
		},
		{
			Descriptor: sleepOrderedHashDescriptor,
			Fields: []javaser.FieldValue{{
				Field: sleepOrderedHashDescriptor.Fields[0], Value: container,
			}},
		},
	}
	container.Data = []javaser.ClassData{
		{
			Descriptor: sleepHashContainerDescriptor,
			Fields: []javaser.FieldValue{{
				Field: sleepHashContainerDescriptor.Fields[0], Value: mapping,
			}},
		},
		{
			Descriptor: sleepOrderedHashContainerDescriptor,
			Fields: []javaser.FieldValue{{
				Field: sleepOrderedHashContainerDescriptor.Fields[0], Value: javaser.Boolean(shouldClean),
			}},
		},
	}
	return container, nil
}

func (state *sleepSerializationEncoder) hashMapData(entries []sleepHashEntry) (javaser.ClassData, error) {
	capacity := sleepHashMapCapacity(len(entries))
	annotation := []javaser.Content{javaTwoIntBlock(capacity, len(entries))}
	for _, entry := range entries {
		scalar, err := state.scalar(entry.value)
		if err != nil {
			return javaser.ClassData{}, err
		}
		annotation = append(annotation, sleepJavaStringValue(entry.key), scalar)
	}
	return javaser.ClassData{
		Descriptor: javaHashMapDescriptor,
		Fields: []javaser.FieldValue{
			{Field: javaHashMapDescriptor.Fields[0], Value: javaser.Float(0.75)},
			{Field: javaHashMapDescriptor.Fields[1], Value: javaser.Int(capacity * 3 / 4)},
		},
		Annotation: annotation,
	}, nil
}

func sleepHashMapCapacity(size int) int32 {
	capacity := int32(16)
	for int64(size) > int64(capacity)*3/4 && capacity <= math.MaxInt32/2 {
		capacity *= 2
	}
	return capacity
}

func javaIntBlock(value int) *javaser.BlockData {
	var data [4]byte
	binary.BigEndian.PutUint32(data[:], uint32(value))
	return &javaser.BlockData{Data: data[:]}
}

func javaTwoIntBlock(first int32, second int) *javaser.BlockData {
	var data [8]byte
	binary.BigEndian.PutUint32(data[:4], uint32(first))
	binary.BigEndian.PutUint32(data[4:], uint32(second))
	return &javaser.BlockData{Data: data[:]}
}

func sleepJavaString(value string) *javaser.String {
	return sleepJavaStringValue(String(value))
}

func sleepJavaStringValue(value Value) *javaser.String {
	units := sleepStringUnits(value)
	return &javaser.String{Value: string(utf16.Decode(units)), UTF16: units}
}

func (state *sleepSerializationEncoder) raw(value Value) (javaser.Value, error) {
	switch value.Kind() {
	case KindNull:
		return javaser.NullValue, nil
	case KindString:
		return sleepJavaStringValue(value), nil
	case KindInt:
		return javaBoxedObject(javaIntegerDescriptor, javaser.Int(value.Int32())), nil
	case KindLong:
		return javaBoxedObject(javaLongDescriptor, javaser.Long(value.Int64())), nil
	case KindDouble:
		return javaBoxedObject(javaDoubleDescriptor, javaser.Double(value.Float64())), nil
	case KindArray:
		array, _ := value.Array()
		return state.array(array)
	case KindHash:
		hash, _ := value.Hash()
		return state.hash(hash)
	case KindObject:
		object, _ := value.Object()
		return state.rawObject(object)
	case KindFunction:
		callable, _ := value.Function()
		closure, ok := callable.(*scriptClosure)
		if !ok || closure == nil {
			return nil, &UnsupportedError{Operation: "serialization", Name: fmt.Sprintf("callable %T", callable)}
		}
		return state.closure(closure)
	default:
		return nil, fmt.Errorf("sleep serialization: unsupported raw kind %s", value.Kind())
	}
}

func (state *sleepSerializationEncoder) rawObject(object any) (javaser.Value, error) {
	switch value := object.(type) {
	case *serializedSleepScalar:
		if value == nil {
			return javaser.NullValue, nil
		}
		return state.scalar(value.value)
	case *serializedJavaObject:
		if value == nil {
			return javaser.NullValue, nil
		}
		if value.graph != nil {
			return value.graph, nil
		}
		return state.raw(value.value)
	case *portableJavaPrimitive:
		return sleepPortableJavaPrimitive(value)
	case string:
		return sleepJavaString(value), nil
	case int:
		return javaBoxedObject(javaIntegerDescriptor, javaser.Int(value)), nil
	case int32:
		return javaBoxedObject(javaIntegerDescriptor, javaser.Int(value)), nil
	case int64:
		return javaBoxedObject(javaLongDescriptor, javaser.Long(value)), nil
	case float64:
		return javaBoxedObject(javaDoubleDescriptor, javaser.Double(value)), nil
	case Value:
		return state.raw(value)
	case classReference:
		return sleepSerializedClass(string(value))
	case sleepClass:
		return sleepSerializedClass(string(value))
	default:
		return nil, &UnsupportedError{Operation: "serialization", Name: fmt.Sprintf("host object %T", object)}
	}
}

func sleepPortableJavaPrimitive(value *portableJavaPrimitive) (javaser.Value, error) {
	if value == nil {
		return javaser.NullValue, nil
	}
	scalar := value.sleepValue()
	switch value.className() {
	case javaBooleanDescriptor.Name:
		return javaObjectWithoutNumber(javaBooleanDescriptor, javaser.Boolean(scalar.Int32() != 0)), nil
	case javaByteDescriptor.Name:
		return javaBoxedObject(javaByteDescriptor, javaser.Byte(int8(scalar.Int32()))), nil
	case javaCharacterDescriptor.Name:
		units := sleepStringUnits(scalar)
		if len(units) == 0 {
			return nil, errors.New("sleep serialization: empty java.lang.Character")
		}
		return javaObjectWithoutNumber(javaCharacterDescriptor, javaser.Char(units[0])), nil
	case javaShortDescriptor.Name:
		return javaBoxedObject(javaShortDescriptor, javaser.Short(int16(scalar.Int32()))), nil
	case javaIntegerDescriptor.Name:
		return javaBoxedObject(javaIntegerDescriptor, javaser.Int(scalar.Int32())), nil
	case javaLongDescriptor.Name:
		return javaBoxedObject(javaLongDescriptor, javaser.Long(scalar.Int64())), nil
	case javaFloatDescriptor.Name:
		return javaBoxedObject(javaFloatDescriptor, javaser.Float(float32(scalar.Float64()))), nil
	case javaDoubleDescriptor.Name:
		return javaBoxedObject(javaDoubleDescriptor, javaser.Double(scalar.Float64())), nil
	default:
		return nil, &UnsupportedError{Operation: "serialization", Name: "Java primitive " + value.className()}
	}
}

func sleepSerializedClass(name string) (javaser.Value, error) {
	name = resolvePortableClassName(name)
	if name != javaStringClassDescriptor.Name {
		return nil, &UnsupportedError{Operation: "serialization", Name: "Java class " + name}
	}
	return &javaser.Class{Descriptor: javaStringClassDescriptor}, nil
}

func javaBoxedObject(descriptor *javaser.ClassDesc, value javaser.Element) *javaser.Object {
	return &javaser.Object{
		Descriptor: descriptor,
		Data: []javaser.ClassData{
			{Descriptor: javaNumberDescriptor},
			{
				Descriptor: descriptor,
				Fields: []javaser.FieldValue{{
					Field: descriptor.Fields[0], Value: value,
				}},
			},
		},
	}
}

func javaObjectWithoutNumber(descriptor *javaser.ClassDesc, value javaser.Element) *javaser.Object {
	return &javaser.Object{
		Descriptor: descriptor,
		Data: []javaser.ClassData{{
			Descriptor: descriptor,
			Fields: []javaser.FieldValue{{
				Field: descriptor.Fields[0], Value: value,
			}},
		}},
	}
}

type sleepSerializationDecoder struct {
	containers      map[*javaser.Object]Value
	closures        map[*javaser.Object]*scriptClosure
	scalarCells     map[*javaser.Object]*Cell
	legacyFunctions map[*javaser.Object]*sleepLegacyDecodedBlock
	script          *Script
}

func decodeSleepScalarStream(reader io.Reader) (Value, int64, error) {
	return decodeSleepScalarStreamForScript(reader, nil)
}

func decodeSleepScalarStreamForScript(reader io.Reader, script *Script) (Value, int64, error) {
	root, consumed, err := decodeSleepJavaStream(reader)
	if err != nil {
		return Null(), consumed, err
	}
	object, ok := root.(*javaser.Object)
	if !ok || object.Descriptor == nil || object.Descriptor.Name != sleepScalarDescriptor.Name {
		return Null(), consumed, fmt.Errorf("java.lang.ClassCastException: serialized root is not sleep.runtime.Scalar")
	}
	state := newSleepSerializationDecoder(script)
	value, err := state.scalar(object)
	return value, consumed, err
}

func decodeSleepRawStream(reader io.Reader) (Value, int64, error) {
	return decodeSleepRawStreamForScript(reader, nil)
}

func decodeSleepRawStreamForScript(reader io.Reader, script *Script) (Value, int64, error) {
	root, consumed, err := decodeSleepJavaStream(reader)
	if err != nil {
		return Null(), consumed, err
	}
	if _, ok := root.(*javaser.Null); ok {
		return Null(), consumed, nil
	}
	state := newSleepSerializationDecoder(script)
	if scalar, ok := root.(*javaser.Object); ok && scalar.Descriptor != nil && scalar.Descriptor.Name == sleepScalarDescriptor.Name {
		value, decodeErr := state.scalar(scalar)
		if decodeErr != nil {
			return Null(), consumed, decodeErr
		}
		return ObjectValue(&serializedSleepScalar{value: value}), consumed, nil
	}
	value, class, err := state.raw(root)
	if err != nil {
		return Null(), consumed, err
	}
	if class == sleepClosureDescriptor.Name {
		return value, consumed, nil
	}
	return ObjectValue(newSerializedJavaObject(class, value, root)), consumed, nil
}

func newSleepSerializationDecoder(script *Script) *sleepSerializationDecoder {
	return &sleepSerializationDecoder{
		containers:      make(map[*javaser.Object]Value),
		closures:        make(map[*javaser.Object]*scriptClosure),
		scalarCells:     make(map[*javaser.Object]*Cell),
		legacyFunctions: make(map[*javaser.Object]*sleepLegacyDecodedBlock),
		script:          script,
	}
}

func decodeSleepJavaStream(reader io.Reader) (javaser.Value, int64, error) {
	decoder := javaser.NewDecoder(reader, javaser.WithClassDataResolver(sleepSerializationClassData))
	root, err := decoder.Decode()
	return root, decoder.BytesRead(), err
}

func (state *sleepSerializationDecoder) scalar(object *javaser.Object) (Value, error) {
	if err := validateSleepDescriptor(object.Descriptor, sleepScalarDescriptor); err != nil {
		return Null(), err
	}
	data, ok := object.DataFor(sleepScalarDescriptor.Name)
	if !ok || len(data.Fields) != 0 || len(data.Annotation) != 3 {
		return Null(), errors.New("sleep serialization: invalid Scalar custom data")
	}
	actual, ok := data.Annotation[0].(javaser.Value)
	if !ok {
		return Null(), errors.New("sleep serialization: invalid Scalar value reference")
	}
	array, ok := data.Annotation[1].(javaser.Value)
	if !ok {
		return Null(), errors.New("sleep serialization: invalid Scalar array reference")
	}
	hash, ok := data.Annotation[2].(javaser.Value)
	if !ok {
		return Null(), errors.New("sleep serialization: invalid Scalar hash reference")
	}
	if !javaSerializationNull(actual) {
		return state.scalarType(actual)
	}
	if !javaSerializationNull(array) {
		container, ok := array.(*javaser.Object)
		if !ok {
			return Null(), errors.New("sleep serialization: Scalar array is not an object")
		}
		return state.array(container)
	}
	if !javaSerializationNull(hash) {
		container, ok := hash.(*javaser.Object)
		if !ok {
			return Null(), errors.New("sleep serialization: Scalar hash is not an object")
		}
		return state.hash(container)
	}
	return Null(), nil
}

func javaSerializationNull(value javaser.Value) bool {
	_, ok := value.(*javaser.Null)
	return ok
}

func (state *sleepSerializationDecoder) scalarType(value javaser.Value) (Value, error) {
	object, ok := value.(*javaser.Object)
	if !ok || object.Descriptor == nil {
		return Null(), errors.New("sleep serialization: Scalar type is not an object")
	}
	switch object.Descriptor.Name {
	case sleepStringValueDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepStringValueDescriptor); err != nil {
			return Null(), err
		}
		field, err := sleepObjectField(object, sleepStringValueDescriptor.Name, "value")
		if err != nil {
			return Null(), err
		}
		text, ok := field.(*javaser.String)
		if !ok {
			return Null(), errors.New("sleep serialization: StringValue.value is not a string")
		}
		return sleepValueFromJavaString(text), nil
	case sleepIntValueDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepIntValueDescriptor); err != nil {
			return Null(), err
		}
		field, err := sleepObjectField(object, sleepIntValueDescriptor.Name, "value")
		if err != nil {
			return Null(), err
		}
		integer, ok := field.(javaser.Int)
		if !ok {
			return Null(), errors.New("sleep serialization: IntValue.value is not an int")
		}
		return Int(int32(integer)), nil
	case sleepLongValueDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepLongValueDescriptor); err != nil {
			return Null(), err
		}
		field, err := sleepObjectField(object, sleepLongValueDescriptor.Name, "value")
		if err != nil {
			return Null(), err
		}
		integer, ok := field.(javaser.Long)
		if !ok {
			return Null(), errors.New("sleep serialization: LongValue.value is not a long")
		}
		return Long(int64(integer)), nil
	case sleepDoubleValueDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepDoubleValueDescriptor); err != nil {
			return Null(), err
		}
		field, err := sleepObjectField(object, sleepDoubleValueDescriptor.Name, "value")
		if err != nil {
			return Null(), err
		}
		number, ok := field.(javaser.Double)
		if !ok {
			return Null(), errors.New("sleep serialization: DoubleValue.value is not a double")
		}
		return Double(float64(number)), nil
	case sleepObjectValueDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepObjectValueDescriptor); err != nil {
			return Null(), err
		}
		field, err := sleepObjectField(object, sleepObjectValueDescriptor.Name, "value")
		if err != nil {
			return Null(), err
		}
		raw, ok := field.(javaser.Value)
		if !ok {
			return Null(), errors.New("sleep serialization: ObjectValue.value is not a reference")
		}
		if javaSerializationNull(raw) {
			return Null(), nil
		}
		decoded, class, err := state.raw(raw)
		if err != nil {
			return Null(), err
		}
		if class == sleepClosureDescriptor.Name {
			return decoded, nil
		}
		return ObjectValue(newSerializedJavaObject(class, decoded, raw)), nil
	default:
		return Null(), fmt.Errorf("sleep serialization: unsupported ScalarType %q", object.Descriptor.Name)
	}
}

func sleepObjectField(object *javaser.Object, className, fieldName string) (javaser.Element, error) {
	data, ok := object.DataFor(className)
	if !ok {
		return nil, fmt.Errorf("sleep serialization: missing class data for %q", className)
	}
	value, ok := data.Field(fieldName)
	if !ok {
		return nil, fmt.Errorf("sleep serialization: missing field %s.%s", className, fieldName)
	}
	return value, nil
}

func (state *sleepSerializationDecoder) array(container *javaser.Object) (Value, error) {
	if existing, ok := state.containers[container]; ok {
		return existing, nil
	}
	if err := validateSleepDescriptor(container.Descriptor, sleepListContainerDescriptor); err != nil {
		return Null(), err
	}
	array := NewArray()
	result := ArrayValue(array)
	state.containers[container] = result
	field, err := sleepObjectField(container, sleepListContainerDescriptor.Name, "values")
	if err != nil {
		return Null(), err
	}
	list, ok := field.(*javaser.Object)
	if !ok {
		return Null(), errors.New("sleep serialization: ListContainer.values is not an object")
	}
	if err := validateSleepDescriptor(list.Descriptor, sleepMyLinkedListDescriptor); err != nil {
		return Null(), err
	}
	data, ok := list.DataFor(sleepMyLinkedListDescriptor.Name)
	if !ok || len(data.Fields) != 0 || len(data.Annotation) == 0 {
		return Null(), errors.New("sleep serialization: invalid MyLinkedList custom data")
	}
	count, values, err := sleepAnnotationCount(data.Annotation)
	if err != nil {
		return Null(), err
	}
	if count != len(values) {
		return Null(), fmt.Errorf("sleep serialization: MyLinkedList count %d does not match %d values", count, len(values))
	}
	decoded := make([]Value, 0, len(values))
	for _, encoded := range values {
		scalar, ok := encoded.(*javaser.Object)
		if !ok {
			return Null(), errors.New("sleep serialization: MyLinkedList element is not a Scalar")
		}
		value, decodeErr := state.scalar(scalar)
		if decodeErr != nil {
			return Null(), decodeErr
		}
		decoded = append(decoded, value)
	}
	var runtime *Runtime
	if state.script != nil {
		runtime = state.script.runtime
	}
	if err := reserveCollectionEntries(runtime, len(decoded)); err != nil {
		return Null(), err
	}
	for _, value := range decoded {
		array.Append(value)
	}
	return result, nil
}

func sleepAnnotationCount(annotation []javaser.Content) (int, []javaser.Value, error) {
	if len(annotation) == 0 {
		return 0, nil, errors.New("sleep serialization: missing custom block data")
	}
	block, ok := annotation[0].(*javaser.BlockData)
	if !ok || len(block.Data) != 4 {
		return 0, nil, errors.New("sleep serialization: invalid custom element count")
	}
	count := int(int32(binary.BigEndian.Uint32(block.Data)))
	if count < 0 {
		return 0, nil, errors.New("sleep serialization: negative custom element count")
	}
	values := make([]javaser.Value, 0, len(annotation)-1)
	for _, content := range annotation[1:] {
		value, ok := content.(javaser.Value)
		if !ok {
			return 0, nil, errors.New("sleep serialization: invalid custom object content")
		}
		values = append(values, value)
	}
	return count, values, nil
}

func (state *sleepSerializationDecoder) hash(container *javaser.Object) (Value, error) {
	if existing, ok := state.containers[container]; ok {
		return existing, nil
	}
	ordered := container.Descriptor != nil && container.Descriptor.Name == sleepOrderedHashContainerDescriptor.Name
	if ordered {
		if err := validateSleepDescriptor(container.Descriptor, sleepOrderedHashContainerDescriptor); err != nil {
			return Null(), err
		}
	} else if err := validateSleepDescriptor(container.Descriptor, sleepHashContainerDescriptor); err != nil {
		return Null(), err
	}
	hash := NewHash()
	if ordered {
		hash = NewOrderedHash()
	}
	result := HashValue(hash)
	state.containers[container] = result
	field, err := sleepObjectField(container, sleepHashContainerDescriptor.Name, "values")
	if err != nil {
		return Null(), err
	}
	mapping, ok := field.(*javaser.Object)
	if !ok || mapping.Descriptor == nil {
		return Null(), errors.New("sleep serialization: HashContainer.values is not an object")
	}
	if ordered {
		if err := validateSleepDescriptor(mapping.Descriptor, sleepOrderedHashDescriptor); err != nil {
			return Null(), err
		}
		access, accessErr := sleepObjectField(mapping, javaLinkedHashMapDescriptor.Name, "accessOrder")
		if accessErr != nil {
			return Null(), accessErr
		}
		accessOrdered, ok := access.(javaser.Boolean)
		if !ok {
			return Null(), errors.New("sleep serialization: LinkedHashMap.accessOrder is not boolean")
		}
		hash.accessOrdered = bool(accessOrdered)
		outer, outerErr := sleepObjectField(mapping, sleepOrderedHashDescriptor.Name, "this$0")
		if outerErr != nil {
			return Null(), outerErr
		}
		if outer != container {
			return Null(), errors.New("sleep serialization: ordered hash outer reference is invalid")
		}
		clean, cleanErr := sleepObjectField(container, sleepOrderedHashContainerDescriptor.Name, "shouldClean")
		if cleanErr != nil {
			return Null(), cleanErr
		}
		shouldClean, ok := clean.(javaser.Boolean)
		if !ok {
			return Null(), errors.New("sleep serialization: OrderedHashContainer.shouldClean is not boolean")
		}
		hash.shouldClean = bool(shouldClean)
	} else if err := validateSleepDescriptor(mapping.Descriptor, javaHashMapDescriptor); err != nil {
		return Null(), err
	}
	entries, err := state.hashEntries(mapping)
	if err != nil {
		return Null(), err
	}
	var runtime *Runtime
	if state.script != nil {
		runtime = state.script.runtime
	}
	if err := reserveCollectionEntries(runtime, len(entries)); err != nil {
		return Null(), err
	}
	for _, entry := range entries {
		hash.SetValue(entry.key, entry.value)
	}
	return result, nil
}

func (state *sleepSerializationDecoder) hashEntries(mapping *javaser.Object) ([]sleepHashEntry, error) {
	data, ok := mapping.DataFor(javaHashMapDescriptor.Name)
	if !ok || len(data.Annotation) == 0 {
		return nil, errors.New("sleep serialization: invalid HashMap custom data")
	}
	block, ok := data.Annotation[0].(*javaser.BlockData)
	if !ok || len(block.Data) != 8 {
		return nil, errors.New("sleep serialization: invalid HashMap capacity/count data")
	}
	count := int(int32(binary.BigEndian.Uint32(block.Data[4:])))
	if count < 0 || len(data.Annotation) != 1+count*2 {
		return nil, fmt.Errorf("sleep serialization: invalid HashMap entry count %d", count)
	}
	entries := make([]sleepHashEntry, 0, count)
	for index := 0; index < count; index++ {
		key, ok := data.Annotation[1+index*2].(*javaser.String)
		if !ok {
			return nil, errors.New("sleep serialization: HashMap key is not a string")
		}
		scalar, ok := data.Annotation[2+index*2].(*javaser.Object)
		if !ok {
			return nil, errors.New("sleep serialization: HashMap value is not a Scalar")
		}
		value, err := state.scalar(scalar)
		if err != nil {
			return nil, err
		}
		entries = append(entries, sleepHashEntry{key: sleepValueFromJavaString(key), value: value})
	}
	return entries, nil
}

func (state *sleepSerializationDecoder) raw(root javaser.Value) (Value, string, error) {
	switch value := root.(type) {
	case *javaser.Null:
		return Null(), "", nil
	case *javaser.String:
		return sleepValueFromJavaString(value), "java.lang.String", nil
	case *javaser.Object:
		if value.Descriptor == nil {
			return Null(), "", errors.New("sleep serialization: raw object has no descriptor")
		}
		switch value.Descriptor.Name {
		case javaIntegerDescriptor.Name:
			if err := validateSleepDescriptor(value.Descriptor, javaIntegerDescriptor); err != nil {
				return Null(), "", err
			}
			field, err := sleepObjectField(value, javaIntegerDescriptor.Name, "value")
			if err != nil {
				return Null(), "", err
			}
			integer, ok := field.(javaser.Int)
			if !ok {
				return Null(), "", errors.New("sleep serialization: Integer.value is not an int")
			}
			return Int(int32(integer)), javaIntegerDescriptor.Name, nil
		case javaLongDescriptor.Name:
			if err := validateSleepDescriptor(value.Descriptor, javaLongDescriptor); err != nil {
				return Null(), "", err
			}
			field, err := sleepObjectField(value, javaLongDescriptor.Name, "value")
			if err != nil {
				return Null(), "", err
			}
			integer, ok := field.(javaser.Long)
			if !ok {
				return Null(), "", errors.New("sleep serialization: Long.value is not a long")
			}
			return Long(int64(integer)), javaLongDescriptor.Name, nil
		case javaDoubleDescriptor.Name:
			if err := validateSleepDescriptor(value.Descriptor, javaDoubleDescriptor); err != nil {
				return Null(), "", err
			}
			field, err := sleepObjectField(value, javaDoubleDescriptor.Name, "value")
			if err != nil {
				return Null(), "", err
			}
			number, ok := field.(javaser.Double)
			if !ok {
				return Null(), "", errors.New("sleep serialization: Double.value is not a double")
			}
			return Double(float64(number)), javaDoubleDescriptor.Name, nil
		case javaBooleanDescriptor.Name:
			if err := validateSleepDescriptor(value.Descriptor, javaBooleanDescriptor); err != nil {
				return Null(), "", err
			}
			field, err := sleepObjectField(value, javaBooleanDescriptor.Name, "value")
			if err != nil {
				return Null(), "", err
			}
			boolean, ok := field.(javaser.Boolean)
			if !ok {
				return Null(), "", errors.New("sleep serialization: Boolean.value is not boolean")
			}
			return Bool(bool(boolean)), javaBooleanDescriptor.Name, nil
		case sleepListContainerDescriptor.Name:
			array, err := state.array(value)
			return array, sleepListContainerDescriptor.Name, err
		case sleepHashContainerDescriptor.Name, sleepOrderedHashContainerDescriptor.Name:
			hash, err := state.hash(value)
			return hash, value.Descriptor.Name, err
		case sleepClosureDescriptor.Name:
			closure, err := state.closure(value)
			return closure, sleepClosureDescriptor.Name, err
		default:
			return Null(), "", fmt.Errorf("sleep serialization: unsupported raw Java class %q", value.Descriptor.Name)
		}
	case *javaser.Class:
		if err := validateSleepDescriptor(value.Descriptor, javaStringClassDescriptor); err != nil {
			return Null(), "", err
		}
		return ObjectValue(classReference(value.Descriptor.Name)), "java.lang.Class", nil
	default:
		return Null(), "", fmt.Errorf("sleep serialization: unsupported raw Java value %T", root)
	}
}

// sleepStringFromJava restores the exact UTF-16 units carried by Java's String
// wire type. That type does not record whether a U+0000..U+00FF unit originated
// as text or as a byte carrier, so OPFOR restores that range with raw-byte
// provenance for reversible binary round trips. Equality and hashing still
// observe only the UTF-16 units.
func sleepStringFromJava(value *javaser.String) string {
	return sleepValueFromJavaString(value).String()
}

func sleepValueFromJavaString(value *javaser.String) Value {
	if value == nil {
		return String("")
	}
	units := value.UTF16
	if units == nil {
		units = utf16.Encode([]rune(value.Value))
	}
	raw := make([]bool, len(units))
	for index, unit := range units {
		// Java serialization does not retain text-vs-byte origin. Preserve
		// OPFOR's established reversible choice for byte-sized code units.
		raw[index] = unit <= 0xff
	}
	return sleepStringValueFromUnits(units, raw)
}

type sleepHandleReader struct {
	handle *sleepIOHandle
	ctx    context.Context
}

func (reader sleepHandleReader) Read(data []byte) (int, error) {
	return reader.handle.readBinaryLockedContext(reader.ctx, data)
}

func (handle *sleepIOHandle) readSleepScalar() (Value, error) {
	return handle.readSleepScalarForScript(nil)
}

func (handle *sleepIOHandle) readSleepScalarForScript(script *Script) (Value, error) {
	return handle.readSleepScalarForScriptContext(context.Background(), script)
}

func (handle *sleepIOHandle) readSleepScalarForScriptContext(ctx context.Context, script *Script) (Value, error) {
	if handle == nil {
		return Null(), io.ErrClosedPipe
	}
	handle.readMu.Lock()
	defer handle.readMu.Unlock()
	returnValue, _, err := decodeSleepScalarStreamForScript(sleepHandleReader{handle: handle, ctx: ctx}, script)
	return returnValue, err
}

func (handle *sleepIOHandle) readSleepRaw() (Value, error) {
	return handle.readSleepRawForScript(nil)
}

func (handle *sleepIOHandle) readSleepRawForScript(script *Script) (Value, error) {
	return handle.readSleepRawForScriptContext(context.Background(), script)
}

func (handle *sleepIOHandle) readSleepRawForScriptContext(ctx context.Context, script *Script) (Value, error) {
	if handle == nil {
		return Null(), io.ErrClosedPipe
	}
	handle.readMu.Lock()
	defer handle.readMu.Unlock()
	returnValue, _, err := decodeSleepRawStreamForScript(sleepHandleReader{handle: handle, ctx: ctx}, script)
	return returnValue, err
}

func (handle *sleepIOHandle) writeSleepBytes(data []byte) error {
	if handle == nil {
		return io.ErrClosedPipe
	}
	written, err := handle.Write(data)
	if err == nil && written != len(data) {
		err = io.ErrShortWrite
	}
	return err
}

func (state *ioBuiltinState) writeObject(ctx context.Context, invocation Invocation) (Value, error) {
	return state.writeSerialized(ctx, invocation, false)
}

func (state *ioBuiltinState) writeAsObject(ctx context.Context, invocation Invocation) (Value, error) {
	return state.writeSerialized(ctx, invocation, true)
}

func (state *ioBuiltinState) writeSerialized(ctx context.Context, invocation Invocation, raw bool) (Value, error) {
	handle, valueIndex, err := state.chooseHandle(invocation, 2)
	if err != nil {
		return Null(), err
	}
	for index := valueIndex; index < len(invocation.Arguments); index++ {
		var data []byte
		if raw {
			data, err = encodeSleepRawStream(invocation.Arg(index))
		} else {
			data, err = encodeSleepScalarStream(invocation.Arg(index))
		}
		if err == nil {
			err = handle.writeSleepBytes(data)
		}
		if err != nil {
			return state.flagSerializationError(ctx, invocation, handle, err)
		}
	}
	return Null(), nil
}

func (state *ioBuiltinState) readObject(ctx context.Context, invocation Invocation) (Value, error) {
	return state.readSerialized(ctx, invocation, false)
}

func (state *ioBuiltinState) readAsObject(ctx context.Context, invocation Invocation) (Value, error) {
	return state.readSerialized(ctx, invocation, true)
}

func (state *ioBuiltinState) readSerialized(ctx context.Context, invocation Invocation, raw bool) (Value, error) {
	handle, _, err := state.chooseHandle(invocation, 1)
	if err != nil {
		return Null(), err
	}
	var value Value
	script := state.serializationScript(ctx, invocation)
	if raw {
		value, err = handle.readSleepRawForScriptContext(ctx, script)
	} else {
		value, err = handle.readSleepScalarForScriptContext(ctx, script)
	}
	if errors.Is(err, io.EOF) {
		closeErr := handle.close()
		if closeErr != nil {
			return state.flagSerializationError(ctx, invocation, handle, closeErr)
		}
		return Null(), nil
	}
	if err != nil {
		if errors.Is(err, ErrResourceLimit) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Null(), err
		}
		return state.flagSerializationError(ctx, invocation, handle, err)
	}
	return value, nil
}

func (state *ioBuiltinState) serializationScript(ctx context.Context, invocation Invocation) *Script {
	return serializationReceivingScript(ctx, invocation)
}

func serializationReceivingScript(ctx context.Context, invocation Invocation) *Script {
	if fiber := currentFiber(ctx); fiber != nil && fiber.closure != nil && fiber.closure.script != nil {
		return fiber.closure.script
	}
	if invocation.Runtime == nil || invocation.Script == 0 {
		return nil
	}
	invocation.Runtime.mu.RLock()
	script := invocation.Runtime.scripts[invocation.Script]
	invocation.Runtime.mu.RUnlock()
	return script
}

func (state *ioBuiltinState) flagSerializationError(ctx context.Context, invocation Invocation, handle *sleepIOHandle, err error) (Value, error) {
	closeErr := handle.close()
	if closeErr != nil {
		err = errors.Join(err, closeErr)
	}
	if errors.Is(err, ErrResourceLimit) {
		return Null(), err
	}
	wrapped := preserveNativeBoundaryError(ctx, fmt.Errorf("java.io.IOException: %w", err))
	if state == nil || state.runtime == nil {
		return Null(), wrapped
	}
	return state.runtime.flagSourceError(invocation, wrapped)
}
