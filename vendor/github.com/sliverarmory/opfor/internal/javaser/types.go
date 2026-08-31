// Package javaser implements a bounded, inert reader and writer for version 5
// Java Object Serialization streams. It represents serialized data as a Go
// object graph; it never loads Java classes or invokes serialization hooks.
package javaser

import "fmt"

// Java Object Serialization stream constants.
const (
	StreamMagic   uint16 = 0xaced
	StreamVersion uint16 = 5

	TCNull           byte = 0x70
	TCReference      byte = 0x71
	TCClassDesc      byte = 0x72
	TCObject         byte = 0x73
	TCString         byte = 0x74
	TCArray          byte = 0x75
	TCClass          byte = 0x76
	TCBlockData      byte = 0x77
	TCEndBlockData   byte = 0x78
	TCReset          byte = 0x79
	TCBlockDataLong  byte = 0x7a
	TCException      byte = 0x7b
	TCLongString     byte = 0x7c
	TCProxyClassDesc byte = 0x7d
	TCEnum           byte = 0x7e

	BaseWireHandle int32 = 0x7e0000
)

// Java Object Serialization class descriptor flags.
const (
	SCWriteMethod    byte = 0x01
	SCSerializable   byte = 0x02
	SCExternalizable byte = 0x04
	SCBlockData      byte = 0x08
	SCEnum           byte = 0x10
)

// Java serialization field and array element type codes.
const (
	TypeByte    byte = 'B'
	TypeChar    byte = 'C'
	TypeDouble  byte = 'D'
	TypeFloat   byte = 'F'
	TypeInt     byte = 'I'
	TypeLong    byte = 'J'
	TypeShort   byte = 'S'
	TypeBoolean byte = 'Z'
	TypeArray   byte = '['
	TypeObject  byte = 'L'
)

// Content is an item that may occur in a serialization stream or a custom
// class-data annotation. Concrete implementations are pointer graph values,
// ClassDesc, and BlockData.
type Content interface {
	isContent()
}

// Element is a value that may occur in an object field or array. It is either
// a Value or one of the strongly typed primitive values below.
type Element interface {
	isElement()
}

// Value is a Java reference value. Every handle-bearing implementation is a
// pointer. Sharing the same pointer in a graph causes Encoder to emit a
// TC_REFERENCE, and Decoder preserves that pointer identity.
type Value interface {
	Content
	Element
	isValue()
}

// Primitive element values. Decoder always returns these exact types, and
// Encoder requires the type matching the descriptor's field or array code.
type (
	Byte    int8
	Char    uint16
	Double  float64
	Float   float32
	Int     int32
	Long    int64
	Short   int16
	Boolean bool
)

func (Byte) isElement()    {}
func (Char) isElement()    {}
func (Double) isElement()  {}
func (Float) isElement()   {}
func (Int) isElement()     {}
func (Long) isElement()    {}
func (Short) isElement()   {}
func (Boolean) isElement() {}

// Null represents TC_NULL. Null values do not receive handles, so callers may
// use either NullValue or another *Null.
type Null struct{}

// NullValue is a convenient shared null value.
var NullValue = &Null{}

func (*Null) isContent() {}
func (*Null) isElement() {}
func (*Null) isValue()   {}

// String represents TC_STRING or TC_LONGSTRING. UTF16 is populated by Decoder
// so unpaired Java UTF-16 surrogate code units can be reproduced losslessly.
// Encoder uses UTF16 when it is non-nil and otherwise encodes Value.
type String struct {
	Value string
	UTF16 []uint16
}

// NewString constructs a Java string from a Go string.
func NewString(value string) *String { return &String{Value: value} }

func (*String) isContent() {}
func (*String) isElement() {}
func (*String) isValue()   {}

// Object is an inert Java object graph node. Data is ordered from the highest
// serializable superclass to Descriptor, matching the wire protocol.
type Object struct {
	Descriptor *ClassDesc
	Data       []ClassData
}

func (*Object) isContent() {}
func (*Object) isElement() {}
func (*Object) isValue()   {}

// Array is an inert Java array node. Descriptor.Name determines the element
// kind, for example "[I" or "[Ljava.lang.Object;".
type Array struct {
	Descriptor *ClassDesc
	Values     []Element
}

func (*Array) isContent() {}
func (*Array) isElement() {}
func (*Array) isValue()   {}

// Class represents a serialized java.lang.Class token (TC_CLASS).
type Class struct {
	Descriptor *ClassDesc
}

func (*Class) isContent() {}
func (*Class) isElement() {}
func (*Class) isValue()   {}

// BlockData is one primitive-data record in a class annotation. Decoder keeps
// record boundaries. Encoder may split records larger than the protocol's
// normalized 1024-byte block size.
type BlockData struct {
	Data []byte
}

func (*BlockData) isContent() {}

// ClassDesc is an inert ObjectStreamClass descriptor. IsProxy selects the
// TC_PROXYCLASSDESC representation; proxy descriptors use ProxyInterfaces and
// do not carry Name, UID, Flags, or Fields on the wire.
type ClassDesc struct {
	Name             string
	SerialVersionUID int64
	Flags            byte
	Fields           []FieldDesc
	Annotation       []Content
	Super            *ClassDesc
	IsProxy          bool
	ProxyInterfaces  []string
}

func (*ClassDesc) isContent() {}

// FieldDesc describes one serialized field. ClassName is required for '[' and
// 'L' fields and itself participates in stream handle identity.
type FieldDesc struct {
	TypeCode  byte
	Name      string
	ClassName *String
}

// FieldValue pairs a descriptor field with its decoded value. Values are kept
// in descriptor order. Field is copied so callers can inspect data without
// chasing descriptor indexes.
type FieldValue struct {
	Field FieldDesc
	Value Element
}

// ClassData contains the data for one class in an Object hierarchy. Fields is
// empty for a custom writeObject method that did not call defaultWriteObject.
// Annotation excludes the implicit terminating TC_ENDBLOCKDATA marker.
type ClassData struct {
	Descriptor *ClassDesc
	Fields     []FieldValue
	Annotation []Content
}

// Field returns the named default field in this class data.
func (d *ClassData) Field(name string) (Element, bool) {
	if d == nil {
		return nil, false
	}
	for i := range d.Fields {
		if d.Fields[i].Field.Name == name {
			return d.Fields[i].Value, true
		}
	}
	return nil, false
}

// DataFor returns the data associated with the named descriptor.
func (o *Object) DataFor(className string) (*ClassData, bool) {
	if o == nil {
		return nil, false
	}
	for i := range o.Data {
		if o.Data[i].Descriptor != nil && o.Data[i].Descriptor.Name == className {
			return &o.Data[i], true
		}
	}
	return nil, false
}

// ClassDataLayout tells Decoder how a class with SC_WRITE_METHOD laid out its
// data. Java's wire format does not say whether writeObject called
// defaultWriteObject, so this decision must come from an allowlisted schema.
type ClassDataLayout uint8

const (
	// ClassDataAuto applies the safe protocol default: ordinary serializable
	// classes have default fields; classes with SC_WRITE_METHOD are rejected.
	ClassDataAuto ClassDataLayout = iota
	// ClassDataDefaultFields decodes descriptor fields and no annotation.
	ClassDataDefaultFields
	// ClassDataDefaultFieldsAndAnnotation decodes descriptor fields followed by
	// custom contents terminated by TC_ENDBLOCKDATA.
	ClassDataDefaultFieldsAndAnnotation
	// ClassDataAnnotationOnly decodes only custom contents terminated by
	// TC_ENDBLOCKDATA, for writeObject methods that omit defaultWriteObject.
	ClassDataAnnotationOnly
)

func (l ClassDataLayout) String() string {
	switch l {
	case ClassDataAuto:
		return "auto"
	case ClassDataDefaultFields:
		return "default fields"
	case ClassDataDefaultFieldsAndAnnotation:
		return "default fields and annotation"
	case ClassDataAnnotationOnly:
		return "annotation only"
	default:
		return fmt.Sprintf("ClassDataLayout(%d)", l)
	}
}

// ClassDataResolver is invoked once per non-proxy descriptor in an object's
// hierarchy. Returning ClassDataAuto uses Decoder's safe default.
type ClassDataResolver func(*ClassDesc) (ClassDataLayout, error)
