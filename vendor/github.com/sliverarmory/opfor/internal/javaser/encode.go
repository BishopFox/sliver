package javaser

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

const normalizedBlockSize = 1024

// Encoder writes independent version 5 Java Object Serialization streams.
// Calling Encode repeatedly writes one complete AC ED 00 05-headed stream per
// call, as required by Sleep's writeObject behavior.
type Encoder struct {
	w   io.Writer
	cfg encoderConfig

	totalWritten int64
	startWritten int64
	lastWritten  int64

	handles         map[Content]int32
	classCount      int
	fieldCount      int
	annotationCount int
}

// NewEncoder constructs a bounded encoder over w.
func NewEncoder(w io.Writer, options ...Option) *Encoder {
	cfg := encoderConfig{limits: DefaultLimits()}
	for _, option := range options {
		if option != nil {
			option.applyEncoder(&cfg)
		}
	}
	cfg.limits = cfg.limits.normalized()
	return &Encoder{w: w, cfg: cfg}
}

// Encode writes value as one complete independent stream.
func (e *Encoder) Encode(value Value) (err error) {
	if e == nil || e.w == nil {
		return fmt.Errorf("java serialization: nil encoder writer")
	}
	e.startWritten = e.totalWritten
	e.lastWritten = 0
	e.handles = make(map[Content]int32)
	e.classCount = 0
	e.fieldCount = 0
	e.annotationCount = 0
	defer func() { e.lastWritten = e.totalWritten - e.startWritten }()

	var header [4]byte
	binary.BigEndian.PutUint16(header[:2], StreamMagic)
	binary.BigEndian.PutUint16(header[2:], StreamVersion)
	if err := e.writeFull(header[:]); err != nil {
		return err
	}
	return e.writeValue(value, 1)
}

// BytesWritten reports bytes emitted by the most recent Encode call,
// including the four-byte stream header.
func (e *Encoder) BytesWritten() int64 { return e.lastWritten }

// Encode writes value as one complete independent stream using a fresh
// encoder.
func Encode(w io.Writer, value Value, options ...Option) error {
	return NewEncoder(w, options...).Encode(value)
}

func (e *Encoder) writeContent(content Content, depth int) error {
	if err := e.checkDepth(depth); err != nil {
		return err
	}
	switch value := content.(type) {
	case nil:
		return e.writeByte(TCNull)
	case *Null:
		return e.writeByte(TCNull)
	case *String:
		return e.writeString(value)
	case *Object:
		return e.writeObject(value, depth)
	case *Array:
		return e.writeArray(value, depth)
	case *Class:
		return e.writeClass(value, depth)
	case *ClassDesc:
		return e.writeClassDesc(value, depth)
	case *BlockData:
		return e.writeBlockData(value)
	default:
		return fmt.Errorf("java serialization: unsupported content type %T", content)
	}
}

func (e *Encoder) writeValue(value Value, depth int) error {
	if isNilValue(value) {
		return e.writeByte(TCNull)
	}
	return e.writeContent(value, depth)
}

func isNilValue(value Value) bool {
	if value == nil {
		return true
	}
	switch typed := value.(type) {
	case *Null:
		return true
	case *String:
		return typed == nil
	case *Object:
		return typed == nil
	case *Array:
		return typed == nil
	case *Class:
		return typed == nil
	default:
		return false
	}
}

func (e *Encoder) writeReferenceIfKnown(content Content) (bool, error) {
	if handle, ok := e.handles[content]; ok {
		if err := e.writeByte(TCReference); err != nil {
			return true, err
		}
		return true, e.writeInt32(handle)
	}
	return false, nil
}

func (e *Encoder) addHandle(content Content) error {
	if len(e.handles) >= e.cfg.limits.MaxHandles {
		return &LimitError{Resource: "handle count", Limit: int64(e.cfg.limits.MaxHandles)}
	}
	e.handles[content] = BaseWireHandle + int32(len(e.handles))
	return nil
}

func (e *Encoder) writeString(value *String) error {
	if value == nil {
		return e.writeByte(TCNull)
	}
	if known, err := e.writeReferenceIfKnown(value); known || err != nil {
		return err
	}
	encoded := encodeModifiedUTF8(value.Value, value.UTF16)
	if int64(len(encoded)) > e.cfg.limits.MaxStringBytes {
		return &LimitError{Resource: "string bytes", Limit: e.cfg.limits.MaxStringBytes}
	}
	if len(encoded) <= math.MaxUint16 {
		if err := e.writeByte(TCString); err != nil {
			return err
		}
		if err := e.addHandle(value); err != nil {
			return err
		}
		if err := e.writeUint16(uint16(len(encoded))); err != nil {
			return err
		}
	} else {
		if err := e.writeByte(TCLongString); err != nil {
			return err
		}
		if err := e.addHandle(value); err != nil {
			return err
		}
		if err := e.writeInt64(int64(len(encoded))); err != nil {
			return err
		}
	}
	return e.writeFull(encoded)
}

func (e *Encoder) writeClassDesc(desc *ClassDesc, depth int) error {
	if err := e.checkDepth(depth); err != nil {
		return err
	}
	if desc == nil {
		return e.writeByte(TCNull)
	}
	if known, err := e.writeReferenceIfKnown(desc); known || err != nil {
		return err
	}
	if e.classCount >= e.cfg.limits.MaxClassDescriptors {
		return &LimitError{Resource: "class descriptor count", Limit: int64(e.cfg.limits.MaxClassDescriptors)}
	}
	e.classCount++
	if desc.IsProxy {
		return e.writeProxyClassDesc(desc, depth)
	}
	if desc.Flags & ^byte(SCWriteMethod|SCSerializable|SCExternalizable|SCBlockData|SCEnum) != 0 {
		return fmt.Errorf("java serialization: class %q has unknown descriptor flags 0x%02x", desc.Name, desc.Flags)
	}
	if len(desc.Fields) > e.cfg.limits.MaxFieldsPerClass {
		return &LimitError{Resource: "fields per class", Limit: int64(e.cfg.limits.MaxFieldsPerClass)}
	}
	if e.fieldCount > e.cfg.limits.MaxTotalFields-len(desc.Fields) {
		return &LimitError{Resource: "total field count", Limit: int64(e.cfg.limits.MaxTotalFields)}
	}
	e.fieldCount += len(desc.Fields)
	if err := e.writeByte(TCClassDesc); err != nil {
		return err
	}
	if err := e.writeUTF(desc.Name); err != nil {
		return fmt.Errorf("java serialization: class name: %w", err)
	}
	if err := e.writeInt64(desc.SerialVersionUID); err != nil {
		return err
	}
	if err := e.addHandle(desc); err != nil {
		return err
	}
	if err := e.writeByte(desc.Flags); err != nil {
		return err
	}
	if err := e.writeUint16(uint16(len(desc.Fields))); err != nil {
		return err
	}
	for index, field := range desc.Fields {
		if !validFieldType(field.TypeCode) {
			return fmt.Errorf("java serialization: class %q field %d has invalid type code 0x%02x", desc.Name, index, field.TypeCode)
		}
		if err := e.writeByte(field.TypeCode); err != nil {
			return err
		}
		if err := e.writeUTF(field.Name); err != nil {
			return fmt.Errorf("java serialization: class %q field name: %w", desc.Name, err)
		}
		if field.TypeCode == TypeArray || field.TypeCode == TypeObject {
			if field.ClassName == nil {
				return fmt.Errorf("java serialization: class %q field %q has no type descriptor string", desc.Name, field.Name)
			}
			if err := e.writeString(field.ClassName); err != nil {
				return err
			}
		}
	}
	if err := e.writeAnnotation(desc.Annotation, depth+1); err != nil {
		return err
	}
	return e.writeClassDesc(desc.Super, depth+1)
}

func (e *Encoder) writeProxyClassDesc(desc *ClassDesc, depth int) error {
	if len(desc.ProxyInterfaces) > e.cfg.limits.MaxProxyInterfaces {
		return &LimitError{Resource: "proxy interface count", Limit: int64(e.cfg.limits.MaxProxyInterfaces)}
	}
	if err := e.writeByte(TCProxyClassDesc); err != nil {
		return err
	}
	if err := e.addHandle(desc); err != nil {
		return err
	}
	if err := e.writeInt32(int32(len(desc.ProxyInterfaces))); err != nil {
		return err
	}
	for _, name := range desc.ProxyInterfaces {
		if err := e.writeUTF(name); err != nil {
			return fmt.Errorf("java serialization: proxy interface name: %w", err)
		}
	}
	if err := e.writeAnnotation(desc.Annotation, depth+1); err != nil {
		return err
	}
	return e.writeClassDesc(desc.Super, depth+1)
}

func (e *Encoder) writeObject(object *Object, depth int) error {
	if object == nil {
		return e.writeByte(TCNull)
	}
	if known, err := e.writeReferenceIfKnown(object); known || err != nil {
		return err
	}
	if object.Descriptor == nil {
		return fmt.Errorf("java serialization: object has a nil class descriptor")
	}
	hierarchy, err := encoderClassHierarchy(object.Descriptor, e.cfg.limits.MaxClassDescriptors)
	if err != nil {
		return err
	}
	if err := e.writeByte(TCObject); err != nil {
		return err
	}
	if err := e.writeClassDesc(object.Descriptor, depth+1); err != nil {
		return err
	}
	if err := e.addHandle(object); err != nil {
		return err
	}
	expectedData := 0
	for _, desc := range hierarchy {
		if !desc.IsProxy {
			expectedData++
		}
	}
	if len(object.Data) != expectedData {
		return fmt.Errorf("java serialization: object %q has %d class-data entries, want %d", object.Descriptor.Name, len(object.Data), expectedData)
	}
	dataIndex := 0
	for _, desc := range hierarchy {
		if desc.IsProxy {
			continue
		}
		data := &object.Data[dataIndex]
		dataIndex++
		if data.Descriptor != desc {
			return fmt.Errorf("java serialization: class-data entry %d refers to the wrong descriptor", dataIndex-1)
		}
		if desc.Flags&SCExternalizable != 0 {
			return fmt.Errorf("java serialization: externalizable class %q is unsupported", desc.Name)
		}
		if desc.Flags&SCEnum != 0 {
			return fmt.Errorf("java serialization: enum class %q is unsupported", desc.Name)
		}
		if desc.Flags&SCSerializable == 0 {
			return fmt.Errorf("java serialization: class %q is not serializable", desc.Name)
		}
		if desc.Flags&SCWriteMethod == 0 {
			if len(data.Annotation) != 0 {
				return fmt.Errorf("java serialization: class %q has annotation data without SC_WRITE_METHOD", desc.Name)
			}
			if err := e.writeFieldValues(desc, data.Fields, depth+1); err != nil {
				return err
			}
			continue
		}
		if len(data.Fields) != 0 {
			if err := e.writeFieldValues(desc, data.Fields, depth+1); err != nil {
				return err
			}
		}
		if err := e.writeAnnotation(data.Annotation, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) writeFieldValues(desc *ClassDesc, values []FieldValue, depth int) error {
	if len(values) != len(desc.Fields) {
		return fmt.Errorf("java serialization: class %q has %d field values, want %d", desc.Name, len(values), len(desc.Fields))
	}
	for index, field := range desc.Fields {
		provided := values[index]
		if provided.Field.TypeCode != 0 || provided.Field.Name != "" {
			if provided.Field.TypeCode != field.TypeCode || provided.Field.Name != field.Name {
				return fmt.Errorf("java serialization: class %q field value %d does not match descriptor field %q", desc.Name, index, field.Name)
			}
		}
		if err := e.writeElement(field.TypeCode, provided.Value, depth); err != nil {
			return fmt.Errorf("java serialization: class %q field %q: %w", desc.Name, field.Name, err)
		}
	}
	return nil
}

func (e *Encoder) writeArray(array *Array, depth int) error {
	if array == nil {
		return e.writeByte(TCNull)
	}
	if known, err := e.writeReferenceIfKnown(array); known || err != nil {
		return err
	}
	if array.Descriptor == nil || len(array.Descriptor.Name) < 2 || array.Descriptor.Name[0] != '[' {
		return fmt.Errorf("java serialization: array has invalid class descriptor")
	}
	if len(array.Values) > e.cfg.limits.MaxArrayLength {
		return &LimitError{Resource: "array length", Limit: int64(e.cfg.limits.MaxArrayLength)}
	}
	elementType := array.Descriptor.Name[1]
	if elementType == TypeArray {
		elementType = TypeObject
	}
	if !validFieldType(elementType) {
		return fmt.Errorf("java serialization: array %q has invalid element type 0x%02x", array.Descriptor.Name, elementType)
	}
	if err := e.writeByte(TCArray); err != nil {
		return err
	}
	if err := e.writeClassDesc(array.Descriptor, depth+1); err != nil {
		return err
	}
	if err := e.addHandle(array); err != nil {
		return err
	}
	if err := e.writeInt32(int32(len(array.Values))); err != nil {
		return err
	}
	for index, value := range array.Values {
		if err := e.writeElement(elementType, value, depth+1); err != nil {
			return fmt.Errorf("java serialization: array %q element %d: %w", array.Descriptor.Name, index, err)
		}
	}
	return nil
}

func (e *Encoder) writeClass(class *Class, depth int) error {
	if class == nil {
		return e.writeByte(TCNull)
	}
	if known, err := e.writeReferenceIfKnown(class); known || err != nil {
		return err
	}
	if class.Descriptor == nil {
		return fmt.Errorf("java serialization: class token has a nil descriptor")
	}
	if err := e.writeByte(TCClass); err != nil {
		return err
	}
	if err := e.writeClassDesc(class.Descriptor, depth+1); err != nil {
		return err
	}
	return e.addHandle(class)
}

func (e *Encoder) writeElement(typeCode byte, element Element, depth int) error {
	switch typeCode {
	case TypeByte:
		value, ok := element.(Byte)
		if !ok {
			return primitiveTypeError(typeCode, element)
		}
		return e.writeByte(byte(value))
	case TypeChar:
		value, ok := element.(Char)
		if !ok {
			return primitiveTypeError(typeCode, element)
		}
		return e.writeUint16(uint16(value))
	case TypeDouble:
		value, ok := element.(Double)
		if !ok {
			return primitiveTypeError(typeCode, element)
		}
		return e.writeUint64(math.Float64bits(float64(value)))
	case TypeFloat:
		value, ok := element.(Float)
		if !ok {
			return primitiveTypeError(typeCode, element)
		}
		return e.writeUint32(math.Float32bits(float32(value)))
	case TypeInt:
		value, ok := element.(Int)
		if !ok {
			return primitiveTypeError(typeCode, element)
		}
		return e.writeInt32(int32(value))
	case TypeLong:
		value, ok := element.(Long)
		if !ok {
			return primitiveTypeError(typeCode, element)
		}
		return e.writeInt64(int64(value))
	case TypeShort:
		value, ok := element.(Short)
		if !ok {
			return primitiveTypeError(typeCode, element)
		}
		return e.writeUint16(uint16(value))
	case TypeBoolean:
		value, ok := element.(Boolean)
		if !ok {
			return primitiveTypeError(typeCode, element)
		}
		if value {
			return e.writeByte(1)
		}
		return e.writeByte(0)
	case TypeArray, TypeObject:
		value, ok := element.(Value)
		if !ok {
			return fmt.Errorf("expected Java reference value, got %T", element)
		}
		return e.writeValue(value, depth)
	default:
		return fmt.Errorf("invalid element type code 0x%02x", typeCode)
	}
}

func primitiveTypeError(typeCode byte, element Element) error {
	return fmt.Errorf("expected primitive %c, got %T", typeCode, element)
}

func (e *Encoder) writeAnnotation(contents []Content, depth int) error {
	for _, content := range contents {
		if e.annotationCount >= e.cfg.limits.MaxAnnotationItems {
			return &LimitError{Resource: "annotation item count", Limit: int64(e.cfg.limits.MaxAnnotationItems)}
		}
		e.annotationCount++
		if err := e.writeContent(content, depth); err != nil {
			return err
		}
	}
	return e.writeByte(TCEndBlockData)
}

func (e *Encoder) writeBlockData(block *BlockData) error {
	var data []byte
	if block != nil {
		data = block.Data
	}
	if int64(len(data)) > e.cfg.limits.MaxBlockBytes {
		return &LimitError{Resource: "block-data bytes", Limit: e.cfg.limits.MaxBlockBytes}
	}
	if len(data) == 0 {
		if err := e.writeByte(TCBlockData); err != nil {
			return err
		}
		return e.writeByte(0)
	}
	for len(data) > 0 {
		length := len(data)
		if length > normalizedBlockSize {
			length = normalizedBlockSize
		}
		chunk := data[:length]
		data = data[length:]
		if length <= math.MaxUint8 {
			if err := e.writeByte(TCBlockData); err != nil {
				return err
			}
			if err := e.writeByte(byte(length)); err != nil {
				return err
			}
		} else {
			if err := e.writeByte(TCBlockDataLong); err != nil {
				return err
			}
			if err := e.writeInt32(int32(length)); err != nil {
				return err
			}
		}
		if err := e.writeFull(chunk); err != nil {
			return err
		}
	}
	return nil
}

func encoderClassHierarchy(desc *ClassDesc, max int) ([]*ClassDesc, error) {
	seen := make(map[*ClassDesc]struct{})
	var derived []*ClassDesc
	for current := desc; current != nil; current = current.Super {
		if _, ok := seen[current]; ok {
			return nil, fmt.Errorf("java serialization: cyclic class descriptor hierarchy")
		}
		seen[current] = struct{}{}
		derived = append(derived, current)
		if len(derived) > max {
			return nil, &LimitError{Resource: "class hierarchy depth", Limit: int64(max)}
		}
	}
	result := make([]*ClassDesc, len(derived))
	for index := range derived {
		result[len(derived)-1-index] = derived[index]
	}
	return result, nil
}

func (e *Encoder) writeUTF(value string) error {
	encoded := encodeModifiedUTF8(value, nil)
	if int64(len(encoded)) > e.cfg.limits.MaxStringBytes {
		return &LimitError{Resource: "string bytes", Limit: e.cfg.limits.MaxStringBytes}
	}
	if len(encoded) > math.MaxUint16 {
		return fmt.Errorf("modified UTF-8 value is %d bytes; maximum is %d", len(encoded), math.MaxUint16)
	}
	if err := e.writeUint16(uint16(len(encoded))); err != nil {
		return err
	}
	return e.writeFull(encoded)
}

func (e *Encoder) writeByte(value byte) error {
	return e.writeFull([]byte{value})
}

func (e *Encoder) writeUint16(value uint16) error {
	var data [2]byte
	binary.BigEndian.PutUint16(data[:], value)
	return e.writeFull(data[:])
}

func (e *Encoder) writeUint32(value uint32) error {
	var data [4]byte
	binary.BigEndian.PutUint32(data[:], value)
	return e.writeFull(data[:])
}

func (e *Encoder) writeUint64(value uint64) error {
	var data [8]byte
	binary.BigEndian.PutUint64(data[:], value)
	return e.writeFull(data[:])
}

func (e *Encoder) writeInt32(value int32) error { return e.writeUint32(uint32(value)) }
func (e *Encoder) writeInt64(value int64) error { return e.writeUint64(uint64(value)) }

func (e *Encoder) writeFull(data []byte) error {
	remaining := e.cfg.limits.MaxTotalBytes - (e.totalWritten - e.startWritten)
	if int64(len(data)) > remaining {
		return &LimitError{Resource: "stream bytes", Limit: e.cfg.limits.MaxTotalBytes}
	}
	for len(data) > 0 {
		written, err := e.w.Write(data)
		if written > 0 {
			e.totalWritten += int64(written)
			data = data[written:]
		}
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

func (e *Encoder) checkDepth(depth int) error {
	if depth > e.cfg.limits.MaxDepth {
		return &LimitError{Resource: "graph depth", Limit: int64(e.cfg.limits.MaxDepth)}
	}
	return nil
}
