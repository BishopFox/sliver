package javaser

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// Decoder reads independent version 5 Java Object Serialization streams. It
// performs only exact-sized reads and never installs a buffered reader, so a
// successful Decode leaves the first byte of a following stream untouched.
type Decoder struct {
	r   io.Reader
	cfg decoderConfig

	totalRead int64
	startRead int64
	lastRead  int64

	handles         []Content
	classCount      int
	fieldCount      int
	annotationCount int
}

// NewDecoder constructs a bounded decoder over r.
func NewDecoder(r io.Reader, options ...Option) *Decoder {
	cfg := decoderConfig{limits: DefaultLimits()}
	for _, option := range options {
		if option != nil {
			option.applyDecoder(&cfg)
		}
	}
	cfg.limits = cfg.limits.normalized()
	return &Decoder{r: r, cfg: cfg}
}

// Decode reads one independent stream and requires its root to be a Java
// reference value. An empty input returns io.EOF; a partial stream returns
// io.ErrUnexpectedEOF. Leading TC_RESET records are honored.
func (d *Decoder) Decode() (Value, error) {
	content, err := d.decodeContent()
	if err != nil {
		return nil, err
	}
	value, ok := content.(Value)
	if !ok {
		return nil, d.protocolf("root content %T is not a Java value", content)
	}
	return value, nil
}

// DecodeContent is like Decode but also permits a class descriptor or block
// data record as the root content.
func (d *Decoder) DecodeContent() (Content, error) {
	return d.decodeContent()
}

// BytesRead reports bytes consumed by the most recent Decode or DecodeContent
// call, including the four-byte stream header.
func (d *Decoder) BytesRead() int64 { return d.lastRead }

// Decode reads one independent stream from r with a fresh decoder.
func Decode(r io.Reader, options ...Option) (Value, error) {
	return NewDecoder(r, options...).Decode()
}

func (d *Decoder) decodeContent() (content Content, err error) {
	if d == nil || d.r == nil {
		return nil, fmt.Errorf("java serialization: nil decoder reader")
	}
	d.startRead = d.totalRead
	d.lastRead = 0
	d.handles = nil
	d.classCount = 0
	d.fieldCount = 0
	d.annotationCount = 0
	defer func() { d.lastRead = d.totalRead - d.startRead }()

	header := make([]byte, 4)
	if err := d.readFull(header); err != nil {
		return nil, err
	}
	magic := binary.BigEndian.Uint16(header[:2])
	version := binary.BigEndian.Uint16(header[2:])
	if magic != StreamMagic {
		return nil, d.protocolf("bad stream magic 0x%04x", magic)
	}
	if version != StreamVersion {
		return nil, d.protocolf("unsupported stream version %d", version)
	}

	for {
		tag, err := d.readByte()
		if err != nil {
			return nil, err
		}
		if tag == TCReset {
			d.handles = nil
			continue
		}
		return d.readContentTag(tag, 1)
	}
}

func (d *Decoder) readContentTag(tag byte, depth int) (Content, error) {
	if err := d.checkDepth(depth); err != nil {
		return nil, err
	}
	switch tag {
	case TCNull:
		return NullValue, nil
	case TCReference:
		return d.readReference()
	case TCClassDesc:
		return d.readNewClassDesc(depth)
	case TCProxyClassDesc:
		return d.readNewProxyClassDesc(depth)
	case TCObject:
		return d.readNewObject(depth)
	case TCString:
		return d.readNewString(false)
	case TCLongString:
		return d.readNewString(true)
	case TCArray:
		return d.readNewArray(depth)
	case TCClass:
		return d.readNewClass(depth)
	case TCBlockData:
		return d.readBlockData(false)
	case TCBlockDataLong:
		return d.readBlockData(true)
	case TCEndBlockData:
		return nil, d.protocolf("unexpected TC_ENDBLOCKDATA")
	case TCReset:
		return nil, d.protocolf("TC_RESET is only valid between top-level contents")
	case TCException:
		return nil, d.protocolf("TC_EXCEPTION is unsupported")
	case TCEnum:
		return nil, d.protocolf("TC_ENUM is unsupported")
	default:
		return nil, d.protocolf("unknown type code 0x%02x", tag)
	}
}

func (d *Decoder) readValue(depth int) (Value, error) {
	tag, err := d.readByte()
	if err != nil {
		return nil, err
	}
	content, err := d.readContentTag(tag, depth)
	if err != nil {
		return nil, err
	}
	value, ok := content.(Value)
	if !ok {
		return nil, d.protocolf("content %T is not valid in a reference-value position", content)
	}
	return value, nil
}

func (d *Decoder) readReference() (Content, error) {
	handle, err := d.readInt32()
	if err != nil {
		return nil, err
	}
	index := int64(handle) - int64(BaseWireHandle)
	if index < 0 || index >= int64(len(d.handles)) {
		return nil, d.protocolf("invalid wire handle 0x%08x", uint32(handle))
	}
	return d.handles[index], nil
}

func (d *Decoder) addHandle(content Content) error {
	if len(d.handles) >= d.cfg.limits.MaxHandles {
		return &LimitError{Resource: "handle count", Limit: int64(d.cfg.limits.MaxHandles)}
	}
	d.handles = append(d.handles, content)
	return nil
}

func (d *Decoder) readNewString(long bool) (*String, error) {
	value := &String{}
	if err := d.addHandle(value); err != nil {
		return nil, err
	}
	var length int64
	if long {
		read, err := d.readInt64()
		if err != nil {
			return nil, err
		}
		length = read
		if length < 0 {
			return nil, d.protocolf("negative long-string length %d", length)
		}
	} else {
		read, err := d.readUint16()
		if err != nil {
			return nil, err
		}
		length = int64(read)
	}
	if err := d.checkLength("string bytes", length, d.cfg.limits.MaxStringBytes); err != nil {
		return nil, err
	}
	data, err := d.readBytes(length)
	if err != nil {
		return nil, err
	}
	decoded, units, err := decodeModifiedUTF8(data)
	if err != nil {
		return nil, d.protocolf("%v", err)
	}
	value.Value = decoded
	value.UTF16 = units
	return value, nil
}

func (d *Decoder) readNewClassDesc(depth int) (*ClassDesc, error) {
	name, err := d.readUTF()
	if err != nil {
		return nil, err
	}
	uid, err := d.readInt64()
	if err != nil {
		return nil, err
	}
	desc := &ClassDesc{Name: name, SerialVersionUID: uid}
	if err := d.addClassDescriptor(desc); err != nil {
		return nil, err
	}
	flags, err := d.readByte()
	if err != nil {
		return nil, err
	}
	desc.Flags = flags
	if flags & ^byte(SCWriteMethod|SCSerializable|SCExternalizable|SCBlockData|SCEnum) != 0 {
		return nil, d.protocolf("class %q has unknown descriptor flags 0x%02x", name, flags)
	}
	fieldCount, err := d.readUint16()
	if err != nil {
		return nil, err
	}
	if int(fieldCount) > d.cfg.limits.MaxFieldsPerClass {
		return nil, &LimitError{Resource: "fields per class", Limit: int64(d.cfg.limits.MaxFieldsPerClass)}
	}
	if d.fieldCount > d.cfg.limits.MaxTotalFields-int(fieldCount) {
		return nil, &LimitError{Resource: "total field count", Limit: int64(d.cfg.limits.MaxTotalFields)}
	}
	d.fieldCount += int(fieldCount)
	desc.Fields = make([]FieldDesc, int(fieldCount))
	for index := range desc.Fields {
		typeCode, err := d.readByte()
		if err != nil {
			return nil, err
		}
		if !validFieldType(typeCode) {
			return nil, d.protocolf("class %q field %d has invalid type code 0x%02x", name, index, typeCode)
		}
		fieldName, err := d.readUTF()
		if err != nil {
			return nil, err
		}
		field := FieldDesc{TypeCode: typeCode, Name: fieldName}
		if typeCode == TypeArray || typeCode == TypeObject {
			className, err := d.readValue(depth + 1)
			if err != nil {
				return nil, err
			}
			stringName, ok := className.(*String)
			if !ok {
				return nil, d.protocolf("class %q field %q type descriptor is %T, not a string", name, fieldName, className)
			}
			field.ClassName = stringName
		}
		desc.Fields[index] = field
	}
	desc.Annotation, err = d.readAnnotation(depth + 1)
	if err != nil {
		return nil, err
	}
	desc.Super, err = d.readClassDesc(depth + 1)
	if err != nil {
		return nil, err
	}
	return desc, nil
}

func (d *Decoder) readNewProxyClassDesc(depth int) (*ClassDesc, error) {
	desc := &ClassDesc{IsProxy: true}
	if err := d.addClassDescriptor(desc); err != nil {
		return nil, err
	}
	count, err := d.readInt32()
	if err != nil {
		return nil, err
	}
	if count < 0 {
		return nil, d.protocolf("negative proxy interface count %d", count)
	}
	if int64(count) > int64(d.cfg.limits.MaxProxyInterfaces) {
		return nil, &LimitError{Resource: "proxy interface count", Limit: int64(d.cfg.limits.MaxProxyInterfaces)}
	}
	desc.ProxyInterfaces = make([]string, int(count))
	for index := range desc.ProxyInterfaces {
		desc.ProxyInterfaces[index], err = d.readUTF()
		if err != nil {
			return nil, err
		}
	}
	desc.Annotation, err = d.readAnnotation(depth + 1)
	if err != nil {
		return nil, err
	}
	desc.Super, err = d.readClassDesc(depth + 1)
	if err != nil {
		return nil, err
	}
	return desc, nil
}

func (d *Decoder) addClassDescriptor(desc *ClassDesc) error {
	if d.classCount >= d.cfg.limits.MaxClassDescriptors {
		return &LimitError{Resource: "class descriptor count", Limit: int64(d.cfg.limits.MaxClassDescriptors)}
	}
	d.classCount++
	return d.addHandle(desc)
}

func (d *Decoder) readClassDesc(depth int) (*ClassDesc, error) {
	if err := d.checkDepth(depth); err != nil {
		return nil, err
	}
	tag, err := d.readByte()
	if err != nil {
		return nil, err
	}
	switch tag {
	case TCNull:
		return nil, nil
	case TCReference:
		content, err := d.readReference()
		if err != nil {
			return nil, err
		}
		desc, ok := content.(*ClassDesc)
		if !ok {
			return nil, d.protocolf("class descriptor reference resolves to %T", content)
		}
		return desc, nil
	case TCClassDesc:
		return d.readNewClassDesc(depth)
	case TCProxyClassDesc:
		return d.readNewProxyClassDesc(depth)
	default:
		return nil, d.protocolf("invalid class descriptor type code 0x%02x", tag)
	}
}

func (d *Decoder) readNewObject(depth int) (*Object, error) {
	desc, err := d.readClassDesc(depth + 1)
	if err != nil {
		return nil, err
	}
	if desc == nil {
		return nil, d.protocolf("TC_OBJECT has a null class descriptor")
	}
	object := &Object{Descriptor: desc}
	if err := d.addHandle(object); err != nil {
		return nil, err
	}
	hierarchy, err := d.classHierarchy(desc)
	if err != nil {
		return nil, err
	}
	object.Data = make([]ClassData, 0, len(hierarchy))
	for _, class := range hierarchy {
		if class.IsProxy {
			continue
		}
		layout, err := d.resolveClassData(class)
		if err != nil {
			return nil, err
		}
		data := ClassData{Descriptor: class}
		if layout == ClassDataDefaultFields || layout == ClassDataDefaultFieldsAndAnnotation {
			data.Fields, err = d.readFieldValues(class, depth+1)
			if err != nil {
				return nil, err
			}
		}
		if layout == ClassDataDefaultFieldsAndAnnotation || layout == ClassDataAnnotationOnly {
			data.Annotation, err = d.readAnnotation(depth + 1)
			if err != nil {
				return nil, err
			}
		}
		object.Data = append(object.Data, data)
	}
	return object, nil
}

func (d *Decoder) resolveClassData(desc *ClassDesc) (ClassDataLayout, error) {
	flags := desc.Flags
	if flags&SCExternalizable != 0 {
		return ClassDataAuto, d.protocolf("externalizable class %q is unsupported", desc.Name)
	}
	if flags&SCEnum != 0 {
		return ClassDataAuto, d.protocolf("enum class %q is unsupported", desc.Name)
	}
	if flags&SCSerializable == 0 {
		return ClassDataAuto, d.protocolf("class %q is not serializable", desc.Name)
	}
	if flags&SCWriteMethod != 0 && flags&SCSerializable == 0 {
		return ClassDataAuto, d.protocolf("class %q has SC_WRITE_METHOD without SC_SERIALIZABLE", desc.Name)
	}
	layout := ClassDataAuto
	var err error
	if d.cfg.resolver != nil {
		layout, err = d.cfg.resolver(desc)
		if err != nil {
			return ClassDataAuto, err
		}
	}
	if layout == ClassDataAuto {
		if flags&SCWriteMethod != 0 {
			return ClassDataAuto, d.protocolf("class %q has custom writeObject data but no class-data layout was registered", desc.Name)
		}
		return ClassDataDefaultFields, nil
	}
	switch layout {
	case ClassDataDefaultFields:
		if flags&SCWriteMethod != 0 {
			return ClassDataAuto, d.protocolf("class %q uses SC_WRITE_METHOD but resolver selected %s", desc.Name, layout)
		}
	case ClassDataDefaultFieldsAndAnnotation, ClassDataAnnotationOnly:
		if flags&SCWriteMethod == 0 {
			return ClassDataAuto, d.protocolf("class %q does not use SC_WRITE_METHOD but resolver selected %s", desc.Name, layout)
		}
	default:
		return ClassDataAuto, d.protocolf("class %q resolver returned invalid layout %d", desc.Name, layout)
	}
	return layout, nil
}

func (d *Decoder) classHierarchy(desc *ClassDesc) ([]*ClassDesc, error) {
	seen := make(map[*ClassDesc]struct{})
	var derived []*ClassDesc
	for current := desc; current != nil; current = current.Super {
		if _, ok := seen[current]; ok {
			return nil, d.protocolf("cyclic class descriptor hierarchy")
		}
		seen[current] = struct{}{}
		derived = append(derived, current)
		if len(derived) > d.cfg.limits.MaxClassDescriptors {
			return nil, &LimitError{Resource: "class hierarchy depth", Limit: int64(d.cfg.limits.MaxClassDescriptors)}
		}
	}
	result := make([]*ClassDesc, len(derived))
	for i := range derived {
		result[len(derived)-1-i] = derived[i]
	}
	return result, nil
}

func (d *Decoder) readFieldValues(desc *ClassDesc, depth int) ([]FieldValue, error) {
	values := make([]FieldValue, len(desc.Fields))
	for index, field := range desc.Fields {
		value, err := d.readElement(field.TypeCode, depth)
		if err != nil {
			return nil, fmt.Errorf("java serialization: class %q field %q: %w", desc.Name, field.Name, err)
		}
		values[index] = FieldValue{Field: field, Value: value}
	}
	return values, nil
}

func (d *Decoder) readNewArray(depth int) (*Array, error) {
	desc, err := d.readClassDesc(depth + 1)
	if err != nil {
		return nil, err
	}
	if desc == nil || len(desc.Name) < 2 || desc.Name[0] != '[' {
		return nil, d.protocolf("TC_ARRAY has invalid class descriptor")
	}
	array := &Array{Descriptor: desc}
	if err := d.addHandle(array); err != nil {
		return nil, err
	}
	length, err := d.readInt32()
	if err != nil {
		return nil, err
	}
	if length < 0 {
		return nil, d.protocolf("negative array length %d", length)
	}
	if int64(length) > int64(d.cfg.limits.MaxArrayLength) {
		return nil, &LimitError{Resource: "array length", Limit: int64(d.cfg.limits.MaxArrayLength)}
	}
	elementType := desc.Name[1]
	if elementType == TypeArray {
		elementType = TypeObject
	}
	if !validFieldType(elementType) {
		return nil, d.protocolf("array %q has invalid element type 0x%02x", desc.Name, elementType)
	}
	array.Values = make([]Element, int(length))
	for index := range array.Values {
		array.Values[index], err = d.readElement(elementType, depth+1)
		if err != nil {
			return nil, fmt.Errorf("java serialization: array %q element %d: %w", desc.Name, index, err)
		}
	}
	return array, nil
}

func (d *Decoder) readNewClass(depth int) (*Class, error) {
	desc, err := d.readClassDesc(depth + 1)
	if err != nil {
		return nil, err
	}
	if desc == nil {
		return nil, d.protocolf("TC_CLASS has a null class descriptor")
	}
	class := &Class{Descriptor: desc}
	if err := d.addHandle(class); err != nil {
		return nil, err
	}
	return class, nil
}

func (d *Decoder) readElement(typeCode byte, depth int) (Element, error) {
	switch typeCode {
	case TypeByte:
		value, err := d.readByte()
		return Byte(int8(value)), err
	case TypeChar:
		value, err := d.readUint16()
		return Char(value), err
	case TypeDouble:
		value, err := d.readUint64()
		return Double(math.Float64frombits(value)), err
	case TypeFloat:
		value, err := d.readUint32()
		return Float(math.Float32frombits(value)), err
	case TypeInt:
		value, err := d.readInt32()
		return Int(value), err
	case TypeLong:
		value, err := d.readInt64()
		return Long(value), err
	case TypeShort:
		value, err := d.readUint16()
		return Short(int16(value)), err
	case TypeBoolean:
		value, err := d.readByte()
		if err != nil {
			return nil, err
		}
		if value != 0 && value != 1 {
			return nil, d.protocolf("invalid boolean byte 0x%02x", value)
		}
		return Boolean(value != 0), nil
	case TypeArray, TypeObject:
		return d.readValue(depth)
	default:
		return nil, d.protocolf("invalid element type code 0x%02x", typeCode)
	}
}

func (d *Decoder) readAnnotation(depth int) ([]Content, error) {
	var contents []Content
	for {
		tag, err := d.readByte()
		if err != nil {
			return nil, err
		}
		if tag == TCEndBlockData {
			return contents, nil
		}
		if tag == TCReset {
			return nil, d.protocolf("TC_RESET is invalid inside class annotation data")
		}
		content, err := d.readContentTag(tag, depth)
		if err != nil {
			return nil, err
		}
		if d.annotationCount >= d.cfg.limits.MaxAnnotationItems {
			return nil, &LimitError{Resource: "annotation item count", Limit: int64(d.cfg.limits.MaxAnnotationItems)}
		}
		d.annotationCount++
		contents = append(contents, content)
	}
}

func (d *Decoder) readBlockData(long bool) (*BlockData, error) {
	var length int64
	if long {
		value, err := d.readInt32()
		if err != nil {
			return nil, err
		}
		if value < 0 {
			return nil, d.protocolf("negative long block-data length %d", value)
		}
		length = int64(value)
	} else {
		value, err := d.readByte()
		if err != nil {
			return nil, err
		}
		length = int64(value)
	}
	if err := d.checkLength("block-data bytes", length, d.cfg.limits.MaxBlockBytes); err != nil {
		return nil, err
	}
	data, err := d.readBytes(length)
	if err != nil {
		return nil, err
	}
	return &BlockData{Data: data}, nil
}

func (d *Decoder) readUTF() (string, error) {
	length, err := d.readUint16()
	if err != nil {
		return "", err
	}
	if err := d.checkLength("string bytes", int64(length), d.cfg.limits.MaxStringBytes); err != nil {
		return "", err
	}
	data, err := d.readBytes(int64(length))
	if err != nil {
		return "", err
	}
	value, _, err := decodeModifiedUTF8(data)
	if err != nil {
		return "", d.protocolf("%v", err)
	}
	return value, nil
}

func (d *Decoder) readBytes(length int64) ([]byte, error) {
	if length < 0 || uint64(length) > uint64(maxInt()) {
		return nil, d.protocolf("byte length %d cannot be represented", length)
	}
	data := make([]byte, int(length))
	if err := d.readFull(data); err != nil {
		return nil, err
	}
	return data, nil
}

func (d *Decoder) readByte() (byte, error) {
	var data [1]byte
	err := d.readFull(data[:])
	return data[0], err
}

func (d *Decoder) readUint16() (uint16, error) {
	var data [2]byte
	if err := d.readFull(data[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(data[:]), nil
}

func (d *Decoder) readUint32() (uint32, error) {
	var data [4]byte
	if err := d.readFull(data[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(data[:]), nil
}

func (d *Decoder) readUint64() (uint64, error) {
	var data [8]byte
	if err := d.readFull(data[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(data[:]), nil
}

func (d *Decoder) readInt32() (int32, error) {
	value, err := d.readUint32()
	return int32(value), err
}

func (d *Decoder) readInt64() (int64, error) {
	value, err := d.readUint64()
	return int64(value), err
}

func (d *Decoder) readFull(buffer []byte) error {
	remaining := d.cfg.limits.MaxTotalBytes - (d.totalRead - d.startRead)
	if int64(len(buffer)) > remaining {
		return &LimitError{Resource: "stream bytes", Limit: d.cfg.limits.MaxTotalBytes}
	}
	read, err := io.ReadFull(d.r, buffer)
	d.totalRead += int64(read)
	return err
}

func (d *Decoder) checkDepth(depth int) error {
	if depth > d.cfg.limits.MaxDepth {
		return &LimitError{Resource: "graph depth", Limit: int64(d.cfg.limits.MaxDepth)}
	}
	return nil
}

func (d *Decoder) checkLength(resource string, length, limit int64) error {
	if length > limit {
		return &LimitError{Resource: resource, Limit: limit}
	}
	return nil
}

func (d *Decoder) protocolf(format string, args ...any) error {
	return &ProtocolError{Offset: d.totalRead - d.startRead, Message: fmt.Sprintf(format, args...)}
}

func validFieldType(typeCode byte) bool {
	switch typeCode {
	case TypeByte, TypeChar, TypeDouble, TypeFloat, TypeInt, TypeLong,
		TypeShort, TypeBoolean, TypeArray, TypeObject:
		return true
	default:
		return false
	}
}

func maxInt() int { return int(^uint(0) >> 1) }
